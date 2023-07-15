package registry

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	log "github.com/duglin/dlog"
	"github.com/duglin/xreg-github/registry"
	"github.com/kylelemons/godebug/diff"
)

func TestMain(m *testing.M) {
	log.SetVerbose(0)
	os.Exit(m.Run())
}

func AssertEqual(t *testing.T, expected, got string, text string) {
	d := Diff(expected, got)
	if d != "" {
		t.Fatalf("Results differ(%s):\n%s", text, d)
	}
}

func AssertEqualFile(t *testing.T, expectedFile, got string, text string) {
	d := Diff(LoadFile("testFiles/"+expectedFile), got)
	if d != "" {
		t.Fatalf("Results differ(%s/%s):\n%s", text, expectedFile, d)
	}
}

func AssertNoError(t *testing.T, err error, text string) {
	if err != nil {
		t.Fatalf("Unexpected error (%s): %s", text, err)
	}
}

func LoadFile(name string) string {
	file, err := ioutil.ReadFile(name)
	if err != nil {
		panic(fmt.Sprintf("Error loading file(%s): %s", name, err))
	}
	return strings.TrimRight(string(file), "\n")
}

func Grep(str string, match string) string {
	var buffer strings.Builder

	for _, line := range strings.Split(str, "\n") {
		if !strings.Contains(line, match) {
			continue
		}
		if buffer.Len() != 0 {
			buffer.WriteString("\n")
		}
		buffer.WriteString(line)
	}
	return buffer.String()
}

func Diff(left, right string) string {
	d := diff.Diff(left, right)
	if d == "" {
		return d
	}
	res := ""
	for i, s := range strings.Split(d, "\n") {
		if len(s) > 0 {
			if s[0] == '-' || s[0] == '+' {
				res += fmt.Sprintf("%d: %s\n", i, s)
			}
		}
	}
	return res
}

func LoadRegistry() *registry.Registry {
	reg := &registry.Registry{
		BaseURL: "http://example.com/",

		Model: &registry.Model{
			Groups: map[string]*registry.GroupModel{
				"endpoints": &registry.GroupModel{
					Singular: "endpoint",
					Plural:   "endpoints",

					Resources: map[string]*registry.ResourceModel{
						"definitions": &registry.ResourceModel{
							Singular: "definition",
							Plural:   "definitions",
							Versions: -1,
						},
					},
				},
				"schemaGroups": &registry.GroupModel{
					Singular: "schemaGroup",
					Plural:   "schemaGroups",

					Resources: map[string]*registry.ResourceModel{
						"schemas": &registry.ResourceModel{
							Singular: "schema",
							Plural:   "schemas",
							Versions: -1,
						},
					},
				},
			},
		},

		ID:          "123-1234-1234",
		Name:        "TestRegistry",
		Description: "Sample xRegistry",
		SpecVersion: "1.0",
		Tags: map[string]string{
			"stage": "production",
		},
		Docs: "https://xregistry.io",
	}

	group := reg.FindOrAddGroup("endpoints", "myqueue1")
	resource := group.FindOrAddResource("definitions", "msg1")
	resource.FindOrAddVersion("1.0")
	resource.FindOrAddVersion("2.0")

	resource = group.FindOrAddResource("definitions", "msg2")
	resource.FindOrAddVersion("1.0")
	resource.FindOrAddVersion("1.1")

	group = reg.FindOrAddGroup("endpoints", "myqueue2")
	resource = group.FindOrAddResource("definitions", "msg3")
	resource.FindOrAddVersion("1.0")
	resource.FindOrAddVersion("2.0")

	group = reg.FindOrAddGroup("schemaGroups", "myschemas")
	resource = group.FindOrAddResource("schemas", "myresource")
	resource.FindOrAddVersion("0.5")
	resource.FindOrAddVersion("1.0-rc")

	return reg
}

func TestLoad(t *testing.T) {
	reg := LoadRegistry()

	flags := &registry.RegistryFlags{
		BaseURL:   "",
		Indent:    "",
		InlineAll: false,
		Self:      false,
		AsDoc:     false,
		Filters:   nil,
	}

	str, err := reg.Get("", flags)
	AssertNoError(t, err, "Getting registry")
	AssertEqualFile(t, "TestLoad-1.json", str, "")

	flags = &registry.RegistryFlags{
		BaseURL:   "",
		Indent:    "",
		InlineAll: true,
		Self:      false,
		AsDoc:     false,
		Filters:   nil,
	}

	str, err = reg.Get("", flags)
	AssertNoError(t, err, "Getting registry")
	AssertEqualFile(t, "TestLoad-2.json", str, "")
}

