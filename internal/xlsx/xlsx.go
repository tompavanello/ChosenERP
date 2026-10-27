// Package xlsx gera planilhas .xlsx validas (OOXML) sem depender de bibliotecas
// externas - o binario da API e FROM scratch e o projeto evita new deps.
//
// Implementa o subconjunto minimo do formato: um unico worksheet com celulas de
// texto inline (inlineStr). Numeros continuam como texto de proposito: os
// relatorios ja formatam valores (R$ / percentuais) e a planilha e para leitura,
// nao para recalculo. O visual (negrito, fundo, bordas, larguras, mesclas,
// congelamento e impressao) e aplicado por estilos OOXML.
package xlsx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// Estilos de celula disponiveis no styles.xml abaixo. StyleDefault nao aplica
// atributo `s` e mantem o comportamento antigo (texto simples).
const (
	StyleDefault = iota
	StyleTitle
	StyleSubtitle
	StyleSection
	StyleHeader
	StyleCellLeft
	StyleCellRight
	StyleCellCenter
	StyleZebraLeft
	StyleZebraRight
	StyleZebraCenter
)

// Cell e uma celula de texto com o estilo visual.
type Cell struct {
	Text  string
	Style int
}

// Column define a largura de uma coluna (em caracteres aproximados).
type Column struct {
	Width float64
}

// Merge mescla as colunas [ColFrom, ColTo] de uma linha (indices 0-based).
type Merge struct {
	Row     int
	ColFrom int
	ColTo   int
}

// Sheet descreve a planilha a ser gerada.
type Sheet struct {
	Name       string
	Columns    []Column
	Rows       [][]Cell
	Merges     []Merge
	FreezeRows int
	Landscape  bool
	FitToWidth bool
}

const (
	contentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
		`<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>` +
		`<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>` +
		`</Types>`

	rootRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
		`</Relationships>`

	stylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
		`<fonts count="5">` +
		`<font><sz val="11"/><color rgb="FF1F2937"/><name val="Calibri"/><family val="2"/></font>` +
		`<font><b/><sz val="11"/><color rgb="FFFFFFFF"/><name val="Calibri"/><family val="2"/></font>` +
		`<font><b/><sz val="14"/><color rgb="FFFFFFFF"/><name val="Calibri"/><family val="2"/></font>` +
		`<font><i/><sz val="10"/><color rgb="FF475569"/><name val="Calibri"/><family val="2"/></font>` +
		`<font><b/><sz val="12"/><color rgb="FF1E3A8A"/><name val="Calibri"/><family val="2"/></font>` +
		`</fonts>` +
		`<fills count="6">` +
		`<fill><patternFill patternType="none"/></fill>` +
		`<fill><patternFill patternType="gray125"/></fill>` +
		`<fill><patternFill patternType="solid"><fgColor rgb="FF1E3A8A"/><bgColor indexed="64"/></patternFill></fill>` +
		`<fill><patternFill patternType="solid"><fgColor rgb="FFDBEAFE"/><bgColor indexed="64"/></patternFill></fill>` +
		`<fill><patternFill patternType="solid"><fgColor rgb="FF0F172A"/><bgColor indexed="64"/></patternFill></fill>` +
		`<fill><patternFill patternType="solid"><fgColor rgb="FFF1F5F9"/><bgColor indexed="64"/></patternFill></fill>` +
		`</fills>` +
		`<borders count="2">` +
		`<border><left/><right/><top/><bottom/><diagonal/></border>` +
		`<border><left/><right/><top/><bottom style="thin"><color rgb="FF94A3B8"/></bottom><diagonal/></border>` +
		`</borders>` +
		`<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>` +
		`<cellXfs count="11">` +
		`<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>` +
		`<xf numFmtId="0" fontId="2" fillId="4" borderId="0" xfId="0" applyFont="1" applyFill="1" applyAlignment="1"><alignment vertical="center"/></xf>` +
		`<xf numFmtId="0" fontId="3" fillId="0" borderId="0" xfId="0" applyFont="1" applyAlignment="1"><alignment vertical="center"/></xf>` +
		`<xf numFmtId="0" fontId="4" fillId="3" borderId="1" xfId="0" applyFont="1" applyFill="1" applyBorder="1" applyAlignment="1"><alignment vertical="center"/></xf>` +
		`<xf numFmtId="0" fontId="1" fillId="2" borderId="1" xfId="0" applyFont="1" applyFill="1" applyBorder="1" applyAlignment="1"><alignment vertical="center" wrapText="1"/></xf>` +
		`<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0" applyAlignment="1"><alignment horizontal="left" vertical="center"/></xf>` +
		`<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0" applyAlignment="1"><alignment horizontal="right" vertical="center"/></xf>` +
		`<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0" applyAlignment="1"><alignment horizontal="center" vertical="center"/></xf>` +
		`<xf numFmtId="0" fontId="0" fillId="5" borderId="0" xfId="0" applyFill="1" applyAlignment="1"><alignment horizontal="left" vertical="center"/></xf>` +
		`<xf numFmtId="0" fontId="0" fillId="5" borderId="0" xfId="0" applyFill="1" applyAlignment="1"><alignment horizontal="right" vertical="center"/></xf>` +
		`<xf numFmtId="0" fontId="0" fillId="5" borderId="0" xfId="0" applyFill="1" applyAlignment="1"><alignment horizontal="center" vertical="center"/></xf>` +
		`</cellXfs>` +
		`<cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles>` +
		`</styleSheet>`
)

func workbookXML(sheetName string) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" ` +
		`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
		`<sheets><sheet name="` + xmlEsc(sheetName) + `" sheetId="1" r:id="rId1"/></sheets>` +
		`</workbook>`
}

const workbookRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>` +
	`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>` +
	`</Relationships>`

