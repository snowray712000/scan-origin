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
	"path/filepath"
	"runtime"
	"strconv"

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

func main() {
	host := flag.String("host", "0.0.0.0", "address to listen on (127.0.0.1 = this machine only)")
	port := flag.Int("port", defaultPort, "port (if not given and busy, a free one is picked)")
	noOpen := flag.Bool("no-open", false, "do not open the browser on this machine")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: scan-origin [-host ADDR] [-port N] [-no-open] [-version] [dataset資料夾，預設為目前資料夾]")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *showVersion {
		fmt.Println("scan-origin", version.Version)
		return
	}

	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
		fmt.Fprintln(os.Stderr, "不是資料夾:", root)
		os.Exit(1)
	}

	portSet := false
	flag.Visit(func(f *flag.Flag) { portSet = portSet || f.Name == "port" })
	ln, err := net.Listen("tcp", net.JoinHostPort(*host, strconv.Itoa(*port)))
	if err != nil && !portSet {
		ln, err = net.Listen("tcp", net.JoinHostPort(*host, "0"))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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
