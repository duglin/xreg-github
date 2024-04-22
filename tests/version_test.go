package tests

import (
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestCreateVersion(t *testing.T) {
	reg := NewRegistry("TestCreateVersion")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)
	d1, _ := reg.AddGroup("dirs", "d1")

	f1, err := d1.AddResource("files", "f1", "v1")
	xNoErr(t, err)
	xCheck(t, f1 != nil, "Creating f1 failed")

	v2, err := f1.AddVersion("v2")
	xNoErr(t, err)
	xCheck(t, v2 != nil, "Creating v2 failed")

	vt, err := f1.AddVersion("v2")
	xCheck(t, vt == nil && err != nil, "Dup v2 should have faile")

	l, err := f1.GetDefault()
	xNoErr(t, err)
	xJSONCheck(t, l, v2)

	d2, err := reg.AddGroup("dirs", "d2")
	xNoErr(t, err)
	xCheck(t, d2 != nil && err == nil, "Creating d2 failed")

	f2, err := d2.AddResource("files", "f1", "v1")
	xNoErr(t, err)
	xCheck(t, f2 != nil, "Creating d2/f1/v1 failed")
	_, err = f2.AddVersion("v1.1")
	xNoErr(t, err)

	// /dirs/d1/f1/v1
	//            /v2
	//      /d2/f1/v1
	//      /d2/f1/v1.1

	// Check basic GET first
	xCheckGet(t, reg, "/dirs/d1/files/f1/versions/v1?meta",
		`{
  "id": "v1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1?meta"
}
`)
	xCheckGet(t, reg, "/dirs/d1/files/f1/versions/xxx", "Not found\n")
	xCheckGet(t, reg, "dirs/d1/files/f1/versions/xxx", "Not found\n")
	xCheckGet(t, reg, "/dirs/d1/files/f1/versions/xxx/yyy", "Not found\n")
	xCheckGet(t, reg, "dirs/d1/files/f1/versions/xxx/yyy", "Not found\n")

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f1":{"versions":{"v1":{},"v1.1":{}}}}}}}`)

	vt, err = f1.FindVersion("v2")
	xNoErr(t, err)
	xCheck(t, vt != nil, "Didn't find v2")
	xJSONCheck(t, vt, v2)

	vt, err = f1.FindVersion("xxx")
	xNoErr(t, err)
	xCheck(t, vt == nil, "Find version xxx should have failed")

	err = v2.Delete("")
	xNoErr(t, err)
	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{}}}}},"d2":{"files":{"f1":{"versions":{"v1":{},"v1.1":{}}}}}}}`)

	vt, err = f1.FindVersion("v2")
	xCheck(t, err == nil && vt == nil, "Finding delete version failed")

	// check that default == v1 now
	// delete v1, check that f1 is deleted too
	err = f1.Refresh()
	xNoErr(t, err)

	xJSONCheck(t, f1.Get("defaultversionid"), "v1")

	vt, err = f1.AddVersion("v2")
	xCheck(t, vt != nil && err == nil, "Adding v2 again")

	vt, err = f1.AddVersion("v3")
	xCheck(t, vt != nil && err == nil, "Added v3")
	xNoErr(t, vt.SetDefault())
	xJSONCheck(t, f1.Get("defaultversionid"), "v3")

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{},"v3":{}}}}},"d2":{"files":{"f1":{"versions":{"v1":{},"v1.1":{}}}}}}}`)
	xCheckGet(t, reg, "/dirs/d1/files/f1?meta", `{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "stickydefaultversion": true,
  "defaultversionid": "v3",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3?meta",

  "versionscount": 3,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)
	vt, err = f1.FindVersion("v2")
	xNoErr(t, err)
	err = vt.Delete("")
	xNoErr(t, err)
	xJSONCheck(t, f1.Get("defaultversionid"), "v3")

	vt, err = f1.FindVersion("v3")
	xNoErr(t, err)
	xCheck(t, vt != nil, "Can't be nil")
	err = vt.Delete("")
	xNoErr(t, err)
	xJSONCheck(t, f1.Get("defaultversionid"), "v1")

	f1, err = d2.FindResource("files", "f1")
	xNoErr(t, err)
	xNoErr(t, f1.SetDefault(v2))
	_, err = f1.AddVersion("v3")
	xNoErr(t, err)
	vt, err = f1.FindVersion("v1")
	xNoErr(t, err)
	xCheck(t, vt != nil, "should not be nil")
	err = vt.Delete("")
	xNoErr(t, err)
	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{}}}}},"d2":{"files":{"f1":{"versions":{"v1.1":{},"v3":{}}}}}}}`)

	err = vt.Delete("v2")
	xCheckErr(t, err, `Can't find next default Version "v2"`)

	vt, err = f1.FindVersion("v1.1")
	xNoErr(t, err)
	xCheck(t, vt != nil, "should not be nil")

	err = vt.Delete("v1.1")
	xCheckErr(t, err, `Can't set defaultversionid to Version being deleted`)

	vt, err = f1.AddVersion("v4")
	xNoErr(t, err)

	err = vt.Delete("v3")
	xNoErr(t, err)

	xCheckGet(t, reg, "dirs/d2/files",
		`{
  "f1": {
    "id": "f1",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d2/files/f1?meta",
    "stickydefaultversion": true,
    "defaultversionid": "v3",
    "defaultversionurl": "http://localhost:8181/dirs/d2/files/f1/versions/v3?meta",

    "versionscount": 2,
    "versionsurl": "http://localhost:8181/dirs/d2/files/f1/versions"
  }
}
`)
}

