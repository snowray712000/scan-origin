// Package dataset scans a dataset folder, decides whether each case is a Die
// scan or an IOS scan, and files new uploads into the IOS / Die folders.
package dataset

import "regexp"

// Kind is the scan type of a case.
type Kind string

const (
	KindIOS Kind = "IOS"
	KindDie Kind = "Die"
)

// dieRe matches "xxxxxx-Tooth_{FDI}.stl". FDI is two digits, first 1-8
// (permanent 1-4, deciduous 5-8), second 1-8. Files such as
// "...-45-crown_cad.stl" carry an FDI number but do not end with Tooth_XX.stl.
var dieRe = regexp.MustCompile(`(?i)-Tooth_[1-8][1-8]\.stl$`)

// IsDieFile reports whether a file name marks its case as a Die scan.
func IsDieFile(name string) bool { return dieRe.MatchString(name) }

// Classify returns KindDie if any file name is a Die file, otherwise KindIOS,
// plus the first Die file found (empty for IOS).
func Classify(fileNames []string) (Kind, string) {
	for _, n := range fileNames {
		if IsDieFile(n) {
			return KindDie, n
		}
	}
	return KindIOS, ""
}
