// Package governance implementa o Módulo 6 (governança institucional): livro de
// atas digital, votação eletrônica com quórum obrigatório e voto secreto,
// assinatura interna das atas, painel de mandatos e convênios/documentação
// legal. O isolamento multi-tenant é responsabilidade do RLS (transação com
// app.tenant_id/app.branch_id).
package governance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/jackc/pgx/v5"
)

// Repo agrupa as consultas de governança. É stateless: toda operação recebe a
// transação já no escopo de RLS.
type Repo struct{}

// ---- Atas (G1) ----

// Minute é uma ata do livro digital.
type Minute struct {
	ID              string    `json:"id"`
	BranchID        string    `json:"branch_id"`
	Title           string    `json:"title"`
	MeetingAt       time.Time `json:"meeting_at"`
	Kind            string    `json:"kind"`
	Body            *string   `json:"body,omitempty"`
	Status          string    `json:"status"`
	QuorumRequired  int       `json:"quorum_required"`
	AttendanceCount int       `json:"attendance_count"`
	SignatureCount  int       `json:"signature_count"`
	VoteCount       int       `json:"vote_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// MinuteInput é o corpo de criação/edição de uma ata.
type MinuteInput struct {
	Title           string  `json:"title"`
	MeetingAt       string  `json:"meeting_at"`
	Kind            *string `json:"kind"`
	Body            *string `json:"body"`
	Status          *string `json:"status"`
	QuorumRequired  *int    `json:"quorum_required"`
	AttendanceCount *int    `json:"attendance_count"`
}

// Signature é uma assinatura interna de uma ata.
type Signature struct {
	ID           string    `json:"id"`
	MinuteID     string    `json:"minute_id"`
	UserID       *string   `json:"user_id,omitempty"`
	SignerName   string    `json:"signer_name"`
	SignerRole   *string   `json:"signer_role,omitempty"`
	DocumentHash string    `json:"document_hash"`
	SignedAt     time.Time `json:"signed_at"`
	Hash         string    `json:"hash"`
}

const minuteCols = `m.id::text, m.branch_id::text, m.title, m.meeting_at, m.kind, m.body, m.status,
	m.quorum_required, m.attendance_count,
	(SELECT count(*) FROM minute_signatures s WHERE s.minute_id = m.id)::int,
	(SELECT count(*) FROM votes v WHERE v.minute_id = m.id)::int,
	m.created_at, m.updated_at`

func scanMinute(row pgx.Row) (*Minute, error) {
	var m Minute
	err := row.Scan(&m.ID, &m.BranchID, &m.Title, &m.MeetingAt, &m.Kind, &m.Body, &m.Status,
		&m.QuorumRequired, &m.AttendanceCount, &m.SignatureCount, &m.VoteCount,
		&m.CreatedAt, &m.UpdatedAt)
	return &m, err
}

// ListMinutes devolve as atas do escopo, opcionalmente filtrando por período.
func (r *Repo) ListMinutes(ctx context.Context, tx pgx.Tx, from, to string) ([]Minute, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+minuteCols+`
		FROM minutes m
		WHERE ($1 = '' OR m.meeting_at::date >= $1::date)
		  AND ($2 = '' OR m.meeting_at::date <= $2::date)
		ORDER BY m.meeting_at DESC LIMIT 300`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Minute{}
	for rows.Next() {
		m, err := scanMinute(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

// GetMinute busca uma ata pelo id.
func (r *Repo) GetMinute(ctx context.Context, tx pgx.Tx, id string) (*Minute, error) {
	return scanMinute(tx.QueryRow(ctx, `SELECT `+minuteCols+` FROM minutes m WHERE m.id = $1::uuid`, id))
}

// CreateMinute insere uma ata. O status inicial é sempre rascunho (a UI pode
// salvar como aprovada, mas não assinada — assinar é uma operação própria).
func (r *Repo) CreateMinute(ctx context.Context, tx pgx.Tx, tenantID, branchID, actorID string, in MinuteInput) (*Minute, error) {
	meetingAt, err := parseTime(in.MeetingAt)
	if err != nil {
		return nil, err
	}
	kind := "assembleia"
	if in.Kind != nil && *in.Kind != "" {
		kind = *in.Kind
	}
	status := "rascunho"
	if in.Status != nil && (*in.Status == "rascunho" || *in.Status == "aprovada") {
		status = *in.Status
	}
	var newID string
	err = tx.QueryRow(ctx, `
		INSERT INTO minutes (tenant_id, branch_id, title, meeting_at, kind, body, status, quorum_required, attendance_count, created_by)
		VALUES ($1, NULLIF($2,'')::uuid, $3, $4, $5, $6, $7, COALESCE($8,0), COALESCE($9,0), $10::uuid)
		RETURNING id::text`,
		tenantID, branchID, in.Title, meetingAt, kind, in.Body, status,
		in.QuorumRequired, in.AttendanceCount, actorID).Scan(&newID)
	if err != nil {
		return nil, err
	}
	return r.GetMinute(ctx, tx, newID)
}

// UpdateMinute edita a ata. Recusa alteração depois de assinada (o documento
// está congelado pela assinatura).
func (r *Repo) UpdateMinute(ctx context.Context, tx pgx.Tx, id string, in MinuteInput) (*Minute, error) {
	var currentStatus string
	if err := tx.QueryRow(ctx, `SELECT status FROM minutes WHERE id = $1::uuid`, id).Scan(&currentStatus); err != nil {
		return nil, err
	}
	if currentStatus == "assinada" {
		return nil, ErrMinuteSigned
	}
	var meetingAt any
	if in.MeetingAt != "" {
		t, err := parseTime(in.MeetingAt)
		if err != nil {
			return nil, err
		}
		meetingAt = t
	}
	var updatedID string
	err := tx.QueryRow(ctx, `
		UPDATE minutes SET
			title = COALESCE(NULLIF($2,''), title),
			meeting_at = COALESCE($3::timestamptz, meeting_at),
			kind = COALESCE($4, kind),
			body = COALESCE($5, body),
			status = COALESCE($6, status),
			quorum_required = COALESCE($7, quorum_required),
			attendance_count = COALESCE($8, attendance_count),
			updated_at = now()
		WHERE id = $1::uuid
		RETURNING id::text`,
		id, in.Title, meetingAt, in.Kind, in.Body, in.Status, in.QuorumRequired, in.AttendanceCount).Scan(&updatedID)
	if err != nil {
		return nil, err
	}
	return r.GetMinute(ctx, tx, updatedID)
}

// DeleteMinute remove uma ata (e as assinaturas por cascade). Recusa se assinada.
func (r *Repo) DeleteMinute(ctx context.Context, tx pgx.Tx, id string) error {
	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM minutes WHERE id = $1::uuid`, id).Scan(&status); err != nil {
		return err
	}
	if status == "assinada" {
		return ErrMinuteSigned
	}
	_, err := tx.Exec(ctx, `DELETE FROM minutes WHERE id = $1::uuid`, id)
	return err
}

