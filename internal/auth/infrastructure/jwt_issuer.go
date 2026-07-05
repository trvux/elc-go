package infrastructure

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

// claims embeds jwt.RegisteredClaims (Subject carries the user ID, per
// golang-jwt/jwt v5 convention) plus the one custom field handlers actually
// need: role, so RequireRole can gate a route without a DB round trip.
type claims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
}

// JWTTokenIssuer implements both domain.TokenIssuer (issuing) and
// httpserver.TokenVerifier (verifying) — one type, since both sides share
// the same secret and claims shape.
type JWTTokenIssuer struct {
	secret []byte
	ttl    time.Duration
}

var (
	_ domain.TokenIssuer       = (*JWTTokenIssuer)(nil)
	_ httpserver.TokenVerifier = (*JWTTokenIssuer)(nil)
)

func NewJWTTokenIssuer(secret string, ttl time.Duration) *JWTTokenIssuer {
	return &JWTTokenIssuer{secret: []byte(secret), ttl: ttl}
}

func (i *JWTTokenIssuer) IssueAccessToken(userID string, role domain.Role) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(i.ttl)),
		},
		Role: string(role),
	})
	return token.SignedString(i.secret)
}

func (i *JWTTokenIssuer) Verify(tokenString string) (httpserver.Claims, error) {
	parsed, err := jwt.ParseWithClaims(tokenString, &claims{}, func(t *jwt.Token) (any, error) {
		return i.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil {
		return httpserver.Claims{}, err
	}

	c, ok := parsed.Claims.(*claims)
	if !ok || !parsed.Valid {
		return httpserver.Claims{}, errors.New("invalid access token")
	}

	return httpserver.Claims{UserID: c.Subject, Role: c.Role}, nil
}
