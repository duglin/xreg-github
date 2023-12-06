package tests

import (
	"testing"
	// "github.com/duglin/xreg-github/registry"
)

func TestBasicFilters(t *testing.T) {
	reg := NewRegistry("TestBasicFilters")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)
	d, _ := reg.AddGroup("dirs", "d1")
	f, _ := d.AddResource("files", "f1", "v1")
	f.AddVersion("v2")
	d, _ = reg.AddGroup("dirs", "d2")
	f, _ = d.AddResource("files", "f2", "v1")
	f.AddVersion("v1.1")

	reg.Set("labels.reg1", "1ger")
	f.Set("labels.file1", "1elif")

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
  "specVersion": "0.5",
  "id": "TestBasicFilters",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "labels": {
    "reg1": "1ger"
  },

  "dirsCount": 2,
  "dirsUrl": "http://localhost:8181/dirs"
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
			Exp: `Not found`,
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
			Exp: `Not found`,
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
			Exp: `Not found`,
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
			Exp: `Not found`,
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
			URL:  "dirs/d1/files/f1/versions/v1?inline&filter=id=v1&meta",
			Exp: `{
  "id": "v1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1"
}
`,
		},
		{
			Name: "Get/filter version - no match",
			URL:  "dirs/d1/files/f1/versions/v1?inline&oneline&filter=id=xxx&meta",
			// Nothing, matched, so 404
			Exp: `Not found`,
		},

		// Some tag filters
		{
			Name: "Get/filter reg.labels - no match",
			URL:  "?filter=labels.reg1=xxx",
			// Nothing, matched, so 404
			Exp: "Not found\n",
		},
		{
			Name: "Get/filter reg.labels - match",
			URL:  "?filter=labels.reg1=1ger",
			Exp: `{
  "specVersion": "0.5",
  "id": "TestBasicFilters",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "labels": {
    "reg1": "1ger"
  },

  "dirsCount": 2,
  "dirsUrl": "http://localhost:8181/dirs"
}
`,
		},
		{
			Name: "Get/filter labels",
			URL:  "?filter=dirs.files.labels.file1=1elif",
			Exp: `{
  "specVersion": "0.5",
  "id": "TestBasicFilters",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "labels": {
    "reg1": "1ger"
  },

  "dirsCount": 1,
  "dirsUrl": "http://localhost:8181/dirs"
}
`,
		},
		{
			Name: "Get/filter dir file.labels - match",
			URL:  "?inline&filter=dirs.files.labels.file1=1elif",
			Exp: `{
  "specVersion": "0.5",
  "id": "TestBasicFilters",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "labels": {
    "reg1": "1ger"
  },

  "dirs": {
    "d2": {
      "id": "d2",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d2",

      "files": {
        "f2": {
          "id": "f2",
          "epoch": 1,
          "self": "http://localhost:8181/dirs/d2/files/f2",
          "latestVersionId": "v1.1",
          "latestVersionUrl": "http://localhost:8181/dirs/d2/files/f2/versions/v1.1",
          "labels": {
            "file1": "1elif"
          },

          "versions": {
            "v1": {
              "id": "v1",
              "epoch": 1,
              "self": "http://localhost:8181/dirs/d2/files/f2/versions/v1"
            },
            "v1.1": {
              "id": "v1.1",
              "epoch": 1,
              "self": "http://localhost:8181/dirs/d2/files/f2/versions/v1.1",
              "latest": true,
              "labels": {
                "file1": "1elif"
              }
            }
          },
          "versionsCount": 2,
          "versionsUrl": "http://localhost:8181/dirs/d2/files/f2/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http://localhost:8181/dirs/d2/files"
    }
  },
  "dirsCount": 1,
  "dirsUrl": "http://localhost:8181/dirs"
}
`,
		},
		{
			Name: "Get/filter dir file.labels - no match empty string",
			URL:  "?inline&filter=dirs.files.labels.file1=",
			Exp:  "Not found\n",
		},
		{
			Name: "Get/filter dir file.labels.xxx - no match empty string",
			URL:  "?inline&filter=dirs.files.labels.xxx=",
			Exp:  "Not found\n",
		},
		{
			Name: "Get/filter dir file.labels.xxx - no match non-empty string",
			URL:  "?inline&filter=dirs.files.labels.xxx",
			Exp:  "Not found\n",
		},
		{
			Name: "Get/filter dir file.labels - match non-empty string",
			URL:  "?inline&filter=dirs.files.labels.file1",
			Exp: `{
  "specVersion": "0.5",
  "id": "TestBasicFilters",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "labels": {
    "reg1": "1ger"
  },

  "dirs": {
    "d2": {
      "id": "d2",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d2",

      "files": {
        "f2": {
          "id": "f2",
          "epoch": 1,
          "self": "http://localhost:8181/dirs/d2/files/f2",
          "latestVersionId": "v1.1",
          "latestVersionUrl": "http://localhost:8181/dirs/d2/files/f2/versions/v1.1",
          "labels": {
            "file1": "1elif"
          },

          "versions": {
            "v1": {
              "id": "v1",
              "epoch": 1,
              "self": "http://localhost:8181/dirs/d2/files/f2/versions/v1"
            },
            "v1.1": {
              "id": "v1.1",
              "epoch": 1,
              "self": "http://localhost:8181/dirs/d2/files/f2/versions/v1.1",
              "latest": true,
              "labels": {
                "file1": "1elif"
              }
            }
          },
          "versionsCount": 2,
          "versionsUrl": "http://localhost:8181/dirs/d2/files/f2/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http://localhost:8181/dirs/d2/files"
    }
  },
  "dirsCount": 1,
  "dirsUrl": "http://localhost:8181/dirs"
}
`,
		},
	}

	for _, test := range tests {
		pass := xCheckGet(t, reg, test.URL, test.Exp)
		if !pass {
			t.Logf("Test name: %s", test.Name)
		}
	}
}

