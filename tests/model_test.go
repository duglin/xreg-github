package tests

import (
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestNoModel(t *testing.T) {
	reg := NewRegistry("TestNoModel")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "reg created didn't work")

	xCheckGet(t, reg, "/model", "{}\n")
	xCheckGet(t, reg, "?model", `{
  "id": "TestNoModel",
  "self": "http://localhost:8080/",
  "model": {}
}
`)

	xCheckGet(t, reg, "/model/foo", "404: Not found\n")
}

func TestGroupModelCreate(t *testing.T) {
	reg := NewRegistry("TestGroupModelCreate")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "reg created didn't work")

	gm, err := reg.Model.AddGroupModel("dirs", "dir", "schema-url")
	xNoErr(t, err)
	xCheck(t, gm != nil, "gm created didn't work")
	xCheckGet(t, reg, "/model", `{
  "groups": [
    {
      "plural": "dirs",
      "singular": "dir",
      "schema": "schema-url"
    }
  ]
}
`)

	reg.Model.SetSchema("model-schema-url")
	xCheckGet(t, reg, "/model", `{
  "schema": "model-schema-url",
  "groups": [
    {
      "plural": "dirs",
      "singular": "dir",
      "schema": "schema-url"
    }
  ]
}
`)

	reg.LoadModel()
	xCheckGet(t, reg, "/model", `{
  "schema": "model-schema-url",
  "groups": [
    {
      "plural": "dirs",
      "singular": "dir",
      "schema": "schema-url"
    }
  ]
}
`)

	reg.Model.SetSchema("")
	xCheckGet(t, reg, "/model", `{
  "groups": [
    {
      "plural": "dirs",
      "singular": "dir",
      "schema": "schema-url"
    }
  ]
}
`)

	// Now error checking
	gm, err = reg.Model.AddGroupModel("dirs1", "", "schema-url") // missing value
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.Model.AddGroupModel("", "", "schema-url") // missing value
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.Model.AddGroupModel("", "", "") // missing value
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.Model.AddGroupModel("", "dir1", "") // missing value
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.Model.AddGroupModel("dirs", "dir", "schema-url") // dup
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.Model.AddGroupModel("dirs1", "dir", "") // dup
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.Model.AddGroupModel("dirs", "dir1", "") // dup
	xCheck(t, gm == nil && err != nil, "gm should have failed")
}

