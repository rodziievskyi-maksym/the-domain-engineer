// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package main

import (
	"context"
	"errors"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateOrganization_passes_the_stored_organization_to_the_callback(t *testing.T) {
	ctx := context.Background()
	db, repo := newRepository(t)

	analytics := seedProduct(t, db, "Analytics")
	helpdesk := seedProduct(t, db, "Helpdesk")

	org := seedOrganization(t, db, "Acme")
	analyticsLicense := seedLicense(t, db, org.OrganizationID, analytics.ProductID, 2)
	helpdeskLicense := seedLicense(t, db, org.OrganizationID, helpdesk.ProductID, 5)

	expected := Organization{
		OrganizationID: org.OrganizationID,
		Name:           "Acme",
		Licenses: []License{
			{LicenseID: analyticsLicense.LicenseID, ProductID: analytics.ProductID, Seats: 2},
			{LicenseID: helpdeskLicense.LicenseID, ProductID: helpdesk.ProductID, Seats: 5},
		},
	}

	callbackCalls := 0
	_, err := repo.UpdateOrganization(ctx, org.OrganizationID, func(ctx context.Context, got *Organization) error {
		callbackCalls++

		diff := cmp.Diff(expected, *got)
		assert.Empty(t, diff, "the callback should receive the organization with every license stored for it (-want +got):\n%s", diff)

		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, 1, callbackCalls, "the callback should run exactly once")
}

func TestUpdateOrganization_persists_and_returns_what_the_callback_changed(t *testing.T) {
	ctx := context.Background()
	db, repo := newRepository(t)

	analytics := seedProduct(t, db, "Analytics")
	helpdesk := seedProduct(t, db, "Helpdesk")

	org := seedOrganization(t, db, "Acme")
	analyticsLicense := seedLicense(t, db, org.OrganizationID, analytics.ProductID, 2)
	helpdeskLicense := seedLicense(t, db, org.OrganizationID, helpdesk.ProductID, 5)

	updated, err := repo.UpdateOrganization(ctx, org.OrganizationID, func(ctx context.Context, org *Organization) error {
		org.Name = "Acme Corp"
		for i := range org.Licenses {
			org.Licenses[i].Seats += 3
		}

		return nil
	})
	require.NoError(t, err)

	expected := Organization{
		OrganizationID: org.OrganizationID,
		Name:           "Acme Corp",
		Licenses: []License{
			{LicenseID: analyticsLicense.LicenseID, ProductID: analytics.ProductID, Seats: 5},
			{LicenseID: helpdeskLicense.LicenseID, ProductID: helpdesk.ProductID, Seats: 8},
		},
	}

	diff := cmp.Diff(expected, updated)
	assert.Empty(t, diff, "UpdateOrganization returned a different organization than the callback left behind (if it's empty, check whether := inside the closure shadows the outer variable) (-want +got):\n%s", diff)

	diff = cmp.Diff(expected, readOrganization(t, db, org.OrganizationID))
	assert.Empty(t, diff, "the stored organization differs from what the callback left behind (-want +got):\n%s", diff)
}

func TestUpdateOrganization_returns_the_callback_error(t *testing.T) {
	ctx := context.Background()
	db, repo := newRepository(t)

	analytics := seedProduct(t, db, "Analytics")
	org := seedOrganization(t, db, "Acme")
	analyticsLicense := seedLicense(t, db, org.OrganizationID, analytics.ProductID, 2)

	errBillingDeclined := errors.New("billing declined the new seats")

	_, err := repo.UpdateOrganization(ctx, org.OrganizationID, func(ctx context.Context, org *Organization) error {
		org.Name = "Acme Corp"
		org.Licenses[0].Seats = 99

		return errBillingDeclined
	})
	require.Error(t, err, "UpdateOrganization should fail when the callback fails")
	assert.ErrorIs(t, err, errBillingDeclined, "the callback's error should come back from UpdateOrganization (when you add context, wrap it so errors.Is still finds it)")

	expected := Organization{
		OrganizationID: org.OrganizationID,
		Name:           "Acme",
		Licenses: []License{
			{LicenseID: analyticsLicense.LicenseID, ProductID: analytics.ProductID, Seats: 2},
		},
	}

	diff := cmp.Diff(expected, readOrganization(t, db, org.OrganizationID))
	assert.Empty(t, diff, "nothing should be stored when the callback fails (-want +got):\n%s", diff)
}

func TestUpdateOrganization_rolls_back_the_name_when_a_license_write_fails(t *testing.T) {
	ctx := context.Background()
	db, repo := newRepository(t)

	analytics := seedProduct(t, db, "Analytics")
	helpdesk := seedProduct(t, db, "Helpdesk")

	org := seedOrganization(t, db, "Acme")
	analyticsLicense := seedLicense(t, db, org.OrganizationID, analytics.ProductID, 2)
	helpdeskLicense := seedLicense(t, db, org.OrganizationID, helpdesk.ProductID, 5)

	_, err := repo.UpdateOrganization(ctx, org.OrganizationID, func(ctx context.Context, org *Organization) error {
		org.Name = "Acme Corp"
		org.Licenses[0].Seats = 4
		// The seats column is CHECK (seats > 0), so this write fails after the
		// name and the first license have already been updated.
		org.Licenses[1].Seats = 0

		return nil
	})
	require.Error(t, err, "a license with zero seats should fail to write")
	assert.NotContains(t, err.Error(), "commit unexpectedly resulted in rollback",
		"an error inside the closure was swallowed: PostgreSQL aborted the transaction, but the closure returned nil")

	expected := Organization{
		OrganizationID: org.OrganizationID,
		Name:           "Acme",
		Licenses: []License{
			{LicenseID: analyticsLicense.LicenseID, ProductID: analytics.ProductID, Seats: 2},
			{LicenseID: helpdeskLicense.LicenseID, ProductID: helpdesk.ProductID, Seats: 5},
		},
	}

	diff := cmp.Diff(expected, readOrganization(t, db, org.OrganizationID))
	assert.Empty(t, diff, "the name and the earlier license should roll back with the failed write: run every query on tx and return every error from the closure (-want +got):\n%s", diff)
}

func TestUpdateOrganization_two_concurrent_seat_increments_both_land(t *testing.T) {
	ctx := context.Background()
	db, repo := newRepository(t)

	analytics := seedProduct(t, db, "Analytics")
	org := seedOrganization(t, db, "Acme")
	analyticsLicense := seedLicense(t, db, org.OrganizationID, analytics.ProductID, 1)

	// Both callbacks read before either writes. Without the barrier the two
	// calls could run one after the other, and reading outside the transaction
	// would pass the test by accident.
	firstRead := make(chan struct{})
	secondRead := make(chan struct{})

	addOneSeat := func(readSignal chan<- struct{}, otherRead <-chan struct{}) error {
		var signalOnce sync.Once

		_, err := repo.UpdateOrganization(ctx, org.OrganizationID, func(ctx context.Context, org *Organization) error {
			signalOnce.Do(func() { close(readSignal) })
			waitFor(t, otherRead)

			// A fresh slice, so a retry counts from what the retry read instead
			// of incrementing the value this callback already incremented.
			licenses := slices.Clone(org.Licenses)
			licenses[0].Seats++
			org.Licenses = licenses

			return nil
		})

		return err
	}

	errs := make(chan error, 2)
	go func() { errs <- addOneSeat(firstRead, secondRead) }()
	go func() { errs <- addOneSeat(secondRead, firstRead) }()

	require.NoError(t, <-errs)
	require.NoError(t, <-errs)

	expected := Organization{
		OrganizationID: org.OrganizationID,
		Name:           "Acme",
		Licenses: []License{
			{LicenseID: analyticsLicense.LicenseID, ProductID: analytics.ProductID, Seats: 3},
		},
	}

	diff := cmp.Diff(expected, readOrganization(t, db, org.OrganizationID))
	assert.Empty(t, diff, "one increment was lost: both callers read 1 seat and wrote 2. Read inside the transaction, so the retry in UpdateInTx runs the read again on fresh data (-want +got):\n%s", diff)
}

func TestUpdateOrganization_three_concurrent_seat_increments_all_land(t *testing.T) {
	ctx := context.Background()
	db, repo := newRepository(t)

	analytics := seedProduct(t, db, "Analytics")
	org := seedOrganization(t, db, "Acme")
	analyticsLicense := seedLicense(t, db, org.OrganizationID, analytics.ProductID, 1)

	// Бар'єр синхронізації для 3 горутин:
	// Всі 3 горутини повинні зайти у свій callback і прочитати Seats = 1
	// ДО того, як будь-яка з них піде далі до UPDATE/COMMIT.
	const numGoroutines = 10

	var readBarrier sync.WaitGroup
	readBarrier.Add(numGoroutines)

	// Канал, який розблокує всі 3 горутини одночасно після того, як кожна засигналить у WaitGroup
	allRead := make(chan struct{})

	addOneSeat := func() error {
		var signalOnce sync.Once

		_, err := repo.UpdateOrganization(ctx, org.OrganizationID, func(ctx context.Context, org *Organization) error {
			// Сигналізуємо, що ПОТОЧНА горутина прочитала дані.
			// sync.Once гарантує, що при ретраї (Retry) signalOnce більше не буде кликати Done()
			signalOnce.Do(func() {
				readBarrier.Done()
			})

			// Чекаємо, поки ВСІ 3 горутини дочитають свої початкові дані (Seats = 1)
			waitFor(t, allRead)

			licenses := slices.Clone(org.Licenses)
			//org.Licenses[0].Seats++
			licenses[0].Seats++
			org.Licenses = licenses

			return nil
		})

		return err
	}

	// Горутина-координатор: чекає, поки всі 3 засигналять у readBarrier,
	// і відкриває allRead для всіх 3 горутин одночасно.
	go func() {
		readBarrier.Wait()
		close(allRead)
	}()

	errs := make(chan error, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			errs <- addOneSeat()
		}()
	}

	// Чекаємо завершення виконання всіх 3 горутин
	for i := 0; i < numGoroutines; i++ {
		require.NoError(t, <-errs)
	}

	// Початкове = 1, додаємо 3 інкременти -> очікуємо 4
	expected := Organization{
		OrganizationID: org.OrganizationID,
		Name:           "Acme",
		Licenses: []License{
			{LicenseID: analyticsLicense.LicenseID, ProductID: analytics.ProductID, Seats: 11},
		},
	}

	diff := cmp.Diff(expected, readOrganization(t, db, org.OrganizationID))
	assert.Empty(t, diff, "increments were lost: all callers read 1 seat concurrently. Read inside the transaction, so retry in UpdateInTx runs read again (-want +got):\n%s", diff)
}

func newRepository(t *testing.T) (*pgxpool.Pool, *OrganizationsRepository) {
	t.Helper()

	postgresURL := os.Getenv("POSTGRES_URL_2")
	require.NotEmpty(t, postgresURL, "POSTGRES_URL must point at a running PostgreSQL")

	ctx := context.Background()

	db, err := pgxpool.New(ctx, postgresURL)
	require.NoError(t, err)
	t.Cleanup(db.Close)

	require.NoError(t, migrate(ctx, db))

	return db, NewOrganizationsRepository(db)
}

func seedProduct(t *testing.T, db DBTX, name string) Product {
	t.Helper()

	product := Product{
		ProductID: uuid.Must(uuid.NewV7()),
		Name:      name,
	}
	require.NoError(t, addProduct(context.Background(), db, product))

	return product
}

func seedOrganization(t *testing.T, db DBTX, name string) OrganizationRow {
	t.Helper()

	row := OrganizationRow{
		OrganizationID: uuid.Must(uuid.NewV7()),
		Name:           name,
	}
	require.NoError(t, addOrganization(context.Background(), db, row))

	return row
}

func seedLicense(t *testing.T, db DBTX, orgID, productID uuid.UUID, seats int) LicenseRow {
	t.Helper()

	row := LicenseRow{
		LicenseID:      uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ProductID:      productID,
		Seats:          seats,
	}
	require.NoError(t, addLicense(context.Background(), db, row))

	return row
}

func readOrganization(t *testing.T, db DBTX, orgID uuid.UUID) Organization {
	t.Helper()

	ctx := context.Background()

	orgRow, err := getOrganization(ctx, db, orgID)
	require.NoError(t, err)

	licenseRows, err := getLicenses(ctx, db, orgID)
	require.NoError(t, err)

	licenses := make([]License, 0, len(licenseRows))
	for _, row := range licenseRows {
		licenses = append(licenses, License{
			LicenseID: row.LicenseID,
			ProductID: row.ProductID,
			Seats:     row.Seats,
		})
	}

	return Organization{
		OrganizationID: orgRow.OrganizationID,
		Name:           orgRow.Name,
		Licenses:       licenses,
	}
}

func waitFor(t *testing.T, ch <-chan struct{}) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(10 * time.Second):
		t.Error("timed out waiting for the other transaction to read: does UpdateOrganization call updateFn?")
	}
}
