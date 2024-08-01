package http

import (
	"context"
	"fmt"
	clihttp "github.com/anoideaopen/channel-transfer/test/integration/clihttp/client"
	"github.com/anoideaopen/channel-transfer/test/integration/clihttp/client/transfer"
	"github.com/anoideaopen/channel-transfer/test/integration/clihttp/models"
	"github.com/go-openapi/runtime"
	"strings"
	"time"
)

// import (
//
//	"context"
//	"fmt"
//	"os"
//	"path/filepath"
//	"strconv"
//	"strings"
//	"syscall"
//	"time"
//
//	clihttp "github.com/anoideaopen/channel-transfer/test/integration/clihttp/client"
//	"github.com/anoideaopen/channel-transfer/test/integration/clihttp/client/transfer"
//	"github.com/anoideaopen/channel-transfer/test/integration/clihttp/models"
//	pbfound "github.com/anoideaopen/foundation/proto"
//	"github.com/anoideaopen/foundation/test/integration/cmn"
//	"github.com/anoideaopen/foundation/test/integration/cmn/client"
//	"github.com/anoideaopen/foundation/test/integration/cmn/fabricnetwork"
//	"github.com/anoideaopen/foundation/test/integration/cmn/runner"
//	"github.com/btcsuite/btcutil/base58"
//	docker "github.com/fsouza/go-dockerclient"
//	"github.com/go-openapi/runtime"
//	httptransport "github.com/go-openapi/runtime/client"
//	"github.com/go-openapi/strfmt"
//	"github.com/google/uuid"
//	"github.com/hyperledger/fabric/integration/nwo"
//	"github.com/hyperledger/fabric/integration/nwo/fabricconfig"
//	runnerFbk "github.com/hyperledger/fabric/integration/nwo/runner"
//	. "github.com/onsi/ginkgo/v2"
//	. "github.com/onsi/gomega"
//	"github.com/onsi/gomega/gbytes"
//	"github.com/tedsuo/ifrit"
//	ginkgomon "github.com/tedsuo/ifrit/ginkgomon_v2"
//	"google.golang.org/grpc/metadata"
//
// )
const (
	transferExecutionTimeout = 100

	ccCCUpper         = "CC"
	ccFiatUpper       = "FIAT"
	ccACLUpper        = "ACL"
	ccIndustrialUpper = "INDUSTRIAL"

	emitAmount = "1000"

	errWrongChannel      = "no channel peers configured for channel"
	errNotValidChannel   = "channel not in configuration list"
	errInsufficientFunds = "failed to subtract token balance: insufficient balance"
	errIncorrectToken    = "token set incorrectly"

	fnChannelTransferByAdmin    = "channelTransferByAdmin"
	fnChannelTransferByCustomer = "channelTransferByCustomer"
)

