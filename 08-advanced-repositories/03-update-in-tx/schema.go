package main

import (
	"context"
	"fmt"
)

const schema = `
CREATE TABLE IF NOT EXISTS stock_items (
	item_id UUID PRIMARY KEY,
	sku TEXT NOT NULL,
	available INT NOT NULL CHECK (available >= 0)
);

CREATE TABLE IF NOT EXISTS reservations (
	reservation_id UUID PRIMARY KEY,
	order_id UUID NOT NULL,
	item_id UUID NOT NULL REFERENCES stock_items (item_id),
	quantity INT NOT NULL CHECK (quantity > 0)
);
`

func migrate(ctx context.Context, db DBTX) error {
	if _, err := db.Exec(ctx, schema); err != nil {
		return fmt.Errorf("failed to create the schema: %w", err)
	}

	return nil
}