func TestResourceModelCreate(t *testing.T) {
	reg := NewRegistry("TestResourceModels")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "reg created didn't work")

	gm, err := reg.Model.AddGroupModel("dirs", "dir", "dirs-url")
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

	gm2, err := reg.Model.AddGroupModel("dirs2", "dir2", "dirs-url")
	xNoErr(t, err)
	xCheck(t, gm != nil, "gm2 should have worked")

	rm2, err = gm2.AddResourceModel("files", "file", 0, true, true)
	xCheck(t, rm != nil && err == nil, "gm2/rm2 should have worked")

	xCheckGet(t, reg, "/model", `{
  "groups": [
    {
      "plural": "dirs",
      "singular": "dir",
      "schema": "dirs-url",
      "resources": [
        {
          "plural": "files",
          "singular": "file",
          "versions": 5,
          "versionId": true,
          "latest": true
        }
      ]
    },
    {
      "plural": "dirs2",
      "singular": "dir2",
      "schema": "dirs-url",
      "resources": [
        {
          "plural": "files",
          "singular": "file",
          "versions": 0,
          "versionId": true,
          "latest": true
        }
      ]
    }
  ]
}
`)

	rm2.Delete()
	xCheckGet(t, reg, "/model", `{
  "groups": [
    {
      "plural": "dirs",
      "singular": "dir",
      "schema": "dirs-url",
      "resources": [
        {
          "plural": "files",
          "singular": "file",
          "versions": 5,
          "versionId": true,
          "latest": true
        }
      ]
    },
    {
      "plural": "dirs2",
      "singular": "dir2",
      "schema": "dirs-url"
    }
  ]
}
`)

	reg.LoadModel()
	xCheckGet(t, reg, "/model", `{
  "groups": [
    {
      "plural": "dirs",
      "singular": "dir",
      "schema": "dirs-url",
      "resources": [
        {
          "plural": "files",
          "singular": "file",
          "versions": 5,
          "versionId": true,
          "latest": true
        }
      ]
    },
    {
      "plural": "dirs2",
      "singular": "dir2",
      "schema": "dirs-url"
    }
  ]
}
`)

	gm2.Delete()
	xCheckGet(t, reg, "/model", `{
  "groups": [
    {
      "plural": "dirs",
      "singular": "dir",
      "schema": "dirs-url",
      "resources": [
        {
          "plural": "files",
          "singular": "file",
          "versions": 5,
          "versionId": true,
          "latest": true
        }
      ]
    }
  ]
}
`)

	reg.LoadModel()
	xCheckGet(t, reg, "/model", `{
  "groups": [
    {
      "plural": "dirs",
      "singular": "dir",
      "schema": "dirs-url",
      "resources": [
        {
          "plural": "files",
          "singular": "file",
          "versions": 5,
          "versionId": true,
          "latest": true
        }
      ]
    }
  ]
}
`)

	newModel := &registry.Model{
		Groups: map[string]*registry.GroupModel{
			"dirs": &registry.GroupModel{
				Plural:   "dirs",
				Singular: "dir",
				Schema:   "dirs-url",
				Resources: map[string]*registry.ResourceModel{
					"files": &registry.ResourceModel{
						Plural:    "files",
						Singular:  "file",
						Versions:  6,
						VersionId: false,
						Latest:    false,
					},
				},
			},
		},
	}

	reg.Model.ApplyNewModel(newModel)
	xCheckGet(t, reg, "/model", `{
  "groups": [
    {
      "plural": "dirs",
      "singular": "dir",
      "schema": "dirs-url",
      "resources": [
        {
          "plural": "files",
          "singular": "file",
          "versions": 6,
          "versionId": false,
          "latest": false
        }
      ]
    }
  ]
}
`)

	reg.LoadModel()
	g, _ := reg.AddGroup("dirs", "dir1")
	g.AddResource("files", "f1", "v1")

	xCheckGet(t, reg, "?model&inline=dirs.files", `{
  "id": "TestResourceModels",
  "self": "http://localhost:8080/",
  "model": {
    "groups": [
      {
        "plural": "dirs",
        "singular": "dir",
        "schema": "dirs-url",
        "resources": [
          {
            "plural": "files",
            "singular": "file",
            "versions": 6,
            "versionId": false,
            "latest": false
          }
        ]
      }
    ]
  },

  "dirs": {
    "dir1": {
      "id": "dir1",
      "self": "http://localhost:8080/dirs/dir1",

      "files": {
        "f1": {
          "id": "f1",
          "self": "http://localhost:8080/dirs/dir1/files/f1",
          "latestId": "v1",
          "latestUrl": "http://localhost:8080/dirs/dir1/files/f1/versions/v1",

          "versionsCount": 1,
          "versionsUrl": "http://localhost:8080/dirs/dir1/files/f1/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http://localhost:8080/dirs/dir1/files"
    }
  },
  "dirsCount": 1,
  "dirsUrl": "http://localhost:8080/dirs"
}
`)

	newModel = &registry.Model{
		Groups: map[string]*registry.GroupModel{
			"dirs": &registry.GroupModel{
				Plural:   "dirs",
				Singular: "dir",
				Schema:   "dirs-url",
				Resources: map[string]*registry.ResourceModel{
					"files2": &registry.ResourceModel{
						Plural:    "files2",
						Singular:  "file",
						Versions:  6,
						VersionId: false,
						Latest:    false,
					},
				},
			},
		},
	}

	reg.Model.ApplyNewModel(newModel)
	xCheckGet(t, reg, "?model&inline=dirs", `{
  "id": "TestResourceModels",
  "self": "http://localhost:8080/",
  "model": {
    "groups": [
      {
        "plural": "dirs",
        "singular": "dir",
        "schema": "dirs-url",
        "resources": [
          {
            "plural": "files2",
            "singular": "file",
            "versions": 6,
            "versionId": false,
            "latest": false
          }
        ]
      }
    ]
  },

  "dirs": {
    "dir1": {
      "id": "dir1",
      "self": "http://localhost:8080/dirs/dir1",

      "files2Count": 0,
      "files2Url": "http://localhost:8080/dirs/dir1/files2"
    }
  },
  "dirsCount": 1,
  "dirsUrl": "http://localhost:8080/dirs"
}
`)

	newModel = &registry.Model{
		Schema: "reg-model-schema",
		Groups: map[string]*registry.GroupModel{
			"dirs": &registry.GroupModel{
				Plural:   "dirs",
				Singular: "dir",
				Schema:   "dirs-url2",
			},
		},
	}

	reg.Model.ApplyNewModel(newModel)
	xCheckGet(t, reg, "?model&inline=dirs", `{
  "id": "TestResourceModels",
  "self": "http://localhost:8080/",
  "model": {
    "schema": "reg-model-schema",
    "groups": [
      {
        "plural": "dirs",
        "singular": "dir",
        "schema": "dirs-url2"
      }
    ]
  },

  "dirs": {
    "dir1": {
      "id": "dir1",
      "self": "http://localhost:8080/dirs/dir1"
    }
  },
  "dirsCount": 1,
  "dirsUrl": "http://localhost:8080/dirs"
}
`)

	newModel = &registry.Model{
		Groups: map[string]*registry.GroupModel{
			"dirs2": &registry.GroupModel{
				Plural:   "dirs2",
				Singular: "dir2",
				Schema:   "dirs-url",
			},
		},
	}
	reg.Model.ApplyNewModel(newModel)
	xCheckGet(t, reg, "?model&inline=", `{
  "id": "TestResourceModels",
  "self": "http://localhost:8080/",
  "model": {
    "groups": [
      {
        "plural": "dirs2",
        "singular": "dir2",
        "schema": "dirs-url"
      }
    ]
  },

  "dirs2": {},
  "dirs2Count": 0,
  "dirs2Url": "http://localhost:8080/dirs2"
}
`)
}

