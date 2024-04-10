package model

import "errors"

// Standardized errors for working with business logic.
var (
	ErrTransferNotFound = errors.New("transfer not found")
	ErrUnknown          = errors.New("unknown error")
)
