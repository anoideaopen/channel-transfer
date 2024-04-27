package test

/*
import (
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/anoideaopen/channel-transfer/models"
	"github.com/anoideaopen/channel-transfer/test/integration/clienthttp/transfer"
	utils "github.com/anoideaopen/testnet-util"
	tr "github.com/anoideaopen/testnet-util/transfer"
	"github.com/google/uuid"
	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/runner"
)

const (
	emitAmount                  = "1000"
	channelTo                   = "cc"
	balanceAfterTransfer        = "750"
	transferAmount              = "250"
	allowedBalanceAfterTransfer = "250"
	transferExecutionTimeout    = 100
	ttlExpireTimeout            = 11
)

// TestCustomerTransferBeforeTtl - adding user, adding 1000 fiat tokens then stoping service and execute channelTransferByCustomer foundation method
// then starting service and checking that transfer status is completed
func TestCustomerTransferBeforeTtl(t *testing.T) {
	runner.Run(t, "TestCustomerTransferBeforeTtl", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestCustomerTransferBeforeTtl")
		t.Tags("chtransfer", "positive")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))
		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)
		transferID := uuid.NewString()

		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, emitAmount)

		t.WithNewStep("Channel transfer by customer before ttl", func(sCtx provider.StepCtx) {
			err := sigstopService()
			sCtx.Require().NoError(err)

			tr.ChannelTransferByCustomer(t, hlfProxy, channelFrom, user, getChannelTransferByCustomerArgs(transferID))

			err = sigcontService()
			sCtx.Require().NoError(err)
			successCommonChecks(t, hlfProxy, user.UserAddressBase58Check, transferID)
		})
	})
}

// TestAdminTransferBeforeTtl - adding user, adding 1000 fiat tokens then stoping service and execute channelTransferByAdmin foundation method
// then starting service and checking that transfer status is completed
func TestAdminTransferBeforeTtl(t *testing.T) {
	runner.Run(t, "TestAdminTransferBeforeTtl", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("channel transfer by admin before ttl")
		t.Tags("chtransfer", "positive")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))
		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)
		transferID := uuid.NewString()

		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, emitAmount)

		t.WithNewStep("Channel transfer by admin before ttl", func(sCtx provider.StepCtx) {
			err := sigstopService()
			sCtx.Require().NoError(err)

			tr.ChannelTransferByAdmin(t, hlfProxy, issuer, channelFrom, getChannelTransferByAdminArgs(transferID, user.UserAddressBase58Check))

			err = sigcontService()
			sCtx.Require().NoError(err)
			successCommonChecks(t, hlfProxy, user.UserAddressBase58Check, transferID)
		})
	})
}

// TransferByCustomerCancelByTimeout - adding user, adding 1000 fiat tokens then stoping service and execute channelTransferByCustomer foundation method
// waiting for ttl expired then starting service and checking that transfer status is canceled
func TestTransferByCustomerCancelByTimeout(t *testing.T) {
	runner.Run(t, "TestTransferByCustomerCancelByTimeout", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestTransferByCustomerCancelByTimeout")
		t.Tags("chtransfer", "positive")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))
		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)
		transferID := uuid.NewString()

		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, emitAmount)

		t.WithNewStep("TestTransferByCustomerCancelByTimeout", func(sCtx provider.StepCtx) {
			err := sigstopService()
			sCtx.Require().NoError(err)

			tr.ChannelTransferByCustomer(t, hlfProxy, channelFrom, user, getChannelTransferByCustomerArgs(transferID))

			waitForTtlExpire()

			err = sigcontService()
			sCtx.Require().NoError(err)
			time.Sleep(time.Second * 6)
			cancelCommonChecks(t, hlfProxy, user.UserAddressBase58Check, transferID)
		})
	})
}

// TestTransferByAdminCancelByTimeout - adding user, adding 1000 fiat tokens then stoping service and execute channelTransferByAdmin foundation method
// waiting for ttl expired then starting service and checking that transfer status is canceled
func TestTransferByAdminCancelByTimeout(t *testing.T) {
	runner.Run(t, "TestTransferByAdminCancelByTimeout", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestTransferByAdminCancelByTimeout")
		t.Tags("chtransfer", "positive")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))
		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)
		transferID := uuid.NewString()

		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, emitAmount)

		t.WithNewStep("TestTransferByAdminCancelByTimeout", func(sCtx provider.StepCtx) {
			err := sigstopService()
			sCtx.Require().NoError(err)

			tr.ChannelTransferByAdmin(t, hlfProxy, issuer, channelFrom, getChannelTransferByAdminArgs(transferID, user.UserAddressBase58Check))

			waitForTtlExpire()

			err = sigcontService()
			sCtx.Require().NoError(err)
			time.Sleep(time.Second * 6)
			cancelCommonChecks(t, hlfProxy, user.UserAddressBase58Check, transferID)
		})
	})
}

// TestTransferToBeforeTtl - adding user, adding 1000 fiat tokens then stoping service and execute channelTransferByAdmin and CreateCCTransferTo foundation methods
// then starting service and checking that transfer status is completed
func TestTransferToBeforeTtl(t *testing.T) {
	runner.Run(t, "TestTransferToBeforeTtl", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestTransferToBeforeTtl")
		t.Tags("chtransfer", "positive")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))
		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)
		transferID := uuid.NewString()

		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, emitAmount)

		t.WithNewStep("TestTransferToBeforeTtl", func(sCtx provider.StepCtx) {
			err := sigstopService()
			sCtx.Require().NoError(err)

			tr.ChannelTransferByAdmin(t, hlfProxy, issuer, channelFrom, getChannelTransferByAdminArgs(transferID, user.UserAddressBase58Check))
			from := tr.ChannelTransferFrom(t, hlfProxy, channelFrom, transferID)
			tr.CreateCCTransferTo(t, hlfProxy, channelTo, from)

			err = sigcontService()
			sCtx.Require().NoError(err)

			successCommonChecks(t, hlfProxy, user.UserAddressBase58Check, transferID)
		})
	})
}

// TestTransferToAfterTtl - adding user, adding 1000 fiat tokens then stoping service and execute channelTransferByAdmin and CreateCCTransferTo foundation methods
// waiting for ttl expired then starting service and checking that transfer status is completed
func TestTransferToAfterTtl(t *testing.T) {
	runner.Run(t, "TestTransferToAfterTtl", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestTransferToAfterTtl")
		t.Tags("chtransfer", "positive")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))
		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)
		transferID := uuid.NewString()

		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, emitAmount)

		t.WithNewStep("TestTransferToAfterTtl", func(sCtx provider.StepCtx) {
			err := sigstopService()
			sCtx.Require().NoError(err)

			tr.ChannelTransferByAdmin(t, hlfProxy, issuer, channelFrom, getChannelTransferByAdminArgs(transferID, user.UserAddressBase58Check))
			from := tr.ChannelTransferFrom(t, hlfProxy, channelFrom, transferID)
			tr.CreateCCTransferTo(t, hlfProxy, channelTo, from)

			waitForTtlExpire()

			err = sigcontService()
			sCtx.Require().NoError(err)
			successCommonChecks(t, hlfProxy, user.UserAddressBase58Check, transferID)
		})
	})
}

// TestTransferFromBeforeTtl - adding user, adding 1000 fiat tokens then stoping service and execute channelTransferByAdmin, CreateCCTransferTo and
// CommitCCTransferFrom foundation methods then starting service and checking that transfer status is completed
func TestTransferFromBeforeTtl(t *testing.T) {
	runner.Run(t, "TestTransferFromBeforeTtl", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestTransferFromBeforeTtl")
		t.Tags("chtransfer", "positive")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))
		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)
		transferID := uuid.NewString()

		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, emitAmount)

		t.WithNewStep("TestTransferFromBeforeTtl", func(sCtx provider.StepCtx) {
			err := sigstopService()
			sCtx.Require().NoError(err)

			tr.ChannelTransferByAdmin(t, hlfProxy, issuer, channelFrom, getChannelTransferByAdminArgs(transferID, user.UserAddressBase58Check))
			from := tr.ChannelTransferFrom(t, hlfProxy, channelFrom, transferID)
			tr.CreateCCTransferTo(t, hlfProxy, channelTo, from)
			tr.ChannelTransferTo(t, hlfProxy, channelTo, transferID)
			tr.CommitCCTransferFrom(t, hlfProxy, channelFrom, transferID)

			err = sigcontService()
			sCtx.Require().NoError(err)
			successCommonChecks(t, hlfProxy, user.UserAddressBase58Check, transferID)
		})
	})
}

// // TestTransferFromAfterTtl - adding user, adding 1000 fiat tokens then stoping service and execute channelTransferByAdmin, CreateCCTransferTo and
// // CommitCCTransferFrom foundation methods waiting for ttl expired then starting service and checking that transfer status is completed
func TestTransferFromAfterTtl(t *testing.T) {
	runner.Run(t, "TestTransferFromAfterTtl", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestTransferFromAfterTtl")
		t.Tags("chtransfer", "positive")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))
		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)
		transferID := uuid.NewString()

		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, emitAmount)

		t.WithNewStep("TestTransferFromAfterTtl", func(sCtx provider.StepCtx) {
			sigstopService()

			tr.ChannelTransferByAdmin(t, hlfProxy, issuer, channelFrom, getChannelTransferByAdminArgs(transferID, user.UserAddressBase58Check))
			from := tr.ChannelTransferFrom(t, hlfProxy, channelFrom, transferID)
			tr.CreateCCTransferTo(t, hlfProxy, channelTo, from)
			tr.ChannelTransferTo(t, hlfProxy, channelTo, transferID)
			tr.CommitCCTransferFrom(t, hlfProxy, channelFrom, transferID)

			waitForTtlExpire()

			sigcontService()
			successCommonChecks(t, hlfProxy, user.UserAddressBase58Check, transferID)
		})
	})
}

// TestDeleteTransferToBeforeTtl - adding user, adding 1000 fiat tokens then stoping service and execute channelTransferByAdmin, CreateCCTransferTo,
// CommitCCTransferFrom and DeleteCCTransferTo foundation methods then starting service and checking that transfer status is completed
func TestDeleteTransferToBeforeTtl(t *testing.T) {
	runner.Run(t, "TestTransferFromBeforeTtl", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestDeleteTransferToBeforeTtl")
		t.Tags("chtransfer", "positive")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))
		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)
		transferID := uuid.NewString()

		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, "1000")

		t.WithNewStep("TestDeleteTransferToBeforeTtl", func(sCtx provider.StepCtx) {
			err := sigstopService()
			sCtx.Require().NoError(err)

			tr.ChannelTransferByAdmin(t, hlfProxy, issuer, channelFrom, getChannelTransferByAdminArgs(transferID, user.UserAddressBase58Check))
			from := tr.ChannelTransferFrom(t, hlfProxy, channelFrom, transferID)
			tr.CreateCCTransferTo(t, hlfProxy, channelTo, from)
			tr.ChannelTransferTo(t, hlfProxy, channelTo, transferID)
			tr.CommitCCTransferFrom(t, hlfProxy, channelFrom, transferID)
			tr.DeleteCCTransferTo(t, hlfProxy, channelTo, transferID)

			err = sigcontService()
			sCtx.Require().NoError(err)
			successCommonChecks(t, hlfProxy, user.UserAddressBase58Check, transferID)
		})
	})
}

// TestDeleteTransferToAfterTtl - adding user, adding 1000 fiat tokens then stoping service and execute channelTransferByAdmin, CreateCCTransferTo,
// CommitCCTransferFrom and DeleteCCTransferTo foundation methods waiting for ttl expired then starting service and checking that transfer status is completed
func TestDeleteTransferToAfterTtl(t *testing.T) {
	runner.Run(t, "TestDeleteTransferToAfterTtl", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestDeleteTransferToAfterTtl")
		t.Tags("chtransfer", "positive")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))
		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)
		transferID := uuid.NewString()

		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, "1000")

		t.WithNewStep("TestDeleteTransferToAfterTtl", func(sCtx provider.StepCtx) {
			err := sigstopService()
			sCtx.Require().NoError(err)

			tr.ChannelTransferByAdmin(t, hlfProxy, issuer, channelFrom, getChannelTransferByAdminArgs(transferID, user.UserAddressBase58Check))
			from := tr.ChannelTransferFrom(t, hlfProxy, channelFrom, transferID)
			tr.CreateCCTransferTo(t, hlfProxy, channelTo, from)
			tr.ChannelTransferTo(t, hlfProxy, channelTo, transferID)
			tr.CommitCCTransferFrom(t, hlfProxy, channelFrom, transferID)
			tr.DeleteCCTransferTo(t, hlfProxy, channelTo, transferID)

			waitForTtlExpire()

			err = sigcontService()
			sCtx.Require().NoError(err)
			successCommonChecks(t, hlfProxy, user.UserAddressBase58Check, transferID)
		})
	})
}

// TestDeleteTransferFromBeforeTtl - adding user, adding 1000 fiat tokens then stoping service and execute channelTransferByAdmin, CreateCCTransferTo,
// CommitCCTransferFrom, DeleteCCTransferTo and DeleteCCTransferFrom foundation methods then starting service and checking that transfer status is completed
func TestDeleteTransferFromBeforeTtl(t *testing.T) {
	runner.Run(t, "TestDeleteTransferFromBeforeTtl", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestDeleteTransferFromBeforeTtl")
		t.Tags("chtransfer", "positive")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))
		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)
		transferID := uuid.NewString()

		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, emitAmount)

		t.WithNewStep("TestDeleteTransferFromBeforeTtl", func(sCtx provider.StepCtx) {
			err := sigstopService()
			sCtx.Require().NoError(err)

			tr.ChannelTransferByAdmin(t, hlfProxy, issuer, channelFrom, getChannelTransferByAdminArgs(transferID, user.UserAddressBase58Check))
			from := tr.ChannelTransferFrom(t, hlfProxy, channelFrom, transferID)
			tr.CreateCCTransferTo(t, hlfProxy, channelTo, from)
			tr.ChannelTransferTo(t, hlfProxy, channelTo, transferID)
			tr.CommitCCTransferFrom(t, hlfProxy, channelFrom, transferID)
			tr.DeleteCCTransferTo(t, hlfProxy, channelTo, transferID)
			tr.DeleteCCTransferFrom(t, hlfProxy, channelFrom, transferID)

			err = sigcontService()
			sCtx.Require().NoError(err)
			utils.CheckBalanceEqual(t, *hlfProxy, user.UserAddressBase58Check, channelFrom, balanceAfterTransfer)
			tr.CheckAllowedBalanceEqual(t, hlfProxy, user.UserAddressBase58Check, channelTo, "FIAT", allowedBalanceAfterTransfer)
		})
	})
}

// TestDeleteTransferFromAfterTtl - adding user, adding 1000 fiat tokens then stoping service and execute channelTransferByAdmin, CreateCCTransferTo,
// CommitCCTransferFrom, DeleteCCTransferTo and DeleteCCTransferFrom foundation methods waiting for ttl expired then starting service and checking that transfer status is completed
func TestDeleteTransferFromAfterTtl(t *testing.T) {
	runner.Run(t, "TestDeleteTransferFromAfterTtl", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestDeleteTransferFromAfterTtl")
		t.Tags("chtransfer", "positive")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))
		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)
		transferID := uuid.NewString()

		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, emitAmount)

		t.WithNewStep("TestDeleteTransferFromBeforeTtl", func(sCtx provider.StepCtx) {
			err := sigstopService()
			sCtx.Require().NoError(err)

			tr.ChannelTransferByAdmin(t, hlfProxy, issuer, channelFrom, getChannelTransferByAdminArgs(transferID, user.UserAddressBase58Check))
			from := tr.ChannelTransferFrom(t, hlfProxy, channelFrom, transferID)
			tr.CreateCCTransferTo(t, hlfProxy, channelTo, from)
			tr.ChannelTransferTo(t, hlfProxy, channelTo, transferID)
			tr.CommitCCTransferFrom(t, hlfProxy, channelFrom, transferID)
			tr.DeleteCCTransferTo(t, hlfProxy, channelTo, transferID)
			tr.DeleteCCTransferFrom(t, hlfProxy, channelFrom, transferID)

			waitForTtlExpire()

			err = sigcontService()
			sCtx.Require().NoError(err)
			utils.CheckBalanceEqual(t, *hlfProxy, user.UserAddressBase58Check, channelFrom, balanceAfterTransfer)
			tr.CheckAllowedBalanceEqual(t, hlfProxy, user.UserAddressBase58Check, channelTo, "FIAT", allowedBalanceAfterTransfer)
		})
	})
}

func successCommonChecks(t provider.T, hlfProxy *utils.HlfProxyService, userAddressBase58Check string, transferID string) {
	t.WithNewStep("Checking that request is COMPLETED", func(sCtx provider.StepCtx) {
		sres, err := checkingStatusWithRetryAndGetResponce(t, hlfProxy, transferID, models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED)
		sCtx.Require().NoError(err)
		sCtx.Require().Equal(models.ChannelTransferTransferStatusResponseStatusSTATUSCOMPLETED, sres.Payload.Status)
		utils.CheckBalanceEqual(t, *hlfProxy, userAddressBase58Check, channelFrom, balanceAfterTransfer)
		tr.CheckAllowedBalanceEqual(t, hlfProxy, userAddressBase58Check, channelTo, "FIAT", allowedBalanceAfterTransfer)
	})
}

func cancelCommonChecks(t provider.T, hlfProxy *utils.HlfProxyService, userAddressBase58Check string, transferID string) {
	t.WithNewStep("Checking that request is CANCELED", func(sCtx provider.StepCtx) {
		sres, err := checkingStatusWithRetryAndGetResponce(t, hlfProxy, transferID, models.ChannelTransferTransferStatusResponseStatusSTATUSCANCELED)
		sCtx.Require().NoError(err)
		sCtx.Require().Equal(models.ChannelTransferTransferStatusResponseStatusSTATUSCANCELED, sres.Payload.Status)
		utils.CheckBalanceEqual(t, *hlfProxy, userAddressBase58Check, channelFrom, emitAmount)
		tr.CheckAllowedBalanceEqual(t, hlfProxy, userAddressBase58Check, channelTo, "FIAT", "0")
	})
}

func waitForTtlExpire() {
	time.Sleep(time.Second * ttlExpireTimeout)
}

func getChannelTransferByCustomerArgs(transferID string) []string {
	return []string{transferID, "CC", "FIAT", transferAmount}
}

func getChannelTransferByAdminArgs(transferID string, UserAddressBase58Check string) []string {
	return []string{transferID, "CC", UserAddressBase58Check, "FIAT", transferAmount}
}

func checkingStatusWithRetryAndGetResponce(t provider.T, hlfProxy *utils.HlfProxyService, transferID string, status models.ChannelTransferTransferStatusResponseStatus) (*transfer.TransferStatusOK, error) {
	i := 0
	for i < transferExecutionTimeout {
		r, err := transferCli.Transfer.TransferStatus(&transfer.TransferStatusParams{IDTransfer: transferID, Context: ctx}, authOpts)
		t.Require().NoError(err)
		if r.Payload.Status == status {
			return r, nil
		} else {
			t.Log("attempt number is " + strconv.Itoa(i) + " current status is: " + string(r.GetPayload().Status))
			i++
			time.Sleep(time.Second * 1)
		}
	}
	return &transfer.TransferStatusOK{}, errors.New("didn't get status: " + string(status))
}
*/
