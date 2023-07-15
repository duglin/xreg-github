package tests

import (
	"reflect"
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestBasicTypes(t *testing.T) {
	reg, _ := registry.NewRegistry("TestBasicFilters")
	defer reg.Delete()

	gm, _ := reg.AddGroupModel("dirs", "dir", "")
	gm.AddResourceModel("files", "file", 0, true, true)

	dir, _ := reg.AddGroup("dirs", "d1")
	file, _ := dir.AddResource("files", "f1", "v1")
	ver, _ := file.FindVersion("v1")

	// /dirs/d1/f1/v1

	type Prop struct {
		Name  string
		Value any
	}

	type Test struct {
		Entity any
		Props  []Prop
	}

	tests := []Test{
		Test{reg, []Prop{
			{"regString1", "str1"},
			{"regString2", ""},
			{"regInt1", 123},
			{"regInt2", -123},
			{"regInt3", 0},
			{"regBool1", true},
			{"regBool2", false},
			{"regFloat1", 123.5},
			{"regFloat2", -123.5},
			{"regFloat3", 124.0},
			{"regFloat4", 0.0},
		}},
		Test{dir, []Prop{
			{"dirString1", "str2"},
			{"dirString2", ""},
			{"dirInt1", 234},
			{"dirInt2", -234},
			{"dirInt3", 0},
			{"dirBool1", true},
			{"dirBool2", false},
			{"dirFloat1", 234.5},
			{"dirFloat2", -234.5},
			{"dirFloat3", 235.0},
			{"dirFloat4", 0.0},
		}},
		Test{file, []Prop{
			{"fileString1", "str3"},
			{"fileString2", ""},
			{"fileInt1", 345},
			{"fileInt2", -345},
			{"fileInt3", 0},
			{"fileBool1", true},
			{"fileBool2", false},
			{"fileFloat1", 345.5},
			{"fileFloat2", -345.5},
			{"fileFloat3", 346.0},
			{"fileFloat4", 0.0},
		}},
		Test{ver, []Prop{
			{"verString1", "str4"},
			{"verString2", ""},
			{"verInt1", 456},
			{"verInt2", -456},
			{"verInt3", 0},
			{"verBool1", true},
			{"verBool2", false},
			{"verFloat1", 456.5},
			{"verFloat2", -456.5},
			{"verFloat3", 457.0},
			{"verFloat4", 0.0},
		}},
	}

	for _, test := range tests {
		var entity *registry.Entity
		eField := reflect.ValueOf(test.Entity).Elem().FieldByName("Entity")
		if !eField.IsValid() {
			panic("help me")
		}
		entity = eField.Addr().Interface().(*registry.Entity)

		for _, prop := range test.Props {
			// Note that for Resources this will set them on the Resourec
			// and not the latest version. We'll test that in a diff test
			registry.SetProp(test.Entity, prop.Name, prop.Value)
		}

		entity.Extensions = map[string]any{} // force delete everything
		entity.Refresh()                     // and then re-get all props from DB

		for _, prop := range test.Props {
			got := entity.Get(prop.Name)
			if got != prop.Value {
				t.Errorf("%T) %s: got %v(%T), expected %v(%T)\n",
					test.Entity, prop.Name, got, got, prop.Value, prop.Value)
			}
		}
	}

	xCheckGet(t, reg, "?inline", `{
  "id": "TestBasicFilters",
  "self": "http:///",
  "regBool1": true,
  "regBool2": false,
  "regFloat1": 123.5,
  "regFloat2": -123.5,
  "regFloat3": 124,
  "regFloat4": 0,
  "regInt1": 123,
  "regInt2": -123,
  "regInt3": 0,
  "regString1": "str1",
  "regString2": "",

  "dirs": {
    "d1": {
      "id": "d1",
      "self": "http:///dirs/d1",
      "dirBool1": true,
      "dirBool2": false,
      "dirFloat1": 234.5,
      "dirFloat2": -234.5,
      "dirFloat3": 235,
      "dirFloat4": 0,
      "dirInt1": 234,
      "dirInt2": -234,
      "dirInt3": 0,
      "dirString1": "str2",
      "dirString2": "",

      "files": {
        "f1": {
          "id": "f1",
          "self": "http:///dirs/d1/files/f1",
          "latestId": "v1",
          "latestUrl": "http:///dirs/d1/files/f1/versions/v1",
          "fileBool1": true,
          "fileBool2": false,
          "fileFloat1": 345.5,
          "fileFloat2": -345.5,
          "fileFloat3": 346,
          "fileFloat4": 0,
          "fileInt1": 345,
          "fileInt2": -345,
          "fileInt3": 0,
          "fileString1": "str3",
          "fileString2": "",
          "verBool1": true,
          "verBool2": false,
          "verFloat1": 456.5,
          "verFloat2": -456.5,
          "verFloat3": 457,
          "verFloat4": 0,
          "verInt1": 456,
          "verInt2": -456,
          "verInt3": 0,
          "verString1": "str4",
          "verString2": "",

          "versions": {
            "v1": {
              "id": "v1",
              "self": "http:///dirs/d1/files/f1/versions/v1",
              "verBool1": true,
              "verBool2": false,
              "verFloat1": 456.5,
              "verFloat2": -456.5,
              "verFloat3": 457,
              "verFloat4": 0,
              "verInt1": 456,
              "verInt2": -456,
              "verInt3": 0,
              "verString1": "str4",
              "verString2": ""
            }
          },
          "versionsCount": 1,
          "versionsUrl": "http:///dirs/d1/files/f1/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http:///dirs/d1/files"
    }
  },
  "dirsCount": 1,
  "dirsUrl": "http:///dirs"
}
`)
}
