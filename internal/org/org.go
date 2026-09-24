// Package org cuida da configuracao do tenant (dados da igreja) e das suas
// filiais/congregacoes. O isolamento multi-tenant e do RLS; a autorizacao
// (apenas Sede/admin) e feita no handler.
package org

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Repo agrupa as consultas de configuracao. E stateless.
type Repo struct{}

// ErrBranchInUse sinaliza que a filial nao pode ser excluida por ter membros.
var ErrBranchInUse = errors.New("nao e possivel excluir: ha membros vinculados a esta filial. Desative-a.")

// ErrBranchParentInvalido sinaliza hierarquia invalida (pai inexistente, o
// proprio no ou um descendente - evitando ciclos).
var ErrBranchParentInvalido = errors.New("filial superior invalida")

// Estrutura de governo das unidades: Matriz (Sede), Filial (congregacao) e
// PAE (Ponto de Atendimento de Evangelizacao).
const (
	BranchKindMatriz = "matriz"
	BranchKindFilial = "filial"
	BranchKindPAE    = "pae"
)

// Erros de validacao do tipo de unidade.
var (
	ErrBranchKindInvalido = errors.New("tipo de unidade invalido (use matriz, filial ou pae)")
	ErrPaeSemSuperior     = errors.New("PAE precisa estar vinculado a uma Matriz ou Filial")
	ErrMatrizComSuperior  = errors.New("Matriz nao pode ter unidade superior")
)

// validateBranchKind aplica as regras estruturais do tipo:
//   - matriz: nao pode ter superior (e a raiz);
//   - filial: sem restricao de vinculo;
//   - pae: exige superior (Matriz ou Filial), validado tambem pelo banco.
func validateBranchKind(kind, parentID string) error {
	switch kind {
	case BranchKindMatriz:
		if parentID != "" {
			return ErrMatrizComSuperior
		}
	case BranchKindFilial:
		// ok
	case BranchKindPAE:
		if parentID == "" {
			return ErrPaeSemSuperior
		}
	default:
		return ErrBranchKindInvalido
	}
	return nil
}

// isDescendantOrSelf informa se candidate esta na subarvore de root (inclusive
// root). E a guarda contra ciclo ao trocar o pai de uma filial.
func isDescendantOrSelf(ctx context.Context, tx pgx.Tx, root, candidate string) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `
		WITH RECURSIVE tree AS (
			SELECT id FROM branches WHERE id = $1::uuid
			UNION ALL
			SELECT b.id FROM branches b JOIN tree t ON b.parent_id = t.id
		)
		SELECT EXISTS(SELECT 1 FROM tree WHERE id = $2::uuid)`, root, candidate).Scan(&exists)
	return exists, err
}

// branchVisible confirma que a filial existe no escopo do usuario.
func branchVisible(ctx context.Context, tx pgx.Tx, id string) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM branches WHERE id = $1::uuid)`, id).Scan(&exists)
	return exists, err
}

// ---- Filiais ----

// Branch e uma filial/congregacao do tenant.
type Branch struct {
	ID          string          `json:"id"`
	ParentID    *string         `json:"parent_id,omitempty"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Kind        string          `json:"kind"`
	CNPJ        *string         `json:"cnpj,omitempty"`
	Address     json.RawMessage `json:"address,omitempty"`
	Geo         json.RawMessage `json:"geo,omitempty"`
	IsActive    bool            `json:"is_active"`
	MemberCount int             `json:"member_count"`
	CreatedAt   time.Time       `json:"created_at"`
	// Canais: permite indicar no grid se a filial tem WhatsApp conectado.
	WhatsAppStatus string `json:"whatsapp_status"`
	WhatsAppPhone  string `json:"whatsapp_phone"`
	// WhatsAppNumber e o numero efetivamente conectado na instancia Evolution.
	WhatsAppNumber string `json:"whatsapp_number"`
}

