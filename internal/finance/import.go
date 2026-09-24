package finance

import (
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/xlsx"
)

// ImportError aponta a linha e o motivo de um lançamento rejeitado na importação.
type ImportError struct {
	Line  int    `json:"line"`
	Error string `json:"error"`
}

// ImportResult resume uma importação em lote.
type ImportResult struct {
	Imported int           `json:"imported"`
	Skipped  int           `json:"skipped"`
	Errors   []ImportError `json:"errors"`
}

type catInfo struct {
	ID   string
	Type string
}

// ParseSheet lê CSV ou XLSX e devolve as linhas como texto.
func ParseSheet(data []byte, filename string) ([][]string, error) {
	if len(data) >= 2 && data[0] == 'P' && data[1] == 'K' {
		return xlsx.Read(data)
	}
	if strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
		return xlsx.Read(data)
	}
	return parseCSV(string(data))
}

func parseCSV(body string) ([][]string, error) {
	body = strings.TrimPrefix(body, "\ufeff")
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("arquivo vazio")
	}
	delim := ';'
	if !strings.Contains(firstLine(body), ";") {
		delim = ','
	}
	rd := csv.NewReader(strings.NewReader(body))
	rd.Comma = delim
	rd.FieldsPerRecord = -1
	rd.TrimLeadingSpace = true
	return rd.ReadAll()
}

// ImportCSV importa lançamentos a partir de um CSV com cabeçalho padrão:
//
//	data;tipo;conta;valor;forma_pagamento;descricao;anonimo;conta_bancaria
//
// Linhas inválidas são ignoradas e devolvidas em `errors`.
func (r *Repo) ImportCSV(ctx context.Context, tx pgx.Tx, tenantID, branchID, actorID, body string) (ImportResult, error) {
	empty := ImportResult{Errors: []ImportError{}}
	records, err := parseCSV(body)
	if err != nil {
		return empty, err
	}
	if len(records) < 2 {
		return empty, fmt.Errorf("CSV não tem linhas de dados")
	}
	idx := headerIndex(records[0])
	for _, req := range []string{"data", "tipo", "conta", "valor"} {
		if _, ok := idx[req]; !ok {
			return empty, fmt.Errorf("coluna obrigatória ausente: %s", req)
		}
	}
	return r.ImportRecords(ctx, tx, tenantID, branchID, actorID, records, 1, idx)
}

