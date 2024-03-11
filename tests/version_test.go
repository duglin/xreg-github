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

	v2, err := f1.AddVersion("v2", true)
	xNoErr(t, err)
	xCheck(t, v2 != nil, "Creating v2 failed")

	vt, err := f1.AddVersion("v2", true)
	xCheck(t, vt == nil && err != nil, "Dup v2 should have faile")

	l, err := f1.GetLatest()
	xNoErr(t, err)
	xJSONCheck(t, l, v2)

	d2, err := reg.AddGroup("dirs", "d2")
	xNoErr(t, err)
	xCheck(t, d2 != nil && err == nil, "Creating d2 failed")

	f2, err := d2.AddResource("files", "f1", "v1")
	xNoErr(t, err)
	xCheck(t, f2 != nil, "Creating d2/f1/v1 failed")
	_, err = f2.AddVersion("v1.1", true)
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

	// check that latest == v1 now
	// delete v1, check that f1 is deleted too
	err = f1.Refresh()
	xNoErr(t, err)

	xJSONCheck(t, f1.Get("latestversionid"), "v1")

	vt, err = f1.AddVersion("v2", true)
	xCheck(t, vt != nil && err == nil, "Adding v2 again")

	vt, err = f1.AddVersion("v3", true)
	xCheck(t, vt != nil && err == nil, "Added v3")
	xJSONCheck(t, f1.Get("latestversionid"), "v3")

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{},"v3":{}}}}},"d2":{"files":{"f1":{"versions":{"v1":{},"v1.1":{}}}}}}}`)
	xCheckGet(t, reg, "/dirs/d1/files/f1?meta", `{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "latestversionid": "v3",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3?meta",

  "versionscount": 3,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)
	vt, err = f1.FindVersion("v2")
	xNoErr(t, err)
	err = vt.Delete("")
	xNoErr(t, err)
	xJSONCheck(t, f1.Get("latestversionid"), "v3")

	vt, err = f1.FindVersion("v3")
	xNoErr(t, err)
	xCheck(t, vt != nil, "Can't be nil")
	err = vt.Delete("")
	xNoErr(t, err)
	xJSONCheck(t, f1.Get("latestversionid"), "v1")

	f1, err = d2.FindResource("files", "f1")
	xNoErr(t, err)
	_, err = f1.AddVersion("v3", false)
	xNoErr(t, err)
	vt, err = f1.FindVersion("v1")
	xNoErr(t, err)
	xCheck(t, vt != nil, "should not be nil")
	err = vt.Delete("")
	xNoErr(t, err)
	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{}}}}},"d2":{"files":{"f1":{"versions":{"v1.1":{},"v3":{}}}}}}}`)

	err = vt.Delete("v2")
	xCheckErr(t, err, `Can't find next latest Version "v2"`)

	vt, err = f1.FindVersion("v1.1")
	xNoErr(t, err)
	xCheck(t, vt != nil, "should not be nil")

	err = vt.Delete("v1.1")
	xCheckErr(t, err, `Can't set latestversionid to Version being deleted`)

	vt, err = f1.AddVersion("v4", true)
	xNoErr(t, err)

	err = vt.Delete("v3")
	xNoErr(t, err)

	xCheckGet(t, reg, "dirs/d2/files",
		`{
  "f1": {
    "id": "f1",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d2/files/f1?meta",
    "latestversionid": "v3",
    "latestversionurl": "http://localhost:8181/dirs/d2/files/f1/versions/v3?meta",

    "versionscount": 2,
    "versionsurl": "http://localhost:8181/dirs/d2/files/f1/versions"
  }
}
`)
}

func TestLatestVersion(t *testing.T) {
	reg := NewRegistry("TestLatestVersion")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	d1, _ := reg.AddGroup("dirs", "d1")
	f1, _ := d1.AddResource("files", "f1", "v1")
	v1, _ := f1.FindVersion("v1")
	v2, _ := f1.AddVersion("v2", true)
	v3, _ := f1.AddVersion("v3", false)
	v4, _ := f1.AddVersion("v4", true)
	v5, _ := f1.AddVersion("v5", false)

	xCheckGet(t, reg, "dirs/d1/files/f1?meta",
		`{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "latestversionid": "v4",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v4?meta",

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
  "latestversionid": "v4",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v4?meta",

  "versionscount": 4,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	err = v3.Delete("v1")
	xCheckErr(t, err, "Can't find next latest Version \"v1\"")
	err = v3.Delete("v2")
	xNoErr(t, err)
	xCheckGet(t, reg, "dirs/d1/files/f1?meta",
		`{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "latestversionid": "v2",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2?meta",

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
  "latestversionid": "v5",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v5?meta",

  "versionscount": 2,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xNoErr(t, v4.Delete(""))
	xNoErr(t, v5.Delete(""))
	xCheckGet(t, reg, "dirs/d1/files/f1?meta", "Not found\n")
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

	f1, err := group.AddResource("files", "f1", "v1",
		registry.Object{"clireq": "test"})
	xNoErr(t, err)
	reg.Commit()

	_, err = f1.AddVersion("v2", true)
	xCheckErr(t, err, "Required property \"clireq\" is missing")
	reg.Rollback()

	v1, err := f1.AddVersion("v2", true, registry.Object{"clireq": "test"})
	xNoErr(t, err)
	reg.Commit()

	err = v1.Set("clireq", nil)
	xCheckErr(t, err, "Required property \"clireq\" is missing")

	err = v1.Set("clireq", "again")
	xNoErr(t, err)
}
