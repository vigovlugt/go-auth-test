package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

func newServer(
	googleConfig *oauth2.Config,
	db *sqlx.DB,
) http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux, db, googleConfig)

	return mux
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db, err := newDb(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	googleConfig := getGoogleConfig()

	srv := newServer(googleConfig, db)

	port := "3000"
	if v := os.Getenv("PORT"); v != "" {
		port = v
	}
	httpServer := &http.Server{
		Addr:    net.JoinHostPort("0.0.0.0", port),
		Handler: srv,
	}

	log.Printf("Starting server on http://localhost:%s", port)
	err = httpServer.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
