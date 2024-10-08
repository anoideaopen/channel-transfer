// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"strconv"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// ChannelTransferTransferBeginCustomerRequest TransferBeginCustomerRequest request for tokens transfer by tokens owner
//
// swagger:model channel_transferTransferBeginCustomerRequest
type ChannelTransferTransferBeginCustomerRequest struct {

	// Amount of tokens to transfer
	Amount string `json:"amount,omitempty"`

	// destination channel
	ChannelTo string `json:"channelTo,omitempty"`

	// general parameters
	Generals *ChannelTransferGeneralParams `json:"generals,omitempty"`

	// transfer ID (should be unique)
	IDTransfer string `json:"idTransfer,omitempty"`

	// options
	Options []*ProtobufOption `json:"options"`

	// Tokens to transfer
	Token string `json:"token,omitempty"`
}

// Validate validates this channel transfer transfer begin customer request
func (m *ChannelTransferTransferBeginCustomerRequest) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateGenerals(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateOptions(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ChannelTransferTransferBeginCustomerRequest) validateGenerals(formats strfmt.Registry) error {
	if swag.IsZero(m.Generals) { // not required
		return nil
	}

	if m.Generals != nil {
		if err := m.Generals.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("generals")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("generals")
			}
			return err
		}
	}

	return nil
}

func (m *ChannelTransferTransferBeginCustomerRequest) validateOptions(formats strfmt.Registry) error {
	if swag.IsZero(m.Options) { // not required
		return nil
	}

	for i := 0; i < len(m.Options); i++ {
		if swag.IsZero(m.Options[i]) { // not required
			continue
		}

		if m.Options[i] != nil {
			if err := m.Options[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("options" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("options" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// ContextValidate validate this channel transfer transfer begin customer request based on the context it is used
func (m *ChannelTransferTransferBeginCustomerRequest) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateGenerals(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateOptions(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ChannelTransferTransferBeginCustomerRequest) contextValidateGenerals(ctx context.Context, formats strfmt.Registry) error {

	if m.Generals != nil {

		if swag.IsZero(m.Generals) { // not required
			return nil
		}

		if err := m.Generals.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("generals")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("generals")
			}
			return err
		}
	}

	return nil
}

func (m *ChannelTransferTransferBeginCustomerRequest) contextValidateOptions(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.Options); i++ {

		if m.Options[i] != nil {

			if swag.IsZero(m.Options[i]) { // not required
				return nil
			}

			if err := m.Options[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("options" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("options" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// MarshalBinary interface implementation
func (m *ChannelTransferTransferBeginCustomerRequest) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ChannelTransferTransferBeginCustomerRequest) UnmarshalBinary(b []byte) error {
	var res ChannelTransferTransferBeginCustomerRequest
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