// BranchInput e o corpo de criacao/edicao de uma filial.
type BranchInput struct {
	Name     string          `json:"name"`
	Slug     *string         `json:"slug"`
	Kind     *string         `json:"kind"`
	CNPJ     *string         `json:"cnpj"`
	ParentID *string         `json:"parent_id"`
	Address  json.RawMessage `json:"address"`
	Geo      json.RawMessage `json:"geo"`
	IsActive *bool           `json:"is_active"`
}

const branchCols = `b.id::text, COALESCE(b.parent_id::text,''), b.name, b.slug, b.kind, b.cnpj,
	COALESCE(b.address::text,''), COALESCE(b.geo::text,''), b.is_active,
	(SELECT count(*) FROM members m WHERE m.branch_id = b.id)::int,
	b.created_at, b.whatsapp_status, COALESCE(b.whatsapp_phone,''), COALESCE(b.whatsapp_number,'')`

func scanBranch(row pgx.Row) (*Branch, error) {
	var b Branch
	var parent, address, geo string
	err := row.Scan(&b.ID, &parent, &b.Name, &b.Slug, &b.Kind, &b.CNPJ, &address, &geo, &b.IsActive, &b.MemberCount, &b.CreatedAt, &b.WhatsAppStatus, &b.WhatsAppPhone, &b.WhatsAppNumber)
	if err != nil {
		return nil, err
	}
	if parent != "" {
		b.ParentID = &parent
	}
	if address != "" {
		b.Address = json.RawMessage(address)
	}
	if geo != "" {
		b.Geo = json.RawMessage(geo)
	}
	return &b, nil
}

// ListBranches devolve as filiais do tenant do contexto.
func (r *Repo) ListBranches(ctx context.Context, tx pgx.Tx) ([]Branch, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+branchCols+`
		FROM branches b
		ORDER BY b.is_active DESC, b.name`)
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

// GetBranch busca uma filial pelo id.
func (r *Repo) GetBranch(ctx context.Context, tx pgx.Tx, id string) (*Branch, error) {
	return scanBranch(tx.QueryRow(ctx, `SELECT `+branchCols+` FROM branches b WHERE b.id = $1::uuid`, id))
}

// CreateBranch insere uma filial no tenant do contexto.
func (r *Repo) CreateBranch(ctx context.Context, tx pgx.Tx, tenantID string, in BranchInput) (*Branch, error) {
	kind := BranchKindFilial
	if in.Kind != nil && *in.Kind != "" {
		kind = *in.Kind
	}
	if err := validateBranchKind(kind, str(in.ParentID)); err != nil {
		return nil, err
	}
	if in.ParentID != nil && *in.ParentID != "" {
		ok, err := branchVisible(ctx, tx, *in.ParentID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrBranchParentInvalido
		}
	}
	var newID string
	err := tx.QueryRow(ctx, `
		INSERT INTO branches (tenant_id, parent_id, name, slug, kind, cnpj, address, geo, is_active)
		VALUES ($1, NULLIF($2,'')::uuid, $3, COALESCE(NULLIF($4,''), lower(regexp_replace($3, '[^a-zA-Z0-9]+', '-', 'g'))),
		        $5, NULLIF($6,''), NULLIF($7,'')::jsonb, NULLIF($8,'')::jsonb, COALESCE($9::boolean, true))
		RETURNING id::text`,
		tenantID, str(in.ParentID), in.Name, str(in.Slug), kind, str(in.CNPJ),
		string(in.Address), string(in.Geo), in.IsActive).Scan(&newID)
	if err != nil {
		return nil, err
	}
	return r.GetBranch(ctx, tx, newID)
}