func TestANDORFilters(t *testing.T) {
	reg := NewRegistry("TestANDORFilters")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)
	d, _ := reg.AddGroup("dirs", "d1")
	f, _ := d.AddResource("files", "f1", "v1")
	f.AddVersion("v2")
	f.Set("name", "f1")
	d, _ = reg.AddGroup("dirs", "d2")
	f, _ = d.AddResource("files", "f2", "v1")
	f.AddVersion("v1.1")
	f.Set("name", "f2")

	gm, _ = reg.Model.AddGroupModel("schemaGroups", "schemaGroup")
	gm.AddResourceModel("schemas", "schema", 0, true, true, true)
	sg, _ := reg.AddGroup("schemaGroups", "sg1")
	s, _ := sg.AddResource("schemas", "s1", "v1.0")
	s.AddVersion("v2.0")

	reg.Set("labels.reg1", "1ger")
	f.Set("labels.file1", "1elif")

	// /dirs/d1/f1/v1     f1.name=f1
	//            /v2
	//      /d2/f2/v1     f2.name=f2
	//             v1.1
	// /schemaGroups/sg1/schemas/s1/v1.0
	//                             /v2.0

	tests := []struct {
		Name string
		URL  string
		Exp  string
	}{
		{
			Name: "AND same obj/level - match",
			URL:  "?oneline&inline&filter=dirs.files.id=f1,dirs.files.name=f1",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}}},"schemaGroups":{}}`,
		},
		{
			Name: "AND same obj/level - no match",
			URL:  "?oneline&inline&filter=dirs.files.id=f1,dirs.files.name=f2",
			Exp:  `Not found`,
		},
		{
			Name: "OR same obj/level - match",
			URL:  "?oneline&inline&filter=dirs.files.id=f1&filter=dirs.files.name=f1",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}}},"schemaGroups":{}}`,
		},
		{
			Name: "multi result 2 levels down - match",
			URL:  "?oneline&inline&filter=dirs.files.versions.id=v1",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{}}}}}},"schemaGroups":{}}`,
		},
		{
			Name: "path + multi result 2 levels down - match",
			URL:  "dirs?oneline&inline&filter=files.versions.id=v1",
			Exp:  `{"d1":{"files":{"f1":{"versions":{"v1":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{}}}}}}`,
		},
		{
			Name: "path + multi result 2 levels down - no match",
			URL:  "dirs?oneline&inline&filter=files.versions.id=xxx",
			Exp:  `{}`,
		},

		// Span group types
		{
			Name: "dirs and schemaGroups - match both",
			URL:  "?oneline&inline&filter=dirs.id=d1&filter=schemaGroups.id=sg1",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}}},"schemaGroups":{"sg1":{"schemas":{"s1":{"versions":{"v1.0":{},"v2.0":{}}}}}}}`,
		},
		{
			Name: "dirs and schemaGroups - match first",
			URL:  "?oneline&inline&filter=dirs.id=d1&filter=schemaGroups.id=xxx",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}}},"schemaGroups":{}}`,
		},
		{
			Name: "dirs and schemaGroups - match second",
			URL:  "?oneline&inline&filter=dirs.id=xxx&filter=schemaGroups.id=sg1",
			Exp:  `{"dirs":{},"schemaGroups":{"sg1":{"schemas":{"s1":{"versions":{"v1.0":{},"v2.0":{}}}}}}}`,
		},
		{
			Name: "dirsOR and schemaGroupsOR - match first",
			URL:  "?oneline&inline&filter=dirs.files.id=f1,dirs.files.versions.id=v2&filter=schemaGroups.schemas.versions.id=v1.0,schemaGroups.schemas.versions.id=v2.0",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v2":{}}}}}},"schemaGroups":{}}`,
		},
		{
			Name: "dirsOR and schemaGroupsOR - match second",
			URL:  "?oneline&inline&filter=dirs.files.id=f1,dirs.files.versions.id=xxx&filter=schemaGroups.schemas.versions.id=v2.0,schemaGroups.schemas.latestVersionId=v2.0",
			Exp:  `{"dirs":{},"schemaGroups":{"sg1":{"schemas":{"s1":{"versions":{"v2.0":{}}}}}}}`,
		},
		{
			Name: "dirsOR and schemaGroupsOR - both match",
			URL:  "?oneline&inline&filter=dirs.files.id=f1,dirs.files.versions.id=v2&filter=schemaGroups.schemas.versions.id=v2.0,schemaGroups.schemas.latestVersionId=v2.0",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v2":{}}}}}},"schemaGroups":{"sg1":{"schemas":{"s1":{"versions":{"v2.0":{}}}}}}}`,
		},
	}

	for _, test := range tests {
		pass := xCheckGet(t, reg, test.URL, test.Exp)
		if !pass {
			t.Logf("Test name: %s", test.Name)
		}
	}
}
