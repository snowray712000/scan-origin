package dataset

import (
	"os"
	"path/filepath"
	"sort"
)

// maxNoFDIListed caps how many case paths without a tooth number are listed.
const maxNoFDIListed = 50

// ToothStat is the number of cases that contain one FDI tooth.
type ToothStat struct {
	FDI string `json:"fdi"`
	IOS int    `json:"ios"`
	Die int    `json:"die"`
}

// TeethStats counts teeth over the classified cases. A case with several teeth
// (a bridge) counts once per distinct FDI.
type TeethStats struct {
	Teeth     []ToothStat `json:"teeth"`
	IOSTeeth  int         `json:"iosTeeth"`  // sum of IOS over Teeth
	DieTeeth  int         `json:"dieTeeth"`  // sum of Die over Teeth
	IOSCases  int         `json:"iosCases"`  // classified IOS cases looked at
	DieCases  int         `json:"dieCases"`  // classified Die cases looked at
	NoFDI     int         `json:"noFdi"`     // cases without any tooth number
	NoFDIList []string    `json:"noFdiList"` // first few of them, as IOS/261022_72/case
}

// CaseRow is one classified case and the FDI teeth found in its file names.
type CaseRow struct {
	Kind  Kind
	Batch string // folder name under IOS/ or Die/, e.g. 261022_72
	Case  string
	FDIs  []string // distinct, sorted; empty if none was found
}

// ScanCases reads the file names of every classified case (one directory
// listing per case; sorted cases are already flattened). Rows come in the
// order IOS then Die, batch by batch, case by case. It is read-only.
func ScanCases(root string) ([]CaseRow, error) {
	var rows []CaseRow
	for _, k := range []Kind{KindIOS, KindDie} {
		kindDir := filepath.Join(root, string(k))
		batches, err := listDirs(kindDir)
		if err != nil {
			return nil, err
		}
		sort.Strings(batches)
		for _, b := range batches {
			cases, err := listDirs(filepath.Join(kindDir, b))
			if err != nil {
				return nil, err
			}
			sort.Strings(cases)
			for _, c := range cases {
				ents, err := os.ReadDir(filepath.Join(kindDir, b, c))
				if err != nil {
					return nil, err
				}
				var files []string
				for _, e := range ents {
					if !e.IsDir() {
						files = append(files, e.Name())
					}
				}
				rows = append(rows, CaseRow{Kind: k, Batch: b, Case: c, FDIs: FDIs(k, files)})
			}
		}
	}
	return rows, nil
}

// SummarizeTeeth counts teeth over the rows of ScanCases.
func SummarizeTeeth(rows []CaseRow) *TeethStats {
	out := &TeethStats{Teeth: []ToothStat{}, NoFDIList: []string{}}
	byFDI := map[string]*ToothStat{}
	for _, r := range rows {
		if r.Kind == KindIOS {
			out.IOSCases++
		} else {
			out.DieCases++
		}
		if len(r.FDIs) == 0 {
			out.NoFDI++
			if len(out.NoFDIList) < maxNoFDIListed {
				out.NoFDIList = append(out.NoFDIList, string(r.Kind)+"/"+r.Batch+"/"+r.Case)
			}
			continue
		}
		for _, f := range r.FDIs {
			ts := byFDI[f]
			if ts == nil {
				ts = &ToothStat{FDI: f}
				byFDI[f] = ts
			}
			if r.Kind == KindIOS {
				ts.IOS++
				out.IOSTeeth++
			} else {
				ts.Die++
				out.DieTeeth++
			}
		}
	}
	for _, ts := range byFDI {
		out.Teeth = append(out.Teeth, *ts)
	}
	sort.Slice(out.Teeth, func(i, j int) bool { return out.Teeth[i].FDI < out.Teeth[j].FDI })
	return out
}

// ScanTeeth counts FDI teeth over the classified cases. It is read-only.
func ScanTeeth(root string) (*TeethStats, error) {
	rows, err := ScanCases(root)
	if err != nil {
		return nil, err
	}
	return SummarizeTeeth(rows), nil
}
