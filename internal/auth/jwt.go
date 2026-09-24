package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessTokenType  = "access"
	RefreshTokenType = "refresh"
	// SelectionTokenType e um token curto (sem tenant ativo) emitido quando a
	// identidade tem mais de uma igreja e precisa escolher qual acessar.
	SelectionTokenType = "select"
)

// Claims representa o payload do token JWT do Chosen ERP.
type Claims struct {
	UserID   string `json:"uid"`
	TenantID string `json:"tid"`
	BranchID string `json:"bid"` // vazio => escopo Sede
	Role     string `json:"role"`
	Type     string `json:"typ"`
	jwt.RegisteredClaims
}

// NewToken gera um token assinado HMAC.
func NewToken(secret string, claims Claims, ttl time.Duration) (string, error) {
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(ttl))
	claims.IssuedAt = jwt.NewNumericDate(time.Now())
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken valida e decodifica um token.
func ParseToken(secret, raw string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
