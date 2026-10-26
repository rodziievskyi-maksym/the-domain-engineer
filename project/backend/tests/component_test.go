// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package tests_test

import (
	"eats/backend/common"
	"eats/backend/orders/adapters/db"
	ordersclient "eats/backend/orders/api/http/client"
	"eats/backend/orders/app"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"eats/backend/common/testutils"
)

func TestComponent_CriticalFlow(t *testing.T) {
	t.Parallel()
	clients := NewTestClients(t)

	ctx := t.Context()

	country := testutils.GenerateRandomCountry()

	customerUUID := registerCustomerInCity(ctx, t, clients, country, "Some city")
	assert.NotEmpty(t, customerUUID)
}

func TestComponent_OnboardRestaurant(t *testing.T) {
	t.Parallel()
	clients := NewTestClients(t)

	ctx := t.Context()

	restUUID := ordersclient.RestaurantUUID{UUID: common.NewUUIDv7()}
	currency := testutils.GenerateRandomCurrency()
	country := testutils.GenerateRandomCountry()
	address := testutils.GenerateRandomOpenapiAddress(country)

	onboardRestaurant(ctx, t, clients, restUUID, currency, address)
}

func TestComponent_CreateQuotes(t *testing.T) {
	t.Parallel()
	clients := NewTestClients(t)

	ctx := t.Context()

	restUUID := ordersclient.RestaurantUUID{UUID: common.NewUUIDv7()}
	currency := testutils.GenerateRandomCurrency()
	country := testutils.GenerateRandomCountry()
	address := testutils.GenerateRandomOpenapiAddress(country)

	customerUUID := registerCustomer(ctx, t, clients, country, address)

	fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!", customerUUID)

	onboardRestaurant(ctx, t, clients, restUUID, currency, address)

	dbPool := testutils.NewDB(t)
	restRepo := db.NewRestaurantRepository(dbPool)

	menuItems, err := restRepo.GetRestaurantMenu(ctx, app.RestaurantUUID{UUID: restUUID.UUID})
	assert.NoError(t, err)
	assert.NotEmpty(t, menuItems)

	ccqp := &ordersclient.CustomerCreateQuoteParams{CustomerUUID: customerUUID}

	orderItems := make([]ordersclient.OrderItem, 0, len(menuItems.Positions))
	for _, position := range menuItems.Positions {
		orderItems = append(orderItems, ordersclient.OrderItem{
			MenuItemUuid: ordersclient.MenuItemUUID{UUID: position.MenuItemUUID.UUID},
			Quantity:     1,
		})
	}

	body := ordersclient.CreateQuoteRequest{
		DeliveryAddress: address,
		Items:           orderItems,
		RestaurantUuid:  ordersclient.RestaurantUUID{UUID: restUUID.UUID},
	}
	resp, err := clients.Orders.CustomerCreateQuoteWithResponse(ctx, ccqp, body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode())
	require.NotNil(t, resp.JSON201)

	require.Equal(t, currency, resp.JSON201.Currency)
	//resp.JSON201.TotalGross
}
