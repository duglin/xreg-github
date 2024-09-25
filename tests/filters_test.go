package tests

import (
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestBasicFilters(t *testing.T) {
	reg := NewRegistry("TestBasicFilters")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)
	_, err = gm.AddResourceModel("files", "file", 0, true, true, true)
	xNoErr(t, err)
	d, _ := reg.AddGroup("dirs", "d1")
	f, _ := d.AddResource("files", "f1", "v1")
	f.AddVersion("v2")
	d, _ = reg.AddGroup("dirs", "d2")
	f, _ = d.AddResource("files", "f2", "v1")
	f.AddVersion("v1.1")

	reg.SetSave("labels.reg1", "1ger")
	f.SetSave("labels.file1", "1elif")

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
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestBasicFilters",
  "self": "http://localhost:8181/",
  "epoch": 1,
  "labels": {
    "reg1": "1ger"
  },
  "createdat": "2024-12-01T12:00:00Z",
  "modifiedat": "2024-12-01T12:00:01Z",

  "dirscount": 2,
  "dirsurl": "http://localhost:8181/dirs"
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
			URL:  "dirs/d1/files/f1/versions/v1$meta?inline&filter=id=v1",
			Exp: `{
  "id": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$meta",
  "epoch": 1,
  "createdat": "2024-12-01T12:00:00Z",
  "modifiedat": "2024-12-01T12:00:00Z"
}
`,
		},
		{
			Name: "Get/filter version - no match",
			URL:  "dirs/d1/files/f1/versions/v1$meta?inline&oneline&filter=id=xxx",
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
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestBasicFilters",
  "self": "http://localhost:8181/",
  "epoch": 1,
  "labels": {
    "reg1": "1ger"
  },
  "createdat": "2024-12-01T12:00:00Z",
  "modifiedat": "2024-12-01T12:00:01Z",

  "dirscount": 2,
  "dirsurl": "http://localhost:8181/dirs"
}
`,
		},
		{
			Name: "Get/filter labels",
			URL:  "?filter=dirs.files.labels.file1=1elif",
			Exp: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestBasicFilters",
  "self": "http://localhost:8181/",
  "epoch": 1,
  "labels": {
    "reg1": "1ger"
  },
  "createdat": "2024-12-01T12:00:00Z",
  "modifiedat": "2024-12-01T12:00:01Z",

  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
}
`,
		},
		{
			Name: "Get/filter dir file.labels - match 1elif",
			URL:  "?inline&filter=dirs.files.labels.file1=1elif",
			Exp: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestBasicFilters",
  "self": "http://localhost:8181/",
  "epoch": 1,
  "labels": {
    "reg1": "1ger"
  },
  "createdat": "2024-12-01T12:00:01Z",
  "modifiedat": "2024-12-01T12:00:02Z",

  "dirs": {
    "d2": {
      "id": "d2",
      "self": "http://localhost:8181/dirs/d2",
      "epoch": 1,
      "createdat": "2024-12-01T12:00:02Z",
      "modifiedat": "2024-12-01T12:00:02Z",

      "files": {
        "f2": {
          "id": "f2",
          "self": "http://localhost:8181/dirs/d2/files/f2$meta",
          "epoch": 1,
          "labels": {
            "file1": "1elif"
          },
          "createdat": "2024-12-01T12:00:02Z",
          "modifiedat": "2024-12-01T12:00:02Z",

          "defaultversionid": "v1.1",
          "defaultversionurl": "http://localhost:8181/dirs/d2/files/f2/versions/v1.1$meta",

          "versions": {
            "v1": {
              "id": "v1",
              "self": "http://localhost:8181/dirs/d2/files/f2/versions/v1$meta",
              "epoch": 1,
              "createdat": "2024-12-01T12:00:02Z",
              "modifiedat": "2024-12-01T12:00:02Z"
            },
            "v1.1": {
              "id": "v1.1",
              "self": "http://localhost:8181/dirs/d2/files/f2/versions/v1.1$meta",
              "epoch": 1,
              "isdefault": true,
              "labels": {
                "file1": "1elif"
              },
              "createdat": "2024-12-01T12:00:02Z",
              "modifiedat": "2024-12-01T12:00:02Z"
            }
          },
          "versionscount": 2,
          "versionsurl": "http://localhost:8181/dirs/d2/files/f2/versions"
        }
      },
      "filescount": 1,
      "filesurl": "http://localhost:8181/dirs/d2/files"
    }
  },
  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
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
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestBasicFilters",
  "self": "http://localhost:8181/",
  "epoch": 1,
  "labels": {
    "reg1": "1ger"
  },
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirs": {
    "d2": {
      "id": "d2",
      "self": "http://localhost:8181/dirs/d2",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "files": {
        "f2": {
          "id": "f2",
          "self": "http://localhost:8181/dirs/d2/files/f2$meta",
          "epoch": 1,
          "labels": {
            "file1": "1elif"
          },
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",

          "defaultversionid": "v1.1",
          "defaultversionurl": "http://localhost:8181/dirs/d2/files/f2/versions/v1.1$meta",

          "versions": {
            "v1": {
              "id": "v1",
              "self": "http://localhost:8181/dirs/d2/files/f2/versions/v1$meta",
              "epoch": 1,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z"
            },
            "v1.1": {
              "id": "v1.1",
              "self": "http://localhost:8181/dirs/d2/files/f2/versions/v1.1$meta",
              "epoch": 1,
              "isdefault": true,
              "labels": {
                "file1": "1elif"
              },
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z"
            }
          },
          "versionscount": 2,
          "versionsurl": "http://localhost:8181/dirs/d2/files/f2/versions"
        }
      },
      "filescount": 1,
      "filesurl": "http://localhost:8181/dirs/d2/files"
    }
  },
  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
}
`,
		},
	}

	for _, test := range tests {
		t.Logf("Test name: %s", test.Name)
		xCheckGet(t, reg, test.URL, test.Exp)
	}
}

func TestANDORFilters(t *testing.T) {
	reg := NewRegistry("TestANDORFilters")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)
	_, err = gm.AddResourceModel("files", "file", 0, true, true, true)
	xNoErr(t, err)
	d, _ := reg.AddGroup("dirs", "d1")
	f, _ := d.AddResource("files", "f1", "v1")
	f.AddVersion("v2")
	f.SetSave("name", "f1")
	d, _ = reg.AddGroup("dirs", "d2")
	f, _ = d.AddResource("files", "f2", "v1")
	f.AddVersion("v1.1")
	f.SetSave("name", "f2")

	gm, err = reg.Model.AddGroupModel("schemagroups", "schemagroup")
	xNoErr(t, err)
	_, err = gm.AddResourceModel("schemas", "schema", 0, true, true, true)
	xNoErr(t, err)
	sg, err := reg.AddGroup("schemagroups", "sg1")
	xNoErr(t, err)
	s, err := sg.AddResource("schemas", "s1", "v1.0")
	xNoErr(t, err)
	s.AddVersion("v2.0")

	reg.SetSave("labels.reg1", "1ger")
	f.SetSave("labels.file1", "1elif")

	// /dirs/d1/f1/v1     f1.name=f1
	//            /v2
	//      /d2/f2/v1     f2.name=f2
	//             v1.1
	// /s/sg1/schemas/s1/v1.0
	//                             /v2.0

	tests := []struct {
		Name string
		URL  string
		Exp  string
	}{
		{
			Name: "AND same obj/level - match",
			URL:  "?oneline&inline&filter=dirs.files.id=f1,dirs.files.name=f1",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}}},"schemagroups":{}}`,
		},
		{
			Name: "AND same obj/level - no match",
			URL:  "?oneline&inline&filter=dirs.files.id=f1,dirs.files.name=f2",
			Exp:  `Not found`,
		},
		{
			Name: "OR same obj/level - match",
			URL:  "?oneline&inline&filter=dirs.files.id=f1&filter=dirs.files.name=f1",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}}},"schemagroups":{}}`,
		},
		{
			Name: "multi result 2 levels down - match",
			URL:  "?oneline&inline&filter=dirs.files.versions.id=v1",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{}}}}}},"schemagroups":{}}`,
		},
		{
			Name: "path + multi result 2 levels down - match",
			URL:  "dirs?oneline&inline&filter=files.versions.id=v1",
			Exp:  `{"d1":{"files":{"f1":{"versions":{"v1":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{}}}}}}`,
		},
		{
			Name: "path + multi result 2 levels down - match",
			URL:  "dirs?oneline&inline&filter=files.versions.id=v1*",
			Exp:  `{"d1":{"files":{"f1":{"versions":{"v1":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}`,
		},
		{
			Name: "path + multi result 2 levels down - no match",
			URL:  "dirs?oneline&inline&filter=files.versions.id=xxx",
			Exp:  `{}`,
		},

		// Span group types
		{
			Name: "dirs and s - match both",
			URL:  "?oneline&inline&filter=dirs.id=d1&filter=schemagroups.id=sg1",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}}},"schemagroups":{"sg1":{"schemas":{"s1":{"versions":{"v1.0":{},"v2.0":{}}}}}}}`,
		},
		{
			Name: "dirs and s - match first",
			URL:  "?oneline&inline&filter=dirs.id=d1&filter=schemagroups.id=xxx",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}}},"schemagroups":{}}`,
		},
		{
			Name: "dirs and s - match second",
			URL:  "?oneline&inline&filter=dirs.id=xxx&filter=schemagroups.id=sg1",
			Exp:  `{"dirs":{},"schemagroups":{"sg1":{"schemas":{"s1":{"versions":{"v1.0":{},"v2.0":{}}}}}}}`,
		},
		{
			Name: "dirsOR and sOR - match first",
			URL:  "?oneline&inline&filter=dirs.files.id=f1,dirs.files.versions.id=v2&filter=schemagroups.schemas.versions.id=v1.0,schemagroups.schemas.versions.id=v2.0",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v2":{}}}}}},"schemagroups":{}}`,
		},
		{
			Name: "dirsOR and sOR - match second",
			URL:  "?oneline&inline&filter=dirs.files.id=f1,dirs.files.versions.id=xxx&filter=schemagroups.schemas.versions.id=v2.0,schemagroups.schemas.defaultversionid=v2.0",
			Exp:  `{"dirs":{},"schemagroups":{"sg1":{"schemas":{"s1":{"versions":{"v2.0":{}}}}}}}`,
		},
		{
			Name: "dirsOR and sOR - both match",
			URL:  "?oneline&inline&filter=dirs.files.id=f1,dirs.files.versions.id=v2&filter=schemagroups.schemas.versions.id=v2.0,schemagroups.schemas.defaultversionid=v2.0",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v2":{}}}}}},"schemagroups":{"sg1":{"schemas":{"s1":{"versions":{"v2.0":{}}}}}}}`,
		},
	}

	for _, test := range tests {
		t.Logf("Test name: %s", test.Name)
		xCheckGet(t, reg, test.URL, test.Exp)
	}
}

