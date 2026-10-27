package httpapi

import (
	"testing"

	"chosenerp/internal/xlsx"
)

func TestBuildSheetLayout(t *testing.T) {
	sheet := buildSheet("Relatorio", "Periodo: 2026", []exportSection{{
		Title:   "Resumo",
		Headers: []string{"Indicador", "Valor"},
		Rows: [][]string{
			{"Saldo", "1.234,56"},
			{"Percentual", "10,5%"},
			{"Observacao", "texto"},
		},
	}})

	// 3 linhas mescladas (titulo, subtitulo e secao) + cabecalho + 3 linhas + espaco.
	if len(sheet.Rows) != 8 {
		t.Fatalf("linhas = %d, queria 8", len(sheet.Rows))
	}
	if len(sheet.Merges) != 3 {
		t.Fatalf("merges = %d, queria 3", len(sheet.Merges))
	}
	for i, m := range sheet.Merges {
		if m.Row != i || m.ColFrom != 0 || m.ColTo != 1 {
			t.Errorf("merge %d = %+v, queria Row=%d ColFrom=0 ColTo=1", i, m, i)
		}
	}
	if sheet.FreezeRows != 2 {
		t.Errorf("freeze = %d, queria 2", sheet.FreezeRows)
	}
	if !sheet.Landscape || !sheet.FitToWidth {
		t.Errorf("esperava paisagem e ajuste a largura")
	}
	if len(sheet.Columns) != 2 {
		t.Fatalf("colunas = %d, queria 2", len(sheet.Columns))
	}
	if sheet.Columns[0].Width < 12 {
		t.Errorf("largura da coluna 0 = %v, queria >= 12", sheet.Columns[0].Width)
	}

	if sheet.Rows[3][0].Style != xlsx.StyleHeader || sheet.Rows[3][1].Style != xlsx.StyleHeader {
		t.Errorf("cabecalho sem estilo de header: %+v", sheet.Rows[3])
	}
	if sheet.Rows[4][0].Style != xlsx.StyleCellLeft {
		t.Errorf("texto deveria ficar a esquerda: %+v", sheet.Rows[4][0])
	}
	if sheet.Rows[4][1].Style != xlsx.StyleCellRight {
		t.Errorf("moeda deveria ficar a direita: %+v", sheet.Rows[4][1])
	}
	if sheet.Rows[5][1].Style != xlsx.StyleZebraRight {
		t.Errorf("percentual zebra deveria ficar a direita: %+v", sheet.Rows[5][1])
	}
	if sheet.Rows[6][1].Style != xlsx.StyleCellLeft {
		t.Errorf("texto deveria ficar a esquerda: %+v", sheet.Rows[6][1])
	}
}

func TestLooksNumeric(t *testing.T) {
	cases := map[string]bool{
		"1.234,56": true,
		"-0,50":    true,
		"12,5%":    true,
		"12.5%":    true,
		"3":        true,
		"":         false,
		"texto":    false,
		"01/03":    false,
		"2026-01":  false,
		"R$ 10,00": false,
	}
	for in, want := range cases {
		if got := looksNumeric(in); got != want {
			t.Errorf("looksNumeric(%q) = %v, queria %v", in, got, want)
		}
	}
}
