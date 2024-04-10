package test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/anoideaopen/channel-transfer/test/integration/clienthttp/models"
	"github.com/anoideaopen/channel-transfer/test/integration/clienthttp/transfer"
	utils "github.com/anoideaopen/testnet-util"
	tr "github.com/anoideaopen/testnet-util/transfer"
	"github.com/google/uuid"
	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/runner"
)

const (
	channelFrom                     = "fiat"
	channelTransferByAdminMethod    = "channelTransferByAdmin"
	channelTransferByCustomerMethod = "channelTransferByCustomer"

	channelFiatUppercase       = "FIAT"
	channelCCUppercase         = "CC"
	channelIndustrialUppercase = "INDUSTRIAL"
	insufficientFundsErr       = "insufficient funds to process"
	channelErr                 = "no channel peers configured for channel"
	notValidChannelErr         = "channel not in configuration list"
	unknownMethodErr           = "unknown method"
)

// TestChannelTransferByAdminHTTP - add user emit fiat tokens and transfer to channel cc with channelTransferByAdmin by HTTP then check balance and allowedBalance
func TestChannelTransferByAdminHTTP(t *testing.T) {
	runner.Run(t, "Channel transfer by admin", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("Channel transfer by admin http")
		t.Tags("positive", "chtransfer")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))

		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)

		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, emitAmount)

		t.WithNewStep("Channel transfer by admin http", func(sCtx provider.StepCtx) {
			transferID := uuid.NewString()
			requestID := uuid.NewString()
			channelTransferArgs := []string{transferID, channelCCUppercase, user.UserAddressBase58Check, channelFiatUppercase, transferAmount}

			sa, err := utils.SignExpand(issuer.IssuerEd25519PrivateKey, issuer.IssuerEd25519PublicKey, channelFrom, channelFrom, channelTransferByAdminMethod, channelTransferArgs, utils.GetNonce(), requestID)
			sCtx.Require().NoError(err)

			transferRequest := &models.ChannelTransferTransferBeginAdminRequest{
				Generals: &models.ChannelTransferGeneralParams{
					MethodName: channelTransferByAdminMethod,
					RequestID:  requestID,
					Chaincode:  channelFrom,
					Channel:    channelFrom,
					Nonce:      sa[len(sa)-3],
					PublicKey:  sa[len(sa)-2],
					Sign:       sa[len(sa)-1],
				},
				IDTransfer: transferID,
				ChannelTo:  channelCCUppercase,
				Address:    user.UserAddressBase58Check,
				Token:      channelFiatUppercase,
				Amount:     transferAmount,
			}

			res, err := transferCli.Transfer.TransferByAdmin(&transfer.TransferByAdminParams{Body: transferRequest, Context: ctx}, authOpts)
			sCtx.Require().NoError(err)
			sCtx.Require().Equal(models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, res.Payload.Status)

			sres, err := checkingStatusWithRetryAndGetResponce(t, hlfProxy, transferID, models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED)
			sCtx.Require().NoError(err)
			sCtx.Require().Equal(models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED, sres.Payload.Status)

			utils.CheckBalanceEqual(t, *hlfProxy, user.UserAddressBase58Check, channelFrom, balanceAfterTransfer)
			tr.CheckAllowedBalanceEqual(t, hlfProxy, user.UserAddressBase58Check, channelTo, channelFiatUppercase, allowedBalanceAfterTransfer)
		})
	})
}

