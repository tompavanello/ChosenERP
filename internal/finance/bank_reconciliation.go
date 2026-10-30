package finance

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
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
	LineNo                 int     `json:"line_no"`
	SequenceNo             int     `json:"sequence_no"`
	PostedAt               string  `json:"posted_at"`
	Amount                 float64 `json:"amount"`
	Direction              string  `json:"direction"`
	Memo                   *string `json:"memo,omitempty"`
	FitID                  *string `json:"fitid,omitempty"`
	Status                 string  `json:"status"`
	TransactionID          *string `json:"transaction_id,omitempty"`
	TransactionDescription *string `json:"transaction_description,omitempty"`
	GroupID                *string `json:"group_id,omitempty"`
	GroupDescription       *string `json:"group_description,omitempty"`
	AttachmentCount        int     `json:"attachment_count"`
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
	CategoryID    string `json:"category_id"`
	Description   string `json:"description"`
	DonorMemberID string `json:"donor_member_id"`
	BenefactorID  string `json:"benefactor_id"`
	DonorName     string `json:"donor_name"`
	SupplierID    string `json:"supplier_id"`
	PaymentMethod string `json:"payment_method"`
	IsAnonymous   bool   `json:"is_anonymous"`
}

type parsedEntry struct {
	PostedAt  time.Time
	Amount    float64
	Direction string
	Memo      string
	FitID     string
	// LineNo e a posicao da linha dentro do arquivo (1..N), preservando a
	// ordem original do extrato para gerar os lancamentos na mesma sequencia.
	LineNo int
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
	upper := strings.ToUpper(text)
	lowerName := strings.ToLower(filename)
	// OFX e OFC compartilham os blocos <STMTTRN>; o OFC e um formato mais antigo
	// com raiz <OFC> (usado por alguns bancos).
	if strings.Contains(upper, "<OFX") || strings.Contains(upper, "<OFC") ||
		strings.HasSuffix(lowerName, ".ofx") || strings.HasSuffix(lowerName, ".ofc") {
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
			e.LineNo = len(out) + 1
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
		case "data", "dia", "date", "dt", "data_mov", "data_movimento", "dt_mov", "data_lancamento":
			idx["data"] = i
		case "valor", "amount", "value", "credito", "debito", "valor_lancamento":
			idx["valor"] = i
		case "descricao", "historico", "memo", "observacao", "lancamento":
			idx["memo"] = i
		case "tipo", "type", "natureza", "deb_cred", "debito_credito", "credito_debito", "dc":
			idx["tipo"] = i
		case "fitid", "id", "documento", "doc", "nr_doc", "num_doc", "numero_documento":
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
		e.LineNo = len(out) + 1
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
			INSERT INTO bank_statement_entries (import_id, posted_at, amount, direction, memo, fitid, line_no, sequence_no)
			VALUES ($1::uuid, $2::date, $3, $4, NULLIF($5,''), NULLIF($6,''), $7, $7)
			ON CONFLICT DO NOTHING`,
			importID, e.PostedAt.Format(dateLayout), e.Amount, e.Direction, e.Memo, e.FitID, e.LineNo); err != nil {
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
			  AND t.type = $4 AND t.voided_at IS NULL AND t.group_id IS NULL
			  AND abs(t.amount - $5) < 0.005
			  AND t.occurred_at::date BETWEEN $6::date - 3 AND $6::date + 3
			  AND NOT EXISTS (
			      SELECT 1 FROM bank_statement_entries be
			      WHERE be.transaction_id = t.id AND be.status IN ('conciliado','lancamento_gerado'))
			ORDER BY abs(t.occurred_at::date - $6::date), t.created_at
			LIMIT 1`, tenantID, branchID, accountID, p.direction, p.amount, p.posted).Scan(&txID)
		if err == nil {
			if _, err := tx.Exec(ctx, `
				UPDATE bank_statement_entries
				SET status = 'conciliado', transaction_id = $2::uuid, updated_at = now()
				WHERE id = $1::uuid`, p.id, *txID); err != nil {
				return err
			}
			continue
		}
		if err != pgx.ErrNoRows {
			return err
		}

		// Sem lancamento individual: tenta casar com um grupo (deposito que
		// reune varios lancamentos).
		var groupID *string
		err = tx.QueryRow(ctx, `
			SELECT g.id::text
			FROM financial_transaction_groups g
			JOIN LATERAL (
			    SELECT SUM(t.amount) AS total
			    FROM financial_transactions t
			    WHERE t.group_id = g.id AND t.voided_at IS NULL
			) agg ON true
			WHERE g.tenant_id = $1::uuid AND g.branch_id = $2::uuid
			  AND g.account_id = $3::uuid AND g.type = $4
			  AND abs(agg.total - $5) < 0.005
			  AND g.occurred_at BETWEEN $6::date - 3 AND $6::date + 3
			  AND NOT EXISTS (
			      SELECT 1 FROM bank_statement_entries be
			      WHERE be.group_id = g.id AND be.status IN ('conciliado','lancamento_gerado'))
			ORDER BY abs(g.occurred_at - $6::date), g.created_at
			LIMIT 1`, tenantID, branchID, accountID, p.direction, p.amount, p.posted).Scan(&groupID)
		if err == pgx.ErrNoRows {
			continue
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE bank_statement_entries
			SET status = 'conciliado', group_id = $2::uuid, updated_at = now()
			WHERE id = $1::uuid`, p.id, *groupID); err != nil {
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
		SELECT e.id::text, e.line_no, e.sequence_no, to_char(e.posted_at,'YYYY-MM-DD'), e.amount::float8, e.direction,
		       e.memo, e.fitid, e.status, e.transaction_id::text, t.description,
		       e.group_id::text, g.description,
		       (SELECT count(*)::int FROM financial_attachments fa
		         WHERE fa.transaction_id = e.transaction_id
		            OR fa.group_id = COALESCE(e.group_id, t.group_id))
		FROM bank_statement_entries e
		LEFT JOIN financial_transactions t ON t.id = e.transaction_id
		LEFT JOIN financial_transaction_groups g ON g.id = e.group_id
		WHERE e.import_id = $1::uuid
		ORDER BY e.posted_at, e.sequence_no, e.line_no, e.created_at, e.id`, importID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BankEntry{}
	for rows.Next() {
		var e BankEntry
		if err := rows.Scan(&e.ID, &e.LineNo, &e.SequenceNo, &e.PostedAt, &e.Amount, &e.Direction,
			&e.Memo, &e.FitID, &e.Status, &e.TransactionID, &e.TransactionDescription,
			&e.GroupID, &e.GroupDescription, &e.AttachmentCount); err != nil {
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
		      WHERE e.status IN ('conciliado','lancamento_gerado')
		        AND (e.transaction_id = t.id
		             OR (t.group_id IS NOT NULL AND e.group_id = t.group_id)))
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
	// Vinculos permitidos por tipo: entrada aceita membro/benfeitor/doador
	// avulso; saida aceita fornecedor. Membro/benfeitor tem precedencia sobre o
	// nome avulso, como no formulario de lancamento.
	donorMemberID := optString(in.DonorMemberID)
	benefactorID := optString(in.BenefactorID)
	donorName := optString(in.DonorName)
	supplierID := optString(in.SupplierID)
	if direction != "income" {
		donorMemberID, benefactorID, donorName = nil, nil, nil
	}
	if direction != "expense" {
		supplierID = nil
	}
	if donorMemberID != nil {
		benefactorID, donorName = nil, nil
	} else if benefactorID != nil {
		donorName = nil
	}
	isAnonymous := in.IsAnonymous && donorMemberID == nil && benefactorID == nil && donorName == nil
	created, _, _, _, err := r.Create(ctx, tx, tenantID, branchID, CreateTxnInput{
		Type:          direction,
		Amount:        amount,
		CategoryID:    &catID,
		AccountID:     acct,
		Description:   &desc,
		OccurredAt:    &posted,
		DonorMemberID: donorMemberID,
		BenefactorID:  benefactorID,
		DonorName:     donorName,
		SupplierID:    supplierID,
		PaymentMethod: optString(in.PaymentMethod),
		IsAnonymous:   isAnonymous,
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
	// Reposiciona a data para seguir a ordem do arquivo (line_no), inclusive
	// reposicionando linhas ja conciliadas (apenas o entry_seq muda).
	if accountID != "" {
		if err := r.alignBankDateSequence(ctx, tx, importID, accountID, posted); err != nil {
			return nil, err
		}
	}
	return created, nil
}

// GeneratePendingItem configura a geracao de uma linha pendente do extrato.
type GeneratePendingItem struct {
	EntryID       string `json:"entry_id"`
	CategoryID    string `json:"category_id"`
	Description   string `json:"description"`
	DonorMemberID string `json:"donor_member_id"`
	BenefactorID  string `json:"benefactor_id"`
	DonorName     string `json:"donor_name"`
	SupplierID    string `json:"supplier_id"`
	PaymentMethod string `json:"payment_method"`
	IsAnonymous   bool   `json:"is_anonymous"`
}

// GeneratePendingFromBank gera os lancamentos das linhas pendentes informadas,
// respeitando a ordem do arquivo (data e line_no). Retorna a quantidade gerada.
func (r *Repo) GeneratePendingFromBank(ctx context.Context, tx pgx.Tx, tenantID, branchID, importID string, items []GeneratePendingItem) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}
	cfg := make(map[string]GeneratePendingItem, len(items))
	for _, it := range items {
		id := strings.TrimSpace(it.EntryID)
		if id == "" {
			continue
		}
		cfg[id] = it
	}
	if len(cfg) == 0 {
		return 0, errors.New("informe ao menos uma linha para gerar")
	}
	rows, err := tx.Query(ctx, `
		SELECT e.id::text
		FROM bank_statement_entries e
		WHERE e.import_id = $1::uuid AND e.status = 'pendente'
		ORDER BY e.posted_at, e.sequence_no, e.line_no, e.created_at, e.id`, importID)
	if err != nil {
		return 0, err
	}
	ordered := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ordered = append(ordered, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	generated := 0
	for _, id := range ordered {
		it, ok := cfg[id]
		if !ok {
			continue
		}
		if _, err := r.GenerateFromBankEntry(ctx, tx, tenantID, branchID, id, GenerateBankEntryInput{
			CategoryID:    it.CategoryID,
			Description:   it.Description,
			DonorMemberID: it.DonorMemberID,
			BenefactorID:  it.BenefactorID,
			DonorName:     it.DonorName,
			SupplierID:    it.SupplierID,
			PaymentMethod: it.PaymentMethod,
			IsAnonymous:   it.IsAnonymous,
		}); err != nil {
			return generated, err
		}
		generated++
	}
	if generated == 0 {
		return 0, errors.New("nenhuma linha pendente selecionada")
	}
	return generated, nil
}

// alignBankDateSequence reordena as linhas visiveis (lancamentos avulsos +
// grupos) de uma conta/data para seguir a ordem de trabalho do extrato
// (sequence_no). As linhas vinculadas ao extrato sao reordenadas ENTRE AS
// POSICOES que ja ocupam; as demais linhas do razao ficam onde estao.
func (r *Repo) alignBankDateSequence(ctx context.Context, tx pgx.Tx, importID, accountID, date string) error {
	type line struct {
		key  string
		kind string
		id   string
	}
	var tenantID, branchID string
	if err := tx.QueryRow(ctx, `
		SELECT tenant_id::text, branch_id::text
		FROM bank_statement_imports WHERE id = $1::uuid`, importID).Scan(&tenantID, &branchID); err != nil {
		return err
	}

	// Ordem de trabalho: entradas vinculadas a um lancamento avulso ou grupo.
	rows, err := tx.Query(ctx, `
		SELECT COALESCE(e.sequence_no, e.line_no),
		       CASE WHEN COALESCE(e.group_id, t.group_id) IS NOT NULL THEN 'group' ELSE 'transaction' END,
		       COALESCE(e.group_id, t.group_id, e.transaction_id)::text
		FROM bank_statement_entries e
		LEFT JOIN financial_transactions t ON t.id = e.transaction_id
		WHERE e.import_id = $1::uuid
		  AND e.posted_at = $2::date
		  AND e.status IN ('conciliado','lancamento_gerado')
		  AND COALESCE(e.group_id, t.group_id, e.transaction_id) IS NOT NULL
		ORDER BY COALESCE(e.sequence_no, e.line_no), e.line_no, e.created_at, e.id`, importID, date)
	if err != nil {
		return err
	}
	fileOrder := []line{}
	for rows.Next() {
		var l line
		var seq int
		if err := rows.Scan(&seq, &l.kind, &l.id); err != nil {
			rows.Close()
			return err
		}
		l.key = l.kind + "|" + l.id
		fileOrder = append(fileOrder, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// Linhas visiveis atuais do escopo.
	rows, err = tx.Query(ctx, `
		SELECT 'transaction', t.id::text, t.entry_seq
		FROM financial_transactions t
		WHERE t.tenant_id = $1::uuid AND t.branch_id = $2::uuid
		  AND t.account_id = $3::uuid
		  AND (t.occurred_at AT TIME ZONE 'UTC')::date = $4::date
		  AND t.group_id IS NULL AND t.voided_at IS NULL
		UNION ALL
		SELECT 'group', g.id::text, g.entry_seq
		FROM financial_transaction_groups g
		WHERE g.tenant_id = $1::uuid AND g.branch_id = $2::uuid
		  AND g.account_id = $3::uuid AND g.occurred_at = $4::date
		ORDER BY 3 NULLS LAST, 2`, tenantID, branchID, accountID, date)
	if err != nil {
		return err
	}
	type currentLine struct {
		key  string
		kind string
		id   string
	}
	current := []currentLine{}
	indexByKey := map[string]int{}
	for rows.Next() {
		var c currentLine
		var seq *int
		if err := rows.Scan(&c.kind, &c.id, &seq); err != nil {
			rows.Close()
			return err
		}
		c.key = c.kind + "|" + c.id
		indexByKey[c.key] = len(current)
		current = append(current, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(current) == 0 {
		return nil
	}

	// Posicoes atualmente ocupadas pelas linhas do extrato. Elas sao mantidas;
	// apenas a ordem ENTRE elas muda, conforme o sequence_no.
	positions := []int{}
	linked := []currentLine{}
	used := map[string]bool{}
	for _, l := range fileOrder {
		idx, ok := indexByKey[l.key]
		if !ok || used[l.key] {
			continue
		}
		used[l.key] = true
		positions = append(positions, idx)
		linked = append(linked, current[idx])
	}
	if len(linked) == 0 {
		return nil
	}
	sort.Ints(positions)
	target := make([]currentLine, len(current))
	copy(target, current)
	for i, pos := range positions {
		target[pos] = linked[i]
	}

	changed := false
	for i := range target {
		if target[i].key != current[i].key {
			changed = true
			break
		}
	}
	if !changed {
		return nil
	}
	for i, l := range target {
		if _, err := tx.Exec(ctx, `SELECT fin_lines_set_seq($1, $2::uuid, $3)`, l.kind, l.id, i+1); err != nil {
			return err
		}
	}
	return nil
}

// MoveBankEntry troca a linha do extrato com a vizinha da mesma data (up/down)
// e realinha a sequencia dos lancamentos vinculados.
func (r *Repo) MoveBankEntry(ctx context.Context, tx pgx.Tx, importID, entryID, direction string) error {
	if direction != "up" && direction != "down" {
		return errors.New("direcao invalida (use up ou down)")
	}
	var posted, accountID string
	if err := tx.QueryRow(ctx, `
		SELECT to_char(e.posted_at,'YYYY-MM-DD'), COALESCE(i.account_id::text,'')
		FROM bank_statement_entries e
		JOIN bank_statement_imports i ON i.id = e.import_id
		WHERE e.id = $1::uuid AND e.import_id = $2::uuid`, entryID, importID).
		Scan(&posted, &accountID); err != nil {
		return err
	}

	rows, err := tx.Query(ctx, `
		SELECT id::text FROM bank_statement_entries
		WHERE import_id = $1::uuid AND posted_at = $2::date
		ORDER BY sequence_no, line_no, created_at, id`, importID, posted)
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	idx := -1
	for i, id := range ids {
		if id == entryID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return pgx.ErrNoRows
	}
	swap := idx - 1
	if direction == "down" {
		swap = idx + 1
	}
	if swap < 0 || swap >= len(ids) {
		return nil
	}
	ids[idx], ids[swap] = ids[swap], ids[idx]

	for i, id := range ids {
		if _, err := tx.Exec(ctx, `
			UPDATE bank_statement_entries SET sequence_no = $2, updated_at = now()
			WHERE id = $1::uuid`, id, i+1); err != nil {
			return err
		}
	}
	if accountID != "" {
		return r.alignBankDateSequence(ctx, tx, importID, accountID, posted)
	}
	return nil
}

// ReorderBankEntries aplica a ordem completa de uma data (usada pelo modal de
// geracao de pendentes). entry_ids precisa conter todas as linhas da data.
func (r *Repo) ReorderBankEntries(ctx context.Context, tx pgx.Tx, importID, date string, entryIDs []string) error {
	if strings.TrimSpace(date) == "" {
		return errors.New("informe a data")
	}
	var accountID string
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(account_id::text,'') FROM bank_statement_imports WHERE id = $1::uuid`, importID).
		Scan(&accountID); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `
		SELECT id::text FROM bank_statement_entries
		WHERE import_id = $1::uuid AND posted_at = $2::date
		ORDER BY sequence_no, line_no, created_at, id`, importID, date)
	if err != nil {
		return err
	}
	all := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		all = append(all, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(entryIDs) != len(all) {
		return errors.New("a ordem precisa incluir todas as linhas da data")
	}
	seen := map[string]bool{}
	for _, id := range entryIDs {
		if seen[id] {
			return errors.New("linha duplicada na reordenacao")
		}
		seen[id] = true
	}
	for _, id := range all {
		if !seen[id] {
			return errors.New("a ordem precisa incluir todas as linhas da data")
		}
	}
	for i, id := range entryIDs {
		if _, err := tx.Exec(ctx, `
			UPDATE bank_statement_entries SET sequence_no = $2, updated_at = now()
			WHERE id = $1::uuid`, id, i+1); err != nil {
			return err
		}
	}
	if accountID != "" {
		return r.alignBankDateSequence(ctx, tx, importID, accountID, date)
	}
	return nil
}

type NewGroupTransactionInput struct {
	CategoryID    string  `json:"category_id"`
	Amount        float64 `json:"amount"`
	Description   string  `json:"description"`
	DonorMemberID string  `json:"donor_member_id"`
	BenefactorID  string  `json:"benefactor_id"`
	DonorName     string  `json:"donor_name"`
	SupplierID    string  `json:"supplier_id"`
	PaymentMethod string  `json:"payment_method"`
	IsAnonymous   bool    `json:"is_anonymous"`
}

type GroupFromBankEntryInput struct {
	TransactionIDs  []string                   `json:"transaction_ids"`
	NewTransactions []NewGroupTransactionInput `json:"new_transactions"`
	Description     string                     `json:"description"`
}

func optString(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// GroupFromBankEntry cria um grupo a partir de uma entrada do extrato. A soma
// dos lancamentos precisa bater com o valor da entrada. A entrada fica
// conciliada e o grupo e conciliado automaticamente.
func (r *Repo) GroupFromBankEntry(ctx context.Context, tx pgx.Tx, tenantID, actorID, importID, entryID string, in GroupFromBankEntryInput) (*TransactionGroup, error) {
	var posted, direction, accountID, branchID, status string
	var amount float64
	var memo *string
	err := tx.QueryRow(ctx, `
		SELECT to_char(e.posted_at,'YYYY-MM-DD'), e.direction, e.amount::float8, e.memo,
		       COALESCE(i.account_id::text,''), i.branch_id::text, e.status
		FROM bank_statement_entries e
		JOIN bank_statement_imports i ON i.id = e.import_id
		WHERE e.id = $1::uuid AND e.import_id = $2::uuid`, entryID, importID).
		Scan(&posted, &direction, &amount, &memo, &accountID, &branchID, &status)
	if err != nil {
		return nil, err
	}
	if status != "pendente" {
		return nil, errors.New("entrada do extrato nao esta pendente")
	}
	if accountID == "" {
		return nil, ErrAccountRequired
	}
	ids := uniqueIDs(in.TransactionIDs)

	// Novos lancamentos criados no proprio modal de agrupamento. Tipo, conta e
	// data sao herdados da linha do extrato; categoria/valor/descricao/doador
	// vem do formulario.
	for _, nt := range in.NewTransactions {
		if strings.TrimSpace(nt.CategoryID) == "" {
			return nil, errors.New("selecione a conta contabil dos novos lancamentos")
		}
		if nt.Amount <= 0 {
			return nil, errors.New("o valor dos novos lancamentos deve ser positivo")
		}
		donorMemberID := optString(nt.DonorMemberID)
		benefactorID := optString(nt.BenefactorID)
		donorName := optString(nt.DonorName)
		supplierID := optString(nt.SupplierID)
		if direction != "income" {
			donorMemberID, benefactorID, donorName = nil, nil, nil
		}
		if direction != "expense" {
			supplierID = nil
		}
		if donorMemberID != nil {
			benefactorID, donorName = nil, nil
		} else if benefactorID != nil {
			donorName = nil
		}
		isAnonymous := nt.IsAnonymous && donorMemberID == nil && benefactorID == nil && donorName == nil
		created, _, _, _, err := r.Create(ctx, tx, tenantID, branchID, CreateTxnInput{
			Type:          direction,
			Amount:        nt.Amount,
			CategoryID:    optString(nt.CategoryID),
			AccountID:     &accountID,
			Description:   optString(nt.Description),
			OccurredAt:    &posted,
			DonorMemberID: donorMemberID,
			BenefactorID:  benefactorID,
			DonorName:     donorName,
			SupplierID:    supplierID,
			PaymentMethod: optString(nt.PaymentMethod),
			IsAnonymous:   isAnonymous,
			ActorID:       actorID,
		})
		if err != nil {
			return nil, err
		}
		ids = append(ids, created.ID)
	}
	ids = uniqueIDs(ids)
	if len(ids) < 2 {
		return nil, ErrGroupMinMembers
	}

	var sum float64
	var found int
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount),0)::float8, count(*)::int
		FROM financial_transactions
		WHERE id = ANY($1::uuid[]) AND tenant_id = $2::uuid
		  AND account_id = $3::uuid AND type = $4
		  AND voided_at IS NULL AND group_id IS NULL AND reconciled_at IS NULL
		  AND account_transfer_id IS NULL
		  AND NOT EXISTS (
		      SELECT 1 FROM bank_statement_entries e
		      WHERE e.transaction_id = financial_transactions.id
		        AND e.status IN ('conciliado','lancamento_gerado'))`,
		ids, tenantID, accountID, direction).Scan(&sum, &found); err != nil {
		return nil, err
	}
	if found != len(ids) {
		return nil, errors.New("um ou mais lancamentos nao conferem com a conta/tipo do extrato")
	}
	var mismatched int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)::int FROM financial_transactions
		WHERE id = ANY($1::uuid[])
		  AND (occurred_at AT TIME ZONE 'UTC')::date <> $2::date`, ids, posted).Scan(&mismatched); err != nil {
		return nil, err
	}
	if mismatched > 0 {
		return nil, fmt.Errorf("%w: ajuste a data dos lancamentos para %s antes de agrupar",
			ErrGroupDateMismatch, posted)
	}
	if math.Abs(sum-amount) >= 0.005 {
		return nil, errors.New("a soma dos lancamentos nao confere com o valor do extrato")
	}

	desc := strings.TrimSpace(in.Description)
	if desc == "" && memo != nil {
		desc = strings.TrimSpace(*memo)
	}
	if desc == "" {
		desc = "Deposito agrupado"
	}
	group, err := r.CreateGroup(ctx, tx, tenantID, actorID, CreateGroupInput{
		TransactionIDs: ids,
		OccurredAt:     posted,
		Description:    &desc,
		AccountID:      &accountID,
	})
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE bank_statement_entries
		SET status = 'conciliado', group_id = $2::uuid, updated_at = now()
		WHERE id = $1::uuid`, entryID, group.ID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE financial_transaction_groups
		SET bank_entry_id = $2::uuid
		WHERE id = $1::uuid`, group.ID, entryID); err != nil {
		return nil, err
	}
	group, err = r.ReconcileGroup(ctx, tx, group.ID, true, actorID)
	if err != nil {
		return nil, err
	}
	var impID string
	if err := tx.QueryRow(ctx, `SELECT import_id::text FROM bank_statement_entries WHERE id = $1::uuid`, entryID).Scan(&impID); err != nil {
		return nil, err
	}
	if err := refreshBankCounters(ctx, tx, impID); err != nil {
		return nil, err
	}
	return group, nil
}

// SequenceLine e uma linha usada na conferencia de sequencia (extrato x razao).
type SequenceLine struct {
	Kind        string  `json:"kind"`
	ID          string  `json:"id"`
	LineNo      int     `json:"line_no,omitempty"`
	EntrySeq    *int    `json:"entry_seq,omitempty"`
	Amount      float64 `json:"amount"`
	Direction   string  `json:"direction"`
	Description *string `json:"description,omitempty"`
}

// SequenceSwap indica que a linha "from" deveria ocupar o lugar da "to".
type SequenceSwap struct {
	FromKind string `json:"from_kind"`
	FromID   string `json:"from_id"`
	ToKind   string `json:"to_kind"`
	ToID     string `json:"to_id"`
}

// SequenceSuggestion e a comparacao de ordem de uma data entre o arquivo
// bancario e o razao, com a ordem sugerida para igualar ao extrato.
type SequenceSuggestion struct {
	Date         string         `json:"date"`
	AccountID    string         `json:"account_id"`
	Bank         []SequenceLine `json:"bank"`
	Ledger       []SequenceLine `json:"ledger"`
	Suggested    []LineRef      `json:"suggested"`
	Swaps        []SequenceSwap `json:"swaps"`
	NeedsReorder bool           `json:"needs_reorder"`
	Applicable   bool           `json:"applicable"`
	Reason       string         `json:"reason,omitempty"`
}

func cents(v float64) int64 {
	return int64(math.Round(v * 100))
}

func sequenceKey(kind, id string) string { return kind + "|" + id }

// BankSequenceSuggestions compara, por data, a ordem das linhas do extrato
// (line_no) com a ordem do razao (entry_seq) e devolve as trocas sugeridas.
func (r *Repo) BankSequenceSuggestions(ctx context.Context, tx pgx.Tx, importID string) ([]SequenceSuggestion, error) {
	var tenantID, branchID, accountID string
	if err := tx.QueryRow(ctx, `
		SELECT tenant_id::text, branch_id::text, COALESCE(account_id::text,'')
		FROM bank_statement_imports WHERE id = $1::uuid`, importID).
		Scan(&tenantID, &branchID, &accountID); err != nil {
		return nil, err
	}
	if accountID == "" {
		return []SequenceSuggestion{}, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT DISTINCT to_char(posted_at,'YYYY-MM-DD')
		FROM bank_statement_entries
		WHERE import_id = $1::uuid
		  AND status IN ('conciliado','lancamento_gerado')
		ORDER BY 1`, importID)
	if err != nil {
		return nil, err
	}
	dates := []string{}
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			rows.Close()
			return nil, err
		}
		dates = append(dates, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := []SequenceSuggestion{}
	for _, date := range dates {
		s, err := r.sequenceSuggestion(ctx, tx, importID, tenantID, branchID, accountID, date)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, nil
}

func (r *Repo) sequenceSuggestion(ctx context.Context, tx pgx.Tx, importID, tenantID, branchID, accountID, date string) (*SequenceSuggestion, error) {
	s := &SequenceSuggestion{Date: date, AccountID: accountID}

	// Lado do banco: linhas vinculadas, na ordem do arquivo.
	rows, err := tx.Query(ctx, `
		SELECT e.line_no, e.amount::float8, e.direction, e.memo,
		       CASE WHEN COALESCE(e.group_id, t.group_id) IS NOT NULL THEN 'group' ELSE 'transaction' END,
		       COALESCE(e.group_id, t.group_id, e.transaction_id)::text
		FROM bank_statement_entries e
		LEFT JOIN financial_transactions t ON t.id = e.transaction_id
		WHERE e.import_id = $1::uuid
		  AND e.posted_at = $2::date
		  AND e.status IN ('conciliado','lancamento_gerado')
		  AND COALESCE(e.group_id, t.group_id, e.transaction_id) IS NOT NULL
		ORDER BY COALESCE(e.sequence_no, e.line_no), e.line_no, e.created_at, e.id`, importID, date)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for rows.Next() {
		var l SequenceLine
		if err := rows.Scan(&l.LineNo, &l.Amount, &l.Direction, &l.Description, &l.Kind, &l.ID); err != nil {
			rows.Close()
			return nil, err
		}
		if seen[sequenceKey(l.Kind, l.ID)] {
			continue
		}
		seen[sequenceKey(l.Kind, l.ID)] = true
		s.Bank = append(s.Bank, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Lado do razao: linhas visiveis do escopo (avulsos + grupos).
	rows, err = tx.Query(ctx, `
		SELECT 'transaction', t.id::text, t.entry_seq, t.amount::float8, t.type,
		       COALESCE(t.description, t.donor_name)
		FROM financial_transactions t
		WHERE t.tenant_id = $1::uuid AND t.branch_id = $2::uuid
		  AND t.account_id = $3::uuid
		  AND (t.occurred_at AT TIME ZONE 'UTC')::date = $4::date
		  AND t.group_id IS NULL AND t.voided_at IS NULL
		UNION ALL
		SELECT 'group', g.id::text, g.entry_seq, COALESCE(agg.total,0)::float8, g.type,
		       g.description
		FROM financial_transaction_groups g
		LEFT JOIN LATERAL (
		    SELECT SUM(t.amount) AS total
		    FROM financial_transactions t
		    WHERE t.group_id = g.id AND t.voided_at IS NULL
		) agg ON true
		WHERE g.tenant_id = $1::uuid AND g.branch_id = $2::uuid
		  AND g.account_id = $3::uuid AND g.occurred_at = $4::date
		ORDER BY 3 NULLS LAST, 2`, tenantID, branchID, accountID, date)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var l SequenceLine
		if err := rows.Scan(&l.Kind, &l.ID, &l.EntrySeq, &l.Amount, &l.Direction, &l.Description); err != nil {
			rows.Close()
			return nil, err
		}
		s.Ledger = append(s.Ledger, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Periodo encerrado nao pode ser reordenado.
	var closed bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM financial_reconciliations fr
			WHERE fr.tenant_id = $1::uuid AND fr.branch_id = $2::uuid
			  AND fr.status = 'conciliada'
			  AND $3::date BETWEEN fr.period_start AND fr.period_end
			UNION ALL
			SELECT 1 FROM financial_audits a
			WHERE a.tenant_id = $1::uuid AND a.branch_id = $2::uuid
			  AND a.status = 'fechada'
			  AND $3::date BETWEEN a.period_start AND a.period_end
		)`, tenantID, branchID, date).Scan(&closed); err != nil {
		return nil, err
	}
	s.Applicable = !closed
	if closed {
		s.Reason = "periodo encerrado: nao e possivel reordenar"
	}

	// Confere cada linha do extrato contra a linha do razao que ela aponta e
	// monta a ordem-alvo. As posicoes ocupadas por linhas vinculadas sao
	// reordenadas conforme o arquivo; as demais linhas do razao permanecem nos
	// seus lugares (inclusive quando ha lancamentos fora do extrato).
	ledgerByKey := make(map[string]int, len(s.Ledger))
	for i, l := range s.Ledger {
		ledgerByKey[sequenceKey(l.Kind, l.ID)] = i
	}
	positions := make([]int, 0, len(s.Bank))
	linked := make([]SequenceLine, 0, len(s.Bank))
	usedPos := map[int]bool{}
	for _, b := range s.Bank {
		idx, ok := ledgerByKey[sequenceKey(b.Kind, b.ID)]
		if !ok {
			continue
		}
		l := s.Ledger[idx]
		if l.Direction != b.Direction || cents(l.Amount) != cents(b.Amount) {
			s.Applicable = false
			s.Reason = "os valores do extrato nao conferem com o razao nesta data"
			return s, nil
		}
		if usedPos[idx] {
			continue
		}
		usedPos[idx] = true
		positions = append(positions, idx)
		linked = append(linked, l)
	}
	if len(linked) == 0 {
		s.Reason = "nenhuma linha do extrato vinculada ao razao nesta data"
		return s, nil
	}
	sort.Ints(positions)
	target := make([]SequenceLine, len(s.Ledger))
	copy(target, s.Ledger)
	for i, pos := range positions {
		target[pos] = linked[i]
	}

	needs := false
	s.Suggested = make([]LineRef, 0, len(target))
	for i := range target {
		s.Suggested = append(s.Suggested, LineRef{Kind: target[i].Kind, ID: target[i].ID})
		if sequenceKey(target[i].Kind, target[i].ID) != sequenceKey(s.Ledger[i].Kind, s.Ledger[i].ID) {
			needs = true
			s.Swaps = append(s.Swaps, SequenceSwap{
				FromKind: s.Ledger[i].Kind, FromID: s.Ledger[i].ID,
				ToKind: target[i].Kind, ToID: target[i].ID,
			})
		}
	}
	s.NeedsReorder = needs
	return s, nil
}

// ApplyBankSequence aplica a ordem sugerida para uma data do extrato.
func (r *Repo) ApplyBankSequence(ctx context.Context, tx pgx.Tx, importID, date string, refs []LineRef) error {
	if len(refs) == 0 {
		return errors.New("nenhuma linha informada")
	}
	var tenantID, branchID, accountID string
	if err := tx.QueryRow(ctx, `
		SELECT tenant_id::text, branch_id::text, COALESCE(account_id::text,'')
		FROM bank_statement_imports WHERE id = $1::uuid`, importID).
		Scan(&tenantID, &branchID, &accountID); err != nil {
		return err
	}
	if accountID == "" {
		return ErrAccountRequired
	}
	seen := map[string]bool{}
	for i, ref := range refs {
		kind, err := normalizeLineKind(ref.Kind)
		if err != nil {
			return err
		}
		id := strings.TrimSpace(ref.ID)
		if id == "" {
			return errors.New("linha sem id")
		}
		key := kind + "|" + id
		if seen[key] {
			return errors.New("linha duplicada na reordenacao")
		}
		seen[key] = true

		var rowTenant, rowBranch, rowAccount, rowDate string
		if kind == "group" {
			err = tx.QueryRow(ctx, `
				SELECT tenant_id::text, branch_id::text, account_id::text, occurred_at::text
				FROM financial_transaction_groups WHERE id = $1::uuid`, id).
				Scan(&rowTenant, &rowBranch, &rowAccount, &rowDate)
		} else {
			err = tx.QueryRow(ctx, `
				SELECT tenant_id::text, branch_id::text, COALESCE(account_id::text,''),
				       (occurred_at AT TIME ZONE 'UTC')::date::text
				FROM financial_transactions WHERE id = $1::uuid AND group_id IS NULL`, id).
				Scan(&rowTenant, &rowBranch, &rowAccount, &rowDate)
		}
		if err != nil {
			return err
		}
		if rowTenant != tenantID || rowBranch != branchID || rowAccount != accountID || rowDate != date {
			return errors.New("as linhas devem ser da mesma filial, conta e data do extrato")
		}
		if _, err := tx.Exec(ctx, `SELECT fin_lines_set_seq($1, $2::uuid, $3)`, kind, id, i+1); err != nil {
			return err
		}
	}
	return nil
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