// Write monta o arquivo .xlsx a partir das linhas de texto (sem estilos).
// sheetName e o nome da aba (ate 31 caracteres no Excel).
func Write(sheetName string, rows [][]string) ([]byte, error) {
	cells := make([][]Cell, len(rows))
	for i, row := range rows {
		cells[i] = make([]Cell, len(row))
		for j, v := range row {
			cells[i][j] = Cell{Text: v}
		}
	}
	return WriteSheet(Sheet{Name: sheetName, Rows: cells})
}

// WriteSheet monta o arquivo .xlsx aplicando os estilos e o layout informados.
func WriteSheet(s Sheet) ([]byte, error) {
	sheetName := s.Name
	if sheetName == "" {
		sheetName = "Relatorio"
	}
	if len([]rune(sheetName)) > 31 {
		sheetName = string([]rune(sheetName)[:31])
	}

	maxCols := len(s.Columns)
	for _, row := range s.Rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}
	for _, m := range s.Merges {
		if m.ColTo+1 > maxCols {
			maxCols = m.ColTo + 1
		}
	}
	if maxCols < 1 {
		maxCols = 1
	}
	lastRow := len(s.Rows)
	if lastRow < 1 {
		lastRow = 1
	}

	var sheet strings.Builder
	sheet.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	sheet.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	if s.FitToWidth {
		sheet.WriteString(`<sheetPr><pageSetUpPr fitToPage="1"/></sheetPr>`)
	}
	fmt.Fprintf(&sheet, `<dimension ref="A1:%s%d"/>`, colName(maxCols-1), lastRow)
	if s.FreezeRows > 0 {
		fmt.Fprintf(&sheet, `<sheetViews><sheetView tabSelected="1" workbookViewId="0">`+
			`<pane ySplit="%d" topLeftCell="A%d" activePane="bottomLeft" state="frozen"/>`+
			`<selection pane="bottomLeft" activeCell="A%d" sqref="A%d"/>`+
			`</sheetView></sheetViews>`, s.FreezeRows, s.FreezeRows+1, s.FreezeRows+1, s.FreezeRows+1)
	} else {
		sheet.WriteString(`<sheetViews><sheetView workbookViewId="0"/></sheetViews>`)
	}
	sheet.WriteString(`<sheetFormatPr defaultRowHeight="15"/>`)
	if len(s.Columns) > 0 {
		sheet.WriteString(`<cols>`)
		for i, c := range s.Columns {
			width := c.Width
			if width <= 0 {
				width = 12
			}
			fmt.Fprintf(&sheet, `<col min="%d" max="%d" width="%s" customWidth="1"/>`,
				i+1, i+1, strconv.FormatFloat(width, 'f', -1, 64))
		}
		sheet.WriteString(`</cols>`)
	}
	sheet.WriteString(`<sheetData>`)
	for i, row := range s.Rows {
		fmt.Fprintf(&sheet, `<row r="%d">`, i+1)
		for j, cell := range row {
			if cell.Text == "" {
				continue
			}
			if cell.Style > StyleDefault {
				fmt.Fprintf(&sheet, `<c r="%s%d" s="%d" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`,
					colName(j), i+1, cell.Style, xmlEsc(cell.Text))
			} else {
				fmt.Fprintf(&sheet, `<c r="%s%d" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`,
					colName(j), i+1, xmlEsc(cell.Text))
			}
		}
		sheet.WriteString(`</row>`)
	}
	sheet.WriteString(`</sheetData>`)

	merges := make([]Merge, 0, len(s.Merges))
	for _, m := range s.Merges {
		if m.ColTo > m.ColFrom && m.Row >= 0 && m.Row < len(s.Rows) {
			merges = append(merges, m)
		}
	}
	if len(merges) > 0 {
		fmt.Fprintf(&sheet, `<mergeCells count="%d">`, len(merges))
		for _, m := range merges {
			fmt.Fprintf(&sheet, `<mergeCell ref="%s%d:%s%d"/>`,
				colName(m.ColFrom), m.Row+1, colName(m.ColTo), m.Row+1)
		}
		sheet.WriteString(`</mergeCells>`)
	}

	sheet.WriteString(`<pageMargins left="0.4" right="0.4" top="0.5" bottom="0.5" header="0.3" footer="0.3"/>`)
	if s.Landscape || s.FitToWidth {
		orientation := "portrait"
		if s.Landscape {
			orientation = "landscape"
		}
		if s.FitToWidth {
			fmt.Fprintf(&sheet, `<pageSetup paperSize="9" orientation="%s" fitToWidth="1" fitToHeight="0"/>`, orientation)
		} else {
			fmt.Fprintf(&sheet, `<pageSetup paperSize="9" orientation="%s"/>`, orientation)
		}
	}
	sheet.WriteString(`</worksheet>`)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	parts := []struct {
		name string
		data string
	}{
		{"[Content_Types].xml", contentTypes},
		{"_rels/.rels", rootRels},
		{"xl/workbook.xml", workbookXML(sheetName)},
		{"xl/_rels/workbook.xml.rels", workbookRels},
		{"xl/styles.xml", stylesXML},
		{"xl/worksheets/sheet1.xml", sheet.String()},
	}
	for _, p := range parts {
		f, err := zw.Create(p.name)
		if err != nil {
			return nil, err
		}
		if _, err := f.Write([]byte(p.data)); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// colName converte o indice 0-based da coluna em referencia (0->A, 26->AA).
func colName(n int) string {
	var s []byte
	for n >= 0 {
		s = append([]byte{byte('A' + n%26)}, s...)
		n = n/26 - 1
	}
	return string(s)
}

func xmlEsc(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&apos;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
