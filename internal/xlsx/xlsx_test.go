package xlsx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func readZip(t *testing.T, b []byte) map[string]string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(rc)
		rc.Close()
		out[f.Name] = string(data)
	}
	return out
}

func TestWrite(t *testing.T) {
	var buf bytes.Buffer
	err := Write(&buf, []Sheet{
		{Name: "摘要", Rows: [][]any{{"項目", "數量"}, {"a<b & \"c\"", 3}, {" lead", 0}, {"bad\x01char", 1}}},
		{Name: "摘要", Rows: [][]any{{"x"}}}, // duplicate name
		{Name: "a/b:c?[d]*e\f_0123456789_0123456789_0123456789", Rows: nil},
	})
	if err != nil {
		t.Fatal(err)
	}
	parts := readZip(t, buf.Bytes())
	for _, name := range []string{"[Content_Types].xml", "_rels/.rels", "xl/workbook.xml", "xl/_rels/workbook.xml.rels", "xl/styles.xml",
		"xl/worksheets/sheet1.xml", "xl/worksheets/sheet2.xml", "xl/worksheets/sheet3.xml"} {
		body, ok := parts[name]
		if !ok {
			t.Errorf("missing part %s", name)
			continue
		}
		d := xml.NewDecoder(strings.NewReader(body))
		for {
			if _, err := d.Token(); err != nil {
				if err != io.EOF {
					t.Errorf("%s not well-formed: %v", name, err)
				}
				break
			}
		}
	}
	wb := parts["xl/workbook.xml"]
	if !strings.Contains(wb, `name="摘要"`) || !strings.Contains(wb, `name="摘要(2)"`) {
		t.Errorf("sheet names: %s", wb)
	}
	if strings.ContainsAny(wb, "?*[]\\") && strings.Contains(wb, `name="a/b`) {
		t.Errorf("invalid chars kept: %s", wb)
	}
	s1 := parts["xl/worksheets/sheet1.xml"]
	for _, want := range []string{
		`<c r="A1" s="1" t="inlineStr"><is><t>項目</t></is></c>`,
		`<c r="B2"><v>3</v></c>`,
		`a&lt;b &amp; &quot;c&quot;`,
		`<t xml:space="preserve"> lead</t>`,
		`<t>badchar</t>`,
		`<pane ySplit="1"`,
	} {
		if !strings.Contains(s1, want) {
			t.Errorf("sheet1 lacks %s\n%s", want, s1)
		}
	}
	if err := Write(&buf, nil); err == nil {
		t.Error("no sheets should fail")
	}
}

func TestColName(t *testing.T) {
	for i, want := range map[int]string{0: "A", 25: "Z", 26: "AA", 27: "AB", 701: "ZZ", 702: "AAA"} {
		if got := colName(i); got != want {
			t.Errorf("colName(%d) = %s, want %s", i, got, want)
		}
	}
}
