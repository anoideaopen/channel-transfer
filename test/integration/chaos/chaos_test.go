package chaos

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	clihttp "github.com/anoideaopen/channel-transfer/test/integration/clihttp/client"
	"github.com/anoideaopen/channel-transfer/test/integration/clihttp/client/transfer"
	"github.com/anoideaopen/channel-transfer/test/integration/clihttp/models"
	"github.com/anoideaopen/foundation/mocks"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/anoideaopen/foundation/test/integration/cmn/client"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/hyperledger/fabric/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/metadata"
)

const (
	ccCCUpper   = "CC"
	ccFiatUpper = "FIAT"

	emitAmount = "1000"

	fnEmit                      = "emit"
	fnBalanceOf                 = "balanceOf"
	fnAllowedBalanceOf          = "allowedBalanceOf"
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

var _ = Describe("Channel transfer chaos tests", func() {
	var (
		ts       *client.FoundationTestSuite
		channels = []string{cmn.ChannelACL, cmn.ChannelCC, cmn.ChannelFiat}
		user     *mocks.UserFoundation

		clientCtx   context.Context
		transferCli *clihttp.CrossChanelTransfer
		auth        runtime.ClientAuthInfoWriter
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
		ttl := fmt.Sprintf("%ds", ttlExpireTimeout)
		ts.InitNetwork(
			channels,
			integration.IdemixBasePort,
			client.WithChannelTransferTTL(ttl),
		)
		ts.DeployChaincodes()
	})
	BeforeEach(func() {
		By("start robot")
		ts.StartRobot()
		By("start channel transfer")
		ts.StartChannelTransfer()
	})
	AfterEach(func() {
		By("stop robot")
		ts.StopRobot()
		By("stop channel transfer")
		ts.StopChannelTransfer()
		By("stop redis")
		ts.StopRedis()
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
		ts.TxInvokeWithSign(
			cmn.ChannelFiat,
			cmn.ChannelFiat,
			ts.Admin(),
			fnEmit,
			"",
			client.NewNonceByTime().Get(),
			user.AddressBase58Check,
			emitAmount,
		).CheckErrorIsNil()

		By("emit check")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnBalanceOf, user.AddressBase58Check).
			CheckBalance(emitAmount)

		By("creating http connection")
		networkFound := ts.NetworkFound
		clientCtx = metadata.NewOutgoingContext(
			context.Background(),
			metadata.Pairs("authorization", networkFound.ChannelTransfer.AccessToken),
		)

		httpAddress := networkFound.ChannelTransfer.HostAddress + ":" + strconv.FormatUint(uint64(networkFound.ChannelTransfer.Ports[cmn.HTTPPort]), 10)
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
		ts.StopChannelTransfer()

		By("Invoking transaction")
		ts.TxInvokeWithSign(
			cmn.ChannelFiat,
			cmn.ChannelFiat,
			ts.Admin(),
			fnChannelTransferByAdmin,
			"0",
			client.NewNonceByTime().Get(),
			transferID,
			ccCCUpper,
			user.AddressBase58Check,
			ccFiatUpper,
			amount,
		).CheckErrorIsNil()

		By("Starting service")
		ts.StartChannelTransfer()

		By("awaiting for channel transfer to respond")
		waitForAnswerAndCheckStatus(
			clientCtx,
			transferCli,
			transferID,
			authOpts,
			models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED,
			"",
			ts.NetworkFound.EventuallyTimeout*2,
		)

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnBalanceOf, user.AddressBase58Check).
			CheckBalance(restAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnAllowedBalanceOf, user.AddressBase58Check, ccFiatUpper).
			CheckBalance(amount)
	})

	It("Customer transfer before TTL test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("stopping channel transfer")
		ts.StopChannelTransfer()

		ts.TxInvokeWithSign(
			cmn.ChannelFiat,
			cmn.ChannelFiat,
			user,
			fnChannelTransferByCustomer,
			"0",
			client.NewNonceByTime().Get(),
			transferID,
			ccCCUpper,
			ccFiatUpper,
			amount,
		).CheckErrorIsNil()

		By("starting channel transfer")
		ts.StartChannelTransfer()

		By("awaiting for channel transfer to respond")
		waitForAnswerAndCheckStatus(
			clientCtx,
			transferCli,
			transferID,
			authOpts,
			models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED,
			"",
			ts.NetworkFound.EventuallyTimeout*2,
		)

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnBalanceOf, user.AddressBase58Check).
			CheckBalance(restAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnAllowedBalanceOf, user.AddressBase58Check, ccFiatUpper).
			CheckBalance(amount)
	})

	It("Transfer by admin cancel by test", func() {
		amount := "250"
		sendAmount := "0"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("Stopping service")
		ts.StopChannelTransfer()

		By("Invoking transaction")
		ts.TxInvokeWithSign(
			cmn.ChannelFiat,
			cmn.ChannelFiat,
			ts.Admin(),
			fnChannelTransferByAdmin,
			"0",
			client.NewNonceByTime().Get(),
			transferID,
			ccCCUpper,
			user.AddressBase58Check,
			ccFiatUpper,
			amount,
		).CheckErrorIsNil()

		By("Waiting TTL for expiring")
		time.Sleep(time.Second * ttlExpireTimeout)

		By("Starting service")
		ts.StartChannelTransfer()

		time.Sleep(time.Second * 6)

		By("awaiting for channel transfer to respond")
		waitForAnswerAndCheckStatus(
			clientCtx,
			transferCli,
			transferID,
			authOpts,
			models.ChannelTransferTransferStatusResponseStatusSTATUSCANCELED,
			"",
			ts.NetworkFound.EventuallyTimeout*2,
		)

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnBalanceOf, user.AddressBase58Check).
			CheckBalance(emitAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnAllowedBalanceOf, user.AddressBase58Check, ccFiatUpper).
			CheckBalance(sendAmount)
	})

	It("Customer transfer cancel by timeout test", func() {
		amount := "250"
		sendAmount := "0"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("stopping channel transfer")
		ts.StopChannelTransfer()

		ts.TxInvokeWithSign(
			cmn.ChannelFiat,
			cmn.ChannelFiat,
			user,
			fnChannelTransferByCustomer,
			"0",
			client.NewNonceByTime().Get(),
			transferID,
			ccCCUpper,
			ccFiatUpper,
			amount,
		).CheckErrorIsNil()

		By("Waiting TTL for expiring")
		time.Sleep(time.Second * ttlExpireTimeout)

		By("Starting service")
		ts.StartChannelTransfer()

		time.Sleep(time.Second * 6)

		By("awaiting for channel transfer to respond")
		waitForAnswerAndCheckStatus(
			clientCtx,
			transferCli,
			transferID,
			authOpts,
			models.ChannelTransferTransferStatusResponseStatusSTATUSCANCELED,
			"",
			ts.NetworkFound.EventuallyTimeout*2,
		)

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnBalanceOf, user.AddressBase58Check).
			CheckBalance(emitAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnAllowedBalanceOf, user.AddressBase58Check, ccFiatUpper).
			CheckBalance(sendAmount)
	})

	It("Channel Transfer To before TTL test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("Stopping service")
		ts.StopChannelTransfer()

		By("Invoking transaction")
		ts.TxInvokeWithSign(
			cmn.ChannelFiat,
			cmn.ChannelFiat,
			ts.Admin(),
			fnChannelTransferByAdmin,
			"0",
			client.NewNonceByTime().Get(),
			transferID,
			ccCCUpper,
			user.AddressBase58Check,
			ccFiatUpper,
			amount,
		).CheckErrorIsNil()

		By("Querying ChannelTransferFrom")
		from := ""
		fChTrFrom := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}
			from = string(out)
			return ""
		}
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnChannelTransferFrom, transferID).
			CheckResponseWithFunc(fChTrFrom)
		Expect(from).NotTo(BeEmpty())

		By("create cc transfer to")
		ts.TxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, fnCreateCCTransferTo, from).CheckErrorIsNil()

		By("Starting service")
		ts.StartChannelTransfer()

		By("awaiting for channel transfer to respond")
		waitForAnswerAndCheckStatus(
			clientCtx,
			transferCli,
			transferID,
			authOpts,
			models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED,
			"",
			ts.NetworkFound.EventuallyTimeout*2,
		)

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnBalanceOf, user.AddressBase58Check).
			CheckBalance(restAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnAllowedBalanceOf, user.AddressBase58Check, ccFiatUpper).
			CheckBalance(amount)
	})

	It("Channel transfer To after TTL test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("Stopping service")
		ts.StopChannelTransfer()

		By("Invoking transaction")
		ts.TxInvokeWithSign(
			cmn.ChannelFiat,
			cmn.ChannelFiat,
			ts.Admin(),
			fnChannelTransferByAdmin,
			"0",
			client.NewNonceByTime().Get(),
			transferID, ccCCUpper,
			user.AddressBase58Check,
			ccFiatUpper,
			amount,
		).CheckErrorIsNil()

		By("Querying ChannelTransferFrom")
		from := ""
		fChTrFrom := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}
			from = string(out)
			return ""
		}
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnChannelTransferFrom, transferID).
			CheckResponseWithFunc(fChTrFrom)
		Expect(from).NotTo(BeEmpty())

		By("create cc transfer to")
		ts.TxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, fnCreateCCTransferTo, from).CheckErrorIsNil()

		By("Waiting TTL for expiring")
		time.Sleep(time.Second * ttlExpireTimeout)

		By("Starting service")
		ts.StartChannelTransfer()

		By("awaiting for channel transfer to respond")
		waitForAnswerAndCheckStatus(
			clientCtx,
			transferCli,
			transferID,
			authOpts,
			models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED,
			"",
			ts.NetworkFound.EventuallyTimeout*2,
		)

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnBalanceOf, user.AddressBase58Check).
			CheckBalance(restAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnAllowedBalanceOf, user.AddressBase58Check, ccFiatUpper).
			CheckBalance(amount)
	})

	It("Channel transfer From before TTL test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("Stopping service")
		ts.StopChannelTransfer()

		By("Invoking transaction")
		ts.TxInvokeWithSign(
			cmn.ChannelFiat,
			cmn.ChannelFiat,
			ts.Admin(),
			fnChannelTransferByAdmin,
			"0",
			client.NewNonceByTime().Get(),
			transferID,
			ccCCUpper,
			user.AddressBase58Check,
			ccFiatUpper,
			amount,
		).CheckErrorIsNil()

		By("Querying ChannelTransferFrom")
		from := ""
		fChTrFrom := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}
			from = string(out)
			return ""
		}
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnChannelTransferFrom, transferID).
			CheckResponseWithFunc(fChTrFrom)
		Expect(from).NotTo(BeEmpty())

		By("create cc transfer to")
		ts.TxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, fnCreateCCTransferTo, from).CheckErrorIsNil()

		By("channel transfer to")
		fChTrTo := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}

			return ""
		}
		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnChannelTransferTo, transferID).CheckResponseWithFunc(fChTrTo)

		By("commit cc transfer from")
		ts.NBTxInvokeByRobot(cmn.ChannelFiat, cmn.ChannelFiat, fnCommitCCTransferFrom, transferID).CheckErrorIsNil()

		By("Starting service")
		ts.StartChannelTransfer()

		By("awaiting for channel transfer to respond")
		waitForAnswerAndCheckStatus(
			clientCtx,
			transferCli,
			transferID,
			authOpts,
			models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED,
			"",
			ts.NetworkFound.EventuallyTimeout*2,
		)

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnBalanceOf, user.AddressBase58Check).
			CheckBalance(restAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnAllowedBalanceOf, user.AddressBase58Check, ccFiatUpper).
			CheckBalance(amount)
	})

	It("Channel transfer From after TTL test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("Stopping service")
		ts.StopChannelTransfer()

		By("Invoking transaction")
		ts.TxInvokeWithSign(
			cmn.ChannelFiat,
			cmn.ChannelFiat,
			ts.Admin(),
			fnChannelTransferByAdmin,
			"0",
			client.NewNonceByTime().Get(),
			transferID,
			ccCCUpper,
			user.AddressBase58Check,
			ccFiatUpper,
			amount,
		).CheckErrorIsNil()

		By("Querying ChannelTransferFrom")
		from := ""
		fChTrFrom := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}
			from = string(out)
			return ""
		}
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnChannelTransferFrom, transferID).
			CheckResponseWithFunc(fChTrFrom)
		Expect(from).NotTo(BeEmpty())

		By("create cc transfer to")
		ts.TxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, fnCreateCCTransferTo, from).CheckErrorIsNil()

		By("channel transfer to")
		fChTrTo := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}

			return ""
		}
		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnChannelTransferTo, transferID).
			CheckResponseWithFunc(fChTrTo)

		By("commit cc transfer from")
		ts.NBTxInvokeByRobot(cmn.ChannelFiat, cmn.ChannelFiat, fnCommitCCTransferFrom, transferID).CheckErrorIsNil()

		By("Waiting TTL for expiring")
		time.Sleep(time.Second * ttlExpireTimeout)

		By("Starting service")
		ts.StartChannelTransfer()

		By("awaiting for channel transfer to respond")
		waitForAnswerAndCheckStatus(
			clientCtx,
			transferCli,
			transferID,
			authOpts,
			models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED,
			"",
			ts.NetworkFound.EventuallyTimeout*2,
		)

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnBalanceOf, user.AddressBase58Check).
			CheckBalance(restAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnAllowedBalanceOf, user.AddressBase58Check, ccFiatUpper).
			CheckBalance(amount)
	})

	It("Delete transfer To before TTL test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("Stopping service")
		ts.StopChannelTransfer()

		By("Invoking transaction")
		ts.TxInvokeWithSign(
			cmn.ChannelFiat,
			cmn.ChannelFiat,
			ts.Admin(),
			fnChannelTransferByAdmin,
			"0",
			client.NewNonceByTime().Get(),
			transferID,
			ccCCUpper,
			user.AddressBase58Check,
			ccFiatUpper,
			amount,
		).CheckErrorIsNil()

		By("Querying ChannelTransferFrom")
		from := ""
		fChTrFrom := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}
			from = string(out)
			return ""
		}
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnChannelTransferFrom, transferID).
			CheckResponseWithFunc(fChTrFrom)
		Expect(from).NotTo(BeEmpty())

		By("create cc transfer to")
		ts.TxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, fnCreateCCTransferTo, from).CheckErrorIsNil()

		By("channel transfer to")
		fChTrTo := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}

			return ""
		}
		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnChannelTransferTo, transferID).
			CheckResponseWithFunc(fChTrTo)

		By("commit cc transfer from")
		ts.NBTxInvokeByRobot(cmn.ChannelFiat, cmn.ChannelFiat, fnCommitCCTransferFrom, transferID).CheckErrorIsNil()

		By("delete cc transfer to")
		ts.NBTxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, fnDeleteCCTransferTo, transferID).CheckErrorIsNil()

		By("Starting service")
		ts.StartChannelTransfer()

		By("awaiting for channel transfer to respond")
		waitForAnswerAndCheckStatus(
			clientCtx,
			transferCli,
			transferID,
			authOpts,
			models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED,
			"",
			ts.NetworkFound.EventuallyTimeout*2,
		)

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnBalanceOf, user.AddressBase58Check).
			CheckBalance(restAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnAllowedBalanceOf, user.AddressBase58Check, ccFiatUpper).
			CheckBalance(amount)
	})

	It("Delete transfer To after TTL test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("Stopping service")
		ts.StopChannelTransfer()

		By("Invoking transaction")
		ts.TxInvokeWithSign(
			cmn.ChannelFiat,
			cmn.ChannelFiat,
			ts.Admin(),
			fnChannelTransferByAdmin,
			"0",
			client.NewNonceByTime().Get(),
			transferID,
			ccCCUpper,
			user.AddressBase58Check,
			ccFiatUpper,
			amount,
		).CheckErrorIsNil()

		By("Querying ChannelTransferFrom")
		from := ""
		fChTrFrom := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}
			from = string(out)
			return ""
		}
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnChannelTransferFrom, transferID).
			CheckResponseWithFunc(fChTrFrom)
		Expect(from).NotTo(BeEmpty())

		By("create cc transfer to")
		ts.TxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, fnCreateCCTransferTo, from).CheckErrorIsNil()

		By("channel transfer to")
		fChTrTo := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}

			return ""
		}
		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnChannelTransferTo, transferID).
			CheckResponseWithFunc(fChTrTo)

		By("commit cc transfer from")
		ts.NBTxInvokeByRobot(cmn.ChannelFiat, cmn.ChannelFiat, fnCommitCCTransferFrom, transferID).CheckErrorIsNil()

		By("delete cc transfer to")
		ts.NBTxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, fnDeleteCCTransferTo, transferID).CheckErrorIsNil()

		By("Waiting TTL for expiring")
		time.Sleep(time.Second * ttlExpireTimeout)

		By("Starting service")
		ts.StartChannelTransfer()

		By("awaiting for channel transfer to respond")
		waitForAnswerAndCheckStatus(
			clientCtx,
			transferCli,
			transferID,
			authOpts,
			models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED,
			"",
			ts.NetworkFound.EventuallyTimeout*2,
		)

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnBalanceOf, user.AddressBase58Check).
			CheckBalance(restAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnAllowedBalanceOf, user.AddressBase58Check, ccFiatUpper).
			CheckBalance(amount)
	})

	It("Delete transfer From before TTL test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("Stopping service")
		ts.StopChannelTransfer()

		By("Invoking transaction")
		ts.TxInvokeWithSign(
			cmn.ChannelFiat,
			cmn.ChannelFiat,
			ts.Admin(),
			fnChannelTransferByAdmin,
			"0",
			client.NewNonceByTime().Get(),
			transferID,
			ccCCUpper,
			user.AddressBase58Check,
			ccFiatUpper,
			amount,
		).CheckErrorIsNil()

		By("Querying ChannelTransferFrom")
		from := ""
		fChTrFrom := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}
			from = string(out)
			return ""
		}
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnChannelTransferFrom, transferID).
			CheckResponseWithFunc(fChTrFrom)
		Expect(from).NotTo(BeEmpty())

		By("create cc transfer to")
		ts.TxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, fnCreateCCTransferTo, from).CheckErrorIsNil()

		By("channel transfer to")
		fChTrTo := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}

			return ""
		}
		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnChannelTransferTo, transferID).
			CheckResponseWithFunc(fChTrTo)

		By("commit cc transfer from")
		ts.NBTxInvokeByRobot(cmn.ChannelFiat, cmn.ChannelFiat, fnCommitCCTransferFrom, transferID).CheckErrorIsNil()

		By("delete cc transfer to")
		ts.NBTxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, fnDeleteCCTransferTo, transferID).CheckErrorIsNil()

		By("delete cc transfer from")
		ts.NBTxInvokeByRobot(cmn.ChannelFiat, cmn.ChannelFiat, fnDeleteCCTransferFrom, transferID).CheckErrorIsNil()

		By("Starting service")
		ts.StartChannelTransfer()

		By("awaiting for channel transfer to respond")
		waitForAnswerAndCheckStatus(
			clientCtx,
			transferCli,
			transferID,
			authOpts,
			models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED,
			"",
			ts.NetworkFound.EventuallyTimeout*2,
		)

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnBalanceOf, user.AddressBase58Check).
			CheckBalance(restAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnAllowedBalanceOf, user.AddressBase58Check, ccFiatUpper).
			CheckBalance(amount)
	})

	It("Delete transfer From after TTL test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		transferID := uuid.NewString()

		By("Stopping service")
		ts.StopChannelTransfer()

		By("Invoking transaction")
		ts.TxInvokeWithSign(
			cmn.ChannelFiat,
			cmn.ChannelFiat,
			ts.Admin(),
			fnChannelTransferByAdmin,
			"0",
			client.NewNonceByTime().Get(),
			transferID,
			ccCCUpper,
			user.AddressBase58Check,
			ccFiatUpper,
			amount,
		).CheckErrorIsNil()

		By("Querying ChannelTransferFrom")
		from := ""
		fChTrFrom := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}
			from = string(out)
			return ""
		}
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnChannelTransferFrom, transferID).
			CheckResponseWithFunc(fChTrFrom)
		Expect(from).NotTo(BeEmpty())

		By("create cc transfer to")
		ts.TxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, fnCreateCCTransferTo, from).CheckErrorIsNil()

		By("channel transfer to")
		fChTrTo := func(out []byte) string {
			if len(out) == 0 {
				return "out is empty"
			}

			return ""
		}
		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnChannelTransferTo, transferID).
			CheckResponseWithFunc(fChTrTo)

		By("commit cc transfer from")
		ts.NBTxInvokeByRobot(cmn.ChannelFiat, cmn.ChannelFiat, fnCommitCCTransferFrom, transferID).CheckErrorIsNil()

		By("delete cc transfer to")
		ts.NBTxInvokeByRobot(cmn.ChannelCC, cmn.ChannelCC, fnDeleteCCTransferTo, transferID).CheckErrorIsNil()

		By("delete cc transfer from")
		ts.NBTxInvokeByRobot(cmn.ChannelFiat, cmn.ChannelFiat, fnDeleteCCTransferFrom, transferID).CheckErrorIsNil()

		By("Waiting TTL for expiring")
		time.Sleep(time.Second * ttlExpireTimeout)

		By("Starting service")
		ts.StartChannelTransfer()

		By("awaiting for channel transfer to respond")
		waitForAnswerAndCheckStatus(
			clientCtx,
			transferCli,
			transferID,
			authOpts,
			models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED,
			"",
			ts.NetworkFound.EventuallyTimeout*2,
		)

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat, fnBalanceOf, user.AddressBase58Check).
			CheckBalance(restAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC, fnAllowedBalanceOf, user.AddressBase58Check, ccFiatUpper).
			CheckBalance(amount)
	})
})

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
	eventuallyTimeout time.Duration,
) {
	Eventually(func() error {
		response, err := transferCli.Transfer.TransferStatus(&transfer.TransferStatusParams{IDTransfer: transferID, Context: clientCtx}, authOpts)
		if err != nil {
			return err
		}
		if expectedStatus != models.ChannelTransferTransferStatusResponseStatusSTATUSERROR && *response.Payload.Status == models.ChannelTransferTransferStatusResponseStatusSTATUSERROR {
			return fmt.Errorf("error occured: %s", response.Payload.Message)
		}
		if err = checkResponseStatus(response.Payload, expectedStatus, expectedError); err != nil {
			return err
		}

		return nil
	}, eventuallyTimeout, time.Second).ShouldNot(HaveOccurred())
}
