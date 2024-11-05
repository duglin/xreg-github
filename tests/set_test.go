package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestSetAttributeNames(t *testing.T) {
	reg := NewRegistry("TestSetAttributeName")
	defer PassDeleteReg(t, reg)

	type test struct {
		name string
		msg  string
	}

	sixty := "a23456789012345678901234567890123456789012345678901234567890"

	tests := []test{
		{sixty + "12", ""},
		{sixty + "123", ""},
		{"_123", ""},
		{"_12_3", ""},
		{"_123_", ""},
		{"_123_", ""},
		{"_", ""},
		{"__", ""},
		{sixty + "1234", "Invalid attribute name: "},
		{"1234", "Invalid attribute name: "},
		{"A", "Invalid attribute name: "},
		{"aA", "Invalid attribute name: "},
		{"_A", "Invalid attribute name: "},
		{"_ _", "Invalid attribute name: "},
	}

	for _, test := range tests {
		_, err := reg.Model.AddAttr(test.name, registry.STRING)
		if test.msg == "" && err != nil {
			t.Errorf("Name: %q failed: %s", test.name, err)
		}
		if test.msg != "" && (err == nil || !strings.HasPrefix(err.Error(), test.msg)) {
			t.Errorf("Name: %q should have failed", test.name)
		}
	}
}

func TestSetResource(t *testing.T) {
	reg := NewRegistry("TestSetResource")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	dir, _ := reg.AddGroup("dirs", "d1")
	file, _ := dir.AddResource("files", "f1", "v1")

	// /dirs/d1/f1/v1

	// Make sure setting it on the version is seen by res.Default and res
	namePP := NewPP().P("name").UI()
	file.SetSave(namePP, "myName")
	ver, _ := file.FindVersion("v1", false)
	val := ver.Get(namePP)
	if val != "myName" {
		t.Errorf("ver.Name is %q, should be 'myName'", val)
	}

	name := file.Get(namePP).(string)
	xCheckEqual(t, "", name, "myName")

	// Verify that nil and "" are treated differently
	ver.SetSave(namePP, nil)
	ver2, _ := file.FindVersion(ver.UID, false)
	xJSONCheck(t, ver2, ver)
	val = ver.Get(namePP)
	xCheck(t, val == nil, "Setting to nil should return nil")

	ver.SetSave(namePP, "")
	ver2, _ = file.FindVersion(ver.UID, false)
	xJSONCheck(t, ver2, ver)
	val = ver.Get(namePP)
	xCheck(t, val == "", "Setting to '' should return ''")
}

func TestSetVersion(t *testing.T) {
	reg := NewRegistry("TestSetVersion")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	dir, _ := reg.AddGroup("dirs", "d1")
	file, _ := dir.AddResource("files", "f1", "v1")
	ver, _ := file.FindVersion("v1", false)

	// /dirs/d1/f1/v1

	// Make sure setting it on the version is seen by res.Default and res
	namePP := NewPP().P("name").UI()
	ver.SetSave(namePP, "myName")
	file, _ = dir.FindResource("files", "f1", false)
	l, err := file.GetDefault()
	xNoErr(t, err)
	xCheck(t, l != nil, "default is nil")
	val := l.Get(namePP)
	if val != "myName" {
		t.Errorf("resource.default.Name is %q, should be 'myName'", val)
	}
	val = file.Get(namePP)
	if val != "myName" {
		t.Errorf("resource.Name is %q, should be 'myName'", val)
	}

	// Make sure we can also still see it from the version itself
	ver, _ = file.FindVersion("v1", false)
	val = ver.Get(namePP)
	if val != "myName" {
		t.Errorf("version.Name is %q, should be 'myName'", val)
	}
}

