package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"eats/backend"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		panic("POSTGRES_URL environment variable is not set")
	}

	dbPgx, err := pgxpool.New(ctx, dsn)
	if err != nil {
		panic(err)
	}

	_, err = backend.New(
		ctx,
		dbPgx,
	)
	if err != nil {
		panic(err)
	}

	fmt.Println("Service initialized")
}
