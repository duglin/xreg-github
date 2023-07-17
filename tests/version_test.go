package tests

import (
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestCreateVersion(t *testing.T) {
	reg, err := registry.NewRegistry("TestCreateVersion")
	defer reg.Delete()
	xCheck(t, reg != nil && err == nil, "can't create reg")

	gm, _ := reg.AddGroupModel("dirs", "dir", "")
	gm.AddResourceModel("files", "file", 0, true, true)
	d1, _ := reg.AddGroup("dirs", "d1")

	f1, err := d1.AddResource("files", "f1", "v1")
	xNoErr(t, err)
	xCheck(t, f1 != nil, "Creating f1 failed")

	v2, err := f1.AddVersion("v2")
	xNoErr(t, err)
	xCheck(t, v2 != nil, "Creating v2 failed")

	vt, err := f1.AddVersion("v2")
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
	_, err = f2.AddVersion("v1.1")
	xNoErr(t, err)

	// /dirs/d1/f1/v1
	//            /v2
	//      /d2/f1/v1
	//      /d2/f1/v1.1

	// Check basic GET first
	xCheckGet(t, reg, "/dirs/d1/files/f1/versions/v1",
		`{
  "id": "v1",
  "self": "http:///dirs/d1/files/f1/versions/v1"
}
`)
	xCheckGet(t, reg, "/dirs/d1/files/f1/versions/xxx", "404: Not found\n")
	xCheckGet(t, reg, "dirs/d1/files/f1/versions/xxx", "404: Not found\n")
	xCheckGet(t, reg, "/dirs/d1/files/f1/versions/xxx/yyy", "404: Not found\n")
	xCheckGet(t, reg, "dirs/d1/files/f1/versions/xxx/yyy", "404: Not found\n")

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f1":{"versions":{"v1":{},"v1.1":{}}}}}}}`)

	vt, err = f1.FindVersion("v2")
	xNoErr(t, err)
	xCheck(t, vt != nil, "Didn't find v2")
	xJSONCheck(t, vt, v2)

	vt, err = f1.FindVersion("xxx")
	xNoErr(t, err)
	xCheck(t, vt == nil, "Find version xxx should have failed")

	err = v2.Delete()
	xNoErr(t, err)
	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{}}}}},"d2":{"files":{"f1":{"versions":{"v1":{},"v1.1":{}}}}}}}`)

	vt, err = f1.FindVersion("v2")
	xCheck(t, err == nil && vt == nil, "Finding delete version failed")

	// check that latest == v1 now
	// delete v1, check that f1 is deleted too
	err = f1.Refresh()
	xNoErr(t, err)

	xJSONCheck(t, f1.Get("latestId"), "v1")

	vt, err = f1.AddVersion("v2")
	xCheck(t, vt != nil && err == nil, "Adding v2 again")

	vt, err = f1.AddVersion("v3")
	xCheck(t, vt != nil && err == nil, "Added v3")
	xJSONCheck(t, f1.Get("latestId"), "v3")

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{},"v3":{}}}}},"d2":{"files":{"f1":{"versions":{"v1":{},"v1.1":{}}}}}}}`)
	xCheckGet(t, reg, "/dirs/d1/files/f1", `{
  "id": "f1",
  "self": "http:///dirs/d1/files/f1",
  "latestId": "v3",
  "latestUrl": "http:///dirs/d1/files/f1/versions/v3",

  "versionsCount": 3,
  "versionsUrl": "http:///dirs/d1/files/f1/versions"
}
`)
	vt, err = f1.FindVersion("v2")
	xNoErr(t, err)
	err = vt.Delete()
	xNoErr(t, err)
	xJSONCheck(t, f1.Get("latestId"), "v3")

	vt, err = f1.FindVersion("v3")
	xNoErr(t, err)
	err = vt.Delete()
	xNoErr(t, err)
	xJSONCheck(t, f1.Get("latestId"), "v1")

	vt, err = f1.FindVersion("v1")
	xNoErr(t, err)
	err = vt.Delete()
	xNoErr(t, err)
	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs":{"d1":{"files":{}},"d2":{"files":{"f1":{"versions":{"v1":{},"v1.1":{}}}}}}}`)
}
