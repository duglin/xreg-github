package tests

import (
	"testing"
)

func TestSetResource(t *testing.T) {
	reg := NewRegistry("TestSetResource")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir", "")
	gm.AddResourceModel("files", "file", 0, true, true)

	dir, _ := reg.AddGroup("dirs", "d1")
	file, _ := dir.AddResource("files", "f1", "v1")

	// /dirs/d1/f1/v1

	// Make sure setting it on the version is seen by res.Latest and res
	file.Set("name", "myName")
	ver, _ := file.FindVersion("v1")
	val := ver.Get("name")
	if val != "myName" {
		t.Errorf("ver.Name is %q, should be 'myName'", val)
	}

	name := file.Get("name").(string)
	xCheckEqual(t, "", name, "myName")

	// Verify that nil and "" are treated differently
	ver.Set("name", nil)
	ver2, _ := file.FindVersion(ver.UID)
	xJSONCheck(t, ver2, ver)
	val = ver.Get("name")
	xCheck(t, val == nil, "Setting to nil should return nil")

	ver.Set("name", "")
	ver2, _ = file.FindVersion(ver.UID)
	xJSONCheck(t, ver2, ver)
	val = ver.Get("name")
	xCheck(t, val == "", "Setting to '' should return ''")
}

func TestSetVersion(t *testing.T) {
	reg := NewRegistry("TestSetVersion")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir", "")
	gm.AddResourceModel("files", "file", 0, true, true)

	dir, _ := reg.AddGroup("dirs", "d1")
	file, _ := dir.AddResource("files", "f1", "v1")
	ver, _ := file.FindVersion("v1")

	// /dirs/d1/f1/v1

	// Make sure setting it on the version is seen by res.Latest and res
	ver.Set("name", "myName")
	file, _ = dir.FindResource("files", "f1")
	l, err := file.GetLatest()
	xNoErr(t, err)
	xCheck(t, l != nil, "latest is nil")
	val := l.Get("name")
	if val != "myName" {
		t.Errorf("resource.latest.Name is %q, should be 'myName'", val)
	}
	val = file.Get("name")
	if val != "myName" {
		t.Errorf("resource.Name is %q, should be 'myName'", val)
	}

	// Make sure we can also still see it from the version itself
	ver, _ = file.FindVersion("v1")
	val = ver.Get("name")
	if val != "myName" {
		t.Errorf("version.Name is %q, should be 'myName'", val)
	}
}

func TestSetDots(t *testing.T) {
	reg := NewRegistry("TestSetDots")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir", "")
	gm.AddResourceModel("files", "file", 0, true, true)

	// check some dots in the prop names - and some labels stuff too
	dir, _ := reg.AddGroup("dirs", "d1")
	err := dir.Set("labels", "xxx")
	xCheck(t, err != nil, "labels=xxx should fail")

	// dots are ok as tag names
	err = dir.Set("labels.xxx.yyy", "xxx")
	xNoErr(t, err)
	err = dir.Set("labels.many.dots", "hello")
	xCheck(t, dir.Get("labels.many.dots") == "hello", "many.dots should work")
	dir.Refresh()
	xCheck(t, dir.Get("labels.many.dots") == "hello", "many.dots should work")
	xCheckGet(t, reg, "/dirs/d1", `{
  "id": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "labels": {
    "many.dots": "hello",
    "xxx.yyy": "xxx"
  },

  "filesCount": 0,
  "filesUrl": "http://localhost:8181/dirs/d1/files"
}
`)

	err = dir.Set("labels", nil)
	xCheck(t, err != nil, "labels=nil should fail")
	xCheck(t, err != nil, "labels.xxx.yyy=nil should fail")
	err = dir.Set("xxx.yyy", "xxx")
	xCheck(t, err != nil, "xxx.yyy=xxx should fail")
	err = dir.Set("xxx.yyy", nil)
	xCheck(t, err != nil, "xxx.yyy=xxx should fail")
	err = dir.Set("xxx.", "xxx")
	xCheck(t, err != nil, "xxx.yyy=xxx should fail")
	err = dir.Set(".xxx", "xxx")
	xCheck(t, err != nil, "xxx.yyy=xxx should fail")
	err = dir.Set(".xxx.", "xxx")
	xCheck(t, err != nil, "xxx.yyy=xxx should fail")
}

