package tests

import (
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestXrefBasic(t *testing.T) {
	reg := NewRegistry("TestXrefBasic")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)
	d, _ := reg.AddGroup("dirs", "d1")
	f1, err := d.AddResource("files", "f1", "v1")

	rows, err := reg.Query("select * from Versions where ResourceSID=?",
		f1.DbSID)
	xNoErr(t, err)
	xCheckEqual(t, "", len(rows), 1) // Just to be sure Query works ok

	_, err = d.AddResourceWithObject("files", "fx", "", registry.Object{
		"meta": map[string]any{
			"xref": f1.Path, // missing leading /
		},
	}, false)
	xCheckErr(t, err, `'xref' (`+f1.Path+`) must be of the form: `+
		`/GROUPS/gID/RESOURCES/rID`)

	_, err = d.AddResourceWithObject("files", "fx", "", registry.Object{
		"meta": map[string]any{
			"xref": "foo/" + f1.Path, // make it bad
		},
	}, false)
	xCheckErr(t, err, `'xref' (foo/`+f1.Path+`) must be of the form: `+
		`/GROUPS/gID/RESOURCES/rID`)

	fx, err := d.AddResourceWithObject("files", "fx", "", registry.Object{
		"meta": map[string]any{
			"xref": "/dirs/d1/files/f1",
		},
	}, false)
	xNoErr(t, err)

	// Grab #createdat so we can make sure it's used when we remove 'xref'
	meta, _ := fx.FindMeta(false)
	oldCreatedAt := meta.Get("#createdat")

	// Make sure the Resource doesn't have any versions in the DB.
	// Use fx.GetVersions() will grab from xref target so don't use that
	rows, err = reg.Query("select * from Versions where ResourceSID=?",
		fx.DbSID)
	xNoErr(t, err)
	xCheckEqual(t, "", len(rows), 0)

	xHTTP(t, reg, "GET", "/dirs/d1/files", "", 200, `{
  "f1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f1$structure",
    "xid": "/dirs/d1/files/f1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versionscount": 1
  },
  "fx": {
    "fileid": "fx",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/fx$structure",
    "xid": "/dirs/d1/files/fx",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
    "versionscount": 1
  }
}
`)

	xNoErr(t, f1.SetSaveDefault("description", "testing xref"))
	xCheckEqual(t, "", fx.Get("description"), "testing xref")

	v1, err := f1.FindVersion("v1", false)
	xNoErr(t, err)
	xNoErr(t, v1.SetSave("name", "v1 name"))

	xHTTP(t, reg, "GET", "/dirs/d1/files?inline", "", 200, `{
  "f1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f1$structure",
    "xid": "/dirs/d1/files/f1",
    "epoch": 2,
    "name": "v1 name",
    "isdefault": true,
    "description": "testing xref",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "http://localhost:8181/dirs/d1/files/f1/meta",
      "xid": "/dirs/d1/files/f1/meta",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",

      "defaultversionid": "v1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure"
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versions": {
      "v1": {
        "fileid": "f1",
        "versionid": "v1",
        "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure",
        "xid": "/dirs/d1/files/f1/versions/v1",
        "epoch": 2,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:02Z"
      }
    },
    "versionscount": 1
  },
  "fx": {
    "fileid": "fx",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/fx$structure",
    "xid": "/dirs/d1/files/fx",
    "epoch": 2,
    "name": "v1 name",
    "isdefault": true,
    "description": "testing xref",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",

    "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
    "meta": {
      "fileid": "fx",
      "self": "http://localhost:8181/dirs/d1/files/fx/meta",
      "xid": "/dirs/d1/files/fx/meta",
      "xref": "/dirs/d1/files/f1",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",

      "defaultversionid": "v1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v1$structure"
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
    "versions": {
      "v1": {
        "fileid": "fx",
        "versionid": "v1",
        "self": "http://localhost:8181/dirs/d1/files/fx/versions/v1$structure",
        "xid": "/dirs/d1/files/fx/versions/v1",
        "epoch": 2,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:02Z"
      }
    },
    "versionscount": 1
  }
}
`)

	// Now clear xref and make sure a version is created
	fx, isNew, err := d.UpsertResourceWithObject("files", "fx", "",
		registry.Object{
			"meta": map[string]any{
				"xref": nil,
			},
		}, registry.ADD_UPDATE, false)
	xNoErr(t, err)
	xCheckEqual(t, "", isNew, false)

	rows, err = reg.Query("select * from Versions where ResourceSID=?",
		fx.DbSID)
	xNoErr(t, err)
	xCheckEqual(t, "", len(rows), 1)

	meta, _ = fx.FindMeta(false)
	if meta.Get("createdat") != oldCreatedAt {
		t.Errorf("CreatedAt has wrong value, should be %q, not %q",
			oldCreatedAt, meta.Get("createdat"))
		t.FailNow()
	}

	xHTTP(t, reg, "GET", "/dirs/d1/files?inline", "", 200, `{
  "f1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f1$structure",
    "xid": "/dirs/d1/files/f1",
    "epoch": 2,
    "name": "v1 name",
    "isdefault": true,
    "description": "testing xref",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "http://localhost:8181/dirs/d1/files/f1/meta",
      "xid": "/dirs/d1/files/f1/meta",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",

      "defaultversionid": "v1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure"
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versions": {
      "v1": {
        "fileid": "f1",
        "versionid": "v1",
        "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure",
        "xid": "/dirs/d1/files/f1/versions/v1",
        "epoch": 2,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:02Z"
      }
    },
    "versionscount": 1
  },
  "fx": {
    "fileid": "fx",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/fx$structure",
    "xid": "/dirs/d1/files/fx",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:03Z",

    "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
    "meta": {
      "fileid": "fx",
      "self": "http://localhost:8181/dirs/d1/files/fx/meta",
      "xid": "/dirs/d1/files/fx/meta",
      "epoch": 3,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:03Z",

      "defaultversionid": "1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1$structure"
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
    "versions": {
      "1": {
        "fileid": "fx",
        "versionid": "1",
        "self": "http://localhost:8181/dirs/d1/files/fx/versions/1$structure",
        "xid": "/dirs/d1/files/fx/versions/1",
        "epoch": 1,
        "isdefault": true,
        "createdat": "2024-01-01T12:00:03Z",
        "modifiedat": "2024-01-01T12:00:03Z"
      }
    },
    "versionscount": 1
  }
}
`)

	// re-Set xref and make sure the version is deleted
	fx, isNew, err = d.UpsertResourceWithObject("files", "fx", "",
		registry.Object{
			"meta": map[string]any{
				"xref": "/" + f1.Path,
			},
		}, registry.ADD_UPDATE, false)
	xNoErr(t, err)
	xCheckEqual(t, "", isNew, false)

	rows, err = reg.Query("select * from Versions where ResourceSID=?",
		fx.DbSID)
	xNoErr(t, err)
	xCheckEqual(t, "", len(rows), 0)

	xHTTP(t, reg, "GET", "/dirs/d1/files?inline", "", 200, `{
  "f1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f1$structure",
    "xid": "/dirs/d1/files/f1",
    "epoch": 2,
    "name": "v1 name",
    "isdefault": true,
    "description": "testing xref",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "http://localhost:8181/dirs/d1/files/f1/meta",
      "xid": "/dirs/d1/files/f1/meta",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",

      "defaultversionid": "v1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure"
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versions": {
      "v1": {
        "fileid": "f1",
        "versionid": "v1",
        "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure",
        "xid": "/dirs/d1/files/f1/versions/v1",
        "epoch": 2,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:02Z"
      }
    },
    "versionscount": 1
  },
  "fx": {
    "fileid": "fx",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/fx$structure",
    "xid": "/dirs/d1/files/fx",
    "epoch": 2,
    "name": "v1 name",
    "isdefault": true,
    "description": "testing xref",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",

    "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
    "meta": {
      "fileid": "fx",
      "self": "http://localhost:8181/dirs/d1/files/fx/meta",
      "xid": "/dirs/d1/files/fx/meta",
      "xref": "/dirs/d1/files/f1",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",

      "defaultversionid": "v1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v1$structure"
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
    "versions": {
      "v1": {
        "fileid": "fx",
        "versionid": "v1",
        "self": "http://localhost:8181/dirs/d1/files/fx/versions/v1$structure",
        "xid": "/dirs/d1/files/fx/versions/v1",
        "epoch": 2,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:02Z"
      }
    },
    "versionscount": 1
  }
}
`)

	// Now clear xref and set some props at the same time
	fx, isNew, err = d.UpsertResourceWithObject("files", "fx", "",
		registry.Object{
			"meta": map[string]any{
				"xref": nil,
			},
			"name":        "fx name",
			"description": "very cool",
		}, registry.ADD_UPDATE, false)
	xNoErr(t, err)
	xCheckEqual(t, "", isNew, false)

	xHTTP(t, reg, "GET", "/dirs/d1/files?inline", "", 200, `{
  "f1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f1$structure",
    "xid": "/dirs/d1/files/f1",
    "epoch": 2,
    "name": "v1 name",
    "isdefault": true,
    "description": "testing xref",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "http://localhost:8181/dirs/d1/files/f1/meta",
      "xid": "/dirs/d1/files/f1/meta",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",

      "defaultversionid": "v1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure"
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versions": {
      "v1": {
        "fileid": "f1",
        "versionid": "v1",
        "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure",
        "xid": "/dirs/d1/files/f1/versions/v1",
        "epoch": 2,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:02Z"
      }
    },
    "versionscount": 1
  },
  "fx": {
    "fileid": "fx",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/fx$structure",
    "xid": "/dirs/d1/files/fx",
    "epoch": 1,
    "name": "fx name",
    "isdefault": true,
    "description": "very cool",
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:03Z",

    "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
    "meta": {
      "fileid": "fx",
      "self": "http://localhost:8181/dirs/d1/files/fx/meta",
      "xid": "/dirs/d1/files/fx/meta",
      "epoch": 4,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:03Z",

      "defaultversionid": "1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1$structure"
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
    "versions": {
      "1": {
        "fileid": "fx",
        "versionid": "1",
        "self": "http://localhost:8181/dirs/d1/files/fx/versions/1$structure",
        "xid": "/dirs/d1/files/fx/versions/1",
        "epoch": 1,
        "name": "fx name",
        "isdefault": true,
        "description": "very cool",
        "createdat": "2024-01-01T12:00:03Z",
        "modifiedat": "2024-01-01T12:00:03Z"
      }
    },
    "versionscount": 1
  }
}
`)

}

