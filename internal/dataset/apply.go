package dataset

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// applyMu serialises Apply calls; two concurrent runs would race on the same folders.
var applyMu sync.Mutex

// Op is one file system change, paths relative to the dataset root (slash separated).
type Op struct {
	Op   string `json:"op"` // flatten | move | rename | mkdir | rmdir
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

// Result is what Apply did.
type Result struct {
	Moved  int      `json:"moved"`
	Failed int      `json:"failed"`
	Ops    []Op     `json:"ops"`
	Errors []string `json:"errors"`
	Log    string   `json:"log,omitempty"` // path of the written move log
}

func chmod777(p string) { _ = os.Chmod(p, 0o777) }

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

// flatten collapses {case}/{case}/{files} into {case}/{files}, repeatedly.
// Nothing is overwritten: on a name clash the inner folder is put back.
func flatten(casePath, name string) (int, error) {
	levels := 0
	for {
		inner := filepath.Join(casePath, name)
		fi, err := os.Lstat(inner)
		if err != nil || !fi.IsDir() {
			return levels, nil
		}
		tmp := filepath.Join(casePath, ".scan-origin-flatten")
		if exists(tmp) {
			return levels, fmt.Errorf("%s 已存在", tmp)
		}
		if err := os.Rename(inner, tmp); err != nil {
			return levels, err
		}
		kids, err := os.ReadDir(tmp)
		if err != nil {
			_ = os.Rename(tmp, inner)
			return levels, err
		}
		for _, k := range kids {
			if exists(filepath.Join(casePath, k.Name())) {
				_ = os.Rename(tmp, inner)
				return levels, fmt.Errorf("展平時名稱衝突：%s", k.Name())
			}
		}
		for _, k := range kids {
			if err := os.Rename(filepath.Join(tmp, k.Name()), filepath.Join(casePath, k.Name())); err != nil {
				return levels, err
			}
		}
		if err := os.Remove(tmp); err != nil {
			return levels, err
		}
		levels++
	}
}

// findBatchDir returns an existing {date_Count} folder of the same date under kindDir.
func findBatchDir(kindDir, date string) (string, bool) {
	names, _ := listDirs(kindDir)
	sort.Strings(names)
	for _, n := range names {
		if m := batchRe.FindStringSubmatch(n); m != nil && m[1] == date {
			return n, true
		}
	}
	return "", false
}

// Apply files every movable case of every pending batch into IOS/{date_N} or
// Die/{date_N}, collapsing doubled case folders first. Existing targets are
// never overwritten (such cases stay where they are and are reported). Every
// change is recorded in <root>/.scan-origin/moves-<time>.json.
func Apply(root string) (*Result, error) {
	applyMu.Lock()
	defer applyMu.Unlock()

	plan, err := BuildPlan(root)
	if err != nil {
		return nil, err
	}
	res := &Result{Ops: []Op{}, Errors: []string{}}
	rel := func(p string) string {
		r, err := filepath.Rel(root, p)
		if err != nil {
			return p
		}
		return filepath.ToSlash(r)
	}
	fail := func(format string, a ...any) {
		res.Failed++
		res.Errors = append(res.Errors, fmt.Sprintf(format, a...))
	}

	for _, bp := range plan.Batches {
		date := batchRe.FindStringSubmatch(bp.Name)[1]
		pendingDir := filepath.Join(root, bp.Name)
		for _, kind := range []Kind{KindIOS, KindDie} {
			var group []Item
			for _, it := range plan.Items {
				if it.Batch != bp.Name {
					continue
				}
				if it.Conflict != "" {
					continue
				}
				if it.Kind == kind {
					group = append(group, it)
				}
			}
			if len(group) == 0 {
				continue
			}
			kindDir := filepath.Join(root, string(kind))
			if !exists(kindDir) {
				if err := os.MkdirAll(kindDir, 0o777); err != nil {
					return res, err
				}
				res.Ops = append(res.Ops, Op{Op: "mkdir", To: rel(kindDir)})
			}
			chmod777(kindDir)

			destName, found := findBatchDir(kindDir, date)
			created := false
			if !found {
				destName = fmt.Sprintf("%s_%d", date, len(group))
				if err := os.Mkdir(filepath.Join(kindDir, destName), 0o777); err != nil {
					return res, err
				}
				created = true
				res.Ops = append(res.Ops, Op{Op: "mkdir", To: rel(filepath.Join(kindDir, destName))})
			}
			destDir := filepath.Join(kindDir, destName)
			chmod777(destDir)

			for _, it := range group {
				src := filepath.Join(pendingDir, it.Case)
				dst := filepath.Join(destDir, it.Case)
				if exists(dst) {
					fail("%s/%s：目的地已存在 %s，未移動", it.Batch, it.Case, rel(dst))
					continue
				}
				if it.Flatten > 0 {
					n, err := flatten(src, it.Case)
					if n > 0 {
						res.Ops = append(res.Ops, Op{Op: "flatten", From: rel(src) + strings.Repeat("/"+it.Case, n), To: rel(src)})
					}
					if err != nil {
						fail("%s/%s：展平失敗：%v", it.Batch, it.Case, err)
						continue
					}
				}
				if err := os.Rename(src, dst); err != nil {
					fail("%s/%s：移動失敗：%v", it.Batch, it.Case, err)
					continue
				}
				res.Moved++
				res.Ops = append(res.Ops, Op{Op: "move", From: rel(src), To: rel(dst)})
			}

			// Keep the folder name {date_Count} equal to what it actually holds.
			cases, _ := listDirs(destDir)
			switch {
			case len(cases) == 0 && created:
				if err := os.Remove(destDir); err == nil {
					res.Ops = append(res.Ops, Op{Op: "rmdir", From: rel(destDir)})
				}
			case len(cases) > 0:
				want := fmt.Sprintf("%s_%d", date, len(cases))
				if want != destName && !exists(filepath.Join(kindDir, want)) {
					if err := os.Rename(destDir, filepath.Join(kindDir, want)); err == nil {
						res.Ops = append(res.Ops, Op{Op: "rename", From: rel(destDir), To: rel(filepath.Join(kindDir, want))})
						chmod777(filepath.Join(kindDir, want))
					}
				}
			}
		}
		// Only succeeds when nothing is left in it.
		if err := os.Remove(pendingDir); err == nil {
			res.Ops = append(res.Ops, Op{Op: "rmdir", From: rel(pendingDir)})
		}
	}

	if len(res.Ops) > 0 {
		if p, err := writeLog(root, res); err != nil {
			res.Errors = append(res.Errors, "無法寫入紀錄檔："+err.Error())
		} else {
			res.Log = rel(p)
		}
	}
	return res, nil
}

func writeLog(root string, res *Result) (string, error) {
	dir := filepath.Join(root, LogDir)
	if err := os.MkdirAll(dir, 0o777); err != nil {
		return "", err
	}
	chmod777(dir)
	now := time.Now()
	p := filepath.Join(dir, "moves-"+now.Format("20060102-150405")+".json")
	b, err := json.MarshalIndent(map[string]any{
		"time":   now.Format(time.RFC3339),
		"moved":  res.Moved,
		"failed": res.Failed,
		"ops":    res.Ops,
		"errors": res.Errors,
	}, "", "  ")
	if err != nil {
		return "", err
	}
	return p, os.WriteFile(p, append(b, '\n'), 0o666)
}
