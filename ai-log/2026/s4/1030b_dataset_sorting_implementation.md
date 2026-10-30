# Dataset sorting implementation (classifier, stats, plan/apply, API, UI)

- Date: 2026-10-30
- Project: scan-origin
- Type: feature
- Related logs: 1030a_bootstrap_from_sermon_proof.md
- Cross-project: none
- Commit: uncommitted

## Goal / why
Someone else's dental dataset was uploaded all mixed together. Cases must be separated by file name into IOS (intra-oral scanner) and Die (desktop scan of plaster models, cut into dies), plus dataset statistics. The program runs on the dataset machine (Ubuntu) inside the dataset folder; the user opens it from a laptop browser.

## Starting state
Only bootstrap files from 1030a (version, tools, docs). Requirements in `doc/261030a/c`.

## What changed (behavior level)
Dataset layout (`{root}` = folder where scan-origin runs, or the positional argument):
```
{root}/IOS/{yyMMdd_Count}/{case}/...    sorted
{root}/Die/{yyMMdd_Count}/{case}/...
{root}/{yyMMdd_Count}/{case}/...        pending upload (name matches ^\d{6}_\d+$)
```
- **Stats** (`GET /api/state`): total (= IOS + Die, classified only), IOS, Die, pending case count, per-date rows (date shown as `2026-10-22`, sorted newest first, IOS and Die summed per date), list of pending batches.
- **Preview** (`POST /api/plan`, read-only) and **Apply** (`POST /api/apply`): for each case folder in each pending batch decide IOS/Die, collapse doubled folders, move it, write a log.
- UI: tiles (已分類總計 / IOS / Die / 待處理), per-date table, pending list, button "預覽整理結果" → plan table (batch, case, badge IOS/Die/略過, basis or note) → "確認執行（移動資料夾）" → result text (moved/failed counts, log path, errors). Apply is only reachable after a preview.

## Design and decisions
- **Classification** (`internal/dataset/classify.go`): Die iff some file name matches `(?i)-Tooth_[1-8][1-8]\.stl$`. Examples that are Die: `Conrad_C-Berry-7155-XT-(4)-F-A3-Tooth_15.stl`. Not Die although carrying an FDI number: `...-45-crown_cad.stl`, `...-45-crown_cad_transform.stl`. A case with no Die file (including an empty case) is IOS. The prefix `-` is required; digits outside 1–8 do not match.
- **Flattening**: `{case}/{case}/{files}` → `{case}/{files}`, repeated for `{case}/{case}/{case}/…`. Classification looks at the file names of the flattened result (so Die files at any nesting level of the chain count). Preview simulates it in memory (`preview`); Apply does it for real (`flatten`): rename inner dir to `.scan-origin-flatten`, check none of its children clash with names in the case dir, move children up, remove the temp dir; on clash the inner dir is renamed back and the case is skipped. Plan marks such cases `conflict` and Apply skips them.
- **Target batch folder**: `{Kind}/{date}_{N}`. If a folder of the same date already exists under that kind, merge into it (first in sorted order); afterwards rename it to `{date}_{actual case count}` if different and the new name is free. Otherwise create `{date}_{number moved}`. A newly created folder that ends up empty is removed. A kind folder that received nothing is left untouched (its name is not "corrected").
- **Never overwrite / never delete content**: if `{Kind}/{batch}/{case}` exists, the case stays in the pending batch and an error line is reported. Only `os.Rename`, `os.Mkdir*`, and `os.Remove` on empty dirs (pending batch folder after it is emptied) are used.
- **Move log**: `<root>/.scan-origin/moves-<yyyymmdd-hhmmss>.json` = `{time, moved, failed, ops[], errors[]}`, ops are `{op: mkdir|flatten|move|rename|rmdir, from?, to?}` with slash-separated paths relative to root. This is the only way to undo (manually); there is no undo feature. Dot folders are ignored by scanning and by plans.
- **Permissions**: `IOS`, `Die`, and the batch folder get `chmod 0777` (explicit chmod, because umask would otherwise narrow `Mkdir` modes; no effect on Windows). Case folders keep their own modes.
- **Network model**: differs from sermon-proof. Default listen `0.0.0.0:8765` (`-host`, `-port`; if the port was not given explicitly and is busy, a random port is used). Because the page is reached by LAN IP, the Host check was dropped; instead: random 16-byte hex token generated at start; `GET /` requires `?t=<token>` (otherwise 403) and embeds the token in the page (`__TOKEN__` replaced; `<meta name="token">`); `/api/*` requires header `X-Token`; non-GET requires `Content-Type: application/json`; a request `Origin` must equal `http://<Host>`. `app.js`/`style.css` are public (no secrets). Start-up prints the local URL and each non-loopback, non-link-local IPv4 URL with the token. No TLS.
- **Front end safety**: all dynamic text via `textContent` (case names are untrusted). Verified with a case folder named `<img src=x onerror=alert(1)>`.
- Pending detection uses only the name pattern `^\d{6}_\d+$` at the root; other root folders are ignored. Loose files inside pending batches (and dot folders) are listed as "ignored" and never touched. Symlinks are treated as files (never followed/moved).

