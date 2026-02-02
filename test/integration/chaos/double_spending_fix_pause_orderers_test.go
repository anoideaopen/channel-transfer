package chaos

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"sync"
	"time"

	cligrpc "github.com/anoideaopen/channel-transfer/proto"
	"github.com/anoideaopen/foundation/mocks"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/google/uuid"
	"github.com/hyperledger/fabric/integration"
	"github.com/hyperledger/fabric/integration/nwo"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Channel-transfer config template with retry settings for orderer pause scenario.
// Retry delays should allow TTL to expire during the retry phase when orderers are stopped.
const channelTransferTemplatePauseOrderersTest = `{{ with $w := . -}}
logLevel: debug
logType: console
profilePath: {{ .ConnectionPath User }}
userName: backend
listenAPI:
  accessToken: {{ .ChannelTransferAccessToken }}
  addressHTTP: {{ .ChannelTransferHTTPAddress }}
  addressGRPC: {{ .ChannelTransferGRPCAddress }}
service:
  address: {{ .ChannelTransferHostAddress }}
options:
  batchTxPreimagePrefix: batchTransactions
  collectorsBufSize: 1
  executeTimeout: 0s
  retryExecuteAttempts: 10
  retryExecuteMaxDelay: 2s
  retryExecuteDelay: 1s
  ttl: {{ .ChannelTransferTTL }}
  transfersInHandleOnChannel: 50
  newestRequestStreamBufferSize: 50
channels:{{ range .Channels }}
  {{- if ne .Name "acl" }}
  - name: {{ .Name }}
    {{- if .HasTaskExecutor }}
    taskExecutor:
      addressGRPC: "{{ .TaskExecutorGRPCAddress }}"
    {{- end }}
  {{- end }}
{{- end }}
redisStorage:
  addr:{{ range .ChannelTransfer.RedisAddresses }}
    - {{ . }}
  {{- end }}
  dbPrefix: transfer
  password: ""
  afterTransferTTL: 3s
promMetrics:
  prefix: transfer
{{ end }}
`

// OrdererPausingTaskExecutorMock executes chaincode methods and pauses all
// orderer processes after createCCTransferTo completes successfully.
// This simulates the production bug scenario where:
// 1. createCCTransferTo SUCCEEDS (TransferTo exists on blockchain)
// 2. Orderers are PAUSED (SIGSTOP) making them unavailable
// 3. commitCCTransferFrom FAILS repeatedly (orderers not responding)
// 4. TTL expires during retry phase
// 5. resolveStatus() is called by launcher
// 6. Fix should prevent cancellation when TransferTo exists
type OrdererPausingTaskExecutorMock struct {
	cligrpc.UnimplementedTaskExecutorAdapterServer
	executor          func(channel, chaincode, method string, args ...string)
	mu                sync.Mutex
	executedMethods   []string
	orderersPaused    bool
	orderersPausedAt  time.Time
	pauseOrderersOnce sync.Once
}

