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
	reg.Model.AddAttr("regInt1", registry.INTEGER)
	reg.Model.AddAttr("regInt2", registry.INTEGER)
	reg.Model.AddAttr("regInt3", registry.INTEGER)
	reg.Model.AddAttr("regString1", registry.STRING)
	reg.Model.AddAttr("regString2", registry.STRING)
	reg.Model.AddAttr("regAnyInt", registry.ANY)
	reg.Model.AddAttr("regAnyStr", registry.ANY)
	reg.Model.AddAttr("regAnyObj", registry.ANY)
	reg.Model.AddAttribute(&registry.Attribute{
		Name:     "regMapInt",
		Type:     registry.MAP,
		KeyType:  registry.STRING,
		ItemType: registry.INTEGER,
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
				Type: registry.INTEGER,
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
	gm.AddAttr("dirInt1", registry.INTEGER)
	gm.AddAttr("dirInt2", registry.INTEGER)
	gm.AddAttr("dirInt3", registry.INTEGER)
	gm.AddAttr("dirString1", registry.STRING)
	gm.AddAttr("dirString2", registry.STRING)

	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, true)
	rm.AddAttr("fileBool1", registry.BOOLEAN)
	rm.AddAttr("fileBool2", registry.BOOLEAN)
	rm.AddAttr("fileDec1", registry.DECIMAL)
	rm.AddAttr("fileDec2", registry.DECIMAL)
	rm.AddAttr("fileDec3", registry.DECIMAL)
	rm.AddAttr("fileDec4", registry.DECIMAL)
	rm.AddAttr("fileInt1", registry.INTEGER)
	rm.AddAttr("fileInt2", registry.INTEGER)
	rm.AddAttr("fileInt3", registry.INTEGER)
	rm.AddAttr("fileString1", registry.STRING)
	rm.AddAttr("fileString2", registry.STRING)

	dir, _ := reg.AddGroup("dirs", "d1")
	file, _ := dir.AddResource("files", "f1", "v1")
	ver, _ := file.FindVersion("v1")

	// /dirs/d1/f1/v1

	type Prop struct {
		Name  string
		Value any
		Fails bool
	}

	type Test struct {
		Entity registry.EntitySetter // any //  *registry.Entity
		Props  []Prop
	}

	tests := []Test{
		Test{reg, []Prop{
			{"regString1", "str1", false},
			{"regString2", "", false},
			{"regInt1", 123, false},
			{"regInt2", -123, false},
			{"regInt3", 0, false},
			{"regBool1", true, false},
			{"regBool2", false, false},
			{"regDec1", 123.5, false},
			{"regDec2", -123.5, false},
			{"regDec3", 124.0, false},
			{"regDec4", 0.0, false},
			{"regMapInt.k1", 123, false},
			{"regMapInt.k2", 234, false},
			{"regMapString.k1", "v1", false},
			{"regMapString.k2", "v2", false},
			{"regObj.objBool", true, false},
			{"regObj.objInt", 345, false},
			{"regObj.objStr", "in1", false},
			{"regAnyInt", "AnyInt", false},
			{"regAnyStr", 234.345, false},
			{"regAnyObj.int", 345, false},
			{"regAnyObj.str", "substr", false},
			{"regAnyObj2.str", "substr", true}, // unknown attr
			{"unknown_str", "error", true},     // unknown attr
			{"unknown_int", 123, true},         // unknown attr
			{"regString1", 123, true},          // bad type
			{"regInt1", "123", true},           // bad type
			{"regBool1", "123", true},          // bad type
			{"regDec1", "123", true},           // bad type
			{"regMapInt", "123", true},         // bad type
			{"regMapInt.k1", "123", true},      // bad type
			{"regMapString.k1", 123, true},     // bad type
		}},
		Test{dir, []Prop{
			{"dirString1", "str2", false},
			{"dirString2", "", false},
			{"dirInt1", 234, false},
			{"dirInt2", -234, false},
			{"dirInt3", 0, false},
			{"dirBool1", true, false},
			{"dirBool2", false, false},
			{"dirDec1", 234.5, false},
			{"dirDec2", -234.5, false},
			{"dirDec3", 235.0, false},
			{"dirDec4", 0.0, false},
		}},
		Test{file, []Prop{
			{"fileString1", "str3", false},
			{"fileString2", "", false},
			{"fileInt1", 345, false},
			{"fileInt2", -345, false},
			{"fileInt3", 0, false},
			{"fileBool1", true, false},
			{"fileBool2", false, false},
			{"fileDec1", 345.5, false},
			{"fileDec2", -345.5, false},
			{"fileDec3", 346.0, false},
			{"fileDec4", 0.0, false},
		}},
		Test{ver, []Prop{
			{"fileString1", "str4", false},
			{"fileString2", "", false},
			{"fileInt1", 456, false},
			{"fileInt2", -456, false},
			{"fileInt3", 0, false},
			{"fileBool1", true, false},
			{"fileBool2", false, false},
			{"fileDec1", 456.5, false},
			{"fileDec2", -456.5, false},
			{"fileDec3", 457.0, false},
			{"fileDec4", 0.0, false},
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
			err := setter.Set(prop.Name, prop.Value)
			if err != nil && !prop.Fails {
				t.Errorf("Error calling set (%q=%v): %s", prop.Name,
					prop.Value, err)
				return // stop fast
			}
		}

		entity.Props = map[string]any{} // force delete everything
		entity.Refresh()                // and then re-get props from DB

		for _, prop := range test.Props {
			if prop.Fails {
				continue
			}
			got := setter.Get(prop.Name) // test.Entity.Get(prop.Name)
			if got != prop.Value {
				t.Errorf("%T) %s: got %v(%T), expected %v(%T)\n",
					test.Entity, prop.Name, got, got, prop.Value, prop.Value)
				return // stop fast
			}
		}
	}

	xCheckGet(t, reg, "?inline", `{
  "specVersion": "0.5",
  "id": "TestBasicTypes",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "regAnyInt": "AnyInt",
  "regAnyObj": {
    "int": 345,
    "str": "substr"
  },
  "regAnyStr": 234.345,
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
