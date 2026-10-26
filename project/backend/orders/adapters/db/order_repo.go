package db

import (
	"context"
	"fmt"
	"time"

	"eats/backend/common"
	"eats/backend/orders/adapters/db/dbmodels"

	"eats/backend/common/shared"
	"eats/backend/orders/app"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrdersRepository struct {
	db *pgxpool.Pool
}

func NewOrdersRepository(db *pgxpool.Pool) app.OrderRepository {
	return &OrdersRepository{db: db}
}

func (r *OrdersRepository) CreateQuote(
	ctx context.Context,
	restaurantUUID app.RestaurantUUID,
	menuItems app.CreateQuoteItems, // 2 french fries, 1 burger
	updateFn func(
		ctx context.Context,
		menuItems map[app.RestaurantMenuItemUUID]app.MenuItem,
		restaurantCurrency shared.Currency,
		restaurantAddress shared.Address,
	) (app.Quote, []app.QuoteMenuItem, error),
) (app.Quote, error) {
	var quote app.Quote

	err := common.UpdateInTx(ctx, r.db, func(ctx context.Context, tx pgx.Tx) error {
		queries := dbmodels.New(tx)

		requestedMenuItemsUUIDs := make([]common.UUID, 0, len(menuItems))
		for _, item := range menuItems {
			requestedMenuItemsUUIDs = append(requestedMenuItemsUUIDs, item.MenuItemUUID.UUID)
		}

		appMenuItems, err := r.getMenuItems(ctx, queries, restaurantUUID, requestedMenuItemsUUIDs)
		if err != nil {
			return err
		}

		restaurant, err := queries.GetRestaurant(ctx, restaurantUUID)
		if err != nil {
			return fmt.Errorf("failed to get restaurant: %w", err)
		}

		var quoteMenuItems []app.QuoteMenuItem
		quote, quoteMenuItems, err = updateFn(
			ctx,
			appMenuItems,
			restaurant.Currency,
			restaurant.Address,
		)
		if err != nil {
			return fmt.Errorf("failed to create quote using updateFn: %w", err)
		}

		err = queries.AddQuote(ctx, dbmodels.AddQuoteParams{
			QuoteUuid:          quote.QuoteUUID,
			CustomerUuid:       quote.CustomerUUID,
			RestaurantUuid:     quote.RestaurantUUID,
			DeliveryAddress:    quote.DeliveryAddress,
			CreatedAt:          time.Now(),
			ItemsSubtotalGross: quote.ItemsSubtotalGross,
			ServiceFeeGross:    quote.ServiceFeeGross,
			DeliveryFeeGross:   quote.DeliveryFeeGross,
			TotalAmountGross:   quote.TotalAmountGross,
			TotalTax:           quote.TotalTax,
			Currency:           quote.Currency,
		})
		if err != nil {
			return fmt.Errorf("failed to add quote: %w", err)
		}

		quoteItems := dbQuoteItemsFromApp(quoteMenuItems, quote)

		if _, err := queries.AddQuoteItems(ctx, quoteItems); err != nil {
			return fmt.Errorf("failed to add quote items for quote %s: %w", quote.QuoteUUID, err)
		}

		return nil
	})
	if err != nil {
		return app.Quote{}, err
	}

	return quote, nil
}

func dbQuoteItemsFromApp(menuItems []app.QuoteMenuItem, quote app.Quote) []dbmodels.AddQuoteItemsParams {
	quoteItems := make([]dbmodels.AddQuoteItemsParams, 0, len(menuItems))
	for _, position := range menuItems {
		quoteItems = append(quoteItems, dbmodels.AddQuoteItemsParams{
			QuoteItemUuid: common.NewUUIDv7(),
			QuoteUuid:     quote.QuoteUUID,
			MenuItemUuid:  position.MenuItemUUID,
			GrossPrice:    position.GrossPrice,
			Quantity:      int32(position.Quantity),
		})
	}
	return quoteItems
}

func (r *OrdersRepository) getMenuItems(
	ctx context.Context,
	queries *dbmodels.Queries,
	restaurantUUID app.RestaurantUUID,
	menuItemsUUIDs []common.UUID,
) (map[app.RestaurantMenuItemUUID]app.MenuItem, error) {
	dbMenuItems, err := queries.GetMenuItemsByUUIDs(ctx, dbmodels.GetMenuItemsByUUIDsParams{
		RestaurantUuid: restaurantUUID,
		Column2:        menuItemsUUIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get menu items: %w", err)
	}

	return appMenuItemsFromDbMenuItems(dbMenuItems), nil
}

func appMenuItemsFromDbMenuItems(dbMenuItems []dbmodels.OrdersRestaurantMenuItem) map[app.RestaurantMenuItemUUID]app.MenuItem {
	appMenuItems := make(map[app.RestaurantMenuItemUUID]app.MenuItem, len(dbMenuItems))

	for _, dbItemPosition := range dbMenuItems {
		appMenuItems[dbItemPosition.RestaurantMenuItemUuid] = app.MenuItem{
			MenuItemUUID: dbItemPosition.RestaurantMenuItemUuid,
			Name:         dbItemPosition.Name,
			Ordering:     dbItemPosition.Ordering,
			GrossPrice:   dbItemPosition.GrossPrice,
			IsArchived:   dbItemPosition.IsArchived,
		}
	}

	return appMenuItems
}
