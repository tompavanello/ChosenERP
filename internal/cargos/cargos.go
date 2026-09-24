// Package cargos gerencia o catalogo de cargos (funcoes/ministerios) da igreja
// e o vinculo dos membros com eles, com controle de mandato.
//
// Requisitos do cliente (CAD100):
//
//	1.3 Funcao/Ministerio - um mesmo membro pode ter MAIS DE UMA funcao.
//	1.4 Controle de mandato - data de inicio, vencimento e situacao
//	    (Ativo / Encerrado).
package cargos

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Cargo e uma funcao/ministerio do catalogo personalizavel da igreja.
type Cargo struct {
	ID           string    `json:"id"`
	BranchID     *string   `json:"branch_id,omitempty"` // vazio => global do tenant
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Kind         string    `json:"kind"` // eclesiastico | lideranca | ensino | apoio | outro
	RequiresTerm bool      `json:"requires_term"`
	IsActive     bool      `json:"is_active"`
	SortOrder    int       `json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
}

// MemberCargo e o mandato de um membro em um cargo.
type MemberCargo struct {
	ID        string  `json:"id"`
	MemberID  string  `json:"member_id"`
	CargoID   string  `json:"cargo_id"`
	CargoName string  `json:"cargo_name"`
	CargoKind string  `json:"cargo_kind"`
	StartedAt *string `json:"started_at,omitempty"`
	EndsAt    *string `json:"ends_at,omitempty"`
	Status    string  `json:"status"` // ativo | encerrado
	Notes     *string `json:"notes,omitempty"`
}

type CreateInput struct {
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	RequiresTerm *bool  `json:"requires_term"`
	SortOrder    *int   `json:"sort_order"`
}

type UpdateInput struct {
	Name         *string `json:"name"`
	Kind         *string `json:"kind"`
	RequiresTerm *bool   `json:"requires_term"`
	IsActive     *bool   `json:"is_active"`
	SortOrder    *int    `json:"sort_order"`
}

// AssignInput cria um mandato.
type AssignInput struct {
	CargoID   string  `json:"cargo_id"`
	StartedAt *string `json:"started_at"`
	EndsAt    *string `json:"ends_at"`
	Status    string  `json:"status"` // vazio => ativo
	Notes     *string `json:"notes"`
}

// UpdateAssignmentInput edita um mandato (PATCH: nil mantem o valor atual).
type UpdateAssignmentInput struct {
	StartedAt *string `json:"started_at"`
	EndsAt    *string `json:"ends_at"`
	Status    *string `json:"status"`
	Notes     *string `json:"notes"`
}

// Kinds sao os agrupamentos aceitos (espelham o CHECK da migracao 000020).
var Kinds = map[string]bool{
	"eclesiastico": true, "lideranca": true, "ensino": true, "apoio": true, "outro": true,
}

// StatusValidos sao as situacoes de mandato (requisito 1.4).
var StatusValidos = map[string]bool{"ativo": true, "encerrado": true}

type Repo struct{}

const cargoCols = `c.id::text, c.branch_id::text, c.name, c.slug, c.kind,
	c.requires_term, c.is_active, c.sort_order, c.created_at`

func scanCargo(row pgx.Row) (*Cargo, error) {
	var c Cargo
	err := row.Scan(&c.ID, &c.BranchID, &c.Name, &c.Slug, &c.Kind,
		&c.RequiresTerm, &c.IsActive, &c.SortOrder, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// List devolve o catalogo do escopo (globais do tenant + da filial), ativos e
// inativos - a UI precisa dos dois para permitir reativar.
func (r *Repo) List(ctx context.Context, tx pgx.Tx) ([]Cargo, error) {
	rows, err := tx.Query(ctx, `SELECT `+cargoCols+` FROM cargos c ORDER BY c.sort_order, c.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Cargo{}
	for rows.Next() {
		c, err := scanCargo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID string, in CreateInput) (*Cargo, error) {
	kind := in.Kind
	if kind == "" {
		kind = "outro"
	}
	term := false
	if in.RequiresTerm != nil {
		term = *in.RequiresTerm
	}
	order := 0
	if in.SortOrder != nil {
		order = *in.SortOrder
	}
	return scanCargo(tx.QueryRow(ctx, `
		INSERT INTO cargos (tenant_id, branch_id, name, slug, kind, requires_term, sort_order)
		VALUES ($1, NULLIF($2,'')::uuid, $3, $4, $5, $6, $7)
		RETURNING `+cargoCols,
		tenantID, branchID, strings.TrimSpace(in.Name), slugify(in.Name), kind, term, order))
}

func (r *Repo) Update(ctx context.Context, tx pgx.Tx, id string, in UpdateInput) (*Cargo, error) {
	// Renomear recalcula o slug para o backfill/legado continuar coerente.
	var name *string
	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		name = &n
	}
	var slug *string
	if name != nil {
		s := slugify(*name)
		slug = &s
	}
	return scanCargo(tx.QueryRow(ctx, `
		UPDATE cargos c SET
			name          = COALESCE($2, c.name),
			slug          = COALESCE($3, c.slug),
			kind          = COALESCE($4, c.kind),
			requires_term = COALESCE($5, c.requires_term),
			is_active     = COALESCE($6, c.is_active),
			sort_order    = COALESCE($7, c.sort_order)
		WHERE c.id = $1::uuid
		RETURNING `+cargoCols,
		id, name, slug, in.Kind, in.RequiresTerm, in.IsActive, in.SortOrder))
}

