package tests

import (
	"reflect"
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestBasicTypes(t *testing.T) {
	reg := NewRegistry("TestBasicTypes")
	defer PassDeleteReg(t, reg)

	reg.Model.AddAttr("regBool1", registry.BOOLEAN)
	reg.Model.AddAttr("regBool2", registry.BOOLEAN)
	reg.Model.AddAttr("regDec1", registry.DECIMAL)
	reg.Model.AddAttr("regDec2", registry.DECIMAL)
	reg.Model.AddAttr("regDec3", registry.DECIMAL)
	reg.Model.AddAttr("regDec4", registry.DECIMAL)
	reg.Model.AddAttr("regInt1", registry.INT)
	reg.Model.AddAttr("regInt2", registry.INT)
	reg.Model.AddAttr("regInt3", registry.INT)
	reg.Model.AddAttr("regString1", registry.STRING)
	reg.Model.AddAttr("regString2", registry.STRING)
	reg.Model.AddAttribute(&registry.Attribute{
		Name:     "regMapInt",
		Type:     registry.MAP,
		KeyType:  registry.STRING,
		ItemType: registry.INT,
	})
	reg.Model.AddAttribute(&registry.Attribute{
		Name:     "regMapString",
		Type:     registry.MAP,
		KeyType:  registry.STRING,
		ItemType: registry.STRING,
	})
	reg.Model.AddAttribute(&registry.Attribute{
		Name: "regObj",
		Type: registry.OBJECT,
		Attributes: map[string]*registry.Attribute{
			"objBool": &registry.Attribute{
				Name: "objBool",
				Type: registry.BOOLEAN,
			},
			"objInt": &registry.Attribute{
				Name: "objInt",
				Type: registry.INT,
			},
			"objStr": &registry.Attribute{
				Name: "objStr",
				Type: registry.STRING,
			},
		},
	})

	// TODO - do we need this?
	reg.Model.Save()

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddAttr("dirBool1", registry.BOOLEAN)
	gm.AddAttr("dirBool2", registry.BOOLEAN)
	gm.AddAttr("dirDec1", registry.DECIMAL)
	gm.AddAttr("dirDec2", registry.DECIMAL)
	gm.AddAttr("dirDec3", registry.DECIMAL)
	gm.AddAttr("dirDec4", registry.DECIMAL)
	gm.AddAttr("dirInt1", registry.INT)
	gm.AddAttr("dirInt2", registry.INT)
	gm.AddAttr("dirInt3", registry.INT)
	gm.AddAttr("dirString1", registry.STRING)
	gm.AddAttr("dirString2", registry.STRING)

	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, true)
	rm.AddAttr("fileBool1", registry.BOOLEAN)
	rm.AddAttr("fileBool2", registry.BOOLEAN)
	rm.AddAttr("fileDec1", registry.DECIMAL)
	rm.AddAttr("fileDec2", registry.DECIMAL)
	rm.AddAttr("fileDec3", registry.DECIMAL)
	rm.AddAttr("fileDec4", registry.DECIMAL)
	rm.AddAttr("fileInt1", registry.INT)
	rm.AddAttr("fileInt2", registry.INT)
	rm.AddAttr("fileInt3", registry.INT)
	rm.AddAttr("fileString1", registry.STRING)
	rm.AddAttr("fileString2", registry.STRING)

	dir, _ := reg.AddGroup("dirs", "d1")
	file, _ := dir.AddResource("files", "f1", "v1")
	ver, _ := file.FindVersion("v1")

	// /dirs/d1/f1/v1

	type Prop struct {
		Name  string
		Value any
	}

	type Test struct {
		Entity registry.EntitySetter // any //  *registry.Entity
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
			{"regDec1", 123.5},
			{"regDec2", -123.5},
			{"regDec3", 124.0},
			{"regDec4", 0.0},
			{"regMapInt.k1", 123},
			{"regMapInt.k2", 234},
			{"regMapString.k1", "v1"},
			{"regMapString.k2", "v2"},
			{"regObj.objBool", true},
			{"regObj.objInt", 345},
			{"regObj.objStr", "in1"},
		}},
		Test{dir, []Prop{
			{"dirString1", "str2"},
			{"dirString2", ""},
			{"dirInt1", 234},
			{"dirInt2", -234},
			{"dirInt3", 0},
			{"dirBool1", true},
			{"dirBool2", false},
			{"dirDec1", 234.5},
			{"dirDec2", -234.5},
			{"dirDec3", 235.0},
			{"dirDec4", 0.0},
		}},
		Test{file, []Prop{
			{"fileString1", "str3"},
			{"fileString2", ""},
			{"fileInt1", 345},
			{"fileInt2", -345},
			{"fileInt3", 0},
			{"fileBool1", true},
			{"fileBool2", false},
			{"fileDec1", 345.5},
			{"fileDec2", -345.5},
			{"fileDec3", 346.0},
			{"fileDec4", 0.0},
		}},
		Test{ver, []Prop{
			{"fileString1", "str4"},
			{"fileString2", ""},
			{"fileInt1", 456},
			{"fileInt2", -456},
			{"fileInt3", 0},
			{"fileBool1", true},
			{"fileBool2", false},
			{"fileDec1", 456.5},
			{"fileDec2", -456.5},
			{"fileDec3", 457.0},
			{"fileDec4", 0.0},
		}},
	}

	for _, test := range tests {
		var entity *registry.Entity
		eField := reflect.ValueOf(test.Entity).Elem().FieldByName("Entity")
		if !eField.IsValid() {
			panic("help me")
		}
		entity = eField.Addr().Interface().(*registry.Entity)
		setter := test.Entity

		for _, prop := range test.Props {
			// Note that for Resources this will set them on the latest Version
			setter.Set(prop.Name, prop.Value)
		}

		entity.Props = map[string]any{} // force delete everything
		entity.Refresh()                // and then re-get props from DB

		for _, prop := range test.Props {
			got := setter.Get(prop.Name) // test.Entity.Get(prop.Name)
			if got != prop.Value {
				t.Errorf("%T) %s: got %v(%T), expected %v(%T)\n",
					test.Entity, prop.Name, got, got, prop.Value, prop.Value)
			}
		}
	}

	xCheckGet(t, reg, "?inline", `{
  "specVersion": "0.5",
  "id": "TestBasicTypes",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "regBool1": true,
  "regBool2": false,
  "regDec1": 123.5,
  "regDec2": -123.5,
  "regDec3": 124,
  "regDec4": 0,
  "regInt1": 123,
  "regInt2": -123,
  "regInt3": 0,
  "regMapInt": {
    "k1": 123,
    "k2": 234
  },
  "regMapString": {
    "k1": "v1",
    "k2": "v2"
  },
  "regObj": {
    "objBool": true,
    "objInt": 345,
    "objStr": "in1"
  },
  "regString1": "str1",
  "regString2": "",

  "dirs": {
    "d1": {
      "id": "d1",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1",
      "dirBool1": true,
      "dirBool2": false,
      "dirDec1": 234.5,
      "dirDec2": -234.5,
      "dirDec3": 235,
      "dirDec4": 0,
      "dirInt1": 234,
      "dirInt2": -234,
      "dirInt3": 0,
      "dirString1": "str2",
      "dirString2": "",

      "files": {
        "f1": {
          "id": "f1",
          "epoch": 1,
          "self": "http://localhost:8181/dirs/d1/files/f1",
          "latestVersionId": "v1",
          "latestVersionUrl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
          "fileBool1": true,
          "fileBool2": false,
          "fileDec1": 456.5,
          "fileDec2": -456.5,
          "fileDec3": 457,
          "fileDec4": 0,
          "fileInt1": 456,
          "fileInt2": -456,
          "fileInt3": 0,
          "fileString1": "str4",
          "fileString2": "",

          "versions": {
            "v1": {
              "id": "v1",
              "epoch": 1,
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
              "latest": true,
              "fileBool1": true,
              "fileBool2": false,
              "fileDec1": 456.5,
              "fileDec2": -456.5,
              "fileDec3": 457,
              "fileDec4": 0,
              "fileInt1": 456,
              "fileInt2": -456,
              "fileInt3": 0,
              "fileString1": "str4",
              "fileString2": ""
            }
          },
          "versionsCount": 1,
          "versionsUrl": "http://localhost:8181/dirs/d1/files/f1/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http://localhost:8181/dirs/d1/files"
    }
  },
  "dirsCount": 1,
  "dirsUrl": "http://localhost:8181/dirs"
}
`)
}
