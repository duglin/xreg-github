package tests

import (
	"testing"
)

func TestExportRoot(t *testing.T) {
	reg := NewRegistry("TestExportRoot")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1$details",
		`{"file": { "hello": "world" }}`, 201, `*`)
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v2$details",
		`{"file": { "hello": "world" }}`, 201, `*`)
	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx$details",
		`{"meta": { "xref": "/dirs/d1/files/f1" }}`, 201, `*`)

	// Full export - 2 different ways
	code, fullBody := xGET(t, "export")
	xCheckEqual(t, "", code, 200)

	code, manualBody := xGET(t, "?compact&inline=*,capabilities,model")
	xCheckEqual(t, "", code, 200)
	xCheckEqual(t, "", fullBody, manualBody)

	xCheckEqual(t, "", fullBody, `{
  "specversion": "0.5",
  "registryid": "TestExportRoot",
  "self": "/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "capabilities": {
    "enforcecompatibility": false,
    "flags": [
      "compact",
      "epoch",
      "filter",
      "inline",
      "nodefaultversionid",
      "nodefaultversionsticky",
      "noepoch",
      "noreadonly",
      "schema",
      "setdefaultversionid",
      "specversion"
    ],
    "mutable": [
      "capabilities",
      "entities",
      "model"
    ],
    "pagination": false,
    "schemas": [
      "xregistry-json/0.5"
    ],
    "shortself": false,
    "specversions": [
      "0.5"
    ]
  },
  "model": {
    "attributes": {
      "specversion": {
        "name": "specversion",
        "type": "string",
        "readonly": true,
        "immutable": true,
        "serverrequired": true
      },
      "registryid": {
        "name": "registryid",
        "type": "string",
        "immutable": true,
        "serverrequired": true
      },
      "self": {
        "name": "self",
        "type": "url",
        "readonly": true,
        "serverrequired": true
      },
      "xid": {
        "name": "xid",
        "type": "url",
        "readonly": true,
        "serverrequired": true
      },
      "epoch": {
        "name": "epoch",
        "type": "uinteger",
        "serverrequired": true
      },
      "name": {
        "name": "name",
        "type": "string"
      },
      "description": {
        "name": "description",
        "type": "string"
      },
      "documentation": {
        "name": "documentation",
        "type": "url"
      },
      "labels": {
        "name": "labels",
        "type": "map",
        "item": {
          "type": "string"
        }
      },
      "createdat": {
        "name": "createdat",
        "type": "timestamp",
        "serverrequired": true
      },
      "modifiedat": {
        "name": "modifiedat",
        "type": "timestamp",
        "serverrequired": true
      }
    },
    "groups": {
      "dirs": {
        "plural": "dirs",
        "singular": "dir",
        "attributes": {
          "dirid": {
            "name": "dirid",
            "type": "string",
            "immutable": true,
            "serverrequired": true
          },
          "self": {
            "name": "self",
            "type": "url",
            "readonly": true,
            "serverrequired": true
          },
          "xid": {
            "name": "xid",
            "type": "url",
            "readonly": true,
            "serverrequired": true
          },
          "epoch": {
            "name": "epoch",
            "type": "uinteger",
            "serverrequired": true
          },
          "name": {
            "name": "name",
            "type": "string"
          },
          "description": {
            "name": "description",
            "type": "string"
          },
          "documentation": {
            "name": "documentation",
            "type": "url"
          },
          "labels": {
            "name": "labels",
            "type": "map",
            "item": {
              "type": "string"
            }
          },
          "createdat": {
            "name": "createdat",
            "type": "timestamp",
            "serverrequired": true
          },
          "modifiedat": {
            "name": "modifiedat",
            "type": "timestamp",
            "serverrequired": true
          }
        },
        "resources": {
          "files": {
            "plural": "files",
            "singular": "file",
            "maxversions": 0,
            "setversionid": true,
            "setdefaultversionsticky": true,
            "hasdocument": true,
            "attributes": {
              "fileid": {
                "name": "fileid",
                "type": "string",
                "immutable": true,
                "serverrequired": true
              },
              "versionid": {
                "name": "versionid",
                "type": "string",
                "immutable": true,
                "serverrequired": true
              },
              "self": {
                "name": "self",
                "type": "url",
                "readonly": true,
                "serverrequired": true
              },
              "xid": {
                "name": "xid",
                "type": "url",
                "readonly": true,
                "serverrequired": true
              },
              "epoch": {
                "name": "epoch",
                "type": "uinteger",
                "serverrequired": true
              },
              "name": {
                "name": "name",
                "type": "string"
              },
              "isdefault": {
                "name": "isdefault",
                "type": "boolean",
                "readonly": true
              },
              "description": {
                "name": "description",
                "type": "string"
              },
              "documentation": {
                "name": "documentation",
                "type": "url"
              },
              "labels": {
                "name": "labels",
                "type": "map",
                "item": {
                  "type": "string"
                }
              },
              "createdat": {
                "name": "createdat",
                "type": "timestamp",
                "serverrequired": true
              },
              "modifiedat": {
                "name": "modifiedat",
                "type": "timestamp",
                "serverrequired": true
              },
              "contenttype": {
                "name": "contenttype",
                "type": "string"
              }
            },
            "metaattributes": {
              "fileid": {
                "name": "fileid",
                "type": "string",
                "immutable": true,
                "serverrequired": true
              },
              "self": {
                "name": "self",
                "type": "url",
                "readonly": true,
                "serverrequired": true
              },
              "xid": {
                "name": "xid",
                "type": "url",
                "readonly": true,
                "serverrequired": true
              },
              "xref": {
                "name": "xref",
                "type": "url"
              },
              "epoch": {
                "name": "epoch",
                "type": "uinteger",
                "serverrequired": true
              },
              "createdat": {
                "name": "createdat",
                "type": "timestamp",
                "serverrequired": true
              },
              "modifiedat": {
                "name": "modifiedat",
                "type": "timestamp",
                "serverrequired": true
              },
              "compatibility": {
                "name": "compatibility",
                "type": "string",
                "enum": [
                  "none",
                  "backward",
                  "backward_transitive",
                  "forward",
                  "forward_transitive",
                  "full",
                  "full_transitive"
                ],
                "strict": false
              },
              "defaultversionid": {
                "name": "defaultversionid",
                "type": "string",
                "serverrequired": true
              },
              "defaultversionurl": {
                "name": "defaultversionurl",
                "type": "url",
                "readonly": true,
                "serverrequired": true
              },
              "defaultversionsticky": {
                "name": "defaultversionsticky",
                "type": "boolean",
                "readonly": true
              }
            }
          }
        }
      }
    }
  },

  "dirsurl": "/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "self": "/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",

          "metaurl": "/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 2,
            "createdat": "2025-01-01T12:00:02Z",
            "modifiedat": "2025-01-01T12:00:04Z",

            "defaultversionid": "v2",
            "defaultversionurl": "/dirs/d1/files/f1/versions/v2$details"
          },
          "versionsurl": "/dirs/d1/files/f1/versions",
          "versions": {
            "v1": {
              "fileid": "f1",
              "versionid": "v1",
              "self": "/dirs/d1/files/f1/versions/v1$details",
              "xid": "/dirs/d1/files/f1/versions/v1",
              "epoch": 1,
              "createdat": "2025-01-01T12:00:02Z",
              "modifiedat": "2025-01-01T12:00:02Z",
              "contenttype": "application/json",
              "file": {
                "hello": "world"
              }
            },
            "v2": {
              "fileid": "f1",
              "versionid": "v2",
              "self": "/dirs/d1/files/f1/versions/v2$details",
              "xid": "/dirs/d1/files/f1/versions/v2",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2025-01-01T12:00:04Z",
              "modifiedat": "2025-01-01T12:00:04Z",
              "contenttype": "application/json",
              "file": {
                "hello": "world"
              }
            }
          },
          "versionscount": 2
        },
        "fx": {
          "fileid": "fx",
          "self": "/dirs/d1/files/fx$details",
          "xid": "/dirs/d1/files/fx",

          "metaurl": "/dirs/d1/files/fx/meta",
          "meta": {
            "fileid": "fx",
            "self": "/dirs/d1/files/fx/meta",
            "xid": "/dirs/d1/files/fx/meta",
            "xref": "/dirs/d1/files/f1"
          }
        }
      },
      "filescount": 2
    }
  },
  "dirscount": 1
}
`)

	// Play with ?export vanilla
	code, fullBody = xGET(t, "export?inline=*")
	xCheckEqual(t, "", code, 200)
	code, manualBody = xGET(t, "?compact&inline=*")
	xCheckEqual(t, "", code, 200)
	xCheckEqual(t, "", fullBody, manualBody)

	xCheckEqual(t, "", fullBody, `{
  "specversion": "0.5",
  "registryid": "TestExportRoot",
  "self": "/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "self": "/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",

          "metaurl": "/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 2,
            "createdat": "2025-01-01T12:00:02Z",
            "modifiedat": "2025-01-01T12:00:04Z",

            "defaultversionid": "v2",
            "defaultversionurl": "/dirs/d1/files/f1/versions/v2$details"
          },
          "versionsurl": "/dirs/d1/files/f1/versions",
          "versions": {
            "v1": {
              "fileid": "f1",
              "versionid": "v1",
              "self": "/dirs/d1/files/f1/versions/v1$details",
              "xid": "/dirs/d1/files/f1/versions/v1",
              "epoch": 1,
              "createdat": "2025-01-01T12:00:02Z",
              "modifiedat": "2025-01-01T12:00:02Z",
              "contenttype": "application/json",
              "file": {
                "hello": "world"
              }
            },
            "v2": {
              "fileid": "f1",
              "versionid": "v2",
              "self": "/dirs/d1/files/f1/versions/v2$details",
              "xid": "/dirs/d1/files/f1/versions/v2",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2025-01-01T12:00:04Z",
              "modifiedat": "2025-01-01T12:00:04Z",
              "contenttype": "application/json",
              "file": {
                "hello": "world"
              }
            }
          },
          "versionscount": 2
        },
        "fx": {
          "fileid": "fx",
          "self": "/dirs/d1/files/fx$details",
          "xid": "/dirs/d1/files/fx",

          "metaurl": "/dirs/d1/files/fx/meta",
          "meta": {
            "fileid": "fx",
            "self": "/dirs/d1/files/fx/meta",
            "xid": "/dirs/d1/files/fx/meta",
            "xref": "/dirs/d1/files/f1"
          }
        }
      },
      "filescount": 2
    }
  },
  "dirscount": 1
}
`)

	// Play with ?compact inline just capabilities
	code, fullBody = xGET(t, "export?inline=capabilities")
	xCheckEqual(t, "", code, 200)
	code, manualBody = xGET(t, "?compact&inline=capabilities")
	xCheckEqual(t, "", code, 200)
	xCheckEqual(t, "", fullBody, manualBody)

	xCheckEqual(t, "", fullBody, `{
  "specversion": "0.5",
  "registryid": "TestExportRoot",
  "self": "/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "capabilities": {
    "enforcecompatibility": false,
    "flags": [
      "compact",
      "epoch",
      "filter",
      "inline",
      "nodefaultversionid",
      "nodefaultversionsticky",
      "noepoch",
      "noreadonly",
      "schema",
      "setdefaultversionid",
      "specversion"
    ],
    "mutable": [
      "capabilities",
      "entities",
      "model"
    ],
    "pagination": false,
    "schemas": [
      "xregistry-json/0.5"
    ],
    "shortself": false,
    "specversions": [
      "0.5"
    ]
  },

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	// Play with ?compact inline just model
	code, fullBody = xGET(t, "export?inline=model")
	xCheckEqual(t, "", code, 200)
	code, manualBody = xGET(t, "?compact&inline=model")
	xCheckEqual(t, "", code, 200)
	xCheckEqual(t, "", fullBody, manualBody)

	xCheckEqual(t, "", fullBody, `{
  "specversion": "0.5",
  "registryid": "TestExportRoot",
  "self": "/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "model": {
    "attributes": {
      "specversion": {
        "name": "specversion",
        "type": "string",
        "readonly": true,
        "immutable": true,
        "serverrequired": true
      },
      "registryid": {
        "name": "registryid",
        "type": "string",
        "immutable": true,
        "serverrequired": true
      },
      "self": {
        "name": "self",
        "type": "url",
        "readonly": true,
        "serverrequired": true
      },
      "xid": {
        "name": "xid",
        "type": "url",
        "readonly": true,
        "serverrequired": true
      },
      "epoch": {
        "name": "epoch",
        "type": "uinteger",
        "serverrequired": true
      },
      "name": {
        "name": "name",
        "type": "string"
      },
      "description": {
        "name": "description",
        "type": "string"
      },
      "documentation": {
        "name": "documentation",
        "type": "url"
      },
      "labels": {
        "name": "labels",
        "type": "map",
        "item": {
          "type": "string"
        }
      },
      "createdat": {
        "name": "createdat",
        "type": "timestamp",
        "serverrequired": true
      },
      "modifiedat": {
        "name": "modifiedat",
        "type": "timestamp",
        "serverrequired": true
      }
    },
    "groups": {
      "dirs": {
        "plural": "dirs",
        "singular": "dir",
        "attributes": {
          "dirid": {
            "name": "dirid",
            "type": "string",
            "immutable": true,
            "serverrequired": true
          },
          "self": {
            "name": "self",
            "type": "url",
            "readonly": true,
            "serverrequired": true
          },
          "xid": {
            "name": "xid",
            "type": "url",
            "readonly": true,
            "serverrequired": true
          },
          "epoch": {
            "name": "epoch",
            "type": "uinteger",
            "serverrequired": true
          },
          "name": {
            "name": "name",
            "type": "string"
          },
          "description": {
            "name": "description",
            "type": "string"
          },
          "documentation": {
            "name": "documentation",
            "type": "url"
          },
          "labels": {
            "name": "labels",
            "type": "map",
            "item": {
              "type": "string"
            }
          },
          "createdat": {
            "name": "createdat",
            "type": "timestamp",
            "serverrequired": true
          },
          "modifiedat": {
            "name": "modifiedat",
            "type": "timestamp",
            "serverrequired": true
          }
        },
        "resources": {
          "files": {
            "plural": "files",
            "singular": "file",
            "maxversions": 0,
            "setversionid": true,
            "setdefaultversionsticky": true,
            "hasdocument": true,
            "attributes": {
              "fileid": {
                "name": "fileid",
                "type": "string",
                "immutable": true,
                "serverrequired": true
              },
              "versionid": {
                "name": "versionid",
                "type": "string",
                "immutable": true,
                "serverrequired": true
              },
              "self": {
                "name": "self",
                "type": "url",
                "readonly": true,
                "serverrequired": true
              },
              "xid": {
                "name": "xid",
                "type": "url",
                "readonly": true,
                "serverrequired": true
              },
              "epoch": {
                "name": "epoch",
                "type": "uinteger",
                "serverrequired": true
              },
              "name": {
                "name": "name",
                "type": "string"
              },
              "isdefault": {
                "name": "isdefault",
                "type": "boolean",
                "readonly": true
              },
              "description": {
                "name": "description",
                "type": "string"
              },
              "documentation": {
                "name": "documentation",
                "type": "url"
              },
              "labels": {
                "name": "labels",
                "type": "map",
                "item": {
                  "type": "string"
                }
              },
              "createdat": {
                "name": "createdat",
                "type": "timestamp",
                "serverrequired": true
              },
              "modifiedat": {
                "name": "modifiedat",
                "type": "timestamp",
                "serverrequired": true
              },
              "contenttype": {
                "name": "contenttype",
                "type": "string"
              }
            },
            "metaattributes": {
              "fileid": {
                "name": "fileid",
                "type": "string",
                "immutable": true,
                "serverrequired": true
              },
              "self": {
                "name": "self",
                "type": "url",
                "readonly": true,
                "serverrequired": true
              },
              "xid": {
                "name": "xid",
                "type": "url",
                "readonly": true,
                "serverrequired": true
              },
              "xref": {
                "name": "xref",
                "type": "url"
              },
              "epoch": {
                "name": "epoch",
                "type": "uinteger",
                "serverrequired": true
              },
              "createdat": {
                "name": "createdat",
                "type": "timestamp",
                "serverrequired": true
              },
              "modifiedat": {
                "name": "modifiedat",
                "type": "timestamp",
                "serverrequired": true
              },
              "compatibility": {
                "name": "compatibility",
                "type": "string",
                "enum": [
                  "none",
                  "backward",
                  "backward_transitive",
                  "forward",
                  "forward_transitive",
                  "full",
                  "full_transitive"
                ],
                "strict": false
              },
              "defaultversionid": {
                "name": "defaultversionid",
                "type": "string",
                "serverrequired": true
              },
              "defaultversionurl": {
                "name": "defaultversionurl",
                "type": "url",
                "readonly": true,
                "serverrequired": true
              },
              "defaultversionsticky": {
                "name": "defaultversionsticky",
                "type": "boolean",
                "readonly": true
              }
            }
          }
        }
      }
    }
  },

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	// Play with ?compact not at root
	xHTTP(t, reg, "GET", "/dirs?compact&inline=*", ``, 200, `{
  "d1": {
    "dirid": "d1",
    "self": "/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:02Z",
    "modifiedat": "2025-01-01T12:00:03Z",

    "filesurl": "/dirs/d1/files",
    "files": {
      "f1": {
        "fileid": "f1",
        "self": "/dirs/d1/files/f1$details",
        "xid": "/dirs/d1/files/f1",

        "metaurl": "/dirs/d1/files/f1/meta",
        "meta": {
          "fileid": "f1",
          "self": "/dirs/d1/files/f1/meta",
          "xid": "/dirs/d1/files/f1/meta",
          "epoch": 2,
          "createdat": "2025-01-01T12:00:02Z",
          "modifiedat": "2025-01-01T12:00:04Z",

          "defaultversionid": "v2",
          "defaultversionurl": "/dirs/d1/files/f1/versions/v2$details"
        },
        "versionsurl": "/dirs/d1/files/f1/versions",
        "versions": {
          "v1": {
            "fileid": "f1",
            "versionid": "v1",
            "self": "/dirs/d1/files/f1/versions/v1$details",
            "xid": "/dirs/d1/files/f1/versions/v1",
            "epoch": 1,
            "createdat": "2025-01-01T12:00:02Z",
            "modifiedat": "2025-01-01T12:00:02Z",
            "contenttype": "application/json",
            "file": {
              "hello": "world"
            }
          },
          "v2": {
            "fileid": "f1",
            "versionid": "v2",
            "self": "/dirs/d1/files/f1/versions/v2$details",
            "xid": "/dirs/d1/files/f1/versions/v2",
            "epoch": 1,
            "isdefault": true,
            "createdat": "2025-01-01T12:00:04Z",
            "modifiedat": "2025-01-01T12:00:04Z",
            "contenttype": "application/json",
            "file": {
              "hello": "world"
            }
          }
        },
        "versionscount": 2
      },
      "fx": {
        "fileid": "fx",
        "self": "/dirs/d1/files/fx$details",
        "xid": "/dirs/d1/files/fx",

        "metaurl": "/dirs/d1/files/fx/meta",
        "meta": {
          "fileid": "fx",
          "self": "/dirs/d1/files/fx/meta",
          "xid": "/dirs/d1/files/fx/meta",
          "xref": "/dirs/d1/files/f1"
        }
      }
    },
    "filescount": 2
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1?compact&inline=*", ``, 200, `{
  "dirid": "d1",
  "self": "/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:02Z",
  "modifiedat": "2025-01-01T12:00:03Z",

  "filesurl": "/dirs/d1/files",
  "files": {
    "f1": {
      "fileid": "f1",
      "self": "/dirs/d1/files/f1$details",
      "xid": "/dirs/d1/files/f1",

      "metaurl": "/dirs/d1/files/f1/meta",
      "meta": {
        "fileid": "f1",
        "self": "/dirs/d1/files/f1/meta",
        "xid": "/dirs/d1/files/f1/meta",
        "epoch": 2,
        "createdat": "2025-01-01T12:00:02Z",
        "modifiedat": "2025-01-01T12:00:04Z",

        "defaultversionid": "v2",
        "defaultversionurl": "/dirs/d1/files/f1/versions/v2$details"
      },
      "versionsurl": "/dirs/d1/files/f1/versions",
      "versions": {
        "v1": {
          "fileid": "f1",
          "versionid": "v1",
          "self": "/dirs/d1/files/f1/versions/v1$details",
          "xid": "/dirs/d1/files/f1/versions/v1",
          "epoch": 1,
          "createdat": "2025-01-01T12:00:02Z",
          "modifiedat": "2025-01-01T12:00:02Z",
          "contenttype": "application/json",
          "file": {
            "hello": "world"
          }
        },
        "v2": {
          "fileid": "f1",
          "versionid": "v2",
          "self": "/dirs/d1/files/f1/versions/v2$details",
          "xid": "/dirs/d1/files/f1/versions/v2",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2025-01-01T12:00:04Z",
          "modifiedat": "2025-01-01T12:00:04Z",
          "contenttype": "application/json",
          "file": {
            "hello": "world"
          }
        }
      },
      "versionscount": 2
    },
    "fx": {
      "fileid": "fx",
      "self": "/dirs/d1/files/fx$details",
      "xid": "/dirs/d1/files/fx",

      "metaurl": "/dirs/d1/files/fx/meta",
      "meta": {
        "fileid": "fx",
        "self": "/dirs/d1/files/fx/meta",
        "xid": "/dirs/d1/files/fx/meta",
        "xref": "/dirs/d1/files/f1"
      }
    }
  },
  "filescount": 2
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files?compact&inline=*", ``, 200, `{
  "f1": {
    "fileid": "f1",
    "self": "/dirs/d1/files/f1$details",
    "xid": "/dirs/d1/files/f1",

    "metaurl": "/dirs/d1/files/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "/dirs/d1/files/f1/meta",
      "xid": "/dirs/d1/files/f1/meta",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:04Z",

      "defaultversionid": "v2",
      "defaultversionurl": "/dirs/d1/files/f1/versions/v2$details"
    },
    "versionsurl": "/dirs/d1/files/f1/versions",
    "versions": {
      "v1": {
        "fileid": "f1",
        "versionid": "v1",
        "self": "/dirs/d1/files/f1/versions/v1$details",
        "xid": "/dirs/d1/files/f1/versions/v1",
        "epoch": 1,
        "createdat": "2025-01-01T12:00:02Z",
        "modifiedat": "2025-01-01T12:00:02Z",
        "contenttype": "application/json",
        "file": {
          "hello": "world"
        }
      },
      "v2": {
        "fileid": "f1",
        "versionid": "v2",
        "self": "/dirs/d1/files/f1/versions/v2$details",
        "xid": "/dirs/d1/files/f1/versions/v2",
        "epoch": 1,
        "isdefault": true,
        "createdat": "2025-01-01T12:00:04Z",
        "modifiedat": "2025-01-01T12:00:04Z",
        "contenttype": "application/json",
        "file": {
          "hello": "world"
        }
      }
    },
    "versionscount": 2
  },
  "fx": {
    "fileid": "fx",
    "self": "/dirs/d1/files/fx$details",
    "xid": "/dirs/d1/files/fx",

    "metaurl": "/dirs/d1/files/fx/meta",
    "meta": {
      "fileid": "fx",
      "self": "/dirs/d1/files/fx/meta",
      "xid": "/dirs/d1/files/fx/meta",
      "xref": "/dirs/d1/files/f1"
    }
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?compact&inline=*", ``, 200, `{
  "fileid": "f1",
  "self": "/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",

  "metaurl": "/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:02Z",
    "modifiedat": "2025-01-01T12:00:04Z",

    "defaultversionid": "v2",
    "defaultversionurl": "/dirs/d1/files/f1/versions/v2$details"
  },
  "versionsurl": "/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "/dirs/d1/files/f1/versions/v1$details",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:02Z",
      "contenttype": "application/json",
      "file": {
        "hello": "world"
      }
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "/dirs/d1/files/f1/versions/v2$details",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2025-01-01T12:00:04Z",
      "modifiedat": "2025-01-01T12:00:04Z",
      "contenttype": "application/json",
      "file": {
        "hello": "world"
      }
    }
  },
  "versionscount": 2
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta?compact&inline=*", ``, 200, `{
  "fileid": "f1",
  "self": "/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:02Z",
  "modifiedat": "2025-01-01T12:00:04Z",

  "defaultversionid": "v2",
  "defaultversionurl": "/dirs/d1/files/f1/versions/v2$details"
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/fx?compact&inline=*", ``, 200, `{
  "fileid": "fx",
  "self": "/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",

  "metaurl": "/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "xref": "/dirs/d1/files/f1"
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/fx/meta?compact&inline=*", ``, 200, `{
  "fileid": "fx",
  "self": "/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1"
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions?compact&inline=*", ``, 200, `{
  "v1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "/dirs/d1/files/f1/versions/v1$details",
    "xid": "/dirs/d1/files/f1/versions/v1",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:02Z",
    "modifiedat": "2025-01-01T12:00:02Z",
    "contenttype": "application/json",
    "file": {
      "hello": "world"
    }
  },
  "v2": {
    "fileid": "f1",
    "versionid": "v2",
    "self": "/dirs/d1/files/f1/versions/v2$details",
    "xid": "/dirs/d1/files/f1/versions/v2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2025-01-01T12:00:04Z",
    "modifiedat": "2025-01-01T12:00:04Z",
    "contenttype": "application/json",
    "file": {
      "hello": "world"
    }
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/v1?compact&inline=*", ``, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "/dirs/d1/files/f1/versions/v1$details",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:02Z",
  "modifiedat": "2025-01-01T12:00:02Z",
  "contenttype": "application/json",
  "file": {
    "hello": "world"
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/fx/versions?compact", ``, 400,
		"'compact' flag not allowed on xref'd Versions\n")

	xHTTP(t, reg, "GET", "/dirs/d1/files/fx/versions/v1?compact", ``, 400,
		"'compact' flag not allowed on xref'd Versions\n")

	// Just some filtering too for fun
	code, fullBody = xGET(t, "export?filter=dirs.files.versions.versionid=v2&inline=*")
	xCheckEqual(t, "", code, 200)
	code, manualBody = xGET(t, "?compact&inline=*&filter=dirs.files.versions.versionid=v2")
	xCheckEqual(t, "", code, 200)
	xCheckEqual(t, "", fullBody, manualBody)

	xCheckEqual(t, "", fullBody, `{
  "specversion": "0.5",
  "registryid": "TestExportRoot",
  "self": "/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "self": "/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",

          "metaurl": "/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 2,
            "createdat": "2025-01-01T12:00:02Z",
            "modifiedat": "2025-01-01T12:00:04Z",

            "defaultversionid": "v2",
            "defaultversionurl": "/dirs/d1/files/f1/versions/v2$details"
          },
          "versionsurl": "/dirs/d1/files/f1/versions",
          "versions": {
            "v2": {
              "fileid": "f1",
              "versionid": "v2",
              "self": "/dirs/d1/files/f1/versions/v2$details",
              "xid": "/dirs/d1/files/f1/versions/v2",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2025-01-01T12:00:04Z",
              "modifiedat": "2025-01-01T12:00:04Z",
              "contenttype": "application/json",
              "file": {
                "hello": "world"
              }
            }
          },
          "versionscount": 1
        },
        "fx": {
          "fileid": "fx",
          "self": "/dirs/d1/files/fx$details",
          "xid": "/dirs/d1/files/fx",

          "metaurl": "/dirs/d1/files/fx/meta",
          "meta": {
            "fileid": "fx",
            "self": "/dirs/d1/files/fx/meta",
            "xid": "/dirs/d1/files/fx/meta",
            "xref": "/dirs/d1/files/f1"
          }
        }
      },
      "filescount": 2
    }
  },
  "dirscount": 1
}
`)

	code, fullBody = xGET(t, "export?filter=dirs.files.versions.versionid=vx&inline=*")
	xCheckEqual(t, "", code, 404)
	code, manualBody = xGET(t, "?compact&inline=*&filter=dirs.files.versions.versionid=vx")
	xCheckEqual(t, "", code, 404)
	xCheckEqual(t, "", fullBody, manualBody)
	xCheckEqual(t, "", fullBody, "Not found\n")

	code, fullBody = xGET(t, "export?filter=dirs.files.versions.versionid=v2,dirs.files.fileid=fx&inline=*")
	xCheckEqual(t, "", code, 200)
	code, manualBody = xGET(t, "?compact&inline=*&filter=dirs.files.versions.versionid=v2,dirs.files.fileid=fx")
	xCheckEqual(t, "", code, 200)
	xCheckEqual(t, "", fullBody, manualBody)

	xCheckEqual(t, "", fullBody, `{
  "specversion": "0.5",
  "registryid": "TestExportRoot",
  "self": "/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "/dirs/d1/files",
      "files": {
        "fx": {
          "fileid": "fx",
          "self": "/dirs/d1/files/fx$details",
          "xid": "/dirs/d1/files/fx",

          "metaurl": "/dirs/d1/files/fx/meta",
          "meta": {
            "fileid": "fx",
            "self": "/dirs/d1/files/fx/meta",
            "xid": "/dirs/d1/files/fx/meta",
            "xref": "/dirs/d1/files/f1"
          }
        }
      },
      "filescount": 1
    }
  },
  "dirscount": 1
}
`)

	// Make sure that ?compact doesn't turn on ?inline by mistake.
	// At one point ?export (?compact) implied ?inline=*

	xHTTP(t, reg, "GET", "?compact", ``, 200,
		`{
  "specversion": "0.5",
  "registryid": "TestExportRoot",
  "self": "/",
  "xid": "/",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/dirs?compact", ``, 200,
		`{
  "d1": {
    "dirid": "d1",
    "self": "/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 2,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "filescount": 2
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1?compact", ``, 200,
		`{
  "dirid": "d1",
  "self": "/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "filescount": 2
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files?compact", ``, 200,
		`{
  "f1": {
    "fileid": "f1",
    "self": "/dirs/d1/files/f1$details",
    "xid": "/dirs/d1/files/f1",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versionscount": 2
  },
  "fx": {
    "fileid": "fx",
    "self": "/dirs/d1/files/fx$details",
    "xid": "/dirs/d1/files/fx",

    "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta"
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?compact", ``, 200,
		`{
  "fileid": "f1",
  "self": "/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta?compact", ``, 200,
		`{
  "fileid": "f1",
  "self": "/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details"
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions?compact", ``, 200,
		`{
  "v1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "/dirs/d1/files/f1/versions/v1$details",
    "xid": "/dirs/d1/files/f1/versions/v1",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "contenttype": "application/json"
  },
  "v2": {
    "fileid": "f1",
    "versionid": "v2",
    "self": "/dirs/d1/files/f1/versions/v2$details",
    "xid": "/dirs/d1/files/f1/versions/v2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "contenttype": "application/json"
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/v1?compact", ``, 200,
		`{
  "fileid": "f1",
  "versionid": "v1",
  "self": "/dirs/d1/files/f1/versions/v1$details",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "contenttype": "application/json"
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/fx?compact", ``, 200,
		`{
  "fileid": "fx",
  "self": "/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta"
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/fx/meta?compact", ``, 200,
		`{
  "fileid": "fx",
  "self": "/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1"
}
`)

	// Test some error cases. Make sure ?compact doesn't change our
	// error checking logic

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/v1/foo?compact", ``, 404,
		"URL is too long\n")

	xHTTP(t, reg, "GET", "/dirs/d1/files/fx/versions/v1/foo?compact", ``, 404,
		"URL is too long\n")

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/vx?compact", ``, 404,
		"Not found\n")

	xHTTP(t, reg, "GET", "/dirs/d1/files/fz/versions?compact", ``, 404,
		"Not found\n")

	xHTTP(t, reg, "GET", "/dirs/d1/files/fz?compact", ``, 404,
		"Not found\n")

	xHTTP(t, reg, "GET", "/dirs/dx/files?compact", ``, 404,
		"Not found\n")

	xHTTP(t, reg, "GET", "/dirs/d1/filesx?compact", ``, 404,
		"Unknown Resource type: filesx\n")

	xHTTP(t, reg, "GET", "/dirs/dx?compact", ``, 404,
		"Not found\n")

	xHTTP(t, reg, "GET", "/dirsx?compact", ``, 404,
		"Unknown Group type: dirsx\n")

	xHTTP(t, reg, "GET", "/dirs/dx/files/fz/versions/vx?compact", ``, 404,
		"Not found\n")

}

func TestExportURLs(t *testing.T) {
	reg := NewRegistry("TestExportURLs")
	defer PassDeleteReg(t, reg)

	gm, _, err := reg.Model.CreateModels("dirs", "dir", "files", "file")
	xNoErr(t, err)
	_, err = gm.AddResourceModelSimple("schemas", "schema")
	xNoErr(t, err)

	xHTTP(t, reg, "GET", "/?compact", "", 200, `{
  "specversion": "0.5",
  "registryid": "TestExportURLs",
  "self": "/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 0
}
`)

	xHTTP(t, reg, "GET", "/?compact&inline=dirs", "", 200, `{
  "specversion": "0.5",
  "registryid": "TestExportURLs",
  "self": "/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",

  "dirsurl": "/dirs",
  "dirs": {},
  "dirscount": 0
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1", "", 201, `*`)

	xHTTP(t, reg, "GET", "/?compact&inline=dirs", "", 200, `{
  "specversion": "0.5",
  "registryid": "TestExportURLs",
  "self": "/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "filescount": 0,
      "schemasurl": "http://localhost:8181/dirs/d1/schemas",
      "schemascount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/dirs?compact&inline=files", "", 200, `{
  "d1": {
    "dirid": "d1",
    "self": "/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:02Z",
    "modifiedat": "2025-01-01T12:00:02Z",

    "filesurl": "/dirs/d1/files",
    "files": {},
    "filescount": 0,
    "schemasurl": "http://localhost:8181/dirs/d1/schemas",
    "schemascount": 0
  }
}
`)

	xHTTP(t, reg, "GET", "/?compact&inline=dirs.files", "", 200, `{
  "specversion": "0.5",
  "registryid": "TestExportURLs",
  "self": "/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:02Z",

      "filesurl": "/dirs/d1/files",
      "files": {},
      "filescount": 0,
      "schemasurl": "http://localhost:8181/dirs/d1/schemas",
      "schemascount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/dirs?compact", "", 200, `{
  "d1": {
    "dirid": "d1",
    "self": "/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:02Z",
    "modifiedat": "2025-01-01T12:00:02Z",

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "filescount": 0,
    "schemasurl": "http://localhost:8181/dirs/d1/schemas",
    "schemascount": 0
  }
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1", "", 201, `*`)

	xHTTP(t, reg, "GET", "/?compact&inline=dirs.files", "", 200, `{
  "specversion": "0.5",
  "registryid": "TestExportURLs",
  "self": "/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "self": "/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
          "versionscount": 1
        }
      },
      "filescount": 1,
      "schemasurl": "http://localhost:8181/dirs/d1/schemas",
      "schemascount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/?compact&inline=dirs.files.meta", "", 200, `{
  "specversion": "0.5",
  "registryid": "TestExportURLs",
  "self": "/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "self": "/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",

          "metaurl": "/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "2025-01-01T12:00:03Z",
            "modifiedat": "2025-01-01T12:00:03Z",

            "defaultversionid": "1",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details"
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
          "versionscount": 1
        }
      },
      "filescount": 1,
      "schemasurl": "http://localhost:8181/dirs/d1/schemas",
      "schemascount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/?compact&inline=dirs.files.versions", "", 200, `{
  "specversion": "0.5",
  "registryid": "TestExportURLs",
  "self": "/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "self": "/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "versionsurl": "/dirs/d1/files/f1/versions",
          "versions": {
            "1": {
              "fileid": "f1",
              "versionid": "1",
              "self": "/dirs/d1/files/f1/versions/1$details",
              "xid": "/dirs/d1/files/f1/versions/1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2025-01-01T12:00:03Z",
              "modifiedat": "2025-01-01T12:00:03Z"
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 1,
      "schemasurl": "http://localhost:8181/dirs/d1/schemas",
      "schemascount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/?compact&inline=dirs.files.versions,dirs.files.meta", "", 200, `{
  "specversion": "0.5",
  "registryid": "TestExportURLs",
  "self": "/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "self": "/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",

          "metaurl": "/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "2025-01-01T12:00:03Z",
            "modifiedat": "2025-01-01T12:00:03Z",

            "defaultversionid": "1",
            "defaultversionurl": "/dirs/d1/files/f1/versions/1$details"
          },
          "versionsurl": "/dirs/d1/files/f1/versions",
          "versions": {
            "1": {
              "fileid": "f1",
              "versionid": "1",
              "self": "/dirs/d1/files/f1/versions/1$details",
              "xid": "/dirs/d1/files/f1/versions/1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2025-01-01T12:00:03Z",
              "modifiedat": "2025-01-01T12:00:03Z"
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 1,
      "schemasurl": "http://localhost:8181/dirs/d1/schemas",
      "schemascount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files?compact&inline=versions,meta", "", 200, `{
  "f1": {
    "fileid": "f1",
    "self": "/dirs/d1/files/f1$details",
    "xid": "/dirs/d1/files/f1",

    "metaurl": "/dirs/d1/files/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "/dirs/d1/files/f1/meta",
      "xid": "/dirs/d1/files/f1/meta",
      "epoch": 1,
      "createdat": "2025-01-01T12:00:03Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "defaultversionid": "1",
      "defaultversionurl": "/dirs/d1/files/f1/versions/1$details"
    },
    "versionsurl": "/dirs/d1/files/f1/versions",
    "versions": {
      "1": {
        "fileid": "f1",
        "versionid": "1",
        "self": "/dirs/d1/files/f1/versions/1$details",
        "xid": "/dirs/d1/files/f1/versions/1",
        "epoch": 1,
        "isdefault": true,
        "createdat": "2025-01-01T12:00:03Z",
        "modifiedat": "2025-01-01T12:00:03Z"
      }
    },
    "versionscount": 1
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files?compact&inline=meta", "", 200, `{
  "f1": {
    "fileid": "f1",
    "self": "/dirs/d1/files/f1$details",
    "xid": "/dirs/d1/files/f1",

    "metaurl": "/dirs/d1/files/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "/dirs/d1/files/f1/meta",
      "xid": "/dirs/d1/files/f1/meta",
      "epoch": 1,
      "createdat": "2025-01-01T12:00:03Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "defaultversionid": "1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details"
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versionscount": 1
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files?compact&inline=versions", "", 200, `{
  "f1": {
    "fileid": "f1",
    "self": "/dirs/d1/files/f1$details",
    "xid": "/dirs/d1/files/f1",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "versionsurl": "/dirs/d1/files/f1/versions",
    "versions": {
      "1": {
        "fileid": "f1",
        "versionid": "1",
        "self": "/dirs/d1/files/f1/versions/1$details",
        "xid": "/dirs/d1/files/f1/versions/1",
        "epoch": 1,
        "isdefault": true,
        "createdat": "2025-01-01T12:00:03Z",
        "modifiedat": "2025-01-01T12:00:03Z"
      }
    },
    "versionscount": 1
  }
}
`)

	xHTTP(t, reg, "GET", "/?compact&inline=dirs.files.meta", "", 200, `{
  "specversion": "0.5",
  "registryid": "TestExportURLs",
  "self": "/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "self": "/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",

          "metaurl": "/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "2025-01-01T12:00:03Z",
            "modifiedat": "2025-01-01T12:00:03Z",

            "defaultversionid": "1",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details"
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
          "versionscount": 1
        }
      },
      "filescount": 1,
      "schemasurl": "http://localhost:8181/dirs/d1/schemas",
      "schemascount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/?compact&inline=dirs.files.versions", "", 200, `{
  "specversion": "0.5",
  "registryid": "TestExportURLs",
  "self": "/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "self": "/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "versionsurl": "/dirs/d1/files/f1/versions",
          "versions": {
            "1": {
              "fileid": "f1",
              "versionid": "1",
              "self": "/dirs/d1/files/f1/versions/1$details",
              "xid": "/dirs/d1/files/f1/versions/1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2025-01-01T12:00:03Z",
              "modifiedat": "2025-01-01T12:00:03Z"
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 1,
      "schemasurl": "http://localhost:8181/dirs/d1/schemas",
      "schemascount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta?compact", "", 200, `{
  "fileid": "f1",
  "self": "/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:03Z",
  "modifiedat": "2025-01-01T12:00:03Z",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details"
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions?compact", "", 200, `{
  "1": {
    "fileid": "f1",
    "versionid": "1",
    "self": "/dirs/d1/files/f1/versions/1$details",
    "xid": "/dirs/d1/files/f1/versions/1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2025-01-01T12:00:03Z",
    "modifiedat": "2025-01-01T12:00:03Z"
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/1?compact", "", 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "/dirs/d1/files/f1/versions/1$details",
  "xid": "/dirs/d1/files/f1/versions/1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:03Z",
  "modifiedat": "2025-01-01T12:00:03Z"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/fx?compact&inline=*", "", 200, `{
  "fileid": "fx",
  "self": "/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",

  "metaurl": "/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "xref": "/dirs/d1/files/f1"
  }
}
`)

	// One file GET of everything

	xHTTP(t, reg, "GET", "/?compact&inline=*", "", 200, `{
  "specversion": "0.5",
  "registryid": "TestExportURLs",
  "self": "/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 3,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "self": "/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",

          "metaurl": "/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "2025-01-01T12:00:04Z",
            "modifiedat": "2025-01-01T12:00:04Z",

            "defaultversionid": "1",
            "defaultversionurl": "/dirs/d1/files/f1/versions/1$details"
          },
          "versionsurl": "/dirs/d1/files/f1/versions",
          "versions": {
            "1": {
              "fileid": "f1",
              "versionid": "1",
              "self": "/dirs/d1/files/f1/versions/1$details",
              "xid": "/dirs/d1/files/f1/versions/1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2025-01-01T12:00:04Z",
              "modifiedat": "2025-01-01T12:00:04Z"
            }
          },
          "versionscount": 1
        },
        "fx": {
          "fileid": "fx",
          "self": "/dirs/d1/files/fx$details",
          "xid": "/dirs/d1/files/fx",

          "metaurl": "/dirs/d1/files/fx/meta",
          "meta": {
            "fileid": "fx",
            "self": "/dirs/d1/files/fx/meta",
            "xid": "/dirs/d1/files/fx/meta",
            "xref": "/dirs/d1/files/f1"
          }
        }
      },
      "filescount": 2,
      "schemasurl": "/dirs/d1/schemas",
      "schemas": {},
      "schemascount": 0
    }
  },
  "dirscount": 1
}
`)

}
