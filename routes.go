package main

import (
	"net/http"

	"github.com/jmoiron/sqlx"
	"golang.org/x/oauth2"
)

func addRoutes(mux *http.ServeMux, db *sqlx.DB, googleConfig *oauth2.Config) {
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, world!"))
	}))
	mux.Handle("/login/google", handleAuthURL("google", googleConfig))
	mux.Handle("/login/google/callback", handleAuthCallback("google", googleConfig, db))
	mux.Handle("/users/me", handleUserMe(db))
}
