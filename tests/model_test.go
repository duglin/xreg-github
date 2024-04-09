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
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
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
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
    }
  }
}
`)
	xCheckGet(t, reg, "?model", `{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestNoModel",
  "epoch": 1,
  "self": "http://localhost:8181/",
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
      "id": {
        "name": "id",
        "type": "string",
        "immutable": true,
        "serverrequired": true
      },
      "name": {
        "name": "name",
        "type": "string"
      },
      "epoch": {
        "name": "epoch",
        "type": "uinteger",
        "serverrequired": true
      },
      "self": {
        "name": "self",
        "type": "url",
        "readonly": true,
        "serverrequired": true
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
      "createdby": {
        "name": "createdby",
        "type": "string",
        "readonly": true
      },
      "createdon": {
        "name": "createdon",
        "type": "timestamp",
        "readonly": true
      },
      "modifiedby": {
        "name": "modifiedby",
        "type": "string",
        "readonly": true
      },
      "modifiedon": {
        "name": "modifiedon",
        "type": "timestamp",
        "readonly": true
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
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
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
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
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
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
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
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
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
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
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
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
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
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
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
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
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
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
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
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
    }
  },
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "attributes": {
        "id": {
          "name": "id",
          "type": "string",
          "immutable": true,
          "serverrequired": true
        },
        "name": {
          "name": "name",
          "type": "string"
        },
        "epoch": {
          "name": "epoch",
          "type": "uinteger",
          "serverrequired": true
        },
        "self": {
          "name": "self",
          "type": "url",
          "readonly": true,
          "serverrequired": true
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
        "origin": {
          "name": "origin",
          "type": "uri"
        },
        "createdby": {
          "name": "createdby",
          "type": "string",
          "readonly": true
        },
        "createdon": {
          "name": "createdon",
          "type": "timestamp",
          "readonly": true
        },
        "modifiedby": {
          "name": "modifiedby",
          "type": "string",
          "readonly": true
        },
        "modifiedon": {
          "name": "modifiedon",
          "type": "timestamp",
          "readonly": true
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
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
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
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
    }
  },
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "attributes": {
        "id": {
          "name": "id",
          "type": "string",
          "immutable": true,
          "serverrequired": true
        },
        "name": {
          "name": "name",
          "type": "string"
        },
        "epoch": {
          "name": "epoch",
          "type": "uinteger",
          "serverrequired": true
        },
        "self": {
          "name": "self",
          "type": "url",
          "readonly": true,
          "serverrequired": true
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
        "origin": {
          "name": "origin",
          "type": "uri"
        },
        "createdby": {
          "name": "createdby",
          "type": "string",
          "readonly": true
        },
        "createdon": {
          "name": "createdon",
          "type": "timestamp",
          "readonly": true
        },
        "modifiedby": {
          "name": "modifiedby",
          "type": "string",
          "readonly": true
        },
        "modifiedon": {
          "name": "modifiedon",
          "type": "timestamp",
          "readonly": true
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
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
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
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
    }
  },
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "attributes": {
        "id": {
          "name": "id",
          "type": "string",
          "immutable": true,
          "serverrequired": true
        },
        "name": {
          "name": "name",
          "type": "string"
        },
        "epoch": {
          "name": "epoch",
          "type": "uinteger",
          "serverrequired": true
        },
        "self": {
          "name": "self",
          "type": "url",
          "readonly": true,
          "serverrequired": true
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
        "origin": {
          "name": "origin",
          "type": "uri"
        },
        "createdby": {
          "name": "createdby",
          "type": "string",
          "readonly": true
        },
        "createdon": {
          "name": "createdon",
          "type": "timestamp",
          "readonly": true
        },
        "modifiedby": {
          "name": "modifiedby",
          "type": "string",
          "readonly": true
        },
        "modifiedon": {
          "name": "modifiedon",
          "type": "timestamp",
          "readonly": true
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
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
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
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
    }
  },
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "attributes": {
        "id": {
          "name": "id",
          "type": "string",
          "immutable": true,
          "serverrequired": true
        },
        "name": {
          "name": "name",
          "type": "string"
        },
        "epoch": {
          "name": "epoch",
          "type": "uinteger",
          "serverrequired": true
        },
        "self": {
          "name": "self",
          "type": "url",
          "readonly": true,
          "serverrequired": true
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
        "origin": {
          "name": "origin",
          "type": "uri"
        },
        "createdby": {
          "name": "createdby",
          "type": "string",
          "readonly": true
        },
        "createdon": {
          "name": "createdon",
          "type": "timestamp",
          "readonly": true
        },
        "modifiedby": {
          "name": "modifiedby",
          "type": "string",
          "readonly": true
        },
        "modifiedon": {
          "name": "modifiedon",
          "type": "timestamp",
          "readonly": true
        }
      },
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 5,
          "setversionid": true,
          "setdefault": true,
          "hasdocument": true,
          "attributes": {
            "id": {
              "name": "id",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            },
            "name": {
              "name": "name",
              "type": "string"
            },
            "epoch": {
              "name": "epoch",
              "type": "uinteger",
              "serverrequired": true
            },
            "self": {
              "name": "self",
              "type": "url",
              "readonly": true,
              "serverrequired": true
            },
            "isdefault": {
              "name": "isdefault",
              "type": "boolean",
              "readonly": true
            },
            "defaultversionid": {
              "name": "defaultversionid",
              "type": "string",
              "readonly": true
            },
            "defaultversionurl": {
              "name": "defaultversionurl",
              "type": "url",
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
            "origin": {
              "name": "origin",
              "type": "uri"
            },
            "createdby": {
              "name": "createdby",
              "type": "string",
              "readonly": true
            },
            "createdon": {
              "name": "createdon",
              "type": "timestamp",
              "readonly": true
            },
            "modifiedby": {
              "name": "modifiedby",
              "type": "string",
              "readonly": true
            },
            "modifiedon": {
              "name": "modifiedon",
              "type": "timestamp",
              "readonly": true
            },
            "contenttype": {
              "name": "contenttype",
              "type": "string"
            }
          }
        }
      }
    },
    "dirs2": {
      "plural": "dirs2",
      "singular": "dir2",
      "attributes": {
        "id": {
          "name": "id",
          "type": "string",
          "immutable": true,
          "serverrequired": true
        },
        "name": {
          "name": "name",
          "type": "string"
        },
        "epoch": {
          "name": "epoch",
          "type": "uinteger",
          "serverrequired": true
        },
        "self": {
          "name": "self",
          "type": "url",
          "readonly": true,
          "serverrequired": true
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
        "origin": {
          "name": "origin",
          "type": "uri"
        },
        "createdby": {
          "name": "createdby",
          "type": "string",
          "readonly": true
        },
        "createdon": {
          "name": "createdon",
          "type": "timestamp",
          "readonly": true
        },
        "modifiedby": {
          "name": "modifiedby",
          "type": "string",
          "readonly": true
        },
        "modifiedon": {
          "name": "modifiedon",
          "type": "timestamp",
          "readonly": true
        }
      },
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "setdefault": true,
          "hasdocument": true,
          "attributes": {
            "id": {
              "name": "id",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            },
            "name": {
              "name": "name",
              "type": "string"
            },
            "epoch": {
              "name": "epoch",
              "type": "uinteger",
              "serverrequired": true
            },
            "self": {
              "name": "self",
              "type": "url",
              "readonly": true,
              "serverrequired": true
            },
            "isdefault": {
              "name": "isdefault",
              "type": "boolean",
              "readonly": true
            },
            "defaultversionid": {
              "name": "defaultversionid",
              "type": "string",
              "readonly": true
            },
            "defaultversionurl": {
              "name": "defaultversionurl",
              "type": "url",
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
            "origin": {
              "name": "origin",
              "type": "uri"
            },
            "createdby": {
              "name": "createdby",
              "type": "string",
              "readonly": true
            },
            "createdon": {
              "name": "createdon",
              "type": "timestamp",
              "readonly": true
            },
            "modifiedby": {
              "name": "modifiedby",
              "type": "string",
              "readonly": true
            },
            "modifiedon": {
              "name": "modifiedon",
              "type": "timestamp",
              "readonly": true
            },
            "contenttype": {
              "name": "contenttype",
              "type": "string"
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
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
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
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
    }
  },
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "attributes": {
        "id": {
          "name": "id",
          "type": "string",
          "immutable": true,
          "serverrequired": true
        },
        "name": {
          "name": "name",
          "type": "string"
        },
        "epoch": {
          "name": "epoch",
          "type": "uinteger",
          "serverrequired": true
        },
        "self": {
          "name": "self",
          "type": "url",
          "readonly": true,
          "serverrequired": true
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
        "origin": {
          "name": "origin",
          "type": "uri"
        },
        "createdby": {
          "name": "createdby",
          "type": "string",
          "readonly": true
        },
        "createdon": {
          "name": "createdon",
          "type": "timestamp",
          "readonly": true
        },
        "modifiedby": {
          "name": "modifiedby",
          "type": "string",
          "readonly": true
        },
        "modifiedon": {
          "name": "modifiedon",
          "type": "timestamp",
          "readonly": true
        }
      },
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 5,
          "setversionid": true,
          "setdefault": true,
          "hasdocument": true,
          "attributes": {
            "id": {
              "name": "id",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            },
            "name": {
              "name": "name",
              "type": "string"
            },
            "epoch": {
              "name": "epoch",
              "type": "uinteger",
              "serverrequired": true
            },
            "self": {
              "name": "self",
              "type": "url",
              "readonly": true,
              "serverrequired": true
            },
            "isdefault": {
              "name": "isdefault",
              "type": "boolean",
              "readonly": true
            },
            "defaultversionid": {
              "name": "defaultversionid",
              "type": "string",
              "readonly": true
            },
            "defaultversionurl": {
              "name": "defaultversionurl",
              "type": "url",
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
            "origin": {
              "name": "origin",
              "type": "uri"
            },
            "createdby": {
              "name": "createdby",
              "type": "string",
              "readonly": true
            },
            "createdon": {
              "name": "createdon",
              "type": "timestamp",
              "readonly": true
            },
            "modifiedby": {
              "name": "modifiedby",
              "type": "string",
              "readonly": true
            },
            "modifiedon": {
              "name": "modifiedon",
              "type": "timestamp",
              "readonly": true
            },
            "contenttype": {
              "name": "contenttype",
              "type": "string"
            }
          }
        }
      }
    },
    "dirs2": {
      "plural": "dirs2",
      "singular": "dir2",
      "attributes": {
        "id": {
          "name": "id",
          "type": "string",
          "immutable": true,
          "serverrequired": true
        },
        "name": {
          "name": "name",
          "type": "string"
        },
        "epoch": {
          "name": "epoch",
          "type": "uinteger",
          "serverrequired": true
        },
        "self": {
          "name": "self",
          "type": "url",
          "readonly": true,
          "serverrequired": true
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
        "origin": {
          "name": "origin",
          "type": "uri"
        },
        "createdby": {
          "name": "createdby",
          "type": "string",
          "readonly": true
        },
        "createdon": {
          "name": "createdon",
          "type": "timestamp",
          "readonly": true
        },
        "modifiedby": {
          "name": "modifiedby",
          "type": "string",
          "readonly": true
        },
        "modifiedon": {
          "name": "modifiedon",
          "type": "timestamp",
          "readonly": true
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
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
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
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
    }
  },
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "attributes": {
        "id": {
          "name": "id",
          "type": "string",
          "immutable": true,
          "serverrequired": true
        },
        "name": {
          "name": "name",
          "type": "string"
        },
        "epoch": {
          "name": "epoch",
          "type": "uinteger",
          "serverrequired": true
        },
        "self": {
          "name": "self",
          "type": "url",
          "readonly": true,
          "serverrequired": true
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
        "origin": {
          "name": "origin",
          "type": "uri"
        },
        "createdby": {
          "name": "createdby",
          "type": "string",
          "readonly": true
        },
        "createdon": {
          "name": "createdon",
          "type": "timestamp",
          "readonly": true
        },
        "modifiedby": {
          "name": "modifiedby",
          "type": "string",
          "readonly": true
        },
        "modifiedon": {
          "name": "modifiedon",
          "type": "timestamp",
          "readonly": true
        }
      },
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 5,
          "setversionid": true,
          "setdefault": true,
          "hasdocument": true,
          "attributes": {
            "id": {
              "name": "id",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            },
            "name": {
              "name": "name",
              "type": "string"
            },
            "epoch": {
              "name": "epoch",
              "type": "uinteger",
              "serverrequired": true
            },
            "self": {
              "name": "self",
              "type": "url",
              "readonly": true,
              "serverrequired": true
            },
            "isdefault": {
              "name": "isdefault",
              "type": "boolean",
              "readonly": true
            },
            "defaultversionid": {
              "name": "defaultversionid",
              "type": "string",
              "readonly": true
            },
            "defaultversionurl": {
              "name": "defaultversionurl",
              "type": "url",
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
            "origin": {
              "name": "origin",
              "type": "uri"
            },
            "createdby": {
              "name": "createdby",
              "type": "string",
              "readonly": true
            },
            "createdon": {
              "name": "createdon",
              "type": "timestamp",
              "readonly": true
            },
            "modifiedby": {
              "name": "modifiedby",
              "type": "string",
              "readonly": true
            },
            "modifiedon": {
              "name": "modifiedon",
              "type": "timestamp",
              "readonly": true
            },
            "contenttype": {
              "name": "contenttype",
              "type": "string"
            }
          }
        }
      }
    },
    "dirs2": {
      "plural": "dirs2",
      "singular": "dir2",
      "attributes": {
        "id": {
          "name": "id",
          "type": "string",
          "immutable": true,
          "serverrequired": true
        },
        "name": {
          "name": "name",
          "type": "string"
        },
        "epoch": {
          "name": "epoch",
          "type": "uinteger",
          "serverrequired": true
        },
        "self": {
          "name": "self",
          "type": "url",
          "readonly": true,
          "serverrequired": true
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
        "origin": {
          "name": "origin",
          "type": "uri"
        },
        "createdby": {
          "name": "createdby",
          "type": "string",
          "readonly": true
        },
        "createdon": {
          "name": "createdon",
          "type": "timestamp",
          "readonly": true
        },
        "modifiedby": {
          "name": "modifiedby",
          "type": "string",
          "readonly": true
        },
        "modifiedon": {
          "name": "modifiedon",
          "type": "timestamp",
          "readonly": true
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
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
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
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
    }
  },
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "attributes": {
        "id": {
          "name": "id",
          "type": "string",
          "immutable": true,
          "serverrequired": true
        },
        "name": {
          "name": "name",
          "type": "string"
        },
        "epoch": {
          "name": "epoch",
          "type": "uinteger",
          "serverrequired": true
        },
        "self": {
          "name": "self",
          "type": "url",
          "readonly": true,
          "serverrequired": true
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
        "origin": {
          "name": "origin",
          "type": "uri"
        },
        "createdby": {
          "name": "createdby",
          "type": "string",
          "readonly": true
        },
        "createdon": {
          "name": "createdon",
          "type": "timestamp",
          "readonly": true
        },
        "modifiedby": {
          "name": "modifiedby",
          "type": "string",
          "readonly": true
        },
        "modifiedon": {
          "name": "modifiedon",
          "type": "timestamp",
          "readonly": true
        }
      },
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 5,
          "setversionid": true,
          "setdefault": true,
          "hasdocument": true,
          "attributes": {
            "id": {
              "name": "id",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            },
            "name": {
              "name": "name",
              "type": "string"
            },
            "epoch": {
              "name": "epoch",
              "type": "uinteger",
              "serverrequired": true
            },
            "self": {
              "name": "self",
              "type": "url",
              "readonly": true,
              "serverrequired": true
            },
            "isdefault": {
              "name": "isdefault",
              "type": "boolean",
              "readonly": true
            },
            "defaultversionid": {
              "name": "defaultversionid",
              "type": "string",
              "readonly": true
            },
            "defaultversionurl": {
              "name": "defaultversionurl",
              "type": "url",
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
            "origin": {
              "name": "origin",
              "type": "uri"
            },
            "createdby": {
              "name": "createdby",
              "type": "string",
              "readonly": true
            },
            "createdon": {
              "name": "createdon",
              "type": "timestamp",
              "readonly": true
            },
            "modifiedby": {
              "name": "modifiedby",
              "type": "string",
              "readonly": true
            },
            "modifiedon": {
              "name": "modifiedon",
              "type": "timestamp",
              "readonly": true
            },
            "contenttype": {
              "name": "contenttype",
              "type": "string"
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
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
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
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
    }
  },
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "attributes": {
        "id": {
          "name": "id",
          "type": "string",
          "immutable": true,
          "serverrequired": true
        },
        "name": {
          "name": "name",
          "type": "string"
        },
        "epoch": {
          "name": "epoch",
          "type": "uinteger",
          "serverrequired": true
        },
        "self": {
          "name": "self",
          "type": "url",
          "readonly": true,
          "serverrequired": true
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
        "origin": {
          "name": "origin",
          "type": "uri"
        },
        "createdby": {
          "name": "createdby",
          "type": "string",
          "readonly": true
        },
        "createdon": {
          "name": "createdon",
          "type": "timestamp",
          "readonly": true
        },
        "modifiedby": {
          "name": "modifiedby",
          "type": "string",
          "readonly": true
        },
        "modifiedon": {
          "name": "modifiedon",
          "type": "timestamp",
          "readonly": true
        }
      },
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 5,
          "setversionid": true,
          "setdefault": true,
          "hasdocument": true,
          "attributes": {
            "id": {
              "name": "id",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            },
            "name": {
              "name": "name",
              "type": "string"
            },
            "epoch": {
              "name": "epoch",
              "type": "uinteger",
              "serverrequired": true
            },
            "self": {
              "name": "self",
              "type": "url",
              "readonly": true,
              "serverrequired": true
            },
            "isdefault": {
              "name": "isdefault",
              "type": "boolean",
              "readonly": true
            },
            "defaultversionid": {
              "name": "defaultversionid",
              "type": "string",
              "readonly": true
            },
            "defaultversionurl": {
              "name": "defaultversionurl",
              "type": "url",
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
            "origin": {
              "name": "origin",
              "type": "uri"
            },
            "createdby": {
              "name": "createdby",
              "type": "string",
              "readonly": true
            },
            "createdon": {
              "name": "createdon",
              "type": "timestamp",
              "readonly": true
            },
            "modifiedby": {
              "name": "modifiedby",
              "type": "string",
              "readonly": true
            },
            "modifiedon": {
              "name": "modifiedon",
              "type": "timestamp",
              "readonly": true
            },
            "contenttype": {
              "name": "contenttype",
              "type": "string"
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
						Plural:       "files",
						Singular:     "file",
						MaxVersions:  6,
						SetVersionId: false,
						SetDefault:   false,
						// Note that hasdocument is missing -> false per golang
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
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
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
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
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
        "id": {
          "name": "id",
          "type": "string",
          "immutable": true,
          "serverrequired": true
        },
        "name": {
          "name": "name",
          "type": "string"
        },
        "epoch": {
          "name": "epoch",
          "type": "uinteger",
          "serverrequired": true
        },
        "self": {
          "name": "self",
          "type": "url",
          "readonly": true,
          "serverrequired": true
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
        "origin": {
          "name": "origin",
          "type": "uri"
        },
        "createdby": {
          "name": "createdby",
          "type": "string",
          "readonly": true
        },
        "createdon": {
          "name": "createdon",
          "type": "timestamp",
          "readonly": true
        },
        "modifiedby": {
          "name": "modifiedby",
          "type": "string",
          "readonly": true
        },
        "modifiedon": {
          "name": "modifiedon",
          "type": "timestamp",
          "readonly": true
        }
      },
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 6,
          "setversionid": false,
          "setdefault": false,
          "hasdocument": false,
          "attributes": {
            "id": {
              "name": "id",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            },
            "name": {
              "name": "name",
              "type": "string"
            },
            "epoch": {
              "name": "epoch",
              "type": "uinteger",
              "serverrequired": true
            },
            "self": {
              "name": "self",
              "type": "url",
              "readonly": true,
              "serverrequired": true
            },
            "isdefault": {
              "name": "isdefault",
              "type": "boolean",
              "readonly": true
            },
            "defaultversionid": {
              "name": "defaultversionid",
              "type": "string",
              "readonly": true
            },
            "defaultversionurl": {
              "name": "defaultversionurl",
              "type": "url",
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
            "origin": {
              "name": "origin",
              "type": "uri"
            },
            "createdby": {
              "name": "createdby",
              "type": "string",
              "readonly": true
            },
            "createdon": {
              "name": "createdon",
              "type": "timestamp",
              "readonly": true
            },
            "modifiedby": {
              "name": "modifiedby",
              "type": "string",
              "readonly": true
            },
            "modifiedon": {
              "name": "modifiedon",
              "type": "timestamp",
              "readonly": true
            },
            "contenttype": {
              "name": "contenttype",
              "type": "string"
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
	reg.LoadModel()

	g, err := reg.AddGroup("dirs", "dir1")
	xNoErr(t, err)
	g.AddResource("files", "f1", "v1")

	xCheckGet(t, reg, "?model&inline=dirs.files", `{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestResourceModels",
  "epoch": 1,
  "self": "http://localhost:8181/",
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
      "id": {
        "name": "id",
        "type": "string",
        "immutable": true,
        "serverrequired": true
      },
      "name": {
        "name": "name",
        "type": "string"
      },
      "epoch": {
        "name": "epoch",
        "type": "uinteger",
        "serverrequired": true
      },
      "self": {
        "name": "self",
        "type": "url",
        "readonly": true,
        "serverrequired": true
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
      "createdby": {
        "name": "createdby",
        "type": "string",
        "readonly": true
      },
      "createdon": {
        "name": "createdon",
        "type": "timestamp",
        "readonly": true
      },
      "modifiedby": {
        "name": "modifiedby",
        "type": "string",
        "readonly": true
      },
      "modifiedon": {
        "name": "modifiedon",
        "type": "timestamp",
        "readonly": true
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
          "id": {
            "name": "id",
            "type": "string",
            "immutable": true,
            "serverrequired": true
          },
          "name": {
            "name": "name",
            "type": "string"
          },
          "epoch": {
            "name": "epoch",
            "type": "uinteger",
            "serverrequired": true
          },
          "self": {
            "name": "self",
            "type": "url",
            "readonly": true,
            "serverrequired": true
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
          "origin": {
            "name": "origin",
            "type": "uri"
          },
          "createdby": {
            "name": "createdby",
            "type": "string",
            "readonly": true
          },
          "createdon": {
            "name": "createdon",
            "type": "timestamp",
            "readonly": true
          },
          "modifiedby": {
            "name": "modifiedby",
            "type": "string",
            "readonly": true
          },
          "modifiedon": {
            "name": "modifiedon",
            "type": "timestamp",
            "readonly": true
          }
        },
        "resources": {
          "files": {
            "plural": "files",
            "singular": "file",
            "maxversions": 6,
            "setversionid": false,
            "setdefault": false,
            "hasdocument": false,
            "attributes": {
              "id": {
                "name": "id",
                "type": "string",
                "immutable": true,
                "serverrequired": true
              },
              "name": {
                "name": "name",
                "type": "string"
              },
              "epoch": {
                "name": "epoch",
                "type": "uinteger",
                "serverrequired": true
              },
              "self": {
                "name": "self",
                "type": "url",
                "readonly": true,
                "serverrequired": true
              },
              "isdefault": {
                "name": "isdefault",
                "type": "boolean",
                "readonly": true
              },
              "defaultversionid": {
                "name": "defaultversionid",
                "type": "string",
                "readonly": true
              },
              "defaultversionurl": {
                "name": "defaultversionurl",
                "type": "url",
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
              "origin": {
                "name": "origin",
                "type": "uri"
              },
              "createdby": {
                "name": "createdby",
                "type": "string",
                "readonly": true
              },
              "createdon": {
                "name": "createdon",
                "type": "timestamp",
                "readonly": true
              },
              "modifiedby": {
                "name": "modifiedby",
                "type": "string",
                "readonly": true
              },
              "modifiedon": {
                "name": "modifiedon",
                "type": "timestamp",
                "readonly": true
              },
              "contenttype": {
                "name": "contenttype",
                "type": "string"
              }
            }
          }
        }
      }
    }
  },

  "dirs": {
    "dir1": {
      "id": "dir1",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/dir1",

      "files": {
        "f1": {
          "id": "f1",
          "epoch": 1,
          "self": "http://localhost:8181/dirs/dir1/files/f1",
          "defaultversionid": "v1",
          "defaultversionurl": "http://localhost:8181/dirs/dir1/files/f1/versions/v1",

          "versionscount": 1,
          "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions"
        }
      },
      "filescount": 1,
      "filesurl": "http://localhost:8181/dirs/dir1/files"
    }
  },
  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
}
`)

	newModel = &registry.Model{
		Groups: map[string]*registry.GroupModel{
			"dirs": &registry.GroupModel{
				Plural:   "dirs",
				Singular: "dir",
				Resources: map[string]*registry.ResourceModel{
					"files2": &registry.ResourceModel{
						Plural:       "files2",
						Singular:     "file",
						MaxVersions:  6,
						SetVersionId: false,
						SetDefault:   false,
					},
				},
			},
		},
	}

	reg.Model.ApplyNewModel(newModel)
	xCheckGet(t, reg, "?model&inline=dirs", `{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestResourceModels",
  "epoch": 1,
  "self": "http://localhost:8181/",
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
      "id": {
        "name": "id",
        "type": "string",
        "immutable": true,
        "serverrequired": true
      },
      "name": {
        "name": "name",
        "type": "string"
      },
      "epoch": {
        "name": "epoch",
        "type": "uinteger",
        "serverrequired": true
      },
      "self": {
        "name": "self",
        "type": "url",
        "readonly": true,
        "serverrequired": true
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
      "createdby": {
        "name": "createdby",
        "type": "string",
        "readonly": true
      },
      "createdon": {
        "name": "createdon",
        "type": "timestamp",
        "readonly": true
      },
      "modifiedby": {
        "name": "modifiedby",
        "type": "string",
        "readonly": true
      },
      "modifiedon": {
        "name": "modifiedon",
        "type": "timestamp",
        "readonly": true
      }
    },
    "groups": {
      "dirs": {
        "plural": "dirs",
        "singular": "dir",
        "attributes": {
          "id": {
            "name": "id",
            "type": "string",
            "immutable": true,
            "serverrequired": true
          },
          "name": {
            "name": "name",
            "type": "string"
          },
          "epoch": {
            "name": "epoch",
            "type": "uinteger",
            "serverrequired": true
          },
          "self": {
            "name": "self",
            "type": "url",
            "readonly": true,
            "serverrequired": true
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
          "origin": {
            "name": "origin",
            "type": "uri"
          },
          "createdby": {
            "name": "createdby",
            "type": "string",
            "readonly": true
          },
          "createdon": {
            "name": "createdon",
            "type": "timestamp",
            "readonly": true
          },
          "modifiedby": {
            "name": "modifiedby",
            "type": "string",
            "readonly": true
          },
          "modifiedon": {
            "name": "modifiedon",
            "type": "timestamp",
            "readonly": true
          }
        },
        "resources": {
          "files2": {
            "plural": "files2",
            "singular": "file",
            "maxversions": 6,
            "setversionid": false,
            "setdefault": false,
            "hasdocument": false,
            "attributes": {
              "id": {
                "name": "id",
                "type": "string",
                "immutable": true,
                "serverrequired": true
              },
              "name": {
                "name": "name",
                "type": "string"
              },
              "epoch": {
                "name": "epoch",
                "type": "uinteger",
                "serverrequired": true
              },
              "self": {
                "name": "self",
                "type": "url",
                "readonly": true,
                "serverrequired": true
              },
              "isdefault": {
                "name": "isdefault",
                "type": "boolean",
                "readonly": true
              },
              "defaultversionid": {
                "name": "defaultversionid",
                "type": "string",
                "readonly": true
              },
              "defaultversionurl": {
                "name": "defaultversionurl",
                "type": "url",
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
              "origin": {
                "name": "origin",
                "type": "uri"
              },
              "createdby": {
                "name": "createdby",
                "type": "string",
                "readonly": true
              },
              "createdon": {
                "name": "createdon",
                "type": "timestamp",
                "readonly": true
              },
              "modifiedby": {
                "name": "modifiedby",
                "type": "string",
                "readonly": true
              },
              "modifiedon": {
                "name": "modifiedon",
                "type": "timestamp",
                "readonly": true
              },
              "contenttype": {
                "name": "contenttype",
                "type": "string"
              }
            }
          }
        }
      }
    }
  },

  "dirs": {
    "dir1": {
      "id": "dir1",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/dir1",

      "files2count": 0,
      "files2url": "http://localhost:8181/dirs/dir1/files2"
    }
  },
  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
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
	xCheckGet(t, reg, "?model&inline=dirs", `{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestResourceModels",
  "epoch": 1,
  "self": "http://localhost:8181/",
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
      "id": {
        "name": "id",
        "type": "string",
        "immutable": true,
        "serverrequired": true
      },
      "name": {
        "name": "name",
        "type": "string"
      },
      "epoch": {
        "name": "epoch",
        "type": "uinteger",
        "serverrequired": true
      },
      "self": {
        "name": "self",
        "type": "url",
        "readonly": true,
        "serverrequired": true
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
      "createdby": {
        "name": "createdby",
        "type": "string",
        "readonly": true
      },
      "createdon": {
        "name": "createdon",
        "type": "timestamp",
        "readonly": true
      },
      "modifiedby": {
        "name": "modifiedby",
        "type": "string",
        "readonly": true
      },
      "modifiedon": {
        "name": "modifiedon",
        "type": "timestamp",
        "readonly": true
      }
    },
    "groups": {
      "dirs": {
        "plural": "dirs",
        "singular": "dir",
        "attributes": {
          "id": {
            "name": "id",
            "type": "string",
            "immutable": true,
            "serverrequired": true
          },
          "name": {
            "name": "name",
            "type": "string"
          },
          "epoch": {
            "name": "epoch",
            "type": "uinteger",
            "serverrequired": true
          },
          "self": {
            "name": "self",
            "type": "url",
            "readonly": true,
            "serverrequired": true
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
          "origin": {
            "name": "origin",
            "type": "uri"
          },
          "createdby": {
            "name": "createdby",
            "type": "string",
            "readonly": true
          },
          "createdon": {
            "name": "createdon",
            "type": "timestamp",
            "readonly": true
          },
          "modifiedby": {
            "name": "modifiedby",
            "type": "string",
            "readonly": true
          },
          "modifiedon": {
            "name": "modifiedon",
            "type": "timestamp",
            "readonly": true
          }
        }
      }
    }
  },

  "dirs": {
    "dir1": {
      "id": "dir1",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/dir1"
    }
  },
  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
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
	xCheckGet(t, reg, "?model&inline=", `{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestResourceModels",
  "epoch": 1,
  "self": "http://localhost:8181/",
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
      "id": {
        "name": "id",
        "type": "string",
        "immutable": true,
        "serverrequired": true
      },
      "name": {
        "name": "name",
        "type": "string"
      },
      "epoch": {
        "name": "epoch",
        "type": "uinteger",
        "serverrequired": true
      },
      "self": {
        "name": "self",
        "type": "url",
        "readonly": true,
        "serverrequired": true
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
      "createdby": {
        "name": "createdby",
        "type": "string",
        "readonly": true
      },
      "createdon": {
        "name": "createdon",
        "type": "timestamp",
        "readonly": true
      },
      "modifiedby": {
        "name": "modifiedby",
        "type": "string",
        "readonly": true
      },
      "modifiedon": {
        "name": "modifiedon",
        "type": "timestamp",
        "readonly": true
      }
    },
    "groups": {
      "dirs2": {
        "plural": "dirs2",
        "singular": "dir2",
        "attributes": {
          "id": {
            "name": "id",
            "type": "string",
            "immutable": true,
            "serverrequired": true
          },
          "name": {
            "name": "name",
            "type": "string"
          },
          "epoch": {
            "name": "epoch",
            "type": "uinteger",
            "serverrequired": true
          },
          "self": {
            "name": "self",
            "type": "url",
            "readonly": true,
            "serverrequired": true
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
          "origin": {
            "name": "origin",
            "type": "uri"
          },
          "createdby": {
            "name": "createdby",
            "type": "string",
            "readonly": true
          },
          "createdon": {
            "name": "createdon",
            "type": "timestamp",
            "readonly": true
          },
          "modifiedby": {
            "name": "modifiedby",
            "type": "string",
            "readonly": true
          },
          "modifiedon": {
            "name": "modifiedon",
            "type": "timestamp",
            "readonly": true
          }
        }
      }
    }
  },

  "dirs2": {},
  "dirs2count": 0,
  "dirs2url": "http://localhost:8181/dirs2"
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
	xCheck(t, rm1 != nil && err == nil, "rm1 should have worked")

	rm2, err := gm1.AddResourceModel("rms2", "rm2", 1, true, true, true)
	xCheck(t, rm2 != nil && err == nil, "rm2 should have worked")

	gm2, err := reg.Model.AddGroupModel("gms2", "gm2")
	xCheck(t, gm1 != nil && err == nil, "gm1 should have worked")

	rm21, err := gm2.AddResourceModel("rms1", "rm1", 2, true, true, true)
	xCheck(t, rm21 != nil && err == nil, "rm21 should have worked")

	rm22, err := gm2.AddResourceModel("rms2", "rm2", 3, true, true, true)
	xCheck(t, rm22 != nil && err == nil, "rm12 should have worked")

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
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
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
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
    }
  },
  "groups": {
    "gms1": {
      "plural": "gms1",
      "singular": "gm1",
      "attributes": {
        "id": {
          "name": "id",
          "type": "string",
          "immutable": true,
          "serverrequired": true
        },
        "name": {
          "name": "name",
          "type": "string"
        },
        "epoch": {
          "name": "epoch",
          "type": "uinteger",
          "serverrequired": true
        },
        "self": {
          "name": "self",
          "type": "url",
          "readonly": true,
          "serverrequired": true
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
        "origin": {
          "name": "origin",
          "type": "uri"
        },
        "createdby": {
          "name": "createdby",
          "type": "string",
          "readonly": true
        },
        "createdon": {
          "name": "createdon",
          "type": "timestamp",
          "readonly": true
        },
        "modifiedby": {
          "name": "modifiedby",
          "type": "string",
          "readonly": true
        },
        "modifiedon": {
          "name": "modifiedon",
          "type": "timestamp",
          "readonly": true
        }
      },
      "resources": {
        "rms1": {
          "plural": "rms1",
          "singular": "rm1",
          "maxversions": 0,
          "setversionid": true,
          "setdefault": true,
          "hasdocument": true,
          "attributes": {
            "id": {
              "name": "id",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            },
            "name": {
              "name": "name",
              "type": "string"
            },
            "epoch": {
              "name": "epoch",
              "type": "uinteger",
              "serverrequired": true
            },
            "self": {
              "name": "self",
              "type": "url",
              "readonly": true,
              "serverrequired": true
            },
            "isdefault": {
              "name": "isdefault",
              "type": "boolean",
              "readonly": true
            },
            "defaultversionid": {
              "name": "defaultversionid",
              "type": "string",
              "readonly": true
            },
            "defaultversionurl": {
              "name": "defaultversionurl",
              "type": "url",
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
            "origin": {
              "name": "origin",
              "type": "uri"
            },
            "createdby": {
              "name": "createdby",
              "type": "string",
              "readonly": true
            },
            "createdon": {
              "name": "createdon",
              "type": "timestamp",
              "readonly": true
            },
            "modifiedby": {
              "name": "modifiedby",
              "type": "string",
              "readonly": true
            },
            "modifiedon": {
              "name": "modifiedon",
              "type": "timestamp",
              "readonly": true
            },
            "contenttype": {
              "name": "contenttype",
              "type": "string"
            }
          }
        },
        "rms2": {
          "plural": "rms2",
          "singular": "rm2",
          "maxversions": 1,
          "setversionid": true,
          "setdefault": true,
          "hasdocument": true,
          "attributes": {
            "id": {
              "name": "id",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            },
            "name": {
              "name": "name",
              "type": "string"
            },
            "epoch": {
              "name": "epoch",
              "type": "uinteger",
              "serverrequired": true
            },
            "self": {
              "name": "self",
              "type": "url",
              "readonly": true,
              "serverrequired": true
            },
            "isdefault": {
              "name": "isdefault",
              "type": "boolean",
              "readonly": true
            },
            "defaultversionid": {
              "name": "defaultversionid",
              "type": "string",
              "readonly": true
            },
            "defaultversionurl": {
              "name": "defaultversionurl",
              "type": "url",
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
            "origin": {
              "name": "origin",
              "type": "uri"
            },
            "createdby": {
              "name": "createdby",
              "type": "string",
              "readonly": true
            },
            "createdon": {
              "name": "createdon",
              "type": "timestamp",
              "readonly": true
            },
            "modifiedby": {
              "name": "modifiedby",
              "type": "string",
              "readonly": true
            },
            "modifiedon": {
              "name": "modifiedon",
              "type": "timestamp",
              "readonly": true
            },
            "contenttype": {
              "name": "contenttype",
              "type": "string"
            }
          }
        }
      }
    },
    "gms2": {
      "plural": "gms2",
      "singular": "gm2",
      "attributes": {
        "id": {
          "name": "id",
          "type": "string",
          "immutable": true,
          "serverrequired": true
        },
        "name": {
          "name": "name",
          "type": "string"
        },
        "epoch": {
          "name": "epoch",
          "type": "uinteger",
          "serverrequired": true
        },
        "self": {
          "name": "self",
          "type": "url",
          "readonly": true,
          "serverrequired": true
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
        "origin": {
          "name": "origin",
          "type": "uri"
        },
        "createdby": {
          "name": "createdby",
          "type": "string",
          "readonly": true
        },
        "createdon": {
          "name": "createdon",
          "type": "timestamp",
          "readonly": true
        },
        "modifiedby": {
          "name": "modifiedby",
          "type": "string",
          "readonly": true
        },
        "modifiedon": {
          "name": "modifiedon",
          "type": "timestamp",
          "readonly": true
        }
      },
      "resources": {
        "rms1": {
          "plural": "rms1",
          "singular": "rm1",
          "maxversions": 2,
          "setversionid": true,
          "setdefault": true,
          "hasdocument": true,
          "attributes": {
            "id": {
              "name": "id",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            },
            "name": {
              "name": "name",
              "type": "string"
            },
            "epoch": {
              "name": "epoch",
              "type": "uinteger",
              "serverrequired": true
            },
            "self": {
              "name": "self",
              "type": "url",
              "readonly": true,
              "serverrequired": true
            },
            "isdefault": {
              "name": "isdefault",
              "type": "boolean",
              "readonly": true
            },
            "defaultversionid": {
              "name": "defaultversionid",
              "type": "string",
              "readonly": true
            },
            "defaultversionurl": {
              "name": "defaultversionurl",
              "type": "url",
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
            "origin": {
              "name": "origin",
              "type": "uri"
            },
            "createdby": {
              "name": "createdby",
              "type": "string",
              "readonly": true
            },
            "createdon": {
              "name": "createdon",
              "type": "timestamp",
              "readonly": true
            },
            "modifiedby": {
              "name": "modifiedby",
              "type": "string",
              "readonly": true
            },
            "modifiedon": {
              "name": "modifiedon",
              "type": "timestamp",
              "readonly": true
            },
            "contenttype": {
              "name": "contenttype",
              "type": "string"
            }
          }
        },
        "rms2": {
          "plural": "rms2",
          "singular": "rm2",
          "maxversions": 3,
          "setversionid": true,
          "setdefault": true,
          "hasdocument": true,
          "attributes": {
            "id": {
              "name": "id",
              "type": "string",
              "immutable": true,
              "serverrequired": true
            },
            "name": {
              "name": "name",
              "type": "string"
            },
            "epoch": {
              "name": "epoch",
              "type": "uinteger",
              "serverrequired": true
            },
            "self": {
              "name": "self",
              "type": "url",
              "readonly": true,
              "serverrequired": true
            },
            "isdefault": {
              "name": "isdefault",
              "type": "boolean",
              "readonly": true
            },
            "defaultversionid": {
              "name": "defaultversionid",
              "type": "string",
              "readonly": true
            },
            "defaultversionurl": {
              "name": "defaultversionurl",
              "type": "url",
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
            "origin": {
              "name": "origin",
              "type": "uri"
            },
            "createdby": {
              "name": "createdby",
              "type": "string",
              "readonly": true
            },
            "createdon": {
              "name": "createdon",
              "type": "timestamp",
              "readonly": true
            },
            "modifiedby": {
              "name": "modifiedby",
              "type": "string",
              "readonly": true
            },
            "modifiedon": {
              "name": "modifiedon",
              "type": "timestamp",
              "readonly": true
            },
            "contenttype": {
              "name": "contenttype",
              "type": "string"
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

	gm, _ := reg.Model.AddGroupModel("dirs1", "dir1")
	gm.AddResourceModel("files", "file", 2, true, false, true)

	d, _ := reg.AddGroup("dirs1", "d1")
	f, _ := d.AddResource("files", "f1", "v1")
	f.AddVersion("v2", true)
	d, _ = reg.AddGroup("dirs1", "d2")
	f, _ = d.AddResource("files", "f2", "v1")
	f.AddVersion("v1.1", true)

	gm2, _ := reg.Model.AddGroupModel("dirs2", "dir2")
	gm2.AddResourceModel("files", "file", 0, false, true, true)
	d2, _ := reg.AddGroup("dirs2", "d2")
	d2.AddResource("files", "f2", "v1")

	// /dirs1/d1/f1/v1
	//            /v2
	//       /d2/f2/v1
	//             v1.1
	// /dirs2/f2/f2/v1

	xCheckGet(t, reg, "?model&inline", `{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestMultModel2Create",
  "epoch": 1,
  "self": "http://localhost:8181/",
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
      "id": {
        "name": "id",
        "type": "string",
        "immutable": true,
        "serverrequired": true
      },
      "name": {
        "name": "name",
        "type": "string"
      },
      "epoch": {
        "name": "epoch",
        "type": "uinteger",
        "serverrequired": true
      },
      "self": {
        "name": "self",
        "type": "url",
        "readonly": true,
        "serverrequired": true
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
      "createdby": {
        "name": "createdby",
        "type": "string",
        "readonly": true
      },
      "createdon": {
        "name": "createdon",
        "type": "timestamp",
        "readonly": true
      },
      "modifiedby": {
        "name": "modifiedby",
        "type": "string",
        "readonly": true
      },
      "modifiedon": {
        "name": "modifiedon",
        "type": "timestamp",
        "readonly": true
      }
    },
    "groups": {
      "dirs1": {
        "plural": "dirs1",
        "singular": "dir1",
        "attributes": {
          "id": {
            "name": "id",
            "type": "string",
            "immutable": true,
            "serverrequired": true
          },
          "name": {
            "name": "name",
            "type": "string"
          },
          "epoch": {
            "name": "epoch",
            "type": "uinteger",
            "serverrequired": true
          },
          "self": {
            "name": "self",
            "type": "url",
            "readonly": true,
            "serverrequired": true
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
          "origin": {
            "name": "origin",
            "type": "uri"
          },
          "createdby": {
            "name": "createdby",
            "type": "string",
            "readonly": true
          },
          "createdon": {
            "name": "createdon",
            "type": "timestamp",
            "readonly": true
          },
          "modifiedby": {
            "name": "modifiedby",
            "type": "string",
            "readonly": true
          },
          "modifiedon": {
            "name": "modifiedon",
            "type": "timestamp",
            "readonly": true
          }
        },
        "resources": {
          "files": {
            "plural": "files",
            "singular": "file",
            "maxversions": 2,
            "setversionid": true,
            "setdefault": false,
            "hasdocument": true,
            "attributes": {
              "id": {
                "name": "id",
                "type": "string",
                "immutable": true,
                "serverrequired": true
              },
              "name": {
                "name": "name",
                "type": "string"
              },
              "epoch": {
                "name": "epoch",
                "type": "uinteger",
                "serverrequired": true
              },
              "self": {
                "name": "self",
                "type": "url",
                "readonly": true,
                "serverrequired": true
              },
              "isdefault": {
                "name": "isdefault",
                "type": "boolean",
                "readonly": true
              },
              "defaultversionid": {
                "name": "defaultversionid",
                "type": "string",
                "readonly": true
              },
              "defaultversionurl": {
                "name": "defaultversionurl",
                "type": "url",
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
              "origin": {
                "name": "origin",
                "type": "uri"
              },
              "createdby": {
                "name": "createdby",
                "type": "string",
                "readonly": true
              },
              "createdon": {
                "name": "createdon",
                "type": "timestamp",
                "readonly": true
              },
              "modifiedby": {
                "name": "modifiedby",
                "type": "string",
                "readonly": true
              },
              "modifiedon": {
                "name": "modifiedon",
                "type": "timestamp",
                "readonly": true
              },
              "contenttype": {
                "name": "contenttype",
                "type": "string"
              }
            }
          }
        }
      },
      "dirs2": {
        "plural": "dirs2",
        "singular": "dir2",
        "attributes": {
          "id": {
            "name": "id",
            "type": "string",
            "immutable": true,
            "serverrequired": true
          },
          "name": {
            "name": "name",
            "type": "string"
          },
          "epoch": {
            "name": "epoch",
            "type": "uinteger",
            "serverrequired": true
          },
          "self": {
            "name": "self",
            "type": "url",
            "readonly": true,
            "serverrequired": true
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
          "origin": {
            "name": "origin",
            "type": "uri"
          },
          "createdby": {
            "name": "createdby",
            "type": "string",
            "readonly": true
          },
          "createdon": {
            "name": "createdon",
            "type": "timestamp",
            "readonly": true
          },
          "modifiedby": {
            "name": "modifiedby",
            "type": "string",
            "readonly": true
          },
          "modifiedon": {
            "name": "modifiedon",
            "type": "timestamp",
            "readonly": true
          }
        },
        "resources": {
          "files": {
            "plural": "files",
            "singular": "file",
            "maxversions": 0,
            "setversionid": false,
            "setdefault": true,
            "hasdocument": true,
            "attributes": {
              "id": {
                "name": "id",
                "type": "string",
                "immutable": true,
                "serverrequired": true
              },
              "name": {
                "name": "name",
                "type": "string"
              },
              "epoch": {
                "name": "epoch",
                "type": "uinteger",
                "serverrequired": true
              },
              "self": {
                "name": "self",
                "type": "url",
                "readonly": true,
                "serverrequired": true
              },
              "isdefault": {
                "name": "isdefault",
                "type": "boolean",
                "readonly": true
              },
              "defaultversionid": {
                "name": "defaultversionid",
                "type": "string",
                "readonly": true
              },
              "defaultversionurl": {
                "name": "defaultversionurl",
                "type": "url",
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
              "origin": {
                "name": "origin",
                "type": "uri"
              },
              "createdby": {
                "name": "createdby",
                "type": "string",
                "readonly": true
              },
              "createdon": {
                "name": "createdon",
                "type": "timestamp",
                "readonly": true
              },
              "modifiedby": {
                "name": "modifiedby",
                "type": "string",
                "readonly": true
              },
              "modifiedon": {
                "name": "modifiedon",
                "type": "timestamp",
                "readonly": true
              },
              "contenttype": {
                "name": "contenttype",
                "type": "string"
              }
            }
          }
        }
      }
    }
  },

  "dirs1": {
    "d1": {
      "id": "d1",
      "epoch": 1,
      "self": "http://localhost:8181/dirs1/d1",

      "files": {
        "f1": {
          "id": "f1",
          "epoch": 1,
          "self": "http://localhost:8181/dirs1/d1/files/f1?meta",
          "defaultversionid": "v2",
          "defaultversionurl": "http://localhost:8181/dirs1/d1/files/f1/versions/v2?meta",

          "versions": {
            "v1": {
              "id": "v1",
              "epoch": 1,
              "self": "http://localhost:8181/dirs1/d1/files/f1/versions/v1?meta"
            },
            "v2": {
              "id": "v2",
              "epoch": 1,
              "self": "http://localhost:8181/dirs1/d1/files/f1/versions/v2?meta",
              "isdefault": true
            }
          },
          "versionscount": 2,
          "versionsurl": "http://localhost:8181/dirs1/d1/files/f1/versions"
        }
      },
      "filescount": 1,
      "filesurl": "http://localhost:8181/dirs1/d1/files"
    },
    "d2": {
      "id": "d2",
      "epoch": 1,
      "self": "http://localhost:8181/dirs1/d2",

      "files": {
        "f2": {
          "id": "f2",
          "epoch": 1,
          "self": "http://localhost:8181/dirs1/d2/files/f2?meta",
          "defaultversionid": "v1.1",
          "defaultversionurl": "http://localhost:8181/dirs1/d2/files/f2/versions/v1.1?meta",

          "versions": {
            "v1": {
              "id": "v1",
              "epoch": 1,
              "self": "http://localhost:8181/dirs1/d2/files/f2/versions/v1?meta"
            },
            "v1.1": {
              "id": "v1.1",
              "epoch": 1,
              "self": "http://localhost:8181/dirs1/d2/files/f2/versions/v1.1?meta",
              "isdefault": true
            }
          },
          "versionscount": 2,
          "versionsurl": "http://localhost:8181/dirs1/d2/files/f2/versions"
        }
      },
      "filescount": 1,
      "filesurl": "http://localhost:8181/dirs1/d2/files"
    }
  },
  "dirs1count": 2,
  "dirs1url": "http://localhost:8181/dirs1",
  "dirs2": {
    "d2": {
      "id": "d2",
      "epoch": 1,
      "self": "http://localhost:8181/dirs2/d2",

      "files": {
        "f2": {
          "id": "f2",
          "epoch": 1,
          "self": "http://localhost:8181/dirs2/d2/files/f2?meta",
          "defaultversionid": "v1",
          "defaultversionurl": "http://localhost:8181/dirs2/d2/files/f2/versions/v1?meta",

          "versions": {
            "v1": {
              "id": "v1",
              "epoch": 1,
              "self": "http://localhost:8181/dirs2/d2/files/f2/versions/v1?meta",
              "isdefault": true
            }
          },
          "versionscount": 1,
          "versionsurl": "http://localhost:8181/dirs2/d2/files/f2/versions"
        }
      },
      "filescount": 1,
      "filesurl": "http://localhost:8181/dirs2/d2/files"
    }
  },
  "dirs2count": 1,
  "dirs2url": "http://localhost:8181/dirs2"
}
`)

	gm, _ = reg.Model.AddGroupModel("dirs0", "dir0")
	gm.AddResourceModel("files", "file", 2, true, false, true)
	gm, _ = reg.Model.AddGroupModel("dirs3", "dir3")
	gm.AddResourceModel("files", "file", 2, true, false, true)

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs0":{},"dirs1":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}},"dirs2":{"d2":{"files":{"f2":{"versions":{"v1":{}}}}}},"dirs3":{}}`)

	gm, _ = reg.Model.AddGroupModel("dirs15", "dir15")
	gm.AddResourceModel("files", "file", 2, true, false, true)

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs0":{},"dirs1":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}},"dirs15":{},"dirs2":{"d2":{"files":{"f2":{"versions":{"v1":{}}}}}},"dirs3":{}}`)

	gm, _ = reg.Model.AddGroupModel("dirs01", "dir01")
	gm, _ = reg.Model.AddGroupModel("dirs02", "dir02")
	gm, _ = reg.Model.AddGroupModel("dirs14", "dir014")
	gm, _ = reg.Model.AddGroupModel("dirs16", "dir016")
	gm, _ = reg.Model.AddGroupModel("dirs4", "dir4")
	gm, _ = reg.Model.AddGroupModel("dirs5", "dir5")

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs0":{},"dirs01":{},"dirs02":{},"dirs1":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}},"dirs14":{},"dirs15":{},"dirs16":{},"dirs2":{"d2":{"files":{"f2":{"versions":{"v1":{}}}}}},"dirs3":{},"dirs4":{},"dirs5":{}}`)
}
