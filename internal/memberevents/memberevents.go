// Package memberevents gerencia o catalogo CONFIGURAVEL de eventos da vida
// eclesiastica do membro (migracao 000065).
//
// Cada tipo declara EXATAMENTE o que "movimenta" no membro quando e lancado no
// historico (simplificado pela 000067):
//   - sets_status      -> situacao resultante (ativo/membro/inativo; NULL = nao mexe);
//   - sets_exit_reason -> motivo da baixa (so quando a situacao resultante e inativo);
//   - sets_date_field  -> data registrada (none/baptism/joined_at/marriage_date).
//
// Na pratica o estado do membro e DERIVADO do historico: o cadastro nao edita
// esses campos, so registra eventos. O catalogo e global do tenant (branch_id
// sempre NULL) e substitui a lista fixa que existia no frontend.
package memberevents

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// EventKind e um tipo de evento do catalogo da igreja.
type EventKind struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Category       string    `json:"category"`
	Tone           string    `json:"tone"`
	SetsStatus     *string   `json:"sets_status,omitempty"`
	SetsExitReason *string   `json:"sets_exit_reason,omitempty"`
	SetsDateField  string    `json:"sets_date_field"`
	IsActive       bool      `json:"is_active"`
	SortOrder      int       `json:"sort_order"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateInput struct {
	Name           string `json:"name"`
	Category       string `json:"category"`
	Tone           string `json:"tone"`
	SetsStatus     string `json:"sets_status"`
	SetsExitReason string `json:"sets_exit_reason"`
	SetsDateField  string `json:"sets_date_field"`
	SortOrder      *int   `json:"sort_order"`
}

type UpdateInput struct {
	Name           *string `json:"name"`
	Category       *string `json:"category"`
	Tone           *string `json:"tone"`
	SetsStatus     *string `json:"sets_status"`
	SetsExitReason *string `json:"sets_exit_reason"`
	SetsDateField  *string `json:"sets_date_field"`
	IsActive       *bool   `json:"is_active"`
	SortOrder      *int    `json:"sort_order"`
}

// Categories agrupam os eventos na UI (espelham o CHECK da migracao 000065).
var Categories = map[string]bool{
	"batismo": true, "profissao": true, "recepcao": true, "retorno": true,
	"saida": true, "ordenacao": true, "disciplina": true, "sistema": true, "outro": true,
}

// Statuses validos (espelham o CHECK de members.membership_status, 000066):
// Ativo Professo / Ativo Nao Professo / Inativo (subdividido por exit_reason).
var Statuses = map[string]bool{
	"active": true, "member": true, "inactive": true,
}

// ExitReasons validos (espelham o CHECK de members.exit_reason).
var ExitReasons = map[string]bool{
	"falecimento": true, "desligamento": true, "transferencia": true,
	"abandono": true, "ausencia": true, "outro": true,
}

// DateFields validos para sets_date_field.
var DateFields = map[string]bool{
	"none": true, "baptism": true, "joined_at": true, "marriage_date": true,
}

// Tones aceitos pelo Badge do design system.
var Tones = map[string]bool{
	"zinc": true, "sky": true, "green": true, "amber": true, "red": true,
	"indigo": true, "brand": true,
}

type Repo struct{}

const cols = `id::text, name, slug, category, tone, sets_status, sets_exit_reason,
	sets_date_field, is_active, sort_order, created_at`

func scan(row pgx.Row) (*EventKind, error) {
	var k EventKind
	err := row.Scan(&k.ID, &k.Name, &k.Slug, &k.Category, &k.Tone, &k.SetsStatus,
		&k.SetsExitReason, &k.SetsDateField, &k.IsActive, &k.SortOrder, &k.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

// List devolve o catalogo do tenant (ativos e inativos - a UI precisa dos dois
// para permitir reativar).
func (r *Repo) List(ctx context.Context, tx pgx.Tx) ([]EventKind, error) {
	rows, err := tx.Query(ctx, `SELECT `+cols+` FROM member_event_kinds ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []EventKind{}
	for rows.Next() {
		k, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *k)
	}
	return out, rows.Err()
}

// GetByID carrega um tipo do escopo (usado pelos efeitos do historico).
func (r *Repo) GetByID(ctx context.Context, tx pgx.Tx, id string) (*EventKind, error) {
	return scan(tx.QueryRow(ctx, `SELECT `+cols+` FROM member_event_kinds WHERE id = $1::uuid`, id))
}

// GetBySlug carrega um tipo pelo slug estavel (chave usada pelo historico).
func (r *Repo) GetBySlug(ctx context.Context, tx pgx.Tx, slug string) (*EventKind, error) {
	return scan(tx.QueryRow(ctx, `SELECT `+cols+` FROM member_event_kinds WHERE slug = $1`, slug))
}

func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID string, in CreateInput) (*EventKind, error) {
	category := in.Category
	if category == "" {
		category = "outro"
	}
	tone := in.Tone
	if tone == "" {
		tone = "zinc"
	}
	dateField := in.SetsDateField
	if dateField == "" {
		dateField = "none"
	}
	order := 0
	if in.SortOrder != nil {
		order = *in.SortOrder
	}
	return scan(tx.QueryRow(ctx, `
		INSERT INTO member_event_kinds
			(tenant_id, name, slug, category, tone, sets_status, sets_exit_reason,
			 sets_date_field, sort_order)
		VALUES ($1, $2, $3, $4, $5, NULLIF(btrim($6),''), NULLIF(btrim($7),''), $8, $9)
		RETURNING `+cols,
		tenantID, strings.TrimSpace(in.Name), slugify(in.Name), category, tone,
		in.SetsStatus, in.SetsExitReason, dateField, order))
}

// Update edita o tipo. O slug NAO e recalculado ao renomear: ele e a chave
// estavel do historico (member_history.kind), entao renomear o rotulo nao pode
// orfanar lancamentos antigos.
func (r *Repo) Update(ctx context.Context, tx pgx.Tx, id string, in UpdateInput) (*EventKind, error) {
	var name *string
	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		name = &n
	}
	return scan(tx.QueryRow(ctx, `
		UPDATE member_event_kinds SET
			name             = COALESCE($2, name),
			category         = COALESCE($3, category),
			tone             = COALESCE($4, tone),
			-- string vazia limpa o efeito (NULL).
			sets_status      = CASE WHEN $5::text IS NULL THEN sets_status
			                        WHEN btrim($5::text) = '' THEN NULL
			                        ELSE $5::text END,
			sets_exit_reason = CASE WHEN $6::text IS NULL THEN sets_exit_reason
			                        WHEN btrim($6::text) = '' THEN NULL
			                        ELSE $6::text END,
			sets_date_field  = COALESCE($7, sets_date_field),
			is_active        = COALESCE($8, is_active),
			sort_order       = COALESCE($9, sort_order)
		WHERE id = $1::uuid
		RETURNING `+cols,
		id, name, in.Category, in.Tone, in.SetsStatus, in.SetsExitReason,
		in.SetsDateField, in.IsActive, in.SortOrder))
}

// InUse conta lancamentos do historico vinculados ao tipo.
func (r *Repo) InUse(ctx context.Context, tx pgx.Tx, id string) (int64, error) {
	var n int64
	err := tx.QueryRow(ctx,
		`SELECT count(*) FROM member_history WHERE event_kind_id = $1::uuid`, id).Scan(&n)
	return n, err
}

func (r *Repo) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM member_event_kinds WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// slugify espelha o de cargos/ministries: sem acentos, minusculo, com hifens.
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
		return "evento"
	}
	return string(out)
}
