package httpapi

import (
	"context"
	"net/http"
	"regexp"
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
		// A igreja primeiro (muda o papel), depois a filial (usa o papel atual).
		a.applyTenantContext(r, claims)
		applyBranchContext(r, claims)
		next.ServeHTTP(w, r.WithContext(withClaims(r.Context(), claims)))
	})
}

// applyTenantContext permite fixar a igreja ativa por requisição, via
// `X-Tenant-Id` (uuid) ou `X-Tenant-Slug`. Só honra quando a identidade tem um
// vínculo ATIVO correspondente — a validação é feita no banco. Ignorado quando
// nenhum header é enviado (o tenant ativo do token prevalece).
func (a *App) applyTenantContext(r *http.Request, claims *auth.Claims) {
	tid := strings.TrimSpace(r.Header.Get("X-Tenant-Id"))
	slug := strings.TrimSpace(r.Header.Get("X-Tenant-Slug"))
	if tid == "" && slug == "" {
		return
	}
	ms, err := a.Auth.Memberships(r.Context(), claims.UserID)
	if err != nil {
		return
	}
	for _, m := range ms {
		if !m.IsActive {
			continue
		}
		if (tid != "" && m.TenantID == tid) || (slug != "" && m.TenantSlug == slug) {
			claims.TenantID = m.TenantID
			claims.BranchID = m.BranchID
			claims.Role = m.Role
			return
		}
	}
}

// applyBranchContext permite que a Sede (super_admin/admin_sede) troque o
// contexto de filial por requisição, via header `X-Branch-Id`:
//   - ausente           -> mantém o contexto do token (Sede = branch vazio);
//   - "all"             -> volta ao escopo Sede (todas as filiais do tenant);
//   - um uuid de filial -> opera como aquela filial (leitura e gravação).
//
// O papel é checado aqui; a filial é validada pelo próprio RLS (tenant do
// contexto), então um id de outro tenant simplesmente não devolve/grava nada.
// Usuário de filial não troca de contexto — o header é ignorado.
func applyBranchContext(r *http.Request, claims *auth.Claims) {
	if claims.Role != "super_admin" && claims.Role != "admin_sede" {
		return
	}
	bid := strings.TrimSpace(r.Header.Get("X-Branch-Id"))
	switch {
	case bid == "":
		// sem header: mantém o escopo do token
	case bid == "all":
		claims.BranchID = ""
	case uuidRe.MatchString(bid):
		claims.BranchID = bid
	default:
		// valor inválido (evita quebrar o cast de current_branch()::uuid)
	}
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
