package model

type TransactionKind int

const (
	TxCancelCCTransferFrom TransactionKind = iota
	TxChannelTransferByAdmin
	TxChannelTransferByCustomer
	TxCreateCCTransferTo
	NbTxCommitCCTransferFrom
	NbTxDeleteCCTransferFrom
	NbTxDeleteCCTransferTo
	QueryChannelTransfersFrom
	QueryChannelTransferTo
	QueryChannelTransferFrom
	TxChannelMultiTransferByAdmin
	TxChannelMultiTransferByCustomer
)

var (
	transactionKindValues = map[string]TransactionKind{
		"cancelCCTransferFrom":           TxCancelCCTransferFrom,
		"channelTransferByAdmin":         TxChannelTransferByAdmin,
		"channelTransferByCustomer":      TxChannelTransferByCustomer,
		"createCCTransferTo":             TxCreateCCTransferTo,
		"commitCCTransferFrom":           NbTxCommitCCTransferFrom,
		"deleteCCTransferFrom":           NbTxDeleteCCTransferFrom,
		"deleteCCTransferTo":             NbTxDeleteCCTransferTo,
		"channelTransfersFrom":           QueryChannelTransfersFrom,
		"channelTransferFrom":            QueryChannelTransferFrom,
		"channelTransferTo":              QueryChannelTransferTo,
		"channelMultiTransferByAdmin":    TxChannelMultiTransferByAdmin,
		"channelMultiTransferByCustomer": TxChannelMultiTransferByCustomer,
	}
	transactionKindKeys = map[TransactionKind]string{
		TxCancelCCTransferFrom:           "cancelCCTransferFrom",
		TxChannelTransferByAdmin:         "channelTransferByAdmin",
		TxChannelTransferByCustomer:      "channelTransferByCustomer",
		TxCreateCCTransferTo:             "createCCTransferTo",
		NbTxCommitCCTransferFrom:         "commitCCTransferFrom",
		NbTxDeleteCCTransferFrom:         "deleteCCTransferFrom",
		NbTxDeleteCCTransferTo:           "deleteCCTransferTo",
		QueryChannelTransfersFrom:        "channelTransfersFrom",
		QueryChannelTransferFrom:         "channelTransferFrom",
		QueryChannelTransferTo:           "channelTransferTo",
		TxChannelMultiTransferByAdmin:    "channelMultiTransferByAdmin",
		TxChannelMultiTransferByCustomer: "channelMultiTransferByCustomer",
	}
)

func (tk TransactionKind) Is(method string) bool {
	_, ok := transactionKindValues[method]
	return ok
}

func (tk TransactionKind) String() string {
	return transactionKindKeys[tk]
}

type StatusKind int

const (
	InProgressTransferFrom StatusKind = iota
	ErrorTransferFrom
	CompletedTransferFrom
	FromBatchNotFound

	InProgressTransferTo
	ErrorTransferTo
	CompletedTransferTo
	ToBatchNotFound

	CommitTransferFrom

	InProgressTransferToDelete
	CompletedTransferToDelete
	ErrorTransferToDelete

	InProgressTransferFromDelete
	CompletedTransferFromDelete
	ErrorTransferFromDelete

	Canceled
	Completed
	UnknownTransferStatus

	InternalErrorTransferStatus
	ErrorChannelToNotFound
	ExistsChannelTo
)

var (
	statusKindKeys = map[StatusKind]string{
		InProgressTransferFrom:       "InProgressTransferFrom",
		InProgressTransferTo:         "InProgressTransferTo",
		InProgressTransferFromDelete: "InProgressTransferFromDelete",
		InProgressTransferToDelete:   "InProgressTransferToDelete",
		CommitTransferFrom:           "CommitTransferFrom",
		CompletedTransferFrom:        "CompletedTransferFrom",
		CompletedTransferTo:          "CompletedTransferTo",
		CompletedTransferFromDelete:  "CompletedTransferFromDelete",
		CompletedTransferToDelete:    "CompletedTransferToDelete",
		ErrorTransferFrom:            "ErrorTransferFrom",
		ErrorTransferTo:              "ErrorTransferTo",
		ErrorTransferFromDelete:      "ErrorTransferFromDelete",
		ErrorTransferToDelete:        "ErrorTransferToDelete",
		Completed:                    "Completed",
		Canceled:                     "Canceled",
		UnknownTransferStatus:        "Unknown",
		ErrorChannelToNotFound:       "ErrorChannelToNotFound",
		ExistsChannelTo:              "ExistsChannelTo",
		FromBatchNotFound:            "FromBatchNotFound",
		ToBatchNotFound:              "ToBatchNotFound",
	}
	statusKindValues = map[string]StatusKind{
		"InProgressTransferFrom":       InProgressTransferFrom,
		"InProgressTransferTo":         InProgressTransferTo,
		"InProgressTransferFromDelete": InProgressTransferFromDelete,
		"InProgressTransferToDelete":   InProgressTransferToDelete,
		"CommitTransferFrom":           CommitTransferFrom,
		"CompletedTransferFrom":        CompletedTransferFrom,
		"CompletedTransferTo":          CompletedTransferTo,
		"CompletedTransferFromDelete":  CompletedTransferFromDelete,
		"CompletedTransferToDelete":    CompletedTransferToDelete,
		"ErrorTransferFrom":            ErrorTransferFrom,
		"ErrorTransferTo":              ErrorTransferTo,
		"ErrorTransferFromDelete":      ErrorTransferFromDelete,
		"ErrorTransferToDelete":        ErrorTransferToDelete,
		"Completed":                    Completed,
		"Canceled":                     Canceled,
		"Unknown":                      UnknownTransferStatus,
		"ErrorChannelToNotFound":       ErrorChannelToNotFound,
		"ExistsChannelTo":              ExistsChannelTo,
		"FromBatchNotFound":            FromBatchNotFound,
		"ToBatchNotFound":              ToBatchNotFound,
	}
)

func (sk StatusKind) String() string {
	return statusKindKeys[sk]
}

func (sk StatusKind) Is(status string) bool {
	_, ok := statusKindValues[status]
	return ok
}
