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

	// check some dots in the prop names - and some tags stuff too
	dir, _ := reg.AddGroup("dirs", "d1")
	err := dir.Set("tags", "xxx")
	xCheck(t, err != nil, "tags=xxx should fail")

	// dots are ok as tag names
	err = dir.Set("tags.xxx.yyy", "xxx")
	xNoErr(t, err)
	err = dir.Set("tags.many.dots", "hello")
	xCheck(t, dir.Get("tags.many.dots") == "hello", "many.dots should work")
	dir.Refresh()
	xCheck(t, dir.Get("tags.many.dots") == "hello", "many.dots should work")
	xCheckGet(t, reg, "/dirs/d1", `{
  "id": "d1",
  "self": "http://localhost:8080/dirs/d1",
  "tags": {
    "many.dots": "hello",
    "xxx.yyy": "xxx"
  },

  "filesCount": 0,
  "filesUrl": "http://localhost:8080/dirs/d1/files"
}
`)

	err = dir.Set("tags", nil)
	xCheck(t, err != nil, "tags=nil should fail")
	xCheck(t, err != nil, "tags.xxx.yyy=nil should fail")
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

func TestSetTags(t *testing.T) {
	reg := NewRegistry("TestSetTags")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir", "")
	gm.AddResourceModel("files", "file", 0, true, true)

	dir, _ := reg.AddGroup("dirs", "d1")
	file, _ := dir.AddResource("files", "f1", "v1")
	ver, _ := file.FindVersion("v1")
	ver2, _ := file.AddVersion("v2")

	// /dirs/d1/f1/v1
	err := reg.Set("tags.r2", 123.234) // notice it's not a string
	xNoErr(t, err)
	reg.Refresh()
	xJSONCheck(t, reg.Get("tags.r2"), 123.234) // won't see it as a string until json
	err = reg.Set("tags.r1", "foo")
	xNoErr(t, err)
	reg.Refresh()
	xJSONCheck(t, reg.Get("tags.r1"), "foo")
	err = reg.Set("tags.r1", nil)
	xNoErr(t, err)
	reg.Refresh()
	xJSONCheck(t, reg.Get("tags.r1"), nil)

	err = dir.Set("tags.d1", "bar")
	xNoErr(t, err)
	dir.Refresh()
	xJSONCheck(t, dir.Get("tags.d1"), "bar")
	// test override
	err = dir.Set("tags.d1", "foo")
	xNoErr(t, err)
	dir.Refresh()
	xJSONCheck(t, dir.Get("tags.d1"), "foo")
	err = dir.Set("tags.d1", nil)
	xNoErr(t, err)
	dir.Refresh()
	xJSONCheck(t, dir.Get("tags.d1"), nil)

	err = file.Set("tags.f1", "foo")
	xNoErr(t, err)
	file.Refresh()
	xJSONCheck(t, file.Get("tags.f1"), "foo")
	err = file.Set("tags.f1", nil)
	xNoErr(t, err)
	file.Refresh()
	xJSONCheck(t, file.Get("tags.f1"), nil)

	err = ver.Set("tags.v1", "foo")
	xNoErr(t, err)
	ver.Refresh()
	xJSONCheck(t, ver.Get("tags.v1"), "foo")
	err = ver.Set("tags.v1", nil)
	xNoErr(t, err)
	ver.Refresh()
	xJSONCheck(t, ver.Get("tags.v1"), nil)

	dir.Set("tags.dd", "dd.foo")
	file.Set("tags.ff", "ff.bar")
	ver.Set("tags.vv", 987.234)
	ver.Set("tags.vv2", "v11")
	ver2.Set("tags.2nd", "3rd")

	xCheckGet(t, reg, "?inline", `{
  "id": "TestSetTags",
  "self": "http://localhost:8080/",
  "tags": {
    "r2": "123.234"
  },

  "dirs": {
    "d1": {
      "id": "d1",
      "self": "http://localhost:8080/dirs/d1",
      "tags": {
        "dd": "dd.foo"
      },

      "files": {
        "f1": {
          "id": "f1",
          "self": "http://localhost:8080/dirs/d1/files/f1",
          "latestId": "v2",
          "latestUrl": "http://localhost:8080/dirs/d1/files/f1/versions/v2",
          "tags": {
            "2nd": "3rd",
            "ff": "ff.bar"
          },

          "versions": {
            "v1": {
              "id": "v1",
              "self": "http://localhost:8080/dirs/d1/files/f1/versions/v1",
              "tags": {
                "vv": "987.234",
                "vv2": "v11"
              }
            },
            "v2": {
              "id": "v2",
              "self": "http://localhost:8080/dirs/d1/files/f1/versions/v2",
              "tags": {
                "2nd": "3rd",
                "ff": "ff.bar"
              }
            }
          },
          "versionsCount": 2,
          "versionsUrl": "http://localhost:8080/dirs/d1/files/f1/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http://localhost:8080/dirs/d1/files"
    }
  },
  "dirsCount": 1,
  "dirsUrl": "http://localhost:8080/dirs"
}
`)

	file.Set("latestId", ver.UID)
	xCheckGet(t, reg, "?inline", `{
  "id": "TestSetTags",
  "self": "http://localhost:8080/",
  "tags": {
    "r2": "123.234"
  },

  "dirs": {
    "d1": {
      "id": "d1",
      "self": "http://localhost:8080/dirs/d1",
      "tags": {
        "dd": "dd.foo"
      },

      "files": {
        "f1": {
          "id": "f1",
          "self": "http://localhost:8080/dirs/d1/files/f1",
          "latestId": "v1",
          "latestUrl": "http://localhost:8080/dirs/d1/files/f1/versions/v1",
          "tags": {
            "vv": "987.234",
            "vv2": "v11"
          },

          "versions": {
            "v1": {
              "id": "v1",
              "self": "http://localhost:8080/dirs/d1/files/f1/versions/v1",
              "tags": {
                "vv": "987.234",
                "vv2": "v11"
              }
            },
            "v2": {
              "id": "v2",
              "self": "http://localhost:8080/dirs/d1/files/f1/versions/v2",
              "tags": {
                "2nd": "3rd",
                "ff": "ff.bar"
              }
            }
          },
          "versionsCount": 2,
          "versionsUrl": "http://localhost:8080/dirs/d1/files/f1/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http://localhost:8080/dirs/d1/files"
    }
  },
  "dirsCount": 1,
  "dirsUrl": "http://localhost:8080/dirs"
}
`)
}
