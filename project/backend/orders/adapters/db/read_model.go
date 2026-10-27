package db

import (
	"context"
	"fmt"

	"eats/backend/orders/adapters/db/dbmodels"

	"github.com/jackc/pgx/v5/pgxpool"

	"eats/backend/orders/api/http"
)

// ReadModel provides read-optimized queries that return HTTP response types directly.
type ReadModel struct {
	db *pgxpool.Pool
}

func NewReadModel(db *pgxpool.Pool) *ReadModel {
	if db == nil {
		panic("db connection pool cannot be nil")
	}
	return &ReadModel{db: db}
}

func (r ReadModel) ListMenuItemsWithRestaurant(ctx context.Context) ([]http.MenuItemWithRestaurant, error) {
	queries := dbmodels.New(r.db)

	list, err := queries.ListMenuItemsWithRestaurant(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing menu items: %w", err)
	}

	return httpMenuItemWithRestaurantFromDB(list), nil
}

func httpMenuItemWithRestaurantFromDB(list []dbmodels.ListMenuItemsWithRestaurantRow) []http.MenuItemWithRestaurant {
	result := make([]http.MenuItemWithRestaurant, 0, len(list))
	for _, item := range list {
		result = append(result, http.MenuItemWithRestaurant{
			Currency:       item.Currency,
			GrossPrice:     item.GrossPrice,
			MenuItemName:   item.MenuItemName,
			MenuItemUuid:   item.MenuItemUuid,
			RestaurantName: item.RestaurantName,
			RestaurantUuid: item.RestaurantUuid,
		})
	}

	return result
}
