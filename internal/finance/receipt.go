package finance

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

// issueReceipt gera um recibo digital (documento) para um lançamento e retorna ref + token.
func (r *Repo) issueReceipt(ctx context.Context, tx pgx.Tx, tenantID, branchID string, t Transaction) (string, string, error) {
	ref := "REC-" + hex.EncodeToString([]byte(t.Hash))[:8]
	token := randomToken(12)

	content, _ := json.Marshal(map[string]any{
		"kind":        "receipt",
		"ref":         ref,
		"tx_id":       t.ID,
		"branch_id":   t.BranchID,
		"type":        t.Type,
		"amount":      t.Amount,
		"currency":    t.Currency,
		"occurred_at": t.OccurredAt.Format(timeLayout),
		"href":        "/api/v1/documents/by-token/" + token,
	})
	_, err := tx.Exec(ctx, `
		INSERT INTO documents (tenant_id, branch_id, kind, title, document_ref, qr_token, content)
		VALUES ($1, NULLIF($2,'')::uuid, 'receipt', $3, $4, $5, $6)`,
		tenantID, branchID, "Recibo "+ref, ref, token, content)
	if err != nil {
		return "", "", err
	}
	return ref, token, nil
}

func (r *Repo) NewMembershipCard(ctx context.Context, tx pgx.Tx, tenantID, branchID, memberID, memberName string) (string, error) {
	token := randomToken(16)
	ref := "CARD-" + hex.EncodeToString([]byte(memberID))[:6]
	content, _ := json.Marshal(map[string]any{
		"kind":      "membership_card",
		"ref":       ref,
		"member_id": memberID,
		"member":    memberName,
		"branch_id": branchID,
		"href":      "/api/v1/documents/by-token/" + token,
	})
	_, err := tx.Exec(ctx, `
		INSERT INTO documents (tenant_id, branch_id, kind, title, document_ref, qr_token, content, member_id)
		VALUES ($1, NULLIF($2,'')::uuid, 'membership_card', $3, $4, $5, $6, $7::uuid)`,
		tenantID, branchID, "Carteirinha de Membro", ref, token, content, memberID)
	return ref, err
}

const timeLayout = "2006-01-02T15:04:05Z07:00"

func randomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
