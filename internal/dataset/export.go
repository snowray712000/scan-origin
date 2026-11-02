package dataset

import (
	"io"
	"strings"

	"scanorigin/internal/xlsx"
)

// ExportXLSX writes the statistics as a workbook: a summary, the per-day
// counts, the per-FDI counts and one row per classified case. It is read-only.
func ExportXLSX(root string, w io.Writer) error {
	st, err := Scan(root)
	if err != nil {
		return err
	}
	rows, err := ScanCases(root)
	if err != nil {
		return err
	}
	teeth := SummarizeTeeth(rows)

	noIOS, noDie := 0, 0
	for _, r := range rows {
		if len(r.FDIs) == 0 {
			if r.Kind == KindIOS {
				noIOS++
			} else {
				noDie++
			}
		}
	}

	summary := xlsx.Sheet{Name: "摘要", Rows: [][]any{
		{"項目", "IOS", "Die", "合計"},
		{"已分類 case 數", st.IOS, st.Die, st.Total},
		{"牙位數（每個 case 的每個牙位算一次）", teeth.IOSTeeth, teeth.DieTeeth, teeth.IOSTeeth + teeth.DieTeeth},
		{"找不到牙位檔的 case 數", noIOS, noDie, noIOS + noDie},
		{"待處理 case 數", "", "", st.Pending},
	}}

	days := xlsx.Sheet{Name: "各上傳日期", Rows: [][]any{{"上傳日期", "IOS", "Die", "小計"}}}
	for _, d := range st.Days {
		days.Rows = append(days.Rows, []any{d.Date, d.IOS, d.Die, d.IOS + d.Die})
	}
	days.Rows = append(days.Rows, []any{"總量", st.IOS, st.Die, st.Total})

	fdi := xlsx.Sheet{Name: "各牙位", Rows: [][]any{{"FDI", "IOS", "Die", "小計"}}}
	for _, t := range teeth.Teeth {
		fdi.Rows = append(fdi.Rows, []any{t.FDI, t.IOS, t.Die, t.IOS + t.Die})
	}
	fdi.Rows = append(fdi.Rows, []any{"總量", teeth.IOSTeeth, teeth.DieTeeth, teeth.IOSTeeth + teeth.DieTeeth})

	cases := xlsx.Sheet{Name: "Case 清單", Rows: [][]any{{"類別", "批次", "Case", "牙位", "牙位數"}}}
	for _, r := range rows {
		cases.Rows = append(cases.Rows, []any{string(r.Kind), r.Batch, r.Case, strings.Join(r.FDIs, ", "), len(r.FDIs)})
	}

	return xlsx.Write(w, []xlsx.Sheet{summary, days, fdi, cases})
}
