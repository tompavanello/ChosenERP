package finance

import (
	"context"
	"encoding/csv"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// BankImport e a importacao do extrato de uma conta bancaria.
type BankImport struct {
	ID             string  `json:"id"`
	BranchID       string  `json:"branch_id"`
	AccountID      *string `json:"account_id,omitempty"`
	AccountName    *string `json:"account_name,omitempty"`
	Filename       string  `json:"filename"`
	Format         string  `json:"format"`
	PeriodStart    *string `json:"period_start,omitempty"`
	PeriodEnd      *string `json:"period_end,omitempty"`
	Status         string  `json:"status"`
	TotalEntries   int     `json:"total_entries"`
	MatchedEntries int     `json:"matched_entries"`
	MissingEntries int     `json:"missing_entries"`
	CreatedAt      string  `json:"created_at"`
}

// BankEntry e uma linha do extrato importado.
type BankEntry struct {
	ID                     string  `json:"id"`
	PostedAt               string  `json:"posted_at"`
	Amount                 float64 `json:"amount"`
	Direction              string  `json:"direction"`
	Memo                   *string `json:"memo,omitempty"`
	FitID                  *string `json:"fitid,omitempty"`
	Status                 string  `json:"status"`
	TransactionID          *string `json:"transaction_id,omitempty"`
	TransactionDescription *string `json:"transaction_description,omitempty"`
}

// BankDivergence e um lancamento do ERP que nao aparece no extrato.
type BankDivergence struct {
	ID           string  `json:"id"`
	OccurredAt   string  `json:"occurred_at"`
	Amount       float64 `json:"amount"`
	Direction    string  `json:"direction"`
	Description  *string `json:"description,omitempty"`
	CategoryName *string `json:"category_name,omitempty"`
}

type ImportBankInput struct {
	AccountID string
	Filename  string
	Raw       []byte
}

type GenerateBankEntryInput struct {
	CategoryID  string `json:"category_id"`
	Description string `json:"description"`
}

type parsedEntry struct {
	PostedAt  time.Time
	Amount    float64
	Direction string
	Memo      string
	FitID     string
}

// ---------------------------------------------------------------------------
// Parsers de extrato
// ---------------------------------------------------------------------------

var (
	ofxTagRe = regexp.MustCompile(`(?is)<([A-Z0-9]+)>\s*([^<\r\n]*)`)
	ofxTrnRe = regexp.MustCompile(`(?is)<STMTTRN>`)
)

// parseStatement detecta o formato (OFX/CSV) e devolve as entradas normalizadas.
func parseStatement(filename string, raw []byte) ([]parsedEntry, string, error) {
	text := string(raw)
	if strings.Contains(strings.ToUpper(text), "<OFX") || strings.HasSuffix(strings.ToLower(filename), ".ofx") {
		entries, err := parseOFX(text)
		return entries, "ofx", err
	}
	entries, err := parseCSVStatement(text)
	return entries, "csv", err
}

func parseOFX(text string) ([]parsedEntry, error) {
	locs := ofxTrnRe.FindAllStringIndex(text, -1)
	out := []parsedEntry{}
	for i, loc := range locs {
		start := loc[1]
		end := len(text)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		block := text[start:end]
		if idx := strings.Index(strings.ToUpper(block), "</BANKTRANLIST>"); idx >= 0 {
			block = block[:idx]
		}
		tags := map[string]string{}
		for _, m := range ofxTagRe.FindAllStringSubmatch(block, -1) {
			key := strings.ToUpper(m[1])
			if _, ok := tags[key]; !ok {
				tags[key] = strings.TrimSpace(m[2])
			}
		}
		e, ok := parseOFXTags(tags)
		if ok {
			out = append(out, e)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("nenhum lancamento OFX (STMTTRN) encontrado")
	}
	return out, nil
}

func parseOFXTags(tags map[string]string) (parsedEntry, bool) {
	amountRaw := tags["TRNAMT"]
	if amountRaw == "" {
		return parsedEntry{}, false
	}
	amount, err := parseValor(amountRaw)
	if err != nil {
		return parsedEntry{}, false
	}
	when, err := parseOFXDate(tags["DTPOSTED"])
	if err != nil {
		return parsedEntry{}, false
	}
	e := parsedEntry{PostedAt: when, Amount: amount, FitID: tags["FITID"]}
	if amount < 0 {
		e.Direction = "expense"
		e.Amount = -amount
	} else {
		e.Direction = "income"
	}
	trn := strings.ToUpper(tags["TRNTYPE"])
	if strings.Contains(trn, "DEBIT") || strings.Contains(trn, "PAYMENT") || strings.Contains(trn, "CHECK") {
		e.Direction = "expense"
	} else if strings.Contains(trn, "CREDIT") || strings.Contains(trn, "DEP") {
		e.Direction = "income"
	}
	if e.Amount <= 0 {
		return parsedEntry{}, false
	}
	if m := strings.TrimSpace(tags["MEMO"]); m != "" {
		e.Memo = m
	} else if m := strings.TrimSpace(tags["NAME"]); m != "" {
		e.Memo = m
	}
	return e, true
}

// parseOFXDate aceita YYYYMMDD ou YYYYMMDDHHMMSS (com eventual fuso entre []).
func parseOFXDate(s string) (time.Time, error) {
	digits := ""
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digits += string(r)
		} else {
			break
		}
	}
	if len(digits) < 8 {
		return time.Time{}, errors.New("data OFX invalida")
	}
	return time.Parse("20060102", digits[:8])
}

