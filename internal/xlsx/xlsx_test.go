package xlsx

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestColName(t *testing.T) {
	cases := map[int]string{0: "A", 1: "B", 25: "Z", 26: "AA", 27: "AB", 51: "AZ", 52: "BA", 701: "ZZ", 702: "AAA"}
	for in, want := range cases {
		if got := colName(in); got != want {
			t.Errorf("colName(%d) = %q, queria %q", in, got, want)
		}
	}
}

func TestWriteProducesValidZipWithParts(t *testing.T) {
	data, err := Write("Aniversariantes", [][]string{
		{"Dia", "Nome"},
		{"1", `Ana & "Bia"`},
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("zip inválido: %v", err)
	}
	found := map[string]bool{}
	for _, f := range zr.File {
		found[f.Name] = true
	}
	for _, want := range []string{
		"[Content_Types].xml", "_rels/.rels", "xl/workbook.xml",
		"xl/_rels/workbook.xml.rels", "xl/styles.xml", "xl/worksheets/sheet1.xml",
	} {
		if !found[want] {
			t.Errorf("parte ausente no xlsx: %s", want)
		}
	}

	rc, err := zr.Open("xl/worksheets/sheet1.xml")
	if err != nil {
		t.Fatalf("abrir sheet1: %v", err)
	}
	defer rc.Close()
	body, _ := io.ReadAll(rc)
	if !strings.Contains(string(body), "Ana &amp; &quot;Bia&quot;") {
		t.Errorf("conteúdo escapado não encontrado: %s", string(body))
	}
}
