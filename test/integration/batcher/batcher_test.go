package batcher

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	cligrpc "github.com/anoideaopen/channel-transfer/proto"
	"github.com/anoideaopen/channel-transfer/test/integration/patch"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	"github.com/hyperledger/fabric/integration"
	"github.com/hyperledger/fabric/integration/nwo"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	ginkgomon "github.com/tedsuo/ifrit/ginkgomon_v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

var _ = Describe("Channel transfer with batcher GRPC tests", func() {
	var (
		channels     = []string{cmn.ChannelAcl, cmn.ChannelCC, cmn.ChannelFiat}
		ts           client.TestSuite
		mockBatcher  ifrit.Process
		networkFound *cmn.NetworkFoundation
		clientCtx    context.Context
		apiClient    cligrpc.APIClient
		conn         *grpc.ClientConn
		user         *client.UserFoundation
	)

	BeforeEach(func() {
		ts = client.NewTestSuite(components)
	})
	AfterEach(func() {
		ts.ShutdownNetwork()
	})

	BeforeEach(func() {
		By("start redis")
		ts.StartRedis()
	})

	BeforeEach(func() {
		ts.InitNetwork(channels, integration.LedgerPort)
		ts.DeployChaincodes()

		By("add admin to acl")
		ts.AddAdminToACL()

		By("add user to acl")
		var err error
		user, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())

		ts.AddUser(user)

		networkFound = ts.NetworkFound()
		patch.ChannelTransferConfigWithBatcher(networkFound, channels, mockBatcherPort())
	})

	BeforeEach(func() {
		By("start mock batcher")
		mockBatcher = startMockBatcher(components)
		By("start channel transfer")
		ts.StartChannelTransfer()
	})

	AfterEach(func() {
		By("stop redis")
		ts.StopRedis()
		By("stop channel transfer")
		ts.StopChannelTransfer()
		By("stop mock batcher")
		stopMockBatcher(mockBatcher)
	})

	It("Submit transaction", func() {
		By("creating grpc connection")
		clientCtx = metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", networkFound.ChannelTransfer.AccessToken))

		transportCredentials := insecure.NewCredentials()
		grpcAddress := networkFound.ChannelTransfer.HostAddress + ":" + strconv.FormatUint(uint64(networkFound.ChannelTransfer.Ports[cmn.GrpcPort]), 10)

		var err error

		conn, err = grpc.NewClient(grpcAddress, grpc.WithTransportCredentials(transportCredentials))
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			err := conn.Close()
			Expect(err).NotTo(HaveOccurred())
		}()

		By("creating channel transfer API client")
		apiClient = cligrpc.NewAPIClient(conn)

		By("creating channel transfer request")
		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, "CC", user.AddressBase58Check, "FIAT", "250"}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{"channelTransferByAdmin", requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
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

		By("sending transfer request")
		r, err := apiClient.TransferByAdmin(clientCtx, transfer)
		Expect(err).NotTo(HaveOccurred())
		Expect(r.Status).To(Equal(cligrpc.TransferStatusResponse_STATUS_IN_PROCESS))
	})
})

func startMockBatcher(components *nwo.Components) ifrit.Process {
	mockBatcherProcess := ifrit.Invoke(mockBatcherRunner(components))
	Eventually(mockBatcherProcess.Ready(), time.Minute).Should(BeClosed())
	return mockBatcherProcess
}

func stopMockBatcher(mockBatcher ifrit.Process) {
	if mockBatcher != nil {
		mockBatcher.Signal(syscall.SIGTERM)
		Eventually(mockBatcher.Wait(), time.Minute).Should(Receive())
	}
}

func mockBatcherRunner(components *nwo.Components) *ginkgomon.Runner {
	cmd := exec.Command(components.Build(mockBatcherModulePath()), "--port", mockBatcherPort())
	return ginkgomon.New(ginkgomon.Config{
		AnsiColorCode:     "yellow",
		Name:              "Mock Batcher",
		Command:           cmd,
		StartCheck:        "listening on port",
		StartCheckTimeout: 15 * time.Second,
	})
}

func mockBatcherModulePath() string {
	return "github.com/anoideaopen/channel-transfer/test/mock/batcher"
}

func mockBatcherPort() string {
	return fmt.Sprintf("%d", integration.LifecyclePort)
}
