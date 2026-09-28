// Comando one-off que importa o historico eclesiastico do sistema legado
// MCM/IPI (tabela cadcre) para o ChosenERP (member_history).
//
// Nao roda no boot da API: e um utilitario operacional, como o resto do
// db/seed. Uso:
//
//	GOOS=windows go run ./cmd/import-legacy-history -dry-run
//	go run ./cmd/import-legacy-history
//
// DSNs (env):
//
//	LEGACY_DATABASE_URL  origem  (default postgres://postgres:sinc@127.0.0.1:5432/ipi)
//	MIGRATE_DATABASE_URL destino (dono/superuser; RLS e append-only contornados)
//
// O casamento do membro e por members.external_id = 'sincad:<cpf legado>' e,
// quando nao existe, por CPF normalizado. A insercao e IDEMPOTENTE (mesmo
// membro + kind + occurred_at nao duplica), entao pode rodar de novo.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// eventoSlug mapeia o codigo do cadorg legado para o slug canonico do catalogo
// (migracao 000065).
var eventoSlug = map[int]string{
	1:  "batismo_profissao_fe",
	2:  "profissao_fe",
	3:  "batismo_crianca",
	4:  "apresentacao_crianca",
	5:  "recebido_transferencia",
	6:  "recebido_jurisdicao",
	7:  "saida_transferencia",
	8:  "saida_falecimento",
	9:  "desligamento_disciplinar",
	10: "desligamento_pedido",
	11: "ordenacao_diacono",
	12: "ordenacao_presbitero",
	13: "dissolucao_pastoral",
	14: "decisao_presbiterio",
	15: "abandono_atividades",
}

// codorgaDesc e so para o relatorio (codigo desconhecido -> descricao).
var eventoDesc = map[int]string{
	1: "BATISMO E PROFISSAO DE FE", 2: "PROFISSAO DE FE", 3: "BATISMO DE CRIANCA",
	4: "APRESENTACAO RECEM NASCIDO", 5: "RECEBIMENTO POR TRANSFERENCIA",
	6: "RECEBIMENTO POR JURISDICAO", 7: "SAIDA POR TRANSFERENCIA",
	8: "SAIDA POR FALECIMENTO", 9: "DESLIGAMENTO ATO DISCIPLINAR",
	10: "DESLIGAMENTO PEDIDO DO MEMBRO", 11: "ORDENACAO DE DIACONO(A)",
	12: "ORDENACAO DE PRESBITERO(A)", 13: "DISSOLUCAO DA RELACOES PASTORAIS",
	14: "POR DECISAO DO PRESBITORIO", 15: "ABANDONO ATIV. ECLESIAST. POR MAIS 1 ANO",
}

type legacyRow struct {
	CPF       string
	EventDate time.Time
	HourMin   int
	Codorga   int
	User      string
	Notes     string
}

var nonDigits = regexp.MustCompile(`\D+`)

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func main() {
	dry := flag.Bool("dry-run", false, "apenas simula, sem gravar no destino")
	flag.Parse()

	srcDSN := envOr("LEGACY_DATABASE_URL", "postgres://postgres:sinc@127.0.0.1:5432/ipi")
	dstDSN := envOr("MIGRATE_DATABASE_URL", envOr("DATABASE_URL", ""))
	if dstDSN == "" && !*dry {
		fatal("defina MIGRATE_DATABASE_URL (dono do schema do ChosenERP)")
	}

	ctx := context.Background()
	// O banco legado (MCM/IPI) costuma estar em LATIN1: forca o cliente a
	// pedir UTF8 para o Postgres converter os acentos na saida (senao os bytes
	// 0xc9 chegam crus e o INSERT no destino UTF8 recusa).
	srcCfg, err := pgxpool.ParseConfig(srcDSN)
	if err != nil {
		fatal("DSN da origem invalido: " + err.Error())
	}
	srcCfg.ConnConfig.RuntimeParams["client_encoding"] = "UTF8"
	src, err := pgxpool.NewWithConfig(ctx, srcCfg)
	if err != nil {
		fatal("conectando na origem: " + err.Error())
	}
	defer src.Close()

	rows, err := loadLegacy(ctx, src)
	if err != nil {
		fatal("lendo cadcre: " + err.Error())
	}
	fmt.Printf("Legado: %d lancamentos em cadcre\n", len(rows))

	if *dry {
		for _, r := range rows {
			fmt.Printf("  cpf=%-20s %s  %-28s notas=%d chars\n",
				r.CPF, r.occurredAt().Format("2006-01-02 15:04"),
				eventoDesc[r.Codorga], len(r.Notes))
		}
		fmt.Println("[dry-run] nada foi gravado.")
		return
	}

	dst, err := pgxpool.New(ctx, dstDSN)
	if err != nil {
		fatal("conectando no destino: " + err.Error())
	}
	defer dst.Close()

	var inserted, duplicated int
	var unmatched []string
	for _, r := range rows {
		memberID, err := findMember(ctx, dst, r.CPF)
		if err != nil {
			fatal("buscando membro: " + err.Error())
		}
		if memberID == "" {
			unmatched = append(unmatched, r.CPF)
			continue
		}
		slug := eventoSlug[r.Codorga]
		if slug == "" {
			slug = "outro"
		}
		tag, err := dst.Exec(ctx, `
			INSERT INTO member_history
				(tenant_id, branch_id, member_id, kind, event_kind_id, notes, occurred_at)
			SELECT m.tenant_id, m.branch_id, m.id, $2,
			       (SELECT id FROM member_event_kinds WHERE tenant_id = m.tenant_id AND slug = $2),
			       NULLIF($3,''), $4::timestamptz
			FROM members m
			WHERE m.id = $1::uuid
			  AND NOT EXISTS (
			      SELECT 1 FROM member_history h
			      WHERE h.member_id = m.id AND h.kind = $2
			        AND h.occurred_at = $4::timestamptz)`,
			memberID, slug, r.Notes, r.occurredAt())
		if err != nil {
			fatal(fmt.Sprintf("inserindo historico do cpf %s: %v", r.CPF, err))
		}
		if tag.RowsAffected() > 0 {
			inserted++
		} else {
			duplicated++
		}
	}

	fmt.Printf("Importados: %d | ja existiam (idempotente): %d | sem membro: %d\n",
		inserted, duplicated, len(unmatched))
	if len(unmatched) > 0 {
		fmt.Println("CPFs sem membro correspondente (revisar manualmente):")
		for _, c := range unmatched {
			fmt.Println("  -", c)
		}
	}
}

