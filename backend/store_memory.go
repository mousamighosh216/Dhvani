//go:build nodeps

package main

import (
	"context"
	"log"

	"dhwani/metadata"
	"dhwani/recognition/matcher"
)

// openStores (nodeps build) keeps everything in RAM; data is lost on exit.
func openStores(context.Context) (metadata.Service, matcher.FingerprintStore, func(), error) {
	m := metadata.NewMemoryStore()
	log.Print("WARNING: nodeps build, using the in-memory store (not PostgreSQL)")
	return m, m, func() {}, nil
}
