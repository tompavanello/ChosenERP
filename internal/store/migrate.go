package store

import (
	"context"
	"fmt"
	"sort"

	"chosenerp/db"
	"github.com/jackc/pgx/v5"
)

// migrator aplica as migrações *.up.sql embutidas, em ordem de versão,
// rastreando o que já foi aplicado na tabela schema_migrations.
func (s *Store) Migrate(ctx context.Context) error {
	// Conexão simples (simple protocol) para suportar múltiplos comandos por arquivo.
	conn, err := pgx.Connect(ctx, s.migrateDSN)
	if err != nil {
		return fmt.Errorf("connect for migrate: %w", err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version text PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied := map[string]bool{}
	rows, err := conn.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("query applied migrations: %w", err)
	}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return err
		}
		applied[v] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	files, err := db.Migrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	var ups []string
	for _, f := range files {
		if !f.IsDir() && len(f.Name()) > len(".up.sql") && f.Name()[len(f.Name())-len(".up.sql"):] == ".up.sql" {
			ups = append(ups, f.Name())
		}
	}
	sort.Strings(ups)

	for _, name := range ups {
		version := name[:len(name)-len(".up.sql")]
		if applied[version] {
			continue
		}
		content, err := db.Migrations.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		tx, err := conn.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(content)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES ($1)`, version); err != nil {
			tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}
