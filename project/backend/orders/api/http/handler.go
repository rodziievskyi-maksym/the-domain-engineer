package http

import (
	"context"
	"fmt"

	"eats/backend/common"

	"eats/backend/common/shared"
	"eats/backend/orders/app"
)

type Handler struct {
	customerService *app.Service
}

func NewHandler(
	customerService *app.Service,
) Handler {
	if customerService == nil {
		panic("customerService cannot be nil")
	}

	return Handler{
		customerService: customerService,
	}
}

func (h Handler) RegisterCustomer(ctx context.Context, request RegisterCustomerRequestObject) (RegisterCustomerResponseObject, error) {
	customerUUID := app.CustomerUUID{UUID: common.NewUUIDv7()}

	addr, err := openapiAddressToSharedAddress(request.Body.Address)
	if err != nil {
		return nil, fmt.Errorf("convert address failed: %w", err)
	}

	err = h.customerService.RegisterCustomer(ctx, app.Customer{
		CustomerUUID: customerUUID,
		Name:         request.Body.Name,
		Email:        string(request.Body.Email),
		Address:      addr,
		PhoneNumber:  request.Body.PhoneNumber,
	})
	if err != nil {
		return nil, err
	}

	return RegisterCustomer201JSONResponse{
		CustomerUuid: customerUUID,
	}, nil
}

func Register(ctx context.Context, e EchoRouter, handler Handler) error {
	RegisterHandlers(e, NewStrictHandler(handler, nil))

	return nil
}

func openapiAddressToSharedAddress(addr Address) (shared.Address, error) {
	sharedAddr, err := shared.NewAddress(
		addr.Line1,
		addr.Line2,
		addr.PostalCode,
		addr.City,
		addr.CountryCode,
	)
	if err != nil {
		return shared.Address{}, err
	}

	return sharedAddr, nil
}
