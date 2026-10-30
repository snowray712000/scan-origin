# Release 1.1.1

- Date: 2026-10-30
- Project: scan-origin
- Type: release
- Related logs: 1030a_bootstrap_from_sermon_proof.md, 1030b_dataset_sorting_implementation.md
- Commit: the release commit tagged `v1.1.1`

## Goal / why
The user tested the current build successfully and asked to publish it as 1.1.1: a git tag, a GitHub release, and the npm packages.

## What changed (behavior level)
Only `internal/version/version.go`: `Version` `0.1.0` -> `1.1.1`. No runtime behavior change. `go vet ./... && go test ./...` passed before release.

## Release procedure (reproducible)
1. Bump `internal/version.Version`, run `go vet ./... && go test ./...`, commit (tracked files must be clean for `tools/release`).
2. `git tag v1.1.1` on that commit; push the branch and the tag.
3. `go run ./tools/release` -> `dist/scan-origin_1.1.1_{windows_amd64.zip,darwin_arm64.tar.gz,linux_amd64.tar.gz}` + `SHA256SUMS.txt`.
4. `go run ./tools/npmpack` -> `dist/npm/*.tgz` (three platform packages, then the `@cadcam/scan-origin` launcher).
5. GitHub release `v1.1.1` with the archives and `SHA256SUMS.txt` attached.
6. `npm publish` each tgz by hand, platform packages first, main package last.

## Pitfalls / notes
- When this release was prepared the repo had no `origin` remote, the branch was `master`, the GitHub repo `snowray712000/scan-origin` did not exist (API 404), and no `gh` CLI or token was available. npm already held `@cadcam/scan-origin` 0.1.0 (and 0.0.0-stage).
