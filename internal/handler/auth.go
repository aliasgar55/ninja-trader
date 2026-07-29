package handler

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"ninja-trader/internal/kite"
	models "ninja-trader/internal/model"
	"time"

	"gorm.io/gorm"
)

type AuthHandler struct {
	KiteClient *kite.Client
	APISecret  string
	Db         *gorm.DB

	userID      string
	tokenExpiry time.Time
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

	h.userID = session.UserID

  expiry := nextDaySixAMIST()
	h.tokenExpiry = expiry

	encToken, err := encrypt(session.AccessToken, h.APISecret)
	if err != nil {
		log.Printf("Failed to encrypt access token: %v", err)
	} else {
		h.setSetting("kite_access_token", encToken)
		h.setSetting("kite_user_id", session.UserID)
		h.setSetting("kite_token_expiry", expiry.Format(time.RFC3339))
		log.Printf("Kite session saved to DB — expires %s", expiry.Format(time.RFC3339))
	}

	log.Printf("Kite login successful — user: %s", session.UserID)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	uid := h.userID
	loggedIn := uid != ""

	if loggedIn && !h.tokenExpiry.IsZero() && time.Now().After(h.tokenExpiry) {
		loggedIn = false
		h.userID = ""
		h.KiteClient.SetAccessToken("")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"logged_in": loggedIn,
		"user_id":   uid,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	h.userID = ""
	h.tokenExpiry = time.Time{}
	h.KiteClient.SetAccessToken("")
	h.setSetting("kite_access_token", "")
	h.setSetting("kite_user_id", "")
	h.setSetting("kite_token_expiry", "")
	log.Println("Kite session logged out")
	http.Redirect(w, r, "/admin?msg=kite_logged_out", http.StatusSeeOther)
}

func (h *AuthHandler) LoadSession() {
	if h.Db == nil {
		return
	}

	expiryStr := h.getSetting("kite_token_expiry")
	if expiryStr == "" {
		return
	}

	expiry, err := time.Parse(time.RFC3339, expiryStr)
	if err != nil {
		log.Printf("Invalid token expiry in DB: %v", err)
		return
	}

	if time.Now().After(expiry) {
		log.Println("Saved Kite token has expired, skipping restore")
		return
	}

	encToken := h.getSetting("kite_access_token")
	if encToken == "" {
		return
	}

	token, err := decrypt(encToken, h.APISecret)
	if err != nil {
		log.Printf("Failed to decrypt access token: %v", err)
		return
	}

	userID := h.getSetting("kite_user_id")

  h.KiteClient.SetAccessToken(token)
	h.userID = userID
	h.tokenExpiry = expiry

	log.Printf("Restored Kite session from DB — user: %s, expires: %s", userID, expiry.Format(time.RFC3339))
}

func (h *AuthHandler) setSetting(key, value string) {
	var setting models.AppSetting
	result := h.Db.Where("key = ?", key).First(&setting)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			h.Db.Create(&models.AppSetting{Key: key, Value: value})
		} else {
			log.Printf("DB error reading setting %s: %v", key, result.Error)
		}
		return
	}
	h.Db.Model(&setting).Update("value", value)
}

func (h *AuthHandler) getSetting(key string) string {
	var setting models.AppSetting
	if err := h.Db.Where("key = ?", key).First(&setting).Error; err != nil {
		return ""
	}
	return setting.Value
}

func deriveKey(apiSecret string) []byte {
	hash := sha256.Sum256([]byte(apiSecret))
	return hash[:]
}

func encrypt(plaintext, apiSecret string) (string, error) {
	key := deriveKey(apiSecret)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes.NewCipher: %w", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("cipher.NewGCM: %w", err)
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce generation: %w", err)
	}
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

func decrypt(ciphertextHex, apiSecret string) (string, error) {
	key := deriveKey(apiSecret)
	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", fmt.Errorf("hex decode: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes.NewCipher: %w", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("cipher.NewGCM: %w", err)
	}
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("aesGCM.Open: %w", err)
	}
	return string(plaintext), nil
}

func nextDaySixAMIST() time.Time {
	ist, _ := time.LoadLocation("Asia/Kolkata")
	now := time.Now().In(ist)
	return time.Date(now.Year(), now.Month(), now.Day()+1, 6, 0, 0, 0, ist)
}
