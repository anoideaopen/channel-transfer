package declare

type TransferStatus string

const (
	InProgressTransferFrom       TransferStatus = "InProgressTransferFrom"
	InProgressTransferTo         TransferStatus = "InProgressTransferTo"
	InProgressTransferFromDelete TransferStatus = "InProgressTransferFromDelete"
	InProgressTransferToDelete   TransferStatus = "InProgressTransferToDelete"
	CompletedTransferFrom        TransferStatus = "CompletedTransferFrom"
	CompletedTransferTo          TransferStatus = "CompletedTransferTo"
	CompletedTransferFromDelete  TransferStatus = "CompletedTransferFromDelete"
	CompletedTransferToDelete    TransferStatus = "CompletedTransferToDelete"
	ErrorTransferFrom            TransferStatus = "ErrorTransferFrom"
	ErrorTransferTo              TransferStatus = "ErrorTransferTo"
	ErrorTransferFromDelete      TransferStatus = "ErrorTransferFromDelete"
	ErrorTransferToDelete        TransferStatus = "ErrorTransferToDelete"
	Completed                    TransferStatus = "Completed"
	Canceled                     TransferStatus = "Canceled"
	UnknownTransferStatus        TransferStatus = "Unknown"
)

func NewTransferStatus(value string) TransferStatus {
	switch value {
	case "InProgressTransferFrom":
		return InProgressTransferFrom
	case "InProgressTransferTo":
		return InProgressTransferTo
	case "InProgressTransferFromDelete":
		return InProgressTransferFromDelete
	case "InProgressTransferToDelete":
		return InProgressTransferToDelete
	case "CompletedTransferFrom":
		return CompletedTransferFrom
	case "CompletedTransferTo":
		return CompletedTransferTo
	case "CompletedTransferFromDelete":
		return CompletedTransferFromDelete
	case "CompletedTransferToDelete":
		return CompletedTransferToDelete
	case "ErrorTransferFrom":
		return ErrorTransferFrom
	case "ErrorTransferTo":
		return ErrorTransferTo
	case "ErrorTransferFromDelete":
		return ErrorTransferFromDelete
	case "ErrorTransferToDelete":
		return ErrorTransferToDelete
	case "Completed":
		return Completed
	case "Canceled":
		return Canceled
	}

	return UnknownTransferStatus
}

type EpochKind int

const (
	CreatedTransferRequest EpochKind = iota // Create transfer request
	CreatedTransferFrom                     // Created transferFrom
	CreatedTransferTo                       // Created transferTo
	FixedTransferFrom                       // Fixed WS of transferFrom
	DeletedTransferTo                       // Deleted WS of transferTo
	DeletedTransferFrom                     // Deleted WS of transferFrom
)