func TestSetLabels(t *testing.T) {
	reg := NewRegistry("TestSetLabels")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir", "")
	gm.AddResourceModel("files", "file", 0, true, true)

	dir, _ := reg.AddGroup("dirs", "d1")
	file, _ := dir.AddResource("files", "f1", "v1")
	ver, _ := file.FindVersion("v1")
	ver2, _ := file.AddVersion("v2")

	// /dirs/d1/f1/v1
	err := reg.Set("labels.r2", 123.234) // notice it's not a string
	xNoErr(t, err)
	reg.Refresh()
	xJSONCheck(t, reg.Get("labels.r2"), 123.234) // won't see it as a string until json
	err = reg.Set("labels.r1", "foo")
	xNoErr(t, err)
	reg.Refresh()
	xJSONCheck(t, reg.Get("labels.r1"), "foo")
	err = reg.Set("labels.r1", nil)
	xNoErr(t, err)
	reg.Refresh()
	xJSONCheck(t, reg.Get("labels.r1"), nil)

	err = dir.Set("labels.d1", "bar")
	xNoErr(t, err)
	dir.Refresh()
	xJSONCheck(t, dir.Get("labels.d1"), "bar")
	// test override
	err = dir.Set("labels.d1", "foo")
	xNoErr(t, err)
	dir.Refresh()
	xJSONCheck(t, dir.Get("labels.d1"), "foo")
	err = dir.Set("labels.d1", nil)
	xNoErr(t, err)
	dir.Refresh()
	xJSONCheck(t, dir.Get("labels.d1"), nil)

	err = file.Set("labels.f1", "foo")
	xNoErr(t, err)
	file.Refresh()
	xJSONCheck(t, file.Get("labels.f1"), "foo")
	err = file.Set("labels.f1", nil)
	xNoErr(t, err)
	file.Refresh()
	xJSONCheck(t, file.Get("labels.f1"), nil)

	err = ver.Set("labels.v1", "foo")
	xNoErr(t, err)
	ver.Refresh()
	xJSONCheck(t, ver.Get("labels.v1"), "foo")
	err = ver.Set("labels.v1", nil)
	xNoErr(t, err)
	ver.Refresh()
	xJSONCheck(t, ver.Get("labels.v1"), nil)

	dir.Set("labels.dd", "dd.foo")
	file.Set("labels.ff", "ff.bar")
	ver.Set("labels.vv", 987.234)
	ver.Set("labels.vv2", "v11")
	ver2.Set("labels.2nd", "3rd")

	xCheckGet(t, reg, "?inline", `{
  "id": "TestSetLabels",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "labels": {
    "r2": "123.234"
  },

  "dirs": {
    "d1": {
      "id": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "labels": {
        "dd": "dd.foo"
      },

      "files": {
        "f1": {
          "id": "f1",
          "self": "http://localhost:8181/dirs/d1/files/f1",
          "latestId": "v2",
          "latestUrl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
          "labels": {
            "2nd": "3rd",
            "ff": "ff.bar"
          },

          "versions": {
            "v1": {
              "id": "v1",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
              "labels": {
                "vv": "987.234",
                "vv2": "v11"
              }
            },
            "v2": {
              "id": "v2",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
              "latest": true,
              "labels": {
                "2nd": "3rd",
                "ff": "ff.bar"
              }
            }
          },
          "versionsCount": 2,
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

	file.SetLatest(ver)
	xCheckGet(t, reg, "?inline", `{
  "id": "TestSetLabels",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "labels": {
    "r2": "123.234"
  },

  "dirs": {
    "d1": {
      "id": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "labels": {
        "dd": "dd.foo"
      },

      "files": {
        "f1": {
          "id": "f1",
          "self": "http://localhost:8181/dirs/d1/files/f1",
          "latestId": "v1",
          "latestUrl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
          "labels": {
            "vv": "987.234",
            "vv2": "v11"
          },

          "versions": {
            "v1": {
              "id": "v1",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
              "latest": true,
              "labels": {
                "vv": "987.234",
                "vv2": "v11"
              }
            },
            "v2": {
              "id": "v2",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
              "labels": {
                "2nd": "3rd",
                "ff": "ff.bar"
              }
            }
          },
          "versionsCount": 2,
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
