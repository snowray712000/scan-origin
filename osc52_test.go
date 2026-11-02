package main

import (
	"strings"
	"testing"
)

func TestOSC52(t *testing.T) {
	seq, ok := osc52("hi")
	if !ok || seq != "\x1b]52;c;aGk=\a" {
		t.Errorf("got %q %v", seq, ok)
	}
	if _, ok := osc52(strings.Repeat("x", 80000)); ok {
		t.Error("an over-long payload must be refused")
	}
}
