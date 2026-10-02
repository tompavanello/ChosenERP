package finance

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Contribution e uma entrada financeira ligada ao proprio membro (dizimo/oferta).
// Projecao minima e de leitura: NAO expoe id de recibo nem outros vinculos, para
// o app do membro nao virar uma janela para o financeiro da igreja.
type Contribution struct {
	ID            string  `json:"id"`
	Amount        float64 `json:"amount"`
	CategoryName  *string `json:"category_name,omitempty"`
	PaymentMethod *string `json:"payment_method,omitempty"`
	Description   *string `json:"description,omitempty"`
	OccurredAt    string  `json:"occurred_at"`
}

// ListContributions retorna as contribuicoes (type=income) de um membro em um
// ano, ignorando estornos, mais recentes primeiro.
func (r *Repo) ListContributions(ctx context.Context, tx pgx.Tx, memberID string, year int) ([]Contribution, error) {
	rows, err := tx.Query(ctx, `
		SELECT t.id::text, t.amount::float8, c.name, t.payment_method,
		       t.description, t.occurred_at::text
		FROM financial_transactions t
		LEFT JOIN financial_categories c ON c.id = t.category_id
		WHERE t.donor_member_id = $1::uuid
		  AND t.type = 'income'
		  AND t.voided_at IS NULL
		  AND EXTRACT(YEAR FROM t.occurred_at) = $2
		ORDER BY t.occurred_at DESC, t.created_at DESC`, memberID, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Contribution{}
	for rows.Next() {
		var c Contribution
		if err := rows.Scan(&c.ID, &c.Amount, &c.CategoryName, &c.PaymentMethod,
			&c.Description, &c.OccurredAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
