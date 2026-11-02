package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Which folder is the dataset: an explicit argument, -here, the folder used on
// the previous run, or (first run) the current folder.
const (
	srcArg  = "arg"  // folder given on the command line
	srcHere = "here" // -here: current folder, and remember it
	srcLast = "last" // remembered from the previous run
	srcCwd  = "cwd"  // nothing remembered (yet), current folder
)

type dirChoice struct {
	Dir    string
	Source string
	Warn   string // set when the remembered folder could not be used
}

type lastDirRecord struct {
	Dir     string `json:"dir"`
	Updated string `json:"updated"`
}

// lastDirFile is where the remembered folder lives, in the per-user config dir
// (not in the dataset and not next to the executable, which may be read-only).
func lastDirFile() (string, error) {
	d, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "scan-origin", "last.json"), nil
}

// loadLastDir returns the remembered folder, or "" if there is none or the file
// is unreadable or corrupt.
func loadLastDir(file string) string {
	b, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	var r lastDirRecord
	if json.Unmarshal(b, &r) != nil {
		return ""
	}
	return r.Dir
}

// saveLastDir writes the record through a temp file so a crash cannot leave a
// half-written last.json.
func saveLastDir(file, dir string) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(lastDirRecord{Dir: dir, Updated: time.Now().Format(time.RFC3339)}, "", "  ")
	if err != nil {
		return err
	}
	tmp := file + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

// chooseDir applies the priority: arg > -here > remembered > cwd. cwd and
// remembered must be absolute; a relative arg is resolved against cwd. A
// remembered folder that no longer exists falls back to cwd with a warning.
func chooseDir(arg string, here bool, cwd, remembered string) (dirChoice, error) {
	switch {
	case arg != "" && here:
		return dirChoice{}, fmt.Errorf("-here 不能和資料夾參數一起用")
	case arg != "":
		if !filepath.IsAbs(arg) {
			arg = filepath.Join(cwd, arg)
		}
		return dirChoice{Dir: filepath.Clean(arg), Source: srcArg}, nil
	case here:
		return dirChoice{Dir: cwd, Source: srcHere}, nil
	case remembered != "":
		if fi, err := os.Stat(remembered); err == nil && fi.IsDir() {
			return dirChoice{Dir: remembered, Source: srcLast}, nil
		}
		return dirChoice{Dir: cwd, Source: srcCwd,
			Warn: fmt.Sprintf("上次的資料夾已不存在或不是資料夾（%s），改用目前資料夾", remembered)}, nil
	}
	return dirChoice{Dir: cwd, Source: srcCwd}, nil
}

// sameDir reports whether a and b are the same existing folder.
func sameDir(a, b string) bool {
	fa, err := os.Stat(a)
	if err != nil {
		return false
	}
	fb, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(fa, fb)
}
