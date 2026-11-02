package dataset

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestIsDieFile(t *testing.T) {
	cases := map[string]bool{
		"Conrad_C-Berry-7155-XT-(4)-F-A3-Tooth_15.stl":    true,
		"Bozman_J-Wickens-7273-XT-(30)-F-D2-Tooth_46.stl": true,
		"Bozman_J-Scott-7149-XT-(19)-F-A3-Tooth_36.stl":   true,
		"x-tooth_51.STL": true,
		"Dobson_R-Bowers-7155-XT-(29)-F-A3.5-45-crown_cad.stl":           false,
		"Dobson_R-Bowers-7155-XT-(29)-F-A3.5-45-crown_cad_transform.stl": false,
		"x-Tooth_09.stl":     false, // first digit 0
		"x-Tooth_19.stl":     false, // second digit 9
		"x-Tooth_1.stl":      false,
		"x-Tooth_15.stl.bak": false,
		"Tooth_15.stl":       false, // needs the "xxxxxx-" prefix
		"x-Tooth_15.obj":     false,
		"x-Tooth_155.stl":    false,
	}
	for name, want := range cases {
		if got := IsDieFile(name); got != want {
			t.Errorf("IsDieFile(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestClassify(t *testing.T) {
	k, basis := Classify([]string{"a.stl", "b-crown_cad.stl", "c-Tooth_26.stl"})
	if k != KindDie || basis != "c-Tooth_26.stl" {
		t.Errorf("got %v %q", k, basis)
	}
	if k, basis := Classify([]string{"a.stl", "b-45-crown_cad.stl"}); k != KindIOS || basis != "" {
		t.Errorf("got %v %q", k, basis)
	}
	if k, _ := Classify(nil); k != KindIOS {
		t.Errorf("empty case should be IOS, got %v", k)
	}
}

func write(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func list(t *testing.T, dir string) []string {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range ents {
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out
}

func TestScan(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "IOS", "261022_72", "c1", "a.stl"))
	write(t, filepath.Join(root, "IOS", "261022_72", "c2", "a.stl"))
	write(t, filepath.Join(root, "IOS", "260910_1", "c3", "a.stl"))
	write(t, filepath.Join(root, "Die", "261022_34", "d1", "x-Tooth_11.stl"))
	write(t, filepath.Join(root, "Die", "261022_34", "stray.txt")) // a file, not a case
	write(t, filepath.Join(root, "261028_169", "p1", "a.stl"))
	write(t, filepath.Join(root, "261028_169", "p2", "a.stl"))
	write(t, filepath.Join(root, "261029_92", "p3", "a.stl"))
	write(t, filepath.Join(root, "other", "p4", "a.stl")) // not a batch name

	st, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if st.IOS != 3 || st.Die != 1 || st.Total != 4 || st.Pending != 3 {
		t.Errorf("totals: %+v", st)
	}
	wantDays := []DayStat{{Date: "2026-10-22", IOS: 2, Die: 1}, {Date: "2026-09-10", IOS: 1}}
	if len(st.Days) != 2 {
		t.Fatalf("days: %+v", st.Days)
	}
	for i, w := range wantDays {
		g := st.Days[i]
		if g.Date != w.Date || g.IOS != w.IOS || g.Die != w.Die {
			t.Errorf("day %d: got %+v want %+v", i, g, w)
		}
	}
	if len(st.PendingBatches) != 2 || st.PendingBatches[0].Name != "261028_169" || st.PendingBatches[0].Cases != 2 {
		t.Errorf("pending: %+v", st.PendingBatches)
	}
}

func TestScanEmpty(t *testing.T) {
	st, err := Scan(t.TempDir())
	if err != nil || st.Total != 0 || st.Pending != 0 || st.Days == nil || st.PendingBatches == nil {
		t.Errorf("%+v %v", st, err)
	}
}

func TestBuildPlanFlatten(t *testing.T) {
	root := t.TempDir()
	// doubled and tripled folders
	write(t, filepath.Join(root, "261028_3", "A", "A", "x-Tooth_15.stl"))
	write(t, filepath.Join(root, "261028_3", "B", "B", "B", "b.stl"))
	// clash: outer already has b.stl
	write(t, filepath.Join(root, "261028_3", "C", "C", "c.stl"))
	write(t, filepath.Join(root, "261028_3", "C", "c.stl"))
	// loose file in the batch
	write(t, filepath.Join(root, "261028_3", "note.txt"))

	plan, err := BuildPlan(root)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]Item{}
	for _, it := range plan.Items {
		got[it.Case] = it
	}
	if a := got["A"]; a.Kind != KindDie || a.Flatten != 1 || a.Conflict != "" {
		t.Errorf("A: %+v", a)
	}
	if b := got["B"]; b.Kind != KindIOS || b.Flatten != 2 {
		t.Errorf("B: %+v", b)
	}
	if c := got["C"]; c.Conflict == "" {
		t.Errorf("C should conflict: %+v", c)
	}
	if !reflect.DeepEqual(plan.Ignored, []string{"261028_3/note.txt"}) {
		t.Errorf("ignored: %v", plan.Ignored)
	}
	bp := plan.Batches[0]
	if bp.IOS != 1 || bp.Die != 1 || bp.Skipped != 1 {
		t.Errorf("batch: %+v", bp)
	}
}

func TestApply(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "261028_3", "A", "A", "x-Tooth_15.stl"))
	write(t, filepath.Join(root, "261028_3", "B", "b-45-crown_cad.stl"))
	write(t, filepath.Join(root, "261028_3", "C", "C", "c.stl"))
	write(t, filepath.Join(root, "261028_3", "C", "c.stl")) // conflict, stays
	write(t, filepath.Join(root, "261029_1", "D", "d.stl"))
	write(t, filepath.Join(root, "Die", "261029_4", "old", "x-Tooth_21.stl")) // same date already filed

	res, err := Apply(root)
	if err != nil {
		t.Fatal(err)
	}
	if res.Moved != 3 || res.Failed != 0 {
		t.Errorf("result: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(root, "Die", "261028_1", "A", "x-Tooth_15.stl")); err != nil {
		t.Error("A not flattened into Die/261028_1:", err)
	}
	if _, err := os.Stat(filepath.Join(root, "IOS", "261028_1", "B", "b-45-crown_cad.stl")); err != nil {
		t.Error("B not in IOS/261028_1:", err)
	}
	if _, err := os.Stat(filepath.Join(root, "IOS", "261029_1", "D", "d.stl")); err != nil {
		t.Error("D not in IOS/261029_1:", err)
	}
	// C is untouched and keeps its batch folder
	if _, err := os.Stat(filepath.Join(root, "261028_3", "C", "C", "c.stl")); err != nil {
		t.Error("C should be untouched:", err)
	}
	if _, err := os.Stat(filepath.Join(root, "261029_1")); !os.IsNotExist(err) {
		t.Error("empty pending batch should be removed")
	}
	// nothing new went into Die/261029_4, so that folder is left alone (name not "fixed")
	if got := list(t, filepath.Join(root, "Die")); !reflect.DeepEqual(got, []string{"261028_1", "261029_4"}) {
		t.Errorf("Die: %v", got)
	}
	if res.Log == "" {
		t.Error("no move log")
	} else if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(res.Log))); err != nil {
		t.Error(err)
	}

	// second run: nothing movable except the conflict, which stays reported
	res2, err := Apply(root)
	if err != nil || res2.Moved != 0 {
		t.Errorf("second run: %+v %v", res2, err)
	}
}

