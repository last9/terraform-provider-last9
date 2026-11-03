package provider

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	docSectionRe     = regexp.MustCompile("^### (Required|Optional|Read-Only)$")
	docTopLevelRe    = regexp.MustCompile("^- `([a-z0-9_]+)` \\(")
	docNestedFieldRe = regexp.MustCompile("^  - `([a-z0-9_]+)` \\([^)]*\\b(Required|Optional)\\)")
)

// docSchemaFields reads the "## Schema" section of a resource doc and returns
// the section ("Required", "Optional", "Read-Only") of every top-level
// attribute, plus "Required"/"Optional" for each nested field as "block.field".
func docSchemaFields(t *testing.T, path string) (topLevel, nested map[string]string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	topLevel, nested = map[string]string{}, map[string]string{}
	inSchema, section, block := false, "", ""
	for _, line := range strings.Split(string(content), "\n") {
		switch {
		case line == "## Schema":
			inSchema = true
			continue
		case strings.HasPrefix(line, "## "):
			inSchema = false
			continue
		}
		if !inSchema {
			continue
		}
		if m := docSectionRe.FindStringSubmatch(line); m != nil {
			section, block = m[1], ""
			continue
		}
		if m := docTopLevelRe.FindStringSubmatch(line); m != nil {
			if section == "" {
				t.Fatalf("%s: attribute %q listed before any ### section", path, m[1])
			}
			if prev, dup := topLevel[m[1]]; dup {
				t.Fatalf("%s: attribute %q listed twice (%s and %s)", path, m[1], prev, section)
			}
			topLevel[m[1]], block = section, m[1]
			continue
		}
		if m := docNestedFieldRe.FindStringSubmatch(line); m != nil {
			if block == "" {
				t.Fatalf("%s: nested field %q has no parent attribute", path, m[1])
			}
			nested[block+"."+m[1]] = m[2]
		}
	}
	if len(topLevel) == 0 {
		t.Fatalf("%s: no attributes found under ## Schema", path)
	}
	return topLevel, nested
}

func schemaSection(s *schema.Schema) string {
	switch {
	case s.Required:
		return "Required"
	case s.Optional:
		return "Optional"
	default:
		return "Read-Only"
	}
}

// schemaFields mirrors docSchemaFields for a resource's schema. Only one level
// of nested blocks is walked, which is all last9_entity has.
func schemaFields(r *schema.Resource) (topLevel, nested map[string]string) {
	topLevel, nested = map[string]string{"id": "Read-Only"}, map[string]string{}
	for name, s := range r.Schema {
		topLevel[name] = schemaSection(s)
		if elem, ok := s.Elem.(*schema.Resource); ok {
			for field, fs := range elem.Schema {
				nested[name+"."+field] = schemaSection(fs)
			}
		}
	}
	return topLevel, nested
}

func diffFieldSections(t *testing.T, kind string, want, got map[string]string) {
	t.Helper()
	var problems []string
	for name, section := range want {
		switch doc, ok := got[name]; {
		case !ok:
			problems = append(problems, name+": missing from doc (schema: "+section+")")
		case doc != section:
			problems = append(problems, name+": doc says "+doc+", schema says "+section)
		}
	}
	for name := range got {
		if _, ok := want[name]; !ok {
			problems = append(problems, name+": documented but not in schema")
		}
	}
	sort.Strings(problems)
	for _, p := range problems {
		t.Errorf("%s %s", kind, p)
	}
}

// The Registry doc for last9_entity is hand-written, so nothing else keeps it
// in step with the schema.
func TestEntityDocMatchesSchema(t *testing.T) {
	wantTop, wantNested := schemaFields(resourceEntity())
	gotTop, gotNested := docSchemaFields(t, "../../docs/resources/entity.md")

	diffFieldSections(t, "attribute", wantTop, gotTop)
	diffFieldSections(t, "nested field", wantNested, gotNested)
}
