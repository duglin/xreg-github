package tests

import (
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestBasicFilters(t *testing.T) {
	reg, _ := registry.NewRegistry("TestBasicFilters")
	defer reg.Delete()

	gm, _ := reg.AddGroupModel("dirs", "dir", "")
	gm.AddResourceModel("files", "file", 0, true, true)
	d, _ := reg.AddGroup("dirs", "d1")
	f, _ := d.AddResource("files", "f1", "v1")
	f.AddVersion("v2")
	d, _ = reg.AddGroup("dirs", "d2")
	f, _ = d.AddResource("files", "f2", "v1")
	f.AddVersion("v1.1")

	reg.Set("tags.reg1", "1ger")
	f.Set("tags.file1", "1elif")

	// /dirs/d1/f1/v1
	//            /v2
	//      /d2/f2/v1
	//             v1.1

	tests := []struct {
		Name string
		URL  string
		Exp  string
	}{
		{
			Name: "No Filter",
			URL:  "?",
			Exp: `{
  "id": "TestBasicFilters",
  "self": "http:///",
  "tags": {
    "reg1": "1ger"
  },

  "dirsCount": 2,
  "dirsUrl": "http:///dirs"
}
`,
		},
		{
			Name: "Inline - No Filter",
			URL:  "?inline&oneline",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "2 leaves match",
			URL:  "?inline&oneline&filter=dirs.files.versions.id=v1",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{}}}}}}}`,
		},
		{
			Name: "Just one leaf - v2",
			URL:  "?inline&oneline&filter=dirs.files.versions.id=v2",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v2":{}}}}}}}`,
		},
		{
			Name: "filter at file level",
			URL:  "?inline&oneline&filter=dirs.files.id=f2",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "get groups, filter at resource level",
			URL:  "dirs?inline&oneline&filter=files.id=f2",
			Exp:  `{"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}`,
		},
		{ // Test some filtering at the root of the GET
			Name: "Get/filter root - match ",
			URL:  "?inline&oneline&filter=id=TestBasicFilters",
			// Return entire tree
			Exp: `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "Get/filter root, no match",
			URL:  "?inline&oneline&filter=id=xxx",
			// Nothing matched so 404
			Exp: `404: Not found`,
		},
		{
			Name: "Get root, filter group coll - match",
			URL:  "?inline&oneline&filter=dirs.id=d1",
			// Just root + dirs/d1
			Exp: `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}}}}`,
		},
		{
			Name: "Get root, filter group coll - no match",
			URL:  "?inline&oneline&filter=dirs.id=xxx",
			// Nothing, matched, so 404
			Exp: `404: Not found`,
		},
		{
			Name: "Get/filter group coll - match",
			URL:  "dirs?inline&oneline&filter=id=d1",
			// dirs coll with just d1
			Exp: `{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}}}`,
		},
		{
			Name: "Get/filter group coll - no match",
			URL:  "dirs?inline&oneline&filter=id=xxx",
			Exp:  "{}",
		},
		{
			Name: "Get/filter group entity - match",
			URL:  "dirs/d1?inline&oneline&filter=id=d1",
			// entire d1 group
			Exp: `{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}}`,
		},
		{
			Name: "Get/filter group entity - no match",
			URL:  "dirs/d1?inline&oneline&filter=id=xxx",
			// Nothing, matched, so 404
			Exp: `404: Not found`,
		},
		{
			Name: "Get group entity, filter resource - match",
			URL:  "dirs/d1?inline&oneline&filter=files.id=f1",
			// entire d1
			Exp: `{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}}`,
		},
		{
			Name: "Get group entity, filter resource - no match",
			URL:  "dirs/d1?inline&oneline&filter=files.id=xxx",
			// Nothing, matched, so 404
			Exp: `404: Not found`,
		},
		{
			Name: "Get/filter version coll - match",
			URL:  "dirs/d1/files/f1/versions?inline&oneline&filter=id=v1",
			Exp:  `{"v1":{}}`,
		},
		{
			Name: "Get/filter version coll - no match",
			URL:  "dirs/d1/files/f1/versions?inline&oneline&filter=id=xxx",
			Exp:  "{}",
		},
		{
			Name: "Get/filter version - match",
			URL:  "dirs/d1/files/f1/versions/v1?inline&filter=id=v1",
			Exp: `{
  "id": "v1",
  "self": "http:///dirs/d1/files/f1/versions/v1"
}
`,
		},
		{
			Name: "Get/filter version - no match",
			URL:  "dirs/d1/files/f1/versions/v1?inline&oneline&filter=id=xxx",
			// Nothing, matched, so 404
			Exp: `404: Not found`,
		},

		// Some tag filters
		{
			Name: "Get/filter reg.tags - no match",
			URL:  "?filter=tags.reg1=xxx",
			// Nothing, matched, so 404
			Exp: "404: Not found\n",
		},
		{
			Name: "Get/filter reg.tags - match",
			URL:  "?filter=tags.reg1=1ger",
			Exp: `{
  "id": "TestBasicFilters",
  "self": "http:///",
  "tags": {
    "reg1": "1ger"
  },

  "dirsCount": 2,
  "dirsUrl": "http:///dirs"
}
`,
		},
		{
			Name: "Get/filter tags",
			URL:  "?filter=dirs.files.tags.file1=1elif",
			Exp: `{
  "id": "TestBasicFilters",
  "self": "http:///",
  "tags": {
    "reg1": "1ger"
  },

  "dirsCount": 1,
  "dirsUrl": "http:///dirs"
}
`,
		},
		{
			Name: "Get/filter dir file.tags - match",
			URL:  "?inline&filter=dirs.files.tags.file1=1elif",
			Exp: `{
  "id": "TestBasicFilters",
  "self": "http:///",
  "tags": {
    "reg1": "1ger"
  },

  "dirs": {
    "d2": {
      "id": "d2",
      "self": "http:///dirs/d2",

      "files": {
        "f2": {
          "id": "f2",
          "self": "http:///dirs/d2/files/f2",
          "latestId": "v1.1",
          "latestUrl": "http:///dirs/d2/files/f2/versions/v1.1",
          "tags": {
            "file1": "1elif"
          },

          "versions": {
            "v1": {
              "id": "v1",
              "self": "http:///dirs/d2/files/f2/versions/v1"
            },
            "v1.1": {
              "id": "v1.1",
              "self": "http:///dirs/d2/files/f2/versions/v1.1",
              "tags": {
                "file1": "1elif"
              }
            }
          },
          "versionsCount": 2,
          "versionsUrl": "http:///dirs/d2/files/f2/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http:///dirs/d2/files"
    }
  },
  "dirsCount": 1,
  "dirsUrl": "http:///dirs"
}
`,
		},
		{
			Name: "Get/filter dir file.tags - no match empty string",
			URL:  "?inline&filter=dirs.files.tags.file1=",
			Exp:  "404: Not found\n",
		},
		{
			Name: "Get/filter dir file.tags.xxx - no match empty string",
			URL:  "?inline&filter=dirs.files.tags.xxx=",
			Exp:  "404: Not found\n",
		},
		{
			Name: "Get/filter dir file.tags.xxx - no match non-empty string",
			URL:  "?inline&filter=dirs.files.tags.xxx",
			Exp:  "404: Not found\n",
		},
		{
			Name: "Get/filter dir file.tags - match non-empty string",
			URL:  "?inline&filter=dirs.files.tags.file1",
			Exp: `{
  "id": "TestBasicFilters",
  "self": "http:///",
  "tags": {
    "reg1": "1ger"
  },

  "dirs": {
    "d2": {
      "id": "d2",
      "self": "http:///dirs/d2",

      "files": {
        "f2": {
          "id": "f2",
          "self": "http:///dirs/d2/files/f2",
          "latestId": "v1.1",
          "latestUrl": "http:///dirs/d2/files/f2/versions/v1.1",
          "tags": {
            "file1": "1elif"
          },

          "versions": {
            "v1": {
              "id": "v1",
              "self": "http:///dirs/d2/files/f2/versions/v1"
            },
            "v1.1": {
              "id": "v1.1",
              "self": "http:///dirs/d2/files/f2/versions/v1.1",
              "tags": {
                "file1": "1elif"
              }
            }
          },
          "versionsCount": 2,
          "versionsUrl": "http:///dirs/d2/files/f2/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http:///dirs/d2/files"
    }
  },
  "dirsCount": 1,
  "dirsUrl": "http:///dirs"
}
`,
		},
	}

	for _, test := range tests {
		xCheckGet(t, reg, test.URL, test.Exp)
	}
}
