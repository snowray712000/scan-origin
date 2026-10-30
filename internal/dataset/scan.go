package dataset

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	DirIOS = "IOS"
	DirDie = "Die"
	// LogDir holds the move logs; it lives directly under the dataset root.
	LogDir = ".scan-origin"
)

// batchRe matches an upload batch folder name {yyMMdd_Count}.
var batchRe = regexp.MustCompile(`^(\d{6})_(\d+)$`)

// DayStat is the number of classified cases of one upload date.
type DayStat struct {
	Date string `json:"date"` // 2026-10-22, or the raw folder name if it is not {yyMMdd_Count}
	IOS  int    `json:"ios"`
	Die  int    `json:"die"`
	key  string
}

// PendingBatch is an upload folder at the dataset root that is not sorted yet.
type PendingBatch struct {
	Name  string `json:"name"`
	Date  string `json:"date"`
	Cases int    `json:"cases"`
}

// Stats is the current state of the dataset folder.
type Stats struct {
	Total          int            `json:"total"` // IOS + Die (classified only)
	IOS            int            `json:"ios"`
	Die            int            `json:"die"`
	Pending        int            `json:"pending"` // cases still waiting in pending batches
	Days           []DayStat      `json:"days"`
	PendingBatches []PendingBatch `json:"pendingBatches"`
}

// DisplayDate turns "261022" into "2026-10-22"; anything else is returned as is.
func DisplayDate(key string) string {
	if t, err := time.Parse("060102", key); err == nil {
		return t.Format("2006-01-02")
	}
	return key
}

// listDirs returns the names of the sub-folders of dir, skipping dot folders.
// A missing dir is not an error (IOS / Die are created on demand).
func listDirs(dir string) ([]string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range ents {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// Scan reads the dataset folder. It only lists directories; it never descends
// into a case folder.
func Scan(root string) (*Stats, error) {
	st := &Stats{Days: []DayStat{}, PendingBatches: []PendingBatch{}}
	days := map[string]*DayStat{}
	for _, k := range []Kind{KindIOS, KindDie} {
		kindDir := filepath.Join(root, string(k))
		batches, err := listDirs(kindDir)
		if err != nil {
			return nil, err
		}
		for _, b := range batches {
			cases, err := listDirs(filepath.Join(kindDir, b))
			if err != nil {
				return nil, err
			}
			key := b
			if m := batchRe.FindStringSubmatch(b); m != nil {
				key = m[1]
			}
			d := days[key]
			if d == nil {
				d = &DayStat{Date: DisplayDate(key), key: key}
				days[key] = d
			}
			if k == KindIOS {
				d.IOS += len(cases)
				st.IOS += len(cases)
			} else {
				d.Die += len(cases)
				st.Die += len(cases)
			}
		}
	}
	for _, d := range days {
		st.Days = append(st.Days, *d)
	}
	sort.Slice(st.Days, func(i, j int) bool { return st.Days[i].key > st.Days[j].key })
	st.Total = st.IOS + st.Die

	top, err := listDirs(root)
	if err != nil {
		return nil, err
	}
	sort.Strings(top)
	for _, name := range top {
		m := batchRe.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		cases, err := listDirs(filepath.Join(root, name))
		if err != nil {
			return nil, err
		}
		st.PendingBatches = append(st.PendingBatches, PendingBatch{Name: name, Date: DisplayDate(m[1]), Cases: len(cases)})
		st.Pending += len(cases)
	}
	return st, nil
}