// ImportRecords importa linhas já extraídas (CSV ou XLSX). `startRow` é o índice
// (0-based) da primeira linha de dados e `mapping` mapeia campo -> índice da
// coluna. Não emite recibo (carga em lote); o hash-chain é do trigger.
func (r *Repo) ImportRecords(ctx context.Context, tx pgx.Tx, tenantID, branchID, actorID string, records [][]string, startRow int, mapping map[string]int) (ImportResult, error) {
	res := ImportResult{Errors: []ImportError{}}
	if startRow < 0 {
		startRow = 0
	}
	catByCode, catByName, err := categoryIndex(ctx, tx)
	if err != nil {
		return res, err
	}
	acctByName, err := accountIndex(ctx, tx)
	if err != nil {
		return res, err
	}
	addErr := func(line int, format string, args ...any) {
		res.Skipped++
		res.Errors = append(res.Errors, ImportError{Line: line, Error: fmt.Sprintf(format, args...)})
	}
	get := func(rec []string, field string) string {
		j, ok := mapping[field]
		if !ok || j < 0 || j >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[j])
	}

	for i := startRow; i < len(records); i++ {
		rec := records[i]
		if emptyRecord(rec) {
			continue
		}
		line := i + 1

		typ, err := parseTipo(get(rec, "tipo"))
		if err != nil {
			addErr(line, "%v", err)
			continue
		}
		key := get(rec, "conta")
		cat, ok := catByCode[key]
		if !ok {
			cat, ok = catByName[strings.ToLower(key)]
		}
		if !ok {
			addErr(line, "conta %q não encontrada", key)
			continue
		}
		if cat.Type != typ {
			addErr(line, "conta %q é de %s, mas o lançamento é de %s", key, cat.Type, typ)
			continue
		}
		amount, err := parseValor(get(rec, "valor"))
		if err != nil || amount <= 0 {
			addErr(line, "valor inválido: %q", get(rec, "valor"))
			continue
		}
		when, err := parseData(get(rec, "data"))
		if err != nil {
			addErr(line, "%v", err)
			continue
		}

		acctID := ""
		if name := get(rec, "conta_bancaria"); name != "" {
			acctID = acctByName[strings.ToLower(name)]
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO financial_transactions
				(tenant_id, branch_id, category_id, type, amount, currency,
				 payment_method, description, occurred_at, is_anonymous, account_id)
			VALUES ($1, NULLIF($2,'')::uuid, $3::uuid, $4, $5, 'BRL',
			        NULLIF($6,''), NULLIF($7,''), $8, $9, NULLIF($10,'')::uuid)`,
			tenantID, branchID, cat.ID, typ, amount,
			normalizePayment(get(rec, "forma_pagamento")), get(rec, "descricao"), when,
			parseBool(get(rec, "anonimo")), acctID); err != nil {
			return res, err
		}
		res.Imported++
	}
	return res, nil
}

func firstLine(body string) string {
	if i := strings.IndexByte(body, '\n'); i >= 0 {
		return body[:i]
	}
	return body
}

func emptyRecord(rec []string) bool {
	for _, c := range rec {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// headerIndex mapeia as colunas do cabeçalho para nomes canônicos.
func headerIndex(header []string) map[string]int {
	idx := map[string]int{}
	for i, h := range header {
		switch normalizeHeader(h) {
		case "data", "dia", "date":
			idx["data"] = i
		case "tipo", "type":
			idx["tipo"] = i
		case "conta", "conta_contabil", "categoria", "codigo", "rubrica":
			idx["conta"] = i
		case "valor", "amount":
			idx["valor"] = i
		case "forma_pagamento", "pagamento":
			idx["forma_pagamento"] = i
		case "descricao", "historico", "observacao":
			idx["descricao"] = i
		case "anonimo", "anonima":
			idx["anonimo"] = i
		case "conta_bancaria", "banco":
			idx["conta_bancaria"] = i
		}
	}
	return idx
}

// normalizeHeader deixa minúsculo, sem acento e com espaços/traços virando '_'.
func normalizeHeader(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer(
		"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
		"é", "e", "ê", "e", "í", "i", "ó", "o", "ô", "o", "õ", "o",
		"ú", "u", "ç", "c",
	).Replace(s)
	s = strings.NewReplacer(" ", "_", "-", "_", ".", "").Replace(s)
	return s
}

func parseTipo(s string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "entrada", "receita", "income", "credito", "crédito":
		return "income", nil
	case "saida", "saída", "despesa", "expense", "debito", "débito":
		return "expense", nil
	}
	return "", fmt.Errorf("tipo inválido: %q (use entrada/saida)", s)
}

// parseValor aceita 1234.56, 1.234,56 e "R$ 1.234,56".
func parseValor(s string) (float64, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, "R$", ""))
	s = strings.ReplaceAll(s, " ", "")
	if s == "" {
		return 0, fmt.Errorf("valor vazio")
	}
	if strings.Contains(s, ",") {
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func parseData(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Now(), nil
	}
	for _, l := range []string{"2006-01-02", "02/01/2006", "02-01-2006", time.RFC3339} {
		if t, err := time.Parse(l, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("data inválida: %q (use AAAA-MM-DD ou DD/MM/AAAA)", s)
}

func normalizePayment(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "pix":
		return "pix"
	case "cartao", "cartão", "card":
		return "card"
	case "boleto":
		return "boleto"
	case "dinheiro", "cash", "especie", "espécie":
		return "cash"
	case "transferencia", "transferência", "transfer":
		return "transfer"
	}
	return ""
}

func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "sim", "s", "yes", "x":
		return true
	}
	return false
}

func categoryIndex(ctx context.Context, tx pgx.Tx) (map[string]catInfo, map[string]catInfo, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, code, name, type FROM financial_categories WHERE is_active`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	byCode := map[string]catInfo{}
	byName := map[string]catInfo{}
	for rows.Next() {
		var id, code, name, typ string
		if err := rows.Scan(&id, &code, &name, &typ); err != nil {
			return nil, nil, err
		}
		info := catInfo{ID: id, Type: typ}
		byCode[code] = info
		byName[strings.ToLower(name)] = info
	}
	return byCode, byName, rows.Err()
}

func accountIndex(ctx context.Context, tx pgx.Tx) (map[string]string, error) {
	rows, err := tx.Query(ctx, `SELECT id::text, name FROM financial_accounts WHERE is_active`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out[strings.ToLower(name)] = id
	}
	return out, rows.Err()
}