// UpdateBranch edita uma filial. Os campos nulos sao preservados.
func (r *Repo) UpdateBranch(ctx context.Context, tx pgx.Tx, id string, in BranchInput) (*Branch, error) {
	// Tipo efetivo (novo ou atual) + superior efetivo, para validar as regras
	// da estrutura Matriz / Filial / PAE mesmo em edicao parcial.
	existing, err := r.GetBranch(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	effKind := existing.Kind
	if in.Kind != nil && *in.Kind != "" {
		effKind = *in.Kind
	}
	effParent := ""
	if existing.ParentID != nil {
		effParent = *existing.ParentID
	}
	if in.ParentID != nil {
		effParent = str(in.ParentID)
	}
	if err := validateBranchKind(effKind, effParent); err != nil {
		return nil, err
	}

	// Valida o novo pai (nao pode ser a propria filial nem um descendente,
	// senao a arvore vira um ciclo e a recursao de escopo estoura).
	if in.ParentID != nil && *in.ParentID != "" {
		ok, err := branchVisible(ctx, tx, *in.ParentID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrBranchParentInvalido
		}
		cycle, err := isDescendantOrSelf(ctx, tx, id, *in.ParentID)
		if err != nil {
			return nil, err
		}
		if cycle {
			return nil, ErrBranchParentInvalido
		}
	}
	var updatedID string
	err = tx.QueryRow(ctx, `
		UPDATE branches SET
			parent_id = CASE WHEN $2::boolean THEN NULLIF($3,'')::uuid ELSE parent_id END,
			name = COALESCE(NULLIF($4,''), name),
			slug = COALESCE(NULLIF($5,''), slug),
			kind = COALESCE($6, kind),
			cnpj = CASE WHEN $7::boolean THEN NULLIF($8,'') ELSE cnpj END,
			address = CASE WHEN $9::boolean THEN NULLIF($10,'')::jsonb ELSE address END,
			geo = CASE WHEN $11::boolean THEN NULLIF($12,'')::jsonb ELSE geo END,
			is_active = COALESCE($13::boolean, is_active),
			updated_at = now()
		WHERE id = $1::uuid
		RETURNING id::text`,
		id, in.ParentID != nil, str(in.ParentID), in.Name, str(in.Slug), in.Kind,
		in.CNPJ != nil, str(in.CNPJ),
		len(in.Address) > 0, string(in.Address), len(in.Geo) > 0, string(in.Geo), in.IsActive).Scan(&updatedID)
	if err != nil {
		return nil, err
	}
	return r.GetBranch(ctx, tx, updatedID)
}

// DeleteBranch exclui uma filial. Recusa quando ha membros vinculados (a FK e
// ON DELETE CASCADE e apagaria os membros em silencio); nesse caso, desative.
func (r *Repo) DeleteBranch(ctx context.Context, tx pgx.Tx, id string) error {
	var members int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM members WHERE branch_id = $1::uuid`, id).Scan(&members); err != nil {
		return err
	}
	if members > 0 {
		return ErrBranchInUse
	}
	tag, err := tx.Exec(ctx, `DELETE FROM branches WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ---- Tenant (dados da igreja) ----

// Tenant sao os dados cadastrais do tenant (igreja), incluindo o branding.
type Tenant struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	LegalName    *string   `json:"legal_name,omitempty"`
	CNPJ         *string   `json:"cnpj,omitempty"`
	Plan         string    `json:"plan"`
	Locale       string    `json:"locale"`
	Timezone     string    `json:"timezone"`
	LogoURL      *string   `json:"logo_url,omitempty"`
	BrandColor   *string   `json:"brand_color,omitempty"`
	FaviconURL   *string   `json:"favicon_url,omitempty"`
	CustomDomain *string   `json:"custom_domain,omitempty"`
	IsActive     bool      `json:"is_active"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TenantInput e o corpo de edicao do tenant.
type TenantInput struct {
	Name         string  `json:"name"`
	LegalName    *string `json:"legal_name"`
	CNPJ         *string `json:"cnpj"`
	Plan         *string `json:"plan"`
	Locale       *string `json:"locale"`
	Timezone     *string `json:"timezone"`
	LogoURL      *string `json:"logo_url"`
	BrandColor   *string `json:"brand_color"`
	FaviconURL   *string `json:"favicon_url"`
	CustomDomain *string `json:"custom_domain"`
}

// GetTenant devolve o tenant do contexto.
func (r *Repo) GetTenant(ctx context.Context, tx pgx.Tx) (*Tenant, error) {
	var t Tenant
	err := tx.QueryRow(ctx, `
		SELECT id::text, name, slug, legal_name, cnpj, plan, locale, timezone,
		       logo_url, brand_color, favicon_url, custom_domain, is_active, updated_at
		FROM tenants WHERE id = current_tenant()`).
		Scan(&t.ID, &t.Name, &t.Slug, &t.LegalName, &t.CNPJ, &t.Plan, &t.Locale, &t.Timezone,
			&t.LogoURL, &t.BrandColor, &t.FaviconURL, &t.CustomDomain, &t.IsActive, &t.UpdatedAt)
	return &t, err
}

// UpdateTenant edita os dados cadastrais e o branding do tenant do contexto.
func (r *Repo) UpdateTenant(ctx context.Context, tx pgx.Tx, in TenantInput) (*Tenant, error) {
	_, err := tx.Exec(ctx, `
		UPDATE tenants SET
			name = COALESCE(NULLIF($1,''), name),
			legal_name = COALESCE($2, legal_name),
			cnpj = COALESCE($3, cnpj),
			plan = COALESCE($4, plan),
			locale = COALESCE($5, locale),
			timezone = COALESCE($6, timezone),
			logo_url = COALESCE($7, logo_url),
			brand_color = COALESCE($8, brand_color),
			favicon_url = COALESCE($9, favicon_url),
			custom_domain = COALESCE($10, custom_domain),
			updated_at = now()
		WHERE id = current_tenant()`,
		in.Name, in.LegalName, in.CNPJ, in.Plan, in.Locale, in.Timezone,
		in.LogoURL, in.BrandColor, in.FaviconURL, in.CustomDomain)
	if err != nil {
		return nil, err
	}
	return r.GetTenant(ctx, tx)
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ---- Painel consolidado (Fase 2 / #28) ----

// BranchSummary e uma linha do painel consolidado Sede > Filiais.
type BranchSummary struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Kind         string  `json:"kind"`
	ParentID     *string `json:"parent_id,omitempty"`
	MemberCount  int     `json:"member_count"`
	VisitorCount int     `json:"visitor_count"`
	Income       float64 `json:"income"`
	Expense      float64 `json:"expense"`
	Net          float64 `json:"net"`
}

// Consolidated devolve, por filial dentro do escopo de leitura do usuario, os
// totais de membros, visitantes e movimentacao financeira no periodo. Para a
// Sede, cobre todo o tenant; para uma congregacao, ela e suas sub-congregacoes.
func (r *Repo) Consolidated(ctx context.Context, tx pgx.Tx, from, to string) ([]BranchSummary, error) {
	rows, err := tx.Query(ctx, `
		SELECT b.id::text, b.name, b.kind, b.parent_id::text,
			(SELECT count(*) FROM members m WHERE m.branch_id = b.id)::int,
			(SELECT count(*) FROM visitors v WHERE v.branch_id = b.id)::int,
			COALESCE((SELECT sum(t.amount) FROM financial_transactions t
				WHERE t.branch_id = b.id AND t.type = 'income' AND t.voided_at IS NULL
				  AND ($1 = '' OR t.occurred_at::date >= $1::date)
				  AND ($2 = '' OR t.occurred_at::date <= $2::date)), 0)::float8,
			COALESCE((SELECT sum(t.amount) FROM financial_transactions t
				WHERE t.branch_id = b.id AND t.type = 'expense' AND t.voided_at IS NULL
				  AND ($1 = '' OR t.occurred_at::date >= $1::date)
				  AND ($2 = '' OR t.occurred_at::date <= $2::date)), 0)::float8
		FROM branches b
		WHERE rls_read_scope(b.id, false) OR is_headquarters()
		ORDER BY b.name`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BranchSummary{}
	for rows.Next() {
		var s BranchSummary
		if err := rows.Scan(&s.ID, &s.Name, &s.Kind, &s.ParentID, &s.MemberCount, &s.VisitorCount, &s.Income, &s.Expense); err != nil {
			return nil, err
		}
		s.Net = s.Income - s.Expense
		out = append(out, s)
	}
	return out, rows.Err()
}
