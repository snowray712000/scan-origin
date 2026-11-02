# Release 1.3.1

- Date: 2026-11-02
- Project: scan-origin
- Type: release
- Related logs: 1030c_release_1_1_1.md, 1102c_fdi_tooth_statistics.md, 1102d_export_excel.md
- Cross-project: none
- Commit: the release commit tagged `v1.3.1`

## Goal / why
The user tested the 1.2.3 (FDI statistics) and 1.2.4 (Excel export) test builds, both OK, and asked for the official release as 1.3.1 (minor bump: two new features since 1.2.2).

## What changed (behavior level)
Only `internal/version/version.go`: `Version` `1.2.4` -> `1.3.1`. Documentation (README "牙位統計" and "匯出 Excel", CLAUDE.md technical notes) was already updated with the features. `go vet ./... && go test ./...` passed before release.

## Release procedure (reproducible)
Same as 1030c_release_1_1_1.md:
1. Bump the version, `go vet ./... && go test ./...`, commit (tracked files must be clean for `tools/release`).
2. `git tag v1.3.1`.
3. `go run ./tools/release` -> `dist/scan-origin_1.3.1_{windows_amd64.zip,darwin_arm64.tar.gz,linux_amd64.tar.gz}` + `SHA256SUMS.txt`.
4. `go run ./tools/npmpack` -> `dist/npm/*.tgz`.
5. The user publishes by hand: `npm publish` the three platform tgz first, the `@cadcam/scan-origin` tgz last, **without** `--tag test` this time (so it becomes `latest`). Test builds 1.2.3 / 1.2.4 were published with `--tag test`.

## Gotchas
- Nothing was pushed by the AI (branch, tags v1.2.3, v1.2.4, v1.3.1 are local until the user asks). `origin` is `https://github.com/snowray712000/scan-origin.git`.
- `dist/` keeps the older archives (1.2.2-1.2.4); `SHA256SUMS.txt` is regenerated for the current version.
