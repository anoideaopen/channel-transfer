// Code generated by go-swagger; DO NOT EDIT.

package transfer

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/anoideaopen/channel-transfer/test/integration/clihttp/models"
)

// TransferStatusReader is a Reader for the TransferStatus structure.
type TransferStatusReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *TransferStatusReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewTransferStatusOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 500:
		result := NewTransferStatusInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewTransferStatusDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewTransferStatusOK creates a TransferStatusOK with default headers values
func NewTransferStatusOK() *TransferStatusOK {
	return &TransferStatusOK{}
}

/*
TransferStatusOK describes a response with status code 200, with default header values.

A successful response.
*/
type TransferStatusOK struct {
	Payload *models.ChannelTransferTransferStatusResponse
}

// IsSuccess returns true when this transfer status o k response has a 2xx status code
func (o *TransferStatusOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this transfer status o k response has a 3xx status code
func (o *TransferStatusOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this transfer status o k response has a 4xx status code
func (o *TransferStatusOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this transfer status o k response has a 5xx status code
func (o *TransferStatusOK) IsServerError() bool {
	return false
}

// IsCode returns true when this transfer status o k response a status code equal to that given
func (o *TransferStatusOK) IsCode(code int) bool {
	return code == 200
}

// Code gets the status code for the transfer status o k response
func (o *TransferStatusOK) Code() int {
	return 200
}

func (o *TransferStatusOK) Error() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[GET /v1/status/{idTransfer}][%d] transferStatusOK %s", 200, payload)
}

func (o *TransferStatusOK) String() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[GET /v1/status/{idTransfer}][%d] transferStatusOK %s", 200, payload)
}

func (o *TransferStatusOK) GetPayload() *models.ChannelTransferTransferStatusResponse {
	return o.Payload
}

func (o *TransferStatusOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ChannelTransferTransferStatusResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewTransferStatusInternalServerError creates a TransferStatusInternalServerError with default headers values
func NewTransferStatusInternalServerError() *TransferStatusInternalServerError {
	return &TransferStatusInternalServerError{}
}

/*
TransferStatusInternalServerError describes a response with status code 500, with default header values.

Internal server error
*/
type TransferStatusInternalServerError struct {
	Payload *models.ChannelTransferErrorResponse
}

// IsSuccess returns true when this transfer status internal server error response has a 2xx status code
func (o *TransferStatusInternalServerError) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this transfer status internal server error response has a 3xx status code
func (o *TransferStatusInternalServerError) IsRedirect() bool {
	return false
}

// IsClientError returns true when this transfer status internal server error response has a 4xx status code
func (o *TransferStatusInternalServerError) IsClientError() bool {
	return false
}

// IsServerError returns true when this transfer status internal server error response has a 5xx status code
func (o *TransferStatusInternalServerError) IsServerError() bool {
	return true
}

// IsCode returns true when this transfer status internal server error response a status code equal to that given
func (o *TransferStatusInternalServerError) IsCode(code int) bool {
	return code == 500
}

// Code gets the status code for the transfer status internal server error response
func (o *TransferStatusInternalServerError) Code() int {
	return 500
}

func (o *TransferStatusInternalServerError) Error() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[GET /v1/status/{idTransfer}][%d] transferStatusInternalServerError %s", 500, payload)
}

func (o *TransferStatusInternalServerError) String() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[GET /v1/status/{idTransfer}][%d] transferStatusInternalServerError %s", 500, payload)
}

func (o *TransferStatusInternalServerError) GetPayload() *models.ChannelTransferErrorResponse {
	return o.Payload
}

func (o *TransferStatusInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ChannelTransferErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewTransferStatusDefault creates a TransferStatusDefault with default headers values
func NewTransferStatusDefault(code int) *TransferStatusDefault {
	return &TransferStatusDefault{
		_statusCode: code,
	}
}

/*
TransferStatusDefault describes a response with status code -1, with default header values.

An unexpected error response.
*/
type TransferStatusDefault struct {
	_statusCode int

	Payload *models.GooglerpcStatus
}

// IsSuccess returns true when this transfer status default response has a 2xx status code
func (o *TransferStatusDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this transfer status default response has a 3xx status code
func (o *TransferStatusDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this transfer status default response has a 4xx status code
func (o *TransferStatusDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this transfer status default response has a 5xx status code
func (o *TransferStatusDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this transfer status default response a status code equal to that given
func (o *TransferStatusDefault) IsCode(code int) bool {
	return o._statusCode == code
}

// Code gets the status code for the transfer status default response
func (o *TransferStatusDefault) Code() int {
	return o._statusCode
}

func (o *TransferStatusDefault) Error() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[GET /v1/status/{idTransfer}][%d] transferStatus default %s", o._statusCode, payload)
}

func (o *TransferStatusDefault) String() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[GET /v1/status/{idTransfer}][%d] transferStatus default %s", o._statusCode, payload)
}

func (o *TransferStatusDefault) GetPayload() *models.GooglerpcStatus {
	return o.Payload
}

func (o *TransferStatusDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.GooglerpcStatus)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