func TestSetDots(t *testing.T) {
	reg := NewRegistry("TestSetDots")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	// check some dots in the prop names - and some labels stuff too
	dir, _ := reg.AddGroup("dirs", "d1")
	labels := NewPP().P("labels")

	// xNoErr(t, reg.Commit())

	err := dir.SetSave(labels.UI(), "xxx")
	xCheck(t, err != nil, "labels=xxx should fail")

	// Nesting under labels should fail
	err = dir.SetSave(labels.P("xxx").P("yyy").UI(), "xy")
	xJSONCheck(t, err, `Attribute "labels.xxx" must be a string`)

	// dots are ok as tag names
	err = dir.SetSave(labels.P("abc.def").UI(), "ABC")
	xNoErr(t, err)
	xJSONCheck(t, dir.Get(labels.P("abc.def").UI()), "ABC")

	dir.Refresh()

	xCheckGet(t, reg, "/dirs/d1", `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "epoch": 1,
  "labels": {
    "abc.def": "ABC"
  },
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filescount": 0,
  "filesurl": "http://localhost:8181/dirs/d1/files"
}
`)

	err = dir.SetSave("labels", nil)
	xJSONCheck(t, err, nil)
	xCheckGet(t, reg, "/dirs/d1", `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "filescount": 0,
  "filesurl": "http://localhost:8181/dirs/d1/files"
}
`)

	err = dir.SetSave(NewPP().P("labels").P("xxx/yyy").UI(), nil)
	xCheck(t, err.Error() == `Unexpected / in "labels.xxx/yyy" at pos 11`,
		fmt.Sprintf("labels.xxx/yyy=nil should fail: %s", err))

	err = dir.SetSave(NewPP().P("labels").P("").P("abc").UI(), nil)
	xJSONCheck(t, err, `Unexpected . in "labels..abc" at pos 8`)

	err = dir.SetSave(NewPP().P("labels").P("xxx.yyy").UI(), "xxx")
	xJSONCheck(t, err, nil)

	err = dir.SetSave(NewPP().P("xxx.yyy").UI(), nil)
	xJSONCheck(t, err, `Invalid extension(s): xxx`)
	xCheck(t, err != nil, "xxx.yyy=nil should fail")
	err = dir.SetSave("xxx.", "xxx")
	xCheck(t, err != nil, "xxx.=xxx should fail")
	err = dir.SetSave(".xxx", "xxx")
	xCheck(t, err != nil, ".xxx=xxx should fail")
	err = dir.SetSave(".xxx.", "xxx")
	xCheck(t, err != nil, ".xxx.=xxx should fail")
}