func TestMultModelCreate(t *testing.T) {
	reg := NewRegistry("TestMultModelCreate")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "reg created didn't work")

	gm1, err := reg.Model.AddGroupModel("gms1", "gm1", "gm1-url")
	xCheck(t, gm1 != nil && err == nil, "gm1 should have worked")

	rm1, err := gm1.AddResourceModel("rms1", "rm1", 0, true, true)
	xCheck(t, rm1 != nil && err == nil, "rm1 should have worked")

	rm2, err := gm1.AddResourceModel("rms2", "rm2", 1, true, true)
	xCheck(t, rm2 != nil && err == nil, "rm2 should have worked")

	gm2, err := reg.Model.AddGroupModel("gms2", "gm2", "gm1-url")
	xCheck(t, gm1 != nil && err == nil, "gm1 should have worked")

	rm21, err := gm2.AddResourceModel("rms1", "rm1", 2, true, true)
	xCheck(t, rm21 != nil && err == nil, "rm21 should have worked")

	rm22, err := gm2.AddResourceModel("rms2", "rm2", 3, true, true)
	xCheck(t, rm22 != nil && err == nil, "rm12 should have worked")

	xCheckGet(t, reg, "/model", `{
  "groups": [
    {
      "plural": "gms1",
      "singular": "gm1",
      "schema": "gm1-url",
      "resources": [
        {
          "plural": "rms1",
          "singular": "rm1",
          "versions": 0,
          "versionId": true,
          "latest": true
        },
        {
          "plural": "rms2",
          "singular": "rm2",
          "versions": 1,
          "versionId": true,
          "latest": true
        }
      ]
    },
    {
      "plural": "gms2",
      "singular": "gm2",
      "schema": "gm1-url",
      "resources": [
        {
          "plural": "rms1",
          "singular": "rm1",
          "versions": 2,
          "versionId": true,
          "latest": true
        },
        {
          "plural": "rms2",
          "singular": "rm2",
          "versions": 3,
          "versionId": true,
          "latest": true
        }
      ]
    }
  ]
}
`)
}

func TestModelAPI(t *testing.T) {
	reg := NewRegistry("TestModelAPI")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "reg created didn't work")

	gm, _ := reg.Model.AddGroupModel("dirs1", "dir1", "")
	gm.AddResourceModel("files", "file", 2, true, false)

	gm2, _ := reg.Model.AddGroupModel("dirs2", "dir2", "")
	gm2.AddResourceModel("files", "file", 0, false, true)

	m := reg.LoadModel()
	xJSONCheck(t, m, reg.Model)
}

