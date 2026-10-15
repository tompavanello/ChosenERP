package finance

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// ErrSplitOver100 sinaliza soma de percentuais ativos acima de 100%.
var ErrSplitOver100 = errors.New("a soma dos percentuais ativos nao pode passar de 100%")

// SplitRule e uma regra de repasse automatico (percentual) para uma filial.
type SplitRule struct {
	ID                  string  `json:"id"`
	DestinationBranchID string  `json:"destination_branch_id"`
	DestinationBranch   string  `json:"destination_branch,omitempty"`
	Name                *string `json:"name,omitempty"`
	Percent             float64 `json:"percent"`
	IsActive            bool    `json:"is_active"`
}

// SplitRuleInput e o corpo de gravacao de uma regra (PUT /finance/split).
type SplitRuleInput struct {
	DestinationBranchID string  `json:"destination_branch_id"`
	Name                *string `json:"name"`
	Percent             float64 `json:"percent"`
	IsActive            *bool   `json:"is_active"`
}

// GetSplit devolve o estado do split (ligado/desligado) e as regras da igreja.
func (r *Repo) GetSplit(ctx context.Context, tx pgx.Tx) (bool, []SplitRule, error) {
	var enabled bool
	if err := tx.QueryRow(ctx, `SELECT split_enabled FROM tenants WHERE id = current_tenant()`).Scan(&enabled); err != nil {
		return false, nil, err
	}
	rows, err := tx.Query(ctx, `
		SELECT tr.id::text, tr.destination_branch_id::text, b.name, tr.name, tr.percent::float8, tr.is_active
		FROM transfer_rules tr
		JOIN branches b ON b.id = tr.destination_branch_id
		ORDER BY b.name`)
	if err != nil {
		return false, nil, err
	}
	defer rows.Close()
	out := []SplitRule{}
	for rows.Next() {
		var s SplitRule
		if err := rows.Scan(&s.ID, &s.DestinationBranchID, &s.DestinationBranch, &s.Name, &s.Percent, &s.IsActive); err != nil {
			return false, nil, err
		}
		out = append(out, s)
	}
	return enabled, out, rows.Err()
}

// SetSplit grava o liga/desliga e SUBSTITUI as regras da igreja (config simples).
func (r *Repo) SetSplit(ctx context.Context, tx pgx.Tx, tenantID string, enabled bool, rules []SplitRuleInput) error {
	total := 0.0
	for _, in := range rules {
		if in.DestinationBranchID == "" || in.Percent <= 0 || in.Percent > 100 {
			continue
		}
		if in.IsActive == nil || *in.IsActive {
			total += in.Percent
		}
	}
	if total > 100.0001 {
		return ErrSplitOver100
	}
	if _, err := tx.Exec(ctx, `
		UPDATE tenants SET split_enabled = $2, updated_at = now() WHERE id = $1::uuid`, tenantID, enabled); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM transfer_rules WHERE tenant_id = $1::uuid`, tenantID); err != nil {
		return err
	}
	for _, in := range rules {
		if in.DestinationBranchID == "" || in.Percent <= 0 || in.Percent > 100 {
			continue
		}
		active := true
		if in.IsActive != nil {
			active = *in.IsActive
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO transfer_rules (tenant_id, destination_branch_id, name, percent, is_active)
			VALUES ($1, $2::uuid, $3, $4, $5)`,
			tenantID, in.DestinationBranchID, in.Name, in.Percent, active); err != nil {
			return err
		}
	}
	return nil
}

// ApplySplit cria os repasses automaticos de uma ENTRADA conforme as regras
// ativas (quando o split esta ligado). Roda dentro da transacao do lancamento,
// entao e aplicado uma unica vez.
//
// O calculo (amount * percent / 100, arredondado a 2 casas) roda em SQL com
// `numeric`, evitando o erro de ponto flutuante que nao fechava a soma (FIN-1).
func (r *Repo) ApplySplit(ctx context.Context, tx pgx.Tx, tenantID, fromBranchID, txID string, _ float64) error {
	var enabled bool
	if err := tx.QueryRow(ctx, `SELECT split_enabled FROM tenants WHERE id = $1::uuid`, tenantID).Scan(&enabled); err != nil {
		if err == pgx.ErrNoRows {
			return nil
		}
		return err
	}
	if !enabled {
		return nil
	}
	_, err := tx.Exec(ctx, `
		WITH calc AS (
			SELECT tr.tenant_id,
			       tr.destination_branch_id AS dest,
			       COALESCE(NULLIF(tr.name,''), 'Split') AS rule_name,
			       ROUND((SELECT amount FROM financial_transactions WHERE id = $4::uuid)
			             * tr.percent / 100, 2) AS amount
			FROM transfer_rules tr
			WHERE tr.tenant_id = $1::uuid AND tr.is_active
			  AND tr.percent > 0
			  AND tr.destination_branch_id IS NOT NULL
			  AND tr.destination_branch_id <> $2::uuid
		)
		INSERT INTO transfers (tenant_id, from_branch_id, to_branch_id, financial_transaction_id, amount, rule_name)
		SELECT tenant_id, $2::uuid, dest, $4::uuid, amount, rule_name
		FROM calc WHERE amount > 0`,
		tenantID, fromBranchID, txID)
	return err
}
