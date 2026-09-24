package store

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"

	"chosenerp/db"
	"github.com/jackc/pgx/v5"
)

// ---------------------------------------------------------------------------
// Harness dos testes de RLS.
//
// Os testes precisam de um PostgreSQL real (as politicas de Row-Level Security
// so existem no banco). Para nao tocar no banco de desenvolvimento, o harness
// cria um banco descartavel `chosenerp_test` a partir do DSN do migrador,
// aplica as mesmas migracoes do binario e semeia fixtures proprias.
//
// Configuracao (tudo opcional, com fallback):
//   CHOSEN_TEST_MIGRATE_URL  -> fallback: MIGRATE_DATABASE_URL  (papel dono/superuser)
//   CHOSEN_TEST_APP_URL      -> fallback: DATABASE_URL          (papel da aplicacao, sujeito a RLS)
//   CHOSEN_TESTS_REQUIRED=1  -> falha em vez de pular, quando nao ha DSN (use no CI)
// ---------------------------------------------------------------------------

const rlsTestDBName = "chosenerp_test"

// errRollback encerra a transacao de teste sem commitar, para que os testes
// (mesmo os de escrita) sejam idempotentes e nao deixem residuo no banco.
var errRollback = errors.New("rls: rollback intencional da transacao de teste")

// IDs fixos das fixtures (UUIDs validos, estaveis entre execucoes).
const (
	fixTenantX = "aaaaaaaa-0000-4000-8000-000000000001"
	fixTenantY = "aaaaaaaa-0000-4000-8000-000000000002"

	fixBranchA = "bbbbbbbb-0000-4000-8000-000000000001" // tenant X
	fixBranchB = "bbbbbbbb-0000-4000-8000-000000000002" // tenant X
	fixBranchC = "bbbbbbbb-0000-4000-8000-000000000003" // tenant X (nao participa de repasses)
	fixBranchY = "cccccccc-0000-4000-8000-000000000001" // tenant Y
	// Sub-congregacao filha da filial A (hierarquia, migracao 000036).
	fixSubBranchA = "bbbbbbbb-0000-4000-8000-000000000004"

	fixMemberA1 = "dddddddd-0000-4000-8000-000000000001"
	fixMemberA2 = "dddddddd-0000-4000-8000-000000000002"
	fixMemberB1 = "dddddddd-0000-4000-8000-000000000003"
	fixMemberB2 = "dddddddd-0000-4000-8000-000000000004"
	fixMemberY1 = "dddddddd-0000-4000-8000-000000000005"

	fixCatA = "eeeeeeee-0000-4000-8000-000000000001"
	fixCatB = "eeeeeeee-0000-4000-8000-000000000002"

	fixTxnA = "feeeeeee-0000-4000-8000-000000000001"
	fixTxnB = "feeeeeee-0000-4000-8000-000000000002"

	fixTxnAccountA = "a5aaaaaa-0000-4000-8000-000000000003"

	fixAttachmentA = "b3aaaaaa-0000-4000-8000-000000000001"

	fixDocA = "f1eeeeee-0000-4000-8000-000000000001"

	fixFamilyA = "a1aaaaaa-0000-4000-8000-000000000001"
	fixFamilyB = "a1aaaaaa-0000-4000-8000-000000000002"

	fixVisitorA = "a2aaaaaa-0000-4000-8000-000000000001"
	fixVisitorB = "a2aaaaaa-0000-4000-8000-000000000002"
	fixVisitorY = "a2aaaaaa-0000-4000-8000-000000000003"
	// Visitante da sub-congregacao: prova que a filial A (pai) enxerga o
	// descendente na leitura, mas a irma B nao.
	fixVisitorSub = "a2aaaaaa-0000-4000-8000-000000000004"

	fixBeneA      = "a3aaaaaa-0000-4000-8000-000000000001"
	fixBeneGlobal = "a3aaaaaa-0000-4000-8000-000000000002"
	fixBeneY      = "a3aaaaaa-0000-4000-8000-000000000003"

	fixMinistryA = "a4aaaaaa-0000-4000-8000-000000000001"
	fixMinistryB = "a4aaaaaa-0000-4000-8000-000000000002"

	fixCargoGlobal = "adaaaaaa-0000-4000-8000-000000000001" // tenant X, branch NULL (global)
	fixCargoA      = "adaaaaaa-0000-4000-8000-000000000002" // tenant X, filial A
	fixCargoB      = "adaaaaaa-0000-4000-8000-000000000003" // tenant X, filial B
	fixCargoY      = "adaaaaaa-0000-4000-8000-000000000004" // tenant Y, filial Y

	fixGroupA = "a5aaaaaa-0000-4000-8000-000000000001"
	fixGroupB = "a5aaaaaa-0000-4000-8000-000000000002"

	fixAccountA = "b1aaaaaa-0000-4000-8000-000000000001" // tenant X, branch A
	fixAccountB = "b1aaaaaa-0000-4000-8000-000000000002" // tenant X, branch B
	fixAccountY = "b1aaaaaa-0000-4000-8000-000000000003" // tenant Y

	fixAttA = "a6aaaaaa-0000-4000-8000-000000000001"
	fixAttB = "a6aaaaaa-0000-4000-8000-000000000002"

	fixRecA = "a7aaaaaa-0000-4000-8000-000000000001"
	fixRecB = "a7aaaaaa-0000-4000-8000-000000000002"

	fixTransferAB = "a8aaaaaa-0000-4000-8000-000000000001"

	fixAnnA      = "a9aaaaaa-0000-4000-8000-000000000001"
	fixAnnGlobal = "a9aaaaaa-0000-4000-8000-000000000002"

	fixTermX = "abaaaaaa-0000-4000-8000-000000000001"

	// Governanca (migracao 000033).
	fixMinuteA    = "c0aaaaaa-0000-4000-8000-000000000001"
	fixMinuteB    = "c0aaaaaa-0000-4000-8000-000000000002"
	fixSignatureA = "c2aaaaaa-0000-4000-8000-000000000001"
	fixVoteA      = "c1aaaaaa-0000-4000-8000-000000000001"
	fixVoteB      = "c1aaaaaa-0000-4000-8000-000000000002"
	fixVoteOptA1  = "c3aaaaaa-0000-4000-8000-000000000001"
	fixVoteOptA2  = "c3aaaaaa-0000-4000-8000-000000000002"
	fixVoteOptB1  = "c3aaaaaa-0000-4000-8000-000000000003"
	fixBallotA    = "c4aaaaaa-0000-4000-8000-000000000001"
	fixBallotB    = "c4aaaaaa-0000-4000-8000-000000000002"
	fixLegalA     = "c5aaaaaa-0000-4000-8000-000000000001"
	fixLegalB     = "c5aaaaaa-0000-4000-8000-000000000002"

	// Identidade global + memberships (migracao 000053).
	fixRoleXSuper = "0a000000-0000-4000-8000-000000000001" // tenant X, super_admin
	fixRoleXSec   = "0a000000-0000-4000-8000-000000000002" // tenant X, secretario
	fixRoleYSuper = "0a000000-0000-4000-8000-000000000003" // tenant Y, super_admin
	fixUserX      = "0b000000-0000-4000-8000-000000000001" // so tenant X (filial A)
	fixUserY      = "0b000000-0000-4000-8000-000000000002" // so tenant Y
	fixUserBoth   = "0b000000-0000-4000-8000-000000000003" // tenant X (Sede) + tenant Y
	fixMemX       = "0c000000-0000-4000-8000-000000000001"
	fixMemY       = "0c000000-0000-4000-8000-000000000002"
	fixMemBothX   = "0c000000-0000-4000-8000-000000000003"
	fixMemBothY   = "0c000000-0000-4000-8000-000000000004"
)

