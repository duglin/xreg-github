package tests

import (
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestCreateResource(t *testing.T) {
	reg, err := registry.NewRegistry("TestCreateResource")
	defer reg.Delete()
	xCheck(t, reg != nil && err == nil, "can't create reg")

	gm, _ := reg.AddGroupModel("dirs", "dir", "")
	gm.AddResourceModel("files", "file", 0, true, true)
	d1, _ := reg.AddGroup("dirs", "d1")

	f1, err := d1.AddResource("files", "f1", "v1")
	xNoErr(t, err)
	xCheck(t, f1 != nil && err == nil, "Creating f1 failed")

	ft, err := d1.AddResource("files", "f1", "v1")
	xCheck(t, ft == nil && err != nil, "Dup f1 should have failed")

	v2, err := f1.AddVersion("v2")
	xNoErr(t, err)
	xCheck(t, v2 != nil && err == nil, "Creating v2 failed")

	vt, err := f1.AddVersion("v2")
	xCheck(t, vt == nil && err != nil, "Dup v2 should have faile")

	d2, err := reg.AddGroup("dirs", "d2")
	xNoErr(t, err)
	xCheck(t, d2 != nil && err == nil, "Creating d2 failed")

	f2, _ := d2.AddResource("files", "f2", "v1")
	f2.AddVersion("v1.1")

	// /dirs/d1/f1/v1
	//            /v2
	//      /d2/f2/v1

	// Check basic GET first
	xCheckGet(t, reg, "/dirs/d1/files/f1",
		`{
  "id": "f1",
  "self": "http:///dirs/d1/files/f1",
  "latestId": "v2",
  "latestUrl": "http:///dirs/d1/files/f1/versions/v2",

  "versionsCount": 2,
  "versionsUrl": "http:///dirs/d1/files/f1/versions"
}
`)
	xCheckGet(t, reg, "/dirs/d1/files/xxx", "404: Not found\n")
	xCheckGet(t, reg, "dirs/d1/files/xxx", "404: Not found\n")
	xCheckGet(t, reg, "/dirs/d1/files/xxx/yyy", `Expected "versions", got: "yyy"`)
	xCheckGet(t, reg, "dirs/d1/files/xxx/yyy", `Expected "versions", got: "yyy"`)

	ft, err = d1.FindResource("files", "f1")
	xNoErr(t, err)
	xCheck(t, ft != nil && err == nil, "Finding f1 failed")
	xJSONCheck(t, ft, f1)

	ft, err = d1.FindResource("files", "xxx")
	xCheck(t, ft == nil && err == nil, "Find files/xxx should have failed")

	ft, err = d1.FindResource("xxx", "xxx")
	xCheck(t, ft == nil && err == nil, "Find xxx/xxx should have failed")

	ft, err = d1.FindResource("xxx", "f1")
	xCheck(t, ft == nil && err == nil, "Find xxx/f1 should have failed")

	err = f1.Delete()
	xNoErr(t, err)

	ft, err = d1.FindResource("files", "f1")
	xCheck(t, err == nil && ft == nil, "Finding delete resource failed")
}

func TestResourceSet(t *testing.T) {
	reg, err := registry.NewRegistry("TestResourceSet")
	defer reg.Delete()
	xCheck(t, reg != nil && err == nil, "can't create reg")

	gm, _ := reg.AddGroupModel("dirs", "dir", "")
	gm.AddResourceModel("files", "file", 0, true, true)
	d1, _ := reg.AddGroup("dirs", "d1")
	f1, _ := d1.AddResource("files", "f1", "v1")

	// /dirs/d1/f1/v1

	f1.Set("name", "myName")
	f1.Set("epoch", 68)
	f1.Set("ext1", "someext")
	f1.Set("ext2", 123)

	// Make sure the props on the resource weren't set
	xCheck(t, f1.Get(".name") == nil, ".name should be nil")
	xCheck(t, f1.Get(".epoch") == nil, ".epoch should be nil")
	xCheck(t, f1.Get(".ext1") == nil, ".ext1 should be nil")
	xCheck(t, f1.Get(".ext2") == nil, ".ext2 should be nil")

	ft, _ := d1.FindResource("files", "f1")

	xJSONCheck(t, ft, f1)

	// Make sure the version was set
	vt, _ := ft.GetLatest()
	xJSONCheck(t, vt.Get("name"), "myName")
	xJSONCheck(t, vt.Get("epoch"), 68)
	xJSONCheck(t, vt.Get("ext1"), "someext")
	xJSONCheck(t, vt.Get("ext2"), 123)
}
