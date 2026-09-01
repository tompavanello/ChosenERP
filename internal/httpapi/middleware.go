package httpapi

import (
	"context"
	"net/http"
	"strings"

	"chosenerp/internal/auth"
)

type ctxKey string

const claimsKey ctxKey = "claims"

// withClaims injeta as claims decodificadas no contexto da requisição.
func withClaims(ctx context.Context, c *auth.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

func claimsFrom(ctx context.Context) (*auth.Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*auth.Claims)
	return c, ok
}

// Authenticator valida o Bearer token e o coloca no contexto.
func (a *App) Authenticator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeErr(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		raw := strings.TrimPrefix(header, "Bearer ")
		claims, err := auth.ParseToken(a.Config.JWTSecret, raw)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "invalid token")
			return
		}
		if claims.Type != auth.AccessTokenType {
			writeErr(w, http.StatusUnauthorized, "expected access token")
			return
		}
		next.ServeHTTP(w, r.WithContext(withClaims(r.Context(), claims)))
	})
}