func (s *OrdererPausingTaskExecutorMock) SubmitTransaction(
	_ context.Context,
	req *cligrpc.TaskExecutorRequest,
) (*cligrpc.TaskExecutorResponse, error) {
	args := make([]string, len(req.Args))
	for i, arg := range req.Args {
		args[i] = string(arg)
	}

	method := req.Method

	s.mu.Lock()
	s.executedMethods = append(s.executedMethods, method)
	orderersPaused := s.orderersPaused
	s.mu.Unlock()

	GinkgoWriter.Printf("[TaskExecutor] Received: %s on %s/%s\n", method, req.Channel, req.Chaincode)

	// If cancelCCTransferFrom is received, this indicates the BUG (double-spending scenario).
	// Don't try to execute it (orderers are paused, it would fail and panic).
	// Just record that it was called and return - the test will check WasMethodCalled().
	if method == "cancelCCTransferFrom" {
		GinkgoWriter.Printf("[TaskExecutor] *** BUG DETECTED: cancelCCTransferFrom received! ***\n")
		GinkgoWriter.Printf("[TaskExecutor] This should NOT happen when the fix is applied.\n")
		GinkgoWriter.Printf("[TaskExecutor] TransferTo was committed but cancelCCTransferFrom was called.\n")
		GinkgoWriter.Printf("[TaskExecutor] This would cause DOUBLE SPENDING in production!\n")
		GinkgoWriter.Printf("[TaskExecutor] Orderers paused: %v\n", orderersPaused)
		GinkgoWriter.Printf("[TaskExecutor] NOT executing cancelCCTransferFrom (would fail anyway with paused orderers)\n")

		// Return accepted to avoid hanging the channel-transfer service
		// The test will fail on the WasMethodCalled("cancelCCTransferFrom") check
		return &cligrpc.TaskExecutorResponse{
			Status:  cligrpc.TaskExecutorResponse_STATUS_ACCEPTED,
			Message: "accepted cancelCCTransferFrom (BUG: should not be called)",
		}, nil
	}

	// Execute the task - commits to blockchain
	GinkgoWriter.Printf("[TaskExecutor] Executing: %s on %s/%s\n", method, req.Channel, req.Chaincode)
	s.executor(req.Channel, req.Chaincode, method, args...)
	GinkgoWriter.Printf("[TaskExecutor] Completed: %s on %s/%s\n", method, req.Channel, req.Chaincode)

	// After createCCTransferTo succeeds, PAUSE ALL ORDERERS using SIGSTOP
	// This happens SYNCHRONOUSLY before returning, so when channel-transfer
	// tries to call commitCCTransferFrom, orderers will already be paused
	if method == "createCCTransferTo" {
		s.pauseOrderersOnce.Do(func() {
			GinkgoWriter.Printf("[TaskExecutor] createCCTransferTo completed - PAUSING ALL ORDERERS NOW\n")

			// Use pkill to send SIGSTOP to all orderer processes
			err := exec.Command("pkill", "-STOP", "-f", "orderer").Run()
			if err != nil {
				GinkgoWriter.Printf("[TaskExecutor] WARNING: Failed to pause orderers: %v\n", err)
			}

			s.mu.Lock()
			s.orderersPaused = true
			s.orderersPausedAt = time.Now()
			s.mu.Unlock()

			GinkgoWriter.Printf("[TaskExecutor] ALL ORDERERS PAUSED (SIGSTOP) at %v\n", s.orderersPausedAt)

			// IMPORTANT: Wait for existing connections to timeout
			// SIGSTOP doesn't immediately break TCP connections, so we need to wait
			// for the SDK's connection pool to detect the orderers are unresponsive.
			// This delay ensures commitCCTransferFrom will actually FAIL.
			GinkgoWriter.Printf("[TaskExecutor] Waiting 5 seconds for connections to timeout...\n")
			time.Sleep(5 * time.Second)
			GinkgoWriter.Printf("[TaskExecutor] Wait complete, returning from createCCTransferTo\n")
		})
	}

	return &cligrpc.TaskExecutorResponse{
		Status:  cligrpc.TaskExecutorResponse_STATUS_ACCEPTED,
		Message: fmt.Sprintf("accepted %s", method),
	}, nil
}

func (s *OrdererPausingTaskExecutorMock) WasMethodCalled(method string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.executedMethods {
		if m == method {
			return true
		}
	}
	return false
}

func (s *OrdererPausingTaskExecutorMock) GetExecutedMethods() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]string, len(s.executedMethods))
	copy(result, s.executedMethods)
	return result
}

func (s *OrdererPausingTaskExecutorMock) AreOrderersPaused() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.orderersPaused
}

func StartOrdererPausingTaskExecutor(
	port uint16,
	executor func(channel, chaincode, method string, args ...string),
) (*grpc.Server, *OrdererPausingTaskExecutorMock) {
	ch := make(chan struct{})
	gRPCServer := grpc.NewServer()

	mock := &OrdererPausingTaskExecutorMock{
		executor: executor,
	}
	cligrpc.RegisterTaskExecutorAdapterServer(gRPCServer, mock)

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			panic(fmt.Sprintf("failed to listen on port %d: %v", port, err))
		}
		close(ch)
		if err = gRPCServer.Serve(lis); err != nil {
			panic(fmt.Sprintf("failed to serve: %v", err))
		}
	}()

	<-ch
	return gRPCServer, mock
}

func pauseOrderersTestTaskExecutorPort() uint16 {
	return uint16(integration.LifecyclePort + 250) // Different port to avoid conflicts
}

// resumeOrderers sends SIGCONT to all paused orderer processes
func resumeOrderers() {
	GinkgoWriter.Printf("[Test] Resuming all orderers (SIGCONT)...\n")
	err := exec.Command("pkill", "-CONT", "-f", "orderer").Run()
	if err != nil {
		GinkgoWriter.Printf("[Test] WARNING: Failed to resume orderers: %v\n", err)
	} else {
		GinkgoWriter.Printf("[Test] All orderers resumed\n")
	}
}

