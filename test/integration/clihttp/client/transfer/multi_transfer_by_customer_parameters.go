// Code generated by go-swagger; DO NOT EDIT.

package transfer

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	"github.com/anoideaopen/channel-transfer/test/integration/clihttp/models"
)

// NewMultiTransferByCustomerParams creates a new MultiTransferByCustomerParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewMultiTransferByCustomerParams() *MultiTransferByCustomerParams {
	return &MultiTransferByCustomerParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewMultiTransferByCustomerParamsWithTimeout creates a new MultiTransferByCustomerParams object
// with the ability to set a timeout on a request.
func NewMultiTransferByCustomerParamsWithTimeout(timeout time.Duration) *MultiTransferByCustomerParams {
	return &MultiTransferByCustomerParams{
		timeout: timeout,
	}
}

// NewMultiTransferByCustomerParamsWithContext creates a new MultiTransferByCustomerParams object
// with the ability to set a context for a request.
func NewMultiTransferByCustomerParamsWithContext(ctx context.Context) *MultiTransferByCustomerParams {
	return &MultiTransferByCustomerParams{
		Context: ctx,
	}
}

// NewMultiTransferByCustomerParamsWithHTTPClient creates a new MultiTransferByCustomerParams object
// with the ability to set a custom HTTPClient for a request.
func NewMultiTransferByCustomerParamsWithHTTPClient(client *http.Client) *MultiTransferByCustomerParams {
	return &MultiTransferByCustomerParams{
		HTTPClient: client,
	}
}

/*
MultiTransferByCustomerParams contains all the parameters to send to the API endpoint

	for the multi transfer by customer operation.

	Typically these are written to a http.Request.
*/
type MultiTransferByCustomerParams struct {

	// Body.
	Body *models.ChannelTransferMultiTransferBeginCustomerRequest

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the multi transfer by customer params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *MultiTransferByCustomerParams) WithDefaults() *MultiTransferByCustomerParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the multi transfer by customer params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *MultiTransferByCustomerParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the multi transfer by customer params
func (o *MultiTransferByCustomerParams) WithTimeout(timeout time.Duration) *MultiTransferByCustomerParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the multi transfer by customer params
func (o *MultiTransferByCustomerParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the multi transfer by customer params
func (o *MultiTransferByCustomerParams) WithContext(ctx context.Context) *MultiTransferByCustomerParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the multi transfer by customer params
func (o *MultiTransferByCustomerParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the multi transfer by customer params
func (o *MultiTransferByCustomerParams) WithHTTPClient(client *http.Client) *MultiTransferByCustomerParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the multi transfer by customer params
func (o *MultiTransferByCustomerParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the multi transfer by customer params
func (o *MultiTransferByCustomerParams) WithBody(body *models.ChannelTransferMultiTransferBeginCustomerRequest) *MultiTransferByCustomerParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the multi transfer by customer params
func (o *MultiTransferByCustomerParams) SetBody(body *models.ChannelTransferMultiTransferBeginCustomerRequest) {
	o.Body = body
}

// WriteToRequest writes these params to a swagger request
func (o *MultiTransferByCustomerParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error
	if o.Body != nil {
		if err := r.SetBodyParam(o.Body); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
