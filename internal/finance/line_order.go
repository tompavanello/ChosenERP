package finance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// LineRef identifica uma linha visivel da sequencia financeira: um lancamento
// avulso ou um agrupamento. O agrupamento ocupa o lugar de todos os seus
// membros na ordem.
type LineRef struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type lineScope struct {
	TenantID  string
	BranchID  string
	AccountID string
	Date      string
}

const emptyUUID = "00000000-0000-0000-0000-000000000000"

func normalizeLineKind(kind string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "transaction", "tx", "lancamento":
		return "transaction", nil
	case "group", "grupo", "agrupamento":
		return "group", nil
	default:
		return "", errors.New("tipo de linha invalido (use transaction ou group)")
	}
}

func lineScopeKey(s lineScope) string {
	return s.TenantID + "|" + s.BranchID + "|" + s.AccountID + "|" + s.Date
}

// nextLineSeq devolve a proxima posicao livre no escopo (filial+conta+data),
// considerando apenas as linhas visiveis (lancamentos avulsos e grupos).
func nextLineSeq(ctx context.Context, tx pgx.Tx, tenantID, branchID, accountID, date string) (int, error) {
	var seq int
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(seq), 0) + 1
		FROM (
		    SELECT entry_seq AS seq
		      FROM financial_transactions
		     WHERE tenant_id = $1::uuid
		       AND branch_id = NULLIF($2,'')::uuid
		       AND COALESCE(account_id, $5::uuid) = COALESCE(NULLIF($3,'')::uuid, $5::uuid)
		       AND (occurred_at AT TIME ZONE 'UTC')::date = $4::date
		       AND group_id IS NULL
		    UNION ALL
		    SELECT entry_seq AS seq
		      FROM financial_transaction_groups
		     WHERE tenant_id = $1::uuid
		       AND branch_id = NULLIF($2,'')::uuid
		       AND account_id = COALESCE(NULLIF($3,'')::uuid, $5::uuid)
		       AND occurred_at = $4::date
		) q`, tenantID, branchID, accountID, date, emptyUUID).Scan(&seq)
	return seq, err
}

// allocateGroupEntrySeq usa a posicao do primeiro membro (MIN) no escopo. Se
// nenhum membro estiver na data do grupo, coloca o grupo no fim da sequencia.
func allocateGroupEntrySeq(ctx context.Context, tx pgx.Tx, ids []string, tenantID, branchID, accountID, date string) (int, error) {
	var min *int
	if err := tx.QueryRow(ctx, `
		SELECT MIN(entry_seq)
		FROM financial_transactions
		WHERE id = ANY($1::uuid[])
		  AND voided_at IS NULL
		  AND (occurred_at AT TIME ZONE 'UTC')::date = $2::date`,
		ids, date).Scan(&min); err != nil {
		return 0, err
	}
	if min != nil {
		return *min, nil
	}
	return nextLineSeq(ctx, tx, tenantID, branchID, accountID, date)
}

// assignFreedLineSeq reposiciona um lancamento que saiu de um grupo no fim da
// sequencia do seu escopo. A sequencia e apenas ordenacao e nao entra no hash.
func assignFreedLineSeq(ctx context.Context, tx pgx.Tx, txID string) error {
	var tenantID, branchID, accountID, date string
	if err := tx.QueryRow(ctx, `
		SELECT tenant_id::text, branch_id::text, COALESCE(account_id::text,''),
		       (occurred_at AT TIME ZONE 'UTC')::date::text
		FROM financial_transactions WHERE id = $1::uuid`, txID).
		Scan(&tenantID, &branchID, &accountID, &date); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		tenantID+"|"+branchID+"|"+accountID+"|"+date); err != nil {
		return err
	}
	seq, err := nextLineSeq(ctx, tx, tenantID, branchID, accountID, date)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE financial_transactions SET entry_seq = $2 WHERE id = $1::uuid`, txID, seq)
	return err
}

