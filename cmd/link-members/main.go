// Comando one-off que faz o backfill do vinculo identidade <-> membro
// (`memberships.member_id`, migracao 000073).
//
// Nao roda no boot da API: e um utilitario operacional. Uso:
//
//	go run ./cmd/link-members -dry-run
//	go run ./cmd/link-members
//
// DSN (env): MIGRATE_DATABASE_URL (dono/superuser; contorna o RLS). E o mesmo
// usado pelas migracoes; o casamento e feito por e-mail (users.email =
// members.email) dentro do mesmo tenant, porque e a unica chave comum - `users`
// guarda so a identidade (sem CPF).
//
// Serao religados apenas memberships ainda sem vinculo (member_id IS NULL) cujo
// papel (role key) bata com -role (default "membro"): assim um admin que
// compartilhe e-mail com um membro nao e vinculado por engano. Idempotente.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "erro: "+msg)
	os.Exit(1)
}

type pending struct {
	MembershipID string
	TenantID     string
	Email        string
}

func main() {
	dry := flag.Bool("dry-run", false, "apenas simula, sem gravar")
	role := flag.String("role", "membro", "role key dos memberships a vincular")
	flag.Parse()

	dsn := strings.TrimSpace(os.Getenv("MIGRATE_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if dsn == "" {
		fatal("defina MIGRATE_DATABASE_URL (dono do schema)")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fatal("conectando: " + err.Error())
	}
	defer pool.Close()

	rows, err := pool.Query(ctx, `
		SELECT m.id::text, m.tenant_id::text, u.email::text
		FROM memberships m
		JOIN users u ON u.id = m.user_id
		JOIN roles r ON r.id = m.role_id
		WHERE m.member_id IS NULL AND r.key = $1
		ORDER BY m.tenant_id`, *role)
	if err != nil {
		fatal("lendo memberships pendentes: " + err.Error())
	}
	var pends []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.MembershipID, &p.TenantID, &p.Email); err != nil {
			fatal("scan membership: " + err.Error())
		}
		pends = append(pends, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		fatal("iterando memberships: " + err.Error())
	}

	fmt.Printf("memberships sem vinculo (role=%s): %d\n", *role, len(pends))

	var linked int
	var skipped []string
	for _, p := range pends {
		// Candidatos: membros do MESMO tenant com o mesmo e-mail e que ainda nao
		// estejam ligados a outra identidade.
		crows, err := pool.Query(ctx, `
			SELECT mem.id::text
			FROM members mem
			WHERE mem.tenant_id = $1::uuid
			  AND mem.email IS NOT NULL
			  AND mem.email = $2::citext
			  AND NOT EXISTS (
			      SELECT 1 FROM memberships x
			      WHERE x.tenant_id = mem.tenant_id AND x.member_id = mem.id)`,
			p.TenantID, p.Email)
		if err != nil {
			fatal("buscando candidatos: " + err.Error())
		}
		var candidates []string
		for crows.Next() {
			var id string
			if err := crows.Scan(&id); err != nil {
				fatal("scan candidato: " + err.Error())
			}
			candidates = append(candidates, id)
		}
		crows.Close()

		switch len(candidates) {
		case 1:
			if *dry {
				fmt.Printf("  [dry] %s  %s -> membro %s\n", p.TenantID, p.Email, candidates[0])
				linked++
				continue
			}
			if _, err := pool.Exec(ctx,
				`UPDATE memberships SET member_id = $1::uuid, updated_at = now()
				 WHERE id = $2::uuid AND member_id IS NULL`,
				candidates[0], p.MembershipID); err != nil {
				fatal("vinculando: " + err.Error())
			}
			linked++
		case 0:
			skipped = append(skipped, fmt.Sprintf("%s / %s: nenhum membro com este e-mail", p.TenantID, p.Email))
		default:
			skipped = append(skipped, fmt.Sprintf("%s / %s: %d membros com este e-mail (ambiguo)", p.TenantID, p.Email, len(candidates)))
		}
	}

	if *dry {
		fmt.Printf("[dry-run] %d seriam vinculados; nada foi gravado.\n", linked)
	} else {
		fmt.Printf("%d vinculados.\n", linked)
	}
	if len(skipped) > 0 {
		fmt.Printf("Nao vinculados (%d):\n", len(skipped))
		for _, s := range skipped {
			fmt.Println("  - " + s)
		}
	}
}
