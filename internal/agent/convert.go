package agent

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	htmlTagRe    = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	htmlStripRe  = regexp.MustCompile(`(?s)<[^>]+>`)
	htmlSpaceRe  = regexp.MustCompile(`[ \t]*\n[ \t\n]*\n[ \t]*`)
	multiSpaceRe = regexp.MustCompile(`[ \t]{2,}`)
)

// ToMarkdown converts an uploaded document to markdown text. Text-ish formats
// are handled natively; pdf/docx are best-effort via pandoc/pdftotext if present.
func ToMarkdown(filename, contentType string, data []byte) (markdown string, err error) {
	ext := strings.ToLower(filename)
	switch {
	case hasSuffixAny(ext, ".md", ".markdown", ".txt", ".text", ""):
		return string(data), nil
	case hasSuffixAny(ext, ".html", ".htm") || strings.Contains(contentType, "html"):
		return htmlToMarkdown(string(data)), nil
	case hasSuffixAny(ext, ".csv"):
		return csvToMarkdown(data), nil
	case hasSuffixAny(ext, ".xlsx") || strings.Contains(contentType, "spreadsheetml"):
		return xlsxToMarkdown(data)
	case hasSuffixAny(ext, ".json") || strings.Contains(contentType, "json"):
		return jsonToMarkdown(data), nil
	case hasSuffixAny(ext, ".pdf"):
		return externalConvert(data, "pdf", "pdftotext", []string{"-", "-"})
	case hasSuffixAny(ext, ".docx", ".doc", ".rtf", ".odt", ".epub"):
		return externalConvert(data, "doc", "pandoc", []string{"-t", "markdown"})
	default:
		// Assume utf-8 text (code files, logs, etc.).
		if isProbablyText(data) {
			return string(data), nil
		}
		return "", fmt.Errorf("unsupported document type %q; paste text instead", filename)
	}
}

type xlsxSharedStrings struct {
	Items []struct {
		Texts []string `xml:"t"`
	} `xml:"si"`
}

type xlsxWorkbook struct {
	Sheets []struct {
		Name string `xml:"name,attr"`
		RID  string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
	} `xml:"sheets>sheet"`
}

type xlsxRelationships struct {
	Items []struct {
		ID     string `xml:"Id,attr"`
		Target string `xml:"Target,attr"`
	} `xml:"Relationship"`
}

type xlsxWorksheet struct {
	Rows []xlsxRow `xml:"sheetData>row"`
}

type xlsxRow struct {
	Index int        `xml:"r,attr"`
	Cells []xlsxCell `xml:"c"`
}

type xlsxCell struct {
	Ref       string `xml:"r,attr"`
	Type      string `xml:"t,attr"`
	Value     string `xml:"v"`
	InlineStr struct {
		Texts []string `xml:"t"`
	} `xml:"is"`
}

type xlsxSheetRef struct {
	Name string
	File string
}

func xlsxToMarkdown(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("read xlsx: %w", err)
	}
	files := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		files[f.Name] = f
	}

	shared, err := readXLSXSharedStrings(files["xl/sharedStrings.xml"])
	if err != nil {
		return "", err
	}
	sheets, err := readXLSXSheetRefs(files)
	if err != nil {
		return "", err
	}
	if len(sheets) == 0 {
		return "", fmt.Errorf("xlsx contains no worksheets")
	}

	var b strings.Builder
	for _, sheet := range sheets {
		f := files[sheet.File]
		if f == nil {
			continue
		}
		md, err := readXLSXWorksheet(f, sheet.Name, shared)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(md) == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(md)
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "", fmt.Errorf("xlsx contains no readable cells")
	}
	return out, nil
}

func readXLSXSharedStrings(f *zip.File) ([]string, error) {
	if f == nil {
		return nil, nil
	}
	var parsed xlsxSharedStrings
	if err := readZipXML(f, &parsed); err != nil {
		return nil, fmt.Errorf("read shared strings: %w", err)
	}
	out := make([]string, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		out = append(out, strings.Join(item.Texts, ""))
	}
	return out, nil
}

