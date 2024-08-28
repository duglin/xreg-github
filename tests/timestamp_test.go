package tests

import (
	"fmt"
	"testing"

	"github.com/duglin/xreg-github/registry"
)

var TSREGEXP = `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[-+]\d{2}:\d{2})`
var TSMASK = TSREGEXP + `||YYYY-MM-DDTHH:MM:SSZ`

// Mask timestamps, but if (for the same input) the same TS is used, make sure
// the mask result is the same for just those two
func MaskTimestamps(input string) string {
	seenTS := map[string]string{}

	replaceFunc := func(input string) string {
		if val, ok := seenTS[input]; ok {
			return val
		}
		val := fmt.Sprintf("YYYY-MM-DDTHH:MM:%02dZ", len(seenTS)+1)
		seenTS[input] = val
		return val
	}

	re := savedREs[TSREGEXP]
	return re.ReplaceAllStringFunc(input, replaceFunc)
}

func TestTimestampRegistry(t *testing.T) {
	reg := NewRegistry("TestTimestampRegistry")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "reg shouldn't be nil")

	// Check basic GET first
	xCheckGet(t, reg, "/",
		`{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestTimestampRegistry",
  "self": "http://localhost:8181/",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`)

	// Should be the same values
	regCreate := reg.Get("createdat")
	regMod := reg.Get("modifiedat")
	xCheckEqual(t, "", regCreate, regMod)

	// Test to make sure modify timestamp changes, but created didn't
	xNoErr(t, reg.SetSave("description", "my docs"))
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestTimestampRegistry",
  "self": "http://localhost:8181/",
  "epoch": 1,
  "description": "my docs",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z"
}
`})

	reg.Refresh()

	xCheckEqual(t, "", reg.Get("createdat"), regCreate)
	xCheck(t, regMod != reg.Get("modifiedat"), "should be new time")

	// Mod should be higher than before
	xCheck(t, ToJSON(reg.Get("modifiedat")) > ToJSON(regMod),
		"Mod should be newer than before")

	regMod = reg.Get("modifiedat")

	xCheck(t, ToJSON(regMod) > ToJSON(regCreate),
		"Mod should be newer than create")

	// Now test with Groups and Resources
	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	_, err = gm.AddResourceModel("files", "file", 0, true, true, true)
	xNoErr(t, err)

	d, _ := reg.AddGroup("dirs", "d1")
	f, _ := d.AddResource("files", "f1", "v1")

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/?inline",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestTimestampRegistry",
  "self": "http://localhost:8181/",
  "epoch": 1,
  "description": "my docs",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirs": {
    "d1": {
      "id": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:03Z",
      "modifiedat": "2024-01-01T12:00:03Z",

      "files": {
        "f1": {
          "id": "f1",
          "self": "http://localhost:8181/dirs/d1/files/f1$meta",
          "epoch": 1,
          "createdat": "2024-01-01T12:00:03Z",
          "modifiedat": "2024-01-01T12:00:03Z",

          "defaultversionid": "v1",
          "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1$meta",

          "versions": {
            "v1": {
              "id": "v1",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$meta",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2024-01-01T12:00:03Z",
              "modifiedat": "2024-01-01T12:00:03Z"
            }
          },
          "versionscount": 1,
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
        }
      },
      "filescount": 1,
      "filesurl": "http://localhost:8181/dirs/d1/files"
    }
  },
  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
}
`})
	dCTime := d.Get("createdat")
	dMTime := d.Get("modifiedat")

	fCTime := f.Get("createdat")
	fMTime := f.Get("modifiedat")

	xCheckEqual(t, "", reg.Get("createdat"), regCreate)
	xCheckEqual(t, "", reg.Get("modifiedat"), regMod)

	xNoErr(t, f.SetSave("description", "myfile"))

	xCheckEqual(t, "", dCTime, d.Get("createdat"))
	xCheckEqual(t, "", dMTime, d.Get("modifiedat"))
	xCheckEqual(t, "", fCTime, f.Get("createdat"))
	xCheck(t, ToJSON(fMTime) < ToJSON(f.Get("modifiedat")),
		"Should not be the same")

	// Close out any lingering tx
	xNoErr(t, reg.Commit())

	/*
	   	reg = NewRegistry("TestTimestampRegistry2")
	   	defer PassDeleteReg(t, reg)
	   	xCheck(t, reg != nil, "reg shouldn't be nil")

	   	xCheckHTTP(t, reg, &HTTPTest{
	   		URL:    "/",
	   		Method: "GET",
	   		Code:   200,
	   		ResBody: `{
	     "specversion": "` + registry.SPECVERSION + `",
	     "id": "TestTimestampRegistry2",
	     "self": "http://localhost:8181/",
	     "epoch": 1,
	     "createdat": "2024-01-01T12:00:01Z",
	     "modifiedat": "2024-01-01T12:00:01Z"
	   }
	   `})
	*/

	// Test updating registry's times
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - set ts",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
			"createdat": "1970-01-02T03:04:05Z",
			"modifiedat": "2000-05-04T03:02:01Z"
		}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `--{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestTimestampRegistry",
  "self": "http://localhost:8181/",
  "epoch": 2,
  "createdat": "1970-01-02T03:04:05Z",
  "modifiedat": "2000-05-04T03:02:01Z",

  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
}
`,
	})
	reg.Refresh()
	// Shouldn't need these, but do it anyway
	xCheckEqual(t, "", reg.Get("createdat"), "1970-01-02T03:04:05Z")
	xCheckEqual(t, "", reg.Get("modifiedat"), "2000-05-04T03:02:01Z")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - set ts",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
			"createdat": null
		}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestTimestampRegistry",
  "self": "http://localhost:8181/",
  "epoch": 3,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",

  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
}
`,
	})

	// Test creating a group and setting it's times
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - set ts",
		URL:        "/dirs/d4",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
			"createdat": "1970-01-02T03:04:05Z",
			"modifiedat": "2000-05-04T03:02:01Z"
		}`,
		Code:       201,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "id": "d4",
  "self": "http://localhost:8181/dirs/d4",
  "epoch": 1,
  "createdat": "1970-01-02T03:04:05Z",
  "modifiedat": "2000-05-04T03:02:01Z",

  "filescount": 0,
  "filesurl": "http://localhost:8181/dirs/d4/files"
}
`,
	})

	g, err := reg.FindGroup("dirs", "d4", false)
	xNoErr(t, err)
	xCheckEqual(t, "", g.Get("createdat"), "1970-01-02T03:04:05Z")
	xCheckEqual(t, "", g.Get("modifiedat"), "2000-05-04T03:02:01Z")

	// Test creating a dir/file/version and setting the version's times
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - set ts",
		URL:        "/dirs/d5/files/f5/versions/v99$meta",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
			"createdat": "1970-01-02T03:04:05Z",
			"modifiedat": "2000-05-04T03:02:01Z"
		}`,
		Code:       201,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "id": "v99",
  "self": "http://localhost:8181/dirs/d5/files/f5/versions/v99$meta",
  "epoch": 1,
  "isdefault": true,
  "createdat": "1970-01-02T03:04:05Z",
  "modifiedat": "2000-05-04T03:02:01Z"
}
`,
	})

	g, err = reg.FindGroup("dirs", "d5", false)
	xNoErr(t, err)
	r, err := g.FindResource("files", "f5", false)
	xNoErr(t, err)
	v, err := r.FindVersion("v99", false)
	xNoErr(t, err)
	xCheckEqual(t, "", v.Get("createdat"), "1970-01-02T03:04:05Z")
	xCheckEqual(t, "", v.Get("modifiedat"), "2000-05-04T03:02:01Z")
}
