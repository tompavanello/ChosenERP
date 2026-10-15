package members

import (
	"strings"
	"testing"
)

func mapper(m map[string]int) func([]string, string) string {
	return func(rec []string, field string) string {
		j, ok := m[field]
		if !ok || j < 0 || j >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[j])
	}
}

func TestBuildMemberInput_FullNameAndFields(t *testing.T) {
	rec := []string{"Joao da Silva", "joao@ex.com", "11988887777", "15/03/1990", "Masculino", "Casado", "123.456.789-09"}
	m := map[string]int{
		"full_name": 0, "email": 1, "phone": 2, "birth_date": 3,
		"gender": 4, "marital_status": 5, "cpf": 6,
	}
	in, err := buildMemberInput(rec, mapper(m))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if in.FirstName != "Joao" || in.LastName != "da Silva" {
		t.Fatalf("nome dividido errado: %q / %q", in.FirstName, in.LastName)
	}
	if in.BirthDate == nil || *in.BirthDate != "1990-03-15" {
		t.Fatalf("data: %v", in.BirthDate)
	}
	if in.Gender == nil || *in.Gender != "male" {
		t.Fatalf("genero: %v", in.Gender)
	}
	if in.MaritalStatus == nil || *in.MaritalStatus != "married" {
		t.Fatalf("estado civil: %v", in.MaritalStatus)
	}
	if in.Email == nil || *in.Email != "joao@ex.com" || in.CPF == nil {
		t.Fatalf("contato/cpf: %+v", in)
	}
	if in.MembershipStatus != "member" {
		t.Fatalf("situacao default deveria ser member: %q", in.MembershipStatus)
	}
}

func TestBuildMemberInput_MissingName(t *testing.T) {
	rec := []string{"", "x@y.z"}
	if _, err := buildMemberInput(rec, mapper(map[string]int{"email": 1})); err == nil {
		t.Fatal("esperava erro por falta de nome")
	}
}

func TestBuildMemberInput_Address(t *testing.T) {
	rec := []string{"Maria", "Souza", "01310-100", "Av. Paulista", "1000", "sala 5", "Bela Vista", "Sao Paulo", "sp"}
	m := map[string]int{
		"first_name": 0, "last_name": 1, "address_zip_code": 2, "address_street": 3,
		"address_number": 4, "address_complement": 5, "address_district": 6,
		"address_city": 7, "address_state": 8,
	}
	in, err := buildMemberInput(rec, mapper(m))
	if err != nil {
		t.Fatalf("erro: %v", err)
	}
	if in.Address == nil || in.Address.City != "Sao Paulo" || in.Address.State != "SP" {
		t.Fatalf("endereco: %+v", in.Address)
	}
}

func TestParseMemberDate(t *testing.T) {
	cases := map[string]string{
		"1990-03-15": "1990-03-15",
		"15/03/1990": "1990-03-15",
		"15-03-1990": "1990-03-15",
	}
	for in, want := range cases {
		got, err := parseMemberDate(in)
		if err != nil || got.Format("2006-01-02") != want {
			t.Errorf("%s => %v (%v), queria %s", in, got, err, want)
		}
	}
	// Serial do Excel: 1 = 1900-01-01; 44927 = 2023-01-01 (aprox.). Valida so a
	// conversao numerica sem depender do dia exato.
	if got, err := parseMemberDate("44927"); err != nil || got.Year() != 2023 {
		t.Errorf("serial excel: %v (%v)", got, err)
	}
	if _, err := parseMemberDate("abacaxi"); err == nil {
		t.Error("esperava erro para data invalida")
	}
}

func TestParseSheet_CSV(t *testing.T) {
	// CSV com ponto-e-virgula.
	rows, err := ParseSheet([]byte("Nome;E-mail\nJoao;a@b.c\n"), "membros.csv")
	if err != nil || len(rows) != 2 || rows[1][0] != "Joao" {
		t.Fatalf("csv ;: %v %v", rows, err)
	}
	// CSV com virgula.
	rows, err = ParseSheet([]byte("Nome,Email\nMaria,m@x.y\n"), "membros.csv")
	if err != nil || rows[1][1] != "m@x.y" {
		t.Fatalf("csv ,: %v %v", rows, err)
	}
}
