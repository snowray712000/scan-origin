// scan-origin: sorts a dataset folder of dental case uploads into IOS / Die
// and shows statistics in a browser.
package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"scanorigin/internal/dataset"
	"scanorigin/internal/server"
	"scanorigin/internal/version"
)

const defaultPort = 8765

func openBrowser(url string) {
	switch runtime.GOOS {
	case "darwin":
		_ = exec.Command("open", url).Start()
	case "windows":
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		_ = exec.Command("xdg-open", url).Start()
	}
}

// lanAddrs lists the IPv4 addresses of this machine, for the URLs to print.
func lanAddrs() []string {
	var out []string
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if ipn, ok := a.(*net.IPNet); ok && !ipn.IP.IsLoopback() && !ipn.IP.IsLinkLocalUnicast() && ipn.IP.To4() != nil {
			out = append(out, ipn.IP.String())
		}
	}
	return out
}

// printTree lists the folder structure to stdout (or outFile), then sends it to
// the clipboard of the terminal's machine via OSC 52 when stdout is a terminal.
// Messages go to stderr so that redirecting stdout stays clean.
func printTree(root string, opt dataset.TreeOptions, outFile string, copyIt bool) error {
	text, err := dataset.Tree(root, opt)
	if err != nil {
		return err
	}
	if outFile != "" {
		if err := os.WriteFile(outFile, []byte(text), 0o644); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "已寫入", outFile)
	} else {
		fmt.Print(text)
	}
	if !copyIt {
		return nil
	}
	fi, err := os.Stdout.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		fmt.Fprintln(os.Stderr, "stdout 不是終端機，未送剪貼簿")
		return nil
	}
	seq, ok := osc52(text)
	if !ok {
		fmt.Fprintf(os.Stderr, "清單太大（%d KB），超過剪貼簿上限，未送出；請加 -depth / -sample / -dirs-only 縮小，或用 -o 存檔\n", len(text)/1024)
		return nil
	}
	fmt.Print(seq)
	fmt.Fprintf(os.Stderr, "已送出剪貼簿（OSC 52）：%d 行、約 %d KB。若貼上是空的，表示終端機不支援（tmux 需 set -g set-clipboard on）\n",
		strings.Count(text, "\n"), (len(text)+1023)/1024)
	return nil
}

func main() {
	host := flag.String("host", "0.0.0.0", "address to listen on (127.0.0.1 = this machine only)")
	port := flag.Int("port", defaultPort, "port (if not given and busy, a free one is picked)")
	noOpen := flag.Bool("no-open", false, "do not open the browser on this machine")
	here := flag.Bool("here", false, "use the current folder as the dataset and remember it (default: the folder used last time)")
	showVersion := flag.Bool("version", false, "print the version and exit")
	tree := flag.Bool("tree", false, "print the file structure (names only) of the folder, send it to the clipboard via OSC 52, and exit")
	depth := flag.Int("depth", 0, "with -tree: list at most N levels (0 = all)")
	sample := flag.Int("sample", 0, "with -tree: list at most N entries per folder (0 = all)")
	dirsOnly := flag.Bool("dirs-only", false, "with -tree: list folders only")
	treeOut := flag.String("o", "", "with -tree: write the tree to this file instead of stdout")
	noCopy := flag.Bool("no-copy", false, "with -tree: do not send the tree to the clipboard")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: scan-origin [-host ADDR] [-port N] [-no-open] [-here] [-version] [dataset資料夾，預設為上次使用的資料夾；第一次為目前資料夾]")
		fmt.Fprintln(os.Stderr, "       scan-origin -tree [-depth N] [-sample N] [-dirs-only] [-o FILE] [-no-copy] [資料夾]")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *showVersion {
		fmt.Println("scan-origin", version.Version)
		return
	}

	portSet := false
	treeOnly := false // a -tree companion flag was given
	flag.Visit(func(f *flag.Flag) {
		portSet = portSet || f.Name == "port"
		switch f.Name {
		case "depth", "sample", "dirs-only", "o", "no-copy":
			treeOnly = true
		}
	})
	if treeOnly && !*tree {
		fmt.Fprintln(os.Stderr, "-depth / -sample / -dirs-only / -o / -no-copy 需搭配 -tree")
		os.Exit(2)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	lastFile, lastErr := lastDirFile()
	remembered := ""
	if lastErr == nil {
		remembered = loadLastDir(lastFile)
	}
	choice, err := chooseDir(flag.Arg(0), *here, cwd, remembered)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	root := choice.Dir
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
		fmt.Fprintln(os.Stderr, "不是資料夾:", root)
		os.Exit(1)
	}
	if choice.Warn != "" {
		fmt.Fprintln(os.Stderr, choice.Warn)
	}
	if choice.Source == srcLast && !sameDir(root, cwd) {
		fmt.Fprintf(os.Stderr, "使用上次的資料夾: %s\n  （目前資料夾是 %s；要改用目前資料夾請加 -here）\n", root, cwd)
	}

	if *tree {
		if *depth < 0 || *sample < 0 {
			fmt.Fprintln(os.Stderr, "-depth 與 -sample 不可為負數")
			os.Exit(2)
		}
		if err := printTree(root, dataset.TreeOptions{MaxDepth: *depth, DirsOnly: *dirsOnly, Sample: *sample}, *treeOut, !*noCopy); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(*host, strconv.Itoa(*port)))
	if err != nil && !portSet {
		ln, err = net.Listen("tcp", net.JoinHostPort(*host, "0"))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if lastErr == nil {
		if err := saveLastDir(lastFile, root); err != nil {
			fmt.Fprintln(os.Stderr, "無法記住這次的資料夾:", err)
		}
	}
	_, p, _ := net.SplitHostPort(ln.Addr().String())
	srv := server.New(root)

	q := "/?t=" + srv.Token()
	hostIP := net.ParseIP(*host)
	localHost := "127.0.0.1"
	if hostIP != nil && !hostIP.IsUnspecified() {
		localHost = *host // listening on one specific address only
	}
	local := "http://" + net.JoinHostPort(localHost, p) + q
	fmt.Println("scan-origin 已啟動，dataset 資料夾:", root, "(Ctrl+C 結束)")
	fmt.Println("  本機:", local)
	if hostIP == nil || hostIP.IsUnspecified() {
		for _, a := range lanAddrs() {
			fmt.Println("  區網:", "http://"+net.JoinHostPort(a, p)+q)
		}
		fmt.Println("網址含 token，知道網址的人就能整理這個資料夾；請勿外流。")
	}
	if !*noOpen {
		openBrowser(local)
	}
	if err := http.Serve(ln, srv.Handler()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
