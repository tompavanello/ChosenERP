package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Bounds define o contexto de segurança do tenant/filial para uma operação.
type Bounds struct {
	TenantID string
	BranchID string // vazio => escopo Sede (is_headquarters)
	Role     string
}

// Store encapsula o pool de conexões da APLICAÇÃO (sujeito a RLS).
// As migrações são executadas com o DSN do migrador (owner do schema).
type Store struct {
	pool       *pgxpool.Pool
	dsn        string
	migrateDSN string
}

func New(ctx context.Context, dsn, migrateDSN string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	return &Store{pool: pool, dsn: dsn, migrateDSN: migrateDSN}, nil
}

func (s *Store) Close() { s.pool.Close() }

// Pool expõe o pool da aplicação para consultas diretas (ex.: login via SECURITY DEFINER).
func (s *Store) Pool() *pgxpool.Pool { return s.pool }

// WithTenant executa fn dentro de uma transação com as variáveis de
// contexto de RLS configuradas (app.tenant_id / app.branch_id / app.role).
// O isolamento multi-tenant é garantido pelo próprio PostgreSQL.
func (s *Store) WithTenant(ctx context.Context, b Bounds, fn func(tx pgx.Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", b.TenantID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('app.branch_id', $1, true)", b.BranchID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('app.role', $1, true)", b.Role); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// IsNotFound converte o erro do pgx em uma flag de "não encontrado".
func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
