// Command dhwani runs the Dhwani music recognition server.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dhwani/api"
	"dhwani/recognition"
)

func main() {
	addr := flag.String("addr", envOr("DHWANI_ADDR", ":8080"), "listen address")
	static := flag.String("static", envOr("DHWANI_STATIC", "../frontend/dist"), "directory with the built frontend (optional)")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	svc, fp, closeStore, err := openStores(ctx)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer closeStore()

	engine := recognition.NewEngine(fp)
	srv := &http.Server{
		Addr:              *addr,
		Handler:           api.NewRouter(engine, svc, *static),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdown)
	}()
	log.Printf("Dhwani listening on %s", *addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
