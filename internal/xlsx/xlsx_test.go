package xlsx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
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
		t.Fatalf("zip invalido: %v", err)
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
		t.Errorf("conteudo escapado nao encontrado: %s", string(body))
	}
}

func TestWriteSheetIncludesStylesAndLayout(t *testing.T) {
	data, err := WriteSheet(Sheet{
		Name:    "Relatorio",
		Columns: []Column{{Width: 20}, {Width: 12}},
		Rows: [][]Cell{
			{{Text: "Titulo", Style: StyleTitle}},
			{{Text: "Indicador", Style: StyleHeader}, {Text: "Valor", Style: StyleHeader}},
			{{Text: "Saldo", Style: StyleCellLeft}, {Text: "1.234,56", Style: StyleCellRight}},
		},
		Merges:     []Merge{{Row: 0, ColFrom: 0, ColTo: 1}},
		FreezeRows: 1,
		Landscape:  true,
		FitToWidth: true,
	})
	if err != nil {
		t.Fatalf("WriteSheet: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("zip invalido: %v", err)
	}
	parts := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("abrir %s: %v", f.Name, err)
		}
		b, _ := io.ReadAll(rc)
		rc.Close()
		parts[f.Name] = string(b)
	}

	if !strings.Contains(parts["xl/styles.xml"], `<b/>`) {
		t.Errorf("styles.xml sem fonte negrito")
	}
	if !strings.Contains(parts["xl/styles.xml"], `patternType="solid"`) {
		t.Errorf("styles.xml sem preenchimento")
	}
	for _, color := range []string{"FF1E3A8A", "FF0F172A", "FFDBEAFE", "FFF1F5F9"} {
		if !strings.Contains(parts["xl/styles.xml"], color) {
			t.Errorf("styles.xml sem a cor %s", color)
		}
	}
	for _, name := range []string{"xl/styles.xml", "xl/worksheets/sheet1.xml"} {
		var v any
		if err := xml.Unmarshal([]byte(parts[name]), &v); err != nil {
			t.Errorf("%s nao e XML valido: %v", name, err)
		}
	}
	sheet := parts["xl/worksheets/sheet1.xml"]
	for _, want := range []string{
		`<cols>`, `<col min="1" max="1" width="20" customWidth="1"/>`,
		`<mergeCells count="1">`, `<mergeCell ref="A1:B1"/>`,
		`<pane ySplit="1"`, `orientation="landscape"`, `fitToWidth="1"`,
		`<c r="A1" s="1" t="inlineStr">`, `<c r="A2" s="4" t="inlineStr">`,
		`<c r="B3" s="6" t="inlineStr">`,
	} {
		if !strings.Contains(sheet, want) {
			t.Errorf("sheet1.xml nao contem %q", want)
		}
	}
}
