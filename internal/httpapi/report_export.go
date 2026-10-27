package httpapi

import (
	"encoding/csv"
	"net/http"
	"strings"

	"chosenerp/internal/xlsx"
)

// exportSection e um bloco de um relatorio (titulo opcional + cabecalho + linhas).
type exportSection struct {
	Title   string
	Headers []string
	Rows    [][]string
}

// writeReport serve um relatorio em CSV, XLSX (Excel) ou PDF (HTML de impressao,
// que o navegador salva como PDF). Centraliza os tres formatos para todos os
// relatorios.
func writeReport(w http.ResponseWriter, filenameBase, title, subtitle, format string, sections []exportSection) {
	// Sem cache: o arquivo mudou de layout e um XLSX antigo em cache daria a
	// impressao de que a formatacao nao foi aplicada.
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	switch format {
	case "xlsx", "excel":
		data, err := xlsx.WriteSheet(buildSheet(title, subtitle, sections))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", `attachment; filename="`+filenameBase+`.xlsx"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	case "csv":
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+filenameBase+`.csv"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(csvFrom(title, subtitle, sections)))
	default: // pdf
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(printHTML(title, subtitle, sections)))
	}
}

// buildSheet monta a planilha formatada: titulo/subtitulo e titulos de secao
// mesclados na largura da tabela, cabecalho em negrito com fundo/borda e corpo
// alinhado (valores numericos/percentuais/moeda a direita). As colunas ganham
// largura proporcional ao maior conteudo e a impressao sai em paisagem.
func buildSheet(title, subtitle string, sections []exportSection) xlsx.Sheet {
	maxCols := 1
	for _, s := range sections {
		if len(s.Headers) > maxCols {
			maxCols = len(s.Headers)
		}
		for _, r := range s.Rows {
			if len(r) > maxCols {
				maxCols = len(r)
			}
		}
	}

	widths := make([]int, maxCols)
	var rows [][]xlsx.Cell
	var merges []xlsx.Merge

	addMerged := func(text string, style int) {
		row := make([]xlsx.Cell, maxCols)
		row[0] = xlsx.Cell{Text: text, Style: style}
		rows = append(rows, row)
		merges = append(merges, xlsx.Merge{Row: len(rows) - 1, ColFrom: 0, ColTo: maxCols - 1})
	}

	frozen := 0
	if title != "" {
		addMerged(title, xlsx.StyleTitle)
		frozen++
	}
	if subtitle != "" {
		addMerged(subtitle, xlsx.StyleSubtitle)
		frozen++
	}

	for _, s := range sections {
		if s.Title != "" {
			addMerged(s.Title, xlsx.StyleSection)
		}
		if len(s.Headers) > 0 {
			row := make([]xlsx.Cell, len(s.Headers))
			for j, h := range s.Headers {
				row[j] = xlsx.Cell{Text: h, Style: xlsx.StyleHeader}
				widths[j] = maxInt(widths[j], runeLen(h))
			}
			rows = append(rows, row)
		}
		for k, r := range s.Rows {
			row := make([]xlsx.Cell, len(r))
			zebra := k%2 == 1
			for j, v := range r {
				style := xlsx.StyleCellLeft
				if looksNumeric(v) {
					style = xlsx.StyleCellRight
				}
				if zebra {
					if style == xlsx.StyleCellRight {
						style = xlsx.StyleZebraRight
					} else {
						style = xlsx.StyleZebraLeft
					}
				}
				row[j] = xlsx.Cell{Text: v, Style: style}
				if j < maxCols {
					widths[j] = maxInt(widths[j], runeLen(v))
				}
			}
			rows = append(rows, row)
		}
		rows = append(rows, nil)
	}

	cols := make([]xlsx.Column, maxCols)
	for i, w := range widths {
		if w < 10 {
			w = 10
		}
		if w > 60 {
			w = 60
		}
		cols[i] = xlsx.Column{Width: float64(w + 2)}
	}

	return xlsx.Sheet{
		Name:       title,
		Columns:    cols,
		Rows:       rows,
		Merges:     merges,
		FreezeRows: frozen,
		Landscape:  true,
		FitToWidth: true,
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func runeLen(s string) int {
	return len([]rune(s))
}

// looksNumeric identifica valores ja formatados como numero, moeda ou
// percentual (ex.: "1.234,56", "-0,50", "12,5%", "3"), que ficam alinhados a
// direita na planilha.
func looksNumeric(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	s = strings.TrimSuffix(s, "%")
	s = strings.TrimPrefix(s, "-")
	seenDigit := false
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			seenDigit = true
		case r == '.' || r == ',':
		default:
			return false
		}
	}
	return seenDigit
}

func csvFrom(title, subtitle string, sections []exportSection) string {
	var b strings.Builder
	b.WriteString("\xEF\xBB\xBF") // BOM: Excel abre com acentos
	w := csv.NewWriter(&b)
	w.Comma = ';'
	if title != "" {
		_ = w.Write([]string{title})
	}
	if subtitle != "" {
		_ = w.Write([]string{subtitle})
	}
	_ = w.Write(nil)
	for _, s := range sections {
		if s.Title != "" {
			_ = w.Write([]string{s.Title})
		}
		if len(s.Headers) > 0 {
			_ = w.Write(s.Headers)
		}
		for _, row := range s.Rows {
			_ = w.Write(row)
		}
		_ = w.Write(nil)
	}
	w.Flush()
	return b.String()
}

func printHTML(title, subtitle string, sections []exportSection) string {
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><title>`)
	b.WriteString(htmlEscape(title))
	b.WriteString(`</title><style>
body{font-family:ui-sans-serif,system-ui,sans-serif;padding:32px;color:#0b1020}
h1{font-size:20px;margin:0 0 2px;color:#1e3a8a}
.sub{color:#475569;margin:0 0 20px;font-size:13px}
h2{font-size:14px;margin:20px 0 6px;color:#1e3a8a}
table{width:100%;border-collapse:collapse;font-size:13px;margin-bottom:6px}
th,td{text-align:left;padding:6px 8px;border-bottom:1px solid #e4e7ee}
th{background:#dbeafe;color:#1e3a8a;font-weight:700}
tbody tr:nth-child(even) td{background:#f1f5f9}
.muted{color:#9ca3af}
.noprint{position:fixed;top:16px;right:16px;padding:8px 14px;border:0;border-radius:8px;background:#0ea5e9;color:#fff;cursor:pointer;font-size:13px}
@media print{.noprint{display:none}body{padding:0}h2{page-break-after:avoid}tr{page-break-inside:avoid}th,tbody tr:nth-child(even) td{-webkit-print-color-adjust:exact;print-color-adjust:exact}}
</style></head><body>`)
	b.WriteString(`<button class="noprint" onclick="window.print()">Imprimir / Salvar em PDF</button>`)
	b.WriteString(`<h1>` + htmlEscape(title) + `</h1>`)
	if subtitle != "" {
		b.WriteString(`<p class="sub">` + htmlEscape(subtitle) + `</p>`)
	}
	for _, s := range sections {
		if s.Title != "" {
			b.WriteString(`<h2>` + htmlEscape(s.Title) + `</h2>`)
		}
		b.WriteString(`<table>`)
		if len(s.Headers) > 0 {
			b.WriteString(`<thead><tr>`)
			for _, h := range s.Headers {
				b.WriteString(`<th>` + htmlEscape(h) + `</th>`)
			}
			b.WriteString(`</tr></thead>`)
		}
		b.WriteString(`<tbody>`)
		if len(s.Rows) == 0 {
			b.WriteString(`<tr><td class="muted">Sem dados.</td></tr>`)
		}
		for _, row := range s.Rows {
			b.WriteString(`<tr>`)
			for _, c := range row {
				b.WriteString(`<td>` + htmlEscape(c) + `</td>`)
			}
			b.WriteString(`</tr>`)
		}
		b.WriteString(`</tbody></table>`)
	}
	b.WriteString(`</body></html>`)
	return b.String()
}
