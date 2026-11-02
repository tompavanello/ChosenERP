// Package support gerencia os chamados abertos pela ajuda do produto. O
// chamado nasce dentro do ERP (inbox da plataforma) e pode disparar uma
// notificacao paralela por e-mail/WhatsApp para a equipe de suporte.
package support

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrInvalid indica validacao de entrada.
var ErrInvalid = errors.New("dados do chamado invalidos")

// Tipos aceitos de chamado.
var Kinds = map[string]bool{
	"duvida": true, "problema": true, "sugestao": true, "acesso": true, "outro": true,
}

// Status aceitos.
var Statuses = map[string]bool{
	"aberto": true, "em_andamento": true, "aguardando": true, "resolvido": true, "fechado": true,
}

// Prioridades aceitas.
var Priorities = map[string]bool{"baixa": true, "normal": true, "alta": true, "urgente": true}

// Ticket e um chamado de suporte.
type Ticket struct {
	ID          string          `json:"id"`
	TenantID    string          `json:"tenant_id"`
	BranchID    *string         `json:"branch_id,omitempty"`
	UserID      string          `json:"user_id"`
	UserName    *string         `json:"user_name,omitempty"`
	UserEmail   *string         `json:"user_email,omitempty"`
	Kind        string          `json:"kind"`
	Subject     string          `json:"subject"`
	Description string          `json:"description"`
	App         string          `json:"app"`
	PagePath    string          `json:"page_path"`
	Status      string          `json:"status"`
	Priority    string          `json:"priority"`
	Context     json.RawMessage `json:"context,omitempty"`
	AISummary   *string         `json:"ai_summary,omitempty"`
	AssignedTo  *string         `json:"assigned_to,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	ResolvedAt  *time.Time      `json:"resolved_at,omitempty"`
	Messages    []Message       `json:"messages,omitempty"`
}

// Message e uma mensagem da conversa do chamado.
type Message struct {
	ID         string    `json:"id"`
	TicketID   string    `json:"ticket_id"`
	AuthorID   *string   `json:"author_user_id,omitempty"`
	AuthorKind string    `json:"author_kind"`
	AuthorName *string   `json:"author_name,omitempty"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateInput e o payload de abertura.
type CreateInput struct {
	Kind        string          `json:"kind"`
	Subject     string          `json:"subject"`
	Description string          `json:"description"`
	App         string          `json:"app"`
	PagePath    string          `json:"page_path"`
	Priority    string          `json:"priority"`
	Context     json.RawMessage `json:"context"`
	AISummary   string          `json:"ai_summary"`
	Transcript  json.RawMessage `json:"ai_transcript"`
}

// UpdateInput altera o estado do chamado (inbox da plataforma).
type UpdateInput struct {
	Status     *string `json:"status"`
	Priority   *string `json:"priority"`
	AssignedTo *string `json:"assigned_to"`
}

// Repo executa as operacoes no escopo RLS da transacao recebida.
type Repo struct{}

const ticketCols = `t.id::text, t.tenant_id::text, t.branch_id::text, t.user_id::text,
	u.full_name, u.email, t.kind, t.subject, t.description, t.app, t.page_path,
	t.status, t.priority, t.context, t.ai_summary, t.assigned_to::text,
	t.created_at, t.updated_at, t.resolved_at`

const ticketJoin = `FROM support_tickets t LEFT JOIN users u ON u.id = t.user_id`

func scanTicket(row pgx.Row) (*Ticket, error) {
	var t Ticket
	var ctxRaw []byte
	err := row.Scan(&t.ID, &t.TenantID, &t.BranchID, &t.UserID, &t.UserName, &t.UserEmail,
		&t.Kind, &t.Subject, &t.Description, &t.App, &t.PagePath, &t.Status, &t.Priority,
		&ctxRaw, &t.AISummary, &t.AssignedTo, &t.CreatedAt, &t.UpdatedAt, &t.ResolvedAt)
	if err != nil {
		return nil, err
	}
	if len(ctxRaw) > 0 {
		t.Context = json.RawMessage(ctxRaw)
	}
	return &t, nil
}

// Create abre um chamado. branchID pode ser vazio (Sede/plataforma).
func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID, userID string, in CreateInput) (*Ticket, error) {
	in.Kind = strings.TrimSpace(strings.ToLower(in.Kind))
	if in.Kind == "" {
		in.Kind = "duvida"
	}
	if !Kinds[in.Kind] {
		return nil, ErrInvalid
	}
	in.Subject = strings.TrimSpace(in.Subject)
	in.Description = strings.TrimSpace(in.Description)
	if in.Subject == "" || in.Description == "" {
		return nil, ErrInvalid
	}
	if len([]rune(in.Subject)) > 200 || len([]rune(in.Description)) > 8000 {
		return nil, ErrInvalid
	}
	if in.App != "member" {
		in.App = "webadmin"
	}
	if in.Priority == "" || !Priorities[in.Priority] {
		in.Priority = "normal"
	}
	ctxJSON := in.Context
	if len(ctxJSON) == 0 || !json.Valid(ctxJSON) {
		ctxJSON = json.RawMessage(`{}`)
	}
	var ai *string
	if s := strings.TrimSpace(in.AISummary); s != "" {
		ai = &s
	}
	var transcript any
	if len(in.Transcript) > 0 && json.Valid(in.Transcript) {
		transcript = string(in.Transcript)
	}
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO support_tickets
			(tenant_id, branch_id, user_id, kind, subject, description, app, page_path, priority, context, ai_summary, ai_transcript)
		VALUES ($1::uuid, NULLIF($2,'')::uuid, $3::uuid, $4, $5, $6, $7, $8, $9, $10::jsonb, $11, $12::jsonb)
		RETURNING id::text`,
		tenantID, branchID, userID, in.Kind, in.Subject, in.Description, in.App, in.PagePath,
		in.Priority, string(ctxJSON), ai, transcript).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, tx, id)
}

// ListMine lista os chamados abertos pela identidade (app do membro/webadmin).
func (r *Repo) ListMine(ctx context.Context, tx pgx.Tx, userID string, app string) ([]Ticket, error) {
	q := `SELECT ` + ticketCols + ` ` + ticketJoin + `
		WHERE t.user_id = $1::uuid AND ($2 = '' OR t.app = $2)
		ORDER BY t.created_at DESC`
	rows, err := tx.Query(ctx, q, userID, app)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Ticket{}
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// ListTenant lista os chamados da igreja (Sede) para leitura interna.
func (r *Repo) ListTenant(ctx context.Context, tx pgx.Tx, status string) ([]Ticket, error) {
	rows, err := tx.Query(ctx, `SELECT `+ticketCols+` `+ticketJoin+`
		WHERE ($1 = '' OR t.status = $1)
		ORDER BY t.created_at DESC`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Ticket{}
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// Get carrega o chamado e a conversa (respeitando o RLS da transacao).
func (r *Repo) Get(ctx context.Context, tx pgx.Tx, id string) (*Ticket, error) {
	t, err := scanTicket(tx.QueryRow(ctx, `SELECT `+ticketCols+` `+ticketJoin+` WHERE t.id = $1::uuid`, id))
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `
		SELECT m.id::text, m.ticket_id::text, m.author_user_id::text, m.author_kind,
		       u.full_name, m.body, m.created_at
		FROM support_ticket_messages m
		LEFT JOIN users u ON u.id = m.author_user_id
		WHERE m.ticket_id = $1::uuid
		ORDER BY m.created_at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.TicketID, &m.AuthorID, &m.AuthorKind, &m.AuthorName, &m.Body, &m.CreatedAt); err != nil {
			return nil, err
		}
		t.Messages = append(t.Messages, m)
	}
	return t, rows.Err()
}

