package tests

import (
	"encoding/json"
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestCapabilitySimple(t *testing.T) {
	reg := NewRegistry("TestCapabilitySimple")
	defer PassDeleteReg(t, reg)

	tests := []struct {
		Name string
		Cap  string
		Exp  string
	}{
		{
			Name: "empty",
			Cap:  `{}`,
			Exp: `{
  "flags": [],
  "mutable": [],
  "pagination": false,
  "schemas": [
    "xregistry-json"
  ],
  "shortself": false,
  "specversions": [
    "0.5"
  ]
}`,
		},
		{
			Name: "full mutable",
			Cap:  `{"mutable":["entities","model","capabilities"]}`,
			Exp: `{
  "flags": [],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "schemas": [
    "xregistry-json"
  ],
  "shortself": false,
  "specversions": [
    "0.5"
  ]
}`,
		},
		{
			Name: "dup mutable",
			Cap:  `{"mutable":["entities","model","entities","capabilities"]}`,
			Exp: `{
  "flags": [],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "schemas": [
    "xregistry-json"
  ],
  "shortself": false,
  "specversions": [
    "0.5"
  ]
}`,
		},
		{
			Name: "star mutable",
			Cap:  `{"mutable":["*"]}`,
			Exp: `{
  "flags": [],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "schemas": [
    "xregistry-json"
  ],
  "shortself": false,
  "specversions": [
    "0.5"
  ]
}`,
		},
		{
			Name: "mutable empty",
			Cap:  `{"mutable":[]}`,
			Exp: `{
  "flags": [],
  "mutable": [],
  "pagination": false,
  "schemas": [
    "xregistry-json"
  ],
  "shortself": false,
  "specversions": [
    "0.5"
  ]
}`,
		},
		{
			Name: "star mutable-bad",
			Cap:  `{"mutable":["model","*"]}`,
			Exp:  `"*" must be the only value specified for "mutable"`,
		},
		{
			Name: "bad mutable-1",
			Cap:  `{"mutable":["xx"]}`,
			Exp:  `Unknown "mutable" value: "xx"`,
		},
		{
			Name: "bad mutable-2",
			Cap:  `{"mutable":["model", "xx"]}`,
			Exp:  `Unknown "mutable" value: "xx"`,
		},
		{
			Name: "bad mutable-3",
			Cap:  `{"mutable":["aa", "model"]}`,
			Exp:  `Unknown "mutable" value: "aa"`,
		},
		{
			Name: "bad mutable-4",
			Cap:  `{"mutable":["entities", "ff", "model"]}`,
			Exp:  `Unknown "mutable" value: "ff"`,
		},

		{
			Name: "missing schema",
			Cap:  `{"schemas":[]}`,
			Exp:  `"schemas" must contain "xRegistry-json"`,
		},
		{
			Name: "missing specversion",
			Cap:  `{"specversions":[]}`,
			Exp:  `"specversions" must contain "` + registry.SPECVERSION + `"`,
		},

		{
			Name: "extra key",
			Cap:  `{"pagination": true, "bad": true}`,
			Exp:  `Unknown capability: "bad"`,
		},
	}

	for _, test := range tests {
		c, err := registry.ParseCapabilitiesJSON([]byte(test.Cap))
		if err == nil {
			err = c.Validate()
		}
		res := ""
		if err != nil {
			res = err.Error()
		} else {
			buf, _ := json.MarshalIndent(c, "", "  ")
			res = string(buf)
		}
		xCheckEqual(t, test.Name, res, test.Exp)
	}
}

func TestCapabilityPath(t *testing.T) {
	reg := NewRegistry("TestCapabilityPath")
	defer PassDeleteReg(t, reg)

	xHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "flags": [
    "epoch",
    "export",
    "filter",
    "inline",
    "nested",
    "nodefaultversionid",
    "nodefaultversionsticky",
    "noepoch",
    "noreadonly",
    "schema",
    "setdefaultversionid"
  ],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "schemas": [
    "xregistry-json"
  ],
  "shortself": false,
  "specversions": [
    "0.5"
  ]
}
`)

	// Verify current epoch value
	xHTTP(t, reg, "GET", "/", ``, 200, `{
  "specversion": "0.5",
  "registryid": "TestCapabilityPath",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z"
}
`)

	// Try to clear it all - some can't be totally erased
	xHTTP(t, reg, "PUT", "/capabilities", `{}`, 200,
		`{
  "flags": [],
  "mutable": [],
  "pagination": false,
  "schemas": [
    "xregistry-json"
  ],
  "shortself": false,
  "specversions": [
    "0.5"
  ]
}
`)

	// Make sure the Registry epoch changed
	xHTTP(t, reg, "GET", "/", ``, 200, `{
  "specversion": "0.5",
  "registryid": "TestCapabilityPath",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
}
`)

	xHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "flags": [],
  "mutable": [],
  "pagination": false,
  "schemas": [
    "xregistry-json"
  ],
  "shortself": false,
  "specversions": [
    "0.5"
  ]
}
`)

	// Setting to nulls
	xHTTP(t, reg, "PUT", "/capabilities", `{
  "flags": null,
  "mutable": null,
  "pagination": null,
  "schemas": null,
  "shortself": null,
  "specversions": null
}`, 200,
		`{
  "flags": [],
  "mutable": [],
  "pagination": false,
  "schemas": [
    "xregistry-json"
  ],
  "shortself": false,
  "specversions": [
    "0.5"
  ]
}
`)

	xHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "flags": [],
  "mutable": [],
  "pagination": false,
  "schemas": [
    "xregistry-json"
  ],
  "shortself": false,
  "specversions": [
    "0.5"
  ]
}
`)

	// Testing setting everything to the default
	xHTTP(t, reg, "PUT", "/capabilities", `{
  "flags": [
    "epoch", "export", "filter", "inline", "nested", "nodefaultversionid",
    "nodefaultversionsticky", "noepoch", "noreadonly", "schema",
	"setdefaultversionid"
  ],
  "mutable": [ "capabilities", "entities", "model" ],
  "pagination": false,
  "schemas": [ "xregistry-json" ],
  "shortself": false,
  "specversions": [ "0.5" ]
}`, 200,
		`{
  "flags": [
    "epoch",
    "export",
    "filter",
    "inline",
    "nested",
    "nodefaultversionid",
    "nodefaultversionsticky",
    "noepoch",
    "noreadonly",
    "schema",
    "setdefaultversionid"
  ],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "schemas": [
    "xregistry-json"
  ],
  "shortself": false,
  "specversions": [
    "0.5"
  ]
}
`)

	xHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "flags": [
    "epoch",
    "export",
    "filter",
    "inline",
    "nested",
    "nodefaultversionid",
    "nodefaultversionsticky",
    "noepoch",
    "noreadonly",
    "schema",
    "setdefaultversionid"
  ],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "schemas": [
    "xregistry-json"
  ],
  "shortself": false,
  "specversions": [
    "0.5"
  ]
}
`)

	// Setting to minimal
	xHTTP(t, reg, "PUT", "/capabilities", `{
}`, 200,
		`{
  "flags": [],
  "mutable": [],
  "pagination": false,
  "schemas": [
    "xregistry-json"
  ],
  "shortself": false,
  "specversions": [
    "0.5"
  ]
}
`)

	xHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "flags": [],
  "mutable": [],
  "pagination": false,
  "schemas": [
    "xregistry-json"
  ],
  "shortself": false,
  "specversions": [
    "0.5"
  ]
}
`)

	// Test some bools
	xHTTP(t, reg, "PUT", "/capabilities", `{
	"pagination": true,
	"shortself": true
}`, 200, `{
  "flags": [],
  "mutable": [],
  "pagination": true,
  "schemas": [
    "xregistry-json"
  ],
  "shortself": true,
  "specversions": [
    "0.5"
  ]
}
`)

	xHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "flags": [],
  "mutable": [],
  "pagination": true,
  "schemas": [
    "xregistry-json"
  ],
  "shortself": true,
  "specversions": [
    "0.5"
  ]
}
`)

	// Setting some arrays to [] are an error because we can't do what they
	// asked - which is different from "null"/absent - which means "default"
	xHTTP(t, reg, "PUT", "/capabilities", `{ "schemas": [] }`,
		400, "\"schemas\" must contain \"xRegistry-json\"\n")

	// Setting some arrays to [] are an error because we can't do what they
	// asked - which is different from "null"/absent - which means "default"
	xHTTP(t, reg, "PUT", "/capabilities", `{ "specversions": [] }`,
		400, "\"specversions\" must contain \"0.5\"\n")

	// Unknown key
	xHTTP(t, reg, "PUT", "/capabilities", `{ "foo": [] }`,
		400, "Unknown capability: \"foo\"\n")
}

