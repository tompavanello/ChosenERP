package finance

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Account e uma conta bancaria ou caixa do tenant/filial.
type Account struct {
	ID              string     `json:"id"`
	BranchID        *string    `json:"branch_id,omitempty"`
	Name            string     `json:"name"`
	Bank            *string    `json:"bank,omitempty"`
	BankCode        *string    `json:"bank_code,omitempty"`
	Agency          *string    `json:"agency,omitempty"`
	AccountNumber   *string    `json:"account_number,omitempty"`
	AccountType     string     `json:"account_type"` // checking | savings | cash
	InitialBalance  float64    `json:"initial_balance"`
	IsActive        bool       `json:"is_active"`
	CreatedAt       time.Time  `json:"created_at"`
}

// CreateAccountInput e o payload para criar uma conta bancaria.
type CreateAccountInput struct {
	BranchID      *string `json:"branch_id,omitempty"`
	Name           string  `json:"name"`
	Bank           *string `json:"bank,omitempty"`
	BankCode       *string `json:"bank_code,omitempty"`
	Agency         *string `json:"agency,omitempty"`
	AccountNumber  *string `json:"account_number,omitempty"`
	AccountType    string  `json:"account_type"`     // checking | savings | cash
	InitialBalance float64 `json:"initial_balance"`
	IsActive       *bool   `json:"is_active,omitempty"`
}

// ListAccounts retorna as contas visiveis no escopo (tenant + branch ou Sede).
func (r *Repo) ListAccounts(ctx context.Context, tx pgx.Tx) ([]Account, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, COALESCE(branch_id::text,''), name, bank, bank_code,
		       agency, account_number, account_type, initial_balance, is_active, created_at
		FROM financial_accounts ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Account{}
	for rows.Next() {
		var a Account
		var bid string
		if err := rows.Scan(&a.ID, &bid, &a.Name, &a.Bank, &a.BankCode,
			&a.Agency, &a.AccountNumber, &a.AccountType, &a.InitialBalance,
			&a.IsActive, &a.CreatedAt); err != nil {
			return nil, err
		}
		if bid != "" {
			a.BranchID = &bid
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CreateAccount insere uma conta bancaria no escopo.
func (r *Repo) CreateAccount(ctx context.Context, tx pgx.Tx, tenantID, branchID string, in CreateAccountInput) (*Account, error) {
	var a Account
	query := `
		INSERT INTO financial_accounts
			(tenant_id, branch_id, name, bank, bank_code, agency,
			 account_number, account_type, initial_balance, is_active)
		VALUES ($1, NULLIF($2,'')::uuid, $3, $4, $5, $6, $7, $8, $9, COALESCE($10, true))
		RETURNING id::text, COALESCE(branch_id::text,''), name, bank, bank_code,
		          agency, account_number, account_type, initial_balance, is_active, created_at`
	err := tx.QueryRow(ctx, query,
		tenantID, branchID, in.Name, in.Bank, in.BankCode, in.Agency,
		in.AccountNumber, in.AccountType, in.InitialBalance, in.IsActive).
		Scan(&a.ID, &a.BranchID, &a.Name, &a.Bank, &a.BankCode,
			&a.Agency, &a.AccountNumber, &a.AccountType, &a.InitialBalance,
			&a.IsActive, &a.CreatedAt)
	return &a, err
}

// UpdateAccount atualiza os dados de uma conta (PATCH: campo nil mantem).
func (r *Repo) UpdateAccount(ctx context.Context, tx pgx.Tx, id string, in UpdateAccountInput) (*Account, error) {
	var a Account
	err := tx.QueryRow(ctx, `
		UPDATE financial_accounts
		SET name = COALESCE($2, name),
		    bank = COALESCE($3, bank),
		    bank_code = COALESCE($4, bank_code),
		    agency = COALESCE($5, agency),
		    account_number = COALESCE($6, account_number),
		    account_type = COALESCE($7, account_type),
		    initial_balance = COALESCE($8, initial_balance),
		    is_active = COALESCE($9, is_active)
		WHERE id = $1::uuid
		RETURNING id::text, COALESCE(branch_id::text,''), name, bank, bank_code,
		          agency, account_number, account_type, initial_balance, is_active, created_at`,
		id, in.Name, in.Bank, in.BankCode, in.Agency, in.AccountNumber,
		in.AccountType, in.InitialBalance, in.IsActive).
		Scan(&a.ID, &a.BranchID, &a.Name, &a.Bank, &a.BankCode,
			&a.Agency, &a.AccountNumber, &a.AccountType, &a.InitialBalance,
			&a.IsActive, &a.CreatedAt)
	return &a, err
}

// ErrAccountInUse indica que a conta tem lancamentos/recorrencias vinculados.
var ErrAccountInUse = errors.New("conta em uso: desative em vez de excluir")

// DeleteAccount exclui uma conta sem uso (senao devolve ErrAccountInUse).
func (r *Repo) DeleteAccount(ctx context.Context, tx pgx.Tx, id string) error {
	var used int
	if err := tx.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM financial_transactions WHERE account_id = $1::uuid)
		     + (SELECT count(*) FROM recurring_donations WHERE account_id = $1::uuid)`, id).Scan(&used); err != nil {
		return err
	}
	if used > 0 {
		return ErrAccountInUse
	}
	tag, err := tx.Exec(ctx, `DELETE FROM financial_accounts WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// UpdateAccountInput e o payload para atualizar uma conta bancaria.
type UpdateAccountInput struct {
	Name           *string  `json:"name,omitempty"`
	Bank           *string  `json:"bank,omitempty"`
	BankCode       *string  `json:"bank_code,omitempty"`
	Agency         *string  `json:"agency,omitempty"`
	AccountNumber  *string  `json:"account_number,omitempty"`
	AccountType    *string  `json:"account_type,omitempty"`
	InitialBalance *float64 `json:"initial_balance,omitempty"`
	IsActive       *bool    `json:"is_active,omitempty"`
}

// GetAccount retorna uma conta pelo id dentro do escopo atual.
func (r *Repo) GetAccount(ctx context.Context, tx pgx.Tx, id string) (*Account, error) {
	var a Account
	err := tx.QueryRow(ctx, `
		SELECT id::text, COALESCE(branch_id::text,''), name, bank, bank_code,
		       agency, account_number, account_type, initial_balance, is_active, created_at
		FROM financial_accounts WHERE id = $1::uuid`, id).
		Scan(&a.ID, &a.BranchID, &a.Name, &a.Bank, &a.BankCode,
			&a.Agency, &a.AccountNumber, &a.AccountType, &a.InitialBalance,
			&a.IsActive, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}
