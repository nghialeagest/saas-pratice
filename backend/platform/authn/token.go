package authn

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenConfig holds JWT signing and expiration configuration.
type TokenConfig struct {
	AccessSecret string        // Secret key for access token (HS512)
	VerifySecret string        // Different secret key for verification token (HS512)
	AccessTTL    time.Duration // Access token lifetime (default 15 minutes)
	VerifyTTL    time.Duration // Verify token lifetime (default 30 minutes)
	Issuer       string        // Expected token issuer
}

// DefaultTokenConfig returns production-ready default token settings.
func DefaultTokenConfig() TokenConfig {
	return TokenConfig{
		AccessSecret: "saasbase-default-access-secret-hs512-change-in-production-min-64-bytes!",
		VerifySecret: "saasbase-default-verify-secret-hs512-different-from-access-min-64-bytes!",
		AccessTTL:    15 * time.Minute,
		VerifyTTL:    30 * time.Minute,
		Issuer:       "saasbase",
	}
}

// AccessClaims contains standard registered claims and SaaS tenancy identifiers.
type AccessClaims struct {
	jwt.RegisteredClaims
	UserID    string `json:"uid"`
	TenantID  string `json:"tid"`
	SessionID string `json:"sid"`
	Username string  `json:"username"`
}

// VerifyClaims contains claims for sensitive single-purpose operations (e.g., email confirmation, 2FA).
type VerifyClaims struct {
	jwt.RegisteredClaims
	UserID  string `json:"uid"`
	Purpose string `json:"purpose"`
}

var (
	ErrInvalidToken        = errors.New("invalid token")
	ErrTokenExpired        = errors.New("token expired")
	ErrUnexpectedAlg       = errors.New("unexpected signing algorithm")
	ErrInvalidIssuer       = errors.New("invalid token issuer")
	ErrIdenticalSecretKeys = errors.New("access and verify secrets must not be identical")
)

// IssueAccess issues a signed access token using HS512 with short TTL.
func IssueAccess(cfg TokenConfig, userID, tenantID, sessionID, username string) (string, error) {
	if cfg.AccessSecret == cfg.VerifySecret {
		return "", ErrIdenticalSecretKeys
	}
	ttl := cfg.AccessTTL
	if ttl == 0 {
		ttl = 15 * time.Minute
	}
	now := time.Now()
	claims := AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		UserID:    userID,
		TenantID:  tenantID,
		SessionID: sessionID,
		Username: username,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	return token.SignedString([]byte(cfg.AccessSecret))
}

// ParseAccess verifies and decodes an access token using HS512.
func ParseAccess(cfg TokenConfig, tokenStr string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &AccessClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS512.Alg() {
			return nil, ErrUnexpectedAlg
		}
		return []byte(cfg.AccessSecret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*AccessClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	if cfg.Issuer != "" && claims.Issuer != cfg.Issuer {
		return nil, ErrInvalidIssuer
	}
	return claims, nil
}

// IssueVerify issues a verification token (e.g. 2FA, email verification) signed with VerifySecret.
func IssueVerify(cfg TokenConfig, userID, purpose string) (string, error) {
	if cfg.AccessSecret == cfg.VerifySecret {
		return "", ErrIdenticalSecretKeys
	}
	ttl := cfg.VerifyTTL
	if ttl == 0 {
		ttl = 30 * time.Minute
	}
	now := time.Now()
	verifyIssuer := cfg.Issuer + "-verify"
	claims := VerifyClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    verifyIssuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		UserID:  userID,
		Purpose: purpose,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	return token.SignedString([]byte(cfg.VerifySecret))
}

// ParseVerify verifies a single-purpose verification token and checks issuer.
func ParseVerify(cfg TokenConfig, tokenStr string) (*VerifyClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &VerifyClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS512.Alg() {
			return nil, ErrUnexpectedAlg
		}
		return []byte(cfg.VerifySecret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*VerifyClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	verifyIssuer := cfg.Issuer + "-verify"
	if claims.Issuer != verifyIssuer {
		return nil, ErrInvalidIssuer
	}
	return claims, nil
}
