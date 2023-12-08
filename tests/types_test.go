package tests

import (
	"fmt"
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
	reg.Model.AddAttr("regUint1", registry.UINTEGER)
	reg.Model.AddAttr("regUint2", registry.UINTEGER)
	reg.Model.AddAttr("regTime1", registry.TIME)

	reg.Model.AddAttr("regAnyArrayInt", registry.ANY)
	reg.Model.AddAttr("regAnyArrayObj", registry.ANY)
	reg.Model.AddAttr("regAnyInt", registry.ANY)
	reg.Model.AddAttr("regAnyStr", registry.ANY)
	reg.Model.AddAttr("regAnyObj", registry.ANY)

	reg.Model.AddAttribute(&registry.Attribute{
		Name: "regArrayArrayInt",
		Type: registry.ARRAY,
		Item: &registry.Item{
			Type: registry.ARRAY,
			Item: &registry.Item{
				Type: registry.INTEGER,
			},
		},
	})

	reg.Model.AddAttribute(&registry.Attribute{
		Name: "regArrayInt",
		Type: registry.ARRAY,
		Item: &registry.Item{Type: registry.INTEGER},
	})

	reg.Model.AddAttribute(&registry.Attribute{
		Name: "regMapInt",
		Type: registry.MAP,
		Item: &registry.Item{Type: registry.INTEGER},
	})
	reg.Model.AddAttribute(&registry.Attribute{
		Name: "regMapString",
		Type: registry.MAP,
		Item: &registry.Item{Type: registry.STRING},
	})
	reg.Model.AddAttribute(&registry.Attribute{
		Name: "regObj",
		Type: registry.OBJECT,
		Item: &registry.Item{
			Attributes: map[string]*registry.Attribute{
				"objBool": &registry.Attribute{
					Name: "objBool",
					Type: registry.BOOLEAN,
				},
				"objInt": &registry.Attribute{
					Name: "objInt",
					Type: registry.INTEGER,
				},
				"objObj": &registry.Attribute{
					Name: "objObj",
					Type: registry.OBJECT,
					Item: &registry.Item{
						Attributes: map[string]*registry.Attribute{
							"ooint": &registry.Attribute{
								Name: "ooint",
								Type: registry.INTEGER,
							},
						},
					},
				},
				"objStr": &registry.Attribute{
					Name: "objStr",
					Type: registry.STRING,
				},
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
		Pass  bool
	}

	type Test struct {
		Entity registry.EntitySetter // any //  *registry.Entity
		Props  []Prop
	}

	tests := []Test{
		Test{reg, []Prop{
			{"regArrayArrayInt[1][1]", 66, true},
			{"regArrayInt[0]", 1, true},
			{"regArrayInt[2]", 3, true},
			{"regArrayInt[1]", 2, true},
			{"regBool1", true, true},
			{"regBool2", false, true},
			{"regDec1", 123.5, true},
			{"regDec2", -123.5, true},
			{"regDec3", 124.0, true},
			{"regDec4", 0.0, true},
			{"regInt1", 123, true},
			{"regInt2", -123, true},
			{"regInt3", 0, true},
			{"regMapInt.k1", 123, true},
			{"regMapInt.k2", 234, true},
			{"regMapString.k1", "v1", true},
			{"regMapString.k2", "v2", true},
			{"regString1", "str1", true},
			{"regString2", "", true},
			{"regTime1", "2006-01-02T15:04:05Z", true},
			{"regUint1", 0, true},
			{"regUint2", 333, true},

			{"regAnyArrayInt[2]", 5, true},
			{"regAnyArrayObj[2].int1", 55, true},
			{"regAnyArrayObj[2].myobj.int2", 555, true},
			{"regAnyArrayObj[0]", -5, true},
			{"regAnyInt", 123, true},
			{"regAnyObj.int", 345, true},
			{"regAnyObj.str", "substr", true},
			{"regAnyStr", "mystr", true},
			{"regObj.objBool", true, true},
			{"regObj.objInt", 345, true},
			{"regObj.objObj.ooint", 999, true},
			{"regObj.objStr", "in1", true},

			{"regAnyObj.nestobj.int", 123, true},

			{"epoch", -123, false},                   // bad uint
			{"regAnyObj2.str", "substr", false},      // unknown attr
			{"regArrayArrayInt[0][0]", "abc", false}, // bad type
			{"regArrayInt[2]", "abc", false},         // bad type
			{"regBool1", "123", false},               // bad type
			{"regDec1", "123", false},                // bad type
			{"regInt1", "123", false},                // bad type
			{"regMapInt", "123", false},              // bad type
			{"regMapInt.k1", "123", false},           // bad type
			{"regMapString.k1", 123, false},          // bad type
			{"regString1", 123, false},               // bad type
			{"regTime", "not a time", false},         // bad date format
			{"regUint1", -1, false},                  // bad uint
			{"unknown_int", 123, false},              // unknown attr
			{"unknown_str", "error", false},          // unknown attr
		}},
		Test{dir, []Prop{
			{"dirString1", "str2", true},
			{"dirString2", "", true},
			{"dirInt1", 234, true},
			{"dirInt2", -234, true},
			{"dirInt3", 0, true},
			{"dirBool1", true, true},
			{"dirBool2", false, true},
			{"dirDec1", 234.5, true},
			{"dirDec2", -234.5, true},
			{"dirDec3", 235.0, true},
			{"dirDec4", 0.0, true},
		}},
		Test{file, []Prop{
			{"fileString1", "str3", true},
			{"fileString2", "", true},
			{"fileInt1", 345, true},
			{"fileInt2", -345, true},
			{"fileInt3", 0, true},
			{"fileBool1", true, true},
			{"fileBool2", false, true},
			{"fileDec1", 345.5, true},
			{"fileDec2", -345.5, true},
			{"fileDec3", 346.0, true},
			{"fileDec4", 0.0, true},
		}},
		Test{ver, []Prop{
			{"fileString1", "str4", true},
			{"fileString2", "", true},
			{"fileInt1", 456, true},
			{"fileInt2", -456, true},
			{"fileInt3", 0, true},
			{"fileBool1", true, true},
			{"fileBool2", false, true},
			{"fileDec1", 456.5, true},
			{"fileDec2", -456.5, true},
			{"fileDec3", 457.0, true},
			{"fileDec4", 0.0, true},
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
			if err != nil && prop.Pass {
				t.Errorf("Error calling set (%q=%v): %s", prop.Name,
					prop.Value, err)
				return // stop fast
			}
			if err == nil && !prop.Pass {
				t.Errorf("Setting (%q=%v) was supposed to fail", prop.Name,
					prop.Value)
				return // stop fast
			}
		}

		entity.Props = map[string]any{} // force delete everything
		entity.Refresh()                // and then re-get props from DB

		for _, prop := range test.Props {
			if !prop.Pass {
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
  "regAnyArrayInt": [
    null,
    null,
    5
  ],
  "regAnyArrayObj": [
    -5,
    null,
    {
      "int1": 55,
      "myobj": {
        "int2": 555
      }
    }
  ],
  "regAnyInt": 123,
  "regAnyObj": {
    "int": 345,
    "nestobj": {
      "int": 123
    },
    "str": "substr"
  },
  "regAnyStr": "mystr",
  "regArrayArrayInt": [
    null,
    [
      null,
      66
    ]
  ],
  "regArrayInt": [
    1,
    2,
    3
  ],
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
    "objObj": {
      "ooint": 999
    },
    "objStr": "in1"
  },
  "regString1": "str1",
  "regString2": "",
  "regTime1": "2006-01-02T15:04:05Z",
  "regUint1": 0,
  "regUint2": 333,

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

func TestWildcardBoolTypes(t *testing.T) {
	reg := NewRegistry("TestWildcardBoolTypes")
	defer PassDeleteReg(t, reg)

	reg.Model.AddAttr("*", registry.BOOLEAN)
	reg.Model.Save()

	err := reg.Set("bogus", "foo")
	xCheck(t, err.Error() == `"foo" should be a boolean`,
		fmt.Sprintf("bogus=foo: %s", err))

	err = reg.Set("ext1", true)
	xCheck(t, err == nil, fmt.Sprintf("set ext1: %s", err))
	reg.Refresh()
	val := reg.Get("ext1")
	xCheck(t, val == true, fmt.Sprintf("get ext1: %v", val))

	err = reg.Set("ext1", false)
	xCheck(t, err == nil, fmt.Sprintf("set ext1-2: %s", err))
	reg.Refresh()
	xCheck(t, reg.Get("ext1") == false, fmt.Sprintf("get ext1-2: %v", val))
}

func TestWildcardAnyTypes(t *testing.T) {
	reg := NewRegistry("TestWildcardAnyTypes")
	defer PassDeleteReg(t, reg)

	reg.Model.AddAttr("*", registry.ANY)
	reg.Model.Save()

	// Make sure we can set the same attr to two different types
	err := reg.Set("ext1", 5.5)
	xCheck(t, err == nil, fmt.Sprintf("set ext1: %s", err))
	reg.Refresh()
	val := reg.Get("ext1")
	xCheck(t, val == 5.5, fmt.Sprintf("get ext1: %v", val))

	err = reg.Set("ext1", "foo")
	xCheck(t, err == nil, fmt.Sprintf("set ext2: %s", err))
	reg.Refresh()
	val = reg.Get("ext1")
	xCheck(t, val == "foo", fmt.Sprintf("get ext2: %v", val))

	// Make sure we add one of a different type
	err = reg.Set("ext2", true)
	xCheck(t, err == nil, fmt.Sprintf("set ext3 %s", err))
	reg.Refresh()
	val = reg.Get("ext2")
	xCheck(t, val == true, fmt.Sprintf("get ext3: %v", val))
}

func TestWildcard2LayersTypes(t *testing.T) {
	reg := NewRegistry("TestWildcardAnyTypes")
	defer PassDeleteReg(t, reg)

	reg.Model.AddAttribute(&registry.Attribute{
		Name: "obj",
		Type: registry.OBJECT,
		Item: &registry.Item{
			Attributes: map[string]*registry.Attribute{
				"map": {
					Name: "map",
					Type: registry.MAP,
					Item: &registry.Item{Type: registry.INTEGER},
				},
				"*": {
					Name: "*",
					Type: registry.ANY,
				},
			},
		},
	})
	reg.Model.Save()

	err := reg.Set("obj.map.k1", 5)
	xCheck(t, err == nil, fmt.Sprintf("set foo.k1: %s", err))
	reg.Refresh()
	val := reg.Get("obj.map.k1")
	xCheck(t, val == 5, fmt.Sprintf("get foo.k1: %v", val))

	err = reg.Set("obj.map.foo.k1.k2", 5)
	xCheck(t, err.Error() == `Traversing into scalar "foo": obj.map.foo.k1.k2`,
		fmt.Sprintf("set obj.map.foo.k1.k2: %s", err))

	err = reg.Set("obj.myany.foo.k1.k2", 5)
	reg.Refresh()
	val = reg.Get("obj.myany.foo.k1.k2")
	xCheck(t, val == 5, fmt.Sprintf("set obj.myany.foo.k1.k2: %v", val))
	val = reg.Get("obj.myany.bogus.k1.k2")
	xCheck(t, val == nil, fmt.Sprintf("set obj.myany.bogus.k1.k2: %v", val))

}