func readXLSXSheetRefs(files map[string]*zip.File) ([]xlsxSheetRef, error) {
	wbFile := files["xl/workbook.xml"]
	if wbFile == nil {
		return fallbackXLSXSheetRefs(files), nil
	}
	var wb xlsxWorkbook
	if err := readZipXML(wbFile, &wb); err != nil {
		return nil, fmt.Errorf("read workbook: %w", err)
	}

	rels := map[string]string{}
	if relFile := files["xl/_rels/workbook.xml.rels"]; relFile != nil {
		var parsed xlsxRelationships
		if err := readZipXML(relFile, &parsed); err != nil {
			return nil, fmt.Errorf("read workbook relationships: %w", err)
		}
		for _, rel := range parsed.Items {
			target := strings.TrimPrefix(path.Clean("/xl/"+rel.Target), "/")
			rels[rel.ID] = target
		}
	}

	refs := make([]xlsxSheetRef, 0, len(wb.Sheets))
	for i, s := range wb.Sheets {
		file := rels[s.RID]
		if file == "" {
			file = fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1)
		}
		name := strings.TrimSpace(s.Name)
		if name == "" {
			name = fmt.Sprintf("Sheet%d", i+1)
		}
		refs = append(refs, xlsxSheetRef{Name: name, File: file})
	}
	return refs, nil
}

