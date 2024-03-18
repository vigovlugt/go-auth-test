package main

import (
	"context"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func newDb(ctx context.Context) (*sqlx.DB, error) {
	db, err := sqlx.ConnectContext(ctx, "sqlite3", "db.sqlite3")
	if err != nil {
		return nil, err
	}

	SetupDb(ctx, db)

	return db, nil
}

func SetupDb(ctx context.Context, db *sqlx.DB) {
	db.MustExecContext(ctx, `
CREATE TABLE IF NOT EXISTS user (
	id TEXT PRIMARY KEY,
	email TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS user_session (
	id TEXT PRIMARY KEY,
	expires_at TIMESTAMPTZ NOT NULL,
	user_id TEXT NOT NULL REFERENCES user(id)
);

CREATE TABLE IF NOT EXISTS oauth_account (
	provider_id TEXT NOT NULL,
	provider_user_id TEXT NOT NULL,
	user_id TEXT NOT NULL,
	PRIMARY KEY (provider_id, provider_user_id),
	FOREIGN KEY (user_id) REFERENCES user(id)
)
	`)

}