## Key files / types / functions
- `internal/dataset/classify.go`: `Kind`, `IsDieFile`, `Classify(names) (Kind, basis)`.
- `internal/dataset/scan.go`: `Scan(root) *Stats`, `DisplayDate`, `listDirs`.
- `internal/dataset/plan.go`: `BuildPlan(root) *Plan{Batches, Items, Ignored}`, `preview`.
- `internal/dataset/apply.go`: `Apply(root) *Result{Moved, Failed, Ops, Errors, Log}` (guarded by a package mutex), `flatten`, `findBatchDir`, `writeLog`.
- `internal/server/server.go`: `New(root)`, `Handler()`, `guard`, `api`, handlers. `main.go`: flags `-host -port -no-open -version`, optional positional dataset dir. `web/{index.html,app.js,style.css,embed.go}`.

## Contracts and algorithms
JSON: `GET /api/state` → `{root, version, stats:{total, ios, die, pending, days:[{date, ios, die}], pendingBatches:[{name, date, cases}]}}`. `POST /api/plan` → `{batches:[{name,date,ios,die,skipped}], items:[{batch,case,kind,basis,flatten,conflict?}], ignored:[]}`. `POST /api/apply` → `{moved, failed, ops, errors, log?}`. Errors: `{error: "..."}` with 403 bad token / 415 non-JSON / 500.

## Gotchas
- A pending batch folder's `Count` in its name is ignored; actual contents are used. Result folder names use the actual number moved/held.
- `os.Remove` of the pending batch dir is used as the "is it empty" test, so leftover loose files keep the batch folder alive.
- Windows `chmod` is a no-op for 0777; on Linux the explicit chmod matters. Linux behaviour of permissions was not run.
- Apply recomputes the plan itself; if the filesystem changed after the preview, the result can differ from what was shown.
- Merging into an existing same-date folder renames that folder (`{date}_{N}`), which changes paths that other tools might hold.

## Verification
`go vet ./...` and `go test ./...` pass: classifier table (including the three Die and two non-Die examples from the docs), `Scan`, plan with double/triple nesting and name clash, `Apply` (flatten, merge-and-rename into an existing date, no-overwrite, idempotent second run, move log written), server token/origin/content-type guards and plan-then-apply. Ran the real binary on a fabricated dataset in a scratch directory (Windows): the stats page, preview and apply through the browser pane matched expectations and the move log looked right. NOT verified: any run against a real dataset, Linux/macOS runtime (including 777 permissions and file-system case sensitivity), access from another machine over a real LAN, Windows firewall behaviour, very large datasets (every request rescans directories; not benchmarked), concurrent Apply from two browsers beyond the mutex.

## Open issues / next steps
- No undo button; no re-classification of cases already inside IOS/Die; no progress indicator for very large applies.
- Possible extras: show the plan's target folder names, export stats as CSV, optional fixed `-token`, TLS or a reverse proxy for untrusted networks.
- Nothing is committed yet.
