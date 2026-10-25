package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/auth"
	"chosenerp/internal/members"
	"chosenerp/internal/store"
	"chosenerp/internal/visitors"
)

// errNoMemberLink indica que a identidade autenticada nao tem cadastro de
// membro vinculado nesta igreja (ver memberships.member_id).
var errNoMemberLink = errors.New("identidade sem vinculo de membro nesta igreja")

// errNoVisitorLink indica que a identidade autenticada nao tem cadastro de
// visitante vinculado nesta igreja (ver memberships.visitor_id).
var errNoVisitorLink = errors.New("identidade sem vinculo de visitante nesta igreja")

// errPlanLimit sinaliza que um teto do plano foi atingido.
var errPlanLimit = errors.New("limite do plano atingido")

// featureForPath mapeia a rota para a feature gateavel ("" = core/plataforma,
// nunca bloqueado). O prefixo mais especifico vence.
func featureForPath(path string) string {
	rest := strings.TrimPrefix(path, "/api/v1/")
	switch {
	case strings.HasPrefix(rest, "finance/"), strings.HasPrefix(rest, "receipts/"),
		strings.HasPrefix(rest, "payment-gateways"), strings.HasPrefix(rest, "payment-products"),
		strings.HasPrefix(rest, "payment-charges"), strings.HasPrefix(rest, "suppliers"),
		strings.HasPrefix(rest, "reports/account-statement"):
		return "finance"
	case strings.HasPrefix(rest, "reports/"):
		return "reports"
	case strings.HasPrefix(rest, "minutes"), strings.HasPrefix(rest, "votes"),
		strings.HasPrefix(rest, "legal-documents"), strings.HasPrefix(rest, "governance/"):
		return "governance"
	case strings.HasPrefix(rest, "ministries"), strings.HasPrefix(rest, "groups"),
		strings.HasPrefix(rest, "materials"):
		return "ministries"
	case strings.HasPrefix(rest, "rosters"):
		return "rosters"
	case strings.HasPrefix(rest, "events"), rest == "event-kinds", strings.HasPrefix(rest, "programacoes"):
		return "events"
	case strings.HasPrefix(rest, "kids/"):
		return "kids"
	case strings.HasPrefix(rest, "prayer-requests"), strings.HasPrefix(rest, "me/prayer-"),
		strings.HasPrefix(rest, "me/visitor/prayer-"):
		return "prayer"
	case strings.HasPrefix(rest, "announcements"), strings.HasPrefix(rest, "notifications/"):
		return "whatsapp"
	}
	return ""
}

