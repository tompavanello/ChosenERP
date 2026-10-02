// Package prayer gerencia os pedidos de oracao do app do membro: criacao,
// mural publico (visibilidade "igreja"), lista dos proprios pedidos, reacao
// anonima "estou orando" e moderacao pastoral. O isolamento multi-tenant e do
// RLS; o filtro fino de visibilidade e aplicado aqui.
package prayer

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrInvalid sinaliza corpo/visibilidade invalidos.
var ErrInvalid = errors.New("dados do pedido invalidos")

var validVisibility = map[string]bool{
	"pastor":          true,
	"pastor_conselho": true,
	"grupo":           true,
	"igreja":          true,
}

var validStatus = map[string]bool{
	"open":     true,
	"answered": true,
	"archived": true,
}

// Request e um pedido de oracao.
type Request struct {
	ID           string    `json:"id"`
	BranchID     string    `json:"branch_id"`
	SmallGroupID *string   `json:"small_group_id,omitempty"`
	AuthorName   *string   `json:"author_name,omitempty"`
	Body         string    `json:"body"`
	Visibility   string    `json:"visibility"`
	IsAnonymous  bool      `json:"is_anonymous"`
	Status       string    `json:"status"`
	AnsweredNote *string   `json:"answered_note,omitempty"`
	PrayerCount  int       `json:"prayer_count"`
	ReactedByMe  bool      `json:"reacted_by_me"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CreateInput e o corpo de criacao.
type CreateInput struct {
	Body         string  `json:"body"`
	Visibility   *string `json:"visibility"`
	IsAnonymous  bool    `json:"is_anonymous"`
	SmallGroupID *string `json:"small_group_id"`
}

type Repo struct{}

const cols = `p.id::text, p.branch_id::text, p.small_group_id::text, p.author_name,
	p.body, p.visibility, p.is_anonymous, p.status, p.answered_note,
	(SELECT count(*) FROM prayer_reactions pr WHERE pr.prayer_request_id = p.id)::int,
	EXISTS (SELECT 1 FROM prayer_reactions pr WHERE pr.prayer_request_id = p.id AND pr.user_id = NULLIF($1,'')::uuid),
	p.created_at, p.updated_at`

func scan(row pgx.Row) (*Request, error) {
	var r Request
	err := row.Scan(&r.ID, &r.BranchID, &r.SmallGroupID, &r.AuthorName, &r.Body,
		&r.Visibility, &r.IsAnonymous, &r.Status, &r.AnsweredNote,
		&r.PrayerCount, &r.ReactedByMe, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	// Nunca expor o nome de um pedido anonimo (vale para mural e lista propria).
	if r.IsAnonymous {
		r.AuthorName = nil
	}
	return &r, nil
}

// Create registra um pedido no escopo da filial do membro.
func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID, memberID, userID, authorName string, in CreateInput) (*Request, error) {
	body := strings.TrimSpace(in.Body)
	if body == "" {
		return nil, ErrInvalid
	}
	vis := "igreja"
	if in.Visibility != nil && strings.TrimSpace(*in.Visibility) != "" {
		vis = strings.TrimSpace(*in.Visibility)
	}
	if !validVisibility[vis] {
		return nil, ErrInvalid
	}
	row := tx.QueryRow(ctx, `
		INSERT INTO prayer_requests
			(tenant_id, branch_id, member_id, small_group_id, author_name, body, visibility, is_anonymous, created_by)
		VALUES ($1::uuid, $2::uuid, NULLIF($3,'')::uuid, NULLIF($4,'')::uuid, NULLIF($5,''), $6, $7, $8, NULLIF($9,'')::uuid)
		RETURNING id::text`,
		tenantID, branchID, memberID, deref(in.SmallGroupID), authorName, body, vis, in.IsAnonymous, userID)
	id := ""
	if err := row.Scan(&id); err != nil {
		return nil, err
	}
	return r.Get(ctx, tx, id, userID)
}

// Get carrega um pedido por id (no escopo RLS), anotando se o viewer ja orou.
func (r *Repo) Get(ctx context.Context, tx pgx.Tx, id, viewerUserID string) (*Request, error) {
	return scan(tx.QueryRow(ctx, `SELECT `+cols+` FROM prayer_requests p WHERE p.id = $2::uuid`,
		viewerUserID, id))
}

// ListMine devolve os pedidos enviados pela propria pessoa.
func (r *Repo) ListMine(ctx context.Context, tx pgx.Tx, memberID, viewerUserID string) ([]Request, error) {
	rows, err := tx.Query(ctx, `SELECT `+cols+`
		FROM prayer_requests p
		WHERE p.member_id = $2::uuid
		ORDER BY p.created_at DESC`, viewerUserID, memberID)
	if err != nil {
		return nil, err
	}
	return collect(rows)
}

// Wall devolve o mural publico: pedidos com visibilidade "igreja" e nao
// arquivados. Os demais escopos sao privados e so aparecem na moderacao.
func (r *Repo) Wall(ctx context.Context, tx pgx.Tx, viewerUserID string) ([]Request, error) {
	rows, err := tx.Query(ctx, `SELECT `+cols+`
		FROM prayer_requests p
		WHERE p.visibility = 'igreja' AND p.status <> 'archived'
		ORDER BY p.created_at DESC`, viewerUserID)
	if err != nil {
		return nil, err
	}
	return collect(rows)
}

// ListModeration devolve os pedidos visiveis a quem modera (todos os escopos),
// com filtro opcional de visibilidade.
func (r *Repo) ListModeration(ctx context.Context, tx pgx.Tx, viewerUserID, visibility string) ([]Request, error) {
	rows, err := tx.Query(ctx, `SELECT `+cols+`
		FROM prayer_requests p
		WHERE ($2 = '' OR p.visibility = $2)
		ORDER BY p.created_at DESC`, viewerUserID, visibility)
	if err != nil {
		return nil, err
	}
	return collect(rows)
}

// React registra "estou orando" (idempotente) e devolve o pedido atualizado.
func (r *Repo) React(ctx context.Context, tx pgx.Tx, tenantID, branchID, requestID, userID, memberID string) (*Request, error) {
	if _, err := tx.Exec(ctx, `
		INSERT INTO prayer_reactions (tenant_id, branch_id, prayer_request_id, user_id, member_id)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, NULLIF($5,'')::uuid)
		ON CONFLICT (prayer_request_id, user_id) DO NOTHING`,
		tenantID, branchID, requestID, userID, memberID); err != nil {
		return nil, err
	}
	return r.Get(ctx, tx, requestID, userID)
}

// UpdateStatus modera o pedido (status e observacao pastoral).
func (r *Repo) UpdateStatus(ctx context.Context, tx pgx.Tx, viewerUserID, id, status, note string) (*Request, error) {
	if !validStatus[status] {
		return nil, ErrInvalid
	}
	tag, err := tx.Exec(ctx, `
		UPDATE prayer_requests
		SET status = $1, answered_note = NULLIF($2,''), updated_at = now()
		WHERE id = $3::uuid`, status, note, id)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	return r.Get(ctx, tx, id, viewerUserID)
}

func collect(rows pgx.Rows) ([]Request, error) {
	defer rows.Close()
	out := []Request{}
	for rows.Next() {
		r, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}
