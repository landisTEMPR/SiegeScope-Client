package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const API_URL = "https://siegescopestats.vercel.app/api/verify-subscriptions" // Update this!

type VerifyRequest struct {
	UserID string `json:"userId"`
	APIKey string `json:"apiKey"`
}

type VerifyResponse struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

type StoredAuth struct {
	APIKey string `json:"apiKey"`
	UserID string `json:"userId"`
	Valid  bool   `json:"valid"`
}

func GetAuthDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".siegescope")
}

func GetAuthFilePath() string {
	return filepath.Join(GetAuthDir(), "auth.json")
}

func SaveAuth(auth *StoredAuth) error {
	dir := GetAuthDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.Marshal(auth)
	if err != nil {
		return err
	}

	return os.WriteFile(GetAuthFilePath(), data, 0600)
}

func LoadAuth() (*StoredAuth, error) {
	data, err := os.ReadFile(GetAuthFilePath())
	if err != nil {
		return nil, err
	}

	var auth StoredAuth
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, err
	}

	return &auth, nil
}

func ClearAuth() error {
	return os.Remove(GetAuthFilePath())
}

func DecodeAPIKey(apiKey string) (string, error) {
	if !strings.HasPrefix(apiKey, "ss_") {
		return "", fmt.Errorf("invalid API key format")
	}

	encoded := strings.TrimPrefix(apiKey, "ss_")

	// Add padding
	for len(encoded)%4 != 0 {
		encoded += "="
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	return string(decoded), nil
}

func VerifySubscription(apiKey string) (*VerifyResponse, error) {
	userID, err := DecodeAPIKey(apiKey)
	if err != nil {
		return nil, err
	}

	reqBody := VerifyRequest{
		UserID: userID,
		APIKey: apiKey,
	}

	jsonData, _ := json.Marshal(reqBody)

	resp, err := http.Post(API_URL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("could not connect to server: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var verifyResp VerifyResponse
	if err := json.Unmarshal(body, &verifyResp); err != nil {
		return nil, err
	}

	return &verifyResp, nil
}

func Activate(apiKey string) error {
	resp, err := VerifySubscription(apiKey)
	if err != nil {
		return err
	}

	if !resp.Valid {
		return fmt.Errorf("invalid subscription: %s", resp.Error)
	}

	userID, _ := DecodeAPIKey(apiKey)

	return SaveAuth(&StoredAuth{
		APIKey: apiKey,
		UserID: userID,
		Valid:  true,
	})
}

func IsActivated() bool {
	auth, err := LoadAuth()
	if err != nil {
		return false
	}
	return auth.Valid
}

func CheckSubscription() (bool, error) {
	auth, err := LoadAuth()
	if err != nil {
		return false, err
	}

	resp, err := VerifySubscription(auth.APIKey)
	if err != nil {
		return false, err
	}

	// Update stored auth
	auth.Valid = resp.Valid
	SaveAuth(auth)

	return resp.Valid, nil
}
