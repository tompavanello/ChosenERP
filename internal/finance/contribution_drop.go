package finance

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// ContributionDrop e um membro que contribuia e parou/reduziu (queda).
type ContributionDrop struct {
	MemberID   string  `json:"member_id"`
	MemberName string  `json:"member_name"`
	LastAt     string  `json:"last_at"`
	Total      float64 `json:"total"`
}

// ContributionDrops lista membros que contribuiram nos ultimos `months` meses,
// mas cuja ULTIMA contribuicao foi ha mais de `recent` meses. Serve de alerta de
// queda/parada de contribuicao, preservando o sigilo (roda no escopo financeiro).
func (r *Repo) ContributionDrops(ctx context.Context, tx pgx.Tx, months, recent int) ([]ContributionDrop, error) {
	if months <= 0 {
		months = 6
	}
	if recent <= 0 {
		recent = 1
	}
	rows, err := tx.Query(ctx, `
		WITH contrib AS (
			SELECT donor_member_id AS member_id,
			       MAX(occurred_at) AS last_at,
			       SUM(amount) AS total
			FROM financial_transactions
			WHERE type = 'income'
			  AND voided_at IS NULL
			  AND donor_member_id IS NOT NULL
			  AND occurred_at >= now() - make_interval(months => $1)
			GROUP BY donor_member_id
		)
		SELECT m.id::text, m.full_name, c.last_at::date::text, c.total::float8
		FROM contrib c
		JOIN members m ON m.id = c.member_id
		WHERE c.last_at < now() - make_interval(months => $2)
		ORDER BY c.last_at`, months, recent)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ContributionDrop{}
	for rows.Next() {
		var d ContributionDrop
		if err := rows.Scan(&d.MemberID, &d.MemberName, &d.LastAt, &d.Total); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