func TestXrefErrors(t *testing.T) {
	reg := NewRegistry("TestXrefErrors")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, false)
	d, _ := reg.AddGroup("dirs", "d1")
	_, err := d.AddResource("files", "f1", "v1")
	xNoErr(t, err)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta",
		`{"xref": "/dirs/d1/files/fx","fileid":"f2"}`, 400,
		"meta.fileid must be \"f1\", not \"f2\"\n")
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta",
		`{"xref": "/dirs/d1/files/fx","epoch":5}`, 400,
		"Attribute \"epoch\"(5) doesn't match existing value (1)\n")
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta",
		`{"xref": "/dirs/d1/files/fx", "modifiedat":"2025-01-01T12:00:00"}`,
		400,
		"Extra attributes (modifiedat) in \"meta\" not allowed when \"xref\" is set\n")
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta",
		`{"foo":"foo","xref": "/dirs/d1/files/fx"}`, 400,
		"Extra attributes (foo) in \"meta\" not allowed when \"xref\" is set\n")

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"meta": {"fileid":"f1", "xref":"/dirs/d1/files/f1"},"epoch":5, "description": "x"}`,
		400,
		"Extra attributes (description,epoch) not allowed when \"xref\" is set\n")
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"meta": {"fileid":"f1", "xref":"/dirs/d1/files/f1"},"epoch":5, "description": "x"}`,
		400,
		"Extra attributes (description,epoch) not allowed when \"xref\" is set\n")

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"fileid": "f2", "meta": {"xref":"/dirs/d1/files/f1"}}`, 400,
		"The \"fileid\" attribute must be set to \"f1\", not \"f2\"\n")
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"meta": {"xref":"/dirs/d1/files/f1","epoch":6}}`, 400,
		"Attribute \"epoch\"(6) doesn't match existing value (1)\n")
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"fileid": "f1", "meta": {"xref":"/dirs/d1/files/f1","modifiedat":"2025-01-01-T:12:00:00"}}`, 400,
		"Extra attributes (modifiedat) in \"meta\" not allowed when \"xref\" is set\n")

	// Works!
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta",
		`{"xref": "/dirs/d1/files/fx", "epoch":1}`,
		200,
		`{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "xref": "/dirs/d1/files/fx"
}
`)
}

func TestXrefRevert(t *testing.T) {
	reg := NewRegistry("TestXrefRevert")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, false)
	d, _ := reg.AddGroup("dirs", "d1")

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v9",
		`{"description":"hi"}`, 201, `{
  "fileid": "f1",
  "versionid": "v9",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v9",
  "xid": "/dirs/d1/files/f1/versions/v9",
  "epoch": 1,
  "isdefault": true,
  "description": "hi",
  "createdat": "2025-01-09T15:59:29.22249886Z",
  "modifiedat": "2025-01-09T15:59:29.22249886Z"
}
`)

	// Revert with no versions (create 2 files so we can grab the TS from f0)
	////////////////////////////////////////////////////////
	xHTTP(t, reg, "POST", "/dirs/d1/files/?inline=meta",
		`{"f0":{}, "fx":{"meta":{"xref":"/dirs/d1/files/f1"}}}`, 200, `{
  "f0": {
    "fileid": "f0",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/f0",
    "xid": "/dirs/d1/files/f0",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

    "metaurl": "http://localhost:8181/dirs/d1/files/f0/meta",
    "meta": {
      "fileid": "f0",
      "self": "http://localhost:8181/dirs/d1/files/f0/meta",
      "xid": "/dirs/d1/files/f0/meta",
      "epoch": 1,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

      "defaultversionid": "1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/f0/versions/1"
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/f0/versions",
    "versionscount": 1
  },
  "fx": {
    "fileid": "fx",
    "versionid": "v9",
    "self": "http://localhost:8181/dirs/d1/files/fx",
    "xid": "/dirs/d1/files/fx",
    "epoch": 1,
    "isdefault": true,
    "description": "hi",
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

    "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
    "meta": {
      "fileid": "fx",
      "self": "http://localhost:8181/dirs/d1/files/fx/meta",
      "xid": "/dirs/d1/files/fx/meta",
      "xref": "/dirs/d1/files/f1",
      "epoch": 1,
      "createdat": "YYYY-MM-DDTHH:MM:02Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

      "defaultversionid": "v9",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v9"
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
    "versionscount": 1
  }
}
`)

	// Grab F0's timestamp so we can compare later
	f0, err := d.FindResource("files", "f0", false)
	xNoErr(t, err)
	f0TS := f0.Get("createdat").(string)
	xCheck(t, f0TS > "2024", "bad ts: %s", f0TS)

	// Notice epoch will be 2 not 1 since it's max(0,fx.epoch)+1
	// Notice meta.createat == f0's createdat, others are now()
	// Make sure we pick up def ver attrs
	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta", `{
  "description": "hello",
  "meta":{"xref":null}
} `, 200, `{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "description": "hello",
  "createdat": "2025-01-01T12:00:02Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:01Z",
    "modifiedat": "2025-01-01T12:00:02Z",

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1"
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)
	fx, err := d.FindResource("files", "fx", false)
	xNoErr(t, err)
	fxMeta, err := fx.FindMeta(false)
	xNoErr(t, err)
	fxMetaTS := fxMeta.Get("createdat").(string)
	xCheck(t, f0TS == fxMetaTS, "Bad ts: %s/%s", f0TS, fxMetaTS)

	// Revert with empty versions
	////////////////////////////////////////////////////////
	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta",
		`{"meta":{"xref":"/dirs/d1/files/f1"}}`, 200, `{
  "fileid": "fx",
  "versionid": "v9",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "description": "hi",
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "xref": "/dirs/d1/files/f1",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",

    "defaultversionid": "v9",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v9"
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta", `{
  "meta":{"xref":null},
  "versions": {}
} `, 200, `{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:02Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "epoch": 3,
    "createdat": "2025-01-01T12:00:01Z",
    "modifiedat": "2025-01-01T12:00:02Z",

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1"
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)
	xNoErr(t, fxMeta.Refresh())
	xNoErr(t, fx.Refresh())
	xCheckEqual(t, "ts check", f0TS, fxMeta.Get("createdat").(string))
	xCheckGreater(t, "ts check", fx.Get("createdat").(string), f0TS)

	// Revert with one version
	////////////////////////////////////////////////////////
	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta",
		`{"meta":{"xref":"/dirs/d1/files/f1"}}`, 200, `{
  "fileid": "fx",
  "versionid": "v9",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "description": "hi",
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "xref": "/dirs/d1/files/f1",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",

    "defaultversionid": "v9",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v9"
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	// Notice "description:bye" is ignored
	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta", `{
  "description": "bye",
  "meta":{"xref":null},
  "versions": { "v1": { "description": "ver1" } }
} `, 200, `{
  "fileid": "fx",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "description": "ver1",
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "epoch": 4,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:01Z",

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v1"
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	xNoErr(t, fxMeta.Refresh())
	xNoErr(t, fx.Refresh())
	xCheckEqual(t, "ts check", f0TS, fxMeta.Get("createdat").(string))
	xCheckGreater(t, "ts check", fx.Get("createdat").(string), f0TS)

	// Revert with two versions - no default
	////////////////////////////////////////////////////////
	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta",
		`{"meta":{"xref":"/dirs/d1/files/f1"}}`, 200, `{
  "fileid": "fx",
  "versionid": "v9",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "description": "hi",
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "xref": "/dirs/d1/files/f1",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",

    "defaultversionid": "v9",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v9"
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	// "description:bye" is ignored
	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta", `{
  "description": "bye",
  "meta":{"xref":null},
  "versions": { "z1": {}, "a1": {} }
} `, 200, `{
  "fileid": "fx",
  "versionid": "z1",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "epoch": 5,
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

    "defaultversionid": "z1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/z1"
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 2
}
`)

	xNoErr(t, fxMeta.Refresh())
	xNoErr(t, fx.Refresh())
	xCheckEqual(t, "ts check", f0TS, fxMeta.Get("createdat").(string))
	xCheckGreater(t, "ts check", fx.Get("createdat").(string), f0TS)

	// Revert with two versions - w/default query param
	////////////////////////////////////////////////////////
	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta",
		`{"meta":{"xref":"/dirs/d1/files/f1"}}`, 200, `{
  "fileid": "fx",
  "versionid": "v9",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "description": "hi",
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "xref": "/dirs/d1/files/f1",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",

    "defaultversionid": "v9",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v9"
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	// Not 100% this is legal per the spec, we should probably reject the
	// query parameter since I think it's only allowed on 'POST /versions'
	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta&setdefaultversionid=bb", `{
  "meta":{"xref":null },
  "versions": { "z2": {}, "b3": {} }
} `, 400, `Version "bb" not found
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta&setdefaultversionid=b3", `{
  "meta":{"xref":null },
  "versions": { "z2": {}, "b3": {} }
} `, 200, `{
  "fileid": "fx",
  "versionid": "b3",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:02Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "epoch": 6,
    "createdat": "2025-01-01T12:00:01Z",
    "modifiedat": "2025-01-01T12:00:02Z",

    "defaultversionid": "b3",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/b3",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 2
}
`)

	xNoErr(t, fxMeta.Refresh())
	xNoErr(t, fx.Refresh())
	xCheckEqual(t, "ts check", f0TS, fxMeta.Get("createdat").(string))
	xCheckGreater(t, "ts check", fx.Get("createdat").(string), f0TS)

	// Revert with two versions - w/default in meta
	////////////////////////////////////////////////////////
	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta",
		`{"meta":{"xref":"/dirs/d1/files/f1"}}`, 200, `{
  "fileid": "fx",
  "versionid": "v9",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "description": "hi",
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "xref": "/dirs/d1/files/f1",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",

    "defaultversionid": "v9",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v9"
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta", `{
  "meta":{"xref":null,
          "defaultversionid": "bb",
          "defaultversionsticky": true },
  "versions": { "z2": {}, "b3": {} }
} `, 400, `Version "bb" not found
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta", `{
  "meta":{"xref":null,
          "defaultversionid": "b3",
          "defaultversionsticky": true },
  "versions": { "z2": {}, "b3": {} }
} `, 200, `{
  "fileid": "fx",
  "versionid": "b3",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:02Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "epoch": 7,
    "createdat": "2025-01-01T12:00:01Z",
    "modifiedat": "2025-01-01T12:00:02Z",

    "defaultversionid": "b3",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/b3",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 2
}
`)
	xNoErr(t, fxMeta.Refresh())
	xNoErr(t, fx.Refresh())
	xCheckEqual(t, "ts check", f0TS, fxMeta.Get("createdat").(string))
	xCheckGreater(t, "ts check", fx.Get("createdat").(string), f0TS)

	// Revert via meta + default
	////////////////////////////////////////////////////////
	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 200, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",

  "defaultversionid": "v9",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v9"
}
`)

	// defaultversionid is ignored because we're not sticky
	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":null,
          "defaultversionid": "bb"}`, 200, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "epoch": 8,
  "createdat": "2025-01-09T23:31:06.391225888Z",
  "modifiedat": "2025-01-09T23:31:07.033435714Z",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1"
}
`)

	// reset again
	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 200, `*`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":null,
          "defaultversionid": "bb",
		  "defaultversionsticky": true}`, 400, `Version "bb" not found
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":null,
		  "defaultversionsticky": true}`, 200, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "epoch": 9,
  "createdat": "2025-01-09T23:16:04.619269627Z",
  "modifiedat": "2025-01-09T23:16:05.273949318Z",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1",
  "defaultversionsticky": true
}
`)

	xNoErr(t, fxMeta.Refresh())
	xNoErr(t, fx.Refresh())
	xCheckEqual(t, "ts check", f0TS, fxMeta.Get("createdat").(string))
	xCheckGreater(t, "ts check", fx.Get("createdat").(string), f0TS)

}

func TestXrefDocs(t *testing.T) {
	reg := NewRegistry("TestXrefRevert")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1", "hello world", 201, "hello world")
	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1"
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1", "", 200, `hello world`)
	// DUG TODO Add more tests around hasdoc + xref
	// xHTTP(t, reg, "GET", "/dirs/d1/files/fx", "", 200, `hello world`)
}