//
//var _ = Describe("Channel transfer HTTP tests", func() {
//	var (
//		testDir          string
//		cli              *docker.Client
//		network          *nwo.Network
//		networkProcess   ifrit.Process
//		ordererProcesses []ifrit.Process
//		peerProcesses    ifrit.Process
//	)
//
//	BeforeEach(func() {
//		networkProcess = nil
//		ordererProcesses = nil
//		peerProcesses = nil
//		var err error
//		testDir, err = os.MkdirTemp("", "foundation")
//		Expect(err).NotTo(HaveOccurred())
//
//		cli, err = docker.NewClientFromEnv()
//		Expect(err).NotTo(HaveOccurred())
//	})
//
//	AfterEach(func() {
//		if networkProcess != nil {
//			networkProcess.Signal(syscall.SIGTERM)
//			Eventually(networkProcess.Wait(), network.EventuallyTimeout).Should(Receive())
//		}
//		if peerProcesses != nil {
//			peerProcesses.Signal(syscall.SIGTERM)
//			Eventually(peerProcesses.Wait(), network.EventuallyTimeout).Should(Receive())
//		}
//		if network != nil {
//			network.Cleanup()
//		}
//		for _, ordererInstance := range ordererProcesses {
//			ordererInstance.Signal(syscall.SIGTERM)
//			Eventually(ordererInstance.Wait(), network.EventuallyTimeout).Should(Receive())
//		}
//		err := os.RemoveAll(testDir)
//		Expect(err).NotTo(HaveOccurred())
//	})
//
//	var (
//		channels            = []string{cmn.ChannelAcl, cmn.ChannelCC, cmn.ChannelFiat, cmn.ChannelIndustrial}
//		ordererRunners      []*ginkgomon.Runner
//		redisProcess        ifrit.Process
//		redisDB             *runner.RedisDB
//		networkFound        *cmn.NetworkFoundation
//		robotProc           ifrit.Process
//		channelTransferProc ifrit.Process
//		skiBackend          string
//		skiRobot            string
//		peer                *nwo.Peer
//		admin               *client.UserFoundation
//		user                *client.UserFoundation
//		feeSetter           *client.UserFoundation
//		feeAddressSetter    *client.UserFoundation
//
//		clientCtx   context.Context
//		transferCli *clihttp.CrossChanelTransfer
//		auth        runtime.ClientAuthInfoWriter
//	)
//	BeforeEach(func() {
//		By("start redis")
//		redisDB = &runner.RedisDB{}
//		redisProcess = ifrit.Invoke(redisDB)
//		Eventually(redisProcess.Ready(), runnerFbk.DefaultStartTimeout).Should(BeClosed())
//		Consistently(redisProcess.Wait()).ShouldNot(Receive())
//	})
//	BeforeEach(func() {
//		networkConfig := nwo.MultiNodeSmartBFT()
//		networkConfig.Channels = nil
//
//		pchs := make([]*nwo.PeerChannel, 0, cap(channels))
//		for _, ch := range channels {
//			pchs = append(pchs, &nwo.PeerChannel{
//				Name:   ch,
//				Anchor: true,
//			})
//		}
//		for _, peer := range networkConfig.Peers {
//			peer.Channels = pchs
//		}
//
//		network = nwo.New(networkConfig, testDir, cli, StartPort(), components)
//		cwd, err := os.Getwd()
//		Expect(err).NotTo(HaveOccurred())
//		network.ExternalBuilders = append(network.ExternalBuilders,
//			fabricconfig.ExternalBuilder{
//				Path:                 filepath.Join(cwd, ".", "externalbuilders", "binary"),
//				Name:                 "binary",
//				PropagateEnvironment: []string{"GOPROXY"},
//			},
//		)
//
//		networkFound = cmn.New(network, channels)
//		networkFound.Robot.RedisAddresses = []string{redisDB.Address()}
//		networkFound.ChannelTransfer.RedisAddresses = []string{redisDB.Address()}
//
//		networkFound.GenerateConfigTree()
//		networkFound.Bootstrap()
//
//		for _, orderer := range network.Orderers {
//			runner := network.OrdererRunner(orderer)
//			runner.Command.Env = append(runner.Command.Env, "FABRIC_LOGGING_SPEC=orderer.consensus.smartbft=debug:grpc=debug")
//			ordererRunners = append(ordererRunners, runner)
//			proc := ifrit.Invoke(runner)
//			ordererProcesses = append(ordererProcesses, proc)
//			Eventually(proc.Ready(), network.EventuallyTimeout).Should(BeClosed())
//		}
//
//		peerGroupRunner, _ := fabricnetwork.PeerGroupRunners(network)
//		peerProcesses = ifrit.Invoke(peerGroupRunner)
//		Eventually(peerProcesses.Ready(), network.EventuallyTimeout).Should(BeClosed())
//
//		By("Joining orderers to channels")
//		for _, channel := range channels {
//			fabricnetwork.JoinChannel(network, channel)
//		}
//
//		By("Waiting for followers to see the leader")
//		Eventually(ordererRunners[1].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))
//		Eventually(ordererRunners[2].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))
//		Eventually(ordererRunners[3].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))
//
//		By("Joining peers to channels")
//		for _, channel := range channels {
//			network.JoinChannel(channel, network.Orderers[0], network.PeersWithChannel(channel)...)
//		}
//
//		peer = network.Peer("Org1", "peer0")
//
//		pathToPrivateKeyBackend := network.PeerUserKey(peer, "User1")
//		skiBackend, err = cmn.ReadSKI(pathToPrivateKeyBackend)
//		Expect(err).NotTo(HaveOccurred())
//
//		pathToPrivateKeyRobot := network.PeerUserKey(peer, "User2")
//		skiRobot, err = cmn.ReadSKI(pathToPrivateKeyRobot)
//		Expect(err).NotTo(HaveOccurred())
//
//		admin, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
//		Expect(err).NotTo(HaveOccurred())
//		Expect(admin.PrivateKeyBytes).NotTo(Equal(nil))
//
//		feeSetter, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
//		Expect(err).NotTo(HaveOccurred())
//		Expect(feeSetter.PrivateKeyBytes).NotTo(Equal(nil))
//
//		feeAddressSetter, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
//		Expect(err).NotTo(HaveOccurred())
//		Expect(feeAddressSetter.PrivateKeyBytes).NotTo(Equal(nil))
//
//		cmn.DeployACL(network, components, peer, testDir, skiBackend, admin.PublicKeyBase58, admin.KeyType)
//		cmn.DeployCC(network, components, peer, testDir, skiRobot, admin.AddressBase58Check)
//		cmn.DeployFiat(network, components, peer, testDir, skiRobot,
//			admin.AddressBase58Check, feeSetter.AddressBase58Check, feeAddressSetter.AddressBase58Check)
//		cmn.DeployIndustrial(network, components, peer, testDir, skiRobot,
//			admin.AddressBase58Check, feeSetter.AddressBase58Check, feeAddressSetter.AddressBase58Check)
//	})
//	BeforeEach(func() {
//		By("start robot")
//		robotRunner := networkFound.RobotRunner()
//		robotProc = ifrit.Invoke(robotRunner)
//		Eventually(robotProc.Ready(), network.EventuallyTimeout).Should(BeClosed())
//
//		By("start channel transfer")
//		channelTransferRunner := networkFound.ChannelTransferRunner()
//		channelTransferProc = ifrit.Invoke(channelTransferRunner)
//		Eventually(channelTransferProc.Ready(), network.EventuallyTimeout).Should(BeClosed())
//	})
//	AfterEach(func() {
//		By("stop robot")
//		if robotProc != nil {
//			robotProc.Signal(syscall.SIGTERM)
//			Eventually(robotProc.Wait(), network.EventuallyTimeout).Should(Receive())
//		}
//
//		By("stop channel transfer")
//		if channelTransferProc != nil {
//			channelTransferProc.Signal(syscall.SIGTERM)
//			Eventually(channelTransferProc.Wait(), network.EventuallyTimeout).Should(Receive())
//		}
//	})
//
//	BeforeEach(func() {
//		By("add admin to acl")
//		client.AddUser(network, peer, network.Orderers[0], admin)
//
//		By("add user to acl")
//		var err error
//		user, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
//		Expect(err).NotTo(HaveOccurred())
//
//		client.AddUser(network, peer, network.Orderers[0], user)
//
//		By("emit tokens")
//		client.TxInvokeWithSign(network, peer, network.Orderers[0],
//			cmn.ChannelFiat, cmn.ChannelFiat, admin,
//			"emit", "", client.NewNonceByTime().Get(), nil, user.AddressBase58Check, emitAmount)
//
//		By("emit check")
//		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
//			"balanceOf", user.AddressBase58Check)
//
//		By("creating http connection")
//		clientCtx = metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", networkFound.ChannelTransfer.AccessToken))
//
//		httpAddress := networkFound.ChannelTransfer.HostAddress + ":" + strconv.FormatUint(uint64(networkFound.ChannelTransfer.Ports[cmn.HttpPort]), 10)
//		transport := httptransport.New(httpAddress, "", nil)
//		transferCli = clihttp.New(transport, strfmt.Default)
//
//		auth = httptransport.APIKeyAuth("authorization", "header", networkFound.ChannelTransfer.AccessToken)
//	})
//
//	It("transfer by admin test", func() {
//		amount := "250"
//		restAmount := "750"
//
//		authOpts := func(c *runtime.ClientOperation) {
//			c.AuthInfo = auth
//		}
//
//		By("creating channel transfer request")
//		transferID := uuid.NewString()
//		channelTransferArgs := []string{transferID, ccCCUpper, user.AddressBase58Check, ccFiatUpper, amount}
//
//		requestID := uuid.NewString()
//		nonce := client.NewNonceByTime().Get()
//		signArgs := append(append([]string{fnChannelTransferByAdmin, requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
//		publicKey, sign, err := admin.Sign(signArgs...)
//		Expect(err).NotTo(HaveOccurred())
//
//		transferRequest := &models.ChannelTransferTransferBeginAdminRequest{
//			Generals: &models.ChannelTransferGeneralParams{
//				MethodName: fnChannelTransferByAdmin,
//				RequestID:  requestID,
//				Chaincode:  cmn.ChannelFiat,
//				Channel:    cmn.ChannelFiat,
//				Nonce:      nonce,
//				PublicKey:  publicKey,
//				Sign:       base58.Encode(sign),
//			},
//			IDTransfer: channelTransferArgs[0],
//			ChannelTo:  channelTransferArgs[1],
//			Address:    channelTransferArgs[2],
//			Token:      channelTransferArgs[3],
//			Amount:     channelTransferArgs[4],
//		}
//
//		By("sending transfer request")
//		res, err := transferCli.Transfer.TransferByAdmin(&transfer.TransferByAdminParams{Body: transferRequest, Context: clientCtx}, authOpts)
//		Expect(err).NotTo(HaveOccurred())
//
//		err = checkResponseStatus(res.GetPayload(), models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, "")
//		Expect(err).NotTo(HaveOccurred())
//
//		By("awaiting for channel transfer to respond")
//		err = waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED, "")
//		Expect(err).NotTo(HaveOccurred())
//
//		By("checking result balances")
//		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(restAmount), nil),
//			"balanceOf", user.AddressBase58Check)
//
//		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amount), nil),
//			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
//	})
//
//	It("transfer by customer test", func() {
//		amount := "250"
//		restAmount := "750"
//
//		authOpts := func(c *runtime.ClientOperation) {
//			c.AuthInfo = auth
//		}
//
//		By("creating channel transfer request")
//		transferID := uuid.NewString()
//		channelTransferArgs := []string{transferID, ccCCUpper, ccFiatUpper, amount}
//
//		requestID := uuid.NewString()
//		nonce := client.NewNonceByTime().Get()
//		signArgs := append(append([]string{fnChannelTransferByCustomer, requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
//		publicKey, sign, err := user.Sign(signArgs...)
//		Expect(err).NotTo(HaveOccurred())
//
//		transferRequest := &models.ChannelTransferTransferBeginCustomerRequest{
//			Generals: &models.ChannelTransferGeneralParams{
//				MethodName: fnChannelTransferByCustomer,
//				RequestID:  requestID,
//				Chaincode:  cmn.ChannelFiat,
//				Channel:    cmn.ChannelFiat,
//				Nonce:      nonce,
//				PublicKey:  publicKey,
//				Sign:       base58.Encode(sign),
//			},
//			IDTransfer: channelTransferArgs[0],
//			ChannelTo:  channelTransferArgs[1],
//			Token:      channelTransferArgs[2],
//			Amount:     channelTransferArgs[3],
//		}
//
//		By("sending transfer request")
//		res, err := transferCli.Transfer.TransferByCustomer(&transfer.TransferByCustomerParams{Body: transferRequest, Context: clientCtx}, authOpts)
//		Expect(err).NotTo(HaveOccurred())
//
//		err = checkResponseStatus(res.GetPayload(), models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, "")
//		Expect(err).NotTo(HaveOccurred())
//
//		By("awaiting for channel transfer to respond")
//		err = waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED, "")
//		Expect(err).NotTo(HaveOccurred())
//
//		By("checking result balances")
//		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(restAmount), nil),
//			"balanceOf", user.AddressBase58Check)
//
//		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amount), nil),
//			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
//	})
//
//	It("transfer status with wrong transfer id test", func() {
//		authOpts := func(c *runtime.ClientOperation) {
//			c.AuthInfo = auth
//		}
//
//		By("requesting status of transfer with id = 1")
//		_, err := transferCli.Transfer.TransferStatus(&transfer.TransferStatusParams{IDTransfer: "1", Context: context.Background()}, authOpts)
//		Expect(err).To(MatchError(ContainSubstring("object not found")))
//	})
//
//	It("admin transfer to wrong channel", func() {
//		amount := "250"
//		sendAmount := "0"
//		wrongChannel := "ASD"
//
//		authOpts := func(c *runtime.ClientOperation) {
//			c.AuthInfo = auth
//		}
//
//		By("creating channel transfer request")
//		transferID := uuid.NewString()
//		channelTransferArgs := []string{transferID, wrongChannel, user.AddressBase58Check, ccFiatUpper, amount}
//
//		requestID := uuid.NewString()
//		nonce := client.NewNonceByTime().Get()
//		signArgs := append(append([]string{fnChannelTransferByAdmin, requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
//		publicKey, sign, err := admin.Sign(signArgs...)
//		Expect(err).NotTo(HaveOccurred())
//
//		transferRequest := &models.ChannelTransferTransferBeginAdminRequest{
//			Generals: &models.ChannelTransferGeneralParams{
//				MethodName: fnChannelTransferByAdmin,
//				RequestID:  requestID,
//				Chaincode:  cmn.ChannelFiat,
//				Channel:    cmn.ChannelFiat,
//				Nonce:      nonce,
//				PublicKey:  publicKey,
//				Sign:       base58.Encode(sign),
//			},
//			IDTransfer: channelTransferArgs[0],
//			ChannelTo:  channelTransferArgs[1],
//			Address:    channelTransferArgs[2],
//			Token:      channelTransferArgs[3],
//			Amount:     channelTransferArgs[4],
//		}
//
//		By("sending transfer request")
//		res, err := transferCli.Transfer.TransferByAdmin(&transfer.TransferByAdminParams{Body: transferRequest, Context: clientCtx}, authOpts)
//		Expect(err).NotTo(HaveOccurred())
//
//		err = checkResponseStatus(res.GetPayload(), models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, "")
//		Expect(err).NotTo(HaveOccurred())
//
//		By("awaiting for channel transfer to respond")
//		err = waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSERROR, errWrongChannel)
//		Expect(err).NotTo(HaveOccurred())
//
//		By("checking result balances")
//		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
//			"balanceOf", user.AddressBase58Check)
//
//		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(sendAmount), nil),
//			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
//	})
//
//	It("client transfer to wrong channel", func() {
//		amount := "250"
//		sendAmount := "0"
//		wrongChannel := "ASD"
//
//		authOpts := func(c *runtime.ClientOperation) {
//			c.AuthInfo = auth
//		}
//
//		By("creating channel transfer request")
//		transferID := uuid.NewString()
//		channelTransferArgs := []string{transferID, wrongChannel, ccFiatUpper, amount}
//
//		requestID := uuid.NewString()
//		nonce := client.NewNonceByTime().Get()
//		signArgs := append(append([]string{fnChannelTransferByCustomer, requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
//		publicKey, sign, err := user.Sign(signArgs...)
//		Expect(err).NotTo(HaveOccurred())
//
//		transferRequest := &models.ChannelTransferTransferBeginCustomerRequest{
//			Generals: &models.ChannelTransferGeneralParams{
//				MethodName: fnChannelTransferByCustomer,
//				RequestID:  requestID,
//				Chaincode:  cmn.ChannelFiat,
//				Channel:    cmn.ChannelFiat,
//				Nonce:      nonce,
//				PublicKey:  publicKey,
//				Sign:       base58.Encode(sign),
//			},
//			IDTransfer: channelTransferArgs[0],
//			ChannelTo:  channelTransferArgs[1],
//			Token:      channelTransferArgs[2],
//			Amount:     channelTransferArgs[3],
//		}
//
//		By("sending transfer request")
//		res, err := transferCli.Transfer.TransferByCustomer(&transfer.TransferByCustomerParams{Body: transferRequest, Context: clientCtx}, authOpts)
//		Expect(err).NotTo(HaveOccurred())
//
//		err = checkResponseStatus(res.GetPayload(), models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, "")
//		Expect(err).NotTo(HaveOccurred())
//
//		By("awaiting for channel transfer to respond")
//		err = waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSERROR, errWrongChannel)
//		Expect(err).NotTo(HaveOccurred())
//
//		By("checking result balances")
//		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
//			"balanceOf", user.AddressBase58Check)
//
//		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(sendAmount), nil),
//			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
//	})
//
//	It("admin transfer to not valid channel", func() {
//		amount := "250"
//		sendAmount := "0"
//
//		authOpts := func(c *runtime.ClientOperation) {
//			c.AuthInfo = auth
//		}
//
//		By("creating channel transfer request")
//		transferID := uuid.NewString()
//		channelTransferArgs := []string{transferID, ccACLUpper, user.AddressBase58Check, ccACLUpper, amount}
//
//		requestID := uuid.NewString()
//		nonce := client.NewNonceByTime().Get()
//		signArgs := append(append([]string{fnChannelTransferByAdmin, requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
//		publicKey, sign, err := admin.Sign(signArgs...)
//		Expect(err).NotTo(HaveOccurred())
//
//		transferRequest := &models.ChannelTransferTransferBeginAdminRequest{
//			Generals: &models.ChannelTransferGeneralParams{
//				MethodName: fnChannelTransferByAdmin,
//				RequestID:  requestID,
//				Chaincode:  cmn.ChannelAcl,
//				Channel:    cmn.ChannelAcl,
//				Nonce:      nonce,
//				PublicKey:  publicKey,
//				Sign:       base58.Encode(sign),
//			},
//			IDTransfer: channelTransferArgs[0],
//			ChannelTo:  ccFiatUpper,
//			Address:    channelTransferArgs[2],
//			Token:      channelTransferArgs[3],
//			Amount:     channelTransferArgs[4],
//		}
//
//		By("sending transfer request")
//		_, err = transferCli.Transfer.TransferByAdmin(&transfer.TransferByAdminParams{Body: transferRequest, Context: clientCtx}, authOpts)
//		Expect(err).Should(ContainSubstring(errNotValidChannel))
//
//		By("checking result balances")
//		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
//			"balanceOf", user.AddressBase58Check)
//
//		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(sendAmount), nil),
//			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
//	})
//
//	It("client transfer to not valid channel", func() {
//		amount := "250"
//		sendAmount := "0"
//
//		authOpts := func(c *runtime.ClientOperation) {
//			c.AuthInfo = auth
//		}
//
//		By("creating channel transfer request")
//		transferID := uuid.NewString()
//		channelTransferArgs := []string{transferID, ccACLUpper, ccACLUpper, amount}
//
//		requestID := uuid.NewString()
//		nonce := client.NewNonceByTime().Get()
//		signArgs := append(append([]string{fnChannelTransferByCustomer, requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
//		publicKey, sign, err := user.Sign(signArgs...)
//		Expect(err).NotTo(HaveOccurred())
//
//		transferRequest := &models.ChannelTransferTransferBeginCustomerRequest{
//			Generals: &models.ChannelTransferGeneralParams{
//				MethodName: fnChannelTransferByCustomer,
//				RequestID:  requestID,
//				Chaincode:  cmn.ChannelAcl,
//				Channel:    cmn.ChannelAcl,
//				Nonce:      nonce,
//				PublicKey:  publicKey,
//				Sign:       base58.Encode(sign),
//			},
//			IDTransfer: channelTransferArgs[0],
//			ChannelTo:  ccFiatUpper,
//			Token:      channelTransferArgs[2],
//			Amount:     channelTransferArgs[3],
//		}
//
//		By("sending transfer request")
//		_, err = transferCli.Transfer.TransferByCustomer(&transfer.TransferByCustomerParams{Body: transferRequest, Context: clientCtx}, authOpts)
//		Expect(err).Should(ContainSubstring(errNotValidChannel))
//
//		By("checking result balances")
//		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
//			"balanceOf", user.AddressBase58Check)
//
//		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(sendAmount), nil),
//			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
//	})
//
//	It("admin transfer insufficient funds", func() {
//		amount := "250"
//		sendAmount := "0"
//
//		authOpts := func(c *runtime.ClientOperation) {
//			c.AuthInfo = auth
//		}
//
//		By("creating new user")
//		user1, err := client.NewUserFoundation(pbfound.KeyType_ed25519)
//		Expect(err).NotTo(HaveOccurred())
//
//		client.AddUser(network, peer, network.Orderers[0], user1)
//
//		By("creating channel transfer request")
//		transferID := uuid.NewString()
//		channelTransferArgs := []string{transferID, ccCCUpper, user1.AddressBase58Check, ccFiatUpper, amount}
//
//		requestID := uuid.NewString()
//		nonce := client.NewNonceByTime().Get()
//		signArgs := append(append([]string{fnChannelTransferByAdmin, requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
//		publicKey, sign, err := admin.Sign(signArgs...)
//		Expect(err).NotTo(HaveOccurred())
//
//		transferRequest := &models.ChannelTransferTransferBeginAdminRequest{
//			Generals: &models.ChannelTransferGeneralParams{
//				MethodName: fnChannelTransferByAdmin,
//				RequestID:  requestID,
//				Chaincode:  cmn.ChannelFiat,
//				Channel:    cmn.ChannelFiat,
//				Nonce:      nonce,
//				PublicKey:  publicKey,
//				Sign:       base58.Encode(sign),
//			},
//			IDTransfer: channelTransferArgs[0],
//			ChannelTo:  channelTransferArgs[1],
//			Address:    channelTransferArgs[2],
//			Token:      channelTransferArgs[3],
//			Amount:     channelTransferArgs[4],
//		}
//
//		By("sending transfer request")
//		res, err := transferCli.Transfer.TransferByAdmin(&transfer.TransferByAdminParams{Body: transferRequest, Context: clientCtx}, authOpts)
//		Expect(err).NotTo(HaveOccurred())
//
//		err = checkResponseStatus(res.GetPayload(), models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, "")
//		Expect(err).NotTo(HaveOccurred())
//
//		By("awaiting for channel transfer to respond")
//		err = waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSERROR, errInsufficientFunds)
//		Expect(err).NotTo(HaveOccurred())
//
//		By("checking result balances")
//		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
//			"balanceOf", user.AddressBase58Check)
//
//		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(sendAmount), nil),
//			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
//
//	})
//
//	It("customer transfer insufficient funds", func() {
//		amount := "250"
//		sendAmount := "0"
//
//		authOpts := func(c *runtime.ClientOperation) {
//			c.AuthInfo = auth
//		}
//
//		By("creating new user")
//		user1, err := client.NewUserFoundation(pbfound.KeyType_ed25519)
//		Expect(err).NotTo(HaveOccurred())
//
//		client.AddUser(network, peer, network.Orderers[0], user1)
//
//		By("creating channel transfer request")
//		transferID := uuid.NewString()
//		channelTransferArgs := []string{transferID, ccCCUpper, ccFiatUpper, amount}
//
//		requestID := uuid.NewString()
//		nonce := client.NewNonceByTime().Get()
//		signArgs := append(append([]string{fnChannelTransferByCustomer, requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
//		publicKey, sign, err := user1.Sign(signArgs...)
//		Expect(err).NotTo(HaveOccurred())
//
//		transferRequest := &models.ChannelTransferTransferBeginCustomerRequest{
//			Generals: &models.ChannelTransferGeneralParams{
//				MethodName: fnChannelTransferByCustomer,
//				RequestID:  requestID,
//				Chaincode:  cmn.ChannelFiat,
//				Channel:    cmn.ChannelFiat,
//				Nonce:      nonce,
//				PublicKey:  publicKey,
//				Sign:       base58.Encode(sign),
//			},
//			IDTransfer: channelTransferArgs[0],
//			ChannelTo:  channelTransferArgs[1],
//			Token:      channelTransferArgs[2],
//			Amount:     channelTransferArgs[3],
//		}
//
//		By("sending transfer request")
//		res, err := transferCli.Transfer.TransferByCustomer(&transfer.TransferByCustomerParams{Body: transferRequest, Context: clientCtx}, authOpts)
//		Expect(err).NotTo(HaveOccurred())
//
//		err = checkResponseStatus(res.GetPayload(), models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, "")
//		Expect(err).NotTo(HaveOccurred())
//
//		By("awaiting for channel transfer to respond")
//		err = waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSERROR, errInsufficientFunds)
//		Expect(err).NotTo(HaveOccurred())
//
//		By("checking result balances")
//		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
//			"balanceOf", user.AddressBase58Check)
//
//		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(sendAmount), nil),
//			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
//	})
//
//	It("admin transfer wrong channel from", func() {
//		amount := "250"
//		sendAmount := "0"
//
//		authOpts := func(c *runtime.ClientOperation) {
//			c.AuthInfo = auth
//		}
//
//		By("creating channel transfer request")
//		transferID := uuid.NewString()
//		channelTransferArgs := []string{transferID, ccCCUpper, user.AddressBase58Check, ccIndustrialUpper, amount}
//
//		requestID := uuid.NewString()
//		nonce := client.NewNonceByTime().Get()
//		signArgs := append(append([]string{fnChannelTransferByAdmin, requestID, cmn.ChannelIndustrial, cmn.ChannelIndustrial}, channelTransferArgs...), nonce)
//		publicKey, sign, err := admin.Sign(signArgs...)
//		Expect(err).NotTo(HaveOccurred())
//
//		transferRequest := &models.ChannelTransferTransferBeginAdminRequest{
//			Generals: &models.ChannelTransferGeneralParams{
//				MethodName: fnChannelTransferByAdmin,
//				RequestID:  requestID,
//				Chaincode:  cmn.ChannelIndustrial,
//				Channel:    cmn.ChannelIndustrial,
//				Nonce:      nonce,
//				PublicKey:  publicKey,
//				Sign:       base58.Encode(sign),
//			},
//			IDTransfer: channelTransferArgs[0],
//			ChannelTo:  channelTransferArgs[1],
//			Address:    channelTransferArgs[2],
//			Token:      channelTransferArgs[3],
//			Amount:     channelTransferArgs[4],
//		}
//
//		By("sending transfer request")
//		res, err := transferCli.Transfer.TransferByAdmin(&transfer.TransferByAdminParams{Body: transferRequest, Context: clientCtx}, authOpts)
//		Expect(err).NotTo(HaveOccurred())
//
//		err = checkResponseStatus(res.GetPayload(), models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, "")
//		Expect(err).NotTo(HaveOccurred())
//
//		By("awaiting for channel transfer to respond")
//		err = waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSERROR, errInsufficientFunds)
//		Expect(err).NotTo(HaveOccurred())
//
//		By("checking result balances")
//		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
//			"balanceOf", user.AddressBase58Check)
//
//		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(sendAmount), nil),
//			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
//	})
//
//	It("customer transfer wrong channel from", func() {
//		amount := "250"
//		sendAmount := "0"
//
//		authOpts := func(c *runtime.ClientOperation) {
//			c.AuthInfo = auth
//		}
//
//		By("creating channel transfer request")
//		transferID := uuid.NewString()
//		channelTransferArgs := []string{transferID, ccCCUpper, ccIndustrialUpper, amount}
//
//		requestID := uuid.NewString()
//		nonce := client.NewNonceByTime().Get()
//		signArgs := append(append([]string{fnChannelTransferByCustomer, requestID, cmn.ChannelIndustrial, cmn.ChannelIndustrial}, channelTransferArgs...), nonce)
//		publicKey, sign, err := user.Sign(signArgs...)
//		Expect(err).NotTo(HaveOccurred())
//
//		transferRequest := &models.ChannelTransferTransferBeginCustomerRequest{
//			Generals: &models.ChannelTransferGeneralParams{
//				MethodName: fnChannelTransferByCustomer,
//				RequestID:  requestID,
//				Chaincode:  cmn.ChannelIndustrial,
//				Channel:    cmn.ChannelIndustrial,
//				Nonce:      nonce,
//				PublicKey:  publicKey,
//				Sign:       base58.Encode(sign),
//			},
//			IDTransfer: channelTransferArgs[0],
//			ChannelTo:  channelTransferArgs[1],
//			Token:      channelTransferArgs[2],
//			Amount:     channelTransferArgs[3],
//		}
//
//		By("sending transfer request")
//		res, err := transferCli.Transfer.TransferByCustomer(&transfer.TransferByCustomerParams{Body: transferRequest, Context: clientCtx}, authOpts)
//		Expect(err).NotTo(HaveOccurred())
//
//		err = checkResponseStatus(res.GetPayload(), models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, "")
//		Expect(err).NotTo(HaveOccurred())
//
//		By("awaiting for channel transfer to respond")
//		err = waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSERROR, errInsufficientFunds)
//		Expect(err).NotTo(HaveOccurred())
//
//		By("checking result balances")
//		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
//			"balanceOf", user.AddressBase58Check)
//
//		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(sendAmount), nil),
//			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
//	})
//
//	It("admin transfer wrong channel to", func() {
//		amount := "250"
//		sendAmount := "0"
//
//		authOpts := func(c *runtime.ClientOperation) {
//			c.AuthInfo = auth
//		}
//
//		By("creating channel transfer request")
//		transferID := uuid.NewString()
//		channelTransferArgs := []string{transferID, ccCCUpper, user.AddressBase58Check, ccIndustrialUpper, amount}
//
//		requestID := uuid.NewString()
//		nonce := client.NewNonceByTime().Get()
//		signArgs := append(append([]string{fnChannelTransferByAdmin, requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
//		publicKey, sign, err := admin.Sign(signArgs...)
//		Expect(err).NotTo(HaveOccurred())
//
//		transferRequest := &models.ChannelTransferTransferBeginAdminRequest{
//			Generals: &models.ChannelTransferGeneralParams{
//				MethodName: fnChannelTransferByAdmin,
//				RequestID:  requestID,
//				Chaincode:  cmn.ChannelFiat,
//				Channel:    cmn.ChannelFiat,
//				Nonce:      nonce,
//				PublicKey:  publicKey,
//				Sign:       base58.Encode(sign),
//			},
//			IDTransfer: channelTransferArgs[0],
//			ChannelTo:  channelTransferArgs[1],
//			Address:    channelTransferArgs[2],
//			Token:      channelTransferArgs[3],
//			Amount:     channelTransferArgs[4],
//		}
//
//		By("sending transfer request")
//		_, err = transferCli.Transfer.TransferByAdmin(&transfer.TransferByAdminParams{Body: transferRequest, Context: clientCtx}, authOpts)
//		Expect(err).Should(ContainSubstring(errIncorrectToken))
//
//		By("checking result balances")
//		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
//			"balanceOf", user.AddressBase58Check)
//
//		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(sendAmount), nil),
//			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
//	})
//
//	It("customer transfer wrong channel to", func() {
//		amount := "250"
//		sendAmount := "0"
//
//		authOpts := func(c *runtime.ClientOperation) {
//			c.AuthInfo = auth
//		}
//
//		By("creating channel transfer request")
//		transferID := uuid.NewString()
//		channelTransferArgs := []string{transferID, ccCCUpper, ccIndustrialUpper, amount}
//
//		requestID := uuid.NewString()
//		nonce := client.NewNonceByTime().Get()
//		signArgs := append(append([]string{fnChannelTransferByCustomer, requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
//		publicKey, sign, err := user.Sign(signArgs...)
//		Expect(err).NotTo(HaveOccurred())
//
//		transferRequest := &models.ChannelTransferTransferBeginCustomerRequest{
//			Generals: &models.ChannelTransferGeneralParams{
//				MethodName: fnChannelTransferByCustomer,
//				RequestID:  requestID,
//				Chaincode:  cmn.ChannelFiat,
//				Channel:    cmn.ChannelFiat,
//				Nonce:      nonce,
//				PublicKey:  publicKey,
//				Sign:       base58.Encode(sign),
//			},
//			IDTransfer: channelTransferArgs[0],
//			ChannelTo:  channelTransferArgs[1],
//			Token:      channelTransferArgs[2],
//			Amount:     channelTransferArgs[3],
//		}
//
//		By("sending transfer request")
//		_, err = transferCli.Transfer.TransferByCustomer(&transfer.TransferByCustomerParams{Body: transferRequest, Context: clientCtx}, authOpts)
//		Expect(err).Should(ContainSubstring(errIncorrectToken))
//
//		By("checking result balances")
//		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
//			"balanceOf", user.AddressBase58Check)
//
//		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
//			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(sendAmount), nil),
//			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
//	})
//})