// TestChannelTransferByCustomerHTTP - add user emit fiat tokens and transfer to channel cc with channelTransferByCustomer by HTTP then check balance and allowedBalance
func TestChannelTransferByCustomerHTTP(t *testing.T) {
	runner.Run(t, "TestChannelTransferByCustomer", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("channel transfer by")
		t.Tags("chtransfer", "positive")
		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))

		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)

		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, "1000")

		t.WithNewStep("Channel transfer by customer http", func(sCtx provider.StepCtx) {
			transferID := uuid.NewString()
			requestID := uuid.NewString()
			channelTransferArgs := []string{transferID, channelCCUppercase, channelFiatUppercase, transferAmount}

			sa, err := utils.SignExpand(user.UserEd25519PrivateKey, user.UserEd25519PublicKey, channelFrom, channelFrom, channelTransferByCustomerMethod, channelTransferArgs, utils.GetNonce(), requestID)
			sCtx.Require().NoError(err)

			transferRequest := &models.ChannelTransferTransferBeginCustomerRequest{
				Generals: &models.ChannelTransferGeneralParams{
					MethodName: channelTransferByCustomerMethod,
					RequestID:  requestID,
					Chaincode:  channelFrom,
					Channel:    channelFrom,
					Nonce:      sa[len(sa)-3],
					PublicKey:  sa[len(sa)-2],
					Sign:       sa[len(sa)-1],
				},
				IDTransfer: transferID,
				ChannelTo:  channelCCUppercase,
				Token:      channelFiatUppercase,
				Amount:     transferAmount,
			}

			res, err := transferCli.Transfer.TransferByCustomer(&transfer.TransferByCustomerParams{Body: transferRequest, Context: context.Background()}, authOpts)
			sCtx.Require().NoError(err)
			sCtx.Require().Equal(models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, res.Payload.Status)

			sres, err := checkingStatusWithRetryAndGetResponce(t, hlfProxy, transferID, models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED)
			sCtx.Require().NoError(err)
			sCtx.Require().Equal(models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED, sres.Payload.Status)

			utils.CheckBalanceEqual(t, *hlfProxy, user.UserAddressBase58Check, channelFrom, balanceAfterTransfer)
			tr.CheckAllowedBalanceEqual(t, hlfProxy, user.UserAddressBase58Check, channelTo, channelFiatUppercase, allowedBalanceAfterTransfer)
		})
	})
}

// TestTransferStatusWrongTransferIdHTTP - get transfer status wrong id by HTTP
func TestTransferStatusWrongTransferIdHTTP(t *testing.T) {
	runner.Run(t, "TestTransferStatusWrongTransferIdHTTP", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("Get transfer status wrong id http")
		t.Tags("negative", "channel-transfer")

		t.WithNewStep("Get transfer status wrong id http", func(sCtx provider.StepCtx) {
			_, err := transferCli.Transfer.TransferStatus(&transfer.TransferStatusParams{IDTransfer: "1", Context: context.Background()}, authOpts)
			sCtx.Require().Contains(err.Error(), "object not found")
		})
	})
}

// TestCustomerTransferWrongChannel - adding user, adding 1000 fiat tokens and execute channelTransferByCustomer foundation method
// with wrong channel then checking that response contains an error
func TestCustomerTransferWrongChannel(t *testing.T) {
	runner.Run(t, "TestCustomerTransferWrongChannel", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestCustomerTransferWrongChannel")
		t.Tags("chtransfer", "negative")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))
		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)

		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, emitAmount)

		t.WithNewStep("TestCustomerTransferWrongChannel", func(sCtx provider.StepCtx) {
			transferID := uuid.NewString()
			requestID := uuid.NewString()
			channelTransferArgs := []string{transferID, "ASD", "FIAT", "250"}

			sa, err := utils.SignExpand(user.UserEd25519PrivateKey, user.UserEd25519PublicKey, channelFrom, channelFrom, channelTransferByCustomerMethod, channelTransferArgs, utils.GetNonce(), requestID)
			sCtx.Require().NoError(err)

			transferRequest := &models.ChannelTransferTransferBeginCustomerRequest{
				Generals: &models.ChannelTransferGeneralParams{
					MethodName: channelTransferByCustomerMethod,
					RequestID:  requestID,
					Chaincode:  channelFrom,
					Channel:    channelFrom,
					Nonce:      sa[len(sa)-3],
					PublicKey:  sa[len(sa)-2],
					Sign:       sa[len(sa)-1],
				},
				IDTransfer: transferID,
				ChannelTo:  "ASD",
				Token:      channelFiatUppercase,
				Amount:     transferAmount,
			}

			res, err := transferCli.Transfer.TransferByCustomer(&transfer.TransferByCustomerParams{Body: transferRequest, Context: context.Background()}, authOpts)
			sCtx.Require().NoError(err)
			sCtx.Require().Equal(models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, res.Payload.Status)

			sres, err := checkingStatusWithRetryAndGetResponce(t, hlfProxy, transferID, models.ChannelTransferTransferStatusResponseStatusSTATUSERROR)
			sCtx.Require().NoError(err)
			sCtx.Require().Contains(sres.Payload.Message, channelErr)

			utils.CheckBalanceEqualWithRetry(t, *hlfProxy, user.UserAddressBase58Check, channelFrom, emitAmount, (time.Second * 1), 20)
			tr.CheckAllowedBalanceEqual(t, hlfProxy, user.UserAddressBase58Check, channelTo, channelFiatUppercase, "0")
		})
	})
}