// bounds monta o contexto de seguranca usado pelo gateway.
func bounds(tenantID, branchID, role string) Bounds {
	return Bounds{TenantID: tenantID, BranchID: branchID, Role: role}
}

// boundsUser inclui a identidade (app.user_id), necessaria para o RLS de users
// e memberships reconhecer a propria pessoa.
func boundsUser(tenantID, branchID, role, userID string) Bounds {
	return Bounds{TenantID: tenantID, BranchID: branchID, Role: role, UserID: userID}
}

var (
	testStore      *Store
	testMigrateDSN string
	testCtx        = context.Background()
)

func TestMain(m *testing.M) {
	code, err := runSuite(m)
	if err != nil {
		fmt.Fprintf(os.Stderr, "store: falha ao preparar a suite de RLS: %v\n", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func runSuite(m *testing.M) (int, error) {
	migrateDSN := firstEnv("CHOSEN_TEST_MIGRATE_URL", "MIGRATE_DATABASE_URL")
	appDSN := firstEnv("CHOSEN_TEST_APP_URL", "DATABASE_URL")

	if migrateDSN == "" || appDSN == "" {
		msg := "store: testes de RLS pulados - defina CHOSEN_TEST_MIGRATE_URL e " +
			"CHOSEN_TEST_APP_URL (ou MIGRATE_DATABASE_URL/DATABASE_URL)"
		if os.Getenv("CHOSEN_TESTS_REQUIRED") != "" {
			return 1, errors.New(msg)
		}
		fmt.Fprintln(os.Stderr, msg)
		return 0, nil
	}

	// Conexao administrativa no banco padrao, so para (re)criar o banco de teste.
	adminDSN, err := rewriteDatabase(migrateDSN, "postgres")
	if err != nil {
		return 1, err
	}
	admin, err := pgx.Connect(testCtx, adminDSN)
	if err != nil {
		return 1, fmt.Errorf("conectar como migrador: %w", err)
	}
	defer admin.Close(testCtx)

	if _, err := admin.Exec(testCtx, fmt.Sprintf(
		`DROP DATABASE IF EXISTS %s WITH (FORCE)`, rlsTestDBName)); err != nil {
		return 1, fmt.Errorf("drop %s: %w", rlsTestDBName, err)
	}
	if _, err := admin.Exec(testCtx, fmt.Sprintf(`CREATE DATABASE %s`, rlsTestDBName)); err != nil {
		return 1, fmt.Errorf("create %s: %w", rlsTestDBName, err)
	}

	migrateTestDSN, err := rewriteDatabase(migrateDSN, rlsTestDBName)
	if err != nil {
		return 1, err
	}

	// GRANT e ALTER DEFAULT PRIVILEGES valem apenas para o database onde sao
	// executados. Um database novo nao os herda, entao o script de setup precisa
	// rodar aqui ANTES das migracoes, para que os privilegios padrao ja cubram
	// as tabelas que serao criadas.
	setup, err := pgx.Connect(testCtx, migrateTestDSN)
	if err != nil {
		return 1, fmt.Errorf("conectar no database de teste: %w", err)
	}
	if _, err := setup.Exec(testCtx, db.Setup); err != nil {
		setup.Close(testCtx)
		return 1, fmt.Errorf("aplicar db/init/setup.sql: %w", err)
	}
	setup.Close(testCtx)
	appTestDSN, err := rewriteDatabase(appDSN, rlsTestDBName)
	if err != nil {
		return 1, err
	}

	st, err := New(testCtx, appTestDSN, migrateTestDSN)
	if err != nil {
		return 1, err
	}
	defer st.Close()

	if err := st.Migrate(testCtx); err != nil {
		return 1, fmt.Errorf("migrate: %w", err)
	}
	if err := seedFixtures(migrateTestDSN); err != nil {
		return 1, err
	}

	testStore = st
	testMigrateDSN = migrateTestDSN
	return m.Run(), nil
}

// withSuperuser executa fn fora do RLS (o migrador e superuser), usado para
// verificar gatilhos/triggers que a RLS esconderia do papel da aplicacao.
func withSuperuser(t *testing.T, fn func(conn *pgx.Conn) error) {
	t.Helper()
	conn, err := pgx.Connect(testCtx, testMigrateDSN)
	if err != nil {
		t.Fatalf("conectar como superuser: %v", err)
	}
	defer conn.Close(testCtx)
	if err := fn(conn); err != nil {
		t.Fatalf("superuser: %v", err)
	}
}

// rewriteDatabase troca o nome do banco de um DSN postgres://... preservando
// usuario, senha, host e parametros de query.
func rewriteDatabase(dsn, database string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("parse DSN: %w", err)
	}
	u.Path = "/" + database
	return u.String(), nil
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

// seedFixtures semeia os dados de teste usando o DSN do MIGRADOR, que e
// superuser e portanto nao e filtrado pelas politicas de RLS.
func seedFixtures(migrateDSN string) error {
	conn, err := pgx.Connect(testCtx, migrateDSN)
	if err != nil {
		return fmt.Errorf("conectar para seed: %w", err)
	}
	defer conn.Close(testCtx)

	stmts := []string{
		`INSERT INTO tenants (id, name, slug) VALUES
			('` + fixTenantX + `', 'Tenant X', 'rls-tenant-x'),
			('` + fixTenantY + `', 'Tenant Y', 'rls-tenant-y')`,

		`INSERT INTO branches (id, tenant_id, name, slug, kind) VALUES
			('` + fixBranchA + `', '` + fixTenantX + `', 'Filial A', 'rls-filial-a', 'filial'),
			('` + fixBranchB + `', '` + fixTenantX + `', 'Filial B', 'rls-filial-b', 'filial'),
			('` + fixBranchC + `', '` + fixTenantX + `', 'Filial C', 'rls-filial-c', 'filial'),
			('` + fixBranchY + `', '` + fixTenantY + `', 'Filial Y', 'rls-filial-y', 'filial')`,

		// PAE supervisionado pela Filial A (hierarquia Matriz > Filial > PAE).
		`INSERT INTO branches (id, tenant_id, parent_id, name, slug, kind) VALUES
			('` + fixSubBranchA + `', '` + fixTenantX + `', '` + fixBranchA + `', 'PAE A', 'rls-pae-a', 'pae')`,

		`INSERT INTO members (id, tenant_id, branch_id, first_name, last_name, full_name, membership_status) VALUES
			('` + fixMemberA1 + `', '` + fixTenantX + `', '` + fixBranchA + `', 'Ana', 'A', 'Ana A', 'active'),
			('` + fixMemberA2 + `', '` + fixTenantX + `', '` + fixBranchA + `', 'Andre', 'A', 'Andre A', 'active'),
			('` + fixMemberB1 + `', '` + fixTenantX + `', '` + fixBranchB + `', 'Bruno', 'B', 'Bruno B', 'active'),
			('` + fixMemberB2 + `', '` + fixTenantX + `', '` + fixBranchB + `', 'Bianca', 'B', 'Bianca B', 'active'),
			('` + fixMemberY1 + `', '` + fixTenantY + `', '` + fixBranchY + `', 'Yara', 'Y', 'Yara Y', 'active')`,

		`INSERT INTO financial_categories (id, tenant_id, branch_id, type, code, name) VALUES
			('` + fixCatA + `', '` + fixTenantX + `', '` + fixBranchA + `', 'income', '1.1', 'Dizimos A'),
			('` + fixCatB + `', '` + fixTenantX + `', '` + fixBranchB + `', 'income', '1.1', 'Dizimos B')`,

		`INSERT INTO financial_accounts (id, tenant_id, branch_id, name, bank, agency, account_number, account_type, initial_balance) VALUES
			('` + fixAccountA + `', '` + fixTenantX + `', '` + fixBranchA + `', 'Conta A', 'Banco A', '1234', '56789-0', 'checking', 1000),
			('` + fixAccountB + `', '` + fixTenantX + `', '` + fixBranchB + `', 'Conta B', 'Banco B', '9876', '12345-6', 'checking', 500),
			('` + fixAccountY + `', '` + fixTenantY + `', '` + fixBranchY + `', 'Conta Y', 'Banco Y', '1111', '22222-2', 'checking', 200)`,

		// hash/prev_hash sao preenchidos pelo trigger de hash-chain.
		`INSERT INTO financial_transactions (id, tenant_id, branch_id, category_id, type, amount, currency, payment_method, description, account_id) VALUES
			('` + fixTxnA + `', '` + fixTenantX + `', '` + fixBranchA + `', '` + fixCatA + `', 'income', 100.00, 'BRL', 'pix', 'dizimo A', '` + fixAccountA + `'),
			('` + fixTxnB + `', '` + fixTenantX + `', '` + fixBranchB + `', '` + fixCatB + `', 'income', 250.00, 'BRL', 'pix', 'dizimo B', '` + fixAccountB + `')`,

		`INSERT INTO financial_attachments (id, tenant_id, branch_id, transaction_id, file_name, file_url, content_type, file_size) VALUES
			('` + fixAttachmentA + `', '` + fixTenantX + `', '` + fixBranchA + `', '` + fixTxnA + `', 'comprovante.jpg', '/api/v1/attachments/test.jpg', 'image/jpeg', 102400)`,

		`INSERT INTO documents (id, tenant_id, branch_id, kind, title, member_id, content) VALUES
			('` + fixDocA + `', '` + fixTenantX + `', '` + fixBranchA + `', 'receipt', 'Recibo A', '` + fixMemberA1 + `', '{}'::jsonb)`,

		`INSERT INTO families (id, tenant_id, branch_id, name) VALUES
			('` + fixFamilyA + `', '` + fixTenantX + `', '` + fixBranchA + `', 'Familia A'),
			('` + fixFamilyB + `', '` + fixTenantX + `', '` + fixBranchB + `', 'Familia B')`,

		`INSERT INTO member_relationships (member_id, related_id, kind) VALUES
			('` + fixMemberA1 + `', '` + fixMemberA2 + `', 'spouse'),
			('` + fixMemberB1 + `', '` + fixMemberB2 + `', 'spouse'),
			('` + fixMemberY1 + `', '` + fixMemberY1 + `', 'discipler')`,

		`INSERT INTO visitors (id, tenant_id, branch_id, first_name, last_name) VALUES
			('` + fixVisitorA + `', '` + fixTenantX + `', '` + fixBranchA + `', 'Vera', 'A'),
			('` + fixVisitorB + `', '` + fixTenantX + `', '` + fixBranchB + `', 'Vitor', 'B'),
			('` + fixVisitorY + `', '` + fixTenantY + `', '` + fixBranchY + `', 'Vera', 'Y'),
			('` + fixVisitorSub + `', '` + fixTenantX + `', '` + fixSubBranchA + `', 'Sub', 'A')`,

		`INSERT INTO benefactors (id, tenant_id, branch_id, name) VALUES
			('` + fixBeneA + `', '` + fixTenantX + `', '` + fixBranchA + `', 'Benfeitor A'),
			('` + fixBeneGlobal + `', '` + fixTenantX + `', NULL, 'Benfeitor Global X'),
			('` + fixBeneY + `', '` + fixTenantY + `', '` + fixBranchY + `', 'Benfeitor Y')`,

		`INSERT INTO ministries (id, tenant_id, branch_id, name, slug) VALUES
			('` + fixMinistryA + `', '` + fixTenantX + `', '` + fixBranchA + `', 'Louvor A', 'rls-louvor-a'),
			('` + fixMinistryB + `', '` + fixTenantX + `', '` + fixBranchB + `', 'Louvor B', 'rls-louvor-b')`,

		`INSERT INTO ministry_members (ministry_id, member_id, role) VALUES
			('` + fixMinistryA + `', '` + fixMemberA1 + `', 'volunteer'),
			('` + fixMinistryB + `', '` + fixMemberB1 + `', 'leader')`,

		// Cargos: um global do tenant X (branch NULL), um por filial, e um do
		// tenant Y - cobre os tres eixos de escopo (global / filial / tenant).
		`INSERT INTO cargos (id, tenant_id, branch_id, name, slug) VALUES
			('` + fixCargoGlobal + `', '` + fixTenantX + `', NULL, 'Cargo Global X', 'rls-cargo-global-x'),
			('` + fixCargoA + `', '` + fixTenantX + `', '` + fixBranchA + `', 'Cargo A', 'rls-cargo-a'),
			('` + fixCargoB + `', '` + fixTenantX + `', '` + fixBranchB + `', 'Cargo B', 'rls-cargo-b'),
			('` + fixCargoY + `', '` + fixTenantY + `', '` + fixBranchY + `', 'Cargo Y', 'rls-cargo-y')`,

		// Um mandato por filial: e o que prova que member_cargos herda o escopo
		// do membro (a tabela nao tem tenant_id/branch_id proprios).
		`INSERT INTO member_cargos (member_id, cargo_id, started_at, status) VALUES
			('` + fixMemberA1 + `', '` + fixCargoA + `', '2024-01-01', 'ativo'),
			('` + fixMemberB1 + `', '` + fixCargoB + `', '2024-01-01', 'ativo')`,

		`INSERT INTO small_groups (id, tenant_id, branch_id, name, kind) VALUES
			('` + fixGroupA + `', '` + fixTenantX + `', '` + fixBranchA + `', 'Celula A', 'cell'),
			('` + fixGroupB + `', '` + fixTenantX + `', '` + fixBranchB + `', 'Celula B', 'cell')`,

		`INSERT INTO group_attendance (id, tenant_id, branch_id, small_group_id, member_id, present) VALUES
			('` + fixAttA + `', '` + fixTenantX + `', '` + fixBranchA + `', '` + fixGroupA + `', '` + fixMemberA1 + `', true),
			('` + fixAttB + `', '` + fixTenantX + `', '` + fixBranchB + `', '` + fixGroupB + `', '` + fixMemberB1 + `', true)`,

		`INSERT INTO recurring_donations (id, tenant_id, branch_id, amount, subtype, frequency, next_run_at, account_id) VALUES
			('` + fixRecA + `', '` + fixTenantX + `', '` + fixBranchA + `', 50.00, 'dizimo', 'monthly', now(), '` + fixAccountA + `'),
			('` + fixRecB + `', '` + fixTenantX + `', '` + fixBranchB + `', 80.00, 'dizimo', 'monthly', now(), '` + fixAccountB + `')`,

		`INSERT INTO transfers (id, tenant_id, from_branch_id, to_branch_id, amount, rule_name) VALUES
			('` + fixTransferAB + `', '` + fixTenantX + `', '` + fixBranchA + `', '` + fixBranchB + `', 25.00, '10% sede')`,

		`INSERT INTO announcements (id, tenant_id, branch_id, title, body) VALUES
			('` + fixAnnA + `', '` + fixTenantX + `', '` + fixBranchA + `', 'Aviso A', 'Corpo A'),
			('` + fixAnnGlobal + `', '` + fixTenantX + `', NULL, 'Aviso Global X', 'Corpo global')`,

		`INSERT INTO document_deliveries (tenant_id, branch_id, document_id, channel, recipient) VALUES
			('` + fixTenantX + `', '` + fixBranchA + `', '` + fixDocA + `', 'email', 'a@teste.local')`,

		`INSERT INTO consent_terms (id, tenant_id, version, title, body) VALUES
			('` + fixTermX + `', '` + fixTenantX + `', 1, 'Termo X', 'Corpo X')`,

		`INSERT INTO member_consents (tenant_id, branch_id, consent_term_id, subject_type, subject_id, consented) VALUES
			('` + fixTenantX + `', '` + fixBranchA + `', '` + fixTermX + `', 'member', '` + fixMemberA1 + `', true)`,

		`INSERT INTO audit_log (tenant_id, action, entity, entity_id, payload) VALUES
			('` + fixTenantX + `', 'member.created', 'members', '` + fixMemberA1 + `', '{}'::jsonb),
			('` + fixTenantY + `', 'member.created', 'members', '` + fixMemberY1 + `', '{}'::jsonb)`,

		// Governanca: uma ata por filial, assinatura e votacao na filial A, e um
		// documento legal por filial.
		`INSERT INTO minutes (id, tenant_id, branch_id, title, meeting_at, kind) VALUES
			('` + fixMinuteA + `', '` + fixTenantX + `', '` + fixBranchA + `', 'Assembleia A', now(), 'assembleia'),
			('` + fixMinuteB + `', '` + fixTenantX + `', '` + fixBranchB + `', 'Assembleia B', now(), 'assembleia')`,

		`INSERT INTO minute_signatures (id, tenant_id, branch_id, minute_id, signer_name, document_hash) VALUES
			('` + fixSignatureA + `', '` + fixTenantX + `', '` + fixBranchA + `', '` + fixMinuteA + `', 'Ana A', 'hash-a')`,

		`INSERT INTO votes (id, tenant_id, branch_id, minute_id, title) VALUES
			('` + fixVoteA + `', '` + fixTenantX + `', '` + fixBranchA + `', '` + fixMinuteA + `', 'Votacao A'),
			('` + fixVoteB + `', '` + fixTenantX + `', '` + fixBranchB + `', NULL, 'Votacao B')`,

		`INSERT INTO vote_options (id, tenant_id, branch_id, vote_id, label, sort_order) VALUES
			('` + fixVoteOptA1 + `', '` + fixTenantX + `', '` + fixBranchA + `', '` + fixVoteA + `', 'Sim', 1),
			('` + fixVoteOptA2 + `', '` + fixTenantX + `', '` + fixBranchA + `', '` + fixVoteA + `', 'Nao', 2),
			('` + fixVoteOptB1 + `', '` + fixTenantX + `', '` + fixBranchB + `', '` + fixVoteB + `', 'Sim', 1)`,

		`INSERT INTO vote_ballots (id, tenant_id, branch_id, vote_id, option_id) VALUES
			('` + fixBallotA + `', '` + fixTenantX + `', '` + fixBranchA + `', '` + fixVoteA + `', '` + fixVoteOptA1 + `'),
			('` + fixBallotB + `', '` + fixTenantX + `', '` + fixBranchB + `', '` + fixVoteB + `', '` + fixVoteOptB1 + `')`,

		`INSERT INTO legal_documents (id, tenant_id, branch_id, kind, title, expires_at) VALUES
			('` + fixLegalA + `', '` + fixTenantX + `', '` + fixBranchA + `', 'alvara', 'Alvara A', current_date + 10),
			('` + fixLegalB + `', '` + fixTenantX + `', '` + fixBranchB + `', 'contrato', 'Contrato B', current_date + 400)`,

		// Identidade global + memberships (000053). O seed roda DEPOIS das
		// migracoes, entao ja usa o schema novo (users sem tenant_id/role_id).
		`INSERT INTO roles (id, tenant_id, key, name, is_system) VALUES
			('` + fixRoleXSuper + `', '` + fixTenantX + `', 'super_admin', 'Super Admin X', true),
			('` + fixRoleXSec + `', '` + fixTenantX + `', 'secretario', 'Secretario X', true),
			('` + fixRoleYSuper + `', '` + fixTenantY + `', 'super_admin', 'Super Admin Y', true)`,

		`INSERT INTO users (id, email, password_hash, full_name, is_active) VALUES
			('` + fixUserX + `', 'user.x@rls.local', 'x', 'User X', true),
			('` + fixUserY + `', 'user.y@rls.local', 'x', 'User Y', true),
			('` + fixUserBoth + `', 'user.both@rls.local', 'x', 'User Both', true)`,

		`INSERT INTO memberships (id, user_id, tenant_id, role_id, branch_id, is_active) VALUES
			('` + fixMemX + `', '` + fixUserX + `', '` + fixTenantX + `', '` + fixRoleXSec + `', '` + fixBranchA + `', true),
			('` + fixMemY + `', '` + fixUserY + `', '` + fixTenantY + `', '` + fixRoleYSuper + `', '` + fixBranchY + `', true),
			('` + fixMemBothX + `', '` + fixUserBoth + `', '` + fixTenantX + `', '` + fixRoleXSuper + `', NULL, true),
			('` + fixMemBothY + `', '` + fixUserBoth + `', '` + fixTenantY + `', '` + fixRoleYSuper + `', '` + fixBranchY + `', true)`,
	}

	for _, s := range stmts {
		if _, err := conn.Exec(testCtx, s); err != nil {
			return fmt.Errorf("seed: %w\n-- SQL: %s", err, s)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers de assercao
// ---------------------------------------------------------------------------

// inBounds executa fn dentro do escopo informado, SEMPRE com rollback, para
// que nenhum teste altere o estado do banco.
func inBounds(t *testing.T, b Bounds, fn func(tx pgx.Tx) error) error {
	t.Helper()
	if testStore == nil {
		t.Fatal("store de teste nao inicializado")
	}
	err := testStore.WithTenant(testCtx, b, func(tx pgx.Tx) error {
		if err := fn(tx); err != nil {
			return err
		}
		return errRollback
	})
	if errors.Is(err, errRollback) {
		return nil
	}
	return err
}

// inSystem executa fn no escopo de sistema (workers internos).
func inSystem(t *testing.T, fn func(tx pgx.Tx) error) error {
	t.Helper()
	err := testStore.WithSystem(testCtx, func(tx pgx.Tx) error {
		if err := fn(tx); err != nil {
			return err
		}
		return errRollback
	})
	if errors.Is(err, errRollback) {
		return nil
	}
	return err
}

// count executa uma contagem dentro da transacao corrente.
func count(t *testing.T, tx pgx.Tx, query string, args ...any) int {
	t.Helper()
	var n int
	if err := tx.QueryRow(testCtx, query, args...).Scan(&n); err != nil {
		t.Fatalf("contagem falhou (%s): %v", query, err)
	}
	return n
}

// ids executa uma consulta de uma coluna uuid e devolve os valores.
func ids(t *testing.T, tx pgx.Tx, query string, args ...any) []string {
	t.Helper()
	rows, err := tx.Query(testCtx, query, args...)
	if err != nil {
		t.Fatalf("consulta falhou (%s): %v", query, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatalf("scan falhou: %v", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}
	return out
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