// AddMessage adiciona uma mensagem a conversa.
func (r *Repo) AddMessage(ctx context.Context, tx pgx.Tx, ticketID, tenantID, branchID string, authorID *string, authorKind, body string) (*Message, error) {
	body = strings.TrimSpace(body)
	if body == "" || len([]rune(body)) > 8000 {
		return nil, ErrInvalid
	}
	if authorKind != "platform" && authorKind != "ai" {
		authorKind = "user"
	}
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO support_ticket_messages (ticket_id, tenant_id, branch_id, author_user_id, author_kind, body)
		VALUES ($1::uuid, $2::uuid, NULLIF($3,'')::uuid, $4::uuid, $5, $6)
		RETURNING id::text`, ticketID, tenantID, branchID, authorID, authorKind, body).Scan(&id)
	if err != nil {
		return nil, err
	}
	_, _ = tx.Exec(ctx, `UPDATE support_tickets SET updated_at = now() WHERE id = $1::uuid`, ticketID)
	var m Message
	err = tx.QueryRow(ctx, `
		SELECT m.id::text, m.ticket_id::text, m.author_user_id::text, m.author_kind,
		       u.full_name, m.body, m.created_at
		FROM support_ticket_messages m
		LEFT JOIN users u ON u.id = m.author_user_id
		WHERE m.id = $1::uuid`, id).Scan(&m.ID, &m.TicketID, &m.AuthorID, &m.AuthorKind, &m.AuthorName, &m.Body, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Update altera status/prioridade/responsavel.
func (r *Repo) Update(ctx context.Context, tx pgx.Tx, id string, in UpdateInput) (*Ticket, error) {
	if in.Status != nil && !Statuses[*in.Status] {
		return nil, ErrInvalid
	}
	if in.Priority != nil && !Priorities[*in.Priority] {
		return nil, ErrInvalid
	}
	_, err := tx.Exec(ctx, `
		UPDATE support_tickets SET
			status = COALESCE($2, status),
			priority = COALESCE($3, priority),
			assigned_to = CASE WHEN $4 = '' THEN NULL ELSE COALESCE($4::uuid, assigned_to) END,
			resolved_at = CASE WHEN $2 IN ('resolvido','fechado') THEN now()
			                   WHEN $2 IS NOT NULL THEN NULL ELSE resolved_at END,
			updated_at = now()
		WHERE id = $1::uuid`,
		id, in.Status, in.Priority, in.AssignedTo)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, tx, id)
}

// AddFeedback grava o feedback de um artigo (uteis para o backlog de conteudo).
func (r *Repo) AddFeedback(ctx context.Context, tx pgx.Tx, tenantID, branchID, userID, app, slug string, helpful bool, comment string) error {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ErrInvalid
	}
	if len([]rune(comment)) > 2000 {
		comment = string([]rune(comment)[:2000])
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO help_feedback (tenant_id, branch_id, user_id, app, article_slug, helpful, comment)
		VALUES (NULLIF($1,'')::uuid, NULLIF($2,'')::uuid, $3::uuid, $4, $5, $6, NULLIF($7,''))`,
		tenantID, branchID, userID, app, slug, helpful, strings.TrimSpace(comment))
	return err
}
