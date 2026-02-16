package task_executor

import (
	"context"
	"fmt"
	"strconv"
	"time"

	cligrpc "github.com/anoideaopen/channel-transfer/proto"
	"github.com/anoideaopen/foundation/mocks"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
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

var _ = Describe("Channel transfer with task executor transaction tests", func() {
	var (
		ts *client.FoundationTestSuite

		channels = []string{cmn.ChannelACL, cmn.ChannelCC, cmn.ChannelFiat}
		user     *mocks.UserFoundation

		clientCtx    context.Context
		apiClient    cligrpc.APIClient
		conn         *grpc.ClientConn
		network      *nwo.Network
		networkFound *cmn.NetworkFoundation

		taskExecutor *grpc.Server
	)

	BeforeEach(func() {
		ts = client.NewTestSuite(components)

		By("start redis")
		ts.StartRedis()

		ts.InitNetwork(
			channels,
			integration.MSPPort,
		)
		ts.DeployChaincodes()

		By("start robot")
		ts.StartRobot()

		By("add admin to acl")
		ts.AddAdminToACL()

		By("add user to acl")
		var err error
		user, err = mocks.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())

		ts.AddUser(user)

		networkFound = ts.NetworkFound
		network = ts.Network

		By("start taskExecutor")
		taskExecutor = StartTaskExecutor()
		By("start channel transfer")
		ts.StartChannelTransfer()
	})
	AfterEach(func() {
		By("stop channel transfer")
		ts.StopChannelTransfer()
		By("stop taskExecutor")
		StopTaskExecutor(taskExecutor)
		By("stop robot")
		ts.StopRobot()
		By("stop redis")
		ts.StopRedis()

		ts.ShutdownNetwork()
	})

	It("transfer created not with channel transfer service", func() {
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

		By("emit tokens")
		emitAmount := "1000"
		ts.ExecuteTaskWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
			"emit", user.AddressBase58Check, emitAmount)

		By("emit check")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
			"balanceOf", user.AddressBase58Check).CheckBalance(emitAmount)

		By("creating channel transfer request")
		transferID := uuid.NewString()

		ts.ExecuteTaskWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
			"channelTransferByAdmin", transferID, "CC", user.AddressBase58Check, "FIAT", "250",
		)

		// let transaction settle
		time.Sleep(time.Second)

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
		Expect(cligrpc.TransferStatusResponse_STATUS_COMPLETED).To(Equal(statusResponse.Status))

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
			"balanceOf", user.AddressBase58Check).CheckBalance("750")

		ts.Query(cmn.ChannelCC, cmn.ChannelCC,
			"allowedBalanceOf", user.AddressBase58Check, "FIAT").CheckBalance("250")
	})

	It("several transfers in one batch", func() {
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

		var (
			emitAmount     = "1000"
			transfersCount = 100
			users          = make([]*mocks.UserFoundation, 0, transfersCount)
			transferIDs    = make([]string, 0, transfersCount)
			tasks          = make([]*pbfound.Task, 0, transfersCount)
		)

		for range transfersCount {
			By("add user to acl")
			user, err = mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			Expect(err).NotTo(HaveOccurred())

			ts.AddUser(user)

			users = append(users, user)

			By("emit tokens")
			ts.ExecuteTaskWithSign(cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
				"emit", user.AddressBase58Check, emitAmount)

			By("emit check")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
				"balanceOf", user.AddressBase58Check).CheckBalance(emitAmount)

			By("creating channel transfer request")
			transferID := uuid.NewString()
			transferIDs = append(transferIDs, transferID)

			By("creating new transfer task")
			task, err := client.CreateTaskWithSignArgs(
				"channelTransferByAdmin",
				cmn.ChannelFiat, cmn.ChannelFiat,
				ts.Admin(),
				transferID,
				"CC",
				user.AddressBase58Check,
				"FIAT",
				"250",
			)
			Expect(err).NotTo(HaveOccurred())

			tasks = append(tasks, task)
		}

		ts.ExecuteTasks(cmn.ChannelFiat, cmn.ChannelFiat, tasks...)

		// let transaction settle
		time.Sleep(time.Second)

		ctx, cancel := context.WithTimeout(clientCtx, network.EventuallyTimeout*2)
		defer cancel()

		for i, transferID := range transferIDs {
			By(fmt.Sprintf("checking transfer status %d", i))
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

			By("awaiting for channel transfer to respond")
			statusResponse, err := apiClient.TransferStatus(ctx, transferStatusRequest)
			Expect(err).NotTo(HaveOccurred())
			Expect(cligrpc.TransferStatusResponse_STATUS_COMPLETED).To(Equal(statusResponse.Status))

			By("checking result balances")
			ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
				"balanceOf", users[i].AddressBase58Check).CheckBalance("750")

			ts.Query(cmn.ChannelCC, cmn.ChannelCC,
				"allowedBalanceOf", users[i].AddressBase58Check, "FIAT").CheckBalance("250")
		}
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
		time.Sleep(time.Second)

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
