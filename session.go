package main

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

const sessionDuration = time.Minute //24 * 7 * time.Hour

func createSession(context context.Context, db *sqlx.DB, userId string) (string, error) {
	sessionId, err := gonanoid.New()
	if err != nil {
		return "", err
	}

	_, err = db.ExecContext(context, `
INSERT INTO user_session (id, expires_at, user_id)
VALUES ($1, $2, $3)
	`, sessionId, time.Now().Add(sessionDuration).Format(time.RFC3339), userId)
	if err != nil {
		return "", err
	}

	return sessionId, nil
}

type Session struct {
	ID        string `db:"id"`
	ExpiresAt string `db:"expires_at"`
	UserID    string `db:"user_id"`
}

func getSession(context context.Context, db *sqlx.DB, sessionId string) (Session, error) {
	var session Session
	err := db.GetContext(context, &session, `
SELECT *
FROM user_session
WHERE id = $1
	`, sessionId, time.Now())
	if err != nil {
		return Session{}, err
	}

	return session, nil
}

func updateSessionExpiresAt(context context.Context, db *sqlx.DB, sessionId string) error {
	_, err := db.ExecContext(context, `
UPDATE user_session
SET expires_at = $1
WHERE id = $2
	`, time.Now().Add(sessionDuration).Format(time.RFC3339), sessionId)
	if err != nil {
		return err
	}

	return nil
}