// TestAdminTransferWrongChannel - adding user, adding 1000 fiat tokens and execute channelTransferByAdmin foundation method
// with wrong channel then checking that response contains an error
func TestAdminTransferWrongChannel(t *testing.T) {
	runner.Run(t, "TestAdminTransferWrongChannel", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestAdminTransferWrongChannel")
		t.Tags("chtransfer", "negative")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))
		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)

		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, emitAmount)

		t.WithNewStep("TestAdminTransferWrongChannel", func(sCtx provider.StepCtx) {
			transferID := uuid.NewString()
			requestID := uuid.NewString()
			channelTransferArgs := []string{transferID, "ASD", user.UserAddressBase58Check, channelFiatUppercase, transferAmount}

			sa, err := utils.SignExpand(issuer.IssuerEd25519PrivateKey, issuer.IssuerEd25519PublicKey, channelFrom, channelFrom, channelTransferByAdminMethod, channelTransferArgs, utils.GetNonce(), requestID)
			sCtx.Require().NoError(err)

			transferRequest := &models.ChannelTransferTransferBeginAdminRequest{
				Generals: &models.ChannelTransferGeneralParams{
					MethodName: channelTransferByAdminMethod,
					RequestID:  requestID,
					Chaincode:  channelFrom,
					Channel:    channelFrom,
					Nonce:      sa[len(sa)-3],
					PublicKey:  sa[len(sa)-2],
					Sign:       sa[len(sa)-1],
				},
				IDTransfer: transferID,
				ChannelTo:  "ASD",
				Address:    user.UserAddressBase58Check,
				Token:      channelFiatUppercase,
				Amount:     transferAmount,
			}

			res, err := transferCli.Transfer.TransferByAdmin(&transfer.TransferByAdminParams{Body: transferRequest, Context: ctx}, authOpts)
			sCtx.Require().NoError(err)
			sCtx.Require().Equal(models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, res.Payload.Status)

			sres, err := checkingStatusWithRetryAndGetResponce(t, hlfProxy, transferID, models.ChannelTransferTransferStatusResponseStatusSTATUSERROR)
			sCtx.Require().NoError(err)
			sCtx.Require().Contains(*&sres.Payload.Message, channelErr)

			utils.CheckBalanceEqualWithRetry(t, *hlfProxy, user.UserAddressBase58Check, channelFrom, emitAmount, (time.Second * 1), 20)
			tr.CheckAllowedBalanceEqual(t, hlfProxy, user.UserAddressBase58Check, channelTo, channelFiatUppercase, "0")
		})
	})
}

// TestCustomerNotValidChannel - adding user, adding 1000 fiat tokens and execute channelTransferByCustomer foundation method
// with not valid channel then checking that response contains an error
func TestCustomerNotValidChannel(t *testing.T) {
	runner.Run(t, "TestCustomerNotValidChannel", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestCustomerNotValidChannel")
		t.Tags("chtransfer", "negative")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))
		user := utils.AddUser(t, *hlfProxy)

		t.WithNewStep("TestCustomerNotValidChannel", func(sCtx provider.StepCtx) {
			transferID := uuid.NewString()
			requestID := uuid.NewString()
			channelTransferArgs := []string{transferID, "ACL", "ACL", "250"}

			sa, err := utils.SignExpand(user.UserEd25519PrivateKey, user.UserEd25519PublicKey, "acl", "acl", channelTransferByCustomerMethod, channelTransferArgs, utils.GetNonce(), requestID)
			sCtx.Require().NoError(err)

			transferRequest := &models.ChannelTransferTransferBeginCustomerRequest{
				Generals: &models.ChannelTransferGeneralParams{
					MethodName: channelTransferByCustomerMethod,
					RequestID:  requestID,
					Chaincode:  "acl",
					Channel:    "acl",
					Nonce:      sa[len(sa)-3],
					PublicKey:  sa[len(sa)-2],
					Sign:       sa[len(sa)-1],
				},
				IDTransfer: transferID,
				ChannelTo:  "FIAT",
				Token:      "ACL",
				Amount:     transferAmount,
			}

			_, err = transferCli.Transfer.TransferByCustomer(&transfer.TransferByCustomerParams{Body: transferRequest, Context: context.Background()}, authOpts)
			sCtx.Require().Contains(err.Error(), notValidChannelErr)
		})
	})
}