func checkResponseStatus(
	payload *models.ChannelTransferTransferStatusResponse,
	expectedStatus models.ChannelTransferTransferStatusResponseStatus,
	expectedError string,
) error {
	if *payload.Status == models.ChannelTransferTransferStatusResponseStatusSTATUSERROR &&
		expectedStatus != models.ChannelTransferTransferStatusResponseStatusSTATUSERROR &&
		expectedError == "" {
		return fmt.Errorf("error occured: %s", payload.Message)
	}
	if expectedError != "" && !strings.Contains(payload.Message, expectedError) {
		return fmt.Errorf("expected %s, got %s", expectedError, payload.Message)
	}
	if *payload.Status != expectedStatus {
		return fmt.Errorf("status %s was not received", string(expectedStatus))
	}

	return nil
}

func waitForAnswerAndCheckStatus(
	clientCtx context.Context,
	transferCli *clihttp.CrossChanelTransfer,
	transferID string,
	authOpts func(c *runtime.ClientOperation),
	expectedStatus models.ChannelTransferTransferStatusResponseStatus,
	expectedError string,
) error {
	var (
		response *transfer.TransferStatusOK
		err      error
	)
	i := 0
	for i < transferExecutionTimeout {
		time.Sleep(time.Second * 1)
		response, err = transferCli.Transfer.TransferStatus(&transfer.TransferStatusParams{IDTransfer: transferID, Context: clientCtx}, authOpts)
		if err != nil {
			return err
		}
		if expectedStatus != models.ChannelTransferTransferStatusResponseStatusSTATUSERROR && *response.Payload.Status == models.ChannelTransferTransferStatusResponseStatusSTATUSERROR {
			return fmt.Errorf("error occured: %s", response.Payload.Message)
		}
		if err := checkResponseStatus(response.Payload, expectedStatus, expectedError); err == nil {
			return nil
		}
		i++
	}
	return fmt.Errorf("status %s was not received", string(expectedStatus))
}
