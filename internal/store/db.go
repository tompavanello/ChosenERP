package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Bounds define o contexto de seguranca do tenant/filial para uma operacao.
type Bounds struct {
	TenantID string
	BranchID string // vazio => escopo Sede (is_headquarters)
	Role     string
	// UserID e a identidade (users.id) da sessao. Alimenta o GUC app.user_id,
	// que o RLS usa para reconhecer a propria identidade (ex.: seletor de igreja).
	UserID string
}

// Store encapsula o pool de conexoes da APLICACAO (sujeito a RLS).
// As migracoes sao executadas com o DSN do migrador (owner do schema).
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

// Pool expoe o pool da aplicacao para consultas diretas (ex.: login via SECURITY DEFINER).
func (s *Store) Pool() *pgxpool.Pool { return s.pool }

// WithTenant executa fn dentro de uma transacao com as variaveis de
// contexto de RLS configuradas (app.tenant_id / app.branch_id / app.role).
// O isolamento multi-tenant e garantido pelo proprio PostgreSQL.
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
	if _, err := tx.Exec(ctx, "SELECT set_config('app.user_id', $1, true)", b.UserID); err != nil {
		return err
	}
	// app.branch_scope = filial do contexto + descendentes (sub-congregacoes),
	// usada pela leitura hierarquica (rls_read_scope). Sem branch (Sede), fica
	// vazio e o is_headquarters() cobre o tenant inteiro.
	if _, err := tx.Exec(ctx, `
		SELECT set_config('app.branch_scope', COALESCE((
			WITH RECURSIVE tree AS (
				SELECT id FROM branches WHERE id = NULLIF($1,'')::uuid
				UNION ALL
				SELECT b.id FROM branches b JOIN tree t ON b.parent_id = t.id
			)
			SELECT string_agg(id::text, ',') FROM tree
		), ''), true)`, b.BranchID); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// IsNotFound converte o erro do pgx em uma flag de "nao encontrado".
func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// DefaultBranchID resolve a filial "raiz" (Sede Matriz) do tenant. E usada
// quando a sessao e de Sede (branch vazio) mas a operacao grava numa tabela
// cujo branch_id e NOT NULL (financeiro, eventos, ...). Prefere a raiz
// (parent_id NULL) e, na falta dela, a filial mais antiga. Devolve
// pgx.ErrNoRows quando o tenant ainda nao tem nenhuma filial.
func DefaultBranchID(ctx context.Context, tx pgx.Tx, tenantID string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `
		SELECT id::text FROM branches
		WHERE tenant_id = $1::uuid
		ORDER BY (parent_id IS NULL) DESC, created_at ASC
		LIMIT 1`, tenantID).Scan(&id)
	return id, err
}

// WithSystem executa fn dentro de uma transacao com contexto de "sistema/Sede"
// (branch NULL + role 'system'), permitindo ler dados de todas as filiais.
// Usado por workers internos (ex.: worker da outbox de envio).
func (s *Store) WithSystem(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', '', true)"); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('app.branch_id', '', true)"); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('app.role', 'system', true)"); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('app.user_id', '', true)"); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('app.branch_scope', '', true)"); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
