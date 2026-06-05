package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var ErrInvalidToken = errors.New("invalid token")
var ErrTokenExpired = errors.New("token expired")

type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

// Claims 表示 access token 内部保存的用户声明。
type Claims struct {
	Subject     string `json:"sub"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	AccountType string `json:"account_type"`
	IssuedAt    int64  `json:"iat"`
	ExpiresAt   int64  `json:"exp"`
}

// JWTManager 负责签发与校验 access token。
type JWTManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewJWTManager 创建一个基于 HS256 的 JWT 管理器。
func NewJWTManager(secret string, ttl time.Duration) (*JWTManager, error) {
	return newJWTManagerWithClock(secret, ttl, func() time.Time {
		return time.Now().UTC()
	})
}

func newJWTManagerWithClock(secret string, ttl time.Duration, now func() time.Time) (*JWTManager, error) {
	if strings.TrimSpace(secret) == "" || ttl <= 0 || now == nil {
		return nil, ErrInvalidAuthConfig
	}

	return &JWTManager{
		secret: []byte(secret),
		ttl:    ttl,
		now:    now,
	}, nil
}

// Issue 为指定用户签发 access token。
func (m *JWTManager) Issue(user User) (string, Claims, error) {
	now := m.now().Unix()
	claims := Claims{
		Subject:     user.ID,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
		AccountType: user.AccountType,
		IssuedAt:    now,
		ExpiresAt:   now + int64(m.ttl/time.Second),
	}

	headerJSON, err := json.Marshal(jwtHeader{
		Algorithm: "HS256",
		Type:      "JWT",
	})
	if err != nil {
		return "", Claims{}, err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", Claims{}, err
	}

	headerPart := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsPart := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := headerPart + "." + claimsPart

	signaturePart := base64.RawURLEncoding.EncodeToString(m.sign(signingInput))
	return signingInput + "." + signaturePart, claims, nil
}

// Parse 校验 access token 并返回解析后的声明。
func (m *JWTManager) Parse(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}

	signingInput := parts[0] + "." + parts[1]
	expected := m.sign(signingInput)
	actual, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	if subtle.ConstantTimeCompare(expected, actual) != 1 {
		return Claims{}, ErrInvalidToken
	}

	headerPayload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var header jwtHeader
	if err := json.Unmarshal(headerPayload, &header); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if header.Algorithm != "HS256" || header.Type != "JWT" {
		return Claims{}, ErrInvalidToken
	}

	claimsPayload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var claims Claims
	if err := json.Unmarshal(claimsPayload, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if claims.Subject == "" || claims.AccountType == "" {
		return Claims{}, ErrInvalidToken
	}
	if m.now().Unix() >= claims.ExpiresAt {
		return Claims{}, ErrTokenExpired
	}

	return claims, nil
}

func (m *JWTManager) sign(payload string) []byte {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(payload))
	return mac.Sum(nil)
}
