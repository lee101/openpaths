package agent

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	markitdown "github.com/conductor-oss/markitdown"
	"github.com/deepteams/webp"
)

var (
	htmlTagRe     = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	htmlStripRe   = regexp.MustCompile(`(?s)<[^>]+>`)
	htmlSpaceRe   = regexp.MustCompile(`[ \t]*\n[ \t\n]*\n[ \t]*`)
	multiSpaceRe  = regexp.MustCompile(`[ \t]{2,}`)
	inlineImageRe = regexp.MustCompile(`(?is)<img\s+[^>]*src=["']data:[^>]+>`)
	imageSourceRe = regexp.MustCompile(`(?is)src=["']data:([^;"']+);base64,([^"']+)["']`)
	imageAltRe    = regexp.MustCompile(`(?is)alt=["']([^"']*)["']`)
)

const (
	maxDocumentImages = 3
	minDocumentWidth  = 320
	minDocumentHeight = 180
	maxDocumentPixels = 12_000_000
	maxImageRatio     = 4.0
)

// ConvertedDocument is the normalized, RAG-ready representation of an upload.
// Image data is kept separately so it never leaks into embedding chunks.
type ConvertedDocument struct {
	Markdown string
	Images   []DocumentImage
	Parser   string
	Seen     int
}

// DocumentImage is a useful embedded image re-encoded for storage.
type DocumentImage struct {
	Placeholder string
	Alt         string
	WebP        []byte
	Width       int
	Height      int
}

// ToMarkdown is the text-only compatibility wrapper around ConvertDocument.
func ToMarkdown(filename, contentType string, data []byte) (markdown string, err error) {
	converted, err := ConvertDocument(filename, contentType, data)
	if err != nil {
		return "", err
	}
	// The compatibility helper has no asset store. Remove placeholders instead of
	// returning large data URIs or broken local references.
	markdown = converted.Markdown
	for _, img := range converted.Images {
		markdown = strings.ReplaceAll(markdown, img.Placeholder, "")
	}
	return strings.TrimSpace(markdown), nil
}

// ConvertDocument converts an upload to Markdown and extracts a small number of
// useful embedded images. Office/PDF/EPUB/PowerPoint inputs use the pure-Go
// MarkItDown port; simple formats retain the small native converters below.
func ConvertDocument(filename, contentType string, data []byte) (ConvertedDocument, error) {
	ext := strings.ToLower(filename)
	var markdown, parser string
	var err error
	switch {
	case hasSuffixAny(ext, ".md", ".markdown", ".txt", ".text", ""):
		markdown, parser = string(data), "native"
	case hasSuffixAny(ext, ".json") || strings.Contains(contentType, "json"):
		markdown, parser = jsonToMarkdown(data), "native"
	case hasSuffixAny(ext, ".csv"):
		markdown, parser = csvToMarkdown(data), "native"
	case hasSuffixAny(ext, ".xlsx") || strings.Contains(contentType, "spreadsheetml"):
		markdown, err = xlsxToMarkdown(data)
		parser = "native"
	case hasSuffixAny(ext, ".pdf", ".docx", ".pptx", ".xls", ".html", ".htm", ".epub", ".ipynb", ".zip") ||
		strings.Contains(contentType, "pdf") || strings.Contains(contentType, "wordprocessingml") ||
		strings.Contains(contentType, "presentationml") || strings.Contains(contentType, "html"):
		markdown, err = markItDown(filename, contentType, data)
		parser = "markitdown"
	case hasSuffixAny(ext, ".doc", ".rtf", ".odt"):
		markdown, err = externalConvert(data, "doc", "pandoc", []string{"-t", "gfm"})
		parser = "pandoc"
	default:
		// Assume utf-8 text (code files, logs, etc.).
		if isProbablyText(data) {
			markdown, parser = string(data), "native"
			break
		}
		return ConvertedDocument{}, fmt.Errorf("unsupported document type %q; paste text instead", filename)
	}
	if err != nil {
		return ConvertedDocument{}, err
	}
	if strings.TrimSpace(markdown) == "" {
		return ConvertedDocument{}, fmt.Errorf("%q did not contain readable text", filename)
	}
	markdown, images, seen := extractUsefulImages(markdown)
	return ConvertedDocument{Markdown: strings.TrimSpace(markdown), Images: images, Parser: parser, Seen: seen}, nil
}

func markItDown(filename, contentType string, data []byte) (string, error) {
	m := markitdown.New(markitdown.WithKeepDataURIs(true))
	result, err := m.ConvertReader(bytes.NewReader(data), markitdown.StreamInfo{
		Extension: strings.ToLower(filepath.Ext(filename)),
		MIMEType:  strings.TrimSpace(strings.Split(contentType, ";")[0]),
		Filename:  filename,
	})
	if err != nil {
		return "", fmt.Errorf("convert %s: %w", filename, err)
	}
	return result.Markdown, nil
}

func extractUsefulImages(markdown string) (string, []DocumentImage, int) {
	seen := 0
	images := make([]DocumentImage, 0, maxDocumentImages)
	markdown = inlineImageRe.ReplaceAllStringFunc(markdown, func(tag string) string {
		seen++
		if len(images) >= maxDocumentImages {
			return ""
		}
		src := imageSourceRe.FindStringSubmatch(tag)
		if len(src) != 3 || strings.EqualFold(src[1], "image/svg+xml") {
			return ""
		}
		raw, err := base64.StdEncoding.DecodeString(strings.Map(func(r rune) rune {
			if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
				return -1
			}
			return r
		}, src[2]))
		if err != nil || len(raw) == 0 {
			return ""
		}
		config, _, err := image.DecodeConfig(bytes.NewReader(raw))
		if err != nil {
			return ""
		}
		width, height := config.Width, config.Height
		ratio := float64(width) / float64(height)
		if width < minDocumentWidth || height < minDocumentHeight || width*height > maxDocumentPixels || ratio > maxImageRatio || ratio < 1/maxImageRatio {
			return ""
		}
		decoded, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			return ""
		}
		var encoded bytes.Buffer
		if err := webp.Encode(&encoded, decoded, &webp.EncoderOptions{Quality: 85, Method: 4}); err != nil {
			return ""
		}
		alt := "Document image"
		if match := imageAltRe.FindStringSubmatch(tag); len(match) == 2 && strings.TrimSpace(match[1]) != "" {
			alt = html.UnescapeString(strings.TrimSpace(match[1]))
		}
		placeholder := fmt.Sprintf("{{OPENPATHS_DOCUMENT_IMAGE_%d}}", len(images)+1)
		images = append(images, DocumentImage{
			Placeholder: placeholder,
			Alt:         alt,
			WebP:        encoded.Bytes(),
			Width:       width,
			Height:      height,
		})
		return placeholder
	})
	return markdown, images, seen
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
