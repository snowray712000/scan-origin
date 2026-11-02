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

// ScanTeeth reads the file names of every classified case (one directory
// listing per case; sorted cases are already flattened) and counts FDI teeth.
// It is read-only.
func ScanTeeth(root string) (*TeethStats, error) {
	out := &TeethStats{Teeth: []ToothStat{}, NoFDIList: []string{}}
	byFDI := map[string]*ToothStat{}
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
				if k == KindIOS {
					out.IOSCases++
				} else {
					out.DieCases++
				}
				fdis := FDIs(k, files)
				if len(fdis) == 0 {
					out.NoFDI++
					if len(out.NoFDIList) < maxNoFDIListed {
						out.NoFDIList = append(out.NoFDIList, string(k)+"/"+b+"/"+c)
					}
					continue
				}
				for _, f := range fdis {
					ts := byFDI[f]
					if ts == nil {
						ts = &ToothStat{FDI: f}
						byFDI[f] = ts
					}
					if k == KindIOS {
						ts.IOS++
						out.IOSTeeth++
					} else {
						ts.Die++
						out.DieTeeth++
					}
				}
			}
		}
	}
	for _, ts := range byFDI {
		out.Teeth = append(out.Teeth, *ts)
	}
	sort.Slice(out.Teeth, func(i, j int) bool { return out.Teeth[i].FDI < out.Teeth[j].FDI })
	return out, nil
}