func (r legacyRow) occurredAt() time.Time {
	d := r.EventDate
	hh, mm := 12, 0
	if r.HourMin > 0 && r.HourMin < 2400 {
		hh, mm = r.HourMin/100, r.HourMin%100
		if mm > 59 {
			mm = 0
		}
	}
	return time.Date(d.Year(), d.Month(), d.Day(), hh, mm, 0, 0, time.UTC)
}

func loadLegacy(ctx context.Context, pool *pgxpool.Pool) ([]legacyRow, error) {
	rows, err := pool.Query(ctx, `
		SELECT dcre_cgcocpf_1::text,
		       COALESCE(dcre_datacon_1, dcre_datainv_1),
		       COALESCE(dcre_horacon_1, 0),
		       COALESCE(dcre_codorga_1, 0),
		       COALESCE(dcre_coduser_1, ''),
		       btrim(COALESCE(dcre_txtobso_1_o1,'') || E'\n' || COALESCE(dcre_txtobso_1_o2,'') || E'\n' ||
		             COALESCE(dcre_txtobso_1_o3,'') || E'\n' || COALESCE(dcre_txtobso_1_o4,'') || E'\n' ||
		             COALESCE(dcre_txtobso_1_o5,'') || E'\n' || COALESCE(dcre_txtobso_1_o6,'') || E'\n' ||
		             COALESCE(dcre_txtobso_1_o7,'') || E'\n' || COALESCE(dcre_txtobso_1_o8,'') || E'\n' ||
		       COALESCE(dcre_observa_1_o1,'') || E'\n' || COALESCE(dcre_observa_1_o2,'') || E'\n' ||
		             COALESCE(dcre_observa_1_o3,''))
		FROM cadcre
		ORDER BY COALESCE(dcre_datacon_1, dcre_datainv_1), COALESCE(dcre_horacon_1,0)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []legacyRow
	for rows.Next() {
		var r legacyRow
		var dt time.Time
		if err := rows.Scan(&r.CPF, &dt, &r.HourMin, &r.Codorga, &r.User, &r.Notes); err != nil {
			return nil, err
		}
		r.EventDate = dt
		// Limpa linhas vazias do texto concatenado.
		var lines []string
		for _, ln := range strings.Split(r.Notes, "\n") {
			if s := strings.TrimSpace(ln); s != "" {
				lines = append(lines, s)
			}
		}
		r.Notes = strings.Join(lines, "\n")
		if strings.TrimSpace(r.User) != "" {
			r.Notes = strings.TrimSpace(r.Notes + "\n(registrado no sistema legado por " + strings.TrimSpace(r.User) + ")")
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// findMember tenta pelo external_id do importador e cai para o CPF normalizado.
func findMember(ctx context.Context, pool *pgxpool.Pool, legacyCPF string) (string, error) {
	var id string
	err := pool.QueryRow(ctx,
		`SELECT id::text FROM members WHERE external_id = $1 LIMIT 1`, "sincad:"+legacyCPF).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	digits := nonDigits.ReplaceAllString(legacyCPF, "")
	if digits == "" {
		return "", nil
	}
	err = pool.QueryRow(ctx,
		`SELECT id::text FROM members
		 WHERE regexp_replace(COALESCE(cpf,''), '\D', '', 'g') = $1
		 LIMIT 1`, digits).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return id, nil
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "erro:", msg)
	os.Exit(1)
}
