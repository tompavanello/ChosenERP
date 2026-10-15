package members

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

// ImportError aponta a linha e o motivo de um membro rejeitado na importacao.
type ImportError struct {
	Line  int    `json:"line"`
	Error string `json:"error"`
}

// ImportResult resume uma importacao de membros em lote.
type ImportResult struct {
	Imported   int           `json:"imported"`
	Skipped    int           `json:"skipped"`
	Errors     []ImportError `json:"errors"`
	CreatedIDs []string      `json:"created_ids,omitempty"`
}

// ParseSheet le CSV ou XLSX e devolve as linhas como texto.
func ParseSheet(data []byte, filename string) ([][]string, error) {
	if len(data) >= 2 && data[0] == 'P' && data[1] == 'K' {
		return xlsx.Read(data)
	}
	if strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
		return xlsx.Read(data)
	}
	return parseMemberCSV(string(data))
}

func parseMemberCSV(body string) ([][]string, error) {
	body = strings.TrimPrefix(body, "\ufeff")
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("arquivo vazio")
	}
	delim := ';'
	if !strings.Contains(memberFirstLine(body), ";") {
		delim = ','
	}
	rd := csv.NewReader(strings.NewReader(body))
	rd.Comma = delim
	rd.FieldsPerRecord = -1
	rd.TrimLeadingSpace = true
	return rd.ReadAll()
}

func memberFirstLine(body string) string {
	if i := strings.IndexByte(body, '\n'); i >= 0 {
		return body[:i]
	}
	return body
}

func memberEmptyRecord(rec []string) bool {
	for _, c := range rec {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// ImportRecords importa linhas ja extraidas (CSV/XLSX). `mapping` mapeia campo
// canonico -> indice da coluna. `quota`, quando informado, e chamado antes de
// cada insercao (para respeitar o limite do plano); ao falhar, interrompe o lote.
func (r *Repo) ImportRecords(ctx context.Context, tx pgx.Tx, tenantID, branchID, actorID string, records [][]string, startRow int, mapping map[string]int, quota func() error) (ImportResult, error) {
	res := ImportResult{Errors: []ImportError{}}
	if startRow < 0 {
		startRow = 0
	}
	get := func(rec []string, field string) string {
		j, ok := mapping[field]
		if !ok || j < 0 || j >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[j])
	}
	addErr := func(line int, format string, args ...any) {
		res.Skipped++
		res.Errors = append(res.Errors, ImportError{Line: line, Error: fmt.Sprintf(format, args...)})
	}

	for i := startRow; i < len(records); i++ {
		rec := records[i]
		if memberEmptyRecord(rec) {
			continue
		}
		line := i + 1
		in, err := buildMemberInput(rec, get)
		if err != nil {
			addErr(line, "%v", err)
			continue
		}
		if quota != nil {
			if err := quota(); err != nil {
				addErr(line, "%v", err)
				break
			}
		}
		created, err := r.Create(ctx, tx, tenantID, branchID, in, actorID)
		if err != nil {
			addErr(line, "%v", err)
			continue
		}
		res.Imported++
		res.CreatedIDs = append(res.CreatedIDs, created.ID)
	}
	return res, nil
}

// buildMemberInput monta um CreateInput a partir de uma linha, com o mapeamento.
func buildMemberInput(rec []string, get func([]string, string) string) (CreateInput, error) {
	first := get(rec, "first_name")
	last := get(rec, "last_name")
	if full := get(rec, "full_name"); full != "" {
		parts := strings.Fields(full)
		if len(parts) > 0 {
			first = parts[0]
			last = strings.Join(parts[1:], " ")
		}
	}
	if first == "" {
		return CreateInput{}, fmt.Errorf("nome do membro e obrigatorio")
	}

	in := CreateInput{
		FirstName:     first,
		LastName:      last,
		Email:         opt(get(rec, "email")),
		Phone:         opt(get(rec, "phone")),
		Whatsapp:      opt(get(rec, "whatsapp")),
		Profession:    opt(get(rec, "profession")),
		Nationality:   opt(get(rec, "nationality")),
		Education:     opt(get(rec, "education")),
		Notes:         opt(get(rec, "notes")),
		CPF:           opt(get(rec, "cpf")),
		RG:            opt(get(rec, "rg")),
		Gender:        normGender(get(rec, "gender")),
		MaritalStatus: normMarital(get(rec, "marital_status")),
		MembershipStatus: normMemberStatus(get(rec, "membership_status")),
	}
	if bd := get(rec, "birth_date"); bd != "" {
		t, err := parseMemberDate(bd)
		if err != nil {
			return CreateInput{}, fmt.Errorf("data de nascimento invalida: %q", bd)
		}
		s := t.Format("2006-01-02")
		in.BirthDate = &s
	}
	addr := Address{
		ZipCode:    get(rec, "address_zip_code"),
		Street:     get(rec, "address_street"),
		Number:     get(rec, "address_number"),
		Complement: get(rec, "address_complement"),
		District:   get(rec, "address_district"),
		City:       get(rec, "address_city"),
		State:      strings.ToUpper(get(rec, "address_state")),
	}
	if addr != (Address{}) {
		in.Address = &addr
	}
	return in, nil
}

func opt(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func normKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Map(deaccentMember, s)
	return s
}

func normGender(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	switch normKey(s) {
	case "male", "masculino", "m", "homem":
		return strp("male")
	case "female", "feminino", "f", "mulher":
		return strp("female")
	case "other", "outro", "outra":
		return strp("other")
	}
	return &s
}

func normMarital(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	switch normKey(s) {
	case "single", "solteiro", "solteira":
		return strp("single")
	case "married", "casado", "casada":
		return strp("married")
	case "divorced", "divorciado", "divorciada":
		return strp("divorced")
	case "widowed", "viuvo", "viuva":
		return strp("widowed")
	}
	return &s
}

func normMemberStatus(s string) string {
	switch normKey(s) {
	case "active", "professo", "ativo", "ativo professo", "batizado":
		return "active"
	default:
		return "member"
	}
}

func strp(s string) *string { return &s }

// parseMemberDate aceita AAAA-MM-DD, DD/MM/AAAA, DD-MM-AAAA e o serial numerico
// de data do Excel (celulas formatadas como data viram numero no xlsx).
func parseMemberDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, l := range []string{"2006-01-02", "02/01/2006", "02-01-2006", "2006/01/02", time.RFC3339} {
		if t, err := time.Parse(l, s); err == nil {
			return t, nil
		}
	}
	if n, err := strconv.Atoi(s); err == nil && n > 0 && n < 100000 {
		// Serial do Excel (sistema 1900): dia 1 = 1900-01-01, com o bug do
		// 1900-02-29 embutido no sistema, por isso a base e 1899-12-30.
		return time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC).AddDate(0, 0, n), nil
	}
	return time.Time{}, fmt.Errorf("formato de data nao reconhecido")
}

// deaccentMember remove acentos para comparar cabecalhos/valores.
func deaccentMember(r rune) rune {
	switch r {
	case '\u00e1', '\u00e0', '\u00e2', '\u00e3', '\u00e4', '\u00e5':
		return 'a'
	case '\u00e9', '\u00e8', '\u00ea', '\u00eb':
		return 'e'
	case '\u00ed', '\u00ec', '\u00ee', '\u00ef':
		return 'i'
	case '\u00f3', '\u00f2', '\u00f4', '\u00f5', '\u00f6':
		return 'o'
	case '\u00fa', '\u00f9', '\u00fb', '\u00fc':
		return 'u'
	case '\u00e7':
		return 'c'
	case '\u00f1':
		return 'n'
	}
	return r
}
