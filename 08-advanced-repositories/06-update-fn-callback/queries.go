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

type OrganizationRow struct {
	OrganizationID uuid.UUID
	Name           string
}

type LicenseRow struct {
	LicenseID      uuid.UUID
	OrganizationID uuid.UUID
	ProductID      uuid.UUID
	Seats          int
}

func addProduct(ctx context.Context, db DBTX, product Product) error {
	_, err := db.Exec(ctx,
		`INSERT INTO products (product_id, name) VALUES ($1, $2)`,
		product.ProductID, product.Name,
	)

	return err
}

func addOrganization(ctx context.Context, db DBTX, row OrganizationRow) error {
	_, err := db.Exec(ctx,
		`INSERT INTO organizations (organization_id, name) VALUES ($1, $2)`,
		row.OrganizationID, row.Name,
	)

	return err
}

func addLicense(ctx context.Context, db DBTX, row LicenseRow) error {
	_, err := db.Exec(ctx,
		`INSERT INTO licenses (license_id, organization_id, product_id, seats) VALUES ($1, $2, $3, $4)`,
		row.LicenseID, row.OrganizationID, row.ProductID, row.Seats,
	)

	return err
}

func getProduct(ctx context.Context, db DBTX, productID uuid.UUID) (Product, error) {
	var product Product

	err := db.QueryRow(ctx,
		`SELECT product_id, name FROM products WHERE product_id = $1`,
		productID,
	).Scan(&product.ProductID, &product.Name)
	if err != nil {
		return Product{}, err
	}

	return product, nil
}

func getOrganization(ctx context.Context, db DBTX, orgID uuid.UUID) (OrganizationRow, error) {
	var row OrganizationRow

	err := db.QueryRow(ctx,
		`SELECT organization_id, name FROM organizations WHERE organization_id = $1`,
		orgID,
	).Scan(&row.OrganizationID, &row.Name)
	if err != nil {
		return OrganizationRow{}, err
	}

	return row, nil
}

func getLicenses(ctx context.Context, db DBTX, orgID uuid.UUID) ([]LicenseRow, error) {
	rows, err := db.Query(ctx,
		`SELECT license_id, organization_id, product_id, seats FROM licenses WHERE organization_id = $1 ORDER BY license_id`,
		orgID,
	)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, pgx.RowToStructByPos[LicenseRow])
}

func setOrganizationName(ctx context.Context, db DBTX, orgID uuid.UUID, name string) error {
	_, err := db.Exec(ctx,
		`UPDATE organizations SET name = $2 WHERE organization_id = $1`,
		orgID, name,
	)

	return err
}

func setLicenseSeats(ctx context.Context, db DBTX, licenseID uuid.UUID, seats int) error {
	_, err := db.Exec(ctx,
		`UPDATE licenses SET seats = $2 WHERE license_id = $1`,
		licenseID, seats,
	)

	return err
}
