package backend

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	echo "github.com/labstack/echo/v4"

	commonHTTP "eats/backend/common/http"
	"eats/backend/common/module"
	"eats/backend/common/module/contracts"
	"eats/backend/orders"
)

type Svc struct {
	echoRouter *echo.Echo

	modules []module.Module

	dbPgx *pgxpool.Pool
}

func New(
	ctx context.Context,
	dbPgx *pgxpool.Pool,
) (Svc, error) {
	e := commonHTTP.NewEcho()

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

	for _, module := range modules {
		err := module.RegisterHttp(ctx, e)
		if err != nil {
			return Svc{}, fmt.Errorf("registering http for module %s failed: %w", module.Name(), err)
		}
	}

	return Svc{
		echoRouter: e,
		modules:    modules,
		dbPgx:      dbPgx,
	}, nil
}

func (s *Svc) Run(port string) error {
	defer s.dbPgx.Close()

	err := s.echoRouter.Start(port)
	if err != nil {
		return fmt.Errorf("starting http server failed: %w", err)
	}

	return nil
}
