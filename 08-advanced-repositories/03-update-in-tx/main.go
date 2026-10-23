package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotEnoughStock = errors.New("not enough stock")

type StockItem struct {
	ItemID    uuid.UUID
	SKU       string
	Available int
}

type ReservationItem struct {
	ItemID   uuid.UUID
	Quantity int
}

type StockRepository struct {
	db *pgxpool.Pool
}

func NewStockRepository(db *pgxpool.Pool) *StockRepository {
	if db == nil {
		panic("db connection pool cannot be nil")
	}

	return &StockRepository{db: db}
}

func (r *StockRepository) ReserveStock(
	ctx context.Context,
	orderID uuid.UUID,
	items []ReservationItem,
) error {
	return UpdateInTx(ctx, r.db, func(ctx context.Context, tx pgx.Tx) error {
		for _, item := range items {
			stock, err := getStockItem(ctx, tx, item.ItemID)
			if err != nil {
				return fmt.Errorf("failed to get stock item %s: %w", item.ItemID, err)
			}

			if item.Quantity > stock.Available {
				return fmt.Errorf("%w: %s has %d left, %d requested", ErrNotEnoughStock, stock.SKU, stock.Available, item.Quantity)
			}

			err = setStockAvailable(ctx, tx, item.ItemID, stock.Available-item.Quantity)
			if err != nil {
				return fmt.Errorf("failed to update stock item %s: %w", item.ItemID, err)
			}

			reservation := ReservationRow{
				ReservationID: uuid.Must(uuid.NewV7()),
				OrderID:       orderID,
				ItemID:        item.ItemID,
				Quantity:      item.Quantity,
			}

			err = addReservation(ctx, tx, reservation)
			if err != nil {
				return fmt.Errorf("failed to add reservation for item %s: %w", item.ItemID, err)
			}
		}

		return nil
	})
}