func TestSetLabels(t *testing.T) {
	reg := NewRegistry("TestSetLabels")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	dir, _ := reg.AddGroup("dirs", "d1")
	file, _ := dir.AddResource("files", "f1", "v1")
	ver, _ := file.FindVersion("v1", false)
	ver2, _ := file.AddVersion("v2")

	// /dirs/d1/f1/v1
	labels := NewPP().P("labels")
	err := reg.SetSave(labels.P("r2").UI(), "123.234")
	xNoErr(t, err)
	reg.Refresh()
	// But it's a string here because labels is a map[string]string
	xJSONCheck(t, reg.Get(labels.P("r2").UI()), "123.234")
	err = reg.SetSave("labels.r1", "foo")
	xNoErr(t, err)
	reg.Refresh()
	xJSONCheck(t, reg.Get(labels.P("r1").UI()), "foo")
	err = reg.SetSave(labels.P("r1").UI(), nil)
	xNoErr(t, err)
	reg.Refresh()
	xJSONCheck(t, reg.Get(labels.P("r1").UI()), nil)

	err = dir.SetSave(labels.P("d1").UI(), "bar")
	xNoErr(t, err)
	dir.Refresh()
	xJSONCheck(t, dir.Get(labels.P("d1").UI()), "bar")
	// test override
	err = dir.SetSave(labels.P("d1").UI(), "foo")
	xNoErr(t, err)
	dir.Refresh()
	xJSONCheck(t, dir.Get(labels.P("d1").UI()), "foo")
	err = dir.SetSave(labels.P("d1").UI(), nil)
	xNoErr(t, err)
	dir.Refresh()
	xJSONCheck(t, dir.Get(labels.P("d1").UI()), nil)

	err = file.SetSave(labels.P("f1").UI(), "foo")
	xNoErr(t, err)
	file.Refresh()
	xJSONCheck(t, file.Get(labels.P("f1").UI()), "foo")
	err = file.SetSave(labels.P("f1").UI(), nil)
	xNoErr(t, err)
	file.Refresh()
	xJSONCheck(t, file.Get(labels.P("f1").UI()), nil)

	// Set before we refresh to see if creating v2 causes issues
	// see comment below too
	err = ver.SetSave(labels.P("v1").UI(), "foo")
	xNoErr(t, err)
	ver.Refresh()
	xJSONCheck(t, ver.Get(labels.P("v1").UI()), "foo")
	err = ver.SetSave(labels.P("v1").UI(), nil)
	xNoErr(t, err)
	ver.Refresh()
	xJSONCheck(t, ver.Get(labels.P("v1").UI()), nil)

	dir.SetSave(labels.P("dd").UI(), "dd.foo")
	file.SetSave(labels.P("ff").UI(), "ff.bar")

	file.SetSave(labels.P("dd-ff").UI(), "dash")
	file.SetSave(labels.P("dd-ff-ff").UI(), "dashes")
	file.SetSave(labels.P("dd_ff").UI(), "under")
	file.SetSave(labels.P("dd.ff").UI(), "dot")

	ver2.Refresh() // very important since ver2 is not stale
	err = ver.SetSave(labels.P("vv").UI(), 987.234)
	if err == nil || err.Error() != `Attribute "labels.vv" must be a string` {
		t.Errorf("wrong err msg: %s", err)
		t.FailNow()
	}
	// ver.Refresh() // undo the change, otherwise next Set() will fail

	// Important test
	// We update v1(ver) after we created v2(ver2). At one point in time
	// this could cause both versions to be tagged as "default". Make sure
	// we don't have that situation. See comment above too
	err = ver.SetSave(labels.P("vv2").UI(), "v11")
	xNoErr(t, err)
	ver2.SetSave(labels.P("2nd").UI(), "3rd")

	xCheckGet(t, reg, "?inline", `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestSetLabels",
  "self": "http://localhost:8181/",
  "epoch": 1,
  "labels": {
    "r2": "123.234"
  },
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "epoch": 1,
      "labels": {
        "dd": "dd.foo"
      },
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "files": {
        "f1": {
          "fileid": "f1",
          "self": "http://localhost:8181/dirs/d1/files/f1$structure",
          "epoch": 1,
          "labels": {
            "2nd": "3rd",
            "dd-ff": "dash",
            "dd-ff-ff": "dashes",
            "dd.ff": "dot",
            "dd_ff": "under",
            "ff": "ff.bar"
          },
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",

          "defaultversionid": "v2",
          "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$structure",

          "versions": {
            "v1": {
              "fileid": "f1",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure",
              "epoch": 1,
              "labels": {
                "vv2": "v11"
              },
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z"
            },
            "v2": {
              "fileid": "f1",
              "versionid": "v2",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2$structure",
              "epoch": 1,
              "isdefault": true,
              "labels": {
                "2nd": "3rd",
                "dd-ff": "dash",
                "dd-ff-ff": "dashes",
                "dd.ff": "dot",
                "dd_ff": "under",
                "ff": "ff.bar"
              },
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z"
            }
          },
          "versionscount": 2,
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
        }
      },
      "filescount": 1,
      "filesurl": "http://localhost:8181/dirs/d1/files"
    }
  },
  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
}
`)

	file.SetDefault(ver)
	xCheckGet(t, reg, "?inline", `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestSetLabels",
  "self": "http://localhost:8181/",
  "epoch": 1,
  "labels": {
    "r2": "123.234"
  },
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "epoch": 1,
      "labels": {
        "dd": "dd.foo"
      },
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "files": {
        "f1": {
          "fileid": "f1",
          "self": "http://localhost:8181/dirs/d1/files/f1$structure",
          "epoch": 1,
          "labels": {
            "vv2": "v11"
          },
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",

          "defaultversionsticky": true,
          "defaultversionid": "v1",
          "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure",

          "versions": {
            "v1": {
              "fileid": "f1",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure",
              "epoch": 1,
              "isdefault": true,
              "labels": {
                "vv2": "v11"
              },
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z"
            },
            "v2": {
              "fileid": "f1",
              "versionid": "v2",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2$structure",
              "epoch": 1,
              "labels": {
                "2nd": "3rd",
                "dd-ff": "dash",
                "dd-ff-ff": "dashes",
                "dd.ff": "dot",
                "dd_ff": "under",
                "ff": "ff.bar"
              },
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z"
            }
          },
          "versionscount": 2,
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
        }
      },
      "filescount": 1,
      "filesurl": "http://localhost:8181/dirs/d1/files"
    }
  },
  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
}
`)
}
