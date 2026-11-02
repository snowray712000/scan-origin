package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChooseDirPriority(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "cwd")
	last := filepath.Join(root, "last")
	other := filepath.Join(root, "other")
	for _, d := range []string{cwd, last, other} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		name       string
		arg        string
		here       bool
		remembered string
		wantDir    string
		wantSrc    string
	}{
		{"first run uses cwd", "", false, "", cwd, srcCwd},
		{"remembered wins over cwd", "", false, last, last, srcLast},
		{"-here wins over remembered", "", true, last, cwd, srcHere},
		{"arg wins over remembered", other, false, last, other, srcArg},
		{"relative arg resolves against cwd", "sub", false, last, filepath.Join(cwd, "sub"), srcArg},
	}
	for _, c := range cases {
		got, err := chooseDir(c.arg, c.here, cwd, c.remembered)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got.Dir != c.wantDir || got.Source != c.wantSrc || got.Warn != "" {
			t.Errorf("%s: got %+v, want dir=%s source=%s", c.name, got, c.wantDir, c.wantSrc)
		}
	}
}

func TestChooseDirHereWithArgRejected(t *testing.T) {
	if _, err := chooseDir("x", true, t.TempDir(), ""); err == nil {
		t.Fatal("-here together with a folder argument must be an error")
	}
}

func TestChooseDirRememberedMissingFallsBack(t *testing.T) {
	cwd := t.TempDir()
	gone := filepath.Join(cwd, "gone")
	got, err := chooseDir("", false, cwd, gone)
	if err != nil {
		t.Fatal(err)
	}
	if got.Dir != cwd || got.Source != srcCwd || got.Warn == "" {
		t.Errorf("got %+v, want cwd fallback with a warning", got)
	}

	file := filepath.Join(cwd, "afile")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got, _ = chooseDir("", false, cwd, file)
	if got.Dir != cwd || got.Warn == "" {
		t.Errorf("a remembered file must not be used as the dataset: %+v", got)
	}
}

func TestLastDirRoundTrip(t *testing.T) {
	file := filepath.Join(t.TempDir(), "sub", "last.json")
	if got := loadLastDir(file); got != "" {
		t.Fatalf("missing file should give empty, got %q", got)
	}
	if err := saveLastDir(file, "/data/set"); err != nil {
		t.Fatal(err)
	}
	if got := loadLastDir(file); got != "/data/set" {
		t.Errorf("got %q", got)
	}
	if err := saveLastDir(file, "/data/other"); err != nil {
		t.Fatal(err)
	}
	if got := loadLastDir(file); got != "/data/other" {
		t.Errorf("overwrite failed, got %q", got)
	}
	if _, err := os.Stat(file + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("temp file left behind: %v", err)
	}
}

func TestLoadLastDirCorrupt(t *testing.T) {
	file := filepath.Join(t.TempDir(), "last.json")
	if err := os.WriteFile(file, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := loadLastDir(file); got != "" {
		t.Errorf("corrupt file should give empty, got %q", got)
	}
}

func TestSameDir(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	if !sameDir(a, a) || sameDir(a, b) || sameDir(a, filepath.Join(a, "nope")) {
		t.Error("sameDir misjudged")
	}
}
