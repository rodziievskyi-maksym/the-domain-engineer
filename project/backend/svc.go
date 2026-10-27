package backend

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	echo "github.com/labstack/echo/v4"
	"golang.org/x/sync/errgroup"

	commonHTTP "eats/backend/common/http"
	"eats/backend/common/log"
	"eats/backend/common/module"
	"eats/backend/common/module/contracts"
	"eats/backend/delivery"
	"eats/backend/orders"
)

type Svc struct {
	echoRouter *echo.Echo

	modules []module.Module

	dbPgx *pgxpool.Pool

	shutdownTimeout time.Duration
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
		delivery.NewModule(),
	}

	for _, module := range modules {
		start := time.Now()

		if err := module.Init(ctx); err != nil {
			return Svc{}, fmt.Errorf("initializing module %s failed: %w", module.Name(), err)
		}

		if err := module.RegisterContracts(ctx, moduleContracts); err != nil {
			return Svc{}, fmt.Errorf("registering module %s failed: %w", module.Name(), err)
		}

		log.FromContext(ctx).With(
			"duration", time.Since(start),
			"module", module.Name(),
		).Debug("Initialized module")
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
		echoRouter:      e,
		modules:         modules,
		dbPgx:           dbPgx,
		shutdownTimeout: 10 * time.Second,
	}, nil
}

func (s *Svc) Run(ctx context.Context, port string) error {
	defer s.dbPgx.Close()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		err := s.echoRouter.Start(port)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("starting http server failed: %w", err)
		}

		return nil
	})

	g.Go(func() error {
		<-ctx.Done()

		slog.Info("Shutting down HTTP server")

		// ctx is already canceled, so the shutdown needs its own deadline.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()

		if err := s.echoRouter.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutting down http server failed: %w", err)
		}

		return nil
	})

	return g.Wait()
}
