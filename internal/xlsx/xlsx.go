// Package xlsx writes a minimal .xlsx workbook (text and number cells, a bold
// header row, frozen header, rough column widths) with the standard library
// only. An .xlsx file is a zip of a few XML parts.
package xlsx

import (
	"archive/zip"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Sheet is one worksheet. Cells are string or int; anything else is written
// with fmt's %v as text. The first row is the header (bold, frozen).
type Sheet struct {
	Name string
	Rows [][]any
}

// maxWidth caps a column width in characters.
const maxWidth = 60

// Write writes the workbook to w. Sheet names are made valid for Excel
// (at most 31 characters, none of []:*?/\ ), and made unique.
func Write(w io.Writer, sheets []Sheet) error {
	if len(sheets) == 0 {
		return fmt.Errorf("xlsx: no sheets")
	}
	names := make([]string, len(sheets))
	used := map[string]bool{}
	for i, s := range sheets {
		n := sheetName(s.Name, i+1)
		for base, k := n, 2; used[strings.ToLower(n)]; k++ {
			suffix := "(" + strconv.Itoa(k) + ")"
			n = truncRunes(base, 31-len(suffix)) + suffix
		}
		used[strings.ToLower(n)] = true
		names[i] = n
	}

	zw := zip.NewWriter(w)
	put := func(name, body string) error {
		f, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = io.WriteString(f, body)
		return err
	}

	const header = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n"
	var ct, wb, rels strings.Builder
	ct.WriteString(header + `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
		`<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>`)
	wb.WriteString(header + `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" ` +
		`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`)
	rels.WriteString(header + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for i, s := range sheets {
		id := i + 1
		fmt.Fprintf(&ct, `<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, id)
		fmt.Fprintf(&wb, `<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, esc(names[i]), id, id)
		fmt.Fprintf(&rels, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, id, id)
		if err := put(fmt.Sprintf("xl/worksheets/sheet%d.xml", id), header+sheetXML(s)); err != nil {
			return err
		}
	}
	ct.WriteString(`</Types>`)
	wb.WriteString(`</sheets></workbook>`)
	fmt.Fprintf(&rels, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`, len(sheets)+1)

	const styles = header + `<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
		`<fonts count="2"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><name val="Calibri"/></font></fonts>` +
		`<fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills>` +
		`<borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>` +
		`<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>` +
		`<cellXfs count="2"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>` +
		`<xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0" applyFont="1"/></cellXfs>` +
		`<cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles></styleSheet>`
	const rootRels = header + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`
	for _, p := range [][2]string{
		{"[Content_Types].xml", ct.String()},
		{"_rels/.rels", rootRels},
		{"xl/workbook.xml", wb.String()},
		{"xl/_rels/workbook.xml.rels", rels.String()},
		{"xl/styles.xml", styles},
	} {
		if err := put(p[0], p[1]); err != nil {
			return err
		}
	}
	return zw.Close()
}

func sheetXML(s Sheet) string {
	var b strings.Builder
	b.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	b.WriteString(`<sheetViews><sheetView workbookViewId="0">`)
	if len(s.Rows) > 1 {
		b.WriteString(`<pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/>`)
	}
	b.WriteString(`</sheetView></sheetViews>`)

	widths := map[int]int{}
	for _, row := range s.Rows {
		for c, v := range row {
			if _, isNum := v.(int); isNum {
				continue
			}
			if w := textWidth(cellText(v)); w > widths[c] {
				widths[c] = w
			}
		}
	}
	if len(widths) > 0 {
		b.WriteString(`<cols>`)
		for c := 0; c <= maxKey(widths); c++ {
			w := widths[c] + 2
			if w < 8 {
				w = 8
			}
			if w > maxWidth {
				w = maxWidth
			}
			fmt.Fprintf(&b, `<col min="%d" max="%d" width="%d" customWidth="1"/>`, c+1, c+1, w)
		}
		b.WriteString(`</cols>`)
	}

	b.WriteString(`<sheetData>`)
	for r, row := range s.Rows {
		fmt.Fprintf(&b, `<row r="%d">`, r+1)
		for c, v := range row {
			ref := colName(c) + strconv.Itoa(r+1)
			style := ""
			if r == 0 {
				style = ` s="1"`
			}
			if n, ok := v.(int); ok {
				fmt.Fprintf(&b, `<c r="%s"%s><v>%d</v></c>`, ref, style, n)
				continue
			}
			t := cellText(v)
			sp := ""
			if t != strings.TrimSpace(t) {
				sp = ` xml:space="preserve"`
			}
			fmt.Fprintf(&b, `<c r="%s"%s t="inlineStr"><is><t%s>%s</t></is></c>`, ref, style, sp, esc(t))
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData></worksheet>`)
	return b.String()
}

func cellText(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func maxKey(m map[int]int) int {
	n := 0
	for k := range m {
		if k > n {
			n = k
		}
	}
	return n
}

// textWidth is the display width in characters; CJK counts double.
func textWidth(s string) int {
	w := 0
	for _, r := range s {
		if r >= 0x2E80 {
			w += 2
		} else {
			w++
		}
	}
	return w
}

// colName turns 0 -> A, 25 -> Z, 26 -> AA.
func colName(i int) string {
	s := ""
	for i++; i > 0; i = (i - 1) / 26 {
		s = string(rune('A'+(i-1)%26)) + s
	}
	return s
}

// esc escapes XML text and drops characters XML 1.0 does not allow.
func esc(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '&':
			b.WriteString("&amp;")
		case r == '<':
			b.WriteString("&lt;")
		case r == '>':
			b.WriteString("&gt;")
		case r == '"':
			b.WriteString("&quot;")
		case r == '\t' || r == '\n' || r == '\r' || (r >= 0x20 && r <= 0xD7FF) || (r >= 0xE000 && r <= 0xFFFD && r != 0xFFFE) || r >= 0x10000:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func truncRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

func sheetName(name string, idx int) string {
	n := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`[]:*?/\`, r) || r < 0x20 {
			return -1
		}
		return r
	}, name)
	n = truncRunes(strings.Trim(n, "'"), 31)
	if n == "" {
		n = "Sheet" + strconv.Itoa(idx)
	}
	return n
}
