package task_executor

import (
	"context"
	"strconv"

	cligrpc "github.com/anoideaopen/channel-transfer/proto"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	"github.com/hyperledger/fabric/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

var _ = Describe("Channel transfer with task executor tests", func() {
	var (
		channels     = []string{cmn.ChannelAcl, cmn.ChannelCC, cmn.ChannelFiat}
		ts           client.TestSuite
		taskExecutor *grpc.Server
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
		ts.InitNetwork(
			channels,
			integration.LedgerPort,
			client.WithTaskExecutorForChannels(taskExecutorHost(), taskExecutorPorts(), cmn.ChannelCC, cmn.ChannelFiat),
		)
		ts.DeployChaincodes()

		By("add admin to acl")
		ts.AddAdminToACL()

		By("add user to acl")
		var err error
		user, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())

		ts.AddUser(user)

		networkFound = ts.NetworkFound()
	})

	BeforeEach(func() {
		By("start taskExecutor")
		taskExecutor = StartTaskExecutor()
		By("start channel transfer")
		ts.StartChannelTransfer()
	})

	AfterEach(func() {
		By("stop redis")
		ts.StopRedis()
		By("stop channel transfer")
		ts.StopChannelTransfer()
		By("stop taskExecutor")
		StopTaskExecutor(taskExecutor)
	})

	It("Submit transaction", func() {
		By("creating grpc connection")
		clientCtx = metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", networkFound.ChannelTransfer.AccessToken))

		transportCredentials := insecure.NewCredentials()
		grpcAddress := networkFound.ChannelTransfer.HostAddress + ":" + strconv.FormatUint(uint64(networkFound.ChannelTransfer.Ports[cmn.GrpcPort]), 10)

		var err error

		newTraceID, _ := trace.TraceIDFromHex("4f92f5c8b5a43a25a891c94b0c0f5a42")
		newSpanID, _ := trace.SpanIDFromHex("1e39285f6c1b41f4")

		clientCtx = metadata.AppendToOutgoingContext(clientCtx, "trace_id", newTraceID.String())
		clientCtx = metadata.AppendToOutgoingContext(clientCtx, "span_id", newSpanID.String())

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
