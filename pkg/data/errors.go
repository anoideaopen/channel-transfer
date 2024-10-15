package data

import "errors"

// Standard data package errors.
var (
	ErrInvalidType    = errors.New("invalid type")
	ErrObjectNotFound = errors.New("object not found")
)
