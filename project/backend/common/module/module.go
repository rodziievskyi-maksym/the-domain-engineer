package module

import (
	"context"

	"eats/backend/common/module/contracts"
)

type Name string

type Module interface {
	Name() Name
	Init(ctx context.Context) error
	RegisterContracts(ctx context.Context, contracts *contracts.Contracts) error
}
