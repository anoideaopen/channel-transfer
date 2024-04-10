package test

import (
	"context"
	"os"
	"testing"
	"time"

	cligrpc "github.com/anoideaopen/channel-transfer/test/integration/cligrpc"
	utils "github.com/anoideaopen/testnet-util"
	tr "github.com/anoideaopen/testnet-util/transfer"
	"github.com/google/uuid"
	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/runner"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/typepb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// TestTransferByAdminGRPC - add user emit fiat tokens and transfer to channel cc with channelTransferByAdmin by GRPC then check balance and allowedBalance
func TestTransferByAdminGRPC(t *testing.T) {
	runner.Run(t, "Channel transfer by admin", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("Channel transfer by admin grpc")
		t.Tags("negative", "channel-transfer")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))

		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)
		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, "1000")

		transportCredentials := insecure.NewCredentials()
		conn, err := grpc.Dial(targetGrpc, grpc.WithTransportCredentials(transportCredentials))
		require.NoError(t, err)
		defer conn.Close()

		c := cligrpc.NewAPIClient(conn)

		t.WithNewStep("Channel transfer by admin grpc", func(sCtx provider.StepCtx) {
			transferID := uuid.NewString()
			channelTransferArgs := []string{transferID, "CC", user.UserAddressBase58Check, "FIAT", "250"}

			sa, err := utils.SignExpand(issuer.IssuerEd25519PrivateKey, issuer.IssuerEd25519PublicKey, channelFrom, channelFrom, channelTransferByAdminMethod, channelTransferArgs, utils.GetNonce(), uuid.NewString())
			sCtx.Require().NoError(err)

			transfer := &cligrpc.TransferBeginAdminRequest{
				Generals: &cligrpc.GeneralParams{
					MethodName: channelTransferByAdminMethod,
					RequestId:  sa[0],
					Chaincode:  sa[1],
					Channel:    sa[2],
					Nonce:      sa[len(sa)-3],
					PublicKey:  sa[len(sa)-2],
					Sign:       sa[len(sa)-1],
				},
				IdTransfer: channelTransferArgs[0],
				ChannelTo:  channelTransferArgs[1],
				Address:    channelTransferArgs[2],
				Token:      channelTransferArgs[3],
				Amount:     channelTransferArgs[4],
			}

			r, err := c.TransferByAdmin(clientCtx, transfer)
			sCtx.Require().NoError(err)
			sCtx.Require().Equal(cligrpc.TransferStatusResponse_STATUS_IN_PROCESS, r.Status)

			transferStatusRequest := &cligrpc.TransferStatusRequest{
				IdTransfer: transferID,
			}

			excludeStatus := cligrpc.TransferStatusResponse_STATUS_IN_PROCESS.String()
			value, err := anypb.New(wrapperspb.String(excludeStatus))
			sCtx.Require().NoError(err)

			transferStatusRequest.Options = append(transferStatusRequest.Options, &typepb.Option{
				Name:  "excludeStatus",
				Value: value,
			})

			ctx, cancel := context.WithTimeout(clientCtx, 120*time.Second)
			defer cancel()

			statusResponce, err := c.TransferStatus(ctx, transferStatusRequest)
			t.Require().NoError(err)
			sCtx.Require().Equal(cligrpc.TransferStatusResponse_STATUS_COMPLETED, statusResponce.Status)

			utils.CheckBalanceEqual(t, *hlfProxy, user.UserAddressBase58Check, "fiat", "750")
			tr.CheckAllowedBalanceEqual(t, hlfProxy, user.UserAddressBase58Check, "cc", "FIAT", "250")
		})
	})
}