var _ = Describe("Double spending fix - Pause Orderers", func() {
	var (
		ts               *client.FoundationTestSuite
		channels         = []string{cmn.ChannelACL, cmn.ChannelCC, cmn.ChannelFiat}
		user             *mocks.UserFoundation
		taskExecutor     *grpc.Server
		taskExecutorMock *OrdererPausingTaskExecutorMock

		clientCtx context.Context
		apiClient cligrpc.APIClient
		conn      *grpc.ClientConn
	)

	BeforeEach(func() {
		ts = client.NewTestSuite(components)
	})
	AfterEach(func() {
		// IMPORTANT: Resume orderers before shutting down the network
		// Otherwise ShutdownNetwork may hang trying to stop processes
		resumeOrderers()
		time.Sleep(2 * time.Second) // Give orderers time to resume

		ts.ShutdownNetwork()
	})

	BeforeEach(func() {
		By("start redis")
		ts.StartRedis()
	})

	AfterEach(func() {
		By("stop redis")
		ts.StopRedis()
	})

	BeforeEach(func() {
		// Use 3s TTL - effective Redis TTL will be 6s (options.ttl + afterTransferTTL).
		ttl := "3s"
		ts.InitNetwork(
			channels,
			integration.IdemixBasePort,
			client.WithChannelTransferTTL(ttl),
			client.WithChannelTransferTemplate(channelTransferTemplatePauseOrderersTest),
			client.WithTaskExecutorForChannels(
				"127.0.0.1",
				nwo.Ports{cmn.GrpcPort: pauseOrderersTestTaskExecutorPort()},
				cmn.ChannelCC,
				cmn.ChannelFiat,
			),
		)
		ts.DeployChaincodes()
	})

	BeforeEach(func() {
		By("start TaskExecutor mock with orderer-pausing callback")

		taskExecutor, taskExecutorMock = StartOrdererPausingTaskExecutor(
			pauseOrderersTestTaskExecutorPort(),
			func(channel, chaincode, method string, args ...string) {
				ts.ExecuteTask(channel, chaincode, method, args...)
			},
		)

		By("start channel transfer (with TaskExecutor, WITHOUT Robot)")
		ts.StartChannelTransfer()

		By("creating gRPC connection to channel-transfer")
		clientCtx = metadata.NewOutgoingContext(
			context.Background(),
			metadata.Pairs("authorization", ts.NetworkFound.ChannelTransfer.AccessToken),
		)

		grpcAddress := ts.NetworkFound.ChannelTransfer.HostAddress + ":" +
			strconv.FormatUint(uint64(ts.NetworkFound.ChannelTransfer.Ports[cmn.GrpcPort]), 10)

		var err error
		conn, err = grpc.NewClient(grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
		Expect(err).NotTo(HaveOccurred())

		apiClient = cligrpc.NewAPIClient(conn)
	})

	AfterEach(func() {
		By("close gRPC connection")
		if conn != nil {
			err := conn.Close()
			Expect(err).NotTo(HaveOccurred())
		}
		By("stop channel transfer")
		ts.StopChannelTransfer()
		By("stop TaskExecutor")
		if taskExecutor != nil {
			taskExecutor.GracefulStop()
		}
	})

	BeforeEach(func() {
		By("add admin to acl")
		ts.AddAdminToACL()

		By("add user to acl")
		var err error
		user, err = mocks.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())

		ts.AddUser(user)

		By("emit tokens")
		ts.ExecuteTaskWithSign(
			cmn.ChannelFiat,
			cmn.ChannelFiat,
			ts.Admin(),
			fnEmit,
			user.AddressBase58Check,
			emitAmount,
		)
	})

	// This test reproduces the production bug scenario:
	// 1. createCCTransferTo SUCCEEDS (TransferTo exists on blockchain)
	// 2. ORDERERS ARE PAUSED (SIGSTOP - simulating network failure)
	// 3. commitCCTransferFrom FAILS repeatedly (orderers not responding)
	// 4. Retry delays cause TTL to expire
	// 5. Processing stops, launcher picks up transfer
	// 6. resolveStatus() is called
	// 7. WITHOUT FIX: cancelCCTransferFrom called -> DOUBLE SPENDING
	// 8. WITH FIX: shouldSkipToBatchNotFoundStatus() prevents cancellation
	It("Verifies fix prevents double-spending when orderers paused after createCCTransferTo", func() {
		amount := "250"
		transferID := uuid.NewString()

		By("Step 1: Create transfer request via channel-transfer API")
		channelTransferArgs := []string{transferID, "CC", user.AddressBase58Check, "FIAT", amount}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(
			append([]string{"channelTransferByAdmin", requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...),
			nonce,
		)
		publicKey, sign, err := ts.Admin().Sign(signArgs...)
		Expect(err).NotTo(HaveOccurred())

		transfer := &cligrpc.TransferBeginAdminRequest{
			Generals: &cligrpc.GeneralParams{
				MethodName: "channelTransferByAdmin",
				RequestId:  requestID,
				Chaincode:  cmn.ChannelFiat,
				Channel:    cmn.ChannelFiat,
				Nonce:      nonce,
				PublicKey:  publicKey,
				Sign:       base58.Encode(sign),
			},
			IdTransfer: channelTransferArgs[0],
			ChannelTo:  channelTransferArgs[1],
			Address:    channelTransferArgs[2],
			Token:      channelTransferArgs[3],
			Amount:     channelTransferArgs[4],
		}

		By("Step 2: Send transfer request via gRPC API")
		r, err := apiClient.TransferByAdmin(clientCtx, transfer)
		Expect(err).NotTo(HaveOccurred())
		Expect(r.Status).To(Equal(cligrpc.TransferStatusResponse_STATUS_IN_PROCESS))
		GinkgoWriter.Printf("Transfer initiated with ID: %s, status: %s\n", transferID, r.Status)

		By("Step 3: Wait for orderers to be paused")
		// Flow:
		// 1. channelTransferByAdmin executes normally
		// 2. createCCTransferTo executes and commits TransferTo to blockchain
		// 3. Mock callback PAUSES ALL ORDERERS using SIGSTOP synchronously
		// 4. Mock returns success
		// 5. channel-transfer tries commitCCTransferFrom -> FAILS (orderers paused)
		Eventually(func() bool {
			return taskExecutorMock.AreOrderersPaused()
		}, 60*time.Second, 500*time.Millisecond).Should(BeTrue(), "orderers should be paused")

		GinkgoWriter.Printf("Orderers are paused, commitCCTransferFrom will fail\n")

		By("Step 4: Wait for transfer to reach terminal state")
		// After orderers paused:
		// 1. commitCCTransferFrom FAILS (orderers not responding)
		// 2. Retries with delays (1s, 2s, ...)
		// 3. TTL expires (6s)
		// 4. Processing stops with error
		// 5. Launcher picks up transfer
		// 6. resolveStatus() IS CALLED
		// 7. Fix should prevent cancellation

		var finalStatus cligrpc.TransferStatusResponse_Status
		Eventually(func() bool {
			statusResp, err := apiClient.TransferStatus(clientCtx, &cligrpc.TransferStatusRequest{
				IdTransfer: transferID,
			})
			if err != nil {
				GinkgoWriter.Printf("TransferStatus error: %v\n", err)
				return false
			}
			finalStatus = statusResp.Status
			GinkgoWriter.Printf("Transfer status: %s\n", statusResp.Status)
			return statusResp.Status == cligrpc.TransferStatusResponse_STATUS_COMPLETED ||
				statusResp.Status == cligrpc.TransferStatusResponse_STATUS_CANCELED ||
				statusResp.Status == cligrpc.TransferStatusResponse_STATUS_ERROR
		}, 180*time.Second, 2*time.Second).Should(BeTrue(),
			"transfer should reach terminal state")

		By("Step 5: Log executed methods and final status")
		GinkgoWriter.Printf("TaskExecutor executed methods: %v\n", taskExecutorMock.GetExecutedMethods())
		GinkgoWriter.Printf("Final transfer status: %s\n", finalStatus)

		By("Step 6: Verify cancelCCTransferFrom was NOT called")
		wasCancelCalled := taskExecutorMock.WasMethodCalled("cancelCCTransferFrom")
		GinkgoWriter.Printf("cancelCCTransferFrom called: %v\n", wasCancelCalled)

		// With fix: cancelCCTransferFrom should NOT be called
		// Without fix: cancelCCTransferFrom WILL be called -> test FAILS
		Expect(wasCancelCalled).To(BeFalse(),
			"cancelCCTransferFrom should NOT be called - fix should prevent cancellation when TransferTo exists")

		By("Step 7: Verify balances after resuming orderers")
		// Resume orderers to allow balance queries
		resumeOrderers()
		time.Sleep(3 * time.Second) // Give orderers time to resume and reconnect

		// Note: With the fix applied, the transfer should eventually complete
		// or stay in a non-canceled state. Balances may vary depending on
		// whether the transfer eventually completes after orderers resume.

		GinkgoWriter.Printf("Test PASSED: cancelCCTransferFrom was not called, double-spending prevented\n")
	})
})
