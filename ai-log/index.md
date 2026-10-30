# scan-origin — ai-log index

scan-origin is a Go tool (stdlib only, embedded vanilla web UI, token-protected HTTP server reachable over the LAN) that sorts a dental-scan dataset folder into IOS (intra-oral scan) and Die (desktop scan of cut plaster dies) cases from file names, and reports counts per upload date.

Rules for writing and reading logs: [README.md](README.md). Read this index first, then open only the logs relevant to your task.

## 2026 · s4 (Oct–Dec)

| Log | Type | Summary | Areas / keywords | Files touched |
|---|---|---|---|---|
| [1030c_release_1_1_1](2026/s4/1030c_release_1_1_1.md) | release | Version 0.1.0 -> 1.1.1; tag, GitHub release and npm publish procedure. | release, tag, npm publish, `@cadcam` | `internal/version/version.go` |
| [1030b_dataset_sorting_implementation](2026/s4/1030b_dataset_sorting_implementation.md) | feature | Full implementation: Die/IOS classifier, scan statistics, plan/apply with flattening and a move log, token-guarded API, web UI. | `Tooth_XX.stl`, classify, flatten, plan, apply, `.scan-origin`, token, `0.0.0.0` | `main.go`, `internal/dataset/*`, `internal/server/*`, `web/*` |
| [1030a_bootstrap_from_sermon_proof](2026/s4/1030a_bootstrap_from_sermon_proof.md) | baseline | Project bootstrapped from the sermon-proof conventions: CLAUDE.md, ai-log, release + npm tooling for `@cadcam/scan-origin`, LICENSE, `.gitignore`. | bootstrap, release, npm, optionalDependencies, MIT, `@cadcam` | `CLAUDE.md`, `README.md`, `tools/*`, `internal/version`, `ai-log/*` |