func TestDefaultVersion(t *testing.T) {
	reg := NewRegistry("TestDefaultVersion")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	d1, _ := reg.AddGroup("dirs", "d1")
	f1, _ := d1.AddResource("files", "f1", "v1")
	v1, _ := f1.FindVersion("v1")
	v2, _ := f1.AddVersion("v2")

	xCheckGet(t, reg, "dirs/d1/files/f1?meta",
		`{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2?meta",

  "versionscount": 2,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	// Doesn't change much, but does make it sticky
	xNoErr(t, f1.SetDefault(v2))

	xCheckGet(t, reg, "dirs/d1/files/f1?meta",
		`{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "stickydefaultversion": true,
  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2?meta",

  "versionscount": 2,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	v3, _ := f1.AddVersion("v3")

	xCheckGet(t, reg, "dirs/d1/files/f1?meta",
		`{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "stickydefaultversion": true,
  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2?meta",

  "versionscount": 3,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	// Now unstick it and it default should be v3 now
	xNoErr(t, f1.SetDefault(nil))
	xCheckGet(t, reg, "dirs/d1/files/f1?meta",
		`{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "v3",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3?meta",

  "versionscount": 3,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	v4, _ := f1.AddVersion("v4")
	xNoErr(t, f1.SetDefault(v4))
	v5, _ := f1.AddVersion("v5")

	xCheckGet(t, reg, "dirs/d1/files/f1?meta",
		`{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "stickydefaultversion": true,
  "defaultversionid": "v4",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v4?meta",

  "versionscount": 5,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	err := v1.Delete("")
	xNoErr(t, err)
	xCheckGet(t, reg, "dirs/d1/files/f1?meta",
		`{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "stickydefaultversion": true,
  "defaultversionid": "v4",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v4?meta",

  "versionscount": 4,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	err = v3.Delete("v1")
	xCheckErr(t, err, "Can't find next default Version \"v1\"")
	err = v3.Delete("v2")
	xNoErr(t, err)
	xCheckGet(t, reg, "dirs/d1/files/f1?meta",
		`{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "stickydefaultversion": true,
  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2?meta",

  "versionscount": 3,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	err = v2.Delete("")
	xNoErr(t, err)
	xCheckGet(t, reg, "dirs/d1/files/f1?meta",
		`{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "v5",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v5?meta",

  "versionscount": 2,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xNoErr(t, v4.Delete(""))
	xCheckGet(t, reg, "dirs/d1/files/f1?meta",
		`{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "v5",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v5?meta",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xNoErr(t, v5.Delete(""))
	xCheckGet(t, reg, "dirs/d1/files/f1?meta", "Not found\n")
}

func TestDefaultVersionMaxVersions(t *testing.T) {
	reg := NewRegistry("TestDefaultVersionMaxVersions")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 3, true, true, true)

	d1, _ := reg.AddGroup("dirs", "d1")
	f1, _ := d1.AddResource("files", "f1", "v1")
	f1.FindVersion("v1")
	f1.AddVersion("v2")
	f1.AddVersion("v3")

	xCheckGet(t, reg, "dirs/d1/files/f1?meta",
		`{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "v3",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3?meta",

  "versionscount": 3,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	v4, _ := f1.AddVersion("v4")

	xCheckGet(t, reg, "dirs/d1/files/f1?meta",
		`{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "v4",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v4?meta",

  "versionscount": 3,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xNoErr(t, f1.SetDefault(v4))
	f1.AddVersion("v5")
	// check def = v4
	f1.AddVersion("v6")
	f1.AddVersion("v7")
	f1.AddVersion("v8")
	// check def = v4    v8, v7, v4

	xCheckGet(t, reg, "dirs/d1/files/f1?meta&inline=versions",
		`{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "stickydefaultversion": true,
  "defaultversionid": "v4",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v4?meta",

  "versions": {
    "v4": {
      "id": "v4",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v4?meta",
      "isdefault": true
    },
    "v7": {
      "id": "v7",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v7?meta"
    },
    "v8": {
      "id": "v8",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v8?meta"
    }
  },
  "versionscount": 3,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

}

func TestVersionRequiredFields(t *testing.T) {
	reg := NewRegistry("TestVersionRequiredFields")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, true)
	_, err := rm.AddAttribute(&registry.Attribute{
		Name:           "clireq",
		Type:           registry.STRING,
		ClientRequired: true,
		ServerRequired: true,
	})
	xNoErr(t, err)

	group, err := reg.AddGroup("dirs", "d1")
	xNoErr(t, err)

	f1, err := group.AddResourceWithObject("files", "f1", "v1",
		registry.Object{"clireq": "test"})
	xNoErr(t, err)
	reg.Commit()

	_, err = f1.AddVersion("v2")
	xCheckErr(t, err, "Required property \"clireq\" is missing")
	reg.Rollback()

	v1, err := f1.AddVersionWithObject("v2", registry.Object{"clireq": "test"})
	xNoErr(t, err)
	reg.Commit()

	err = v1.SetSave("clireq", nil)
	xCheckErr(t, err, "Required property \"clireq\" is missing")

	err = v1.SetSave("clireq", "again")
	xNoErr(t, err)
}
