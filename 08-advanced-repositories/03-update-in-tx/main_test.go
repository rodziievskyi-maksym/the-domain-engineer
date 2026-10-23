// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReserveStock_decrements_the_stock_and_records_the_reservations(t *testing.T) {
	ctx := context.Background()
	db, repo := newRepository(t)

	keyboard := seedStockItem(t, db, "KEYB-01", 10)
	mouse := seedStockItem(t, db, "MOUS-01", 5)
	orderID := uuid.Must(uuid.NewV7())

	err := repo.ReserveStock(ctx, orderID, []ReservationItem{
		{ItemID: keyboard.ItemID, Quantity: 3},
		{ItemID: mouse.ItemID, Quantity: 5},
	})
	require.NoError(t, err)

	assertAvailable(t, db, keyboard.ItemID, 7)
	assertAvailable(t, db, mouse.ItemID, 0)

	expected := []ReservationRow{
		{OrderID: orderID, ItemID: keyboard.ItemID, Quantity: 3},
		{OrderID: orderID, ItemID: mouse.ItemID, Quantity: 5},
	}

	stored, err := getReservations(ctx, db, orderID)
	require.NoError(t, err)

	diff := cmp.Diff(expected, stored,
		cmpopts.IgnoreFields(ReservationRow{}, "ReservationID"),
		cmpopts.SortSlices(func(a, b ReservationRow) bool { return a.Quantity < b.Quantity }),
	)
	assert.Empty(t, diff, "stored reservations differ from the requested ones (-want +got):\n%s", diff)

	for _, reservation := range stored {
		assert.NotEqual(t, uuid.Nil, reservation.ReservationID, "every reservation needs its own reservation_id")
	}
}

func TestReserveStock_rolls_back_the_earlier_items_when_one_is_short(t *testing.T) {
	ctx := context.Background()
	db, repo := newRepository(t)

	keyboard := seedStockItem(t, db, "KEYB-01", 10)
	mouse := seedStockItem(t, db, "MOUS-01", 2)
	orderID := uuid.Must(uuid.NewV7())

	err := repo.ReserveStock(ctx, orderID, []ReservationItem{
		{ItemID: keyboard.ItemID, Quantity: 3},
		{ItemID: mouse.ItemID, Quantity: 5},
	})
	require.Error(t, err, "reserving more units than are available should fail")
	assert.ErrorIs(t, err, ErrNotEnoughStock, "the caller should be able to tell short stock from a database failure (when you add context, wrap it so errors.Is still finds it)")

	assertAvailable(t, db, keyboard.ItemID, 10, "the first item was already decremented when the second one failed: run every query on tx so the whole reservation rolls back")
	assertAvailable(t, db, mouse.ItemID, 2)

	stored, err := getReservations(ctx, db, orderID)
	require.NoError(t, err)
	assert.Empty(t, stored, "a reservation that fails halfway should leave no rows behind")
}

func TestReserveStock_rolls_back_when_an_item_does_not_exist(t *testing.T) {
	ctx := context.Background()
	db, repo := newRepository(t)

	keyboard := seedStockItem(t, db, "KEYB-01", 10)
	orderID := uuid.Must(uuid.NewV7())

	err := repo.ReserveStock(ctx, orderID, []ReservationItem{
		{ItemID: keyboard.ItemID, Quantity: 3},
		{ItemID: uuid.Must(uuid.NewV7()), Quantity: 1},
	})
	require.Error(t, err, "reserving an item that isn't in the warehouse should fail")
	assert.ErrorIs(t, err, pgx.ErrNoRows, "the read failed: return that error from the closure instead of swallowing it")

	assertAvailable(t, db, keyboard.ItemID, 10, "any error returned from the closure has to roll back the writes that came before it")

	stored, err := getReservations(ctx, db, orderID)
	require.NoError(t, err)
	assert.Empty(t, stored, "a reservation that fails halfway should leave no rows behind")
}

func newRepository(t *testing.T) (*pgxpool.Pool, *StockRepository) {
	t.Helper()

	postgresURL := os.Getenv("POSTGRES_URL")
	require.NotEmpty(t, postgresURL, "POSTGRES_URL must point at a running PostgreSQL")

	ctx := context.Background()

	db, err := pgxpool.New(ctx, postgresURL)
	require.NoError(t, err)
	t.Cleanup(db.Close)

	require.NoError(t, migrate(ctx, db))

	return db, NewStockRepository(db)
}

func seedStockItem(t *testing.T, db DBTX, sku string, available int) StockItem {
	t.Helper()

	item := StockItem{
		ItemID:    uuid.Must(uuid.NewV7()),
		SKU:       sku,
		Available: available,
	}
	require.NoError(t, addStockItem(context.Background(), db, item))

	return item
}

func assertAvailable(t *testing.T, db DBTX, itemID uuid.UUID, expected int, msgAndArgs ...any) {
	t.Helper()

	item, err := getStockItem(context.Background(), db, itemID)
	require.NoError(t, err)
	assert.Equal(t, expected, item.Available, msgAndArgs...)
}
