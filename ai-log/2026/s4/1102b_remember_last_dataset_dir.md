# Remember the last dataset folder, `-here` to override

- Date: 2026-11-02
- Project: scan-origin
- Type: feature
- Related logs: 1102a_tree_listing_flag.md, 1030b_dataset_sorting_implementation.md
- Cross-project: none
- Commit: uncommitted

## Goal / why
The user had to `cd` into the dataset folder before every run. They asked that the first run use the current folder, later runs use the folder used last time from anywhere, and a flag force "this folder" as the new one. The tool moves folders, so silently acting on a folder other than the one you are standing in is dangerous; the agreed design adds a visible notice.

## Starting state
`main.go` took an optional positional folder, default `.`. Nothing was remembered.

## What changed (behavior level)
Folder choice, highest priority first:
1. positional folder argument (relative paths resolved against the cwd);
2. `-here`: the cwd;
3. the remembered folder;
4. the cwd (first run, or nothing usable remembered).

- `-here` together with a positional folder -> message and exit 2.
- The chosen folder is written to the record only after the server listens successfully (so a failed start or a non-folder argument never changes it). `-tree` resolves the folder the same way but never writes.
- If the source is the remembered folder and it is not the same folder as the cwd (`os.SameFile`), stderr says `使用上次的資料夾: X` and `（目前資料夾是 Y；要改用目前資料夾請加 -here）`.
- If the remembered path no longer exists or is a file: fall back to the cwd with a warning on stderr (not an error).
- A failure to write the record only prints a warning; the server still starts. If `os.UserConfigDir()` fails, remembering is skipped entirely.
- The web header already shows `s.root` (from `/api/status`), so no front-end change was needed.

## Design and decisions
- Flag named `-here` (not `-init`/`-set`): "init" suggests creating something.
- Record location: `os.UserConfigDir()/scan-origin/last.json` — not in the dataset (would be a new file inside user data), not next to the executable (npm install location may be read-only). Windows `%AppData%`, Linux `~/.config`, macOS `~/Library/Application Support`.
- Positional argument is remembered too (the user named it explicitly).
- Corrupt or missing JSON is treated as "nothing remembered" silently.

## Key files / types / functions
- `lastdir.go` (package main): `lastDirFile`, `loadLastDir(file)`, `saveLastDir(file, dir)`, `chooseDir(arg, here, cwd, remembered) (dirChoice, error)`, `sameDir`, source constants `srcArg/srcHere/srcLast/srcCwd`.
- `main.go`: `-here` flag, resolution block after the `-tree` companion-flag check, save right after `net.Listen` succeeds.
- Tests: `lastdir_test.go` (priority table, `-here`+arg rejected, missing/file remembered path, save/load round trip incl. no leftover `.tmp`, corrupt JSON, `sameDir`).

## Contracts and algorithms
`last.json` (written via `last.json.tmp` + `os.Rename`, dir created with 0755, file 0644):

```json
{
  "dir": "C:\\data\\set",
  "updated": "2026-11-02T10:14:57+08:00"
}
```

`updated` is informational only; nothing reads it.

## Gotchas
- Removing the old `filepath.Abs` call left `path/filepath` unused in `main.go` (compile error); import dropped.
- Tests must never touch the real config dir; all helpers take paths as parameters. The manual check set `APPDATA` to a temp dir (Windows).

## Verification
`gofmt -l .` clean, `go vet ./... && go test ./...` pass. Manual run with `APPDATA` pointed at a temp dir: run in A (record written), run in B (notice printed, still used A), `-here` in B (used B, record updated), `-here x` (exit 2).

## Open issues / next steps
- Not committed. No release/version bump done.
- Concurrent runs with different folders just overwrite each other's record (last writer wins); judged acceptable.