func parseCSVStatement(text string) ([]parsedEntry, error) {
	text = strings.TrimPrefix(text, "\ufeff")
	sep := ','
	firstLine := firstLine(text)
	if strings.Count(firstLine, ";") > strings.Count(firstLine, ",") {
		sep = ';'
	}
	rd := csv.NewReader(strings.NewReader(text))
	rd.Comma = sep
	rd.FieldsPerRecord = -1
	rd.LazyQuotes = true
	records, err := rd.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, errors.New("extrato CSV vazio ou sem linhas de dados")
	}
	header := records[0]
	idx := map[string]int{}
	for i, h := range header {
		switch normalizeHeader(h) {
		case "data", "dia", "date", "dt":
			idx["data"] = i
		case "valor", "amount", "value", "credito", "debito":
			idx["valor"] = i
		case "descricao", "historico", "memo", "observacao", "lancamento":
			idx["memo"] = i
		case "tipo", "type", "natureza":
			idx["tipo"] = i
		case "fitid", "id", "documento":
			idx["fitid"] = i
		}
	}
	di, ok := idx["data"]
	if !ok {
		return nil, errors.New("extrato CSV sem coluna de data")
	}
	vi, ok := idx["valor"]
	if !ok {
		return nil, errors.New("extrato CSV sem coluna de valor")
	}
	get := func(rec []string, key string) string {
		j, ok := idx[key]
		if !ok || j < 0 || j >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[j])
	}
	out := []parsedEntry{}
	for _, rec := range records[1:] {
		if di >= len(rec) || vi >= len(rec) || emptyRecord(rec) {
			continue
		}
		when, err := parseData(rec[di])
		if err != nil {
			continue
		}
		amount, err := parseValor(rec[vi])
		if err != nil || amount == 0 {
			continue
		}
		e := parsedEntry{PostedAt: when, FitID: get(rec, "fitid"), Memo: get(rec, "memo")}
		if amount < 0 {
			e.Direction = "expense"
			e.Amount = -amount
		} else {
			e.Direction = "income"
			e.Amount = amount
		}
		if t := strings.ToLower(get(rec, "tipo")); t != "" {
			if strings.Contains(t, "debito") || strings.Contains(t, "saida") || t == "d" {
				e.Direction = "expense"
			} else if strings.Contains(t, "credito") || strings.Contains(t, "entrada") || t == "c" {
				e.Direction = "income"
			}
		}
		out = append(out, e)
	}
	if len(out) == 0 {
		return nil, errors.New("extrato CSV sem lancamentos validos")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Repositorio
// ---------------------------------------------------------------------------

const bankImportCols = `i.id::text, i.branch_id::text, i.account_id::text, a.name,
	i.filename, i.format, to_char(i.period_start,'YYYY-MM-DD'), to_char(i.period_end,'YYYY-MM-DD'),
	i.status, i.total_entries, i.matched_entries, i.missing_entries,
	to_char(i.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF')`

const bankImportJoin = `
	FROM bank_statement_imports i
	LEFT JOIN financial_accounts a ON a.id = i.account_id`

func scanBankImport(row pgx.Row) (*BankImport, error) {
	var b BankImport
	var start, end *string
	err := row.Scan(&b.ID, &b.BranchID, &b.AccountID, &b.AccountName,
		&b.Filename, &b.Format, &start, &end, &b.Status,
		&b.TotalEntries, &b.MatchedEntries, &b.MissingEntries, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	b.PeriodStart = start
	b.PeriodEnd = end
	return &b, nil
}

func (r *Repo) ListBankImports(ctx context.Context, tx pgx.Tx) ([]BankImport, error) {
	rows, err := tx.Query(ctx, `SELECT `+bankImportCols+bankImportJoin+` ORDER BY i.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BankImport{}
	for rows.Next() {
		b, err := scanBankImport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}

func (r *Repo) GetBankImport(ctx context.Context, tx pgx.Tx, id string) (*BankImport, error) {
	return scanBankImport(tx.QueryRow(ctx, `SELECT `+bankImportCols+bankImportJoin+` WHERE i.id = $1::uuid`, id))
}

// ImportBankStatement le o arquivo, grava as entradas e roda o casamento.
func (r *Repo) ImportBankStatement(ctx context.Context, tx pgx.Tx, tenantID, branchID, actorID string, in ImportBankInput) (*BankImport, error) {
	if strings.TrimSpace(in.AccountID) == "" {
		return nil, errors.New("selecione a conta bancaria do extrato")
	}
	var acctName string
	if err := tx.QueryRow(ctx, `SELECT name FROM financial_accounts WHERE id = $1::uuid`, in.AccountID).Scan(&acctName); err != nil {
		return nil, errors.New("conta bancaria nao encontrada no escopo")
	}
	entries, format, err := parseStatement(in.Filename, in.Raw)
	if err != nil {
		return nil, err
	}
	minD, maxD := entries[0].PostedAt, entries[0].PostedAt
	for _, e := range entries {
		if e.PostedAt.Before(minD) {
			minD = e.PostedAt
		}
		if e.PostedAt.After(maxD) {
			maxD = e.PostedAt
		}
	}
	var importID string
	err = tx.QueryRow(ctx, `
		INSERT INTO bank_statement_imports
			(tenant_id, branch_id, account_id, filename, format, period_start, period_end, total_entries, created_by)
		VALUES ($1, $2::uuid, $3::uuid, $4, $5, $6::date, $7::date, $8, NULLIF($9,'')::uuid)
		RETURNING id::text`,
		tenantID, branchID, in.AccountID, in.Filename, format,
		minD.Format(dateLayout), maxD.Format(dateLayout), len(entries), actorID).Scan(&importID)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if _, err := tx.Exec(ctx, `
			INSERT INTO bank_statement_entries (import_id, posted_at, amount, direction, memo, fitid)
			VALUES ($1::uuid, $2::date, $3, $4, NULLIF($5,''), NULLIF($6,''))
			ON CONFLICT DO NOTHING`,
			importID, e.PostedAt.Format(dateLayout), e.Amount, e.Direction, e.Memo, e.FitID); err != nil {
			return nil, err
		}
	}
	if err := r.matchBankEntries(ctx, tx, importID); err != nil {
		return nil, err
	}
	return r.GetBankImport(ctx, tx, importID)
}

// matchBankEntries casa as entradas pendentes com lancamentos do ERP por
// valor, tipo e data (tolerancia de 3 dias). Cada lancamento so pode ser
// conciliado por uma entrada.
func (r *Repo) matchBankEntries(ctx context.Context, tx pgx.Tx, importID string) error {
	var tenantID, branchID, accountID string
	if err := tx.QueryRow(ctx, `
		SELECT tenant_id::text, branch_id::text, COALESCE(account_id::text,'')
		FROM bank_statement_imports WHERE id = $1::uuid`, importID).Scan(&tenantID, &branchID, &accountID); err != nil {
		return err
	}
	if accountID == "" {
		return nil
	}
	type pend struct {
		id, posted, direction string
		amount                float64
	}
	rows, err := tx.Query(ctx, `
		SELECT id::text, to_char(posted_at,'YYYY-MM-DD'), amount::float8, direction
		FROM bank_statement_entries
		WHERE import_id = $1::uuid AND status = 'pendente'
		ORDER BY posted_at, id`, importID)
	if err != nil {
		return err
	}
	list := []pend{}
	for rows.Next() {
		var p pend
		if err := rows.Scan(&p.id, &p.posted, &p.amount, &p.direction); err != nil {
			rows.Close()
			return err
		}
		list = append(list, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, p := range list {
		var txID *string
		err := tx.QueryRow(ctx, `
			SELECT t.id::text
			FROM financial_transactions t
			WHERE t.tenant_id = $1::uuid AND t.branch_id = $2::uuid AND t.account_id = $3::uuid
			  AND t.type = $4 AND t.voided_at IS NULL
			  AND abs(t.amount - $5) < 0.005
			  AND t.occurred_at::date BETWEEN $6::date - 3 AND $6::date + 3
			  AND NOT EXISTS (
			      SELECT 1 FROM bank_statement_entries be
			      WHERE be.transaction_id = t.id AND be.status IN ('conciliado','lancamento_gerado'))
			ORDER BY abs(t.occurred_at::date - $6::date), t.created_at
			LIMIT 1`, tenantID, branchID, accountID, p.direction, p.amount, p.posted).Scan(&txID)
		if err == pgx.ErrNoRows {
			continue
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE bank_statement_entries
			SET status = 'conciliado', transaction_id = $2::uuid, updated_at = now()
			WHERE id = $1::uuid`, p.id, *txID); err != nil {
			return err
		}
	}
	return refreshBankCounters(ctx, tx, importID)
}

func refreshBankCounters(ctx context.Context, tx pgx.Tx, importID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE bank_statement_imports i
		SET matched_entries = sub.matched, missing_entries = sub.missing, updated_at = now()
		FROM (
			SELECT count(*) FILTER (WHERE status IN ('conciliado','lancamento_gerado')) AS matched,
			       count(*) FILTER (WHERE status = 'pendente') AS missing
			FROM bank_statement_entries WHERE import_id = $1::uuid
		) sub
		WHERE i.id = $1::uuid`, importID)
	return err
}

func (r *Repo) ListBankEntries(ctx context.Context, tx pgx.Tx, importID string) ([]BankEntry, error) {
	rows, err := tx.Query(ctx, `
		SELECT e.id::text, to_char(e.posted_at,'YYYY-MM-DD'), e.amount::float8, e.direction,
		       e.memo, e.fitid, e.status, e.transaction_id::text, t.description
		FROM bank_statement_entries e
		LEFT JOIN financial_transactions t ON t.id = e.transaction_id
		WHERE e.import_id = $1::uuid
		ORDER BY e.posted_at, e.created_at, e.id`, importID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BankEntry{}
	for rows.Next() {
		var e BankEntry
		if err := rows.Scan(&e.ID, &e.PostedAt, &e.Amount, &e.Direction,
			&e.Memo, &e.FitID, &e.Status, &e.TransactionID, &e.TransactionDescription); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repo) ListBankDivergences(ctx context.Context, tx pgx.Tx, importID string) ([]BankDivergence, error) {
	rows, err := tx.Query(ctx, `
		SELECT t.id::text, to_char(t.occurred_at,'YYYY-MM-DD'), t.amount::float8, t.type,
		       t.description, c.name
		FROM bank_statement_imports i
		JOIN financial_transactions t
		  ON t.tenant_id = i.tenant_id AND t.branch_id = i.branch_id
		 AND t.account_id = i.account_id
		 AND t.voided_at IS NULL
		 AND t.occurred_at::date BETWEEN i.period_start AND i.period_end
		LEFT JOIN financial_categories c ON c.id = t.category_id
		WHERE i.id = $1::uuid
		  AND NOT EXISTS (
		      SELECT 1 FROM bank_statement_entries e
		      WHERE e.transaction_id = t.id AND e.status IN ('conciliado','lancamento_gerado'))
		ORDER BY t.occurred_at, t.created_at, t.id`, importID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BankDivergence{}
	for rows.Next() {
		var d BankDivergence
		if err := rows.Scan(&d.ID, &d.OccurredAt, &d.Amount, &d.Direction, &d.Description, &d.CategoryName); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// IgnoreBankEntry marca uma entrada do extrato como ignorada.
func (r *Repo) IgnoreBankEntry(ctx context.Context, tx pgx.Tx, entryID string) error {
	tag, err := tx.Exec(ctx, `
		UPDATE bank_statement_entries
		SET status = 'ignorado', updated_at = now()
		WHERE id = $1::uuid AND status = 'pendente'`, entryID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// GenerateFromBankEntry cria um lancamento a partir de uma entrada do extrato
// sem correspondencia e vincula os dois.
func (r *Repo) GenerateFromBankEntry(ctx context.Context, tx pgx.Tx, tenantID, branchID, entryID string, in GenerateBankEntryInput) (*Transaction, error) {
	if strings.TrimSpace(in.CategoryID) == "" {
		return nil, errors.New("selecione a conta contabil do lancamento")
	}
	var posted, direction, accountID string
	var amount float64
	var memo *string
	err := tx.QueryRow(ctx, `
		SELECT to_char(e.posted_at,'YYYY-MM-DD'), e.direction, e.amount::float8, e.memo,
		       COALESCE(i.account_id::text,'')
		FROM bank_statement_entries e
		JOIN bank_statement_imports i ON i.id = e.import_id
		WHERE e.id = $1::uuid AND e.status = 'pendente'`, entryID).
		Scan(&posted, &direction, &amount, &memo, &accountID)
	if err != nil {
		return nil, err
	}
	desc := in.Description
	if strings.TrimSpace(desc) == "" && memo != nil {
		desc = *memo
	}
	if strings.TrimSpace(desc) == "" {
		desc = "Lancamento gerado da conciliacao bancaria"
	}
	catID := in.CategoryID
	var acct *string
	if accountID != "" {
		acct = &accountID
	}
	created, _, _, _, err := r.Create(ctx, tx, tenantID, branchID, CreateTxnInput{
		Type:        direction,
		Amount:      amount,
		CategoryID:  &catID,
		AccountID:   acct,
		Description: &desc,
		OccurredAt:  &posted,
	})
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE bank_statement_entries
		SET status = 'lancamento_gerado', transaction_id = $2::uuid, updated_at = now()
		WHERE id = $1::uuid`, entryID, created.ID); err != nil {
		return nil, err
	}
	// Atualiza os contadores da importacao (matched/missing).
	var importID string
	if err := tx.QueryRow(ctx, `SELECT import_id::text FROM bank_statement_entries WHERE id = $1::uuid`, entryID).Scan(&importID); err != nil {
		return nil, err
	}
	if err := refreshBankCounters(ctx, tx, importID); err != nil {
		return nil, err
	}
	return created, nil
}

func (r *Repo) DeleteBankImport(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM bank_statement_imports WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
