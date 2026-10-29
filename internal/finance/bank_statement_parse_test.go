package finance

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseStatementFiles usa os extratos reais de Docs/ofx. Os tres arquivos
// (OFX, OFC e TXT) tem 133 movimentos; o OFX/OFC repetem FITID/documento em
// varias linhas legitimas, que NAO podem ser descartadas.
func TestParseStatementFiles(t *testing.T) {
	cases := []struct {
		file   string
		format string
	}{
		{"ofx.ofx", "ofx"},
		{"ofc.ofc", "ofx"},
		{"txt.txt", "csv"},
	}

	var ofxEntries []parsedEntry
	var txtEntries []parsedEntry
	for _, tc := range cases {
		raw, err := os.ReadFile(filepath.Join("..", "..", "Docs", "ofx", tc.file))
		if err != nil {
			t.Fatalf("ler %s: %v", tc.file, err)
		}
		entries, format, err := parseStatement(tc.file, raw)
		if err != nil {
			t.Fatalf("parse %s: %v", tc.file, err)
		}
		if format != tc.format {
			t.Errorf("%s: formato = %s, queria %s", tc.file, format, tc.format)
		}
		if len(entries) != 133 {
			t.Errorf("%s: %d lancamentos, queria 133", tc.file, len(entries))
		}
		for i, e := range entries {
			if e.LineNo != i+1 {
				t.Fatalf("%s: line_no[%d] = %d, queria %d", tc.file, i, e.LineNo, i+1)
			}
			if e.Amount <= 0 {
				t.Errorf("%s: linha %d com valor <= 0 (%v)", tc.file, e.LineNo, e.Amount)
			}
			if e.Direction != "income" && e.Direction != "expense" {
				t.Errorf("%s: linha %d direcao invalida %q", tc.file, e.LineNo, e.Direction)
			}
		}
		switch tc.file {
		case "ofx.ofx":
			ofxEntries = entries
		case "txt.txt":
			txtEntries = entries
		}
	}

	// O mesmo FITID aparece varias vezes legitimamente no OFX.
	dup := 0
	for _, e := range ofxEntries {
		if e.FitID == "10610" {
			dup++
		}
	}
	if dup != 5 {
		t.Errorf("FITID 10610: %d ocorrencias, queria 5", dup)
	}

	// No TXT o sinal vem na coluna Deb_Cred (D/C), nao no valor.
	if len(txtEntries) > 0 {
		first := txtEntries[0]
		if first.Direction != "expense" || first.Amount != 1600 {
			t.Errorf("TXT primeira linha: %s %v, queria expense 1600", first.Direction, first.Amount)
		}
	}
	foundCredit := false
	for _, e := range txtEntries {
		if e.Direction == "income" && e.Amount == 750 {
			foundCredit = true
			break
		}
	}
	if !foundCredit {
		t.Errorf("TXT: nao encontrou o credito de 750")
	}
}
