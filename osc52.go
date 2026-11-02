package main

import "encoding/base64"

// maxOSC52 is the longest base64 payload we hand to the terminal. Many
// terminals drop a longer sequence silently, which is worse than refusing.
const maxOSC52 = 100000

// osc52 returns the escape sequence that makes a terminal put text on the
// clipboard of the machine the terminal runs on (works through ssh). ok is
// false when the payload is too long to be sent reliably.
func osc52(text string) (seq string, ok bool) {
	payload := base64.StdEncoding.EncodeToString([]byte(text))
	if len(payload) > maxOSC52 {
		return "", false
	}
	return "\x1b]52;c;" + payload + "\a", true
}
