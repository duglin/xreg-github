package tests

import (
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestNoModel(t *testing.T) {
	reg := NewRegistry("TestNoModel")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "reg create didn't work")

	xCheckGet(t, reg, "/model", `{
  "schemas": [
    "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
  ],
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
  }
}
`)
	xCheckGet(t, reg, "?inline=model", `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestNoModel",
  "self": "http://localhost:8181/",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "model": {
    "schemas": [
      "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
    ],
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
    }
  }
}
`)

	xCheckGet(t, reg, "/model/foo", "Not found\n")
}

func TestRegSchema(t *testing.T) {
	reg := NewRegistry("TestRegSchema")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "reg create didn't work")

	reg.Model.AddSchema("schema1")
	xCheckGet(t, reg, "/model", `{
  "schemas": [
    "schema1",
    "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
  ],
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
  }
}
`)

	reg.Model.AddSchema("schema2")
	xCheckGet(t, reg, "/model", `{
  "schemas": [
    "schema1",
    "schema2",
    "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
  ],
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
  }
}
`)

	reg.Model.DelSchema("schema1")
	xCheckGet(t, reg, "/model", `{
  "schemas": [
    "schema2",
    "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
  ],
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
  }
}
`)

	reg.Model.DelSchema("schema2")
	xCheckGet(t, reg, "/model", `{
  "schemas": [
    "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
  ],
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
  }
}
`)
}

func TestGroupModelCreate(t *testing.T) {
	reg := NewRegistry("TestGroupModelCreate")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "reg create didn't work")

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)

	xCheckGet(t, reg, "/model", `{
  "schemas": [
    "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
  ],
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
      }
    }
  }
}
`)

	xCheckGet(t, reg, "/model", `{
  "schemas": [
    "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
  ],
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
      }
    }
  }
}
`)

	xCheckGet(t, reg, "/model", `{
  "schemas": [
    "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
  ],
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
      }
    }
  }
}
`)

	// Now error checking
	gm, err = reg.Model.AddGroupModel("dirs1", "") // missing value
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.Model.AddGroupModel("", "") // missing value
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.Model.AddGroupModel("", "") // missing value
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.Model.AddGroupModel("", "dir1") // missing value
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.Model.AddGroupModel("dirs", "dir") // dup
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.Model.AddGroupModel("dirs1", "dir") // dup
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.Model.AddGroupModel("dirs", "dir1") // dup
	xCheck(t, gm == nil && err != nil, "gm should have failed")
}

