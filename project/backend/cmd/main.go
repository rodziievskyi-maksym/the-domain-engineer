package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"eats/backend"
	"eats/backend/common/log"
)

func main() {
	ctx := context.Background()

	log.Init(slog.LevelInfo)

	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		panic("POSTGRES_URL environment variable is not set")
	}

	dbPgx, err := pgxpool.New(ctx, dsn)
	if err != nil {
		panic(err)
	}

	svc, err := backend.New(
		ctx,
		dbPgx,
	)
	if err != nil {
		panic(err)
	}

	if err := svc.Run(":8080"); err != nil {
		panic(err)
	}
}
