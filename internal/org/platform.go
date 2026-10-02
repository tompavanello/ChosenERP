package org

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---- Console da plataforma: igrejas por id ----

// AdminTenantInput edita uma igreja pelo console da plataforma (sempre por id).
// Campos nulos sao preservados; `limits` so substitui quando informado.
type AdminTenantInput struct {
	Name         *string         `json:"name"`
	Slug         *string         `json:"slug"`
	LegalName    *string         `json:"legal_name"`
	CNPJ         *string         `json:"cnpj"`
	Plan         *string         `json:"plan"`
	Locale       *string         `json:"locale"`
	Timezone     *string         `json:"timezone"`
	LogoURL      *string         `json:"logo_url"`
	BrandColor   *string         `json:"brand_color"`
	FaviconURL   *string         `json:"favicon_url"`
	CustomDomain *string         `json:"custom_domain"`
	Limits       json.RawMessage `json:"limits"`
	Features     json.RawMessage `json:"features"`
	IsActive     *bool           `json:"is_active"`
}

// tenantAdminCols e a projecao usada pelo console (inclui limits/features).
const tenantAdminCols = `id::text, name, slug, legal_name, cnpj, plan, locale, timezone,
	logo_url, brand_color, favicon_url, custom_domain,
	COALESCE(limits::text,''), COALESCE(features::text,''), is_active, updated_at`

