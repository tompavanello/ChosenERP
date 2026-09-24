package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrClosed indica que a auditoria ja foi fechada (documento imutavel).
var ErrClosed = errors.New("auditoria fechada: documento imutavel")

type Audit struct {
	ID            string     `json:"id"`
	BranchID      string     `json:"branch_id"`
	Title         string     `json:"title"`
	PeriodStart   string     `json:"period_start"`
	PeriodEnd     string     `json:"period_end"`
	Status        string     `json:"status"`
	Notes         *string    `json:"notes,omitempty"`
	ClosedAt      *time.Time `json:"closed_at,omitempty"`
	SignerName    *string    `json:"signer_name,omitempty"`
	SignerRole    *string    `json:"signer_role,omitempty"`
	SignatureHash *string    `json:"signature_hash,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`

	TotalItems    int     `json:"total_items"`
	AuditedItems  int     `json:"audited_items"`
	TotalAmount   float64 `json:"total_amount"`
	AuditedAmount float64 `json:"audited_amount"`
}

type Item struct {
	TransactionID   string       `json:"transaction_id"`
	OccurredAt      time.Time    `json:"occurred_at"`
	Type            string       `json:"type"`
	Amount          float64      `json:"amount"`
	Currency        string       `json:"currency"`
	Description     *string      `json:"description,omitempty"`
	CategoryName    *string      `json:"category_name,omitempty"`
	AccountName     *string      `json:"account_name,omitempty"`
	SupplierName    *string      `json:"supplier_name,omitempty"`
	PaymentMethod   *string      `json:"payment_method,omitempty"`
	IsAnonymous     bool         `json:"is_anonymous"`
	DonorName       *string      `json:"donor_name,omitempty"`
	ReceiptRef      *string      `json:"receipt_ref,omitempty"`
	AttachmentCount int          `json:"attachment_count"`
	Allocations     []Allocation `json:"allocations"`
	Audited         bool         `json:"audited"`
	AuditedAt       *time.Time   `json:"audited_at,omitempty"`
	Notes           *string      `json:"notes,omitempty"`
}

// Allocation e o rateio do lancamento em um evento (custo real por evento).
type Allocation struct {
	EventID   string  `json:"event_id"`
	EventName string  `json:"event_name"`
	Amount    float64 `json:"amount"`
}

type CreateInput struct {
	Title       string  `json:"title"`
	PeriodStart string  `json:"period_start"`
	PeriodEnd   string  `json:"period_end"`
	Notes       *string `json:"notes"`
}

type MarkInput struct {
	// TransactionIDs vazio = marca/desmarca TODOS os lancamentos do periodo.
	TransactionIDs []string `json:"transaction_ids"`
	Audited        bool     `json:"audited"`
}

type CloseInput struct {
	SignerName string `json:"signer_name"`
	SignerRole string `json:"signer_role"`
}

type Repo struct{}

const auditCols = `a.id::text, a.branch_id::text, a.title, a.period_start::text, a.period_end::text,
	a.status, a.notes, a.closed_at, a.signer_name, a.signer_role, a.signature_hash, a.created_at,
	COALESCE(count(i.transaction_id),0)::int,
	COALESCE(count(i.transaction_id) FILTER (WHERE i.audited),0)::int,
	COALESCE(sum(t.amount),0)::float8,
	COALESCE(sum(t.amount) FILTER (WHERE i.audited),0)::float8`

func scanAudit(row pgx.Row) (*Audit, error) {
	var a Audit
	err := row.Scan(&a.ID, &a.BranchID, &a.Title, &a.PeriodStart, &a.PeriodEnd,
		&a.Status, &a.Notes, &a.ClosedAt, &a.SignerName, &a.SignerRole, &a.SignatureHash, &a.CreatedAt,
		&a.TotalItems, &a.AuditedItems, &a.TotalAmount, &a.AuditedAmount)
	return &a, err
}

