package dataset

import (
	"path/filepath"
	"strings"
	"testing"
)

func treeFixture(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "ds")
	write(t, filepath.Join(root, "IOS", "261022_2", "caseA", "a.stl"))
	write(t, filepath.Join(root, "IOS", "261022_2", "caseA", "b.stl"))
	write(t, filepath.Join(root, "IOS", "261022_2", "caseB", "c.stl"))
	write(t, filepath.Join(root, "Die", "261022_1", "caseC", "x-Tooth_15.stl"))
	write(t, filepath.Join(root, LogDir, "moves-1.json")) // must not be listed
	write(t, filepath.Join(root, "readme.txt"))
	return root
}

func TestTreeFull(t *testing.T) {
	got, err := Tree(treeFixture(t), TreeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want := `ds/
├── Die/
│   └── 261022_1/
│       └── caseC/
│           └── x-Tooth_15.stl
├── IOS/
│   └── 261022_2/
│       ├── caseA/
│       │   ├── a.stl
│       │   └── b.stl
│       └── caseB/
│           └── c.stl
└── readme.txt

7 directories, 5 files
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if strings.Contains(got, LogDir) {
		t.Error("move-log folder must be left out")
	}
}

func TestTreeDepth(t *testing.T) {
	got, err := Tree(treeFixture(t), TreeOptions{MaxDepth: 2})
	if err != nil {
		t.Fatal(err)
	}
	want := `ds/
├── Die/
│   └── 261022_1/ ...
├── IOS/
│   └── 261022_2/ ...
└── readme.txt

4 directories, 1 files (truncated: counts cover the listed entries only)
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestTreeDirsOnly(t *testing.T) {
	got, err := Tree(treeFixture(t), TreeOptions{DirsOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, ".stl") || strings.Contains(got, "readme") {
		t.Errorf("files listed in dirs-only mode:\n%s", got)
	}
	if !strings.Contains(got, "caseB/") {
		t.Errorf("folders missing:\n%s", got)
	}
}

func TestTreeSample(t *testing.T) {
	got, err := Tree(treeFixture(t), TreeOptions{Sample: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "└── ... (+2 more)") || strings.Contains(got, "IOS/") {
		t.Errorf("root should keep 1 entry and note 2 more:\n%s", got)
	}
	if !strings.Contains(got, "truncated") {
		t.Errorf("summary should say truncated:\n%s", got)
	}
}

func TestTreeMissingRoot(t *testing.T) {
	if _, err := Tree(filepath.Join(t.TempDir(), "nope"), TreeOptions{}); err == nil {
		t.Error("expected an error for a missing folder")
	}
}
