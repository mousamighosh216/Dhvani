//go:build !nodeps

package main

import (
	"context"
	"log"

	"dhwani/metadata"
	"dhwani/recognition/matcher"
)

// openStores connects to PostgreSQL (DATABASE_URL).
func openStores(ctx context.Context) (metadata.Service, matcher.FingerprintStore, func(), error) {
	dsn := envOr("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/dhwani?sslmode=disable")
	pg, err := metadata.NewPostgresStore(ctx, dsn)
	if err != nil {
		return nil, nil, nil, err
	}
	log.Print("using PostgreSQL")
	return pg, pg, pg.Close, nil
}
