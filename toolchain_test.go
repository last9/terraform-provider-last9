package main

import (
	"errors"
	"fmt"
	"go/version"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
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

// workflow is the part of a GitHub Actions workflow the CI matrix guard reads.
type workflow struct {
	Jobs map[string]struct {
		Strategy struct {
			Matrix map[string]yaml.Node `yaml:"matrix"`
		} `yaml:"strategy"`
	} `yaml:"jobs"`
}

// ciMatrixGoVersions returns every go-version matrix entry in a workflow. It
// fails on entries it cannot read as a version, so a malformed matrix can
// never pass the guard by contributing nothing to compare.
func ciMatrixGoVersions(data []byte) ([]string, error) {
	var wf workflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, err
	}
	var versions []string
	for _, name := range slices.Sorted(maps.Keys(wf.Jobs)) {
		matrix, ok := wf.Jobs[name].Strategy.Matrix["go-version"]
		if !ok {
			continue
		}
		if matrix.Kind != yaml.SequenceNode || len(matrix.Content) == 0 {
			return nil, fmt.Errorf("job %s: go-version must be a non-empty list", name)
		}
		for _, entry := range matrix.Content {
			if entry.Kind != yaml.ScalarNode || entry.Value == "" {
				return nil, fmt.Errorf("job %s: go-version entry on line %d is not a version", name, entry.Line)
			}
			versions = append(versions, entry.Value)
		}
	}
	if len(versions) == 0 {
		return nil, errors.New("no go-version matrix")
	}
	return versions, nil
}

func TestCIMatrixGoVersionsParsesEveryEntry(t *testing.T) {
	tests := []struct {
		name     string
		workflow string
		want     []string
	}{
		{"single quoted", "go-version: ['1.27.2']", []string{"1.27.2"}},
		{"double quoted", `go-version: ["1.27.2"]`, []string{"1.27.2"}},
		{"unquoted", "go-version: [1.27.2]", []string{"1.27.2"}},
		{"mixed quotes", `go-version: ['1.27.3', "1.27.2", 1.27.1]`, []string{"1.27.3", "1.27.2", "1.27.1"}},
		{"block sequence", "go-version:\n  - '1.27.3'\n  - 1.27.2", []string{"1.27.3", "1.27.2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow := "jobs:\n  test:\n    strategy:\n      matrix:\n        " +
				strings.ReplaceAll(tt.workflow, "\n", "\n        ")
			got, err := ciMatrixGoVersions([]byte(workflow))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCIMatrixGoVersionsRejectsMalformedMatrix(t *testing.T) {
	tests := []struct {
		name     string
		workflow string
	}{
		{"no matrix", "jobs:\n  test:\n    runs-on: ubuntu-latest"},
		{"empty list", "jobs:\n  test:\n    strategy:\n      matrix:\n        go-version: []"},
		{"blank entry", "jobs:\n  test:\n    strategy:\n      matrix:\n        go-version: ['']"},
		{"scalar", "jobs:\n  test:\n    strategy:\n      matrix:\n        go-version: '1.27.3'"},
		{"nested list", "jobs:\n  test:\n    strategy:\n      matrix:\n        go-version: [['1.27.3']]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := ciMatrixGoVersions([]byte(tt.workflow)); err == nil {
				t.Fatalf("got %q, want an error", got)
			}
		})
	}
}

// CI must test the toolchain releases are built with; older ones cannot
// build a module whose go directive is newer.
func TestCIMatrixMatchesGoMod(t *testing.T) {
	want := goModVersion(t)
	data, err := os.ReadFile(".github/workflows/test.yml")
	if err != nil {
		t.Fatalf("read test workflow: %v", err)
	}
	versions, err := ciMatrixGoVersions(data)
	if err != nil {
		t.Fatalf("test workflow: %v", err)
	}
	for _, got := range versions {
		if got != want {
			t.Errorf("CI matrix tests Go %s; go.mod requires %s", got, want)
		}
	}
}
