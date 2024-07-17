package chaos

import (
	"context"
	"fmt"
	"github.com/anoideaopen/channel-transfer/test/integration/clihttp/client/transfer"
	"github.com/anoideaopen/channel-transfer/test/integration/clihttp/models"
	"github.com/google/uuid"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	clihttp "github.com/anoideaopen/channel-transfer/test/integration/clihttp/client"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/anoideaopen/foundation/test/integration/cmn/fabricnetwork"
	"github.com/anoideaopen/foundation/test/integration/cmn/runner"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
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
	transferExecutionTimeout = 100

	ccCCUpper   = "CC"
	ccFiatUpper = "FIAT"

	emitAmount = "1000"

	fnChannelTransferByAdmin    = "channelTransferByAdmin"
	fnChannelTransferByCustomer = "channelTransferByCustomer"
	fnChannelTransferFrom       = "channelTransferFrom"
	fnCreateCCTransferTo        = "createCCTransferTo"
	fnChannelTransferTo         = "channelTransferTo"
	fnCommitCCTransferFrom      = "commitCCTransferFrom"
	fnDeleteCCTransferTo        = "deleteCCTransferTo"
	fnDeleteCCTransferFrom      = "deleteCCTransferFrom"

	ttlExpireTimeout = 11
)

var _ = Describe("Channel transfer HTTP tests", func() {
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
		channels            = []string{cmn.ChannelAcl, cmn.ChannelCC, cmn.ChannelFiat, cmn.ChannelIndustrial}
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
	)
	BeforeEach(func() {
		By("start redis")
		redisDB = &runner.RedisDB{}
		redisProcess = ifrit.Invoke(redisDB)
		Eventually(redisProcess.Ready(), runnerFbk.DefaultStartTimeout).Should(BeClosed())
		Consistently(redisProcess.Wait()).ShouldNot(Receive())
	})
	AfterEach(func() {
		By("stop redis " + redisDB.Address())
		if redisProcess != nil {
			redisProcess.Signal(syscall.SIGTERM)
			Eventually(redisProcess.Wait(), time.Minute).Should(Receive())
		}
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
		networkFound.ChannelTransfer.TTL = fmt.Sprintf("%ds", ttlExpireTimeout)

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
		cmn.DeployFiat(network, components, peer, testDir, skiRobot,
			admin.AddressBase58Check, feeSetter.AddressBase58Check, feeAddressSetter.AddressBase58Check)
		cmn.DeployIndustrial(network, components, peer, testDir, skiRobot,
			admin.AddressBase58Check, feeSetter.AddressBase58Check, feeAddressSetter.AddressBase58Check)
	})
	BeforeEach(func() {
		By("start robot")
		robotRunner := networkFound.RobotRunner()
		robotProc = ifrit.Invoke(robotRunner)
		Eventually(robotProc.Ready(), network.EventuallyTimeout).Should(BeClosed())

		By("start channel transfer")
		channelTransferProc = startChannelTransfer(networkFound, network.EventuallyTimeout)
	})
	AfterEach(func() {
		By("stop robot")
		if robotProc != nil {
			robotProc.Signal(syscall.SIGTERM)
			Eventually(robotProc.Wait(), network.EventuallyTimeout).Should(Receive())
		}

		By("stop channel transfer")
		stopChannelTransfer(channelTransferProc, network.EventuallyTimeout)
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
		client.TxInvokeWithSign(network, peer, network.Orderers[0],
			cmn.ChannelFiat, cmn.ChannelFiat, admin,
			"emit", "", client.NewNonceByTime().Get(), nil, user.AddressBase58Check, emitAmount)

		By("emit check")
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
			"balanceOf", user.AddressBase58Check)

		By("creating http connection")
		clientCtx = metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", networkFound.ChannelTransfer.AccessToken))

		httpAddress := networkFound.ChannelTransfer.HostAddress + ":" + strconv.FormatUint(uint64(networkFound.ChannelTransfer.Ports[cmn.HttpPort]), 10)
		transport := httptransport.New(httpAddress, "", nil)
		transferCli = clihttp.New(transport, strfmt.Default)

		auth = httptransport.APIKeyAuth("authorization", "header", networkFound.ChannelTransfer.AccessToken)
	})

	It("Admin transfer before TTL test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("Stopping service")
		stopChannelTransfer(channelTransferProc, network.EventuallyTimeout)

		By("Invoking transaction")
		client.TxInvokeWithSign(network, peer, network.Orderers[0], cmn.ChannelFiat, cmn.ChannelFiat, admin, fnChannelTransferByAdmin, "0", client.NewNonceByTime().Get(), nil, transferID, ccCCUpper, user.AddressBase58Check, ccFiatUpper, amount)

		By("Starting service")
		channelTransferProc = startChannelTransfer(networkFound, network.EventuallyTimeout)

		By("awaiting for channel transfer to respond")
		err := waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED, "")
		Expect(err).NotTo(HaveOccurred())

		By("checking result balances")
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(restAmount), nil),
			"balanceOf", user.AddressBase58Check)

		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amount), nil),
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)

	})

	It("Customer transfer before TTL test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		stopChannelTransfer(channelTransferProc, network.EventuallyTimeout)

		client.TxInvokeWithSign(network, peer, network.Orderers[0], cmn.ChannelFiat, cmn.ChannelFiat, user, fnChannelTransferByCustomer, "0", client.NewNonceByTime().Get(), nil, transferID, ccCCUpper, ccFiatUpper, amount)

		channelTransferProc = startChannelTransfer(networkFound, network.EventuallyTimeout)

		By("awaiting for channel transfer to respond")
		err := waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED, "")
		Expect(err).NotTo(HaveOccurred())

		By("checking result balances")
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(restAmount), nil),
			"balanceOf", user.AddressBase58Check)

		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amount), nil),
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)

	})

	It("Transfer by admin cancel by test", func() {
		amount := "250"
		sendAmount := "0"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("Stopping service")
		stopChannelTransfer(channelTransferProc, network.EventuallyTimeout)

		By("Invoking transaction")
		client.TxInvokeWithSign(network, peer, network.Orderers[0], cmn.ChannelFiat, cmn.ChannelFiat, admin, fnChannelTransferByAdmin, "0", client.NewNonceByTime().Get(), nil, transferID, ccCCUpper, user.AddressBase58Check, ccFiatUpper, amount)

		By("Waiting TTL for expiring")
		time.Sleep(time.Second * ttlExpireTimeout)

		By("Starting service")
		channelTransferProc = startChannelTransfer(networkFound, network.EventuallyTimeout)

		time.Sleep(time.Second * 6)

		By("awaiting for channel transfer to respond")
		err := waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSCANCELED, "")
		Expect(err).NotTo(HaveOccurred())

		By("checking result balances")
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
			"balanceOf", user.AddressBase58Check)

		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(sendAmount), nil),
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
	})

	It("Customer transfer cancel by timeout test", func() {
		amount := "250"
		sendAmount := "0"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		stopChannelTransfer(channelTransferProc, network.EventuallyTimeout)

		client.TxInvokeWithSign(network, peer, network.Orderers[0], cmn.ChannelFiat, cmn.ChannelFiat, user, fnChannelTransferByCustomer, "0", client.NewNonceByTime().Get(), nil, transferID, ccCCUpper, ccFiatUpper, amount)

		By("Waiting TTL for expiring")
		time.Sleep(time.Second * ttlExpireTimeout)

		By("Starting service")
		channelTransferProc = startChannelTransfer(networkFound, network.EventuallyTimeout)

		time.Sleep(time.Second * 6)

		By("awaiting for channel transfer to respond")
		err := waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSCANCELED, "")
		Expect(err).NotTo(HaveOccurred())

		By("checking result balances")
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(emitAmount), nil),
			"balanceOf", user.AddressBase58Check)

		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(sendAmount), nil),
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
	})

	It("Channel Transfer To before TTL test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("Stopping service")
		stopChannelTransfer(channelTransferProc, network.EventuallyTimeout)

		By("Invoking transaction")
		client.TxInvokeWithSign(network, peer, network.Orderers[0], cmn.ChannelFiat, cmn.ChannelFiat, admin, fnChannelTransferByAdmin, "0", client.NewNonceByTime().Get(), nil, transferID, ccCCUpper, user.AddressBase58Check, ccFiatUpper, amount)

		By("Querying ChannelTransferFrom")
		from := ""
		fChTrFrom := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}
			from = string(out)
			return ""
		}
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, fabricnetwork.CheckResult(fChTrFrom, nil),
			fnChannelTransferFrom, transferID)
		Expect(from).NotTo(BeEmpty())

		By("create cc transfer to")
		client.TxInvokeByRobot(network, peer, network.Orderers[0],
			cmn.ChannelCC, cmn.ChannelCC, nil, fnCreateCCTransferTo, from)

		By("Starting service")
		channelTransferProc = startChannelTransfer(networkFound, network.EventuallyTimeout)

		By("awaiting for channel transfer to respond")
		err := waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED, "")
		Expect(err).NotTo(HaveOccurred())

		By("checking result balances")
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(restAmount), nil),
			"balanceOf", user.AddressBase58Check)

		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amount), nil),
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
	})

	It("Channel transfer To after TTL test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("Stopping service")
		stopChannelTransfer(channelTransferProc, network.EventuallyTimeout)

		By("Invoking transaction")
		client.TxInvokeWithSign(network, peer, network.Orderers[0], cmn.ChannelFiat, cmn.ChannelFiat, admin, fnChannelTransferByAdmin, "0", client.NewNonceByTime().Get(), nil, transferID, ccCCUpper, user.AddressBase58Check, ccFiatUpper, amount)

		By("Querying ChannelTransferFrom")
		from := ""
		fChTrFrom := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}
			from = string(out)
			return ""
		}
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, fabricnetwork.CheckResult(fChTrFrom, nil),
			fnChannelTransferFrom, transferID)
		Expect(from).NotTo(BeEmpty())

		By("create cc transfer to")
		client.TxInvokeByRobot(network, peer, network.Orderers[0],
			cmn.ChannelCC, cmn.ChannelCC, nil, fnCreateCCTransferTo, from)

		By("Waiting TTL for expiring")
		time.Sleep(time.Second * ttlExpireTimeout)

		By("Starting service")
		channelTransferProc = startChannelTransfer(networkFound, network.EventuallyTimeout)

		By("awaiting for channel transfer to respond")
		err := waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED, "")
		Expect(err).NotTo(HaveOccurred())

		By("checking result balances")
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(restAmount), nil),
			"balanceOf", user.AddressBase58Check)

		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amount), nil),
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
	})

	It("Channel transfer From before TTL test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("Stopping service")
		stopChannelTransfer(channelTransferProc, network.EventuallyTimeout)

		By("Invoking transaction")
		client.TxInvokeWithSign(network, peer, network.Orderers[0], cmn.ChannelFiat, cmn.ChannelFiat, admin, fnChannelTransferByAdmin, "0", client.NewNonceByTime().Get(), nil, transferID, ccCCUpper, user.AddressBase58Check, ccFiatUpper, amount)

		By("Querying ChannelTransferFrom")
		from := ""
		fChTrFrom := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}
			from = string(out)
			return ""
		}
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, fabricnetwork.CheckResult(fChTrFrom, nil),
			fnChannelTransferFrom, transferID)
		Expect(from).NotTo(BeEmpty())

		By("create cc transfer to")
		client.TxInvokeByRobot(network, peer, network.Orderers[0],
			cmn.ChannelCC, cmn.ChannelCC, nil, fnCreateCCTransferTo, from)

		By("channel transfer to")
		fChTrTo := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}

			return ""
		}
		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC, fabricnetwork.CheckResult(fChTrTo, nil),
			fnChannelTransferTo, transferID)

		By("commit cc transfer from")
		client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
			cmn.ChannelFiat, cmn.ChannelFiat, fnCommitCCTransferFrom, transferID)

		By("Starting service")
		channelTransferProc = startChannelTransfer(networkFound, network.EventuallyTimeout)

		By("awaiting for channel transfer to respond")
		err := waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED, "")
		Expect(err).NotTo(HaveOccurred())

		By("checking result balances")
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(restAmount), nil),
			"balanceOf", user.AddressBase58Check)

		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amount), nil),
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
	})

	It("Channel transfer From after TTL test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("Stopping service")
		stopChannelTransfer(channelTransferProc, network.EventuallyTimeout)

		By("Invoking transaction")
		client.TxInvokeWithSign(network, peer, network.Orderers[0], cmn.ChannelFiat, cmn.ChannelFiat, admin, fnChannelTransferByAdmin, "0", client.NewNonceByTime().Get(), nil, transferID, ccCCUpper, user.AddressBase58Check, ccFiatUpper, amount)

		By("Querying ChannelTransferFrom")
		from := ""
		fChTrFrom := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}
			from = string(out)
			return ""
		}
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, fabricnetwork.CheckResult(fChTrFrom, nil),
			fnChannelTransferFrom, transferID)
		Expect(from).NotTo(BeEmpty())

		By("create cc transfer to")
		client.TxInvokeByRobot(network, peer, network.Orderers[0],
			cmn.ChannelCC, cmn.ChannelCC, nil, fnCreateCCTransferTo, from)

		By("channel transfer to")
		fChTrTo := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}

			return ""
		}
		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC, fabricnetwork.CheckResult(fChTrTo, nil),
			fnChannelTransferTo, transferID)

		By("commit cc transfer from")
		client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
			cmn.ChannelFiat, cmn.ChannelFiat, fnCommitCCTransferFrom, transferID)

		By("Waiting TTL for expiring")
		time.Sleep(time.Second * ttlExpireTimeout)

		By("Starting service")
		channelTransferProc = startChannelTransfer(networkFound, network.EventuallyTimeout)

		By("awaiting for channel transfer to respond")
		err := waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED, "")
		Expect(err).NotTo(HaveOccurred())

		By("checking result balances")
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(restAmount), nil),
			"balanceOf", user.AddressBase58Check)

		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amount), nil),
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
	})

	It("Delete transfer To before TTL test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("Stopping service")
		stopChannelTransfer(channelTransferProc, network.EventuallyTimeout)

		By("Invoking transaction")
		client.TxInvokeWithSign(network, peer, network.Orderers[0], cmn.ChannelFiat, cmn.ChannelFiat, admin, fnChannelTransferByAdmin, "0", client.NewNonceByTime().Get(), nil, transferID, ccCCUpper, user.AddressBase58Check, ccFiatUpper, amount)

		By("Querying ChannelTransferFrom")
		from := ""
		fChTrFrom := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}
			from = string(out)
			return ""
		}
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, fabricnetwork.CheckResult(fChTrFrom, nil),
			fnChannelTransferFrom, transferID)
		Expect(from).NotTo(BeEmpty())

		By("create cc transfer to")
		client.TxInvokeByRobot(network, peer, network.Orderers[0],
			cmn.ChannelCC, cmn.ChannelCC, nil, fnCreateCCTransferTo, from)

		By("channel transfer to")
		fChTrTo := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}

			return ""
		}
		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC, fabricnetwork.CheckResult(fChTrTo, nil),
			fnChannelTransferTo, transferID)

		By("commit cc transfer from")
		client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
			cmn.ChannelFiat, cmn.ChannelFiat, fnCommitCCTransferFrom, transferID)

		By("delete cc transfer to")
		client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
			cmn.ChannelCC, cmn.ChannelCC, fnDeleteCCTransferTo, transferID)

		By("Starting service")
		channelTransferProc = startChannelTransfer(networkFound, network.EventuallyTimeout)

		By("awaiting for channel transfer to respond")
		err := waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED, "")
		Expect(err).NotTo(HaveOccurred())

		By("checking result balances")
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(restAmount), nil),
			"balanceOf", user.AddressBase58Check)

		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amount), nil),
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
	})

	It("Delete transfer To after TTL test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("Stopping service")
		stopChannelTransfer(channelTransferProc, network.EventuallyTimeout)

		By("Invoking transaction")
		client.TxInvokeWithSign(network, peer, network.Orderers[0], cmn.ChannelFiat, cmn.ChannelFiat, admin, fnChannelTransferByAdmin, "0", client.NewNonceByTime().Get(), nil, transferID, ccCCUpper, user.AddressBase58Check, ccFiatUpper, amount)

		By("Querying ChannelTransferFrom")
		from := ""
		fChTrFrom := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}
			from = string(out)
			return ""
		}
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, fabricnetwork.CheckResult(fChTrFrom, nil),
			fnChannelTransferFrom, transferID)
		Expect(from).NotTo(BeEmpty())

		By("create cc transfer to")
		client.TxInvokeByRobot(network, peer, network.Orderers[0],
			cmn.ChannelCC, cmn.ChannelCC, nil, fnCreateCCTransferTo, from)

		By("channel transfer to")
		fChTrTo := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}

			return ""
		}
		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC, fabricnetwork.CheckResult(fChTrTo, nil),
			fnChannelTransferTo, transferID)

		By("commit cc transfer from")
		client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
			cmn.ChannelFiat, cmn.ChannelFiat, fnCommitCCTransferFrom, transferID)

		By("delete cc transfer to")
		client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
			cmn.ChannelCC, cmn.ChannelCC, fnDeleteCCTransferTo, transferID)

		By("Waiting TTL for expiring")
		time.Sleep(time.Second * ttlExpireTimeout)

		By("Starting service")
		channelTransferProc = startChannelTransfer(networkFound, network.EventuallyTimeout)

		By("awaiting for channel transfer to respond")
		err := waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED, "")
		Expect(err).NotTo(HaveOccurred())

		By("checking result balances")
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(restAmount), nil),
			"balanceOf", user.AddressBase58Check)

		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amount), nil),
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
	})

	It("Delete transfer From before TTL test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("Stopping service")
		stopChannelTransfer(channelTransferProc, network.EventuallyTimeout)

		By("Invoking transaction")
		client.TxInvokeWithSign(network, peer, network.Orderers[0], cmn.ChannelFiat, cmn.ChannelFiat, admin, fnChannelTransferByAdmin, "0", client.NewNonceByTime().Get(), nil, transferID, ccCCUpper, user.AddressBase58Check, ccFiatUpper, amount)

		By("Querying ChannelTransferFrom")
		from := ""
		fChTrFrom := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}
			from = string(out)
			return ""
		}
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, fabricnetwork.CheckResult(fChTrFrom, nil),
			fnChannelTransferFrom, transferID)
		Expect(from).NotTo(BeEmpty())

		By("create cc transfer to")
		client.TxInvokeByRobot(network, peer, network.Orderers[0],
			cmn.ChannelCC, cmn.ChannelCC, nil, fnCreateCCTransferTo, from)

		By("channel transfer to")
		fChTrTo := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}

			return ""
		}
		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC, fabricnetwork.CheckResult(fChTrTo, nil),
			fnChannelTransferTo, transferID)

		By("commit cc transfer from")
		client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
			cmn.ChannelFiat, cmn.ChannelFiat, fnCommitCCTransferFrom, transferID)

		By("delete cc transfer to")
		client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
			cmn.ChannelCC, cmn.ChannelCC, fnDeleteCCTransferTo, transferID)

		By("delete cc transfer from")
		client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
			cmn.ChannelFiat, cmn.ChannelFiat, fnDeleteCCTransferFrom, transferID)

		By("Starting service")
		channelTransferProc = startChannelTransfer(networkFound, network.EventuallyTimeout)

		By("awaiting for channel transfer to respond")
		err := waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED, "")
		Expect(err).NotTo(HaveOccurred())

		By("checking result balances")
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(restAmount), nil),
			"balanceOf", user.AddressBase58Check)

		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amount), nil),
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
	})

	It("Delete transfer From after TTL test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("Stopping service")
		stopChannelTransfer(channelTransferProc, network.EventuallyTimeout)

		By("Invoking transaction")
		client.TxInvokeWithSign(network, peer, network.Orderers[0], cmn.ChannelFiat, cmn.ChannelFiat, admin, fnChannelTransferByAdmin, "0", client.NewNonceByTime().Get(), nil, transferID, ccCCUpper, user.AddressBase58Check, ccFiatUpper, amount)

		By("Querying ChannelTransferFrom")
		from := ""
		fChTrFrom := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}
			from = string(out)
			return ""
		}
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat, fabricnetwork.CheckResult(fChTrFrom, nil),
			fnChannelTransferFrom, transferID)
		Expect(from).NotTo(BeEmpty())

		By("create cc transfer to")
		client.TxInvokeByRobot(network, peer, network.Orderers[0],
			cmn.ChannelCC, cmn.ChannelCC, nil, fnCreateCCTransferTo, from)

		By("channel transfer to")
		fChTrTo := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}

			return ""
		}
		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC, fabricnetwork.CheckResult(fChTrTo, nil),
			fnChannelTransferTo, transferID)

		By("commit cc transfer from")
		client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
			cmn.ChannelFiat, cmn.ChannelFiat, fnCommitCCTransferFrom, transferID)

		By("delete cc transfer to")
		client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
			cmn.ChannelCC, cmn.ChannelCC, fnDeleteCCTransferTo, transferID)

		By("delete cc transfer from")
		client.NBTxInvokeByRobot(network, peer, network.Orderers[0], nil,
			cmn.ChannelFiat, cmn.ChannelFiat, fnDeleteCCTransferFrom, transferID)

		By("Waiting TTL for expiring")
		time.Sleep(time.Second * ttlExpireTimeout)

		By("Starting service")
		channelTransferProc = startChannelTransfer(networkFound, network.EventuallyTimeout)

		By("awaiting for channel transfer to respond")
		err := waitForAnswerAndCheckStatus(clientCtx, transferCli, transferID, authOpts, models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED, "")
		Expect(err).NotTo(HaveOccurred())

		By("checking result balances")
		client.Query(network, peer, cmn.ChannelFiat, cmn.ChannelFiat,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(restAmount), nil),
			"balanceOf", user.AddressBase58Check)

		client.Query(network, peer, cmn.ChannelCC, cmn.ChannelCC,
			fabricnetwork.CheckResult(fabricnetwork.CheckBalance(amount), nil),
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper)
	})

})

func startChannelTransfer(networkFound *cmn.NetworkFoundation, timeout time.Duration) ifrit.Process {
	channelTransferRunner := networkFound.ChannelTransferRunner()
	channelTransferProc := ifrit.Invoke(channelTransferRunner)
	Eventually(channelTransferProc.Ready(), timeout).Should(BeClosed())

	return channelTransferProc
}

func stopChannelTransfer(channelTransferProc ifrit.Process, timeout time.Duration) {
	if channelTransferProc != nil {
		channelTransferProc.Signal(syscall.SIGTERM)
		Eventually(channelTransferProc.Wait(), timeout).Should(Receive())
	}
}

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
		return fmt.Errorf("status %s was not received, got %s", string(expectedStatus), *payload.Status)
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
	return fmt.Errorf("status %s was not received, got %s", string(expectedStatus), *response.Payload.Status)
}