func TestResourceModelCreate(t *testing.T) {
	reg := NewRegistry("TestResourceModels")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "reg create didn't work")

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)
	xCheck(t, gm != nil, "gm should have worked")

	rm, err := gm.AddResourceModel("files", "file", 5, true, true, true)
	xNoErr(t, err)
	xCheck(t, rm != nil, "rm should have worked")

	rm2, err := gm.AddResourceModel("files", "file", 0, true, true, true)
	xCheck(t, rm2 == nil && err != nil, "rm2 should have failed")

	rm2, err = gm.AddResourceModel("files2", "file", 0, true, true, true)
	xCheck(t, rm2 == nil && err != nil, "rm2 should have failed")

	rm2, err = gm.AddResourceModel("", "file2", 0, true, true, true)
	xCheck(t, rm2 == nil && err != nil, "rm2 should have failed")

	rm2, err = gm.AddResourceModel("files2", "", 0, true, true, true)
	xCheck(t, rm2 == nil && err != nil, "rm2 should have failed")

	rm2, err = gm.AddResourceModel("files2", "file2", -1, true, true, true)
	xCheck(t, rm2 == nil && err != nil, "rm2 should have failed")

	gm2, err := reg.Model.AddGroupModel("dirs2", "dir2")
	xNoErr(t, err)
	xCheck(t, gm != nil, "gm2 should have worked")

	rm2, err = gm2.AddResourceModel("files", "file", 0, true, true, true)
	xCheck(t, rm != nil && err == nil, "gm2/rm2 should have worked")

	xCheckGet(t, reg, "/model", `{
  "schemas": [
    "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
  ],
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
          "maxversions": 5,
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
            "self": {
              "name": "self",
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
            },
            "fileid": {
              "name": "fileid",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            }
          }
        }
      }
    },
    "dirs2": {
      "plural": "dirs2",
      "singular": "dir2",
      "attributes": {
        "dir2id": {
          "name": "dir2id",
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
            "self": {
              "name": "self",
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
            },
            "fileid": {
              "name": "fileid",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            }
          }
        }
      }
    }
  }
}
`)

	rm2.Delete()
	xCheckGet(t, reg, "/model", `{
  "schemas": [
    "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
  ],
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
          "maxversions": 5,
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
            "self": {
              "name": "self",
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
            },
            "fileid": {
              "name": "fileid",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            }
          }
        }
      }
    },
    "dirs2": {
      "plural": "dirs2",
      "singular": "dir2",
      "attributes": {
        "dir2id": {
          "name": "dir2id",
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
      }
    }
  }
}
`)

	xCheckGet(t, reg, "/model", `{
  "schemas": [
    "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
  ],
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
          "maxversions": 5,
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
            "self": {
              "name": "self",
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
            },
            "fileid": {
              "name": "fileid",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            }
          }
        }
      }
    },
    "dirs2": {
      "plural": "dirs2",
      "singular": "dir2",
      "attributes": {
        "dir2id": {
          "name": "dir2id",
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
      }
    }
  }
}
`)

	gm2.Delete()
	xCheckGet(t, reg, "/model", `{
  "schemas": [
    "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
  ],
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
          "maxversions": 5,
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
            "self": {
              "name": "self",
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
            },
            "fileid": {
              "name": "fileid",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            }
          }
        }
      }
    }
  }
}
`)

	xCheckGet(t, reg, "/model", `{
  "schemas": [
    "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
  ],
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
          "maxversions": 5,
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
            "self": {
              "name": "self",
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
            },
            "fileid": {
              "name": "fileid",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            }
          }
        }
      }
    }
  }
}
`)

	newModel := &registry.Model{
		Attributes: registry.Attributes{
			"mystr": &registry.Attribute{
				Name: "mystr",
				Type: registry.STRING,
			},
		},
		Groups: map[string]*registry.GroupModel{
			"dirs": &registry.GroupModel{
				Plural:   "dirs",
				Singular: "dir",
				Resources: map[string]*registry.ResourceModel{
					"files": &registry.ResourceModel{
						Plural:           "files",
						Singular:         "file",
						MaxVersions:      6,
						SetVersionId:     registry.PtrBool(false),
						SetDefaultSticky: registry.PtrBool(false),
						HasDocument:      registry.PtrBool(false),
					},
				},
			},
		},
	}

	xNoErr(t, reg.Model.ApplyNewModel(newModel))
	xCheckGet(t, reg, "/model", `{
  "schemas": [
    "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
  ],
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
    },
    "mystr": {
      "name": "mystr",
      "type": "string"
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
          "maxversions": 6,
          "setversionid": false,
          "setdefaultversionsticky": false,
          "hasdocument": false,
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
            "self": {
              "name": "self",
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
            },
            "fileid": {
              "name": "fileid",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            }
          }
        }
      }
    }
  }
}
`)

	// Make sure we allow, but ignore updates to "model"
	newModel = &registry.Model{
		Attributes: registry.Attributes{
			"model": &registry.Attribute{
				Name: "model",
				Type: registry.STRING,
			},
		},
	}
	err = reg.Model.ApplyNewModel(newModel)
	xNoErr(t, err)

	// Rollback since the previous "newModel" erased too much
	xNoErr(t, reg.Rollback())
	reg.Refresh()
	reg.LoadModel()

	g, err := reg.AddGroup("dirs", "dir1")
	xNoErr(t, err)
	g.AddResource("files", "f1", "v1")

	xCheckGet(t, reg, "?inline=model,dirs.files", `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestResourceModels",
  "self": "http://localhost:8181/",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "model": {
    "schemas": [
      "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
    ],
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
      },
      "mystr": {
        "name": "mystr",
        "type": "string"
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
            "maxversions": 6,
            "setversionid": false,
            "setdefaultversionsticky": false,
            "hasdocument": false,
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
              "self": {
                "name": "self",
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
              },
              "fileid": {
                "name": "fileid",
                "type": "string",
                "immutable": true,
                "serverrequired": true
              }
            }
          }
        }
      }
    }
  },

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "dir1": {
      "dirid": "dir1",
      "self": "http://localhost:8181/dirs/dir1",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/dir1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "v1",
          "self": "http://localhost:8181/dirs/dir1/files/f1",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",

          "metaurl": "http://localhost:8181/dirs/dir1/files/f1/meta",
          "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions",
          "versionscount": 1
        }
      },
      "filescount": 1
    }
  },
  "dirscount": 1
}
`)

	newModel = &registry.Model{
		Groups: map[string]*registry.GroupModel{
			"dirs": &registry.GroupModel{
				Plural:   "dirs",
				Singular: "dir",
				Resources: map[string]*registry.ResourceModel{
					"files2": &registry.ResourceModel{
						Plural:           "files2",
						Singular:         "file",
						MaxVersions:      6,
						SetVersionId:     registry.PtrBool(false),
						SetDefaultSticky: registry.PtrBool(false),
						HasDocument:      registry.PtrBool(false),
					},
				},
			},
		},
	}

	reg.Model.ApplyNewModel(newModel)
	xCheckGet(t, reg, "?inline=model&inline=dirs", `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestResourceModels",
  "self": "http://localhost:8181/",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "model": {
    "schemas": [
      "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
    ],
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
          "files2": {
            "plural": "files2",
            "singular": "file",
            "maxversions": 6,
            "setversionid": false,
            "setdefaultversionsticky": false,
            "hasdocument": false,
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
              "self": {
                "name": "self",
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
              },
              "fileid": {
                "name": "fileid",
                "type": "string",
                "immutable": true,
                "serverrequired": true
              }
            }
          }
        }
      }
    }
  },

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "dir1": {
      "dirid": "dir1",
      "self": "http://localhost:8181/dirs/dir1",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "files2url": "http://localhost:8181/dirs/dir1/files2",
      "files2count": 0
    }
  },
  "dirscount": 1
}
`)

	newModel = &registry.Model{
		Groups: map[string]*registry.GroupModel{
			"dirs": &registry.GroupModel{
				Plural:   "dirs",
				Singular: "dir",
			},
		},
	}

	reg.Model.ApplyNewModel(newModel)
	xCheckGet(t, reg, "?inline=model&inline=dirs", `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestResourceModels",
  "self": "http://localhost:8181/",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "model": {
    "schemas": [
      "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
    ],
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
        }
      }
    }
  },

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "dir1": {
      "dirid": "dir1",
      "self": "http://localhost:8181/dirs/dir1",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z"
    }
  },
  "dirscount": 1
}
`)

	newModel = &registry.Model{
		Groups: map[string]*registry.GroupModel{
			"dirs2": &registry.GroupModel{
				Plural:   "dirs2",
				Singular: "dir2",
			},
		},
	}
	reg.Model.ApplyNewModel(newModel)
	xCheckGet(t, reg, "?inline=model&inline=", `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestResourceModels",
  "self": "http://localhost:8181/",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "model": {
    "schemas": [
      "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
    ],
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
      "dirs2": {
        "plural": "dirs2",
        "singular": "dir2",
        "attributes": {
          "dir2id": {
            "name": "dir2id",
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
        }
      }
    }
  },

  "dirs2url": "http://localhost:8181/dirs2",
  "dirs2": {},
  "dirs2count": 0
}
`)
}

