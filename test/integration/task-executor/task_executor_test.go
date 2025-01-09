package task_executor

import (
	"context"
	"strconv"
	"time"

	cligrpc "github.com/anoideaopen/channel-transfer/proto"
	"github.com/anoideaopen/foundation/mocks"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	"github.com/hyperledger/fabric/integration"
	"github.com/hyperledger/fabric/integration/nwo"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/typepb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var _ = Describe("Channel transfer with task executor tests", func() {
	var (
		channels     = []string{cmn.ChannelACL, cmn.ChannelCC, cmn.ChannelFiat}
		ts           *client.FoundationTestSuite
		taskExecutor *grpc.Server
		network      *nwo.Network
		networkFound *cmn.NetworkFoundation
		clientCtx    context.Context
		apiClient    cligrpc.APIClient
		conn         *grpc.ClientConn
		user         *mocks.UserFoundation
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
		user, err = mocks.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())

		ts.AddUser(user)

		networkFound = ts.NetworkFound
		network = ts.Network
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

	It("transfer with insufficient balance", func() {
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

		ts.ExecuteTaskWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
			"channelTransferByAdmin", transferID, "CC", user.AddressBase58Check, "FIAT", "250",
		)

		// let transaction settle
		time.Sleep(time.Second * 5)

		By("checking transfer status")
		transferStatusRequest := &cligrpc.TransferStatusRequest{
			IdTransfer: transferID,
		}

		excludeStatus := cligrpc.TransferStatusResponse_STATUS_IN_PROCESS.String()
		value, err := anypb.New(wrapperspb.String(excludeStatus))
		Expect(err).NotTo(HaveOccurred())

		transferStatusRequest.Options = append(transferStatusRequest.Options, &typepb.Option{
			Name:  "excludeStatus",
			Value: value,
		})

		ctx, cancel := context.WithTimeout(clientCtx, network.EventuallyTimeout*2)
		defer cancel()

		By("awaiting for channel transfer to respond")
		statusResponse, err := apiClient.TransferStatus(ctx, transferStatusRequest)
		Expect(err).NotTo(HaveOccurred())
		Expect(statusResponse.Status).To(Equal(cligrpc.TransferStatusResponse_STATUS_ERROR))
		Expect(statusResponse.Message).To(ContainSubstring("insufficient balance"))
	})
})
