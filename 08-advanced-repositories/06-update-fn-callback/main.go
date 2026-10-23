package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Product struct {
	ProductID uuid.UUID
	Name      string
}

type License struct {
	LicenseID uuid.UUID
	ProductID uuid.UUID
	Seats     int
}

type Organization struct {
	OrganizationID uuid.UUID
	Name           string
	Licenses       []License
}

var ErrSeatCapExceeded = errors.New("seat cap exceeded")

const (
	seatCap         = 5
	uncappedProduct = "Analytics"
)

type OrganizationsRepository struct {
	db *pgxpool.Pool
}

func NewOrganizationsRepository(db *pgxpool.Pool) *OrganizationsRepository {
	if db == nil {
		panic("db connection pool cannot be nil")
	}

	return &OrganizationsRepository{db: db}
}

func (r *OrganizationsRepository) AddSeats(
	ctx context.Context,
	orgID uuid.UUID,
	productID uuid.UUID,
	count int,
) error {
	return UpdateInTx(ctx, r.db, func(ctx context.Context, tx pgx.Tx) error {
		return addSeats(ctx, tx, orgID, productID, count)
	})
}

func (r *OrganizationsRepository) RenameAndAddSeats(
	ctx context.Context,
	orgID uuid.UUID,
	name string,
	productID uuid.UUID,
	count int,
) error {
	return UpdateInTx(ctx, r.db, func(ctx context.Context, tx pgx.Tx) error {
		if err := setOrganizationName(ctx, tx, orgID, name); err != nil {
			return fmt.Errorf("failed to rename organization %s: %w", orgID, err)
		}

		return addSeats(ctx, tx, orgID, productID, count)
	})
}

func addSeats(ctx context.Context, tx pgx.Tx, orgID, productID uuid.UUID, count int) error {
	licenses, err := getLicenses(ctx, tx, orgID)
	if err != nil {
		return fmt.Errorf("failed to get licenses of organization %s: %w", orgID, err)
	}

	for _, license := range licenses {
		if license.ProductID != productID {
			continue
		}

		product, err := getProduct(ctx, tx, productID)
		if err != nil {
			return fmt.Errorf("failed to get product %s: %w", productID, err)
		}

		if product.Name != uncappedProduct && license.Seats+count > seatCap {
			return fmt.Errorf("%w: %s allows at most %d seats", ErrSeatCapExceeded, product.Name, seatCap)
		}

		return setLicenseSeats(ctx, tx, license.LicenseID, license.Seats+count)
	}

	return fmt.Errorf("organization %s has no license for product %s", orgID, productID)
}

func (r *OrganizationsRepository) UpdateOrganization(
	ctx context.Context,
	orgID uuid.UUID,
	updateFn func(ctx context.Context, org *Organization) error,
) (Organization, error) {
	err := UpdateInTx(ctx, r.db, func(ctx context.Context, tx pgx.Tx) error {
		// TODO: load the organization with its licenses, hand them to updateFn,
		// and persist what it leaves behind.
		return nil
	})
	if err != nil {
		return Organization{}, err
	}

	return Organization{}, nil
}
