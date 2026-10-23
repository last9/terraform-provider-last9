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

var (
	matrixGoVersions = regexp.MustCompile(`go-version:\s*\[([^\]]*)\]`)
	quotedVersion    = regexp.MustCompile(`'([^']*)'`)
)

// CI must test the toolchain releases are built with; older ones cannot
// build a module whose go directive is newer.
func TestCIMatrixMatchesGoMod(t *testing.T) {
	want := goModVersion(t)
	data, err := os.ReadFile(".github/workflows/test.yml")
	if err != nil {
		t.Fatalf("read test workflow: %v", err)
	}
	matrices := matrixGoVersions.FindAllSubmatch(data, -1)
	if len(matrices) == 0 {
		t.Fatal("test workflow has no go-version matrix")
	}
	for _, matrix := range matrices {
		for _, v := range quotedVersion.FindAllSubmatch(matrix[1], -1) {
			if got := string(v[1]); got != want {
				t.Errorf("CI matrix tests Go %s; go.mod requires %s", got, want)
			}
		}
	}
}
