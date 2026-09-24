package httpapi

import (
	"encoding/csv"
	"net/http"
	"strings"

	"chosenerp/internal/xlsx"
)

// exportSection é um bloco de um relatório (título opcional + cabeçalho + linhas).
type exportSection struct {
	Title   string
	Headers []string
	Rows    [][]string
}

// writeReport serve um relatório em CSV, XLSX (Excel) ou PDF (HTML de impressão,
// que o navegador salva como PDF). Centraliza os três formatos para todos os
// relatórios.
func writeReport(w http.ResponseWriter, filenameBase, title, subtitle, format string, sections []exportSection) {
	switch format {
	case "xlsx", "excel":
		data, err := xlsx.Write(title, flattenForXLSX(title, subtitle, sections))
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

func flattenForXLSX(title, subtitle string, sections []exportSection) [][]string {
	var rows [][]string
	if title != "" {
		rows = append(rows, []string{title})
	}
	if subtitle != "" {
		rows = append(rows, []string{subtitle})
	}
	for _, s := range sections {
		if s.Title != "" {
			rows = append(rows, []string{s.Title})
		}
		if len(s.Headers) > 0 {
			rows = append(rows, s.Headers)
		}
		rows = append(rows, s.Rows...)
		rows = append(rows, nil)
	}
	return rows
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
h1{font-size:20px;margin:0 0 2px}
.sub{color:#6b7280;margin:0 0 20px;font-size:13px}
h2{font-size:14px;margin:20px 0 6px;color:#111827}
table{width:100%;border-collapse:collapse;font-size:13px;margin-bottom:6px}
th,td{text-align:left;padding:6px 8px;border-bottom:1px solid #e4e7ee}
th{background:#f8fafc}
.muted{color:#9ca3af}
.noprint{position:fixed;top:16px;right:16px;padding:8px 14px;border:0;border-radius:8px;background:#0ea5e9;color:#fff;cursor:pointer;font-size:13px}
@media print{.noprint{display:none}body{padding:0}h2{page-break-after:avoid}tr{page-break-inside:avoid}}
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
