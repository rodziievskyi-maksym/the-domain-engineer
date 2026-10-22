package http

import (
	"context"
	"eats/backend/common"
)

type CustomerRepository interface {
	RegisterCustomer(ctx context.Context, customerUUID common.UUID, customer RegisterCustomer) error
}

type Handler struct {
	customerRepo CustomerRepository
}

func NewHandler(customerRepo CustomerRepository) Handler {
	if customerRepo == nil {
		panic("customerRepo is nil")
	}

	return Handler{
		customerRepo: customerRepo,
	}
}

func Register(ctx context.Context, e common.EchoRouter, handler Handler) error {
	RegisterHandlers(e, NewStrictHandler(handler, nil))

	return nil
}

func (h Handler) RegisterCustomer(ctx context.Context, request RegisterCustomerRequestObject) (RegisterCustomerResponseObject, error) {
	customerUUID := common.NewUUIDv7()

	if err := h.customerRepo.RegisterCustomer(ctx, customerUUID, RegisterCustomer{
		Name:        request.Body.Name,
		Email:       request.Body.Email,
		PhoneNumber: request.Body.PhoneNumber,
		Address:     request.Body.Address,
	}); err != nil {
		return nil, err
	}

	return RegisterCustomer201JSONResponse{
		CustomerUuid: customerUUID,
	}, nil
}
