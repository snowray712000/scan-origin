package dataset

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TreeOptions limits how much of the folder tree is listed.
type TreeOptions struct {
	MaxDepth int  // 0 = unlimited; 1 = only the children of root
	DirsOnly bool // list folders only
	Sample   int  // 0 = all; otherwise at most this many entries per folder
}

type treeWriter struct {
	b         strings.Builder
	opt       TreeOptions
	dirs      int
	files     int
	truncated bool
}

// Tree renders the file structure below root as an indented tree (names only,
// never file contents), in the style of `tree`: folders first, then files, each
// sorted by name. Folder names end with "/". The move-log folder at the root is
// left out. Symlinks are not followed. It is read-only.
func Tree(root string, opt TreeOptions) (string, error) {
	w := &treeWriter{opt: opt}
	label := filepath.Base(root)
	if label == "." || label == string(filepath.Separator) || label == "" {
		label = root
	}
	w.b.WriteString(strings.TrimSuffix(label, string(filepath.Separator)) + "/\n")
	if err := w.walk(root, "", 0); err != nil {
		return "", err
	}
	fmt.Fprintf(&w.b, "\n%d directories, %d files", w.dirs, w.files)
	if w.truncated {
		w.b.WriteString(" (truncated: counts cover the listed entries only)")
	}
	w.b.WriteString("\n")
	return w.b.String(), nil
}

func (w *treeWriter) walk(dir, prefix string, depth int) error {
	ents, err := os.ReadDir(dir) // sorted by file name
	if err != nil {
		return err
	}
	var dirs, files []os.DirEntry
	for _, e := range ents {
		switch {
		case depth == 0 && e.Name() == LogDir:
		case e.IsDir():
			dirs = append(dirs, e)
		case !w.opt.DirsOnly:
			files = append(files, e)
		}
	}
	all := append(dirs, files...)
	omitted := 0
	if w.opt.Sample > 0 && len(all) > w.opt.Sample {
		omitted = len(all) - w.opt.Sample
		all = all[:w.opt.Sample]
		w.truncated = true
	}
	for i, e := range all {
		branch, next := "├── ", "│   "
		if i == len(all)-1 && omitted == 0 {
			branch, next = "└── ", "    "
		}
		if !e.IsDir() {
			w.files++
			w.b.WriteString(prefix + branch + e.Name() + "\n")
			continue
		}
		w.dirs++
		name := e.Name() + "/"
		child := filepath.Join(dir, e.Name())
		if w.opt.MaxDepth > 0 && depth+1 >= w.opt.MaxDepth {
			if sub, err := os.ReadDir(child); err == nil && len(sub) > 0 {
				name += " ..." // there is more below, not listed
				w.truncated = true
			}
			w.b.WriteString(prefix + branch + name + "\n")
			continue
		}
		w.b.WriteString(prefix + branch + name + "\n")
		if err := w.walk(child, prefix+next, depth+1); err != nil {
			w.b.WriteString(prefix + next + "[unreadable: " + err.Error() + "]\n")
		}
	}
	if omitted > 0 {
		w.b.WriteString(fmt.Sprintf("%s└── ... (+%d more)\n", prefix, omitted))
	}
	return nil
}