func TestCapabilityAttr(t *testing.T) {
	reg := NewRegistry("TestCapabilityAttr")
	defer PassDeleteReg(t, reg)

	// Verify epoch value
	xHTTP(t, reg, "GET", "/", ``, 200, `{
  "specversion": "0.5",
  "registryid": "TestCapabilityAttr",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z"
}
`)

	// Try to clear it all - some can't be totally erased.
	// Notice epoch value changed
	xHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities": {} }`, 200,
		`{
  "specversion": "0.5",
  "registryid": "TestCapabilityAttr",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "capabilities": {
    "flags": [],
    "mutable": [],
    "pagination": false,
    "schemas": [
      "xregistry-json"
    ],
    "shortself": false,
    "specversions": [
      "0.5"
    ]
  }
}
`)

	// Setting to nulls
	// notice ?inline is still disabled!
	xHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities": {
  "flags": null,
  "mutable": null,
  "pagination": null,
  "schemas": null,
  "shortself": null,
  "specversions": null
}}`, 200,
		`{
  "specversion": "0.5",
  "registryid": "TestCapabilityAttr",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
}
`)

	xHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "flags": [],
  "mutable": [],
  "pagination": false,
  "schemas": [
    "xregistry-json"
  ],
  "shortself": false,
  "specversions": [
    "0.5"
  ]
}
`)

	// Testing setting everything to the default
	// inline still disabled
	xHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities": {
  "flags": [
    "epoch", "export", "filter", "inline", "nested", "nodefaultversionid",
    "nodefaultversionsticky", "noepoch", "noreadonly", "schema",
	"setdefaultversionid"
  ],
  "mutable": [ "capabilities", "entities", "model" ],
  "pagination": false,
  "schemas": [ "xregistry-json" ],
  "shortself": false,
  "specversions": [ "0.5" ]
}}`, 200,
		`{
  "specversion": "0.5",
  "registryid": "TestCapabilityAttr",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
}
`)

	xHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "flags": [
    "epoch",
    "export",
    "filter",
    "inline",
    "nested",
    "nodefaultversionid",
    "nodefaultversionsticky",
    "noepoch",
    "noreadonly",
    "schema",
    "setdefaultversionid"
  ],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "schemas": [
    "xregistry-json"
  ],
  "shortself": false,
  "specversions": [
    "0.5"
  ]
}
`)

	// Setting to minimal
	// inline still enabled
	xHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities": {
  "flags": [],
  "mutable": [],
  "pagination": true,
  "schemas": ["xregistry-json"],
  "shortself": true,
  "specversions": ["0.5"]
}}`, 200,
		`{
  "specversion": "0.5",
  "registryid": "TestCapabilityAttr",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 5,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "capabilities": {
    "flags": [],
    "mutable": [],
    "pagination": true,
    "schemas": [
      "xregistry-json"
    ],
    "shortself": true,
    "specversions": [
      "0.5"
    ]
  }
}
`)

	xHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "flags": [],
  "mutable": [],
  "pagination": true,
  "schemas": [
    "xregistry-json"
  ],
  "shortself": true,
  "specversions": [
    "0.5"
  ]
}
`)

	// Setting some arrays to [] are an error because we can't do what they
	// asked - which is different from "null"/absent - which means "default"
	xHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities":
	    {"schemas": [] }}`,
		400, "\"schemas\" must contain \"xRegistry-json\"\n")

	// Setting some arrays to [] are an error because we can't do what they
	// asked - which is different from "null"/absent - which means "default"
	xHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities":
	    {"specversions": [] }}`,
		400, "\"specversions\" must contain \"0.5\"\n")

	// Unknown key
	xHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities":
	    {"foo": [] }}`,
		400, "Unknown capability: \"foo\"\n")

}

// "epoch", "export", "filter", "inline",
// "nested", "nodefaultversionid", "nodefaultversionsticky",
// "noepoch", "noreadonly", "schema", "setdefaultversionid"})

func TestCapabilityFlags(t *testing.T) {
	reg := NewRegistry("TestCapabilityFlags")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, false)

	xHTTP(t, reg, "PUT", "/capabilities", `{"mutable":["*"]}`, 200, `{
  "flags": [],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "schemas": [
    "xregistry-json"
  ],
  "shortself": false,
  "specversions": [
    "0.5"
  ]
}
`)

	// Create a test file
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// TODO add a test for ?export once we support it

	// Test ?filter & ?inline - notice value isn't even analyzed
	xHTTP(t, reg, "GET", "/dirs/d1/files?filter=foo&inline=bar", `{}`, 200, `{
  "f1": {
    "fileid": "f1",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/f1",
    "xid": "/dirs/d1/files/f1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versionscount": 1
  }
}
`)

	// Bad epoch should be ignored
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1?epoch=99", `{}`, 204, ``)

	// Test ?nested & ?setdefaultversionid
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1?nested&setdefaultversionid=x", `{
  "versions": { "v1":{}, "v2":{} }
}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// Test ?schema
	xHTTP(t, reg, "GET", "/model?schema=foo", ``, 200, `*`)

	// TODO nodefaultversionid, nodefaultversionsticky, noepoch, noreadonly
}
