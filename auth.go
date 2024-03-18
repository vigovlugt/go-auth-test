package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func getGoogleConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Endpoint:     google.Endpoint,
		RedirectURL:  "http://localhost:3000/login/google/callback",
		Scopes:       []string{"email", "profile"},
	}
}

func generateRandomState() (string, error) {
	b := make([]byte, 16)
	n, err := rand.Read(b)
	if n != len(b) || err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(b), nil
}

func handleAuthURL(providerName string, config *oauth2.Config) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			state, err := generateRandomState()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     providerName + "_oauth_state",
				Value:    state,
				Path:     "/",
				Secure:   os.Getenv("APP_ENV") != "development",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})

			url := config.AuthCodeURL(state)
			http.Redirect(w, r, url, http.StatusFound)
		},
	)
}

func handleAuthCallback(providerName string, config *oauth2.Config, db *sqlx.DB) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			state := r.URL.Query().Get("state")
			cookieState, err := r.Cookie(providerName + "_oauth_state")
			if err != nil {
				http.Error(w, "Invalid state", http.StatusBadRequest)
				return
			}

			if state != cookieState.Value {
				http.Error(w, "Invalid state", http.StatusBadRequest)
				return
			}

			code := r.URL.Query().Get("code")
			token, err := config.Exchange(r.Context(), code)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			client := config.Client(r.Context(), token)
			userInfoResponse, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if userInfoResponse.StatusCode != http.StatusOK {
				http.Error(w, "Failed to get user info", http.StatusInternalServerError)
				return
			}

			type UserInfo struct {
				Sub           string `json:"sub"`
				Name          string `json:"name"`
				Email         string `json:"email"`
				EmailVerified bool   `json:"email_verified"`
			}

			var userInfo UserInfo

			err = json.NewDecoder(userInfoResponse.Body).Decode(&userInfo)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

			// Get existing user by provider ID
			userId, err := getUserByProviderID(r.Context(), db, providerName, userInfo.Sub)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Sign up the user if they don't exist
			if userId == "" {
				userId, err = createUser(r.Context(), db, userInfo.Email, providerName, userInfo.Sub)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}

			// Create a session for the user
			sessionId, err := createSession(r.Context(), db, userId)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     "session",
				Value:    sessionId,
				Path:     "/",
				Secure:   os.Getenv("APP_ENV") != "development",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   60 * 60 * 24 * 7,
			})

			http.Redirect(w, r, "/", http.StatusFound)
		},
	)
}

func validateSession(db *sqlx.DB, r *http.Request) (string, error) {
	sessionIdCookie, err := r.Cookie("session")
	if err != nil {
		return "", errors.New("No session cookie")
	}
	sessionId := sessionIdCookie.Value

	session, err := getSession(r.Context(), db, sessionId)
	if err != nil {
		return "", err
	}

	log.Println(session.ExpiresAt)

	expiresAt, err := time.Parse(time.RFC3339, session.ExpiresAt)
	if err != nil {
		return "", err
	}

	if expiresAt.Before(time.Now()) {
		return "", errors.New("Session expired")
	}

	// If the session is more than halfway through its duration, update the expires_at time
	if expiresAt.Sub(time.Now()) < sessionDuration/2 {
		err = updateSessionExpiresAt(r.Context(), db, sessionId)
		if err != nil {
			return "", err
		}
	}

	return session.UserID, nil
}