// TestAdminNotValidChannel - adding user, adding 1000 fiat tokens and execute channelTransferByAdmin foundation method
// with not valid channel then checking that response contains an error
func TestAdminNotValidChannel(t *testing.T) {
	runner.Run(t, "TestAdminNotValidChannel", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestAdminNotValidChannel")
		t.Tags("chtransfer", "negative")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))
		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)

		t.WithNewStep("TestAdminNotValidChannel", func(sCtx provider.StepCtx) {
			transferID := uuid.NewString()
			requestID := uuid.NewString()
			channelTransferArgs := []string{transferID, "ACL", user.UserAddressBase58Check, channelFiatUppercase, transferAmount}

			sa, err := utils.SignExpand(issuer.IssuerEd25519PrivateKey, issuer.IssuerEd25519PublicKey, "acl", "acl", channelTransferByAdminMethod, channelTransferArgs, utils.GetNonce(), requestID)
			sCtx.Require().NoError(err)

			transferRequest := &models.ChannelTransferTransferBeginAdminRequest{
				Generals: &models.ChannelTransferGeneralParams{
					MethodName: channelTransferByCustomerMethod,
					RequestID:  requestID,
					Chaincode:  "acl",
					Channel:    "acl",
					Nonce:      sa[len(sa)-3],
					PublicKey:  sa[len(sa)-2],
					Sign:       sa[len(sa)-1],
				},
				IDTransfer: transferID,
				ChannelTo:  "FIAT",
				Address:    user.UserAddressBase58Check,
				Token:      "ACL",
				Amount:     transferAmount,
			}

			_, err = transferCli.Transfer.TransferByAdmin(&transfer.TransferByAdminParams{Body: transferRequest, Context: ctx}, authOpts)
			sCtx.Require().Contains(err.Error(), notValidChannelErr)
		})
	})
}

// TestCustomerTransferInsufficientFunds - adding user and execute channelTransferByCustomer foundation method then checking that response contains an error
func TestCustomerTransferInsufficientFunds(t *testing.T) {
	runner.Run(t, "TestCustomerTransferInsufficientFunds", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestCustomerTransferInsufficientFunds")
		t.Tags("chtransfer", "negative")
		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))

		user := utils.AddUser(t, *hlfProxy)

		t.WithNewStep("TestCustomerTransferInsufficientFunds", func(sCtx provider.StepCtx) {
			transferID := uuid.NewString()
			requestID := uuid.NewString()
			channelTransferArgs := []string{transferID, channelCCUppercase, channelFiatUppercase, transferAmount}

			sa, err := utils.SignExpand(user.UserEd25519PrivateKey, user.UserEd25519PublicKey, channelFrom, channelFrom, channelTransferByCustomerMethod, channelTransferArgs, utils.GetNonce(), requestID)
			sCtx.Require().NoError(err)

			transferRequest := &models.ChannelTransferTransferBeginCustomerRequest{
				Generals: &models.ChannelTransferGeneralParams{
					MethodName: channelTransferByCustomerMethod,
					RequestID:  requestID,
					Chaincode:  channelFrom,
					Channel:    channelFrom,
					Nonce:      sa[len(sa)-3],
					PublicKey:  sa[len(sa)-2],
					Sign:       sa[len(sa)-1],
				},
				IDTransfer: transferID,
				ChannelTo:  channelCCUppercase,
				Token:      channelFiatUppercase,
				Amount:     transferAmount,
			}

			res, err := transferCli.Transfer.TransferByCustomer(&transfer.TransferByCustomerParams{Body: transferRequest, Context: ctx}, authOpts)
			sCtx.Require().NoError(err)
			sCtx.Require().Equal(models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, res.Payload.Status)

			sres, err := checkingStatusWithRetryAndGetResponce(t, hlfProxy, transferID, models.ChannelTransferTransferStatusResponseStatusSTATUSERROR)
			sCtx.Require().NoError(err)
			sCtx.Require().Equal(*&sres.Payload.Message, insufficientFundsErr)

			utils.CheckBalanceEqual(t, *hlfProxy, user.UserAddressBase58Check, channelFrom, "0")
			tr.CheckAllowedBalanceEqual(t, hlfProxy, user.UserAddressBase58Check, channelTo, "FIAT", "0")
		})
	})
}

