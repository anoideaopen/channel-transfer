package methods

import "github.com/anoideaopen/channel-transfer/pkg/model"

func IsCreateTransferMethod(method string) bool {
	switch method {
	case model.TxChannelTransferByCustomer.String(),
		model.TxChannelTransferByAdmin.String(),
		model.TxChannelMultiTransferByCustomer.String(),
		model.TxChannelMultiTransferByAdmin.String(),
		model.TxCreateCCTransferTo.String():
		return true
	default:
	}
	return false

}

func IsTransferFromMethod(method string) bool {
	switch method {
	case model.TxChannelTransferByCustomer.String(),
		model.TxChannelTransferByAdmin.String(),
		model.TxChannelMultiTransferByCustomer.String(),
		model.TxChannelMultiTransferByAdmin.String():
		return true
	default:
	}
	return false
}

func IsByAdminMethod(method string) bool {
	switch method {
	case model.TxChannelTransferByAdmin.String(),
		model.TxChannelMultiTransferByAdmin.String():
		return true
	default:
	}
	return false
}

func IsMultiTransferMethod(method string) bool {
	switch method {
	case model.TxChannelMultiTransferByAdmin.String(),
		model.TxChannelMultiTransferByCustomer.String():
		return true
	default:
	}
	return false
}
