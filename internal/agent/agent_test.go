package agent

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestIsReadOnlySQL(t *testing.T) {
	ok := []string{"SELECT 1", "select * from users where id=1", "WITH t AS (SELECT 1) SELECT * FROM t"}
	bad := []string{"DELETE FROM users", "select 1; drop table users", "update users set x=1", "INSERT INTO t VALUES (1)", ""}
	for _, q := range ok {
		if !isReadOnlySQL(q) {
			t.Errorf("expected read-only: %q", q)
		}
	}
	for _, q := range bad {
		if isReadOnlySQL(q) {
			t.Errorf("expected rejected: %q", q)
		}
	}
}

func TestToMarkdown(t *testing.T) {
	md, err := ToMarkdown("a.csv", "text/csv", []byte("a,b\n1,2\n"))
	if err != nil || md == "" {
		t.Fatalf("csv: %v %q", err, md)
	}
	md, err = ToMarkdown("a.html", "text/html", []byte("<h1>Hi</h1><p>body</p>"))
	if err != nil || md == "" {
		t.Fatalf("html: %v %q", err, md)
	}
	if _, err := ToMarkdown("a.md", "", []byte("# Title")); err != nil {
		t.Fatalf("md: %v", err)
	}
}

func TestToMarkdownXLSX(t *testing.T) {
	md, err := ToMarkdown("book.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", testXLSX(t))
	if err != nil {
		t.Fatalf("xlsx: %v", err)
	}
	for _, want := range []string{"## Leads", "| Name | Score | Active |", "| Ada | 42 | TRUE |"} {
		if !strings.Contains(md, want) {
			t.Fatalf("xlsx markdown missing %q:\n%s", want, md)
		}
	}
}

func TestExtractUsefulImagesFiltersAndCompresses(t *testing.T) {
	large := testPNG(t, 640, 360)
	small := testPNG(t, 120, 120)
	markdown := fmt.Sprintf(`<p>Before</p><img src="data:image/png;base64,%s" alt="Useful chart"/><img src="data:image/png;base64,%s" alt="Tiny icon"/><p>After</p>`,
		base64.StdEncoding.EncodeToString(large), base64.StdEncoding.EncodeToString(small))

	got, images, seen := extractUsefulImages(markdown)
	if seen != 2 {
		t.Fatalf("seen = %d, want 2", seen)
	}
	if len(images) != 1 {
		t.Fatalf("images = %d, want 1", len(images))
	}
	if !strings.Contains(got, images[0].Placeholder) || strings.Contains(got, "Tiny icon") || strings.Contains(got, "base64") {
		t.Fatalf("unexpected normalized markdown: %s", got)
	}
	if images[0].Alt != "Useful chart" || images[0].Width != 640 || images[0].Height != 360 {
		t.Fatalf("image metadata = %#v", images[0])
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(images[0].WebP))
	if err != nil || format != "webp" || config.Width != 640 || config.Height != 360 {
		t.Fatalf("webp = %s %dx%d err=%v", format, config.Width, config.Height, err)
	}
}

func TestChunk(t *testing.T) {
	if got := Chunk("", 100, 10); got != nil {
		t.Errorf("empty -> %v", got)
	}
	long := make([]byte, 5000)
	for i := range long {
		long[i] = 'a'
	}
	c := Chunk(string(long), 1400, 160)
	if len(c) < 3 {
		t.Errorf("expected multiple chunks, got %d", len(c))
	}
}

func testXLSX(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	addZipFile(t, zw, "[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
  <Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>
</Types>`)
	addZipFile(t, zw, "xl/workbook.xml", `<?xml version="1.0" encoding="UTF-8"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets><sheet name="Leads" sheetId="1" r:id="rId1"/></sheets>
</workbook>`)
	addZipFile(t, zw, "xl/_rels/workbook.xml.rels", `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
</Relationships>`)
	addZipFile(t, zw, "xl/sharedStrings.xml", `<?xml version="1.0" encoding="UTF-8"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <si><t>Name</t></si><si><t>Score</t></si><si><t>Active</t></si><si><t>Ada</t></si>
</sst>`)
	addZipFile(t, zw, "xl/worksheets/sheet1.xml", `<?xml version="1.0" encoding="UTF-8"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c><c r="C1" t="s"><v>2</v></c></row>
    <row r="2"><c r="A2" t="s"><v>3</v></c><c r="B2"><v>42</v></c><c r="C2" t="b"><v>1</v></c></row>
  </sheetData>
</worksheet>`)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func addZipFile(t *testing.T, zw *zip.Writer, name, body string) {
	t.Helper()
	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
}

func testPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x % 255), G: uint8(y % 255), B: 140, A: 255})
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