func TestApplyMergesIntoExistingDay(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "IOS", "261028_1", "old", "a.stl"))
	write(t, filepath.Join(root, "261028_2", "n1", "a.stl"))
	write(t, filepath.Join(root, "261028_2", "n2", "a.stl"))

	if _, err := Apply(root); err != nil {
		t.Fatal(err)
	}
	if got := list(t, filepath.Join(root, "IOS")); !reflect.DeepEqual(got, []string{"261028_3"}) {
		t.Fatalf("IOS: %v", got)
	}
	if got := list(t, filepath.Join(root, "IOS", "261028_3")); !reflect.DeepEqual(got, []string{"n1", "n2", "old"}) {
		t.Errorf("cases: %v", got)
	}
}

func TestApplyNeverOverwrites(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "IOS", "261028_1", "same", "keep.stl"))
	write(t, filepath.Join(root, "261028_1", "same", "new.stl"))

	res, err := Apply(root)
	if err != nil {
		t.Fatal(err)
	}
	if res.Moved != 0 || res.Failed != 1 {
		t.Errorf("result: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(root, "IOS", "261028_1", "same", "keep.stl")); err != nil {
		t.Error("existing case was disturbed")
	}
	if _, err := os.Stat(filepath.Join(root, "261028_1", "same", "new.stl")); err != nil {
		t.Error("pending case should stay put")
	}
}