func (r *Repo) List(ctx context.Context, tx pgx.Tx) ([]Audit, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+auditCols+`
		FROM financial_audits a
		LEFT JOIN financial_audit_items i ON i.audit_id = a.id
		LEFT JOIN financial_transactions t ON t.id = i.transaction_id
		GROUP BY a.id
		ORDER BY a.created_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Audit{}
	for rows.Next() {
		a, err := scanAudit(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *Repo) Get(ctx context.Context, tx pgx.Tx, id string) (*Audit, error) {
	return scanAudit(tx.QueryRow(ctx, `
		SELECT `+auditCols+`
		FROM financial_audits a
		LEFT JOIN financial_audit_items i ON i.audit_id = a.id
		LEFT JOIN financial_transactions t ON t.id = i.transaction_id
		WHERE a.id = $1::uuid
		GROUP BY a.id`, id))
}

// ListItems devolve as linhas do periodo; antes, sincroniza os itens que ainda
// nao estao na auditoria (so enquanto aberta).
func (r *Repo) ListItems(ctx context.Context, tx pgx.Tx, id string) ([]Item, error) {
	if err := r.syncItems(ctx, tx, id); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `
		SELECT i.transaction_id::text, t.occurred_at, t.type, t.amount::float8, t.currency, t.description,
		       c.name, ac.name, s.name, t.payment_method, t.is_anonymous,
		       COALESCE(dm.full_name, db.name) AS donor_name, doc.document_ref,
		       i.audited, i.audited_at, i.notes,
		       (SELECT count(*) FROM financial_attachments fa WHERE fa.transaction_id = t.id)::int
		FROM financial_audit_items i
		JOIN financial_transactions t ON t.id = i.transaction_id
		LEFT JOIN financial_categories c ON c.id = t.category_id
		LEFT JOIN financial_accounts ac ON ac.id = t.account_id
		LEFT JOIN suppliers s ON s.id = t.supplier_id
		LEFT JOIN members dm ON dm.id = t.donor_member_id
		LEFT JOIN benefactors db ON db.id = t.benefactor_id
		LEFT JOIN documents doc ON doc.kind = 'receipt' AND doc.content->>'tx_id' = t.id::text
		WHERE i.audit_id = $1::uuid
		ORDER BY t.occurred_at, t.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Item{}
	byTx := map[string]int{}
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.TransactionID, &it.OccurredAt, &it.Type, &it.Amount, &it.Currency, &it.Description,
			&it.CategoryName, &it.AccountName, &it.SupplierName, &it.PaymentMethod, &it.IsAnonymous,
			&it.DonorName, &it.ReceiptRef, &it.Audited, &it.AuditedAt, &it.Notes,
			&it.AttachmentCount); err != nil {
			return nil, err
		}
		it.Allocations = []Allocation{}
		byTx[it.TransactionID] = len(out)
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Rateio por evento (uma consulta para todos os lancamentos da auditoria).
	allocRows, err := tx.Query(ctx, `
		SELECT a.transaction_id::text, a.event_id::text,
		       COALESCE(to_char(e.starts_at,'DD/MM/YYYY') || ' - ' || k.name, 'Evento'),
		       a.amount::float8
		FROM financial_event_allocations a
		LEFT JOIN church_events e ON e.id = a.event_id
		LEFT JOIN event_kinds k ON k.id = e.kind_id
		WHERE a.transaction_id IN (
		    SELECT transaction_id FROM financial_audit_items WHERE audit_id = $1::uuid)
		ORDER BY a.created_at`, id)
	if err != nil {
		return nil, err
	}
	defer allocRows.Close()
	for allocRows.Next() {
		var txID string
		var al Allocation
		if err := allocRows.Scan(&txID, &al.EventID, &al.EventName, &al.Amount); err != nil {
			return nil, err
		}
		if idx, ok := byTx[txID]; ok {
			out[idx].Allocations = append(out[idx].Allocations, al)
		}
	}
	return out, allocRows.Err()
}

