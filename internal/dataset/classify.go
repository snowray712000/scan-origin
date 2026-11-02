// Package dataset scans a dataset folder, decides whether each case is a Die
// scan or an IOS scan, and files new uploads into the IOS / Die folders.
package dataset

import (
	"regexp"
	"sort"
)

// Kind is the scan type of a case.
type Kind string

const (
	KindIOS Kind = "IOS"
	KindDie Kind = "Die"
)

// dieRe matches "xxxxxx-Tooth_{FDI}.stl". FDI is two digits, first 1-8
// (permanent 1-4, deciduous 5-8), second 1-8. Files such as
// "...-45-crown_cad.stl" carry an FDI number but do not end with Tooth_XX.stl.
var dieRe = regexp.MustCompile(`(?i)-Tooth_([1-8][1-8])\.stl$`)

// iosToothRe matches "xxxxxx-{FDI}-crown_cad.stl", the crown design of an IOS
// case. The "..._transform.stl" companion does not match, so a tooth is not
// counted twice.
var iosToothRe = regexp.MustCompile(`(?i)-([1-8][1-8])-crown_cad\.stl$`)

// IsDieFile reports whether a file name marks its case as a Die scan.
func IsDieFile(name string) bool { return dieRe.MatchString(name) }

// FDIs returns the distinct FDI tooth numbers of a case, sorted. A Die case
// takes them from "-Tooth_XX.stl", an IOS case from "-XX-crown_cad.stl".
func FDIs(kind Kind, fileNames []string) []string {
	re := iosToothRe
	if kind == KindDie {
		re = dieRe
	}
	seen := map[string]bool{}
	var out []string
	for _, n := range fileNames {
		if m := re.FindStringSubmatch(n); m != nil && !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	sort.Strings(out)
	return out
}

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
