package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"ninja-trader/internal/kite"
	"sync"
)

type AuthHandler struct {
	KiteClient *kite.Client
	APISecret  string

	mu     sync.RWMutex
	userID string
}

func (h *AuthHandler) IsLoggedIn() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.userID != ""
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	loginURL := h.KiteClient.GetLoginURL()
	http.Redirect(w, r, loginURL, http.StatusFound)
}

func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status != "success" {
		http.Error(w, "Login failed or was cancelled", http.StatusUnauthorized)
		return
	}

	requestToken := r.URL.Query().Get("request_token")
	if requestToken == "" {
		http.Error(w, "Missing request_token", http.StatusBadRequest)
		return
	}

	session, err := h.KiteClient.GenerateSession(requestToken, h.APISecret)
	if err != nil {
		log.Printf("Kite GenerateSession error: %v", err)
		http.Error(w, "Failed to generate session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.KiteClient.SetAccessToken(session.AccessToken)

	h.mu.Lock()
	h.userID = session.UserID
	h.mu.Unlock()

	log.Printf("Kite login successful — user: %s", session.UserID)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	uid := h.userID
	h.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logged_in": uid != "",
		"user_id":   uid,
	})
}
