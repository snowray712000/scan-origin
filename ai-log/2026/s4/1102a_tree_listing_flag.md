# `-tree`: print the folder structure and send it to the clipboard

- Date: 2026-11-02
- Project: scan-origin
- Type: feature
- Related logs: 1030b_dataset_sorting_implementation.md
- Cross-project: none
- Commit: uncommitted

## Goal / why
The user works over ssh on the headless ubuntu box that holds the dataset. A real path list of the dataset helps both AI and development, so they wanted one command that prints the file structure (names only, no contents) of the current folder, puts it on the clipboard, and lets them paste it elsewhere by hand.

## Starting state
`main.go` only started the HTTP server. There was no one-shot mode.

## What changed (behavior level)
`scan-origin -tree [-depth N] [-sample N] [-dirs-only] [-o FILE] [-no-copy] [folder]` (folder defaults to `.`) is a one-shot command: it never starts the server and never opens a port or a token. It is read-only, so the "never overwrite / only rename" rules are untouched.

- Prints the tree to stdout (or to `-o FILE`, then stdout is empty).
- If stdout is a terminal (`os.ModeCharDevice`) and `-no-copy` is not given, it also writes an OSC 52 sequence to stdout, which makes the user's local terminal put the text on its clipboard (works through ssh). Status text goes to stderr, so `> file` redirects stay clean and no escape sequence ends up in a file. When stdout is not a terminal it says so on stderr and skips the sequence.
- `-depth`, `-sample`, `-dirs-only`, `-o`, `-no-copy` without `-tree` exit with status 2 and a message; negative `-depth` / `-sample` also exit 2.

## Design and decisions
- Output format chosen by the user: indented tree, "AI-readable first" (not one full relative path per line). Auto-send to the clipboard is the default (user: "順手送").
- Why OSC 52: the dataset machine has no display, so `xclip` / `wl-copy` cannot work, and no third-party packages are allowed. Windows Terminal, iTerm2, kitty and WezTerm accept OSC 52 writes; inside tmux the user needs `set -g set-clipboard on`; GNU screen is not handled.
- Size guard: terminals may silently drop long sequences, so when the base64 payload exceeds `maxOSC52` (100000 bytes, roughly 70 KB of text) the sequence is NOT sent; stderr tells the user to narrow with `-depth` / `-sample` / `-dirs-only` or use `-o`. The tree is still printed.
- `-sample N` limits entries per folder (folders first, then files), so a huge batch folder shows a few cases, then `... (+M more)`. `-depth N`: 1 = children of root only; a folder cut by the depth limit that still has content is shown as `name/ ...`.
- Summary line is always `D directories, F files`; when anything was cut it appends ` (truncated: counts cover the listed entries only)`, because counts are of the listed entries, not the whole dataset.
- Root label is only the base name of the folder (`dataset/`), not the absolute path, so pasting does not leak the machine's path layout.

## Key files / types / functions
- `internal/dataset/tree.go`: `TreeOptions{MaxDepth, DirsOnly, Sample}`, `Tree(root, opt) (string, error)`.
- `osc52.go` (package main): `osc52(text) (seq, ok)`, `maxOSC52`.
- `main.go`: flags, `printTree(...)`.
- Tests: `internal/dataset/tree_test.go`, `osc52_test.go`.

## Contracts and algorithms
Tree format (box-drawing, 4-column steps, folders end with `/`, folders sorted before files, each group sorted by name via `os.ReadDir`):

```
ds/
├── Die/
│   └── 261022_1/
│       └── caseC/
│           └── x-Tooth_15.stl
├── IOS/ ...
└── readme.txt

7 directories, 5 files
```

- The `.scan-origin` move-log folder is skipped at the root only (depth 0). Other dot-names are listed.
- Symlinks are not followed (`DirEntry.IsDir()` is false for them, so they appear as files).
- An unreadable sub-folder prints `[unreadable: <err>]` in place of its children; an unreadable root returns an error (exit 1).
- OSC 52: `"\x1b]52;c;" + base64(text) + "\a"`.

## Gotchas
- The OSC 52 path can only be checked in a real terminal; tests cover the sequence format and the size guard, not delivery.
- A test first expected 3 directories where the fixture has 4 (Die, IOS and two batch folders); the code was right.

## Verification
`gofmt -l .` clean, `go vet ./... && go test ./...` pass. Ran `go run . -tree -depth 2 <tmp dir>` (tree printed, "not a terminal" note on stderr) and `go run . -depth 2` (rejected, exit 2).

## Open issues / next steps
- Not tried in a real ssh session with Windows Terminal; if a terminal rejects the sequence, `-o FILE` plus `scp` is the fallback.