func scanTenantAdmin(row pgx.Row) (*Tenant, error) {
	var t Tenant
	var limits, features string
	err := row.Scan(&t.ID, &t.Name, &t.Slug, &t.LegalName, &t.CNPJ, &t.Plan, &t.Locale, &t.Timezone,
		&t.LogoURL, &t.BrandColor, &t.FaviconURL, &t.CustomDomain, &limits, &features, &t.IsActive, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if limits != "" {
		t.Limits = json.RawMessage(limits)
	}
	if features != "" {
		t.Features = json.RawMessage(features)
	}
	return &t, nil
}

// GetTenantAdmin carrega uma igreja por id (fora do RLS; exige platform admin).
func (r *Repo) GetTenantAdmin(ctx context.Context, tx pgx.Tx, id string) (*Tenant, error) {
	return scanTenantAdmin(tx.QueryRow(ctx,
		`SELECT `+tenantAdminCols+` FROM tenants WHERE id = $1::uuid`, id))
}

// UpdateTenantAdmin edita uma igreja por id. O slug, quando informado, e validado.
func (r *Repo) UpdateTenantAdmin(ctx context.Context, tx pgx.Tx, id string, in AdminTenantInput) (*Tenant, error) {
	var slugArg *string
	if in.Slug != nil && strings.TrimSpace(*in.Slug) != "" {
		s, err := validTenantSlug(*in.Slug)
		if err != nil {
			return nil, err
		}
		slugArg = &s
	}
	_, err := tx.Exec(ctx, `
		UPDATE tenants SET
			name = COALESCE(NULLIF($2,''), name),
			slug = COALESCE($3, slug),
			legal_name = COALESCE($4, legal_name),
			cnpj = COALESCE($5, cnpj),
			plan = COALESCE(NULLIF($6,''), plan),
			locale = COALESCE($7, locale),
			timezone = COALESCE($8, timezone),
			logo_url = COALESCE($9, logo_url),
			brand_color = COALESCE($10, brand_color),
			favicon_url = COALESCE($11, favicon_url),
			custom_domain = COALESCE($12, custom_domain),
			limits = CASE WHEN $13::boolean THEN $14::jsonb ELSE limits END,
			features = CASE WHEN $15::boolean THEN $16::jsonb ELSE features END,
			is_active = COALESCE($17::boolean, is_active),
			updated_at = now()
		WHERE id = $1::uuid`,
		id, str(in.Name), slugArg, in.LegalName, in.CNPJ, str(in.Plan), in.Locale, in.Timezone,
		in.LogoURL, in.BrandColor, in.FaviconURL, in.CustomDomain,
		len(in.Limits) > 0, featuresArg(in.Limits), len(in.Features) > 0, featuresArg(in.Features), in.IsActive)
	if err != nil {
		return nil, err
	}
	return r.GetTenantAdmin(ctx, tx, id)
}

// TenantUsage resume o uso da igreja para comparar com os limites do plano.
type TenantUsage struct {
	Members      int   `json:"members"`
	Branches     int   `json:"branches"`
	Users        int   `json:"users"`
	StorageBytes int64 `json:"storage_bytes"`
}

// TenantUsage conta membros, filiais, usuarios e o tamanho dos anexos.
func (r *Repo) TenantUsage(ctx context.Context, tx pgx.Tx, id string) (*TenantUsage, error) {
	var u TenantUsage
	err := tx.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM members WHERE tenant_id = $1::uuid)::int,
			(SELECT count(*) FROM branches WHERE tenant_id = $1::uuid)::int,
			(SELECT count(*) FROM memberships WHERE tenant_id = $1::uuid)::int,
			COALESCE((SELECT sum(file_size) FROM financial_attachments WHERE tenant_id = $1::uuid), 0)::bigint`,
		id).Scan(&u.Members, &u.Branches, &u.Users, &u.StorageBytes)
	return &u, err
}

// ---- Catalogo de planos ----

// Plan e um plano do catalogo da plataforma.
type Plan struct {
	Key          string          `json:"key"`
	Name         string          `json:"name"`
	Description  *string         `json:"description,omitempty"`
	PriceCents   int             `json:"price_cents"`
	Currency     string          `json:"currency"`
	MaxMembers   *int            `json:"max_members,omitempty"`
	MaxBranches  *int            `json:"max_branches,omitempty"`
	MaxUsers     *int            `json:"max_users,omitempty"`
	MaxStorageMB *int            `json:"max_storage_mb,omitempty"`
	Features     json.RawMessage `json:"features,omitempty"`
	IsActive     bool            `json:"is_active"`
	SortOrder    int             `json:"sort_order"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// PlanInput e o corpo de criacao/edicao de um plano. Nos limites, 0 significa
// ilimitado (vira NULL). Campo nulo preserva o valor atual.
type PlanInput struct {
	Key          string          `json:"key"`
	Name         string          `json:"name"`
	Description  *string         `json:"description"`
	PriceCents   *int            `json:"price_cents"`
	Currency     *string         `json:"currency"`
	MaxMembers   *int            `json:"max_members"`
	MaxBranches  *int            `json:"max_branches"`
	MaxUsers     *int            `json:"max_users"`
	MaxStorageMB *int            `json:"max_storage_mb"`
	Features     json.RawMessage `json:"features"`
	IsActive     *bool           `json:"is_active"`
	SortOrder    *int            `json:"sort_order"`
}

const planCols = `key, name, description, price_cents, currency, max_members, max_branches,
	max_users, max_storage_mb, COALESCE(features::text,'{}'), is_active, sort_order, updated_at`

func scanPlan(row pgx.Row) (*Plan, error) {
	var p Plan
	var features string
	err := row.Scan(&p.Key, &p.Name, &p.Description, &p.PriceCents, &p.Currency,
		&p.MaxMembers, &p.MaxBranches, &p.MaxUsers, &p.MaxStorageMB, &features,
		&p.IsActive, &p.SortOrder, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.Features = json.RawMessage(features)
	return &p, nil
}

// ListPlans lista o catalogo de planos.
func (r *Repo) ListPlans(ctx context.Context, tx pgx.Tx) ([]Plan, error) {
	rows, err := tx.Query(ctx, `SELECT `+planCols+` FROM plans ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Plan{}
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// GetPlan carrega um plano pela chave.
func (r *Repo) GetPlan(ctx context.Context, tx pgx.Tx, key string) (*Plan, error) {
	return scanPlan(tx.QueryRow(ctx, `SELECT `+planCols+` FROM plans WHERE key = $1`, key))
}

// CreatePlan insere um plano no catalogo.
func (r *Repo) CreatePlan(ctx context.Context, tx pgx.Tx, in PlanInput) (*Plan, error) {
	key := strings.ToLower(strings.TrimSpace(in.Key))
	name := strings.TrimSpace(in.Name)
	if key == "" || name == "" {
		return nil, ErrPlanInvalido
	}
	features := "{}"
	if len(in.Features) > 0 {
		features = string(in.Features)
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO plans (key, name, description, price_cents, currency,
		                   max_members, max_branches, max_users, max_storage_mb, features, is_active, sort_order)
		VALUES ($1, $2, $3, COALESCE($4,0), COALESCE($5,'BRL'),
		        $6, $7, $8, $9, $10::jsonb, COALESCE($11,true), COALESCE($12,0))`,
		key, name, in.Description, in.PriceCents, in.Currency,
		nullIfNonPositive(in.MaxMembers), nullIfNonPositive(in.MaxBranches),
		nullIfNonPositive(in.MaxUsers), nullIfNonPositive(in.MaxStorageMB),
		features, in.IsActive, in.SortOrder)
	if err != nil {
		return nil, err
	}
	return r.GetPlan(ctx, tx, key)
}

// UpdatePlan edita um plano do catalogo.
func (r *Repo) UpdatePlan(ctx context.Context, tx pgx.Tx, key string, in PlanInput) (*Plan, error) {
	_, err := tx.Exec(ctx, `
		UPDATE plans SET
			name = COALESCE(NULLIF($2,''), name),
			description = COALESCE($3, description),
			price_cents = COALESCE($4, price_cents),
			currency = COALESCE($5, currency),
			max_members = CASE WHEN $6::boolean THEN $7 ELSE max_members END,
			max_branches = CASE WHEN $8::boolean THEN $9 ELSE max_branches END,
			max_users = CASE WHEN $10::boolean THEN $11 ELSE max_users END,
			max_storage_mb = CASE WHEN $12::boolean THEN $13 ELSE max_storage_mb END,
			features = CASE WHEN $14::boolean THEN $15::jsonb ELSE features END,
			is_active = COALESCE($16::boolean, is_active),
			sort_order = COALESCE($17, sort_order),
			updated_at = now()
		WHERE key = $1`,
		key, strings.TrimSpace(in.Name), in.Description, in.PriceCents, in.Currency,
		in.MaxMembers != nil, nullIfNonPositive(in.MaxMembers),
		in.MaxBranches != nil, nullIfNonPositive(in.MaxBranches),
		in.MaxUsers != nil, nullIfNonPositive(in.MaxUsers),
		in.MaxStorageMB != nil, nullIfNonPositive(in.MaxStorageMB),
		len(in.Features) > 0, featuresArg(in.Features), in.IsActive, in.SortOrder)
	if err != nil {
		return nil, err
	}
	return r.GetPlan(ctx, tx, key)
}

// ErrPlanInvalido sinaliza chave/nome ausentes.
var ErrPlanInvalido = errors.New("chave e nome do plano sao obrigatorios")

// nullIfNonPositive converte <= 0 em NULL (ilimitado).
func nullIfNonPositive(v *int) *int {
	if v == nil || *v <= 0 {
		return nil
	}
	return v
}

// featuresArg evita enviar "" (JSON invalido) quando o campo nao foi informado:
// no UPDATE, o parametro jsonb e convertido mesmo quando o CASE nao o usa.
func featuresArg(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return string(raw)
}

// ListBranchesByTenant lista as filiais de uma igreja (console da plataforma,
// fora do RLS).
func (r *Repo) ListBranchesByTenant(ctx context.Context, tx pgx.Tx, tenantID string) ([]Branch, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+branchCols+`
		FROM branches b
		WHERE b.tenant_id = $1::uuid
		ORDER BY b.is_active DESC, b.name`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Branch{}
	for rows.Next() {
		b, err := scanBranch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}

// ---- Estatisticas gerais da plataforma ----

// PlanCount e a contagem de igrejas por plano.
type PlanCount struct {
	Plan  string `json:"plan"`
	Count int    `json:"count"`
}

// PlatformStats resume a plataforma inteira (sem dados operacionais de igreja).
type PlatformStats struct {
	Tenants       int           `json:"tenants"`
	TenantsActive int           `json:"tenants_active"`
	Members       int           `json:"members"`
	Users         int           `json:"users"`
	Branches      int           `json:"branches"`
	StorageBytes  int64         `json:"storage_bytes"`
	ByPlan        []PlanCount   `json:"by_plan"`
	RecentTenants []AdminTenant `json:"recent_tenants"`
}

// Stats agrega contagens globais das igrejas (roda em WithSystem).
func (r *Repo) Stats(ctx context.Context, tx pgx.Tx) (*PlatformStats, error) {
	var s PlatformStats
	if err := tx.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM tenants)::int,
			(SELECT count(*) FROM tenants WHERE is_active)::int,
			(SELECT count(*) FROM members)::int,
			(SELECT count(*) FROM users)::int,
			(SELECT count(*) FROM branches)::int,
			COALESCE((SELECT sum(file_size) FROM financial_attachments), 0)::bigint`).
		Scan(&s.Tenants, &s.TenantsActive, &s.Members, &s.Users, &s.Branches, &s.StorageBytes); err != nil {
		return nil, err
	}

	prows, err := tx.Query(ctx, `SELECT plan, count(*)::int FROM tenants GROUP BY plan ORDER BY count(*) DESC`)
	if err != nil {
		return nil, err
	}
	s.ByPlan = []PlanCount{}
	for prows.Next() {
		var pc PlanCount
		if err := prows.Scan(&pc.Plan, &pc.Count); err != nil {
			prows.Close()
			return nil, err
		}
		s.ByPlan = append(s.ByPlan, pc)
	}
	prows.Close()
	if err := prows.Err(); err != nil {
		return nil, err
	}

	rrows, err := tx.Query(ctx, `
		SELECT id::text, name, slug, plan, is_active, created_at,
			(SELECT count(*) FROM branches b WHERE b.tenant_id = t.id)::int,
			(SELECT count(*) FROM members m WHERE m.tenant_id = t.id)::int
		FROM tenants t
		ORDER BY created_at DESC
		LIMIT 5`)
	if err != nil {
		return nil, err
	}
	s.RecentTenants = []AdminTenant{}
	for rrows.Next() {
		var t AdminTenant
		if err := rrows.Scan(&t.ID, &t.Name, &t.Slug, &t.Plan, &t.IsActive,
			&t.CreatedAt, &t.BranchCount, &t.MemberCount); err != nil {
			rrows.Close()
			return nil, err
		}
		s.RecentTenants = append(s.RecentTenants, t)
	}
	rrows.Close()
	return &s, rrows.Err()
}
