package main

import (
	"context"
	"fmt"
)

const schema = `
CREATE TABLE IF NOT EXISTS products (
	product_id UUID PRIMARY KEY,
	name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS organizations (
	organization_id UUID PRIMARY KEY,
	name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS licenses (
	license_id UUID PRIMARY KEY,
	organization_id UUID NOT NULL REFERENCES organizations (organization_id),
	product_id UUID NOT NULL REFERENCES products (product_id),
	seats INT NOT NULL CHECK (seats > 0)
);
`

func migrate(ctx context.Context, db DBTX) error {
	if _, err := db.Exec(ctx, schema); err != nil {
		return fmt.Errorf("failed to create the schema: %w", err)
	}

	return nil
}