// syncItems materializa os lancamentos do periodo como itens da auditoria.
// O guard do banco recusa quando a auditoria esta fechada.
func (r *Repo) syncItems(ctx context.Context, tx pgx.Tx, id string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO financial_audit_items (audit_id, transaction_id)
		SELECT a.id, t.id
		FROM financial_audits a
		JOIN financial_transactions t
		  ON t.occurred_at::date BETWEEN a.period_start AND a.period_end
		 AND t.voided_at IS NULL
		 AND t.tenant_id = a.tenant_id
		WHERE a.id = $1::uuid AND a.status = 'aberta'
		ON CONFLICT (audit_id, transaction_id) DO NOTHING`, id)
	return err
}

func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID, actorID string, in CreateInput) (*Audit, error) {
	title := in.Title
	if title == "" {
		title = "Auditoria financeira"
	}
	if in.PeriodStart == "" || in.PeriodEnd == "" {
		return nil, errors.New("period_start e period_end sao obrigatorios")
	}
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO financial_audits (tenant_id, branch_id, title, period_start, period_end, notes, created_by)
		VALUES ($1, $2::uuid, $3, $4::date, $5::date, $6, NULLIF($7,'')::uuid)
		RETURNING id::text`,
		tenantID, branchID, title, in.PeriodStart, in.PeriodEnd, in.Notes, actorID).Scan(&id)
	if err != nil {
		return nil, err
	}
	if err := r.syncItems(ctx, tx, id); err != nil {
		return nil, err
	}
	return r.Get(ctx, tx, id)
}

// Mark marca/desmarca os lancamentos (todos quando a lista vem vazia).
func (r *Repo) Mark(ctx context.Context, tx pgx.Tx, id string, in MarkInput, actorID string) error {
	a, err := r.Get(ctx, tx, id)
	if err != nil {
		return err
	}
	if a.Status == "fechada" {
		return ErrClosed
	}
	if err := r.syncItems(ctx, tx, id); err != nil {
		return err
	}
	if len(in.TransactionIDs) == 0 {
		_, err = tx.Exec(ctx, `
			UPDATE financial_audit_items
			SET audited = $2, audited_at = CASE WHEN $2 THEN now() ELSE NULL END,
			    audited_by = CASE WHEN $2 THEN NULLIF($3,'')::uuid ELSE NULL END
			WHERE audit_id = $1::uuid`, id, in.Audited, actorID)
		return err
	}
	_, err = tx.Exec(ctx, `
		UPDATE financial_audit_items
		SET audited = $3, audited_at = CASE WHEN $3 THEN now() ELSE NULL END,
		    audited_by = CASE WHEN $3 THEN NULLIF($4,'')::uuid ELSE NULL END
		WHERE audit_id = $1::uuid AND transaction_id = ANY($2::uuid[])`, id, in.TransactionIDs, in.Audited, actorID)
	return err
}

// Close sela a auditoria com a assinatura do responsavel e o hash do conteudo.
func (r *Repo) Close(ctx context.Context, tx pgx.Tx, id string, in CloseInput, actorID string) (*Audit, error) {
	a, err := r.Get(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if a.Status == "fechada" {
		return nil, ErrClosed
	}
	if in.SignerName == "" {
		return nil, errors.New("signer_name e obrigatorio")
	}
	items, err := r.ListItems(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s|%s|%s\n", a.ID, a.BranchID, a.PeriodStart, a.PeriodEnd, a.Title)
	for _, it := range items {
		fmt.Fprintf(h, "%s|%t|%.2f|%s\n", it.TransactionID, it.Audited, it.Amount, it.OccurredAt.UTC().Format(time.RFC3339))
	}
	sum := hex.EncodeToString(h.Sum(nil))

	_, err = tx.Exec(ctx, `
		UPDATE financial_audits
		SET status = 'fechada', closed_at = now(), closed_by = NULLIF($2,'')::uuid,
		    signer_name = $3, signer_role = NULLIF($4,''), signature_hash = $5, updated_at = now()
		WHERE id = $1::uuid`, id, actorID, in.SignerName, in.SignerRole, sum)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, tx, id)
}

// Delete remove a auditoria (e seus itens, em cascata). Acao de manutencao -
// usada para descartar uma auditoria criada indevidamente. O guard libera o
// cascade mesmo com a auditoria fechada.
func (r *Repo) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM financial_audits WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
