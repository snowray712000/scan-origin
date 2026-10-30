# Bootstrap from the sermon-proof conventions

- Date: 2026-10-30
- Project: scan-origin
- Type: baseline
- Related logs: 1030b_dataset_sorting_implementation.md
- Cross-project: `D:\coding\信望愛\人工校正語音辨識\` (sermon-proof; read-only reference: CLAUDE.md, ai-log/README.md, tools/release, tools/npmpack, main.go, internal/server)
- Commit: uncommitted

## Goal / why
The user wrote three requirement files in `doc/` (`261030a_專案目的.md` purpose, `261030b_專案初始prompt.md` bootstrap prompt, `261030c_程式架構與細節.md` folder layout and rules) and asked to build the project from them. File b says: adopt sermon-proof's way of working (CLAUDE.md, ai-log, repo hygiene, cross-platform release + npm), asking the user first instead of guessing.

## Starting state
`D:\coding\scan-origin` contained only `doc/`. Not a git repo. Go 1.27.2, Node 24, npm 10 installed on Windows 10 (laptop dev machine). The reference path in the prompt (`/Users/Shared/coding/...`) is a Mac path; the same project lives at `D:\coding\信望愛\人工校正語音辨識\` on this machine.

## What changed (behavior level)
Parameters asked (AskUserQuestion) and the answers used:
- `APP_NAME` = `scan-origin` (from doc a); module `scanorigin`; Go stdlib only; vanilla HTML/JS/CSS via `go:embed`.
- npm scope `@cadcam` (doc a says organ `@cadcam`; template default `@fhlnet` was NOT used).
- GitHub repo `https://github.com/snowray712000/scan-origin`.
- Platforms: windows/amd64, darwin/arm64, linux/amd64.
- License: MIT.
- Bind address: **default `0.0.0.0` + random token** (differs from the template's 127.0.0.1-only, because the browser runs on another machine).
- Raw-data read-only: not applicable (the tool's purpose is to move dataset folders; see 1030b). No sibling projects.

Created: `go.mod` (`module scanorigin`, `go 1.22`), `.gitignore` (`.DS_Store`, `.claude/`, `/scan-origin`, `/scan-origin.exe`, `*.exe`, `*.test`, `/dist/`), `LICENSE` (MIT, snowray712000), `internal/version/version.go` (`0.1.0`), `tools/release`, `tools/npmpack`, `CLAUDE.md`, `README.md`, `ai-log/*`, and `git init` (no commit made).

## Design and decisions
- `tools/release` and `tools/npmpack` were copied from sermon-proof and only the hard-coded spots changed: import path `scanorigin/internal/version`, `appName`/`mainPkg`/exe names `scan-origin`, `repoURL`, default `-scope cadcam`, descriptions and README texts, launcher `PACKAGES` (via `@SCOPE@` placeholder). Additions over the reference: release archives also contain `LICENSE`; package.json has `"license": "MIT"` (sermon-proof had none).
- Behavior kept from the reference: archive names `<app>_<ver>_<os>_<arch>.{zip|tar.gz}` with one top folder; sorted `SHA256SUMS.txt`; `CGO_ENABLED=0 go build -trimpath -ldflags "-s -w"`; mtimes from HEAD commit; refuses a dirty tracked tree (`-allow-dirty`); npmpack verifies archives against the sums and never recompiles; esbuild-style optionalDependencies; publish order platform packages first, main last; tools print `npm publish` commands and never upload.
- Not copied: file dialogs, audio, locate, transcript packages (irrelevant). No CI, no code signing (same as the reference).

## Key files / types / functions
`tools/release/main.go`, `tools/npmpack/main.go`, `internal/version/version.go`, `CLAUDE.md`, `ai-log/README.md`.

## Contracts and algorithms
Release flow: bump `internal/version` → commit → `go run ./tools/release` → `go run ./tools/npmpack` → manual `npm publish` in the printed order. `tools/release` needs ≥1 commit (`git log -1 --format=%ct`).

## Gotchas
- Release cannot run until the first commit exists (the user has not asked for a commit yet). To test the tools without touching this repo, copy the tree into a scratch directory, `git init` + commit there, and run the tools there.
- Git on Windows prints LF→CRLF warnings (`core.autocrlf`); no `.gitattributes` was added.

## Verification
In a scratch copy with one commit: `go run ./tools/release` produced the three archives + `SHA256SUMS.txt`; `go run ./tools/npmpack` produced four `.tgz`; installing the win32-x64 tgz and the launcher tgz into a scratch npm project and running `node_modules/.bin/scan-origin -version` printed `scan-origin 0.1.0`. `GOOS=windows|darwin build` succeeded (linux cross-build succeeded inside `tools/release`). NOT verified: running on real Linux/macOS, an actual `npm publish`, the registry install path, the darwin/linux npm packages at runtime.

## Open issues / next steps
No CI, no code signing, no `NPM_TOKEN` automation. User needs an npm account with the `@cadcam` organization and `npm login` before publishing. First commit not made.
