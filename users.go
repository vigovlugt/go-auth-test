package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jmoiron/sqlx"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type User struct {
	ID    string `db:"id" json:"id"`
	Email string `db:"email" json:"email"`
}

func handleUserMe(db *sqlx.DB) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userId, err := validateSession(db, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		user, err := getUser(db, userId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

func getUserByProviderID(context context.Context, db *sqlx.DB, provider, providerUserID string) (string, error) {
	rows, err := db.QueryContext(context, `
SELECT user_id
FROM oauth_account
WHERE provider_id = $1 AND provider_user_id = $2
	`, provider, providerUserID)
	if err != nil {
		return "", err
	}

	userId := ""
	if !rows.Next() {
		return "", nil
	}
	err = rows.Scan(&userId)
	if err != nil {
		return "", err
	}

	return userId, nil
}

func createUser(context context.Context, db *sqlx.DB, email string, provider, providerUserID string) (string, error) {
	userId, err := gonanoid.New()
	if err != nil {
		return "", err
	}

	tx, err := db.BeginTx(context, nil)
	if err != nil {
		return "", err
	}

	_, err = tx.ExecContext(context, `
INSERT INTO user (id, email)
VALUES ($1, $2)
	`, userId, email)
	if err != nil {
		return "", err
	}

	_, err = tx.ExecContext(context, `
INSERT INTO oauth_account (provider_id, provider_user_id, user_id)
VALUES ($1, $2, $3)
	`, provider, providerUserID, userId)
	if err != nil {
		return "", err
	}

	err = tx.Commit()
	if err != nil {
		return "", err
	}

	return userId, nil
}

func getUser(db *sqlx.DB, id string) (User, error) {
	var user User

	err := db.Get(&user, "SELECT * FROM user WHERE id = $1", id)
	if err != nil {
		return User{}, err
	}

	return user, nil
}
