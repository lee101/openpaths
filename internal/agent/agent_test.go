package agent

import (
	"archive/zip"
	"bytes"
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