// InUse conta quantos mandatos (inclusive encerrados) referenciam o cargo.
// O cargo nao e apagavel enquanto houver historico: ON DELETE CASCADE apagaria
// o historico de mandato, que e justamente o que o requisito 1.4 preserva.
func (r *Repo) InUse(ctx context.Context, tx pgx.Tx, id string) (int64, error) {
	var n int64
	err := tx.QueryRow(ctx,
		`SELECT count(*) FROM member_cargos WHERE cargo_id = $1::uuid`, id).Scan(&n)
	return n, err
}

func (r *Repo) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM cargos WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

const memberCargoCols = `mc.id::text, mc.member_id::text, mc.cargo_id::text, c.name, c.kind,
	mc.started_at::text, mc.ends_at::text, mc.status, mc.notes`

func scanMemberCargo(row pgx.Row) (*MemberCargo, error) {
	var mc MemberCargo
	err := row.Scan(&mc.ID, &mc.MemberID, &mc.CargoID, &mc.CargoName, &mc.CargoKind,
		&mc.StartedAt, &mc.EndsAt, &mc.Status, &mc.Notes)
	if err != nil {
		return nil, err
	}
	return &mc, nil
}

// ListByMember devolve os cargos de um membro - ativos primeiro, depois os
// encerrados, cada grupo pelo mandato mais recente.
func (r *Repo) ListByMember(ctx context.Context, tx pgx.Tx, memberID string) ([]MemberCargo, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+memberCargoCols+`
		FROM member_cargos mc
		JOIN cargos c ON c.id = mc.cargo_id
		WHERE mc.member_id = $1::uuid
		ORDER BY (mc.status = 'ativo') DESC, mc.started_at DESC NULLS LAST, c.name`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MemberCargo{}
	for rows.Next() {
		mc, err := scanMemberCargo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *mc)
	}
	return out, rows.Err()
}

// Assign vincula um cargo ao membro.
//
// ON CONFLICT DO NOTHING sem alvo e obrigatorio: a tabela tem dois indices
// unicos parciais (com e sem data de inicio) e o alvo inferido nao cobre ambos.
func (r *Repo) Assign(ctx context.Context, tx pgx.Tx, memberID string, in AssignInput) (*MemberCargo, error) {
	status := in.Status
	if status == "" {
		status = "ativo"
	}
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO member_cargos (member_id, cargo_id, started_at, ends_at, status, notes)
		VALUES ($1::uuid, $2::uuid, NULLIF($3,'')::date, NULLIF($4,'')::date, $5, $6)
		ON CONFLICT DO NOTHING
		RETURNING id::text`,
		memberID, in.CargoID, in.StartedAt, in.EndsAt, status, nullStr(in.Notes)).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.getAssignment(ctx, tx, memberID, id)
}

// UpdateAssignment edita o mandato. Distingue tres casos por data, porque o
// PATCH precisa poder LIMPAR uma data (ex.: reabrir um mandato encerrado):
//
//	ausente (NULL)  => mantem o valor atual
//	"" (string vazia) => grava NULL
//	"2024-01-31"    => grava a data
//
// Sem o caso da string vazia nao haveria como desfazer um vencimento: com
// COALESCE($::date, ...) a unica alternativa seria mandar uma data falsa.
func (r *Repo) UpdateAssignment(ctx context.Context, tx pgx.Tx, memberID, id string, in UpdateAssignmentInput) (*MemberCargo, error) {
	_, err := tx.Exec(ctx, `
		UPDATE member_cargos SET
			started_at = CASE WHEN $3::text IS NULL THEN started_at
			                  WHEN btrim($3::text) = '' THEN NULL
			                  ELSE $3::text::date END,
			ends_at    = CASE WHEN $4::text IS NULL THEN ends_at
			                  WHEN btrim($4::text) = '' THEN NULL
			                  ELSE $4::text::date END,
			status     = COALESCE($5, status),
			notes      = COALESCE($6, notes)
		WHERE id = $1::uuid AND member_id = $2::uuid`,
		id, memberID, in.StartedAt, in.EndsAt, in.Status, nullStr(in.Notes))
	if err != nil {
		return nil, err
	}
	return r.getAssignment(ctx, tx, memberID, id)
}

func (r *Repo) Unassign(ctx context.Context, tx pgx.Tx, memberID, id string) error {
	tag, err := tx.Exec(ctx,
		`DELETE FROM member_cargos WHERE id = $1::uuid AND member_id = $2::uuid`, id, memberID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repo) getAssignment(ctx context.Context, tx pgx.Tx, memberID, id string) (*MemberCargo, error) {
	return scanMemberCargo(tx.QueryRow(ctx, `
		SELECT `+memberCargoCols+`
		FROM member_cargos mc
		JOIN cargos c ON c.id = mc.cargo_id
		WHERE mc.member_id = $1::uuid AND mc.id = $2::uuid`, memberID, id))
}

// slugify espelha o de ministries: sem acentos, minusculo, com hifens.
func slugify(s string) string {
	out := make([]rune, 0, len(s))
	lastDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
			lastDash = false
		default:
			if len(out) > 0 && !lastDash {
				out = append(out, '-')
				lastDash = true
			}
		}
	}
	for len(out) > 0 && out[len(out)-1] == '-' {
		out = out[:len(out)-1]
	}
	if len(out) == 0 {
		return "cargo"
	}
	return string(out)
}

func nullStr(s *string) *string {
	if s != nil && *s == "" {
		return nil
	}
	return s
}