// TestTransferByCustomerGRPC - add user emit fiat tokens and transfer to channel cc with channelTransferByCustomer by GRPC then check balance and allowedBalance
func TestTransferByCustomerGRPC(t *testing.T) {
	runner.Run(t, "Channel transfer by customer", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("Channel transfer by customer grpc")
		t.Tags("negative", "channel-transfer")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))

		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)
		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, "1000")

		transportCredentials := insecure.NewCredentials()
		conn, err := grpc.Dial(targetGrpc, grpc.WithTransportCredentials(transportCredentials))
		require.NoError(t, err)
		defer conn.Close()

		c := cligrpc.NewAPIClient(conn)

		t.WithNewStep("Channel transfer by customer grpc", func(sCtx provider.StepCtx) {
			transferID := uuid.NewString()
			channelTransferArgs := []string{transferID, "CC", "FIAT", "250"}

			sa, err := utils.SignExpand(user.UserEd25519PrivateKey, user.UserEd25519PublicKey, channelFrom, channelFrom, channelTransferByCustomerMethod, channelTransferArgs, utils.GetNonce(), uuid.NewString())
			sCtx.Require().NoError(err)

			transfer := &cligrpc.TransferBeginCustomerRequest{
				Generals: &cligrpc.GeneralParams{
					MethodName: channelTransferByCustomerMethod,
					RequestId:  sa[0],
					Chaincode:  sa[1],
					Channel:    sa[2],
					Nonce:      sa[len(sa)-3],
					PublicKey:  sa[len(sa)-2],
					Sign:       sa[len(sa)-1],
				},
				IdTransfer: channelTransferArgs[0],
				ChannelTo:  channelTransferArgs[1],
				Token:      channelTransferArgs[2],
				Amount:     channelTransferArgs[3],
			}

			r, err := c.TransferByCustomer(clientCtx, transfer)
			sCtx.Require().NoError(err)
			sCtx.Require().Equal(cligrpc.TransferStatusResponse_STATUS_IN_PROCESS, r.Status)

			transferStatusRequest := &cligrpc.TransferStatusRequest{
				IdTransfer: transferID,
			}

			excludeStatus := cligrpc.TransferStatusResponse_STATUS_IN_PROCESS.String()
			value, err := anypb.New(wrapperspb.String(excludeStatus))
			sCtx.Require().NoError(err)

			transferStatusRequest.Options = append(transferStatusRequest.Options, &typepb.Option{
				Name:  "excludeStatus",
				Value: value,
			})

			ctx, cancel := context.WithTimeout(clientCtx, 120*time.Second)
			defer cancel()

			statusResponce, err := c.TransferStatus(ctx, transferStatusRequest)
			t.Require().NoError(err)
			sCtx.Require().Equal(cligrpc.TransferStatusResponse_STATUS_COMPLETED, statusResponce.Status)

			utils.CheckBalanceEqual(t, *hlfProxy, user.UserAddressBase58Check, "fiat", "750")
			tr.CheckAllowedBalanceEqual(t, hlfProxy, user.UserAddressBase58Check, "cc", "FIAT", "250")
		})
	})
}

// TestTransferStatusWrongTransferIdGRPC - get transfer status wrong id by GRPC
func TestTransferStatusWrongTransferIdGRPC(t *testing.T) {
	runner.Run(t, "TestTransferStatusWrongTransferIdGRPC", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("Get transfer status wrong id grpc")
		t.Tags("negative", "channel-transfer")

		transportCredentials := insecure.NewCredentials()
		conn, err := grpc.Dial(targetGrpc, grpc.WithTransportCredentials(transportCredentials))
		require.NoError(t, err)
		defer conn.Close()

		c := cligrpc.NewAPIClient(conn)

		t.WithNewStep("Get transfer status wrong id grpc", func(sCtx provider.StepCtx) {
			transferStatusRequest := &cligrpc.TransferStatusRequest{
				IdTransfer: "1",
			}
			_, err := c.TransferStatus(clientCtx, transferStatusRequest)
			sCtx.Require().Contains(err.Error(), "object not found")
		})
	})
}

// TestGRPCTransferStatusFilter - get transfer status with filter by GRPC
func TestGRPCTransferStatusFilter(t *testing.T) {
	runner.Run(t, "TestGRPCTransferStatusFilter", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestGRPCTransferStatusFilter")
		t.Tags("channel-transfer")

		hlfProxy := utils.NewHlfProxyService(os.Getenv(utils.HlfProxyURL), os.Getenv(utils.HlfProxyAuthToken))

		issuer := utils.AddIssuer(t, *hlfProxy, os.Getenv(utils.FiatIssuerPrivateKey))
		user := utils.AddUser(t, *hlfProxy)
		utils.EmitGetTxIDAndCheckBalance(t, *hlfProxy, user.UserAddressBase58Check, issuer, channelFrom, channelFrom, "1000")

		transportCredentials := insecure.NewCredentials()
		conn, err := grpc.Dial(targetGrpc, grpc.WithTransportCredentials(transportCredentials))
		require.NoError(t, err)
		defer conn.Close()

		c := cligrpc.NewAPIClient(conn)

		t.WithNewStep("Get transfer status with filter", func(sCtx provider.StepCtx) {
			transferID := uuid.NewString()
			channelTransferArgs := []string{transferID, "CC", "FIAT", "250"}

			sa, err := utils.SignExpand(user.UserEd25519PrivateKey, user.UserEd25519PublicKey, channelFrom, channelFrom, channelTransferByCustomerMethod, channelTransferArgs, utils.GetNonce(), uuid.NewString())
			sCtx.Require().NoError(err)

			transfer := &cligrpc.TransferBeginCustomerRequest{
				Generals: &cligrpc.GeneralParams{
					MethodName: channelTransferByCustomerMethod,
					RequestId:  sa[0],
					Chaincode:  sa[1],
					Channel:    sa[2],
					Nonce:      sa[len(sa)-3],
					PublicKey:  sa[len(sa)-2],
					Sign:       sa[len(sa)-1],
				},
				IdTransfer: channelTransferArgs[0],
				ChannelTo:  channelTransferArgs[1],
				Token:      channelTransferArgs[2],
				Amount:     channelTransferArgs[3],
			}

			r, err := c.TransferByCustomer(clientCtx, transfer)
			sCtx.Require().NoError(err)
			sCtx.Require().Equal(cligrpc.TransferStatusResponse_STATUS_IN_PROCESS, r.Status)

			transferStatusRequest := &cligrpc.TransferStatusRequest{
				IdTransfer: transferID,
			}

			excludeStatus := cligrpc.TransferStatusResponse_STATUS_IN_PROCESS.String()
			value, err := anypb.New(wrapperspb.String(excludeStatus))
			sCtx.Require().NoError(err)

			transferStatusRequest.Options = append(transferStatusRequest.Options, &typepb.Option{
				Name:  "excludeStatus",
				Value: value,
			})

			ctx, cancel := context.WithTimeout(clientCtx, 120*time.Second)
			defer cancel()

			statusResponce, err := c.TransferStatus(ctx, transferStatusRequest)
			t.Require().NoError(err)
			sCtx.Require().Equal(cligrpc.TransferStatusResponse_STATUS_COMPLETED, statusResponce.Status)
		})
	})
}

