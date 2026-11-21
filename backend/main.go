package main

import (
	"context"
	"crypto/rand"
	"log"
	"net/http"
	"os"

	"dhwani/api"
	"dhwani/metadata"
)

func main() {
	var st metadata.Store
	if url := os.Getenv("DATABASE_URL"); url != "" {
		pg, err := metadata.NewPostgres(context.Background(), url)
		if err != nil {
			log.Fatalf("database: %v", err)
		}
		st = pg
	} else {
		log.Println("DATABASE_URL not set: using in-memory store (data is lost on exit)")
		st = metadata.NewMemory()
	}

	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) < 16 {
		secret = make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			log.Fatal(err)
		}
		log.Println("JWT_SECRET not set (needs 16+ chars): using a random key; sign-ins end when the server restarts")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Dhvani API on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, (&api.Server{Store: st, Secret: secret}).Routes()))
}
