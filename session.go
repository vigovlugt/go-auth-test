package main

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

func createSession(context context.Context, db *sqlx.DB, userId string) (string, error) {
	sessionId, err := gonanoid.New()
	if err != nil {
		return "", err
	}

	_, err = db.ExecContext(context, `
INSERT INTO user_session (id, expires_at, user_id)
VALUES ($1, $2, $3)
	`, sessionId, time.Now().Add(24*7*time.Hour), userId)
	if err != nil {
		return "", err
	}

	return sessionId, nil
}

func getSessionUserId(context context.Context, db *sqlx.DB, sessionId string) (string, error) {
	rows, err := db.QueryContext(context, `
SELECT user_id
FROM user_session
WHERE id = $1 AND expires_at > $2
	`, sessionId, time.Now())
	if err != nil {
		return "", err
	}

	userId := ""
	if !rows.Next() {
		return "", err
	}

	err = rows.Scan(&userId)
	if err != nil {
		return "", err
	}

	return userId, nil
}
