package tests

import (
	"fmt"
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
	xCheckGet(t, reg, "/dirs/d1/files/f1/versions/v1$structure",
		`{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`)
	xCheckGet(t, reg, "/dirs/d1/files/f1/versions/xxx", "Not found\n")
	xCheckGet(t, reg, "dirs/d1/files/f1/versions/xxx", "Not found\n")
	xCheckGet(t, reg, "/dirs/d1/files/f1/versions/xxx/yyy", "URL is too long\n")
	xCheckGet(t, reg, "dirs/d1/files/f1/versions/xxx/yyy", "URL is too long\n")

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`)

	vt, err = f1.FindVersion("v2", false)
	xNoErr(t, err)
	xCheck(t, vt != nil, "Didn't find v2")
	xJSONCheck(t, vt, v2)

	vt, err = f1.FindVersion("xxx", false)
	xNoErr(t, err)
	xCheck(t, vt == nil, "Find version xxx should have failed")

	err = v2.DeleteSetNextVersion("")
	xNoErr(t, err)
	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{}}}}},"d2":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`)

	vt, err = f1.FindVersion("v2", false)
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
		`{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{},"v3":{}}}}},"d2":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`)
	xCheckGet(t, reg, "/dirs/d1/files/f1$structure?inline=meta", `{
  "fileid": "f1",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/d1/files/f1$structure",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "epoch": 3,
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "defaultversionid": "v3",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3$structure",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)
	vt, err = f1.FindVersion("v2", false)
	xNoErr(t, err)
	err = vt.DeleteSetNextVersion("")
	xNoErr(t, err)
	xJSONCheck(t, f1.Get("defaultversionid"), "v3")

	vt, err = f1.FindVersion("v3", false)
	xNoErr(t, err)
	xCheck(t, vt != nil, "Can't be nil")
	err = vt.DeleteSetNextVersion("")
	xNoErr(t, err)
	xJSONCheck(t, f1.Get("defaultversionid"), "v1")

	f1, err = d2.FindResource("files", "f1", false)
	xNoErr(t, err)
	xNoErr(t, f1.SetDefault(v2))
	_, err = f1.AddVersion("v3")
	xNoErr(t, err)
	vt, err = f1.FindVersion("v1", false)
	xNoErr(t, err)
	xCheck(t, vt != nil, "should not be nil")
	err = vt.DeleteSetNextVersion("")
	xNoErr(t, err)
	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{}}}}},"d2":{"files":{"f1":{"meta":{},"versions":{"v1.1":{},"v3":{}}}}}}}`)

	err = vt.DeleteSetNextVersion("v2")
	xCheckErr(t, err, `Can't find next default Version "v2"`)

	vt, err = f1.FindVersion("v1.1", false)
	xNoErr(t, err)
	xCheck(t, vt != nil, "should not be nil")

	err = vt.DeleteSetNextVersion("v1.1")
	xCheckErr(t, err, `Can't set defaultversionid to Version being deleted`)

	vt, err = f1.AddVersion("v4")
	xNoErr(t, err)

	err = vt.DeleteSetNextVersion("v3")
	xNoErr(t, err)

	xCheckGet(t, reg, "dirs/d2/files?inline=meta",
		`{
  "f1": {
    "fileid": "f1",
    "versionid": "v3",
    "self": "http://localhost:8181/dirs/d2/files/f1$structure",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "metaurl": "http://localhost:8181/dirs/d2/files/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "http://localhost:8181/dirs/d2/files/f1/meta",
      "epoch": 3,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:03Z",

      "defaultversionid": "v3",
      "defaultversionurl": "http://localhost:8181/dirs/d2/files/f1/versions/v3$structure",
      "defaultversionsticky": true
    },
    "versionsurl": "http://localhost:8181/dirs/d2/files/f1/versions",
    "versionscount": 2
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
	v1, _ := f1.FindVersion("v1", false)
	v2, _ := f1.AddVersion("v2")

	xCheckGet(t, reg, "dirs/d1/files/f1$structure?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$structure",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$structure"
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	// Doesn't change much, but does make it sticky
	xNoErr(t, f1.SetDefault(v2))

	xCheckGet(t, reg, "dirs/d1/files/f1$structure?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$structure",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$structure",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	v3, _ := f1.AddVersion("v3")

	xCheckGet(t, reg, "dirs/d1/files/f1$structure?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$structure",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$structure",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

	// Now unstick it and it default should be v3 now
	xNoErr(t, f1.SetDefault(nil))
	xCheckGet(t, reg, "dirs/d1/files/f1$structure?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/d1/files/f1$structure",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "epoch": 3,
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:03Z",

    "defaultversionid": "v3",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3$structure"
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

	v4, _ := f1.AddVersion("v4")
	xNoErr(t, f1.SetDefault(v4))
	v5, _ := f1.AddVersion("v5")

	xCheckGet(t, reg, "dirs/d1/files/f1$structure?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v4",
  "self": "http://localhost:8181/dirs/d1/files/f1$structure",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "epoch": 4,
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "defaultversionid": "v4",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v4$structure",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 5
}
`)

	err := v1.DeleteSetNextVersion("")
	xNoErr(t, err)
	xCheckGet(t, reg, "dirs/d1/files/f1$structure?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v4",
  "self": "http://localhost:8181/dirs/d1/files/f1$structure",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "epoch": 5,
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:03Z",

    "defaultversionid": "v4",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v4$structure",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 4
}
`)

	err = v3.DeleteSetNextVersion("v1")
	xCheckErr(t, err, "Can't find next default Version \"v1\"")
	err = v3.DeleteSetNextVersion("v2")
	xNoErr(t, err)
	xCheckGet(t, reg, "dirs/d1/files/f1$structure?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$structure",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "epoch": 6,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$structure",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

	err = v2.DeleteSetNextVersion("")
	xNoErr(t, err)
	xCheckGet(t, reg, "dirs/d1/files/f1$structure?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v5",
  "self": "http://localhost:8181/dirs/d1/files/f1$structure",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "epoch": 7,
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:03Z",

    "defaultversionid": "v5",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v5$structure"
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	xNoErr(t, v4.DeleteSetNextVersion(""))
	xCheckGet(t, reg, "dirs/d1/files/f1$structure?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v5",
  "self": "http://localhost:8181/dirs/d1/files/f1$structure",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "epoch": 8,
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:03Z",

    "defaultversionid": "v5",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v5$structure"
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	xNoErr(t, v5.DeleteSetNextVersion(""))
	xCheckGet(t, reg, "dirs/d1/files/f1$structure", "Not found\n")
}

func TestDefaultVersionMaxVersions(t *testing.T) {
	reg := NewRegistry("TestDefaultVersionMaxVersions")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 3, true, true, true)

	d1, _ := reg.AddGroup("dirs", "d1")
	f1, _ := d1.AddResource("files", "f1", "v1")
	f1.FindVersion("v1", false)
	f1.AddVersion("v2")
	f1.AddVersion("v3")

	xCheckGet(t, reg, "dirs/d1/files/f1$structure?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/d1/files/f1$structure",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "defaultversionid": "v3",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3$structure"
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

	v4, _ := f1.AddVersion("v4")

	xCheckGet(t, reg, "dirs/d1/files/f1$structure?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v4",
  "self": "http://localhost:8181/dirs/d1/files/f1$structure",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "defaultversionid": "v4",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v4$structure"
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

	xNoErr(t, f1.SetDefault(v4))
	f1.AddVersion("v5")
	// check def = v4
	f1.AddVersion("v6")
	f1.AddVersion("v7")
	f1.AddVersion("v8")
	// check def = v4    v8, v7, v4

	xCheckGet(t, reg, "dirs/d1/files/f1$structure?inline=versions,meta",
		`{
  "fileid": "f1",
  "versionid": "v4",
  "self": "http://localhost:8181/dirs/d1/files/f1$structure",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "epoch": 3,
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:03Z",

    "defaultversionid": "v4",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v4$structure",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v4": {
      "fileid": "f1",
      "versionid": "v4",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v4$structure",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z"
    },
    "v7": {
      "fileid": "f1",
      "versionid": "v7",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v7$structure",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:03Z",
      "modifiedat": "2024-01-01T12:00:03Z"
    },
    "v8": {
      "fileid": "f1",
      "versionid": "v8",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v8$structure",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:03Z",
      "modifiedat": "2024-01-01T12:00:03Z"
    }
  },
  "versionscount": 3
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
		registry.Object{"clireq": "test"}, false, false)
	xNoErr(t, err)
	reg.Commit()

	_, err = f1.AddVersion("v2")
	xCheckErr(t, err, "Required property \"clireq\" is missing")
	reg.Rollback()
	reg.Refresh()

	v1, _, err := f1.UpsertVersionWithObject("v2", registry.Object{"clireq": "test"}, registry.ADD_ADD)
	xNoErr(t, err)
	reg.Commit()

	err = v1.SetSave("clireq", nil)
	xCheckErr(t, err, "Required property \"clireq\" is missing")

	err = v1.SetSave("clireq", "again")
	xNoErr(t, err)
}

