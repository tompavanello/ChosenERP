package kids

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type rowScanner interface{ Scan(dest ...any) error }

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// validateAges garante faixa etaria coerente (0..17, min <= max). Idade nula e
// permitida (turma/trilha sem faixa definida).
func validateAges(min, max *int) error {
	if min != nil && *min < 0 {
		return fmt.Errorf("%w: age_min nao pode ser negativo", ErrInvalidInput)
	}
	if max != nil && *max < 0 {
		return fmt.Errorf("%w: age_max nao pode ser negativo", ErrInvalidInput)
	}
	if min != nil && max != nil && *max < *min {
		return fmt.Errorf("%w: age_max deve ser >= age_min", ErrInvalidInput)
	}
	return nil
}

// execExpectRow executa um comando e devolve pgx.ErrNoRows quando nada foi
// afetado (usado para DELETE de recurso inexistente).
func execExpectRow(ctx context.Context, tx pgx.Tx, sql string, args ...any) error {
	tag, err := tx.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
