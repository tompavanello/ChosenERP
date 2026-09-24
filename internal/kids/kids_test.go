package kids

import "testing"

func intp(v int) *int { return &v }

func TestValidateAges(t *testing.T) {
	if err := validateAges(intp(4), intp(8)); err != nil {
		t.Errorf("faixa valida rejeitada: %v", err)
	}
	if err := validateAges(nil, nil); err != nil {
		t.Errorf("faixa ausente deveria ser aceita: %v", err)
	}
	if err := validateAges(intp(8), intp(4)); err == nil {
		t.Error("max < min deveria falhar")
	}
	if err := validateAges(intp(-1), nil); err == nil {
		t.Error("idade negativa deveria falhar")
	}
}

func TestNewSecurityCode(t *testing.T) {
	for i := 0; i < 50; i++ {
		code := newSecurityCode()
		if len(code) != 4 {
			t.Fatalf("codigo com tamanho %d: %q", len(code), code)
		}
		for _, c := range code {
			if c < '0' || c > '9' {
				t.Fatalf("codigo nao numerico: %q", code)
			}
		}
	}
}
