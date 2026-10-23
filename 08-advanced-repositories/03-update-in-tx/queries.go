package main

import (
	"context"

	"github.com/google/uuid"
	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX is satisfied by both *pgxpool.Pool and pgx.Tx.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type ReservationRow struct {
	ReservationID uuid.UUID
	OrderID       uuid.UUID
	ItemID        uuid.UUID
	Quantity      int
}

func addStockItem(ctx context.Context, db DBTX, item StockItem) error {
	_, err := db.Exec(ctx,
		`INSERT INTO stock_items (item_id, sku, available) VALUES ($1, $2, $3)`,
		item.ItemID, item.SKU, item.Available,
	)

	return err
}

func getStockItem(ctx context.Context, db DBTX, itemID uuid.UUID) (StockItem, error) {
	var item StockItem

	err := db.QueryRow(ctx,
		`SELECT item_id, sku, available FROM stock_items WHERE item_id = $1`,
		itemID,
	).Scan(&item.ItemID, &item.SKU, &item.Available)
	if err != nil {
		return StockItem{}, err
	}

	return item, nil
}

func setStockAvailable(ctx context.Context, db DBTX, itemID uuid.UUID, available int) error {
	_, err := db.Exec(ctx,
		`UPDATE stock_items SET available = $2 WHERE item_id = $1`,
		itemID, available,
	)

	return err
}

func addReservation(ctx context.Context, db DBTX, row ReservationRow) error {
	_, err := db.Exec(ctx,
		`INSERT INTO reservations (reservation_id, order_id, item_id, quantity) VALUES ($1, $2, $3, $4)`,
		row.ReservationID, row.OrderID, row.ItemID, row.Quantity,
	)

	return err
}

func getReservations(ctx context.Context, db DBTX, orderID uuid.UUID) ([]ReservationRow, error) {
	rows, err := db.Query(ctx,
		`SELECT reservation_id, order_id, item_id, quantity FROM reservations WHERE order_id = $1`,
		orderID,
	)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, pgx.RowToStructByPos[ReservationRow])
}