func TestMultModel2Create(t *testing.T) {
	reg := NewRegistry("TestMultModel2Create")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "reg created didn't work")

	gm, _ := reg.Model.AddGroupModel("dirs1", "dir1", "")
	gm.AddResourceModel("files", "file", 2, true, false)

	d, _ := reg.AddGroup("dirs1", "d1")
	f, _ := d.AddResource("files", "f1", "v1")
	f.AddVersion("v2")
	d, _ = reg.AddGroup("dirs1", "d2")
	f, _ = d.AddResource("files", "f2", "v1")
	f.AddVersion("v1.1")

	gm2, _ := reg.Model.AddGroupModel("dirs2", "dir2", "")
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
  "self": "http://localhost:8080/",
  "model": {
    "groups": [
      {
        "plural": "dirs1",
        "singular": "dir1",
        "resources": [
          {
            "plural": "files",
            "singular": "file",
            "versions": 2,
            "versionId": true,
            "latest": false
          }
        ]
      },
      {
        "plural": "dirs2",
        "singular": "dir2",
        "resources": [
          {
            "plural": "files",
            "singular": "file",
            "versions": 0,
            "versionId": false,
            "latest": true
          }
        ]
      }
    ]
  },

  "dirs1": {
    "d1": {
      "id": "d1",
      "self": "http://localhost:8080/dirs1/d1",

      "files": {
        "f1": {
          "id": "f1",
          "self": "http://localhost:8080/dirs1/d1/files/f1",
          "latestId": "v2",
          "latestUrl": "http://localhost:8080/dirs1/d1/files/f1/versions/v2",

          "versions": {
            "v1": {
              "id": "v1",
              "self": "http://localhost:8080/dirs1/d1/files/f1/versions/v1"
            },
            "v2": {
              "id": "v2",
              "self": "http://localhost:8080/dirs1/d1/files/f1/versions/v2",
              "latest": true
            }
          },
          "versionsCount": 2,
          "versionsUrl": "http://localhost:8080/dirs1/d1/files/f1/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http://localhost:8080/dirs1/d1/files"
    },
    "d2": {
      "id": "d2",
      "self": "http://localhost:8080/dirs1/d2",

      "files": {
        "f2": {
          "id": "f2",
          "self": "http://localhost:8080/dirs1/d2/files/f2",
          "latestId": "v1.1",
          "latestUrl": "http://localhost:8080/dirs1/d2/files/f2/versions/v1.1",

          "versions": {
            "v1": {
              "id": "v1",
              "self": "http://localhost:8080/dirs1/d2/files/f2/versions/v1"
            },
            "v1.1": {
              "id": "v1.1",
              "self": "http://localhost:8080/dirs1/d2/files/f2/versions/v1.1",
              "latest": true
            }
          },
          "versionsCount": 2,
          "versionsUrl": "http://localhost:8080/dirs1/d2/files/f2/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http://localhost:8080/dirs1/d2/files"
    }
  },
  "dirs1Count": 2,
  "dirs1Url": "http://localhost:8080/dirs1",
  "dirs2": {
    "d2": {
      "id": "d2",
      "self": "http://localhost:8080/dirs2/d2",

      "files": {
        "f2": {
          "id": "f2",
          "self": "http://localhost:8080/dirs2/d2/files/f2",
          "latestId": "v1",
          "latestUrl": "http://localhost:8080/dirs2/d2/files/f2/versions/v1",

          "versions": {
            "v1": {
              "id": "v1",
              "self": "http://localhost:8080/dirs2/d2/files/f2/versions/v1",
              "latest": true
            }
          },
          "versionsCount": 1,
          "versionsUrl": "http://localhost:8080/dirs2/d2/files/f2/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http://localhost:8080/dirs2/d2/files"
    }
  },
  "dirs2Count": 1,
  "dirs2Url": "http://localhost:8080/dirs2"
}
`)

	gm, _ = reg.Model.AddGroupModel("dirs0", "dir0", "")
	gm.AddResourceModel("files", "file", 2, true, false)
	gm, _ = reg.Model.AddGroupModel("dirs3", "dir3", "")
	gm.AddResourceModel("files", "file", 2, true, false)

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs0":{},"dirs1":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}},"dirs2":{"d2":{"files":{"f2":{"versions":{"v1":{}}}}}},"dirs3":{}}`)

	gm, _ = reg.Model.AddGroupModel("dirs15", "dir15", "")
	gm.AddResourceModel("files", "file", 2, true, false)

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs0":{},"dirs1":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}},"dirs15":{},"dirs2":{"d2":{"files":{"f2":{"versions":{"v1":{}}}}}},"dirs3":{}}`)

	gm, _ = reg.Model.AddGroupModel("dirs01", "dir01", "")
	gm, _ = reg.Model.AddGroupModel("dirs02", "dir02", "")
	gm, _ = reg.Model.AddGroupModel("dirs14", "dir014", "")
	gm, _ = reg.Model.AddGroupModel("dirs16", "dir016", "")
	gm, _ = reg.Model.AddGroupModel("dirs4", "dir4", "")
	gm, _ = reg.Model.AddGroupModel("dirs5", "dir5", "")

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs0":{},"dirs01":{},"dirs02":{},"dirs1":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}},"dirs14":{},"dirs15":{},"dirs16":{},"dirs2":{"d2":{"files":{"f2":{"versions":{"v1":{}}}}}},"dirs3":{},"dirs4":{},"dirs5":{}}`)
}
