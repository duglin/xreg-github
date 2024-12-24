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
	}, false, false)
	xCheckErr(t, err, `'xref' (`+f1.Path+`) must be of the form: `+
		`/GROUPS/gID/RESOURCES/rID`)

	_, err = d.AddResourceWithObject("files", "fx", "", registry.Object{
		"meta": map[string]any{
			"xref": "foo/" + f1.Path, // make it bad
		},
	}, false, false)
	xCheckErr(t, err, `'xref' (foo/`+f1.Path+`) must be of the form: `+
		`/GROUPS/gID/RESOURCES/rID`)

	fx, err := d.AddResourceWithObject("files", "fx", "", registry.Object{
		"meta": map[string]any{
			"xref": "/dirs/d1/files/f1",
		},
	}, false, false)
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
		}, registry.ADD_UPDATE, false, false)
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
      "epoch": 1,
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
		}, registry.ADD_UPDATE, false, false)
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
		}, registry.ADD_UPDATE, false, false)
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
      "epoch": 2,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:04Z",

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