// SignMinute registra a assinatura interna do usuário e congela a ata
// (status = assinada). O document_hash sela o conteúdo atual.
func (r *Repo) SignMinute(ctx context.Context, tx pgx.Tx, id, userID, signerName, signerRole string) (*Signature, error) {
	m, err := r.GetMinute(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	body := ""
	if m.Body != nil {
		body = *m.Body
	}
	sum := sha256.Sum256([]byte(id + "|" + m.Title + "|" + body + "|" + m.MeetingAt.Format(time.RFC3339)))
	docHash := hex.EncodeToString(sum[:])

	var newID string
	err = tx.QueryRow(ctx, `
		INSERT INTO minute_signatures (tenant_id, branch_id, minute_id, user_id, signer_name, signer_role, document_hash)
		SELECT m.tenant_id, m.branch_id, m.id, $2::uuid, $3, NULLIF($4,''), $5
		FROM minutes m WHERE m.id = $1::uuid
		RETURNING id::text`,
		id, userID, signerName, signerRole, docHash).Scan(&newID)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE minutes SET status = 'assinada', updated_at = now() WHERE id = $1::uuid`, id); err != nil {
		return nil, err
	}
	return r.GetSignature(ctx, tx, newID)
}

// GetSignature busca uma assinatura pelo id.
func (r *Repo) GetSignature(ctx context.Context, tx pgx.Tx, id string) (*Signature, error) {
	var s Signature
	err := tx.QueryRow(ctx, `
		SELECT id::text, minute_id::text, user_id::text, signer_name, signer_role, document_hash, signed_at, hash
		FROM minute_signatures WHERE id = $1::uuid`, id).
		Scan(&s.ID, &s.MinuteID, &s.UserID, &s.SignerName, &s.SignerRole, &s.DocumentHash, &s.SignedAt, &s.Hash)
	return &s, err
}

// ListSignatures devolve as assinaturas de uma ata, na ordem da cadeia.
func (r *Repo) ListSignatures(ctx context.Context, tx pgx.Tx, minuteID string) ([]Signature, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, minute_id::text, user_id::text, signer_name, signer_role, document_hash, signed_at, hash
		FROM minute_signatures WHERE minute_id = $1::uuid ORDER BY signed_at, id`, minuteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Signature{}
	for rows.Next() {
		var s Signature
		if err := rows.Scan(&s.ID, &s.MinuteID, &s.UserID, &s.SignerName, &s.SignerRole, &s.DocumentHash, &s.SignedAt, &s.Hash); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// appendMinuteResult acrescenta um bloco de texto ao corpo da ata (usado pela
// apuração automática ao encerrar uma votação vinculada).
func (r *Repo) appendMinuteResult(ctx context.Context, tx pgx.Tx, minuteID, block string) error {
	_, err := tx.Exec(ctx, `
		UPDATE minutes SET body = COALESCE(body, '') || $2, updated_at = now()
		WHERE id = $1::uuid AND status <> 'assinada'`, minuteID, block)
	return err
}

func parseTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02T15:04", s)
}
