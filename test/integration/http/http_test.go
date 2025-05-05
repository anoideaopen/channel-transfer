package http

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
	"github.com/btcsuite/btcd/btcutil/base58"
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

var _ = Describe("Channel transfer HTTP tests", func() {
	var ts *client.FoundationTestSuite

	BeforeEach(func() {
		ts = client.NewTestSuite(components)
	})

	AfterEach(func() {
		ts.ShutdownNetwork()
	})

	var (
		channels = []string{cmn.ChannelACL, cmn.ChannelCC, cmn.ChannelFiat, cmn.ChannelIndustrial}
		user     *mocks.UserFoundation

		networkFound *cmn.NetworkFoundation

		clientCtx   context.Context
		transferCli *clihttp.CrossChanelTransfer
		auth        runtime.ClientAuthInfoWriter
	)
	BeforeEach(func() {
		By("start redis")
		ts.StartRedis()
	})
	BeforeEach(func() {
		ts.InitNetwork(
			channels,
			integration.GossipBasePort,
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
		networkFound = ts.NetworkFound

		By("add admin to acl")
		ts.AddAdminToACL()

		By("add user to acl")
		var err error
		user, err = mocks.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())

		ts.AddUser(user)

		By("emit tokens")
		ts.TxInvokeWithSign(
			cmn.ChannelFiat, cmn.ChannelFiat, ts.Admin(),
			"emit", "", client.NewNonceByTime().Get(), user.AddressBase58Check, emitAmount).CheckErrorIsNil()

		By("emit check")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
			"balanceOf", user.AddressBase58Check).CheckBalance(emitAmount)

		By("creating http connection")
		clientCtx = metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", networkFound.ChannelTransfer.AccessToken))

		httpAddress := networkFound.ChannelTransfer.HostAddress + ":" + strconv.FormatUint(uint64(networkFound.ChannelTransfer.Ports[cmn.HTTPPort]), 10)
		transport := httptransport.New(httpAddress, "", nil)
		transferCli = clihttp.New(transport, strfmt.Default)

		auth = httptransport.APIKeyAuth("authorization", "header", networkFound.ChannelTransfer.AccessToken)
	})

	It("transfer by admin test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		By("creating channel transfer request")
		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, ccCCUpper, user.AddressBase58Check, ccFiatUpper, amount}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{fnChannelTransferByAdmin, requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
		publicKey, sign, err := ts.Admin().Sign(signArgs...)
		Expect(err).NotTo(HaveOccurred())

		transferRequest := &models.ChannelTransferTransferBeginAdminRequest{
			Generals: &models.ChannelTransferGeneralParams{
				MethodName: fnChannelTransferByAdmin,
				RequestID:  requestID,
				Chaincode:  cmn.ChannelFiat,
				Channel:    cmn.ChannelFiat,
				Nonce:      nonce,
				PublicKey:  publicKey,
				Sign:       base58.Encode(sign),
			},
			IDTransfer: channelTransferArgs[0],
			ChannelTo:  channelTransferArgs[1],
			Address:    channelTransferArgs[2],
			Token:      channelTransferArgs[3],
			Amount:     channelTransferArgs[4],
		}

		By("sending transfer request")
		res, err := transferCli.Transfer.TransferByAdmin(&transfer.TransferByAdminParams{Body: transferRequest, Context: clientCtx}, authOpts)
		Expect(err).NotTo(HaveOccurred())

		err = checkResponseStatus(res.GetPayload(), models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, "")
		Expect(err).NotTo(HaveOccurred())

		By("awaiting for channel transfer to respond")
		waitForAnswerAndCheckStatus(
			clientCtx,
			transferCli,
			transferID,
			authOpts,
			models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED,
			"",
			networkFound.EventuallyTimeout*2)

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
			"balanceOf", user.AddressBase58Check).CheckBalance(restAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC,
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper).CheckBalance(amount)
	})

	It("transfer by customer test", func() {
		amount := "250"
		restAmount := "750"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		By("creating channel transfer request")
		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, ccCCUpper, ccFiatUpper, amount}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{fnChannelTransferByCustomer, requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
		publicKey, sign, err := user.Sign(signArgs...)
		Expect(err).NotTo(HaveOccurred())

		transferRequest := &models.ChannelTransferTransferBeginCustomerRequest{
			Generals: &models.ChannelTransferGeneralParams{
				MethodName: fnChannelTransferByCustomer,
				RequestID:  requestID,
				Chaincode:  cmn.ChannelFiat,
				Channel:    cmn.ChannelFiat,
				Nonce:      nonce,
				PublicKey:  publicKey,
				Sign:       base58.Encode(sign),
			},
			IDTransfer: channelTransferArgs[0],
			ChannelTo:  channelTransferArgs[1],
			Token:      channelTransferArgs[2],
			Amount:     channelTransferArgs[3],
		}

		By("sending transfer request")
		res, err := transferCli.Transfer.TransferByCustomer(&transfer.TransferByCustomerParams{Body: transferRequest, Context: clientCtx}, authOpts)
		Expect(err).NotTo(HaveOccurred())

		err = checkResponseStatus(res.GetPayload(), models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, "")
		Expect(err).NotTo(HaveOccurred())

		By("awaiting for channel transfer to respond")
		waitForAnswerAndCheckStatus(
			clientCtx,
			transferCli,
			transferID,
			authOpts,
			models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED,
			"",
			networkFound.EventuallyTimeout*2,
		)

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
			"balanceOf", user.AddressBase58Check).CheckBalance(restAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC,
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper).CheckBalance(amount)
	})

	It("transfer status with wrong transfer id test", func() {
		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		By("requesting status of transfer with id = 1")
		_, err := transferCli.Transfer.TransferStatus(&transfer.TransferStatusParams{IDTransfer: "1", Context: context.Background()}, authOpts)
		Expect(err).To(MatchError(ContainSubstring("object not found")))
	})

	It("admin transfer to wrong channel", func() {
		amount := "250"
		sendAmount := "0"
		wrongChannel := "ASD"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		By("creating channel transfer request")
		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, wrongChannel, user.AddressBase58Check, ccFiatUpper, amount}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{fnChannelTransferByAdmin, requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
		publicKey, sign, err := ts.Admin().Sign(signArgs...)
		Expect(err).NotTo(HaveOccurred())

		transferRequest := &models.ChannelTransferTransferBeginAdminRequest{
			Generals: &models.ChannelTransferGeneralParams{
				MethodName: fnChannelTransferByAdmin,
				RequestID:  requestID,
				Chaincode:  cmn.ChannelFiat,
				Channel:    cmn.ChannelFiat,
				Nonce:      nonce,
				PublicKey:  publicKey,
				Sign:       base58.Encode(sign),
			},
			IDTransfer: channelTransferArgs[0],
			ChannelTo:  channelTransferArgs[1],
			Address:    channelTransferArgs[2],
			Token:      channelTransferArgs[3],
			Amount:     channelTransferArgs[4],
		}

		By("sending transfer request")
		res, err := transferCli.Transfer.TransferByAdmin(&transfer.TransferByAdminParams{Body: transferRequest, Context: clientCtx}, authOpts)
		Expect(err).NotTo(HaveOccurred())

		err = checkResponseStatus(res.GetPayload(), models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, "")
		Expect(err).NotTo(HaveOccurred())

		By("awaiting for channel transfer to respond")
		waitForAnswerAndCheckStatus(
			clientCtx,
			transferCli,
			transferID,
			authOpts,
			models.ChannelTransferTransferStatusResponseStatusSTATUSERROR,
			errWrongChannel,
			networkFound.EventuallyTimeout*2,
		)

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
			"balanceOf", user.AddressBase58Check).CheckBalance(emitAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC,
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper).CheckBalance(sendAmount)
	})

	It("client transfer to wrong channel", func() {
		amount := "250"
		sendAmount := "0"
		wrongChannel := "ASD"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		By("creating channel transfer request")
		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, wrongChannel, ccFiatUpper, amount}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{fnChannelTransferByCustomer, requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
		publicKey, sign, err := user.Sign(signArgs...)
		Expect(err).NotTo(HaveOccurred())

		transferRequest := &models.ChannelTransferTransferBeginCustomerRequest{
			Generals: &models.ChannelTransferGeneralParams{
				MethodName: fnChannelTransferByCustomer,
				RequestID:  requestID,
				Chaincode:  cmn.ChannelFiat,
				Channel:    cmn.ChannelFiat,
				Nonce:      nonce,
				PublicKey:  publicKey,
				Sign:       base58.Encode(sign),
			},
			IDTransfer: channelTransferArgs[0],
			ChannelTo:  channelTransferArgs[1],
			Token:      channelTransferArgs[2],
			Amount:     channelTransferArgs[3],
		}

		By("sending transfer request")
		res, err := transferCli.Transfer.TransferByCustomer(&transfer.TransferByCustomerParams{Body: transferRequest, Context: clientCtx}, authOpts)
		Expect(err).NotTo(HaveOccurred())

		err = checkResponseStatus(res.GetPayload(), models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, "")
		Expect(err).NotTo(HaveOccurred())

		By("awaiting for channel transfer to respond")
		waitForAnswerAndCheckStatus(
			clientCtx,
			transferCli,
			transferID,
			authOpts,
			models.ChannelTransferTransferStatusResponseStatusSTATUSERROR,
			errWrongChannel,
			networkFound.EventuallyTimeout*2,
		)

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
			"balanceOf", user.AddressBase58Check).CheckBalance(emitAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC,
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper).CheckBalance(sendAmount)
	})

	It("admin transfer to not valid channel", func() {
		amount := "250"
		sendAmount := "0"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		By("creating channel transfer request")
		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, ccACLUpper, user.AddressBase58Check, ccACLUpper, amount}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{fnChannelTransferByAdmin, requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
		publicKey, sign, err := ts.Admin().Sign(signArgs...)
		Expect(err).NotTo(HaveOccurred())

		transferRequest := &models.ChannelTransferTransferBeginAdminRequest{
			Generals: &models.ChannelTransferGeneralParams{
				MethodName: fnChannelTransferByAdmin,
				RequestID:  requestID,
				Chaincode:  cmn.ChannelACL,
				Channel:    cmn.ChannelACL,
				Nonce:      nonce,
				PublicKey:  publicKey,
				Sign:       base58.Encode(sign),
			},
			IDTransfer: channelTransferArgs[0],
			ChannelTo:  ccFiatUpper,
			Address:    channelTransferArgs[2],
			Token:      channelTransferArgs[3],
			Amount:     channelTransferArgs[4],
		}

		By("sending transfer request")
		_, err = transferCli.Transfer.TransferByAdmin(&transfer.TransferByAdminParams{Body: transferRequest, Context: clientCtx}, authOpts)
		Expect(err).Should(ContainSubstring(errNotValidChannel))

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
			"balanceOf", user.AddressBase58Check).CheckBalance(emitAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC,
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper).CheckBalance(sendAmount)
	})

	It("client transfer to not valid channel", func() {
		amount := "250"
		sendAmount := "0"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		By("creating channel transfer request")
		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, ccACLUpper, ccACLUpper, amount}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{fnChannelTransferByCustomer, requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
		publicKey, sign, err := user.Sign(signArgs...)
		Expect(err).NotTo(HaveOccurred())

		transferRequest := &models.ChannelTransferTransferBeginCustomerRequest{
			Generals: &models.ChannelTransferGeneralParams{
				MethodName: fnChannelTransferByCustomer,
				RequestID:  requestID,
				Chaincode:  cmn.ChannelACL,
				Channel:    cmn.ChannelACL,
				Nonce:      nonce,
				PublicKey:  publicKey,
				Sign:       base58.Encode(sign),
			},
			IDTransfer: channelTransferArgs[0],
			ChannelTo:  ccFiatUpper,
			Token:      channelTransferArgs[2],
			Amount:     channelTransferArgs[3],
		}

		By("sending transfer request")
		_, err = transferCli.Transfer.TransferByCustomer(&transfer.TransferByCustomerParams{Body: transferRequest, Context: clientCtx}, authOpts)
		Expect(err).Should(ContainSubstring(errNotValidChannel))

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
			"balanceOf", user.AddressBase58Check).CheckBalance(emitAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC,
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper).CheckBalance(sendAmount)
	})

	It("admin transfer insufficient funds", func() {
		amount := "250"
		sendAmount := "0"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		By("creating new user")
		user1, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())

		ts.AddUser(user1)

		By("creating channel transfer request")
		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, ccCCUpper, user1.AddressBase58Check, ccFiatUpper, amount}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{fnChannelTransferByAdmin, requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
		publicKey, sign, err := ts.Admin().Sign(signArgs...)
		Expect(err).NotTo(HaveOccurred())

		transferRequest := &models.ChannelTransferTransferBeginAdminRequest{
			Generals: &models.ChannelTransferGeneralParams{
				MethodName: fnChannelTransferByAdmin,
				RequestID:  requestID,
				Chaincode:  cmn.ChannelFiat,
				Channel:    cmn.ChannelFiat,
				Nonce:      nonce,
				PublicKey:  publicKey,
				Sign:       base58.Encode(sign),
			},
			IDTransfer: channelTransferArgs[0],
			ChannelTo:  channelTransferArgs[1],
			Address:    channelTransferArgs[2],
			Token:      channelTransferArgs[3],
			Amount:     channelTransferArgs[4],
		}

		By("sending transfer request")
		res, err := transferCli.Transfer.TransferByAdmin(&transfer.TransferByAdminParams{Body: transferRequest, Context: clientCtx}, authOpts)
		Expect(err).NotTo(HaveOccurred())

		err = checkResponseStatus(res.GetPayload(), models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, "")
		Expect(err).NotTo(HaveOccurred())

		By("awaiting for channel transfer to respond")
		waitForAnswerAndCheckStatus(
			clientCtx,
			transferCli,
			transferID,
			authOpts,
			models.ChannelTransferTransferStatusResponseStatusSTATUSERROR,
			errInsufficientFunds,
			networkFound.EventuallyTimeout*2,
		)

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
			"balanceOf", user.AddressBase58Check).CheckBalance(emitAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC,
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper).CheckBalance(sendAmount)
	})

	It("customer transfer insufficient funds", func() {
		amount := "250"
		sendAmount := "0"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		By("creating new user")
		user1, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
		Expect(err).NotTo(HaveOccurred())

		ts.AddUser(user1)

		By("creating channel transfer request")
		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, ccCCUpper, ccFiatUpper, amount}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{fnChannelTransferByCustomer, requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
		publicKey, sign, err := user1.Sign(signArgs...)
		Expect(err).NotTo(HaveOccurred())

		transferRequest := &models.ChannelTransferTransferBeginCustomerRequest{
			Generals: &models.ChannelTransferGeneralParams{
				MethodName: fnChannelTransferByCustomer,
				RequestID:  requestID,
				Chaincode:  cmn.ChannelFiat,
				Channel:    cmn.ChannelFiat,
				Nonce:      nonce,
				PublicKey:  publicKey,
				Sign:       base58.Encode(sign),
			},
			IDTransfer: channelTransferArgs[0],
			ChannelTo:  channelTransferArgs[1],
			Token:      channelTransferArgs[2],
			Amount:     channelTransferArgs[3],
		}

		By("sending transfer request")
		res, err := transferCli.Transfer.TransferByCustomer(&transfer.TransferByCustomerParams{Body: transferRequest, Context: clientCtx}, authOpts)
		Expect(err).NotTo(HaveOccurred())

		err = checkResponseStatus(res.GetPayload(), models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, "")
		Expect(err).NotTo(HaveOccurred())

		By("awaiting for channel transfer to respond")
		waitForAnswerAndCheckStatus(
			clientCtx,
			transferCli,
			transferID,
			authOpts,
			models.ChannelTransferTransferStatusResponseStatusSTATUSERROR,
			errInsufficientFunds, networkFound.EventuallyTimeout*2,
		)

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
			"balanceOf", user.AddressBase58Check).CheckBalance(emitAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC,
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper).CheckBalance(sendAmount)
	})

	It("admin transfer wrong channel from", func() {
		amount := "250"
		sendAmount := "0"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		By("creating channel transfer request")
		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, ccCCUpper, user.AddressBase58Check, ccIndustrialUpper, amount}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{fnChannelTransferByAdmin, requestID, cmn.ChannelIndustrial, cmn.ChannelIndustrial}, channelTransferArgs...), nonce)
		publicKey, sign, err := ts.Admin().Sign(signArgs...)
		Expect(err).NotTo(HaveOccurred())

		transferRequest := &models.ChannelTransferTransferBeginAdminRequest{
			Generals: &models.ChannelTransferGeneralParams{
				MethodName: fnChannelTransferByAdmin,
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
			Token:      channelTransferArgs[3],
			Amount:     channelTransferArgs[4],
		}

		By("sending transfer request")
		res, err := transferCli.Transfer.TransferByAdmin(&transfer.TransferByAdminParams{Body: transferRequest, Context: clientCtx}, authOpts)
		Expect(err).NotTo(HaveOccurred())

		err = checkResponseStatus(res.GetPayload(), models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, "")
		Expect(err).NotTo(HaveOccurred())

		By("awaiting for channel transfer to respond")
		waitForAnswerAndCheckStatus(
			clientCtx,
			transferCli,
			transferID,
			authOpts,
			models.ChannelTransferTransferStatusResponseStatusSTATUSERROR,
			errInsufficientFunds,
			networkFound.EventuallyTimeout*2,
		)

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
			"balanceOf", user.AddressBase58Check).CheckBalance(emitAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC,
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper).CheckBalance(sendAmount)
	})

	It("customer transfer wrong channel from", func() {
		amount := "250"
		sendAmount := "0"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		By("creating channel transfer request")
		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, ccCCUpper, ccIndustrialUpper, amount}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{fnChannelTransferByCustomer, requestID, cmn.ChannelIndustrial, cmn.ChannelIndustrial}, channelTransferArgs...), nonce)
		publicKey, sign, err := user.Sign(signArgs...)
		Expect(err).NotTo(HaveOccurred())

		transferRequest := &models.ChannelTransferTransferBeginCustomerRequest{
			Generals: &models.ChannelTransferGeneralParams{
				MethodName: fnChannelTransferByCustomer,
				RequestID:  requestID,
				Chaincode:  cmn.ChannelIndustrial,
				Channel:    cmn.ChannelIndustrial,
				Nonce:      nonce,
				PublicKey:  publicKey,
				Sign:       base58.Encode(sign),
			},
			IDTransfer: channelTransferArgs[0],
			ChannelTo:  channelTransferArgs[1],
			Token:      channelTransferArgs[2],
			Amount:     channelTransferArgs[3],
		}

		By("sending transfer request")
		res, err := transferCli.Transfer.TransferByCustomer(&transfer.TransferByCustomerParams{Body: transferRequest, Context: clientCtx}, authOpts)
		Expect(err).NotTo(HaveOccurred())

		err = checkResponseStatus(res.GetPayload(), models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, "")
		Expect(err).NotTo(HaveOccurred())

		By("awaiting for channel transfer to respond")
		waitForAnswerAndCheckStatus(
			clientCtx,
			transferCli,
			transferID,
			authOpts,
			models.ChannelTransferTransferStatusResponseStatusSTATUSERROR,
			errInsufficientFunds,
			networkFound.EventuallyTimeout*2,
		)

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
			"balanceOf", user.AddressBase58Check).CheckBalance(emitAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC,
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper).CheckBalance(sendAmount)
	})

	It("admin transfer wrong channel to", func() {
		amount := "250"
		sendAmount := "0"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		By("creating channel transfer request")
		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, ccCCUpper, user.AddressBase58Check, ccIndustrialUpper, amount}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{fnChannelTransferByAdmin, requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
		publicKey, sign, err := ts.Admin().Sign(signArgs...)
		Expect(err).NotTo(HaveOccurred())

		transferRequest := &models.ChannelTransferTransferBeginAdminRequest{
			Generals: &models.ChannelTransferGeneralParams{
				MethodName: fnChannelTransferByAdmin,
				RequestID:  requestID,
				Chaincode:  cmn.ChannelFiat,
				Channel:    cmn.ChannelFiat,
				Nonce:      nonce,
				PublicKey:  publicKey,
				Sign:       base58.Encode(sign),
			},
			IDTransfer: channelTransferArgs[0],
			ChannelTo:  channelTransferArgs[1],
			Address:    channelTransferArgs[2],
			Token:      channelTransferArgs[3],
			Amount:     channelTransferArgs[4],
		}

		By("sending transfer request")
		_, err = transferCli.Transfer.TransferByAdmin(&transfer.TransferByAdminParams{Body: transferRequest, Context: clientCtx}, authOpts)
		Expect(err).Should(ContainSubstring(errIncorrectToken))

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
			"balanceOf", user.AddressBase58Check).CheckBalance(emitAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC,
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper).CheckBalance(sendAmount)
	})

	It("customer transfer wrong channel to", func() {
		amount := "250"
		sendAmount := "0"

		authOpts := func(c *runtime.ClientOperation) {
			c.AuthInfo = auth
		}

		By("creating channel transfer request")
		transferID := uuid.NewString()
		channelTransferArgs := []string{transferID, ccCCUpper, ccIndustrialUpper, amount}

		requestID := uuid.NewString()
		nonce := client.NewNonceByTime().Get()
		signArgs := append(append([]string{fnChannelTransferByCustomer, requestID, cmn.ChannelFiat, cmn.ChannelFiat}, channelTransferArgs...), nonce)
		publicKey, sign, err := user.Sign(signArgs...)
		Expect(err).NotTo(HaveOccurred())

		transferRequest := &models.ChannelTransferTransferBeginCustomerRequest{
			Generals: &models.ChannelTransferGeneralParams{
				MethodName: fnChannelTransferByCustomer,
				RequestID:  requestID,
				Chaincode:  cmn.ChannelFiat,
				Channel:    cmn.ChannelFiat,
				Nonce:      nonce,
				PublicKey:  publicKey,
				Sign:       base58.Encode(sign),
			},
			IDTransfer: channelTransferArgs[0],
			ChannelTo:  channelTransferArgs[1],
			Token:      channelTransferArgs[2],
			Amount:     channelTransferArgs[3],
		}

		By("sending transfer request")
		_, err = transferCli.Transfer.TransferByCustomer(&transfer.TransferByCustomerParams{Body: transferRequest, Context: clientCtx}, authOpts)
		Expect(err).Should(ContainSubstring(errIncorrectToken))

		By("checking result balances")
		ts.Query(cmn.ChannelFiat, cmn.ChannelFiat,
			"balanceOf", user.AddressBase58Check).CheckBalance(emitAmount)

		ts.Query(cmn.ChannelCC, cmn.ChannelCC,
			"allowedBalanceOf", user.AddressBase58Check, ccFiatUpper).CheckBalance(sendAmount)
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
	eventuallyTimeout time.Duration,
) {
	Eventually(func() error {
		response, err := transferCli.Transfer.TransferStatus(&transfer.TransferStatusParams{
			IDTransfer: transferID,
			Context:    clientCtx,
		}, authOpts)
		if err != nil {
			return err
		}
		if expectedStatus != models.ChannelTransferTransferStatusResponseStatusSTATUSERROR &&
			*response.Payload.Status == models.ChannelTransferTransferStatusResponseStatusSTATUSERROR {
			return fmt.Errorf("error occured: %s", response.Payload.Message)
		}
		if err = checkResponseStatus(response.Payload, expectedStatus, expectedError); err != nil {
			return err
		}

		return nil
	}, eventuallyTimeout, time.Second).ShouldNot(HaveOccurred())
}
