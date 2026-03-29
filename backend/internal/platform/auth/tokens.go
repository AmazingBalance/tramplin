package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

type Claims struct {
	Subject      string    `json:"sub"`
	Role         string    `json:"role"`
	TokenVersion int       `json:"tokenVersion"`
	Type         TokenType `json:"type"`
	ExpiresAt    time.Time `json:"exp"`
}

func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	digest := sha256.Sum256(append(salt, []byte(password)...))
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(digest[:]), nil
}

func CheckPassword(storedHash, password string) bool {
	parts := strings.Split(storedHash, ":")
	if len(parts) != 2 {
		return false
	}

	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}

	digest := sha256.Sum256(append(salt, []byte(password)...))
	return hmac.Equal([]byte(parts[1]), []byte(hex.EncodeToString(digest[:])))
}

func IssueToken(secret string, claims Claims) (string, error) {
	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signature := sign(secret, payload)
	return payload + "." + signature, nil
}

func ParseToken(secret, raw string, expectedType TokenType) (*Claims, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid token format")
	}

	payload, signature := parts[0], parts[1]
	if !hmac.Equal([]byte(signature), []byte(sign(secret, payload))) {
		return nil, errors.New("invalid token signature")
	}

	decoded, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, errors.New("invalid token payload")
	}

	var claims Claims
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, errors.New("invalid token claims")
	}

	if claims.Type != expectedType {
		return nil, errors.New("unexpected token type")
	}

	if time.Now().UTC().After(claims.ExpiresAt) {
		return nil, errors.New("token expired")
	}

	return &claims, nil
}

func sign(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