// TestGRPCTransferWrongStatusFilter - get transfer status with wrong filter by GRPC
func TestGRPCTransferWrongStatusFilter(t *testing.T) {
	runner.Run(t, "TestGRPCTransferWrongStatusFilter", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("TestGRPCTransferWrongStatusFilter")
		t.Tags("negative", "channel-transfer")

		transportCredentials := insecure.NewCredentials()
		conn, err := grpc.Dial(targetGrpc, grpc.WithTransportCredentials(transportCredentials))
		require.NoError(t, err)
		defer conn.Close()

		c := cligrpc.NewAPIClient(conn)

		t.WithNewStep("Get transfer status with wrong filter value STATUS_CANCELED", func(sCtx provider.StepCtx) {
			transferStatusRequest := &cligrpc.TransferStatusRequest{
				IdTransfer: "1",
			}

			excludeStatus := cligrpc.TransferStatusResponse_STATUS_CANCELED.String()
			value, err := anypb.New(wrapperspb.String(excludeStatus))
			sCtx.Require().NoError(err)

			transferStatusRequest.Options = append(transferStatusRequest.Options, &typepb.Option{
				Name:  "excludeStatus",
				Value: value,
			})

			_, err = c.TransferStatus(clientCtx, transferStatusRequest)
			sCtx.Require().Contains(err.Error(), "exclude status not valid")
		})

		t.WithNewStep("Get transfer status with wrong filter value STATUS_COMPLETED", func(sCtx provider.StepCtx) {
			transferStatusRequest := &cligrpc.TransferStatusRequest{
				IdTransfer: "1",
			}

			excludeStatus := cligrpc.TransferStatusResponse_STATUS_COMPLETED.String()
			value, err := anypb.New(wrapperspb.String(excludeStatus))
			sCtx.Require().NoError(err)

			transferStatusRequest.Options = append(transferStatusRequest.Options, &typepb.Option{
				Name:  "excludeStatus",
				Value: value,
			})

			_, err = c.TransferStatus(clientCtx, transferStatusRequest)
			sCtx.Require().Contains(err.Error(), "exclude status not valid")
		})

		t.WithNewStep("Get transfer status with wrong filter value STATUS_ERROR", func(sCtx provider.StepCtx) {
			transferStatusRequest := &cligrpc.TransferStatusRequest{
				IdTransfer: "1",
			}

			excludeStatus := cligrpc.TransferStatusResponse_STATUS_ERROR.String()
			value, err := anypb.New(wrapperspb.String(excludeStatus))
			sCtx.Require().NoError(err)

			transferStatusRequest.Options = append(transferStatusRequest.Options, &typepb.Option{
				Name:  "excludeStatus",
				Value: value,
			})

			_, err = c.TransferStatus(clientCtx, transferStatusRequest)
			sCtx.Require().Contains(err.Error(), "exclude status not valid")
		})

		t.WithNewStep("Get transfer status with wrong filter value", func(sCtx provider.StepCtx) {
			transferStatusRequest := &cligrpc.TransferStatusRequest{
				IdTransfer: "1",
			}

			excludeStatus := "1"
			value, err := anypb.New(wrapperspb.String(excludeStatus))
			sCtx.Require().NoError(err)

			transferStatusRequest.Options = append(transferStatusRequest.Options, &typepb.Option{
				Name:  "excludeStatus",
				Value: value,
			})

			_, err = c.TransferStatus(clientCtx, transferStatusRequest)
			sCtx.Require().Contains(err.Error(), "exclude status not found")
		})
	})
}