// TestAdminTransferInsufficientFunds - adding user and execute channelTransferByAdmin foundation method then checking that response contains an error
func TestAdminTransferInsufficientFunds(t *testing.T) {
	runner.Run(t, "TestAdminTransferInsufficientFunds", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestAdminTransferInsufficientFunds")
		t.Tags("chtransfer", "negative")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))

		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)

		t.WithNewStep("TestAdminTransferInsufficientFunds", func(sCtx provider.StepCtx) {
			transferID := uuid.NewString()
			requestID := uuid.NewString()
			channelTransferArgs := []string{transferID, channelCCUppercase, user.UserAddressBase58Check, channelFiatUppercase, transferAmount}

			sa, err := utils.SignExpand(issuer.IssuerEd25519PrivateKey, issuer.IssuerEd25519PublicKey, channelFrom, channelFrom, channelTransferByAdminMethod, channelTransferArgs, utils.GetNonce(), requestID)
			sCtx.Require().NoError(err)

			transferRequest := &models.ChannelTransferTransferBeginAdminRequest{
				Generals: &models.ChannelTransferGeneralParams{
					MethodName: channelTransferByAdminMethod,
					RequestID:  requestID,
					Chaincode:  channelFrom,
					Channel:    channelFrom,
					Nonce:      sa[len(sa)-3],
					PublicKey:  sa[len(sa)-2],
					Sign:       sa[len(sa)-1],
				},
				IDTransfer: transferID,
				ChannelTo:  channelCCUppercase,
				Address:    user.UserAddressBase58Check,
				Token:      channelFiatUppercase,
				Amount:     transferAmount,
			}

			res, err := transferCli.Transfer.TransferByAdmin(&transfer.TransferByAdminParams{Body: transferRequest, Context: ctx}, authOpts)
			sCtx.Require().NoError(err)
			sCtx.Require().Equal(models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, res.Payload.Status)

			sres, err := checkingStatusWithRetryAndGetResponce(t, hlfProxy, transferID, models.ChannelTransferTransferStatusResponseStatusSTATUSERROR)
			sCtx.Require().NoError(err)
			sCtx.Require().Equal(*&sres.Payload.Message, insufficientFundsErr)

			utils.CheckBalanceEqual(t, *hlfProxy, user.UserAddressBase58Check, channelFrom, "0")
			tr.CheckAllowedBalanceEqual(t, hlfProxy, user.UserAddressBase58Check, channelTo, "FIAT", "0")
		})
	})
}

// TestAdminTransferWrongFoundationChannelTo - adding user and execute channelTransferByAdmin foundation method to chaincode with low version of service on channel To
// then checking that response contains an error
func TestAdminTransferWrongFoundationChannelTo(t *testing.T) {
	runner.Run(t, "TestAdminTransferWrongFoundationChannelTo", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestAdminTransferWrongFoundationChannelTo")
		t.Tags("chtransfer", "negative")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))

		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)

		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, emitAmount)

		t.WithNewStep("TestAdminTransferWrongFoundationChannelTo", func(sCtx provider.StepCtx) {
			transferID := uuid.NewString()
			requestID := uuid.NewString()
			channelTransferArgs := []string{transferID, "INDUSTRIAL", user.UserAddressBase58Check, channelFiatUppercase, transferAmount}

			sa, err := utils.SignExpand(issuer.IssuerEd25519PrivateKey, issuer.IssuerEd25519PublicKey, channelFrom, channelFrom, channelTransferByAdminMethod, channelTransferArgs, utils.GetNonce(), requestID)
			sCtx.Require().NoError(err)

			transferRequest := &models.ChannelTransferTransferBeginAdminRequest{
				Generals: &models.ChannelTransferGeneralParams{
					MethodName: channelTransferByAdminMethod,
					RequestID:  requestID,
					Chaincode:  channelFrom,
					Channel:    channelFrom,
					Nonce:      sa[len(sa)-3],
					PublicKey:  sa[len(sa)-2],
					Sign:       sa[len(sa)-1],
				},
				IDTransfer: transferID,
				ChannelTo:  "INDUSTRIAL",
				Address:    user.UserAddressBase58Check,
				Token:      channelFiatUppercase,
				Amount:     transferAmount,
			}

			res, err := transferCli.Transfer.TransferByAdmin(&transfer.TransferByAdminParams{Body: transferRequest, Context: ctx}, authOpts)
			sCtx.Require().NoError(err)
			sCtx.Require().Equal(models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, res.Payload.Status)

			sres, err := checkingStatusWithRetryAndGetResponce(t, hlfProxy, transferID, models.ChannelTransferTransferStatusResponseStatusSTATUSCANCELED)
			sCtx.Require().NoError(err)
			sCtx.Require().Contains(*&sres.Payload.Message, unknownMethodErr)

			utils.CheckBalanceEqualWithRetry(t, *hlfProxy, user.UserAddressBase58Check, channelFrom, emitAmount, (time.Second * 1), 20)
			tr.CheckAllowedBalanceEqual(t, hlfProxy, user.UserAddressBase58Check, channelTo, channelFiatUppercase, "0")
		})
	})
}