func TestWildcards(t *testing.T) {
	reg := NewRegistry("TestWildcards")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)
	_, err = gm.AddResourceModel("files", "file", 0, true, true, true)
	xNoErr(t, err)

	d, _ := reg.AddGroup("dirs", "d1")
	f, _ := d.AddResource("files", "f1", "v1")
	f.AddVersion("v2")
	f.SetSave("name", "f1")
	d, _ = reg.AddGroup("dirs", "d2")
	f, _ = d.AddResource("files", "f2", "v1")
	f.AddVersion("v1.1")
	f.SetSave("name", "f123")
	f, _ = d.AddResource("files", "f3", "v1")
	f.AddVersion("v1.1")
	f.SetSave("name", "g%d")
	f, _ = d.AddResource("files", "f4", "v1") // No name at all

	tests := []struct {
		Name string
		URL  string
		Exp  string
	}{
		{
			Name: "wildcard at start",
			URL:  "?oneline&inline&filter=dirs.files.name=*3",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "wildcard at end - 1",
			URL:  "?oneline&inline&filter=dirs.files.name=f*",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "wildcard at end - 2",
			URL:  "?oneline&inline&filter=dirs.files.name=f12*",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "wildcard at both ends - 1",
			URL:  "?oneline&inline&filter=dirs.files.name=*f12*",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "wildcard at both ends - 2",
			URL:  "?oneline&inline&filter=dirs.files.name=*12*",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "wildcard at both ends - 3",
			URL:  "?oneline&inline&filter=dirs.files.name=*3*",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "double wildcard - 1",
			URL:  "?oneline&inline&filter=dirs.files.name=**3",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "double wildcard - 2",
			URL:  "?oneline&inline&filter=dirs.files.name=**2**",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "double wildcard - 3",
			URL:  "?oneline&inline&filter=dirs.files.name=f**3",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "multi-wildcard - 1",
			URL:  "?oneline&inline&filter=dirs.files.name=f*1*2*3",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "multi-wildcard - 2",
			URL:  "?oneline&inline&filter=dirs.files.name=*f*1*2*3",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "multi-wildcard - 3",
			URL:  "?oneline&inline&filter=dirs.files.name=f*1*2*3*",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "escape - 1",
			URL:  "?oneline&inline&filter=dirs.files.name=g%25d",
			Exp:  `{"dirs":{"d2":{"files":{"f3":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "escape - 2",
			URL:  "?oneline&inline&filter=dirs.files.name=g*d",
			Exp:  `{"dirs":{"d2":{"files":{"f3":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "escape - 3",
			URL:  "?oneline&inline&filter=dirs.files.name=g\\*d",
			Exp:  `Not found`,
		},
		{
			Name: "all - 1",
			URL:  "?oneline&inline&filter=dirs.files.name", // name must be set
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}},"f3":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "all - 2",
			URL:  "?oneline&inline&filter=dirs.files.name=*", // name must be set
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}},"f3":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "all - 3",
			URL:  "?oneline&inline", // verify same as name=* + f4
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}},"f3":{"versions":{"v1":{},"v1.1":{}}},"f4":{"versions":{"v1":{}}}}}}}`,
		},
		{
			Name: "fail - 1",
			URL:  "?oneline&inline&filter=dirs.files.name=f*x",
			Exp:  `Not found`,
		},
		{
			Name: "fail - 2",
			URL:  "?oneline&inline&filter=dirs.files.name=*f",
			Exp:  `Not found`,
		},
		{
			Name: "fail - 3",
			URL:  "?oneline&inline&filter=dirs.files.name=z*",
			Exp:  `Not found`,
		},
		{
			Name: "fail - 4",
			URL:  "?oneline&inline&filter=dirs.files.name=*z*",
			Exp:  `Not found`,
		},
		{
			Name: "fail - 5",
			URL:  "?oneline&inline&filter=dirs.files.name=**z**",
			Exp:  `Not found`,
		},
		{
			Name: "fail - 6",
			URL:  "?oneline&inline&filter=dirs.files.description=*",
			Exp:  `Not found`,
		},
	}

	for _, test := range tests {
		t.Logf("Test name: %s", test.Name)
		xCheckGet(t, reg, test.URL, test.Exp)
	}
}