// ReorderLines renumera 1..N as linhas visiveis do escopo (lancamentos avulsos
// + grupos) na ordem exata do slice. Todas as linhas do escopo precisam estar
// presentes e sem conciliacao (ou periodo encerrado).
func (r *Repo) ReorderLines(ctx context.Context, tx pgx.Tx, refs []LineRef) error {
	if len(refs) == 0 {
		return nil
	}
	refMap := map[string]struct{}{}
	var scope lineScope
	first := true
	for _, ref := range refs {
		kind, err := normalizeLineKind(ref.Kind)
		if err != nil {
			return err
		}
		id := strings.TrimSpace(ref.ID)
		if id == "" {
			return errors.New("linha sem id")
		}
		key := kind + "|" + id
		if _, dup := refMap[key]; dup {
			return errors.New("linha duplicada na reordenacao")
		}
		refMap[key] = struct{}{}

		var s lineScope
		switch kind {
		case "transaction":
			var groupID *string
			var reconciledAt *time.Time
			var voidedAt *time.Time
			if err := tx.QueryRow(ctx, `
				SELECT tenant_id::text, branch_id::text, COALESCE(account_id::text,''),
				       (occurred_at AT TIME ZONE 'UTC')::date::text,
				       group_id::text, reconciled_at, voided_at
				FROM financial_transactions WHERE id = $1::uuid`, id).
				Scan(&s.TenantID, &s.BranchID, &s.AccountID, &s.Date,
					&groupID, &reconciledAt, &voidedAt); err != nil {
				return err
			}
			if groupID != nil {
				return ErrTransactionGrouped
			}
			if voidedAt != nil {
				return ErrTransactionVoided
			}
			if reconciledAt != nil {
				return ErrTransactionReconciled
			}
		case "group":
			var status string
			if err := tx.QueryRow(ctx, `
				SELECT tenant_id::text, branch_id::text, account_id::text,
				       occurred_at::text, status
				FROM financial_transaction_groups WHERE id = $1::uuid`, id).
				Scan(&s.TenantID, &s.BranchID, &s.AccountID, &s.Date, &status); err != nil {
				return err
			}
			if status == "conciliado" {
				return ErrGroupReconciled
			}
		}

		if first {
			scope = s
			first = false
		} else if s != scope {
			return fmt.Errorf("as linhas devem ser da mesma filial, conta e data")
		}
	}

	if err := assertDateOpen(ctx, tx, scope.TenantID, scope.BranchID, scope.Date); err != nil {
		return err
	}
	// Mesmo lock usado no Create, para nao correr com insercoes/agrupamentos.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		lineScopeKey(scope)); err != nil {
		return err
	}

	rows, err := tx.Query(ctx, `
		SELECT 'transaction', t.id::text
		FROM financial_transactions t
		WHERE t.tenant_id = $1::uuid
		  AND t.branch_id = NULLIF($2,'')::uuid
		  AND COALESCE(t.account_id, $5::uuid) = COALESCE(NULLIF($3,'')::uuid, $5::uuid)
		  AND (t.occurred_at AT TIME ZONE 'UTC')::date = $4::date
		  AND t.group_id IS NULL
		  AND t.voided_at IS NULL
		UNION ALL
		SELECT 'group', g.id::text
		FROM financial_transaction_groups g
		WHERE g.tenant_id = $1::uuid
		  AND g.branch_id = NULLIF($2,'')::uuid
		  AND g.account_id = COALESCE(NULLIF($3,'')::uuid, $5::uuid)
		  AND g.occurred_at = $4::date`,
		scope.TenantID, scope.BranchID, scope.AccountID, scope.Date, emptyUUID)
	if err != nil {
		return err
	}
	count := 0
	for rows.Next() {
		var kind, id string
		if err := rows.Scan(&kind, &id); err != nil {
			rows.Close()
			return err
		}
		if _, ok := refMap[kind+"|"+id]; !ok {
			rows.Close()
			return errors.New("reordene todas as linhas do escopo (filial/conta/data)")
		}
		count++
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if count != len(refs) {
		return errors.New("reordene todas as linhas do escopo (filial/conta/data)")
	}

	for i, ref := range refs {
		kind, _ := normalizeLineKind(ref.Kind)
		id := strings.TrimSpace(ref.ID)
		var err error
		if kind == "group" {
			_, err = tx.Exec(ctx,
				`UPDATE financial_transaction_groups SET entry_seq = $2 WHERE id = $1::uuid`, id, i+1)
		} else {
			_, err = tx.Exec(ctx,
				`UPDATE financial_transactions SET entry_seq = $2 WHERE id = $1::uuid`, id, i+1)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
