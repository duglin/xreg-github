package tests

import (
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestCreateResource(t *testing.T) {
	reg := NewRegistry("TestCreateResource")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)
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
	xCheckGet(t, reg, "/dirs/d1/files/f1?meta",
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
	xCheckGet(t, reg, "/dirs/d1/files/xxx", "Not found\n")
	xCheckGet(t, reg, "dirs/d1/files/xxx", "Not found\n")
	xCheckGet(t, reg, "/dirs/d1/files/xxx/yyy", "Expected \"versions\", got: yyy\n")
	xCheckGet(t, reg, "dirs/d1/files/xxx/yyy", "Expected \"versions\", got: yyy\n")

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
	reg := NewRegistry("TestResourceSet")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, true)
	rm.AddAttr("ext1", registry.STRING)
	rm.AddAttr("ext2", registry.INTEGER)

	d1, _ := reg.AddGroup("dirs", "d1")
	f1, _ := d1.AddResource("files", "f1", "v1")

	// /dirs/d1/f1/v1

	xNoErr(t, f1.SetSave("name", "myName"))
	xNoErr(t, f1.SetSave("epoch", 68))
	xNoErr(t, f1.SetSave("ext1", "someext"))
	xNoErr(t, f1.SetSave("ext2", 123))

	// Make sure the props on the resource weren't set
	xCheck(t, f1.Entity.Get("name") == nil, "name should be nil")
	xCheck(t, f1.Entity.Get("epoch") == 1, "epoch should be 1")
	xCheck(t, f1.Entity.Get("ext1") == nil, "ext1 should be nil")
	xCheck(t, f1.Entity.Get("ext2") == nil, "ext2 should be nil")

	ft, _ := d1.FindResource("files", "f1")

	xJSONCheck(t, ft, f1)

	// Make sure the version was set
	vt, _ := ft.GetDefault()
	xJSONCheck(t, vt.Get("name"), "myName")
	xJSONCheck(t, vt.Get("epoch"), 68)
	xJSONCheck(t, vt.Get("ext1"), "someext")
	xJSONCheck(t, vt.Get("ext2"), 123)
}

func TestResourceRequiredFields(t *testing.T) {
	reg := NewRegistry("TestResourceRequiredFields")
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
	reg.Commit()

	_, err = group.AddResource("files", "f1", "v1")
	xCheckErr(t, err, "Required property \"clireq\" is missing")
	reg.Rollback()

	f1, err := group.AddResource("files", "f1", "v1",
		registry.Object{"clireq": "test"})
	xNoErr(t, err)
	reg.Commit()

	err = f1.SetSave("clireq", nil)
	xCheckErr(t, err, "Required property \"clireq\" is missing")

	err = f1.SetSave("clireq", "again")
	xNoErr(t, err)
}

func TestResourceMaxVersions(t *testing.T) {
	reg := NewRegistry("TestResourceMaxVersions")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)
	d1, _ := reg.AddGroup("dirs", "d1")

	_, err = gm.AddResourceModelFull(&registry.ResourceModel{
		Plural:      "files",
		Singular:    "file",
		MaxVersions: -1,
	})
	xCheckErr(t, err, `'versions'(-1) must be >= 0`)

	rm, err := gm.AddResourceModelFull(&registry.ResourceModel{
		Plural:      "files",
		Singular:    "file",
		MaxVersions: 1, // ONLY ALLOW 1 VERSION
	})
	xNoErr(t, err)

	f1, err := d1.AddResource("files", "f1", "v1")
	xCheck(t, f1 != nil && err == nil, "Creating f1 failed: %s", err)
	vers, err := f1.GetVersions()
	xNoErr(t, err)
	xCheck(t, len(vers) == 1, "Should be just one version")

	defaultV, err := f1.GetDefault()
	xCheck(t, defaultV != nil && err == nil && defaultV.UID == "v1",
		"err: %q default: %s", err, ToJSON(defaultV))

	// Create v2 and bump v1 out of the list
	v2, err := f1.AddVersion("v2")
	xCheck(t, v2 != nil && err == nil, "Creating v2 failed: %s", err)
	defaultV, err = f1.GetDefault()
	xCheck(t, defaultV != nil && err == nil && defaultV.UID == "v2",
		"err: %q default: %s", err, ToJSON(defaultV))
	vers, err = f1.GetVersions()
	xNoErr(t, err)
	xCheck(t, len(vers) == 1 && vers[0].Object["id"] == "v2", "Should be v2")

	err = rm.SetMaxVersions(2)
	xNoErr(t, err)

	// Create v3, but keep v2 as default
	xNoErr(t, f1.SetDefault(v2))
	v3, err := f1.AddVersion("v3")
	xCheck(t, v3 != nil && err == nil, "Creating v3 failed: %s", err)
	defaultV, err = f1.GetDefault()
	xCheck(t, defaultV != nil && err == nil && defaultV.UID == "v2",
		"err: %q defaultV: %s", err, ToJSON(defaultV))
	vers, err = f1.GetVersions()
	xNoErr(t, err)
	xCheck(t, len(vers) == 2, "Should be 2")
	xCheck(t, vers[0].Object["id"] == "v2", "0=v2")
	xCheck(t, vers[1].Object["id"] == "v3", "1=v3")

	// Create v4, which should bump v3 out of the list, not v2 (default)
	v4, err := f1.AddVersion("v4")
	xCheck(t, v4 != nil && err == nil, "Creating v4 failed: %s", err)
	defaultV, err = f1.GetDefault()
	xCheck(t, defaultV != nil && err == nil && defaultV.UID == "v2",
		"err: %q defaultV: %s", err, ToJSON(defaultV))
	vers, err = f1.GetVersions()
	xNoErr(t, err)
	xCheck(t, len(vers) == 2, "Should be 2, but is: %d", len(vers))
	xCheck(t, len(vers) == 2, "Should be 2, but is: %s", ToJSON(vers))
	xCheck(t, vers[0].Object["id"] == "v2", "0=v2")
	xCheck(t, vers[1].Object["id"] == "v4", "1=v4")

	err = rm.SetMaxVersions(0)
	xNoErr(t, err)

	v5, err := f1.AddVersion("v5")
	xNoErr(t, err)
	xNoErr(t, f1.SetDefault(v5))
	_, err = f1.AddVersion("v6")
	xNoErr(t, err)
	_, err = f1.AddVersion("v7")
	xNoErr(t, err)
	_, err = f1.AddVersion("v8")
	xNoErr(t, err)
	_, err = f1.AddVersion("v9")
	xNoErr(t, err)
	vers, err = f1.GetVersions()
	xNoErr(t, err)
	xCheck(t, len(vers) == 7, "Should be 7, but is: %d", len(vers))
	xCheck(t, len(vers) == 7, "Should be 7, but is: %s", ToJSON(vers))
	defaultV, err = f1.GetDefault()
	xCheck(t, defaultV != nil && err == nil && defaultV.UID == "v5",
		"err: %q defaultV: %s", err, ToJSON(defaultV))

	// Now set maxVer to 1 and just v5 should remain
	err = rm.SetMaxVersions(1)
	xNoErr(t, err)

	vers, err = f1.GetVersions()
	xNoErr(t, err)
	xCheck(t, len(vers) == 1, "Should be 1, but is: %d", len(vers))
	xCheck(t, len(vers) == 1, "Should be 1, but is: %s", ToJSON(vers))
	xCheck(t, vers[0].Object["id"] == "v5", "0=v5")
}
