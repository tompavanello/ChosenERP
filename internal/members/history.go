package members

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// HistoryEntry é um evento do histórico eclesiástico do membro (requisito 1.8).
type HistoryEntry struct {
	ID         string    `json:"id"`
	MemberID   string    `json:"member_id"`
	OccurredAt time.Time `json:"occurred_at"`
	Kind       string    `json:"kind"`
	Notes      *string   `json:"notes,omitempty"`
	CreatedBy  *string   `json:"created_by,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// historyKindForStatus mapeia a nova situação para o tipo de evento do
// histórico. Eventos que não mudam situação são lançados à mão.
func historyKindForStatus(status string) string {
	switch status {
	case "dismissed":
		return "baixa_rol"
	case "transferred":
		return "transferencia"
	case "deceased":
		return "falecimento"
	case "active":
		return "reativacao"
	default:
		return "status"
	}
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// insertHistory grava um evento derivando tenant/branch do próprio membro (o
// INSERT..SELECT garante que o RLS WITH CHECK do histórico passe e evita que o
// chamador precise carregar o contexto). Usado pelos fluxos automáticos.
func insertHistory(ctx context.Context, tx pgx.Tx, memberID, kind, notes, actorID string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO member_history (tenant_id, branch_id, member_id, kind, notes, created_by)
		SELECT m.tenant_id, m.branch_id, m.id, $2, $3, $4::uuid
		FROM members m WHERE m.id = $1::uuid`,
		memberID, kind, nullStr(notes), nullStr(actorID))
	return err
}

// AddHistory insere um evento manual (batismo infantil, recebido por jurisdição,
// etc.). `occurredAt` vazio usa agora.
func (r *Repo) AddHistory(ctx context.Context, tx pgx.Tx, memberID, kind, notes, occurredAt, actorID string) (*HistoryEntry, error) {
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO member_history (tenant_id, branch_id, member_id, kind, notes, occurred_at, created_by)
		SELECT m.tenant_id, m.branch_id, m.id, $2, $3, COALESCE($4::timestamptz, now()), $5::uuid
		FROM members m WHERE m.id = $1::uuid
		RETURNING id::text`,
		memberID, kind, nullStr(notes), nullStr(occurredAt), nullStr(actorID)).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.getHistory(ctx, tx, id)
}

func (r *Repo) getHistory(ctx context.Context, tx pgx.Tx, id string) (*HistoryEntry, error) {
	var h HistoryEntry
	err := tx.QueryRow(ctx, `
		SELECT id::text, member_id::text, occurred_at, kind, notes, created_by::text, created_at
		FROM member_history WHERE id = $1::uuid`, id).
		Scan(&h.ID, &h.MemberID, &h.OccurredAt, &h.Kind, &h.Notes, &h.CreatedBy, &h.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// ListHistory devolve a linha do tempo do membro, do mais recente para o mais
// antigo.
func (r *Repo) ListHistory(ctx context.Context, tx pgx.Tx, memberID string) ([]HistoryEntry, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, member_id::text, occurred_at, kind, notes, created_by::text, created_at
		FROM member_history
		WHERE member_id = $1::uuid
		ORDER BY occurred_at DESC, created_at DESC`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HistoryEntry{}
	for rows.Next() {
		var h HistoryEntry
		if err := rows.Scan(&h.ID, &h.MemberID, &h.OccurredAt, &h.Kind, &h.Notes, &h.CreatedBy, &h.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
