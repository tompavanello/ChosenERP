package finance

import "testing"

// TestHeaderIndexEstorno garante que a coluna opcional `estorno` e reconhecida
// na importacao de planilhas (o campo vira sign=-1 no lancamento).
func TestHeaderIndexEstorno(t *testing.T) {
	idx := headerIndex([]string{"data", "tipo", "conta", "valor", "estorno", "conta_bancaria"})
	j, ok := idx["estorno"]
	if !ok || j != 4 {
		t.Fatalf("estorno: indice=%d ok=%v, queria 4", j, ok)
	}
}

func TestParseBoolEstorno(t *testing.T) {
	for _, s := range []string{"sim", "s", "1", "true", "yes", "x"} {
		if !parseBool(s) {
			t.Errorf("parseBool(%q) = false, queria true", s)
		}
	}
	for _, s := range []string{"nao", "n", "0", "false", ""} {
		if parseBool(s) {
			t.Errorf("parseBool(%q) = true, queria false", s)
		}
	}
}
