package backend

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"eats/backend/common/module"
	"eats/backend/common/module/contracts"
	"eats/backend/orders"
)

type Svc struct {
	modules []module.Module

	dbPgx *pgxpool.Pool
}

func New(
	ctx context.Context,
	dbPgx *pgxpool.Pool,
) (Svc, error) {
	// We use a pointer here so modules can register their contracts during Init(),
	// then all modules can call each other after initialization completes.
	moduleContracts := &contracts.Contracts{}

	modules := []module.Module{
		orders.NewModule(dbPgx, moduleContracts),
	}

	for _, module := range modules {
		if err := module.Init(ctx); err != nil {
			return Svc{}, fmt.Errorf("initializing module %s failed: %w", module.Name(), err)
		}

		if err := module.RegisterContracts(ctx, moduleContracts); err != nil {
			return Svc{}, fmt.Errorf("registering module %s failed: %w", module.Name(), err)
		}
	}

	if err := moduleContracts.Verify(); err != nil {
		return Svc{}, fmt.Errorf("verifying module contracts failed: %w", err)
	}

	return Svc{
		modules: modules,
		dbPgx:   dbPgx,
	}, nil
}
