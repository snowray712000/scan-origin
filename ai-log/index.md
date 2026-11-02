# scan-origin — ai-log index

scan-origin is a Go tool (stdlib only, embedded vanilla web UI, token-protected HTTP server reachable over the LAN) that sorts a dental-scan dataset folder into IOS (intra-oral scan) and Die (desktop scan of cut plaster dies) cases from file names, and reports counts per upload date.

Rules for writing and reading logs: [README.md](README.md). Read this index first, then open only the logs relevant to your task.

## 2026 · s4 (Oct–Dec)

| Log | Type | Summary | Areas / keywords | Files touched |
|---|---|---|---|---|
| [1102e_release_1_3_1](2026/s4/1102e_release_1_3_1.md) | release | Version 1.2.4 -> 1.3.1 (FDI statistics + Excel export); tag and npm publish procedure, no `--tag test`. | release, tag, npm publish, 1.3.1 | `internal/version/version.go` |
| [1102d_export_excel](2026/s4/1102d_export_excel.md) | feature | "匯出 Excel" button: `GET /api/export.xlsx` downloads a 4-sheet workbook (summary, per day, per FDI, case list) written by a stdlib-only xlsx writer; released as 1.2.4. | xlsx, excel, export, `archive/zip`, inlineStr, `ScanCases`, token fetch blob | `internal/xlsx/*`, `internal/dataset/export.go`, `internal/dataset/teeth.go`, `internal/server/server.go`, `web/*` |
| [1102c_fdi_tooth_statistics](2026/s4/1102c_fdi_tooth_statistics.md) | feature | Per-FDI tooth counts (IOS from `-XX-crown_cad.stl`, Die from `-Tooth_XX.stl`) with totals, via new `GET /api/teeth` and a page section. | FDI, tooth, crown_cad, Tooth_XX, `/api/teeth`, ScanTeeth | `internal/dataset/classify.go`, `internal/dataset/teeth.go`, `internal/server/server.go`, `web/*` |
| [1102b_remember_last_dataset_dir](2026/s4/1102b_remember_last_dataset_dir.md) | feature | Dataset folder is remembered in the user config dir and reused from anywhere; priority arg > `-here` > remembered > cwd; notice when it differs from the cwd. | `-here`, `last.json`, UserConfigDir, cwd, remembered folder | `main.go`, `lastdir.go`, `lastdir_test.go` |
| [1102a_tree_listing_flag](2026/s4/1102a_tree_listing_flag.md) | feature | `-tree` one-shot mode: prints the folder structure (names only), sends it to the clipboard via OSC 52 over ssh; `-depth`, `-sample`, `-dirs-only`, `-o`, `-no-copy`. | `-tree`, OSC 52, clipboard, ssh, file listing | `main.go`, `osc52.go`, `internal/dataset/tree.go` |
| [1030c_release_1_1_1](2026/s4/1030c_release_1_1_1.md) | release | Version 0.1.0 -> 1.1.1; tag, GitHub release and npm publish procedure. | release, tag, npm publish, `@cadcam` | `internal/version/version.go` |
| [1030b_dataset_sorting_implementation](2026/s4/1030b_dataset_sorting_implementation.md) | feature | Full implementation: Die/IOS classifier, scan statistics, plan/apply with flattening and a move log, token-guarded API, web UI. | `Tooth_XX.stl`, classify, flatten, plan, apply, `.scan-origin`, token, `0.0.0.0` | `main.go`, `internal/dataset/*`, `internal/server/*`, `web/*` |
| [1030a_bootstrap_from_sermon_proof](2026/s4/1030a_bootstrap_from_sermon_proof.md) | baseline | Project bootstrapped from the sermon-proof conventions: CLAUDE.md, ai-log, release + npm tooling for `@cadcam/scan-origin`, LICENSE, `.gitignore`. | bootstrap, release, npm, optionalDependencies, MIT, `@cadcam` | `CLAUDE.md`, `README.md`, `tools/*`, `internal/version`, `ai-log/*` |
