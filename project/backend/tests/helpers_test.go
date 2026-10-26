// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package tests_test

import (
	"context"
	"eats/backend/common"
	"encoding/json"
	"net/http"
	"testing"

	gofakeit "github.com/brianvoe/gofakeit/v7"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"eats/backend/common/shared"
	"eats/backend/common/testutils"
	ordersclient "eats/backend/orders/api/http/client"
)

func registerCustomer(ctx context.Context, t *testing.T, clients TestClients, country shared.CountryCode, address ordersclient.Address) ordersclient.CustomerUUID {
	t.Helper()

	customerToCreate := ordersclient.RegisterCustomer{
		Name:        gofakeit.Name(),
		Email:       openapi_types.Email(gofakeit.Email()),
		Address:     address,
		PhoneNumber: gofakeit.Phone(),
	}

	resp, err := clients.Orders.RegisterCustomerWithResponse(ctx, customerToCreate)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode())
	require.NotNil(t, resp.JSON201)

	return resp.JSON201.CustomerUuid
}

func registerCustomerInCity(ctx context.Context, t *testing.T, clients TestClients, country shared.CountryCode, city string) ordersclient.CustomerUUID {
	t.Helper()

	customerToCreate := ordersclient.RegisterCustomer{
		Name:        gofakeit.Name(),
		Email:       openapi_types.Email(gofakeit.Email()),
		Address:     testutils.GenerateOpenapiAddressInCity(country, city),
		PhoneNumber: gofakeit.Phone(),
	}

	resp, err := clients.Orders.RegisterCustomerWithResponse(ctx, customerToCreate)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode())
	require.NotNil(t, resp.JSON201)

	return resp.JSON201.CustomerUuid
}

func onboardRestaurant(ctx context.Context, t *testing.T, clients TestClients, restaurantUUID ordersclient.RestaurantUUID, currency shared.Currency, address ordersclient.Address) {
	t.Helper()

	menuItem := ordersclient.MenuItem{
		GrossPrice: decimal.New(10, 1),
		Name:       gofakeit.Name(),
		Ordering:   1,
		Uuid:       ordersclient.MenuItemUUID{UUID: common.NewUUIDv7()},
	}

	restaurantToCreate := ordersclient.OnboardRestaurant{
		Address:     address,
		Currency:    currency,
		Description: gofakeit.Quote(),
		Name:        gofakeit.Name(),
		MenuItems:   []ordersclient.MenuItem{menuItem},
	}

	operatorUUID := &ordersclient.OnboardRestaurantParams{OperatorUUID: common.NewUUIDv7()}

	resp, err := clients.Orders.OnboardRestaurantWithResponse(ctx, restaurantUUID, operatorUUID, restaurantToCreate)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode())
}

func assertJsonReprEqual(t *testing.T, expected, actual any) {
	t.Helper()

	expectedJSON, err := json.Marshal(expected)
	require.NoError(t, err)

	actualJSON, err := json.Marshal(actual)
	require.NoError(t, err)

	require.JSONEq(t, string(expectedJSON), string(actualJSON))
}