func TestVersionOrdering(t *testing.T) {
	// Make sure that "latest" is based on "createdat" first and then
	// case insensitive "ID"s (smallest == oldest)
	reg := NewRegistry("TestVersionOrdering")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, false)
	d1, _ := reg.AddGroup("dirs", "d1")
	f1, _ := d1.AddResource("files", "f1", "z5")
	f1.AddVersion("v2")
	f1.AddVersion("v9")
	f1.AddVersion("V3")
	f1.AddVersion("V1")
	f1.AddVersion("Z1")
	f1.AddVersion("v5")

	t0 := "2020-01-02T12:00:00Z"
	t1 := "2024-01-02T12:00:00Z"
	t2 := "2023-11-22T01:02:03Z"
	t9 := "2025-01-02T12:00:00Z"
	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1?nested", `{
	  "versions": {
	    "z5": { "createdat": "`+t1+`","modifiedat":"`+t2+`" },
	    "v2": { "createdat": "`+t1+`","modifiedat":"`+t2+`" },
	    "V3": { "createdat": "`+t0+`","modifiedat":"`+t2+`" },
	    "V1": { "createdat": "`+t9+`","modifiedat":"`+t2+`" },
	    "Z1": { "createdat": "`+t1+`","modifiedat":"`+t2+`" },
	    "v9": { "createdat": "`+t1+`","modifiedat":"`+t2+`" },
	    "v5": { "createdat": "`+t1+`","modifiedat":"`+t2+`" }
	  }
    }`, 200, `--{
  "fileid": "f1",
  "versionid": "v5",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "`+t1+`",
  "modifiedat": "`+t2+`",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 7
}
`)
	ids := []string{"v5", "V1", "z5", "Z1", "v9", "v2", "V3"}

	for i, id := range ids {
		xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/"+id, ``, 204, ``)
		if i == len(ids)-1 {
			break
		}

		ct := t1
		if id == "v5" {
			ct = t9
		}
		if id == "v2" {
			ct = t0
		}

		xHTTP(t, reg, "GET", "/dirs/d1/files/f1", ``, 200, fmt.Sprintf(`--{
  "fileid": "f1",
  "versionid": "%s",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "`+ct+`",
  "modifiedat": "`+t2+`",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": %d
}
`, ids[i+1], 6-i))
	}

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1", ``, 404, `Not found
`)

}