func TestMultModelCreate(t *testing.T) {
	reg := NewRegistry("TestMultModelCreate")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "reg create didn't work")

	gm1, err := reg.Model.AddGroupModel("gms1", "gm1")
	xCheck(t, gm1 != nil && err == nil, "gm1 should have worked")

	rm1, err := gm1.AddResourceModel("rms1", "rm1", 0, true, true, true)
	xCheck(t, rm1 != nil && err == nil, "rm1 should have worked: %s", err)

	rm2, err := gm1.AddResourceModel("rms2", "rm2", 1, true, false, true)
	xCheck(t, rm2 != nil && err == nil, "rm2 should have worked: %s", err)

	gm2, err := reg.Model.AddGroupModel("gms2", "gm2")
	xCheck(t, gm1 != nil && err == nil, "gm1 should have worked: %s", err)

	rm21, err := gm2.AddResourceModel("rms1", "rm1", 2, true, true, true)
	xCheck(t, rm21 != nil && err == nil, "rm21 should have worked: %s", err)

	rm22, err := gm2.AddResourceModel("rms2", "rm2", 3, true, true, true)
	xCheck(t, rm22 != nil && err == nil, "rm12 should have worked: %s", err)

	xCheckGet(t, reg, "/model", `{
  "schemas": [
    "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
  ],
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
    "gms1": {
      "plural": "gms1",
      "singular": "gm1",
      "attributes": {
        "gm1id": {
          "name": "gm1id",
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
        "rms1": {
          "plural": "rms1",
          "singular": "rm1",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "attributes": {
            "rm1id": {
              "name": "rm1id",
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
            "self": {
              "name": "self",
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
            },
            "rm1id": {
              "name": "rm1id",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            }
          }
        },
        "rms2": {
          "plural": "rms2",
          "singular": "rm2",
          "maxversions": 1,
          "setversionid": true,
          "setdefaultversionsticky": false,
          "hasdocument": true,
          "attributes": {
            "rm2id": {
              "name": "rm2id",
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
            "self": {
              "name": "self",
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
            },
            "rm2id": {
              "name": "rm2id",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            }
          }
        }
      }
    },
    "gms2": {
      "plural": "gms2",
      "singular": "gm2",
      "attributes": {
        "gm2id": {
          "name": "gm2id",
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
        "rms1": {
          "plural": "rms1",
          "singular": "rm1",
          "maxversions": 2,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "attributes": {
            "rm1id": {
              "name": "rm1id",
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
            "self": {
              "name": "self",
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
            },
            "rm1id": {
              "name": "rm1id",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            }
          }
        },
        "rms2": {
          "plural": "rms2",
          "singular": "rm2",
          "maxversions": 3,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "attributes": {
            "rm2id": {
              "name": "rm2id",
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
            "self": {
              "name": "self",
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
            },
            "rm2id": {
              "name": "rm2id",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            }
          }
        }
      }
    }
  }
}
`)
}

