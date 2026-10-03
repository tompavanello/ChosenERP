// Package requests gerencia as solicitacoes/formularios do app do membro:
// o membro cria e acompanha; a secretaria responde (status + observacao).
package requests

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrInvalidKind / ErrInvalidStatus marcam validacao de entrada.
var (
	ErrInvalidKind   = errors.New("tipo de solicitacao invalido")
	ErrInvalidStatus = errors.New("status invalido")
)

// Kinds sao os tipos de solicitacao aceitos.
var Kinds = map[string]bool{
	"cadastro": true, "carta": true, "visita": true, "batismo": true,
	"profissao_fe": true, "transferencia": true, "casamento": true,
	"evento": true, "outro": true,
}

// Request e uma solicitacao do membro.
type Request struct {
	ID          string     `json:"id"`
	MemberID    *string    `json:"member_id,omitempty"`
	MemberName  *string    `json:"member_name,omitempty"`
	Kind        string     `json:"kind"`
	Subject     string     `json:"subject"`
	Message     *string    `json:"message,omitempty"`
	Status      string     `json:"status"`
	Response    *string    `json:"response,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	RespondedAt *time.Time `json:"responded_at,omitempty"`
}

type CreateInput struct {
	Kind    string  `json:"kind"`
	Subject string  `json:"subject"`
	Message *string `json:"message"`
}

type UpdateInput struct {
	Status   string  `json:"status"`
	Response *string `json:"response"`
}

type Repo struct{}

const cols = `r.id::text, r.member_id::text, mb.full_name, r.kind, r.subject, r.message,
	r.status, r.response, r.created_at, r.updated_at, r.responded_at`

func scan(row pgx.Row) (*Request, error) {
	var r Request
	err := row.Scan(&r.ID, &r.MemberID, &r.MemberName, &r.Kind, &r.Subject, &r.Message,
		&r.Status, &r.Response, &r.CreatedAt, &r.UpdatedAt, &r.RespondedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

const fromJoin = `FROM member_requests r LEFT JOIN members mb ON mb.id = r.member_id`

// List lista as solicitacoes do escopo (admin/secretaria), com filtro de status.
func (r *Repo) List(ctx context.Context, tx pgx.Tx, status string) ([]Request, error) {
	rows, err := tx.Query(ctx, `SELECT `+cols+` `+fromJoin+`
		WHERE ($1 = '' OR r.status = $1)
		ORDER BY r.created_at DESC`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Request{}
	for rows.Next() {
		it, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *it)
	}
	return out, rows.Err()
}

// ListForMember lista as solicitacoes do proprio membro.
func (r *Repo) ListForMember(ctx context.Context, tx pgx.Tx, memberID string) ([]Request, error) {
	rows, err := tx.Query(ctx, `SELECT `+cols+` `+fromJoin+`
		WHERE r.member_id = $1::uuid
		ORDER BY r.created_at DESC`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Request{}
	for rows.Next() {
		it, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *it)
	}
	return out, rows.Err()
}

// Get devolve uma solicitacao pelo id.
func (r *Repo) Get(ctx context.Context, tx pgx.Tx, id string) (*Request, error) {
	return scan(tx.QueryRow(ctx, `SELECT `+cols+` `+fromJoin+` WHERE r.id = $1::uuid`, id))
}

// Create insere uma solicitacao do membro.
func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID, memberID, userID string, in CreateInput) (*Request, error) {
	if !Kinds[in.Kind] {
		return nil, ErrInvalidKind
	}
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO member_requests (tenant_id, branch_id, member_id, user_id, kind, subject, message)
		VALUES ($1, $2::uuid, NULLIF($3,'')::uuid, NULLIF($4,'')::uuid, $5, $6, $7)
		RETURNING id::text`,
		tenantID, branchID, memberID, userID, in.Kind, in.Subject, in.Message).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, tx, id)
}

// Update responde a solicitacao (status + observacao).
func (r *Repo) Update(ctx context.Context, tx pgx.Tx, id string, in UpdateInput, responderID string) (*Request, error) {
	switch in.Status {
	case "pendente", "em_andamento", "concluida", "cancelada":
	default:
		return nil, ErrInvalidStatus
	}
	tag, err := tx.Exec(ctx, `
		UPDATE member_requests SET
			status = $2,
			response = COALESCE($3, response),
			responded_by = NULLIF($4,'')::uuid,
			responded_at = CASE WHEN $2 IN ('concluida','cancelada') THEN now() ELSE responded_at END,
			updated_at = now()
		WHERE id = $1::uuid`, id, in.Status, in.Response, responderID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	return r.Get(ctx, tx, id)
}