func fallbackXLSXSheetRefs(files map[string]*zip.File) []xlsxSheetRef {
	var names []string
	for name := range files {
		if strings.HasPrefix(name, "xl/worksheets/") && strings.HasSuffix(name, ".xml") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	refs := make([]xlsxSheetRef, 0, len(names))
	for i, name := range names {
		refs = append(refs, xlsxSheetRef{Name: fmt.Sprintf("Sheet%d", i+1), File: name})
	}
	return refs
}

func readXLSXWorksheet(f *zip.File, sheetName string, shared []string) (string, error) {
	var ws xlsxWorksheet
	if err := readZipXML(f, &ws); err != nil {
		return "", fmt.Errorf("read worksheet %s: %w", sheetName, err)
	}
	rows := make([][]string, 0, len(ws.Rows))
	maxCols := 0
	for _, row := range ws.Rows {
		values := map[int]string{}
		for _, cell := range row.Cells {
			col := xlsxColumnIndex(cell.Ref)
			if col < 0 {
				col = len(values)
			}
			values[col] = xlsxCellText(cell, shared)
			if col+1 > maxCols {
				maxCols = col + 1
			}
		}
		if len(values) == 0 {
			continue
		}
		out := make([]string, maxCols)
		for col, value := range values {
			if col >= 0 && col < len(out) {
				out[col] = value
			}
		}
		rows = append(rows, out)
	}
	if len(rows) == 0 {
		return "", nil
	}
	for i := range rows {
		if len(rows[i]) < maxCols {
			rows[i] = append(rows[i], make([]string, maxCols-len(rows[i]))...)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n\n", markdownTableCell(sheetName))
	for i, row := range rows {
		writeMarkdownRow(&b, row)
		if i == 0 {
			b.WriteString("|")
			for range row {
				b.WriteString(" --- |")
			}
			b.WriteByte('\n')
		}
		if i >= 5000 {
			break
		}
	}
	return strings.TrimSpace(b.String()), nil
}

func xlsxCellText(cell xlsxCell, shared []string) string {
	switch cell.Type {
	case "s":
		idx, err := strconv.Atoi(strings.TrimSpace(cell.Value))
		if err == nil && idx >= 0 && idx < len(shared) {
			return shared[idx]
		}
	case "inlineStr":
		return strings.Join(cell.InlineStr.Texts, "")
	case "b":
		if strings.TrimSpace(cell.Value) == "1" {
			return "TRUE"
		}
		if strings.TrimSpace(cell.Value) == "0" {
			return "FALSE"
		}
	}
	return strings.TrimSpace(cell.Value)
}

func xlsxColumnIndex(ref string) int {
	col := 0
	seen := false
	for _, r := range ref {
		if r >= 'A' && r <= 'Z' {
			col = col*26 + int(r-'A'+1)
			seen = true
			continue
		}
		if r >= 'a' && r <= 'z' {
			col = col*26 + int(r-'a'+1)
			seen = true
			continue
		}
		break
	}
	if !seen {
		return -1
	}
	return col - 1
}

func writeMarkdownRow(b *strings.Builder, row []string) {
	b.WriteString("|")
	for _, cell := range row {
		b.WriteByte(' ')
		b.WriteString(markdownTableCell(cell))
		b.WriteString(" |")
	}
	b.WriteByte('\n')
}

func markdownTableCell(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	return strings.TrimSpace(s)
}

func readZipXML(f *zip.File, v any) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	raw, err := io.ReadAll(rc)
	if err != nil {
		return err
	}
	return xml.Unmarshal(raw, v)
}

func htmlToMarkdown(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?i)<br\s*/?>`).ReplaceAllString(s, "\n")
	s = regexp.MustCompile(`(?i)</(p|div|h[1-6]|li|tr)>`).ReplaceAllString(s, "\n")
	s = htmlStripRe.ReplaceAllString(s, "")
	s = htmlEntities(s)
	s = htmlSpaceRe.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

func htmlEntities(s string) string {
	r := strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", "\"", "&#39;", "'", "&nbsp;", " ")
	return r.Replace(s)
}

func csvToMarkdown(data []byte) string {
	rec, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil || len(rec) == 0 {
		return string(data)
	}
	var b strings.Builder
	for i, row := range rec {
		b.WriteString("| " + strings.Join(row, " | ") + " |\n")
		if i == 0 {
			b.WriteString("|" + strings.Repeat(" --- |", len(row)) + "\n")
		}
		if i > 5000 {
			break
		}
	}
	return b.String()
}

func jsonToMarkdown(data []byte) string {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return string(data)
	}
	pretty, _ := json.MarshalIndent(v, "", "  ")
	return "```json\n" + string(pretty) + "\n```"
}

func externalConvert(data []byte, kind, bin string, args []string) (string, error) {
	path, err := exec.LookPath(bin)
	if err != nil {
		return "", fmt.Errorf("%s document support requires %q on PATH; paste text instead", kind, bin)
	}
	cmd := exec.Command(path, args...)
	cmd.Stdin = strings.NewReader(string(data))
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("convert %s: %w", kind, err)
	}
	return string(out), nil
}

func hasSuffixAny(s string, suffixes ...string) bool {
	for _, suf := range suffixes {
		if suf == "" {
			if !strings.Contains(s, ".") {
				return true
			}
			continue
		}
		if strings.HasSuffix(s, suf) {
			return true
		}
	}
	return false
}

func isProbablyText(data []byte) bool {
	n := len(data)
	if n > 2048 {
		n = 2048
	}
	for _, b := range data[:n] {
		if b == 0 {
			return false
		}
	}
	return true
}

// Chunk splits markdown into overlapping chunks suitable for embedding.
func Chunk(text string, size, overlap int) []string {
	text = multiSpaceRe.ReplaceAllString(strings.TrimSpace(text), " ")
	if size <= 0 {
		size = 1400
	}
	if overlap < 0 || overlap >= size {
		overlap = 160
	}
	runes := []rune(text)
	if len(runes) <= size {
		if len(runes) == 0 {
			return nil
		}
		return []string{string(runes)}
	}
	var out []string
	for start := 0; start < len(runes); start += size - overlap {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, strings.TrimSpace(string(runes[start:end])))
		if end == len(runes) {
			break
		}
	}
	return out
}