func TestFDIs(t *testing.T) {
	ios := FDIs(KindIOS, []string{
		"A-35-crown_cad.stl", "A-35-crown_cad_transform.stl", "A-Tooth_36.stl", "A-36-crown_cad.STL", "A-09-crown_cad.stl",
	})
	if !reflect.DeepEqual(ios, []string{"35", "36"}) {
		t.Errorf("ios: %v", ios)
	}
	die := FDIs(KindDie, []string{"A-36-crown_cad.stl", "A-Tooth_36.stl", "A-Tooth_26.stl", "A-Tooth_99.stl"})
	if !reflect.DeepEqual(die, []string{"26", "36"}) {
		t.Errorf("die: %v", die)
	}
	if got := FDIs(KindIOS, []string{"a.stl"}); len(got) != 0 {
		t.Errorf("none: %v", got)
	}
}

func TestScanTeeth(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "IOS", "261022_2", "c1", "c1-35-crown_cad.stl"))
	write(t, filepath.Join(root, "IOS", "261022_2", "c1", "c1-35-crown_cad_transform.stl"))
	write(t, filepath.Join(root, "IOS", "261022_2", "c2", "c2-35-crown_cad.stl"))
	write(t, filepath.Join(root, "IOS", "261022_2", "c2", "c2-36-crown_cad.stl")) // bridge
	write(t, filepath.Join(root, "IOS", "261022_2", "c3", "a.stl"))               // no tooth
	write(t, filepath.Join(root, "Die", "261022_1", "d1", "d1-36-crown_cad.stl"))
	write(t, filepath.Join(root, "Die", "261022_1", "d1", "d1-Tooth_36.stl"))
	write(t, filepath.Join(root, "261028_1", "p1", "p1-Tooth_11.stl")) // pending, not counted

	got, err := ScanTeeth(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []ToothStat{{"35", 2, 0}, {"36", 1, 1}}
	if !reflect.DeepEqual(got.Teeth, want) {
		t.Errorf("teeth: %+v", got.Teeth)
	}
	if got.IOSTeeth != 3 || got.DieTeeth != 1 || got.IOSCases != 3 || got.DieCases != 1 || got.NoFDI != 1 {
		t.Errorf("totals: %+v", got)
	}
	if len(got.NoFDIList) != 1 || got.NoFDIList[0] != "IOS/261022_2/c3" {
		t.Errorf("noFdiList: %v", got.NoFDIList)
	}
	if e, err := ScanTeeth(t.TempDir()); err != nil || e.Teeth == nil || e.NoFDIList == nil {
		t.Errorf("empty: %+v %v", e, err)
	}
}

func TestExportXLSX(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "IOS", "261022_1", "c1", "c1-35-crown_cad.stl"))
	write(t, filepath.Join(root, "Die", "261022_1", "d1", "d1-Tooth_36.stl"))
	write(t, filepath.Join(root, "261028_1", "p1", "a.stl"))
	var buf bytes.Buffer
	if err := ExportXLSX(root, &buf); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var cases string
	for _, f := range zr.File {
		if f.Name == "xl/worksheets/sheet4.xml" {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			cases = string(b)
		}
	}
	for _, want := range []string{"<t>c1</t>", "<t>d1</t>", "<t>35</t>", "<t>36</t>"} {
		if !strings.Contains(cases, want) {
			t.Errorf("case sheet lacks %s: %s", want, cases)
		}
	}
	if strings.Contains(cases, "p1") {
		t.Error("pending case must not be listed")
	}
	if err := ExportXLSX(t.TempDir(), io.Discard); err != nil {
		t.Errorf("empty root: %v", err)
	}
}
