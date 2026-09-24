package xlsx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Read extrai as linhas de uma planilha .xlsx (primeiro worksheet) como texto.
// Suporta sharedStrings, inlineStr e valores numericos. Nao avalia formulas
// (usa o ultimo valor calculado gravado no arquivo).
func Read(data []byte) ([][]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("xlsx invalido: %w", err)
	}
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	shared, err := readSharedStrings(files["xl/sharedStrings.xml"])
	if err != nil {
		return nil, err
	}
	sheetFile := files["xl/worksheets/sheet1.xml"]
	if sheetFile == nil {
		return nil, fmt.Errorf("xlsx sem worksheet")
	}
	return readSheet(sheetFile, shared)
}

func readSharedStrings(f *zip.File) ([]string, error) {
	if f == nil {
		return nil, nil
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var sst struct {
		SI []struct {
			T string   `xml:"t"`
			R []string `xml:"r>t"`
		} `xml:"si"`
	}
	if err := xml.NewDecoder(rc).Decode(&sst); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(sst.SI))
	for _, si := range sst.SI {
		txt := si.T
		if txt == "" && len(si.R) > 0 {
			txt = strings.Join(si.R, "")
		}
		out = append(out, txt)
	}
	return out, nil
}

func readSheet(f *zip.File, shared []string) ([][]string, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	raw, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	var sheet struct {
		Rows []struct {
			Cells []struct {
				R  string `xml:"r,attr"`
				T  string `xml:"t,attr"`
				V  string `xml:"v"`
				Is struct {
					T string `xml:"t"`
				} `xml:"is"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
	}
	if err := xml.Unmarshal(raw, &sheet); err != nil {
		return nil, err
	}
	out := make([][]string, 0, len(sheet.Rows))
	for _, r := range sheet.Rows {
		row := []string{}
		for _, c := range r.Cells {
			col := colIndex(c.R)
			for len(row) <= col {
				row = append(row, "")
			}
			row[col] = cellValue(c.T, c.V, c.Is.T, shared)
		}
		out = append(out, row)
	}
	return out, nil
}

func cellValue(t, v, inline string, shared []string) string {
	switch t {
	case "s":
		idx, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil && idx >= 0 && idx < len(shared) {
			return shared[idx]
		}
		return ""
	case "inlineStr":
		return inline
	default:
		return v
	}
}

// colIndex converte a referencia da celula (ex.: "AB12") no indice da coluna.
func colIndex(ref string) int {
	n := 0
	for _, r := range ref {
		if r >= 'A' && r <= 'Z' {
			n = n*26 + int(r-'A'+1)
		} else {
			break
		}
	}
	if n == 0 {
		return 0
	}
	return n - 1
}
