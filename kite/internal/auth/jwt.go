package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	AccessLevelReadOnly = 0
	AccessLevelUser     = 1
	AccessLevelManager  = 2
	AccessLevelAdmin    = 3
)

type TokenService struct {
	secret []byte
	ttl    time.Duration
}

type Claims struct {
	Subject     string `json:"sub"`
	AccessLevel int    `json:"access_level"`
	IssuedAt    int64  `json:"iat"`
	ExpiresAt   int64  `json:"exp"`
}

func NewTokenService(secret string, ttl time.Duration) (*TokenService, error) {
	if secret == "" {
		return nil, fmt.Errorf("jwt secret is required")
	}

	return &TokenService{
		secret: []byte(secret),
		ttl:    ttl,
	}, nil
}

func (s *TokenService) IssueAccessToken(subject string, accessLevel int) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(s.ttl)

	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	claims := Claims{
		Subject:     subject,
		AccessLevel: accessLevel,
		IssuedAt:    now.Unix(),
		ExpiresAt:   expiresAt.Unix(),
	}

	encodedHeader, err := encodeJSON(header)
	if err != nil {
		return "", time.Time{}, err
	}

	encodedClaims, err := encodeJSON(claims)
	if err != nil {
		return "", time.Time{}, err
	}

	signingInput := encodedHeader + "." + encodedClaims
	signature := s.sign(signingInput)

	return signingInput + "." + signature, expiresAt, nil
}

func (s *TokenService) VerifyAccessToken(token string) (Claims, error) {
	token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, fmt.Errorf("invalid token format")
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSignature := s.sign(signingInput)
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSignature)) {
		return Claims{}, fmt.Errorf("invalid token signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, fmt.Errorf("invalid token payload: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, fmt.Errorf("invalid token claims: %w", err)
	}

	if claims.ExpiresAt <= time.Now().UTC().Unix() {
		return Claims{}, fmt.Errorf("token expired")
	}

	return claims, nil
}

func (s *TokenService) sign(signingInput string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(signingInput))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func encodeJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(data), nil
}

func constantTimeEqual(a string, b string) bool {
	return hmac.Equal([]byte(a), []byte(b))
}

func ValidateCredentials(username string, password string, adminUsername string, adminPassword string) bool {
	username = strings.TrimSpace(username)

	if adminUsername == "" || adminPassword == "" {
		return false
	}

	return constantTimeEqual(username, adminUsername) && constantTimeEqual(password, adminPassword)
}