// TestCustomerTransferWrongFoundationChannelTo - adding user and execute channelTransferByCustomer foundation method to chaincode with low version of service on channel To
// then checking that response contains an error
func TestCustomerTransferWrongFoundationChannelTo(t *testing.T) {
	runner.Run(t, "TestCustomerTransferWrongFoundationChannelTo", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestCustomerTransferWrongFoundationChannelTo")
		t.Tags("chtransfer", "negative")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))

		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)

		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, emitAmount)

		t.WithNewStep("TestCustomerTransferWrongFoundationChannelTo", func(sCtx provider.StepCtx) {
			transferID := uuid.NewString()
			requestID := uuid.NewString()
			channelTransferArgs := []string{transferID, "INDUSTRIAL", channelFiatUppercase, transferAmount}

			sa, err := utils.SignExpand(user.UserEd25519PrivateKey, user.UserEd25519PublicKey, channelFrom, channelFrom, channelTransferByCustomerMethod, channelTransferArgs, utils.GetNonce(), requestID)
			sCtx.Require().NoError(err)

			transferRequest := &models.ChannelTransferTransferBeginCustomerRequest{
				Generals: &models.ChannelTransferGeneralParams{
					MethodName: channelTransferByCustomerMethod,
					RequestID:  requestID,
					Chaincode:  channelFrom,
					Channel:    channelFrom,
					Nonce:      sa[len(sa)-3],
					PublicKey:  sa[len(sa)-2],
					Sign:       sa[len(sa)-1],
				},
				IDTransfer: transferID,
				ChannelTo:  "INDUSTRIAL",
				Token:      channelFiatUppercase,
				Amount:     transferAmount,
			}

			res, err := transferCli.Transfer.TransferByCustomer(&transfer.TransferByCustomerParams{Body: transferRequest, Context: ctx}, authOpts)
			sCtx.Require().NoError(err)
			sCtx.Require().Equal(models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, res.Payload.Status)

			sres, err := checkingStatusWithRetryAndGetResponce(t, hlfProxy, transferID, models.ChannelTransferTransferStatusResponseStatusSTATUSCANCELED)
			sCtx.Require().NoError(err)
			sCtx.Require().Contains(*&sres.Payload.Message, unknownMethodErr)

			utils.CheckBalanceEqualWithRetry(t, *hlfProxy, user.UserAddressBase58Check, channelFrom, emitAmount, (time.Second * 1), 20)
			tr.CheckAllowedBalanceEqual(t, hlfProxy, user.UserAddressBase58Check, channelTo, channelFiatUppercase, "0")
		})
	})
}