func TestModelAPI(t *testing.T) {
	reg := NewRegistry("TestModelAPI")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "reg create didn't work")

	gm, _ := reg.Model.AddGroupModel("dirs1", "dir1")
	gm.AddResourceModel("files", "file", 2, true, false, true)

	gm2, _ := reg.Model.AddGroupModel("dirs2", "dir2")
	gm2.AddResourceModel("files", "file", 0, false, true, true)

	m := reg.LoadModel()
	xJSONCheck(t, m, reg.Model)
}

func TestMultModel2Create(t *testing.T) {
	reg := NewRegistry("TestMultModel2Create")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "reg create didn't work")
	reg.SaveAllAndCommit()
	reg.Refresh()

	gm, _ := reg.Model.AddGroupModel("dirs1", "dir1")
	gm.AddResourceModel("files", "file", 2, true, false, true)

	d, _ := reg.AddGroup("dirs1", "d1")
	f, _ := d.AddResource("files", "f1", "v1")
	f.AddVersion("v2")
	d, _ = reg.AddGroup("dirs1", "d2")
	f, _ = d.AddResource("files", "f2", "v1")
	f.AddVersion("v1.1")

	gm2, _ := reg.Model.AddGroupModel("dirs2", "dir2")
	gm2.AddResourceModel("files", "file", 0, false, true, true)
	d2, _ := reg.AddGroup("dirs2", "d2")
	d2.AddResource("files", "f2", "v1")

	// /dirs1/d1/f1/v1
	//            /v2
	//       /d2/f2/v1
	//             v1.1
	// /dirs2/f2/f2/v1

	xCheckGet(t, reg, "?inline=model&inline", `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestMultModel2Create",
  "self": "http://localhost:8181/",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "model": {
    "schemas": [
      "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
    ],
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
      "dirs1": {
        "plural": "dirs1",
        "singular": "dir1",
        "attributes": {
          "dir1id": {
            "name": "dir1id",
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
            "maxversions": 2,
            "setversionid": true,
            "setdefaultversionsticky": false,
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
              "self": {
                "name": "self",
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
              },
              "fileid": {
                "name": "fileid",
                "type": "string",
                "immutable": true,
                "serverrequired": true
              }
            }
          }
        }
      },
      "dirs2": {
        "plural": "dirs2",
        "singular": "dir2",
        "attributes": {
          "dir2id": {
            "name": "dir2id",
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
            "setversionid": false,
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
              "self": {
                "name": "self",
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
              },
              "fileid": {
                "name": "fileid",
                "type": "string",
                "immutable": true,
                "serverrequired": true
              }
            }
          }
        }
      }
    }
  },

  "dirs1url": "http://localhost:8181/dirs1",
  "dirs1": {
    "d1": {
      "dir1id": "d1",
      "self": "http://localhost:8181/dirs1/d1",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs1/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "v2",
          "self": "http://localhost:8181/dirs1/d1/files/f1$structure",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",

          "metaurl": "http://localhost:8181/dirs1/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "http://localhost:8181/dirs1/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "2024-01-01T12:00:02Z",
            "modifiedat": "2024-01-01T12:00:02Z",

            "defaultversionid": "v2",
            "defaultversionurl": "http://localhost:8181/dirs1/d1/files/f1/versions/v2$structure"
          },
          "versionsurl": "http://localhost:8181/dirs1/d1/files/f1/versions",
          "versions": {
            "v1": {
              "fileid": "f1",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs1/d1/files/f1/versions/v1$structure",
              "epoch": 1,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z"
            },
            "v2": {
              "fileid": "f1",
              "versionid": "v2",
              "self": "http://localhost:8181/dirs1/d1/files/f1/versions/v2$structure",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z"
            }
          },
          "versionscount": 2
        }
      },
      "filescount": 1
    },
    "d2": {
      "dir1id": "d2",
      "self": "http://localhost:8181/dirs1/d2",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs1/d2/files",
      "files": {
        "f2": {
          "fileid": "f2",
          "versionid": "v1.1",
          "self": "http://localhost:8181/dirs1/d2/files/f2$structure",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",

          "metaurl": "http://localhost:8181/dirs1/d2/files/f2/meta",
          "meta": {
            "fileid": "f2",
            "self": "http://localhost:8181/dirs1/d2/files/f2/meta",
            "epoch": 1,
            "createdat": "2024-01-01T12:00:02Z",
            "modifiedat": "2024-01-01T12:00:02Z",

            "defaultversionid": "v1.1",
            "defaultversionurl": "http://localhost:8181/dirs1/d2/files/f2/versions/v1.1$structure"
          },
          "versionsurl": "http://localhost:8181/dirs1/d2/files/f2/versions",
          "versions": {
            "v1": {
              "fileid": "f2",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs1/d2/files/f2/versions/v1$structure",
              "epoch": 1,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z"
            },
            "v1.1": {
              "fileid": "f2",
              "versionid": "v1.1",
              "self": "http://localhost:8181/dirs1/d2/files/f2/versions/v1.1$structure",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z"
            }
          },
          "versionscount": 2
        }
      },
      "filescount": 1
    }
  },
  "dirs1count": 2,
  "dirs2url": "http://localhost:8181/dirs2",
  "dirs2": {
    "d2": {
      "dir2id": "d2",
      "self": "http://localhost:8181/dirs2/d2",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs2/d2/files",
      "files": {
        "f2": {
          "fileid": "f2",
          "versionid": "v1",
          "self": "http://localhost:8181/dirs2/d2/files/f2$structure",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",

          "metaurl": "http://localhost:8181/dirs2/d2/files/f2/meta",
          "meta": {
            "fileid": "f2",
            "self": "http://localhost:8181/dirs2/d2/files/f2/meta",
            "epoch": 1,
            "createdat": "2024-01-01T12:00:02Z",
            "modifiedat": "2024-01-01T12:00:02Z",

            "defaultversionid": "v1",
            "defaultversionurl": "http://localhost:8181/dirs2/d2/files/f2/versions/v1$structure"
          },
          "versionsurl": "http://localhost:8181/dirs2/d2/files/f2/versions",
          "versions": {
            "v1": {
              "fileid": "f2",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs2/d2/files/f2/versions/v1$structure",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z"
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 1
    }
  },
  "dirs2count": 1
}
`)

	gm, _ = reg.Model.AddGroupModel("dirs0", "dir0")
	gm.AddResourceModel("files", "file", 2, true, false, true)
	gm, _ = reg.Model.AddGroupModel("dirs3", "dir3")
	gm.AddResourceModel("files", "file", 2, true, false, true)

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs0":{},"dirs1":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}},"dirs2":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{}}}}}},"dirs3":{}}`)

	gm, _ = reg.Model.AddGroupModel("dirs15", "dir15")
	gm.AddResourceModel("files", "file", 2, true, false, true)

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs0":{},"dirs1":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}},"dirs15":{},"dirs2":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{}}}}}},"dirs3":{}}`)

	gm, _ = reg.Model.AddGroupModel("dirs01", "dir01")
	gm, _ = reg.Model.AddGroupModel("dirs02", "dir02")
	gm, _ = reg.Model.AddGroupModel("dirs14", "dir014")
	gm, _ = reg.Model.AddGroupModel("dirs16", "dir016")
	gm, _ = reg.Model.AddGroupModel("dirs4", "dir4")
	gm, _ = reg.Model.AddGroupModel("dirs5", "dir5")

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs0":{},"dirs01":{},"dirs02":{},"dirs1":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}},"dirs14":{},"dirs15":{},"dirs16":{},"dirs2":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{}}}}}},"dirs3":{},"dirs4":{},"dirs5":{}}`)
}
