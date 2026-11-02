# Export statistics to an .xlsx download

- Date: 2026-11-02
- Project: scan-origin
- Type: feature
- Related logs: 1102c_fdi_tooth_statistics.md
- Cross-project: none
- Commit: bfd31b4 (released as v1.2.4; official as 1.3.1)

## Goal / why
After 1.2.3 (FDI statistics, tested OK by the user) they wanted the numbers in Excel. Constraint: no third-party packages, so the workbook is written by hand with `archive/zip`. Released as 1.2.4.

## Starting state
`ScanTeeth(root)` did the per-case file listing and aggregation in one function; the page only showed tables.

## What changed (behavior level)
- Header button "匯出 Excel" -> browser downloads `scan-origin-YYYYMMDD-HHMMSS.xlsx` (server time).
- Four sheets, first row bold and frozen, column widths estimated from the text (CJK counts double, 8..60):
  1. `摘要`: columns 項目/IOS/Die/合計; rows 已分類 case 數, 牙位數（每個 case 的每個牙位算一次）, 找不到牙位檔的 case 數, 待處理 case 數 (only the 合計 cell filled).
  2. `各上傳日期`: 上傳日期/IOS/Die/小計 (newest first, same as the page) + a 總量 row.
  3. `各牙位`: FDI/IOS/Die/小計 sorted by FDI + a 總量 row.
  4. `Case 清單`: 類別/批次/Case/牙位/牙位數, one row per classified case (IOS first, then Die, by batch then case name); 牙位 is the FDIs joined by ", ".
- Only classified cases are included; pending batches appear only as the pending count.

## Design and decisions
- Custom minimal writer `internal/xlsx` rather than CSV: the user wants several tables in one file and Chinese must just work. Cells are `int` (numeric) or text via `t="inlineStr"` (no sharedStrings), so nothing is ever evaluated as a formula.
- Download must carry `X-Token`; a plain `<a href>` cannot, so `app.js` fetches a blob and clicks a temporary link (`URL.revokeObjectURL` after 10 s). Errors come back as JSON `{error}` and go to the red banner.
- The server builds the workbook in a `bytes.Buffer` first so a scan error is a JSON 500 instead of a truncated file.
- Refactor: `ScanCases(root) []CaseRow{Kind,Batch,Case,FDIs}` does the listing; `SummarizeTeeth(rows)` aggregates; `ScanTeeth` = both. `/api/teeth` output is unchanged. The export reuses the same rows for sheets 3 and 4, so the numbers always agree.

## Key files / types / functions
- `internal/xlsx/xlsx.go`: `Sheet{Name, Rows [][]any}`, `Write(w, sheets)`. Parts written: `[Content_Types].xml`, `_rels/.rels`, `xl/workbook.xml`, `xl/_rels/workbook.xml.rels`, `xl/styles.xml` (2 fonts, 2 cellXfs: normal / bold), `xl/worksheets/sheetN.xml`. Sheet names: strip `[]:*?/\`, max 31 runes, de-duplicated case-insensitively with `(2)`, empty -> `SheetN`. `esc` escapes XML and drops characters illegal in XML 1.0 (control chars). Leading/trailing spaces get `xml:space="preserve"`.
- `internal/dataset/export.go`: `ExportXLSX(root, w)`; `internal/dataset/teeth.go`: `ScanCases`, `SummarizeTeeth`.
- `internal/server/server.go`: `GET /api/export.xlsx` (token + origin guard like other APIs; `Content-Disposition: attachment`).
- `web/index.html` (`#exportBtn`), `web/app.js` (click handler).

## Contracts and algorithms
`GET /api/export.xlsx` with header `X-Token` -> 200, `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`, `Content-Disposition: attachment; filename="scan-origin-20261102-153000.xlsx"`, `Cache-Control: no-store`; failure -> JSON `{"error":…}`.

## Gotchas
- Excel is strict about the part set: styles.xml must exist and be related from the workbook, `cellXfs` index 1 is the bold style used by the header row, and the `<pane>` for freezing needs `topLeftCell` and `state="frozen"`.
- No xlsx reader library is available (no openpyxl). Verification used the installed Excel through COM (PowerShell `New-Object -ComObject Excel.Application`, `DisplayAlerts=$false`, open, read `UsedRange` text): it opened the file with no repair prompt and all four sheets read back correctly.

## Verification
`gofmt`, `go vet ./... && go test ./...` pass. New tests: `internal/xlsx` (parts present and well-formed, escaping, duplicate/invalid sheet names, `colName`), `TestExportXLSX` in dataset (pending excluded, FDIs present, empty root OK) and in server (token required, `PK` zip body, attachment header). Manual: generated a workbook from a temp dataset and opened it in Excel via COM.

## Open issues / next steps
Browser download click not exercised in a real browser by the AI. Very large datasets: the whole workbook is built in memory (fine for tens of thousands of rows).
