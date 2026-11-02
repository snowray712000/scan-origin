# Per-FDI tooth statistics

- Date: 2026-11-02
- Project: scan-origin
- Type: feature
- Related logs: 1030b_dataset_sorting_implementation.md
- Cross-project: none
- Commit: b7428f6 (released as v1.2.3; official as 1.3.1)

## Goal / why
The page only showed IOS / Die case counts. The user wanted to know which teeth the dataset contains: count per FDI number and a grand total.

## Starting state
`Scan` only listed directories (never descended into a case). File names were read only by `BuildPlan`/`Classify` for pending cases.

## What changed (behavior level)
- New section "各牙位（FDI）數量" on the page: table FDI / IOS / Die / 小計, sorted by FDI, with a footer row "總量". A note line shows the number of cases looked at; a second line lists (up to 50) classified cases with no tooth file.
- Only classified cases (`IOS/…`, `Die/…`) are counted; pending batches are not.
- Where the FDI comes from depends on the case's folder (kind), not on what other files exist:
  - IOS case: `…-{FDI}-crown_cad.stl` (the `…_transform.stl` companion does not match, so no double count).
  - Die case: `…-Tooth_{FDI}.stl`. Die cases also hold `…-36-crown_cad.stl`; that is ignored for Die.
- FDI = two digits, each 1–8, case-insensitive. A case with several teeth (bridge) counts once per distinct FDI, so the tooth total can exceed the case count.

## Design and decisions
- Separate endpoint `GET /api/teeth` instead of extending `/api/state`: it needs one directory listing per case (hundreds–thousands), while state stays cheap. The page renders state first, then requests teeth ("統計中…" meanwhile). `refresh()` triggers both.
- Sorted cases are already flattened, so only the case folder itself is listed (no recursion, no flatten preview).
- Kept the two regexes together in `classify.go`; `dieRe` gained a capture group for the FDI.

## Key files / types / functions
- `internal/dataset/classify.go`: `iosToothRe`, `FDIs(kind, fileNames) []string` (distinct, sorted).
- `internal/dataset/teeth.go`: `ToothStat{fdi, ios, die}`, `TeethStats{teeth, iosTeeth, dieTeeth, iosCases, dieCases, noFdi, noFdiList}`, `ScanTeeth(root)`.
- `internal/server/server.go`: `GET /api/teeth` (same token/guard as the other API).
- `web/index.html`, `web/app.js` (`renderTeeth`, `refreshTeeth`), `web/style.css` (`tfoot .total`). All text via `textContent`.

## Contracts and algorithms
`GET /api/teeth` (header `X-Token`) ->
```json
{"teeth":[{"fdi":"35","ios":2,"die":0},{"fdi":"36","ios":1,"die":1}],
 "iosTeeth":3,"dieTeeth":1,"iosCases":3,"dieCases":1,
 "noFdi":1,"noFdiList":["IOS/261022_2/c3"]}
```
`teeth` and `noFdiList` are never null. Totals = sums over `teeth`.

## Gotchas
- The `tree.txt` the user supplied (a sample of a real dataset) shows IOS and Die cases both carry `…-36-crown_cad.stl`; using crown_cad for Die would be redundant at best, so the source is chosen by kind.
- Counting by grep over the tree doubled the numbers because of `_transform.stl`; the regex anchors on `-crown_cad.stl$` to avoid that.

## Verification
`gofmt`, `go vet ./... && go test ./...` pass. New tests: `TestFDIs`, `TestScanTeeth` (bridge, transform ignored, Die ignores crown_cad, pending ignored, no-FDI case, empty root), `TestTeethNeedsToken`.

## Open issues / next steps
Not checked in a real browser against a large dataset. If listing thousands of cases over a network drive is slow, cache the result or add progress.
