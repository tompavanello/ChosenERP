package finance

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Category é um item do plano de contas (income | expense).
type Category struct {
	ID       string  `json:"id"`
	BranchID *string `json:"branch_id,omitempty"`
	Type     string  `json:"type"`
	Code     string  `json:"code"`
	Name     string  `json:"name"`
}

// Transaction é um lançamento financeiro (append-only).
type Transaction struct {
	ID            string    `json:"id"`
	BranchID      string    `json:"branch_id"`
	CategoryID    *string   `json:"category_id,omitempty"`
	CategoryName  *string   `json:"category_name,omitempty"`
	Type          string    `json:"type"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	PaymentMethod *string   `json:"payment_method,omitempty"`
	IsAnonymous   bool      `json:"is_anonymous"`
	Description   *string   `json:"description,omitempty"`
	ReceiptIssued bool      `json:"receipt_issued"`
	ReceiptID     *string   `json:"receipt_id,omitempty"`
	ReceiptRef    *string   `json:"receipt_ref,omitempty"`
	ReceiptToken  *string   `json:"receipt_token,omitempty"`
	Hash          string    `json:"hash"`
	OccurredAt    time.Time `json:"occurred_at"`
}

type BalanceItem struct {
	CategoryID string  `json:"category_id"`
	Category   string  `json:"category"`
	Type       string  `json:"type"`
	Total      float64 `json:"total"`
}

type Balance struct {
	Income     float64       `json:"income"`
	Expense    float64       `json:"expense"`
	Net        float64       `json:"net"`
	ByCategory []BalanceItem `json:"by_category"`
}

type Repo struct{}

// ListCategories retorna o plano de contas do escopo.
func (r *Repo) ListCategories(ctx context.Context, tx pgx.Tx) ([]Category, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, COALESCE(branch_id::text,''), type, code, name
		FROM financial_categories ORDER BY type, code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		var c Category
		var bid string
		if err := rows.Scan(&c.ID, &bid, &c.Type, &c.Code, &c.Name); err != nil {
			return nil, err
		}
		if bid != "" {
			c.BranchID = &bid
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

type CreateCategoryInput struct {
	Type string `json:"type"`
	Code string `json:"code"`
	Name string `json:"name"`
}

func (r *Repo) CreateCategory(ctx context.Context, tx pgx.Tx, tenantID, branchID string, in CreateCategoryInput) (*Category, error) {
	var c Category
	err := tx.QueryRow(ctx, `
		INSERT INTO financial_categories (tenant_id, branch_id, type, code, name)
		VALUES ($1, NULLIF($2,'')::uuid, $3, $4, $5)
		RETURNING id::text, COALESCE(branch_id::text,''), type, code, name`,
		tenantID, branchID, in.Type, in.Code, in.Name).
		Scan(&c.ID, &c.BranchID, &c.Type, &c.Code, &c.Name)
	return &c, err
}

type CreateTxnInput struct {
	Type          string  `json:"type"`
	Amount        float64 `json:"amount"`
	CategoryID    *string `json:"category_id"`
	PaymentMethod *string `json:"payment_method"`
	IsAnonymous   bool    `json:"is_anonymous"`
	Description   *string `json:"description"`
	DonorMemberID *string `json:"donor_member_id"`
	BenefactorID  *string `json:"benefactor_id"`
	OccurredAt    *string `json:"occurred_at"`
}

// Create insere um lançamento e retorna a transação + ref e token de recibo gerado.
func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID string, in CreateTxnInput) (*Transaction, string, string, error) {
	when := time.Now()
	if in.OccurredAt != nil && *in.OccurredAt != "" {
		if t, err := time.Parse(time.RFC3339, *in.OccurredAt); err == nil {
			when = t
		}
	}
	status := false
	var t Transaction
	err := tx.QueryRow(ctx, `
		INSERT INTO financial_transactions
			(tenant_id, branch_id, category_id, type, amount, currency,
			 payment_method, donor_member_id, benefactor_id, is_anonymous, description, occurred_at)
		VALUES ($1, NULLIF($2,'')::uuid, NULLIF($3,'')::uuid, $4, $5, 'BRL',
			 $6, NULLIF($7,'')::uuid, NULLIF($8,'')::uuid, $9, $10, $11)
		RETURNING id::text, branch_id::text, category_id::text, type, amount::float8,
		          currency, payment_method, is_anonymous, description, receipt_issued, hash, occurred_at`,
		tenantID, branchID, nullIfEmpty(in.CategoryID), in.Type, in.Amount,
		in.PaymentMethod, in.DonorMemberID, in.BenefactorID, in.IsAnonymous, in.Description, when).
		Scan(&t.ID, &t.BranchID, &t.CategoryID, &t.Type, &t.Amount, &t.Currency,
			&t.PaymentMethod, &t.IsAnonymous, &t.Description, &status, &t.Hash, &t.OccurredAt)
	if err != nil {
		return nil, "", "", err
	}
	t.ReceiptIssued = status

	// Gera recibo digital automático (documento).
	ref, token, err := r.issueReceipt(ctx, tx, tenantID, branchID, t)
	if err != nil {
		return nil, "", "", err
	}
	return &t, ref, token, nil
}

// List retorna lançamentos do escopo, com filtro opcional de tipo.
func (r *Repo) List(ctx context.Context, tx pgx.Tx, kind string) ([]Transaction, error) {
	rows, err := tx.Query(ctx, `
		SELECT t.id::text, t.branch_id::text, t.category_id::text, c.name,
		       t.type, t.amount::float8, t.currency, t.payment_method,
		       t.is_anonymous, t.description, t.receipt_issued, t.hash, t.occurred_at,
		       d.id::text, d.document_ref, d.qr_token
		FROM financial_transactions t
		LEFT JOIN financial_categories c ON c.id = t.category_id
		LEFT JOIN documents d ON d.kind = 'receipt' AND d.content->>'tx_id' = t.id::text
		WHERE ($1 = '' OR t.type = $1)
		ORDER BY t.occurred_at DESC LIMIT 200`, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Transaction{}
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.BranchID, &t.CategoryID, &t.CategoryName,
			&t.Type, &t.Amount, &t.Currency, &t.PaymentMethod,
			&t.IsAnonymous, &t.Description, &t.ReceiptIssued, &t.Hash, &t.OccurredAt,
			&t.ReceiptID, &t.ReceiptRef, &t.ReceiptToken); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// SumBalance retorna entradas/saídas agregadas e por categoria (balancete/DRE).
func (r *Repo) SumBalance(ctx context.Context, tx pgx.Tx, kind string) (Balance, error) {
	var b Balance
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount) FILTER (WHERE type='income'),0),
		       COALESCE(SUM(amount) FILTER (WHERE type='expense'),0)
		FROM financial_transactions
		WHERE ($1 = '' OR type = $1)`, kind).
		Scan(&b.Income, &b.Expense)
	if err != nil {
		return b, err
	}
	b.Net = b.Income - b.Expense

	rows, err := tx.Query(ctx, `
		SELECT COALESCE(t.category_id::text,''), COALESCE(c.name,'Sem categoria'), t.type, SUM(t.amount)::float8
		FROM financial_transactions t
		LEFT JOIN financial_categories c ON c.id = t.category_id
		WHERE ($1 = '' OR t.type = $1)
		GROUP BY 1,2,3 ORDER BY 4 DESC`, kind)
	if err != nil {
		return b, err
	}
	defer rows.Close()
	for rows.Next() {
		var it BalanceItem
		if err := rows.Scan(&it.CategoryID, &it.Category, &it.Type, &it.Total); err != nil {
			return b, err
		}
		b.ByCategory = append(b.ByCategory, it)
	}
	return b, rows.Err()
}

func nullIfEmpty(s *string) *string {
	if s != nil && *s == "" {
		return nil
	}
	return s
}