// TestAdminTransferWrongFoundationChannelFrom - adding user and execute channelTransferByAdmin foundation method to chaincode with low version of service on channel From
// then checking that response contains an error
func TestAdminTransferWrongFoundationChannelFrom(t *testing.T) {
	runner.Run(t, "TestAdminTransferWrongFoundationChannelFrom", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestAdminTransferWrongFoundationChannelFrom")
		t.Tags("positive", "chtransfer")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))

		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)

		t.WithNewStep("TestAdminTransferWrongFoundationChannelFrom", func(sCtx provider.StepCtx) {
			transferID := uuid.NewString()
			requestID := uuid.NewString()
			channelTransferArgs := []string{transferID, channelCCUppercase, user.UserAddressBase58Check, "INDUSTRIAL", transferAmount}

			sa, err := utils.SignExpand(issuer.IssuerEd25519PrivateKey, issuer.IssuerEd25519PublicKey, "industrial", "industrial", channelTransferByAdminMethod, channelTransferArgs, utils.GetNonce(), requestID)
			sCtx.Require().NoError(err)

			transferRequest := &models.ChannelTransferTransferBeginAdminRequest{
				Generals: &models.ChannelTransferGeneralParams{
					MethodName: channelTransferByAdminMethod,
					RequestID:  requestID,
					Chaincode:  "industrial",
					Channel:    "industrial",
					Nonce:      sa[len(sa)-3],
					PublicKey:  sa[len(sa)-2],
					Sign:       sa[len(sa)-1],
				},
				IDTransfer: transferID,
				ChannelTo:  channelCCUppercase,
				Address:    user.UserAddressBase58Check,
				Token:      "INDUSTRIAL",
				Amount:     transferAmount,
			}

			res, err := transferCli.Transfer.TransferByAdmin(&transfer.TransferByAdminParams{Body: transferRequest, Context: ctx}, authOpts)
			sCtx.Require().NoError(err)
			sCtx.Require().Equal(models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, res.Payload.Status)

			sres, err := checkingStatusWithRetryAndGetResponce(t, hlfProxy, transferID, models.ChannelTransferTransferStatusResponseStatusSTATUSERROR)
			sCtx.Require().NoError(err)
			sCtx.Require().Contains(*&sres.Payload.Message, unknownMethodErr)

			utils.CheckBalanceEqual(t, *hlfProxy, user.UserAddressBase58Check, channelFrom, "0")
			tr.CheckAllowedBalanceEqual(t, hlfProxy, user.UserAddressBase58Check, channelTo, channelFiatUppercase, "0")
		})
	})
}

// TestChannelTransferByCustomerChannelFrom - adding user and execute channelTransferByCustomer foundation method to chaincode with low version of service on channel From
// then checking that response contains an error
func TestCustomerTransferWrongFoundationChannelFrom(t *testing.T) {
	runner.Run(t, "TestChannelTransferByCustomerChannelFrom", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestChannelTransferByCustomerChannelFrom")
		t.Tags("chtransfer", "positive")
		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))

		user := utils.AddUser(t, *hlfProxy)

		t.WithNewStep("TestChannelTransferByCustomerChannelFrom", func(sCtx provider.StepCtx) {
			transferID := uuid.NewString()
			requestID := uuid.NewString()
			channelTransferArgs := []string{transferID, channelCCUppercase, "INDUSTRIAL", transferAmount}

			sa, err := utils.SignExpand(user.UserEd25519PrivateKey, user.UserEd25519PublicKey, "industrial", "industrial", channelTransferByCustomerMethod, channelTransferArgs, utils.GetNonce(), requestID)
			sCtx.Require().NoError(err)

			transferRequest := &models.ChannelTransferTransferBeginCustomerRequest{
				Generals: &models.ChannelTransferGeneralParams{
					MethodName: channelTransferByCustomerMethod,
					RequestID:  requestID,
					Chaincode:  "industrial",
					Channel:    "industrial",
					Nonce:      sa[len(sa)-3],
					PublicKey:  sa[len(sa)-2],
					Sign:       sa[len(sa)-1],
				},
				IDTransfer: transferID,
				ChannelTo:  channelCCUppercase,
				Token:      "INDUSTRIAL",
				Amount:     transferAmount,
			}

			res, err := transferCli.Transfer.TransferByCustomer(&transfer.TransferByCustomerParams{Body: transferRequest, Context: context.Background()}, authOpts)
			sCtx.Require().NoError(err)
			sCtx.Require().Equal(models.ChannelTransferTransferStatusResponseStatusSTATUSINPROCESS, res.Payload.Status)

			sres, err := checkingStatusWithRetryAndGetResponce(t, hlfProxy, transferID, models.ChannelTransferTransferStatusResponseStatusSTATUSERROR)
			sCtx.Require().NoError(err)
			sCtx.Require().Equal(models.ChannelTransferTransferStatusResponseStatusSTATUSERROR, sres.Payload.Status)
			sCtx.Require().Contains(sres.Payload.Message, unknownMethodErr)

			utils.CheckBalanceEqual(t, *hlfProxy, user.UserAddressBase58Check, channelFrom, "0")
			tr.CheckAllowedBalanceEqual(t, hlfProxy, user.UserAddressBase58Check, channelTo, channelFiatUppercase, "0")
		})
	})
}
