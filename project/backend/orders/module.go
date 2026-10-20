package orders

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"eats/backend/common/module"
	"eats/backend/common/module/contracts"
	ordersModule "eats/backend/orders/api/module"
)

type Module struct {
	pgxDb *pgxpool.Pool

	modules *contracts.Contracts
}

func NewModule(pgxDb *pgxpool.Pool, modules *contracts.Contracts) *Module {
	return &Module{
		pgxDb:   pgxDb,
		modules: modules,
	}
}

func (m *Module) Name() module.Name {
	return "orders"
}

func (m *Module) Init(ctx context.Context) error {
	return nil
}

func (m *Module) RegisterContracts(ctx context.Context, contracts *contracts.Contracts) error {
	contracts.Orders = ordersModule.Orders{}
	return nil
}
