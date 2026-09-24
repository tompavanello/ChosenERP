package finance

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5"
)

// issueReceipt gera um recibo digital (documento) para um lançamento e retorna
// o id do documento, ref e token (o id permite enfileirar o envio automático).
func (r *Repo) issueReceipt(ctx context.Context, tx pgx.Tx, tenantID, branchID string, t Transaction) (string, string, string, error) {
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
	var docID string
	err := tx.QueryRow(ctx, `
		INSERT INTO documents (tenant_id, branch_id, kind, title, document_ref, qr_token, content)
		VALUES ($1, NULLIF($2,'')::uuid, 'receipt', $3, $4, $5, $6)
		RETURNING id::text`,
		tenantID, branchID, "Recibo "+ref, ref, token, content).
		Scan(&docID)
	if err != nil {
		return "", "", "", err
	}
	return docID, ref, token, nil
}

// CardRef devolve o número estável da carteirinha de um membro.
//
// A fórmula anterior — "CARD-" + hex.EncodeToString([]byte(memberID))[:6] —
// hex-encodava o TEXTO do UUID e pegava os 6 primeiros caracteres, ou seja, os
// 3 primeiros caracteres do UUID. Todos os membros saíam com o mesmo número
// (CARD-646464 nos fixtures), então o número não identificava ninguém.
// Agora são os 8 primeiros dígitos hex do UUID (32 bits).
func CardRef(memberID string) string {
	h := strings.ToUpper(strings.ReplaceAll(memberID, "-", ""))
	if len(h) > 8 {
		h = h[:8]
	}
	return "CARD-" + h
}

// IssueMembershipCard devolve (ref, token) da carteirinha do membro, criando-a
// se ainda não existir. Idempotente: cliques repetidos devolvem a MESMA
// carteirinha, garantido pelo índice único parcial
// uq_documents_membership_card_per_member (migração 000020).
//
// Por que a CTE em vez de ON CONFLICT ... DO UPDATE: o DO UPDATE reavalia a
// policy WITH CHECK da linha existente, e rls_write é branch exato. Se a
// carteirinha foi criada pela filial A e quem clica é um usuário de escopo Sede
// (branch vazio), o upsert falharia com violação de RLS. A CTE nunca faz UPDATE.
func (r *Repo) IssueMembershipCard(ctx context.Context, tx pgx.Tx, tenantID, branchID, memberID, memberName string) (string, string, error) {
	ref := CardRef(memberID)
	token := randomToken(16)
	content, _ := json.Marshal(map[string]any{
		"kind":      "membership_card",
		"ref":       ref,
		"member_id": memberID,
		"member":    memberName,
		"branch_id": branchID,
		"href":      "/api/v1/documents/by-token/" + token,
	})

	var gotRef, gotToken string
	err := tx.QueryRow(ctx, `
		WITH nova AS (
			INSERT INTO documents (tenant_id, branch_id, kind, title, document_ref,
			                       qr_token, content, member_id)
			VALUES ($1, NULLIF($2,'')::uuid, 'membership_card', $3, $4, $5, $6, $7::uuid)
			ON CONFLICT DO NOTHING
			RETURNING document_ref, qr_token
		)
		SELECT document_ref, qr_token FROM nova
		UNION ALL
		SELECT document_ref, qr_token FROM documents
		 WHERE member_id = $7::uuid AND kind = 'membership_card'
		LIMIT 1`,
		tenantID, branchID, "Carteirinha de Membro", ref, token, content, memberID).
		Scan(&gotRef, &gotToken)
	if err != nil {
		return "", "", err
	}
	return gotRef, gotToken, nil
}

// GetMembershipCard devolve (ref, token) da carteirinha já emitida, sem criar
// nada. Usado por GET /members/{id}/card.
func (r *Repo) GetMembershipCard(ctx context.Context, tx pgx.Tx, memberID string) (string, string, error) {
	var ref, token string
	err := tx.QueryRow(ctx, `
		SELECT document_ref, qr_token FROM documents
		WHERE member_id = $1::uuid AND kind = 'membership_card'
		ORDER BY created_at DESC LIMIT 1`, memberID).Scan(&ref, &token)
	if err != nil {
		return "", "", err
	}
	return ref, token, nil
}

const timeLayout = "2006-01-02T15:04:05Z07:00"

func randomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
