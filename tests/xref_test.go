package tests

import (
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestXrefBasic(t *testing.T) {
	reg := NewRegistry("TestXrefBasic")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)
	d, _ := reg.AddGroup("dirs", "d1")
	f1, err := d.AddResource("files", "f1", "v1")

	rows, err := reg.Query("select * from Versions where ResourceSID=?",
		f1.DbSID)
	xNoErr(t, err)
	xCheckEqual(t, "", len(rows), 1) // Just to be sure Query works ok

	_, err = d.AddResourceWithObject("files", "fx", "", registry.Object{
		"xref": "/" + f1.Path, // make it bad
	}, false, false)
	xCheckErr(t, err, `'xref' (/`+f1.Path+`) must be of the form: `+
		`GROUPs/gID/RESOURCEs/rID`)

	fx, err := d.AddResourceWithObject("files", "fx", "", registry.Object{
		"xref": "dirs/d1/files/f1",
	}, false, false)
	xNoErr(t, err)

	// Make sure the Resource doesn't have any versions in the DB.
	// Use fx.GetVersions() will grab from xref target so don't use that
	rows, err = reg.Query("select * from Versions where ResourceSID=?",
		fx.DbSID)
	xNoErr(t, err)
	xCheckEqual(t, "", len(rows), 0)

	xHTTP(t, reg, "GET", "/dirs/d1/files", "", 200, `{
  "f1": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1$structure",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure",

    "versionscount": 1,
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
  },
  "fx": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx$structure",
    "xref": "dirs/d1/files/f1",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v1$structure",

    "versionscount": 1,
    "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions"
  }
}
`)

	xNoErr(t, f1.SetSave("description", "testing xref"))
	xCheckEqual(t, "", fx.Get("description"), "testing xref")

	v1, err := f1.FindVersion("v1", false)
	xNoErr(t, err)
	xNoErr(t, v1.SetSave("name", "v1 name"))

	xHTTP(t, reg, "GET", "/dirs/d1/files?inline", "", 200, `{
  "f1": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1$structure",
    "epoch": 2,
    "name": "v1 name",
    "description": "testing xref",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure",

    "versions": {
      "v1": {
        "fileid": "f1",
        "versionid": "v1",
        "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure",
        "epoch": 2,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "YYYY-MM-DDTHH:MM:01Z",
        "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
      }
    },
    "versionscount": 1,
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
  },
  "fx": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx$structure",
    "xref": "dirs/d1/files/f1",
    "epoch": 2,
    "name": "v1 name",
    "description": "testing xref",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v1$structure",

    "versions": {
      "v1": {
        "fileid": "fx",
        "versionid": "v1",
        "self": "http://localhost:8181/dirs/d1/files/fx/versions/v1$structure",
        "epoch": 2,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "YYYY-MM-DDTHH:MM:01Z",
        "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
      }
    },
    "versionscount": 1,
    "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions"
  }
}
`)

	// Now clear xref and make sure a version is created
	fx, isNew, err := d.UpsertResourceWithObject("files", "fx", "",
		registry.Object{
			"xref": nil,
		}, registry.ADD_UPDATE, false, false)
	xNoErr(t, err)
	xCheckEqual(t, "", isNew, false)

	rows, err = reg.Query("select * from Versions where ResourceSID=?",
		fx.DbSID)
	xNoErr(t, err)
	xCheckEqual(t, "", len(rows), 1)

	xHTTP(t, reg, "GET", "/dirs/d1/files?inline", "", 200, `{
  "f1": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1$structure",
    "epoch": 2,
    "name": "v1 name",
    "description": "testing xref",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure",

    "versions": {
      "v1": {
        "fileid": "f1",
        "versionid": "v1",
        "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure",
        "epoch": 2,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "YYYY-MM-DDTHH:MM:01Z",
        "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
      }
    },
    "versionscount": 1,
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
  },
  "fx": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx$structure",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:03Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:03Z",

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1$structure",

    "versions": {
      "1": {
        "fileid": "fx",
        "versionid": "1",
        "self": "http://localhost:8181/dirs/d1/files/fx/versions/1$structure",
        "epoch": 1,
        "isdefault": true,
        "createdat": "YYYY-MM-DDTHH:MM:03Z",
        "modifiedat": "YYYY-MM-DDTHH:MM:03Z"
      }
    },
    "versionscount": 1,
    "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions"
  }
}
`)

	// re-Set xref and make sure the version is deleted
	fx, isNew, err = d.UpsertResourceWithObject("files", "fx", "",
		registry.Object{
			"xref": f1.Path,
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
    "self": "http://localhost:8181/dirs/d1/files/f1$structure",
    "epoch": 2,
    "name": "v1 name",
    "description": "testing xref",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure",

    "versions": {
      "v1": {
        "fileid": "f1",
        "versionid": "v1",
        "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure",
        "epoch": 2,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "YYYY-MM-DDTHH:MM:01Z",
        "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
      }
    },
    "versionscount": 1,
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
  },
  "fx": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx$structure",
    "xref": "dirs/d1/files/f1",
    "epoch": 2,
    "name": "v1 name",
    "description": "testing xref",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v1$structure",

    "versions": {
      "v1": {
        "fileid": "fx",
        "versionid": "v1",
        "self": "http://localhost:8181/dirs/d1/files/fx/versions/v1$structure",
        "epoch": 2,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "YYYY-MM-DDTHH:MM:01Z",
        "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
      }
    },
    "versionscount": 1,
    "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions"
  }
}
`)

	// Now clear xref and set some props at the same time
	fx, isNew, err = d.UpsertResourceWithObject("files", "fx", "",
		registry.Object{
			"xref":        nil,
			"name":        "fx name",
			"description": "very cool",
		}, registry.ADD_UPDATE, false, false)
	xNoErr(t, err)
	xCheckEqual(t, "", isNew, false)

	xHTTP(t, reg, "GET", "/dirs/d1/files?inline", "", 200, `{
  "f1": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1$structure",
    "epoch": 2,
    "name": "v1 name",
    "description": "testing xref",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure",

    "versions": {
      "v1": {
        "fileid": "f1",
        "versionid": "v1",
        "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$structure",
        "epoch": 2,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "YYYY-MM-DDTHH:MM:01Z",
        "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
      }
    },
    "versionscount": 1,
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
  },
  "fx": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx$structure",
    "epoch": 1,
    "name": "fx name",
    "description": "very cool",
    "createdat": "YYYY-MM-DDTHH:MM:03Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:03Z",

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1$structure",

    "versions": {
      "1": {
        "fileid": "fx",
        "versionid": "1",
        "self": "http://localhost:8181/dirs/d1/files/fx/versions/1$structure",
        "epoch": 1,
        "name": "fx name",
        "isdefault": true,
        "description": "very cool",
        "createdat": "YYYY-MM-DDTHH:MM:03Z",
        "modifiedat": "YYYY-MM-DDTHH:MM:03Z"
      }
    },
    "versionscount": 1,
    "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions"
  }
}
`)

}
