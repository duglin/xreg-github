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
  "mutable": [],
  "pagination": false,
  "queryparameters": [],
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
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "queryparameters": [],
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
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "queryparameters": [],
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
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "queryparameters": [],
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
  "mutable": [],
  "pagination": false,
  "queryparameters": [],
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