// featureHandler bloqueia rotas de modulos nao incluidos no plano da igreja.
// Roda depois do Authenticator (precisa das claims). Plataforma (sem tenant)
// nao tem features de igreja e passa direto.
func (a *App) featureHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := featureForPath(r.URL.Path)
		if key == "" {
			next.ServeHTTP(w, r)
			return
		}
		claims, ok := claimsFrom(r.Context())
		if !ok {
			writeErr(w, http.StatusUnauthorized, "unauthenticated")
			return
		}
		if claims.TenantID == "" {
			next.ServeHTTP(w, r)
			return
		}
		allowed := false
		err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
			ent, e := a.Org.EffectiveForTenant(r.Context(), tx, claims.TenantID)
			if e != nil {
				return e
			}
			allowed = ent.Has(key)
			return nil
		})
		if err != nil {
			writeServerError(w, err)
			return
		}
		if !allowed {
			writeErr(w, http.StatusForbidden, "plan_feature_disabled")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// enforceQuota checa o teto do plano antes de uma criacao. kind:
// members|branches|users|storage. Roda na transacao de escopo do chamador.
func (a *App) enforceQuota(ctx context.Context, tx pgx.Tx, tenantID, kind string, addBytes int64) error {
	ent, err := a.Org.EffectiveForTenant(ctx, tx, tenantID)
	if err != nil {
		return err
	}
	switch kind {
	case "members":
		if ent.Limits.MaxMembers != nil && ent.Usage.Members >= *ent.Limits.MaxMembers {
			return fmt.Errorf("%w: limite de membros (%d)", errPlanLimit, *ent.Limits.MaxMembers)
		}
	case "branches":
		if ent.Limits.MaxBranches != nil && ent.Usage.Branches >= *ent.Limits.MaxBranches {
			return fmt.Errorf("%w: limite de filiais (%d)", errPlanLimit, *ent.Limits.MaxBranches)
		}
	case "users":
		if ent.Limits.MaxUsers != nil && ent.Usage.Users >= *ent.Limits.MaxUsers {
			return fmt.Errorf("%w: limite de usuarios (%d)", errPlanLimit, *ent.Limits.MaxUsers)
		}
	case "storage":
		if ent.Limits.MaxStorageMB != nil {
			limit := int64(*ent.Limits.MaxStorageMB) * 1024 * 1024
			if ent.Usage.StorageBytes+addBytes > limit {
				return fmt.Errorf("%w: limite de armazenamento (%d MB)", errPlanLimit, *ent.Limits.MaxStorageMB)
			}
		}
	}
	return nil
}

// writeQuotaErr traduz o erro de quota para HTTP. Devolve true se escreveu.
func writeQuotaErr(w http.ResponseWriter, err error) bool {
	if errors.Is(err, errPlanLimit) {
		writeErr(w, http.StatusForbidden, err.Error())
		return true
	}
	return false
}

// writeServerError registra o erro internamente e devolve uma mensagem generica
// (nunca vaza err.Error()/SQL ao cliente em 5xx).
func writeServerError(w http.ResponseWriter, err error) {
	log.Printf("httpapi: erro interno: %v", err)
	writeErr(w, http.StatusInternalServerError, "erro interno")
}

// validatePathUUIDs confere que os parametros de rota informados sao UUIDs.
// Devolve false (e escreve 400) quando algum for invalido - evita que um cast
// `::uuid` sem tratamento vire 500 (pgx 22P02).
func validatePathUUIDs(w http.ResponseWriter, r *http.Request, names ...string) bool {
	for _, n := range names {
		v := strings.TrimSpace(r.PathValue(n))
		if !uuidRe.MatchString(v) {
			writeErr(w, http.StatusBadRequest, "identificador invalido")
			return false
		}
	}
	return true
}

// uuidParams e um middleware que valida os parametros de rota informados como
// UUID antes de chamar o handler.
func (a *App) uuidParams(names ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !validatePathUUIDs(w, r, names...) {
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// uuidGuard valida globalmente os parametros de rota que sao SEMPRE UUID,
// evitando que um id malformado chegue a um cast `::uuid` e vire 500. Rotas
// cujo `{id}` NAO e UUID (a Biblia usa ids externos) ficam de fora.
func (a *App) uuidGuard(next http.Handler) http.Handler {
	names := []string{"id", "userId", "linkId", "memberId", "entryId", "assignmentId"}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/bible/") {
			for _, n := range names {
				v := strings.TrimSpace(r.PathValue(n))
				if v == "" {
					continue
				}
				if !uuidRe.MatchString(v) {
					writeErr(w, http.StatusBadRequest, "identificador invalido")
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// isPlatformAdmin informa (sem escrever resposta) se a identidade e operador
// da plataforma. Usado para permitir acoes reservadas (ex.: trocar o plano).
func (a *App) isPlatformAdmin(ctx context.Context, claims *auth.Claims) bool {
	var ok bool
	_ = a.Store.WithTenant(ctx, boundsFromClaims(claims), func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT is_platform_admin()`).Scan(&ok)
	})
	return ok
}

// memberForClaims resolve o MEMBRO da sessao dentro da transacao. E o ponto
// unico de resolucao do escopo self do app do membro.
func (a *App) memberForClaims(ctx context.Context, tx pgx.Tx, claims *auth.Claims) (*members.Member, error) {
	id, err := memberIDForClaims(ctx, tx, claims)
	if err != nil {
		return nil, err
	}
	if id == "" {
		return nil, errNoMemberLink
	}
	return a.Members.Get(ctx, tx, id)
}

// writeMemberErr traduz os erros comuns das rotas self para HTTP.
func writeMemberErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errNoMemberLink):
		writeErr(w, http.StatusForbidden, err.Error())
	case store.IsNotFound(err):
		writeErr(w, http.StatusNotFound, "membro nao encontrado")
	default:
		writeErr(w, http.StatusInternalServerError, err.Error())
	}
}

// hasPerm verifica se o papel do usuario tem a permissao informada. O JWT so
// carrega o papel, entao a permissao e resolvida no banco
// (roles -> role_permissions -> permissions) dentro do escopo do tenant.
func (a *App) hasPerm(ctx context.Context, tx pgx.Tx, claims *auth.Claims, perm string) (bool, error) {
	// Sede (super_admin/admin_sede) acessa todos os modulos liberados para a
	// igreja; o gate de plano (featureHandler) continua valendo separadamente.
	if auth.IsHQRole(claims.Role) {
		return true, nil
	}
	var ok bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM roles r
			JOIN role_permissions rp ON rp.role_id = r.id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE r.tenant_id = $1::uuid AND r.key = $2 AND p.key = $3)`,
		claims.TenantID, claims.Role, perm).Scan(&ok)
	return ok, err
}

// adminOnly restringe a rota a Sede (super_admin/admin_sede). Complementa o
// gate de permissao para operacoes sensiveis que o frontend so mostra ao admin.
func (a *App) adminOnly(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(r.Context())
		if !ok {
			writeErr(w, http.StatusUnauthorized, "unauthenticated")
			return
		}
		if !isAdmin(claims.Role) {
			writeErr(w, http.StatusForbidden, "somente admin_sede ou super_admin")
			return
		}
		next(w, r)
	})
}

// perm envolve um handler exigindo uma permissao. Usa o mesmo escopo RLS da
// sessao; nega com 403 quando o papel nao possui a permissao. Serve para
// proteger rotas sensiveis (ex.: moderacao de pedidos de oracao).
func (a *App) perm(permission string, next http.HandlerFunc) http.Handler {
	return a.permAny(permission)(next)
}

// permAny exige AO MENOS UMA das permissoes informadas. Usado quando a rota e
// consumida por papeis diferentes (ex.: seletor de filiais em varias telas).
func (a *App) permAny(permissions ...string) func(http.HandlerFunc) http.Handler {
	return func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := claimsFrom(r.Context())
			if !ok {
				writeErr(w, http.StatusUnauthorized, "unauthenticated")
				return
			}
			allowed := false
			err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
				for _, p := range permissions {
					okp, e := a.hasPerm(r.Context(), tx, claims, p)
					if e != nil {
						return e
					}
					if okp {
						allowed = true
						return nil
					}
				}
				return nil
			})
			if err != nil {
				writeServerError(w, err)
				return
			}
			if !allowed {
				writeErr(w, http.StatusForbidden, "forbidden")
				return
			}
			next(w, r)
		})
	}
}

