package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolConfig define os limites do pool da aplicacao. Valores zero mantem o
// default do pgxpool (max(4, NumCPU)) ou nao aplicam o respectivo timeout.
type PoolConfig struct {
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
	StatementTimeout  time.Duration
	IdleInTxTimeout   time.Duration
	ApplicationName   string
}

// DefaultPoolConfig mantem o comportamento anterior (usado pelos testes e por
// chamadas que nao informam configuracao).
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{ApplicationName: "chosenerp-api"}
}

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
	return NewWithConfig(ctx, dsn, migrateDSN, DefaultPoolConfig())
}

// NewWithConfig cria o pool com limites explicitos. Em producao a API usa
// DB_MAX_CONNS/DB_MIN_CONNS para nao ficar presa no default de 4 conexoes.
func NewWithConfig(ctx context.Context, dsn, migrateDSN string, pc PoolConfig) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}
	if pc.MaxConns > 0 {
		cfg.MaxConns = pc.MaxConns
	}
	if pc.MinConns > 0 {
		cfg.MinConns = pc.MinConns
	}
	if pc.MaxConnLifetime > 0 {
		cfg.MaxConnLifetime = pc.MaxConnLifetime
	}
	if pc.MaxConnIdleTime > 0 {
		cfg.MaxConnIdleTime = pc.MaxConnIdleTime
	}
	if pc.HealthCheckPeriod > 0 {
		cfg.HealthCheckPeriod = pc.HealthCheckPeriod
	}
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	if pc.ApplicationName != "" {
		cfg.ConnConfig.RuntimeParams["application_name"] = pc.ApplicationName
	}
	// timeouts no proprio connection string evitam transacao presa segurando
	// uma conexao do pool. Aplicam-se apenas as conexoes da aplicacao (o
	// migrador usa outra conexao e continua sem limite).
	if pc.StatementTimeout > 0 {
		// O Postgres espera milissegundos (ou unidade explicita). O formato
		// time.Duration.String() ("1m0s") NAO e aceito pelo GUC.
		cfg.ConnConfig.RuntimeParams["statement_timeout"] = strconv.FormatInt(pc.StatementTimeout.Milliseconds(), 10)
	}
	if pc.IdleInTxTimeout > 0 {
		cfg.ConnConfig.RuntimeParams["idle_in_transaction_session_timeout"] = strconv.FormatInt(pc.IdleInTxTimeout.Milliseconds(), 10)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
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

	// Um unico round-trip: os 4 GUCs de contexto + app.branch_scope (filial do
	// contexto + descendentes, usada por rls_read_scope). Sem branch (Sede), o
	// scope fica vazio e is_headquarters() cobre o tenant inteiro.
	if _, err := tx.Exec(ctx, `
		WITH RECURSIVE tree AS (
			SELECT id FROM branches WHERE id = NULLIF($2,'')::uuid
			UNION ALL
			SELECT b.id FROM branches b JOIN tree t ON b.parent_id = t.id
		)
		SELECT set_config('app.tenant_id', $1, true),
		       set_config('app.branch_id', $2, true),
		       set_config('app.role', $3, true),
		       set_config('app.user_id', $4, true),
		       set_config('app.branch_scope',
		                  COALESCE((SELECT string_agg(id::text, ',') FROM tree), ''), true)`,
		b.TenantID, b.BranchID, b.Role, b.UserID); err != nil {
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
