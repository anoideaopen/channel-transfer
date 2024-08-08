package http

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	clihttp "github.com/anoideaopen/channel-transfer/test/integration/clihttp/client"
	"github.com/anoideaopen/channel-transfer/test/integration/clihttp/client/transfer"
	"github.com/anoideaopen/channel-transfer/test/integration/clihttp/models"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/anoideaopen/foundation/test/integration/cmn/fabricnetwork"
	"github.com/anoideaopen/foundation/test/integration/cmn/runner"
	"github.com/btcsuite/btcutil/base58"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/fabricconfig"
	runnerFbk "github.com/hyperledger/fabric/integration/nwo/runner"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/tedsuo/ifrit"
	ginkgomon "github.com/tedsuo/ifrit/ginkgomon_v2"
	"google.golang.org/grpc/metadata"
)

const (
	fnChannelMultiTransferByAdmin    = "channelMultiTransferByAdmin"
	fnChannelMultiTransferByCustomer = "channelMultiTransferByCustomer"
)

var _ = Describe("Channel multi transfer HTTP tests", func() {
	var (
		testDir          string
		cli              *docker.Client
		network          *nwo.Network
		networkProcess   ifrit.Process
		ordererProcesses []ifrit.Process
		peerProcesses    ifrit.Process
	)

	BeforeEach(func() {
		networkProcess = nil
		ordererProcesses = nil
		peerProcesses = nil
		var err error
		testDir, err = os.MkdirTemp("", "foundation")
		Expect(err).NotTo(HaveOccurred())

		cli, err = docker.NewClientFromEnv()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if networkProcess != nil {
			networkProcess.Signal(syscall.SIGTERM)
			Eventually(networkProcess.Wait(), network.EventuallyTimeout).Should(Receive())
		}
		if peerProcesses != nil {
			peerProcesses.Signal(syscall.SIGTERM)
			Eventually(peerProcesses.Wait(), network.EventuallyTimeout).Should(Receive())
		}
		if network != nil {
			network.Cleanup()
		}
		for _, ordererInstance := range ordererProcesses {
			ordererInstance.Signal(syscall.SIGTERM)
			Eventually(ordererInstance.Wait(), network.EventuallyTimeout).Should(Receive())
		}
		err := os.RemoveAll(testDir)
		Expect(err).NotTo(HaveOccurred())
	})

	var (
		channels            = []string{cmn.ChannelAcl, cmn.ChannelCC, cmn.ChannelIndustrial}
		ordererRunners      []*ginkgomon.Runner
		redisProcess        ifrit.Process
		redisDB             *runner.RedisDB
		networkFound        *cmn.NetworkFoundation
		robotProc           ifrit.Process
		channelTransferProc ifrit.Process
		skiBackend          string
		skiRobot            string
		peer                *nwo.Peer
		admin               *client.UserFoundation
		user                *client.UserFoundation
		feeSetter           *client.UserFoundation
		feeAddressSetter    *client.UserFoundation

		clientCtx   context.Context
		transferCli *clihttp.CrossChanelTransfer
		auth        runtime.ClientAuthInfoWriter

		transferItems              []*models.ChannelTransferTransferItem
		initialBalances            []*models.ChannelTransferTransferItem
		expectedIndustrialBalances []*models.ChannelTransferTransferItem
	)
	BeforeEach(func() {
		By("start redis")
		redisDB = &runner.RedisDB{}
		redisProcess = ifrit.Invoke(redisDB)
		Eventually(redisProcess.Ready(), runnerFbk.DefaultStartTimeout).Should(BeClosed())
		Consistently(redisProcess.Wait()).ShouldNot(Receive())
	})
	BeforeEach(func() {
		networkConfig := nwo.MultiNodeSmartBFT()
		networkConfig.Channels = nil

		pchs := make([]*nwo.PeerChannel, 0, cap(channels))
		for _, ch := range channels {
			pchs = append(pchs, &nwo.PeerChannel{
				Name:   ch,
				Anchor: true,
			})
		}
		for _, peer := range networkConfig.Peers {
			peer.Channels = pchs
		}

		network = nwo.New(networkConfig, testDir, cli, StartPort(), components)
		cwd, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		network.ExternalBuilders = append(network.ExternalBuilders,
			fabricconfig.ExternalBuilder{
				Path:                 filepath.Join(cwd, ".", "externalbuilders", "binary"),
				Name:                 "binary",
				PropagateEnvironment: []string{"GOPROXY"},
			},
		)

		networkFound = cmn.New(network, channels)
		networkFound.Robot.RedisAddresses = []string{redisDB.Address()}
		networkFound.ChannelTransfer.RedisAddresses = []string{redisDB.Address()}

		networkFound.GenerateConfigTree()
		networkFound.Bootstrap()

		for _, orderer := range network.Orderers {
			runner := network.OrdererRunner(orderer)
			runner.Command.Env = append(runner.Command.Env, "FABRIC_LOGGING_SPEC=orderer.consensus.smartbft=debug:grpc=debug")
			ordererRunners = append(ordererRunners, runner)
			proc := ifrit.Invoke(runner)
			ordererProcesses = append(ordererProcesses, proc)
			Eventually(proc.Ready(), network.EventuallyTimeout).Should(BeClosed())
		}

		peerGroupRunner, _ := fabricnetwork.PeerGroupRunners(network)
		peerProcesses = ifrit.Invoke(peerGroupRunner)
		Eventually(peerProcesses.Ready(), network.EventuallyTimeout).Should(BeClosed())

		By("Joining orderers to channels")
		for _, channel := range channels {
			fabricnetwork.JoinChannel(network, channel)
		}

		By("Waiting for followers to see the leader")
		Eventually(ordererRunners[1].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))
		Eventually(ordererRunners[2].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))
		Eventually(ordererRunners[3].Err(), network.EventuallyTimeout, time.Second).Should(gbytes.Say("Message from 1"))

		By("Joining peers to channels")
		for _, channel := range channels {
			network.JoinChannel(channel, network.Orderers[0], network.PeersWithChannel(channel)...)
		}

		peer = network.Peer("Org1", "peer0")

		pathToPrivateKeyBackend := network.PeerUserKey(peer, "User1")
		skiBackend, err = cmn.ReadSKI(pathToPrivateKeyBackend)
		Expect(err).NotTo(HaveOccurred())

		pathToPrivateKeyRobot := network.PeerUserKey(peer, "User2")
		skiRobot, err = cmn.ReadSKI(pathToPrivateKeyRobot)
		Expect(err).NotTo(HaveOccurred())

		admin, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())
		Expect(admin.PrivateKeyBytes).NotTo(Equal(nil))

		feeSetter, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())
		Expect(feeSetter.PrivateKeyBytes).NotTo(Equal(nil))

		feeAddressSetter, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())
		Expect(feeAddressSetter.PrivateKeyBytes).NotTo(Equal(nil))

		cmn.DeployACL(network, components, peer, testDir, skiBackend, admin.PublicKeyBase58, admin.KeyType)
		cmn.DeployCC(network, components, peer, testDir, skiRobot, admin.AddressBase58Check)
		cmn.DeployIndustrial(network, components, peer, testDir, skiRobot,
			admin.AddressBase58Check, feeSetter.AddressBase58Check, feeAddressSetter.AddressBase58Check)
	})
	BeforeEach(func() {
		By("start robot")
		robotRunner := networkFound.RobotRunner()
		robotProc = ifrit.Invoke(robotRunner)
		Eventually(robotProc.Ready(), network.EventuallyTimeout).Should(BeClosed())

		By("start channel transfer")
		channelTransferRunner := networkFound.ChannelTransferRunner()
		channelTransferProc = ifrit.Invoke(channelTransferRunner)
		Eventually(channelTransferProc.Ready(), network.EventuallyTimeout).Should(BeClosed())
	})
	AfterEach(func() {
		By("stop robot")
		if robotProc != nil {
			robotProc.Signal(syscall.SIGTERM)
			Eventually(robotProc.Wait(), network.EventuallyTimeout).Should(Receive())
		}

		By("stop channel transfer")
		if channelTransferProc != nil {
			channelTransferProc.Signal(syscall.SIGTERM)
			Eventually(channelTransferProc.Wait(), network.EventuallyTimeout).Should(Receive())
		}
	})

	BeforeEach(func() {
		By("add admin to acl")
		client.AddUser(network, peer, network.Orderers[0], admin)

		By("add user to acl")
		var err error
		user, err = client.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())

		client.AddUser(network, peer, network.Orderers[0], user)

		By("emit tokens")
		client.NBTxInvokeWithSign(network, peer, network.Orderers[0], nil, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
			admin, "initialize", "", client.NewNonceByTime().Get())

		initialBalances = []*models.ChannelTransferTransferItem{
			{
				Token:  "INDUSTRIAL_202009",
				Amount: "10000000000000",
			},
			{
				Token:  "INDUSTRIAL_202010",
				Amount: "100000000000000",
			},
			{
				Token:  "INDUSTRIAL_202011",
				Amount: "200000000000000",
			},
			{
				Token:  "INDUSTRIAL_202012",
				Amount: "50000000000000",
			},
		}

		transferItems = []*models.ChannelTransferTransferItem{
			{
				Token:  "INDUSTRIAL_202009",
				Amount: "1000000000000",
			},
			{
				Token:  "INDUSTRIAL_202010",
				Amount: "10000000000000",
			},
			{
				Token:  "INDUSTRIAL_202011",
				Amount: "20000000000000",
			},
			{
				Token:  "INDUSTRIAL_202012",
				Amount: "5000000000000",
			},
		}

		expectedIndustrialBalances = []*models.ChannelTransferTransferItem{
			{
				Token:  "INDUSTRIAL_202009",
				Amount: "9000000000000",
			},
			{
				Token:  "INDUSTRIAL_202010",
				Amount: "90000000000000",
			},
			{
				Token:  "INDUSTRIAL_202011",
				Amount: "180000000000000",
			},
			{
				Token:  "INDUSTRIAL_202012",
				Amount: "45000000000000",
			},
		}

		for _, initial := range initialBalances {
			group := strings.Split(initial.Token, "_")[1]

			client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
				fabricnetwork.CheckResult(fabricnetwork.CheckIndustrialBalance(group, initial.Amount), nil),
				"industrialBalanceOf", admin.AddressBase58Check)

			client.TxInvokeWithSign(network, peer, network.Orderers[0], cmn.ChannelIndustrial, cmn.ChannelIndustrial,
				admin, "transferIndustrial", "", client.NewNonceByTime().Get(), nil,
				user.AddressBase58Check, group, initial.Amount, "transfer industrial tokens")

			client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
				fabricnetwork.CheckResult(fabricnetwork.CheckIndustrialBalance(group, initial.Amount), nil),
				"industrialBalanceOf", user.AddressBase58Check)
		}

		By("creating http connection")
		clientCtx = metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", networkFound.ChannelTransfer.AccessToken))

		httpAddress := networkFound.ChannelTransfer.HostAddress + ":" + strconv.FormatUint(uint64(networkFound.ChannelTransfer.Ports[cmn.HttpPort]), 10)
		transport := httptransport.New(httpAddress, "", nil)
		transferCli = clihttp.New(transport, strfmt.Default)

		auth = httptransport.APIKeyAuth("authorization", "header", networkFound.ChannelTransfer.AccessToken)
	})

	It("multi transfer by admin test", func() {
		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		By("creating channel transfer request")
		items, err := json.Marshal(transferItems)
		Expect(err).NotTo(HaveOccurred())

		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, ccCCUpper, user.AddressBase58Check, string(items)}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{fnChannelMultiTransferByAdmin, requestID, cmn.ChannelIndustrial, cmn.ChannelIndustrial}, channelTransferArgs...), nonce)
		publicKey, sign, err := admin.Sign(signArgs...)
		Expect(err).NotTo(HaveOccurred())

		transferRequest := &models.ChannelTransferMultiTransferBeginAdminRequest{
			Generals: &models.ChannelTransferGeneralParams{
				MethodName: fnChannelMultiTransferByAdmin,
				RequestID:  requestID,
				Chaincode:  cmn.ChannelIndustrial,
				Channel:    cmn.ChannelIndustrial,
				Nonce:      nonce,
				PublicKey:  publicKey,
				Sign:       base58.Encode(sign),
			},
			IDTransfer: channelTransferArgs[0],
			ChannelTo:  channelTransferArgs[1],
			Address:    channelTransferArgs[2],
			Items:      transferItems,
		}

		By("sending transfer request")
		res, err := transferCli.Transfer.MultiTransferByAdmin(&transfer.MultiTransferByAdminParams{Body: transferRequest, Context: clientCtx}, authOpts)
		Expect(err).NotTo(HaveOccurred())

		err = checkResponseStatus(res.GetPayload(), models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, "")
		Expect(err).NotTo(HaveOccurred())

		By("awaiting for channel transfer to respond")
		err = waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED, "")
		Expect(err).NotTo(HaveOccurred())

		By("checking result balances")
		for i, expected := range expectedIndustrialBalances {
			group := strings.Split(expected.Token, "_")[1]

			client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
				fabricnetwork.CheckResult(fabricnetwork.CheckIndustrialBalance(group, expected.Amount), nil),
				"industrialBalanceOf", user.AddressBase58Check)

			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(transferItems[i].Amount), nil),
				"allowedBalanceOf", user.AddressBase58Check, transferItems[i].Token)
		}
	})

	It("multi transfer by customer test", func() {
		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		By("creating channel transfer request")
		items, err := json.Marshal(transferItems)
		Expect(err).NotTo(HaveOccurred())

		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, ccCCUpper, string(items)}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{fnChannelMultiTransferByCustomer, requestID, cmn.ChannelIndustrial, cmn.ChannelIndustrial}, channelTransferArgs...), nonce)
		publicKey, sign, err := user.Sign(signArgs...)
		Expect(err).NotTo(HaveOccurred())

		transferRequest := &models.ChannelTransferMultiTransferBeginCustomerRequest{
			Generals: &models.ChannelTransferGeneralParams{
				MethodName: fnChannelMultiTransferByCustomer,
				RequestID:  requestID,
				Chaincode:  cmn.ChannelIndustrial,
				Channel:    cmn.ChannelIndustrial,
				Nonce:      nonce,
				PublicKey:  publicKey,
				Sign:       base58.Encode(sign),
			},
			IDTransfer: channelTransferArgs[0],
			ChannelTo:  channelTransferArgs[1],
			Items:      transferItems,
		}

		By("sending transfer request")
		res, err := transferCli.Transfer.MultiTransferByCustomer(&transfer.MultiTransferByCustomerParams{Body: transferRequest, Context: clientCtx}, authOpts)
		Expect(err).NotTo(HaveOccurred())

		err = checkResponseStatus(res.GetPayload(), models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, "")
		Expect(err).NotTo(HaveOccurred())

		By("awaiting for channel transfer to respond")
		err = waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED, "")
		Expect(err).NotTo(HaveOccurred())

		By("checking result balances")
		for i, expected := range expectedIndustrialBalances {
			group := strings.Split(expected.Token, "_")[1]

			client.Query(network, peer, cmn.ChannelIndustrial, cmn.ChannelIndustrial,
				fabricnetwork.CheckResult(fabricnetwork.CheckIndustrialBalance(group, expected.Amount), nil),
				"industrialBalanceOf", user.AddressBase58Check)

			client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
				fabricnetwork.CheckResult(fabricnetwork.CheckBalance(transferItems[i].Amount), nil),
				"allowedBalanceOf", user.AddressBase58Check, transferItems[i].Token)
		}
	})

})
