package http

import (
	"context"
	"eats/backend/common/shared"
	"eats/backend/orders/adapters/db/dbmodels"

	"eats/backend/common"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	db *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) Handler {
	if db == nil {
		panic("db is nil")
	}

	return Handler{
		db: db,
	}
}

func Register(ctx context.Context, e common.EchoRouter, handler Handler) error {
	RegisterHandlers(e, NewStrictHandler(handler, nil))

	return nil
}

func (h Handler) RegisterCustomer(ctx context.Context, request RegisterCustomerRequestObject) (RegisterCustomerResponseObject, error) {
	queries := dbmodels.New(h.db)

	address, err := openapiAddressToSharedAddress(request.Body.Address)
	if err != nil {
		return nil, err
	}

	customerUUID := common.NewUUIDv7()

	err = queries.InsertCustomer(ctx, dbmodels.InsertCustomerParams{
		CustomerUuid: customerUUID,
		Name:         request.Body.Name,
		Email:        string(request.Body.Email),
		Address:      address,
		PhoneNumber:  request.Body.PhoneNumber,
	})
	if err != nil {
		return nil, err
	}

	return RegisterCustomer201JSONResponse{
		CustomerUuid: customerUUID,
	}, nil
}

func openapiAddressToSharedAddress(addr Address) (shared.Address, error) {
	return shared.NewAddress(
		addr.Line1,
		addr.Line2,
		addr.PostalCode,
		addr.City,
		addr.CountryCode,
	)
}
