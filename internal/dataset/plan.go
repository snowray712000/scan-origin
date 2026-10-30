package dataset

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Item is one case folder inside a pending batch and what would happen to it.
type Item struct {
	Batch    string `json:"batch"`
	Case     string `json:"case"`
	Kind     Kind   `json:"kind"`
	Basis    string `json:"basis"`              // the Tooth_XX.stl file that made it a Die, empty for IOS
	Flatten  int    `json:"flatten"`            // {case}/{case}/... levels to collapse first
	Conflict string `json:"conflict,omitempty"` // non-empty: the case is skipped
}

// BatchPlan summarises one pending batch.
type BatchPlan struct {
	Name    string `json:"name"`
	Date    string `json:"date"`
	IOS     int    `json:"ios"`
	Die     int    `json:"die"`
	Skipped int    `json:"skipped"`
}

// Plan is what Apply would do right now. It is computed from the file system
// and changes nothing.
type Plan struct {
	Batches []BatchPlan `json:"batches"`
	Items   []Item      `json:"items"`
	Ignored []string    `json:"ignored"` // loose files / dot folders inside pending batches
}

type node struct {
	name string
	dir  bool
	path string
}

func readNodes(dir string) ([]node, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]node, 0, len(ents))
	for _, e := range ents {
		out = append(out, node{e.Name(), e.IsDir(), filepath.Join(dir, e.Name())})
	}
	return out, nil
}

// preview simulates collapsing {case}/{case}/{files} and returns the file names
// the case would hold afterwards, the number of levels collapsed, and a
// non-empty conflict if an inner name clashes with an outer one.
func preview(casePath, name string) (files []string, levels int, conflict string, err error) {
	cur, err := readNodes(casePath)
	if err != nil {
		return nil, 0, "", err
	}
	for {
		inner := -1
		for i, n := range cur {
			if n.dir && n.name == name {
				inner = i
				break
			}
		}
		if inner < 0 {
			break
		}
		kids, err := readNodes(cur[inner].path)
		if err != nil {
			return nil, 0, "", err
		}
		rest := make([]node, 0, len(cur)-1+len(kids))
		have := map[string]bool{}
		for i, n := range cur {
			if i != inner {
				rest = append(rest, n)
				have[n.name] = true
			}
		}
		for _, k := range kids {
			if have[k.name] {
				return nil, levels, fmt.Sprintf("展平時名稱衝突：%s", k.name), nil
			}
		}
		cur = append(rest, kids...)
		levels++
	}
	for _, n := range cur {
		if !n.dir {
			files = append(files, n.name)
		}
	}
	sort.Strings(files)
	return files, levels, "", nil
}

// BuildPlan inspects every case folder in every pending batch at the root.
func BuildPlan(root string) (*Plan, error) {
	plan := &Plan{Batches: []BatchPlan{}, Items: []Item{}, Ignored: []string{}}
	top, err := listDirs(root)
	if err != nil {
		return nil, err
	}
	sort.Strings(top)
	for _, batch := range top {
		m := batchRe.FindStringSubmatch(batch)
		if m == nil {
			continue
		}
		bp := BatchPlan{Name: batch, Date: DisplayDate(m[1])}
		ents, err := os.ReadDir(filepath.Join(root, batch))
		if err != nil {
			return nil, err
		}
		for _, e := range ents {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				plan.Ignored = append(plan.Ignored, batch+"/"+e.Name())
				continue
			}
			it := Item{Batch: batch, Case: e.Name()}
			files, levels, conflict, err := preview(filepath.Join(root, batch, e.Name()), e.Name())
			if err != nil {
				return nil, err
			}
			it.Flatten = levels
			it.Conflict = conflict
			if conflict == "" {
				it.Kind, it.Basis = Classify(files)
			}
			switch {
			case conflict != "":
				bp.Skipped++
			case it.Kind == KindDie:
				bp.Die++
			default:
				bp.IOS++
			}
			plan.Items = append(plan.Items, it)
		}
		plan.Batches = append(plan.Batches, bp)
	}
	return plan, nil
}
