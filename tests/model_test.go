package tests

import (
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestNoModel(t *testing.T) {
	reg, err := registry.NewRegistry("TestNoModel")
	defer reg.Delete()
	xNoErr(t, err)
	xCheck(t, reg != nil, "reg created didn't work")

	xCheckGet(t, reg, "/model", "{}\n")
	xCheckGet(t, reg, "?model", `{
  "id": "TestNoModel",
  "self": "http:///",
  "model": {}
}
`)

	xCheckGet(t, reg, "/model/foo", "404: Not found")
}

func TestGroupModelCreate(t *testing.T) {
	reg, err := registry.NewRegistry("TestGroupModelCreate")
	defer reg.Delete()
	xNoErr(t, err)
	xCheck(t, reg != nil, "reg created didn't work")

	gm, err := reg.AddGroupModel("dirs", "dir", "schema-url")
	xNoErr(t, err)
	xCheck(t, gm != nil, "gm created didn't work")
	xCheckGet(t, reg, "/model", `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "schema": "schema-url"
    }
  }
}
`)

	// Now error checking
	gm, err = reg.AddGroupModel("dirs1", "", "schema-url") // missing value
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.AddGroupModel("", "", "schema-url") // missing value
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.AddGroupModel("", "", "") // missing value
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.AddGroupModel("", "dir1", "") // missing value
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.AddGroupModel("dirs", "dir", "schema-url") // dup
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.AddGroupModel("dirs1", "dir", "") // dup
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.AddGroupModel("dirs", "dir1", "") // dup
	xCheck(t, gm == nil && err != nil, "gm should have failed")
}

func TestResourceModelCreate(t *testing.T) {
	reg, err := registry.NewRegistry("TestResourceModels")
	defer reg.Delete()
	xNoErr(t, err)
	xCheck(t, reg != nil, "reg created didn't work")

	gm, err := reg.AddGroupModel("dirs", "dir", "dirs-url")
	xNoErr(t, err)
	xCheck(t, gm != nil, "gm should have worked")

	rm, err := gm.AddResourceModel("files", "file", 5, true, true)
	xNoErr(t, err)
	xCheck(t, rm != nil, "rm should have worked")

	rm2, err := gm.AddResourceModel("files", "file", 0, true, true)
	xCheck(t, rm2 == nil && err != nil, "rm2 should have failed")

	rm2, err = gm.AddResourceModel("files2", "file", 0, true, true)
	xCheck(t, rm2 == nil && err != nil, "rm2 should have failed")

	rm2, err = gm.AddResourceModel("", "file2", 0, true, true)
	xCheck(t, rm2 == nil && err != nil, "rm2 should have failed")

	rm2, err = gm.AddResourceModel("files2", "", 0, true, true)
	xCheck(t, rm2 == nil && err != nil, "rm2 should have failed")

	rm2, err = gm.AddResourceModel("files2", "file2", -1, true, true)
	xCheck(t, rm2 == nil && err != nil, "rm2 should have failed")

	gm2, err := reg.AddGroupModel("dirs2", "dir2", "dirs-url")
	xNoErr(t, err)
	xCheck(t, gm != nil, "gm2 should have worked")

	rm2, err = gm2.AddResourceModel("files", "file", 0, true, true)
	xCheck(t, rm != nil && err == nil, "gm2/rm2 should have worked")

	xCheckGet(t, reg, "/model", `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "schema": "dirs-url",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "versions": 5,
          "versionId": true,
          "latest": true
        }
      }
    },
    "dirs2": {
      "plural": "dirs2",
      "singular": "dir2",
      "schema": "dirs-url",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "versions": 0,
          "versionId": true,
          "latest": true
        }
      }
    }
  }
}
`)
}

func TestMultModelCreate(t *testing.T) {
	reg, err := registry.NewRegistry("TestMultModelCreate")
	defer reg.Delete()
	xNoErr(t, err)
	xCheck(t, reg != nil, "reg created didn't work")

	gm1, err := reg.AddGroupModel("gms1", "gm1", "gm1-url")
	xCheck(t, gm1 != nil && err == nil, "gm1 should have worked")

	rm1, err := gm1.AddResourceModel("rms1", "rm1", 0, true, true)
	xCheck(t, rm1 != nil && err == nil, "rm1 should have worked")

	rm2, err := gm1.AddResourceModel("rms2", "rm2", 1, true, true)
	xCheck(t, rm2 != nil && err == nil, "rm2 should have worked")

	gm2, err := reg.AddGroupModel("gms2", "gm2", "gm1-url")
	xCheck(t, gm1 != nil && err == nil, "gm1 should have worked")

	rm21, err := gm2.AddResourceModel("rms1", "rm1", 2, true, true)
	xCheck(t, rm21 != nil && err == nil, "rm21 should have worked")

	rm22, err := gm2.AddResourceModel("rms2", "rm2", 3, true, true)
	xCheck(t, rm22 != nil && err == nil, "rm12 should have worked")

	xCheckGet(t, reg, "/model", `{
  "groups": {
    "gms1": {
      "plural": "gms1",
      "singular": "gm1",
      "schema": "gm1-url",
      "resources": {
        "rms1": {
          "plural": "rms1",
          "singular": "rm1",
          "versions": 0,
          "versionId": true,
          "latest": true
        },
        "rms2": {
          "plural": "rms2",
          "singular": "rm2",
          "versions": 1,
          "versionId": true,
          "latest": true
        }
      }
    },
    "gms2": {
      "plural": "gms2",
      "singular": "gm2",
      "schema": "gm1-url",
      "resources": {
        "rms1": {
          "plural": "rms1",
          "singular": "rm1",
          "versions": 2,
          "versionId": true,
          "latest": true
        },
        "rms2": {
          "plural": "rms2",
          "singular": "rm2",
          "versions": 3,
          "versionId": true,
          "latest": true
        }
      }
    }
  }
}
`)
}

func TestModelAPI(t *testing.T) {
	reg, err := registry.NewRegistry("TestModelAPI")
	defer reg.Delete()
	xNoErr(t, err)
	xCheck(t, reg != nil, "reg created didn't work")

	gm, _ := reg.AddGroupModel("dirs1", "dir1", "")
	gm.AddResourceModel("files", "file", 2, true, false)

	gm2, _ := reg.AddGroupModel("dirs2", "dir2", "")
	gm2.AddResourceModel("files", "file", 0, false, true)

	m := reg.LoadModel()
	xJSONCheck(t, m, reg.Model)
}

func TestMultModel2Create(t *testing.T) {
	reg, err := registry.NewRegistry("TestMultModel2Create")
	defer reg.Delete()
	xNoErr(t, err)
	xCheck(t, reg != nil, "reg created didn't work")

	gm, _ := reg.AddGroupModel("dirs1", "dir1", "")
	gm.AddResourceModel("files", "file", 2, true, false)

	d, _ := reg.AddGroup("dirs1", "d1")
	f, _ := d.AddResource("files", "f1", "v1")
	f.AddVersion("v2")
	d, _ = reg.AddGroup("dirs1", "d2")
	f, _ = d.AddResource("files", "f2", "v1")
	f.AddVersion("v1.1")

	gm2, _ := reg.AddGroupModel("dirs2", "dir2", "")
	gm2.AddResourceModel("files", "file", 0, false, true)
	d2, _ := reg.AddGroup("dirs2", "d2")
	d2.AddResource("files", "f2", "v1")

	// /dirs1/d1/f1/v1
	//            /v2
	//       /d2/f2/v1
	//             v1.1
	// /dirs2/f2/f2/v1

	xCheckGet(t, reg, "?model&inline", `{
  "id": "TestMultModel2Create",
  "self": "http:///",
  "model": {
    "groups": {
      "dirs1": {
        "plural": "dirs1",
        "singular": "dir1",
        "resources": {
          "files": {
            "plural": "files",
            "singular": "file",
            "versions": 2,
            "versionId": true,
            "latest": false
          }
        }
      },
      "dirs2": {
        "plural": "dirs2",
        "singular": "dir2",
        "resources": {
          "files": {
            "plural": "files",
            "singular": "file",
            "versions": 0,
            "versionId": false,
            "latest": true
          }
        }
      }
    }
  },

  "dirs1": {
    "d1": {
      "id": "d1",
      "self": "http:///dirs1/d1",

      "files": {
        "f1": {
          "id": "f1",
          "self": "http:///dirs1/d1/files/f1",
          "latestId": "v2",
          "latestUrl": "http:///dirs1/d1/files/f1/versions/v2",

          "versions": {
            "v1": {
              "id": "v1",
              "self": "http:///dirs1/d1/files/f1/versions/v1"
            },
            "v2": {
              "id": "v2",
              "self": "http:///dirs1/d1/files/f1/versions/v2"
            }
          },
          "versionsCount": 2,
          "versionsUrl": "http:///dirs1/d1/files/f1/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http:///dirs1/d1/files"
    },
    "d2": {
      "id": "d2",
      "self": "http:///dirs1/d2",

      "files": {
        "f2": {
          "id": "f2",
          "self": "http:///dirs1/d2/files/f2",
          "latestId": "v1.1",
          "latestUrl": "http:///dirs1/d2/files/f2/versions/v1.1",

          "versions": {
            "v1": {
              "id": "v1",
              "self": "http:///dirs1/d2/files/f2/versions/v1"
            },
            "v1.1": {
              "id": "v1.1",
              "self": "http:///dirs1/d2/files/f2/versions/v1.1"
            }
          },
          "versionsCount": 2,
          "versionsUrl": "http:///dirs1/d2/files/f2/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http:///dirs1/d2/files"
    }
  },
  "dirs1Count": 2,
  "dirs1Url": "http:///dirs1",
  "dirs2": {
    "d2": {
      "id": "d2",
      "self": "http:///dirs2/d2",

      "files": {
        "f2": {
          "id": "f2",
          "self": "http:///dirs2/d2/files/f2",
          "latestId": "v1",
          "latestUrl": "http:///dirs2/d2/files/f2/versions/v1",

          "versions": {
            "v1": {
              "id": "v1",
              "self": "http:///dirs2/d2/files/f2/versions/v1"
            }
          },
          "versionsCount": 1,
          "versionsUrl": "http:///dirs2/d2/files/f2/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http:///dirs2/d2/files"
    }
  },
  "dirs2Count": 1,
  "dirs2Url": "http:///dirs2"
}
`)

	gm, _ = reg.AddGroupModel("dirs0", "dir0", "")
	gm.AddResourceModel("files", "file", 2, true, false)
	gm, _ = reg.AddGroupModel("dirs3", "dir3", "")
	gm.AddResourceModel("files", "file", 2, true, false)

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs0":{},"dirs1":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}},"dirs2":{"d2":{"files":{"f2":{"versions":{"v1":{}}}}}},"dirs3":{}}`)

	gm, _ = reg.AddGroupModel("dirs15", "dir15", "")
	gm.AddResourceModel("files", "file", 2, true, false)

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs0":{},"dirs1":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}},"dirs15":{},"dirs2":{"d2":{"files":{"f2":{"versions":{"v1":{}}}}}},"dirs3":{}}`)

	gm, _ = reg.AddGroupModel("dirs01", "dir01", "")
	gm, _ = reg.AddGroupModel("dirs02", "dir02", "")
	gm, _ = reg.AddGroupModel("dirs14", "dir014", "")
	gm, _ = reg.AddGroupModel("dirs16", "dir016", "")
	gm, _ = reg.AddGroupModel("dirs4", "dir4", "")
	gm, _ = reg.AddGroupModel("dirs5", "dir5", "")

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs0":{},"dirs01":{},"dirs02":{},"dirs1":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}},"dirs14":{},"dirs15":{},"dirs16":{},"dirs2":{"d2":{"files":{"f2":{"versions":{"v1":{}}}}}},"dirs3":{},"dirs4":{},"dirs5":{}}`)
}
