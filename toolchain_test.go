package main

import (
	"go/version"
	"os"
	"regexp"
	"testing"
)

// minSecureGo is the oldest Go release that blocks the compromised eMudhra
// intermediate CAs outright. Release builds use go.mod's go directive, so it
// must never drop below this.
const minSecureGo = "go1.27.3"

var goDirective = regexp.MustCompile(`(?m)^go (\S+)$`)

func goModVersion(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	m := goDirective.FindSubmatch(data)
	if m == nil {
		t.Fatal("go.mod has no go directive")
	}
	return string(m[1])
}

func TestGoModRequiresSecureToolchain(t *testing.T) {
	got := "go" + goModVersion(t)
	if version.Compare(got, minSecureGo) < 0 {
		t.Fatalf("go.mod requires %s; release builds need at least %s", got, minSecureGo)
	}
}