func TestSimpleFilters(t *testing.T) {
	reg := LoadRegistry()

	flags := &registry.RegistryFlags{
		InlineAll: true,
		Filters:   nil,
	}

	type Test struct {
		name         string
		filters      []string
		expectedFile string
		expected     string
	}

	for _, test := range []Test{
		Test{
			name:    "No filters",
			filters: []string{},
			expected: `  "id": "123-1234-1234",
      "id": "myqueue1",
          "id": "msg1",
              "id": "1.0",
              "id": "2.0",
          "id": "msg2",
              "id": "1.0",
              "id": "1.1",
      "id": "myqueue2",
          "id": "msg3",
              "id": "1.0",
              "id": "2.0",
      "id": "myschemas",
          "id": "myresource",
              "id": "0.5",
              "id": "1.0-rc",`,
		},
		Test{
			name: "Single endpoints",
			filters: []string{
				"endpoints.id=myqueue1",
			},
			expected: `  "id": "123-1234-1234",
      "id": "myqueue1",
          "id": "msg1",
              "id": "1.0",
              "id": "2.0",
          "id": "msg2",
              "id": "1.0",
              "id": "1.1",
      "id": "myschemas",
          "id": "myresource",
              "id": "0.5",
              "id": "1.0-rc",`,
		},
		Test{
			name: "Single Group - endpoints and schemagroups",
			filters: []string{
				"endpoints.id=myqueue1",
				"schemaGroups.schemas.versions.id=0.5",
			},
			expected: `  "id": "123-1234-1234",
      "id": "myqueue1",
          "id": "msg1",
              "id": "1.0",
              "id": "2.0",
          "id": "msg2",
              "id": "1.0",
              "id": "1.1",
      "id": "myschemas",
          "id": "myresource",
              "id": "0.5",`,
		},
		Test{
			name: "Nested endpoints filter - def id",
			filters: []string{
				"endpoints.id=myqueue1",
				"endpoints.definitions.id=msg1",
			},
			expected: `  "id": "123-1234-1234",
      "id": "myqueue1",
          "id": "msg1",
              "id": "1.0",
              "id": "2.0",
      "id": "myschemas",
          "id": "myresource",
              "id": "0.5",
              "id": "1.0-rc",`,
		},
		Test{
			name: "Nested endpoints filter - ver id - single",
			filters: []string{
				"endpoints.id=myqueue1",
				"endpoints.definitions.versions.id=2.0",
			},
			expected: `  "id": "123-1234-1234",
      "id": "myqueue1",
          "id": "msg1",
              "id": "2.0",
      "id": "myschemas",
          "id": "myresource",
              "id": "0.5",
              "id": "1.0-rc",`,
		},
		Test{
			name: "Nested endpoints filter - ver id - multiple",
			filters: []string{
				"endpoints.id=myqueue1",
				"endpoints.definitions.versions.id=1.0",
			},
			expected: `  "id": "123-1234-1234",
      "id": "myqueue1",
          "id": "msg1",
              "id": "1.0",
          "id": "msg2",
              "id": "1.0",
      "id": "myschemas",
          "id": "myresource",
              "id": "0.5",
              "id": "1.0-rc",`,
		},
		Test{
			name: "Nested endpoints filter - no ver",
			filters: []string{
				"endpoints.id=myqueue1",
				"endpoints.definitions.versions.id=xxx",
			},
			expected: `  "id": "123-1234-1234",
      "id": "myschemas",
          "id": "myresource",
              "id": "0.5",
              "id": "1.0-rc",`,
		},
		Test{
			name: "No matching",
			filters: []string{
				"endpoints.id=xxx",
				"schemaGroups.id=xxx",
			},
			expectedFile: "TestNoMatch.json",
		},
	} {
		flags.Filters = test.filters
		str, _ := reg.Get("", flags)
		if test.expectedFile != "" {
			AssertEqualFile(t, test.expectedFile, str, test.name)
		} else {
			AssertEqual(t, test.expected, Grep(str, `"id":`), test.name)
		}
	}
}