// memberIDForClaims resolve o registro em `members` vinculado a identidade na
// igreja ativa (memberships.member_id). Devolve "" quando nao ha vinculo.
//
// Toda rota de "app do membro" DEVE derivar o membro daqui, nunca de um id
// enviado pelo cliente - e o unico jeito de garantir o escopo self.
func memberIDForClaims(ctx context.Context, tx pgx.Tx, claims *auth.Claims) (string, error) {
	var id *string
	err := tx.QueryRow(ctx, `
		SELECT member_id::text
		FROM memberships
		WHERE user_id = $1::uuid AND tenant_id = $2::uuid`,
		claims.UserID, claims.TenantID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if id == nil {
		return "", nil
	}
	return *id, nil
}

// visitorForClaims resolve o VISITANTE da sessao (memberships.visitor_id). E o
// ponto unico de resolucao do escopo self do app do visitante.
func (a *App) visitorForClaims(ctx context.Context, tx pgx.Tx, claims *auth.Claims) (*visitors.Visitor, error) {
	id, err := visitorIDForClaims(ctx, tx, claims)
	if err != nil {
		return nil, err
	}
	if id == "" {
		return nil, errNoVisitorLink
	}
	return a.Visitors.Get(ctx, tx, id)
}

// writeVisitorErr traduz os erros comuns das rotas self do visitante para HTTP.
func writeVisitorErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errNoVisitorLink):
		writeErr(w, http.StatusForbidden, err.Error())
	case store.IsNotFound(err):
		writeErr(w, http.StatusNotFound, "visitante nao encontrado")
	default:
		writeErr(w, http.StatusInternalServerError, err.Error())
	}
}

// visitorIDForClaims resolve o registro em `visitors` vinculado a identidade na
// igreja ativa (memberships.visitor_id). Devolve "" quando nao ha vinculo.
func visitorIDForClaims(ctx context.Context, tx pgx.Tx, claims *auth.Claims) (string, error) {
	var id *string
	err := tx.QueryRow(ctx, `
		SELECT visitor_id::text
		FROM memberships
		WHERE user_id = $1::uuid AND tenant_id = $2::uuid`,
		claims.UserID, claims.TenantID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if id == nil {
		return "", nil
	}
	return *id, nil
}
