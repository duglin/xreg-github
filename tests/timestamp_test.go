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
		val := fmt.Sprintf("YYYY-MM-DDTHH:MM:%dZ", len(seenTS)+1)
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
  "epoch": 1,
  "self": "http://localhost:8181/",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`)

	xCheckGet(t, reg, "/",
		`{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestTimestampRegistry",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`)

	// Note that turning on Tracking will set the timetamps immediately
	// on any newly touched entity

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/",
		Method: "GET",
		Code:   200,
		// BodyMasks: []string{TSMASK},
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestTimestampRegistry",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`})

	// Should be the same values
	regCreate := reg.Get("createdat")
	regMod := reg.Get("modifiedat")

	xNoErr(t, reg.SetSave("description", "my docs"))
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/",
		Method: "GET",
		Code:   200,
		// BodyMasks: []string{TSMASK},
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestTimestampRegistry",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "description": "my docs",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z"
}
`})

	reg.Refresh()

	xCheckEqual(t, "", reg.Get("createdat"), regCreate)
	xCheck(t, regMod != reg.Get("modifiedat"), "should be new time")

	regCreate = reg.Get("createdat")
	regMod = reg.Get("modifiedat")

	// Now test with Groups and Resources
	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	_, err = gm.AddResourceModel("files", "file", 0, true, true, true)
	xNoErr(t, err)

	d, _ := reg.AddGroup("dirs", "d1")
	f, _ := d.AddResource("files", "f1", "v1")

	xCheckHTTP(t, reg, &HTTPTest{
		URL:       "/?inline",
		Method:    "GET",
		Code:      200,
		BodyMasks: []string{TSMASK},
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestTimestampRegistry",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "description": "my docs",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "dirs": {
    "d1": {
      "id": "d1",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1",
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "files": {
        "f1": {
          "id": "f1",
          "epoch": 1,
          "self": "http://localhost:8181/dirs/d1/files/f1?meta",
          "defaultversionid": "v1",
          "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1?meta",
          "createdat": "YYYY-MM-DDTHH:MM:SSZ",
          "modifiedat": "YYYY-MM-DDTHH:MM:SSZ",

          "versions": {
            "v1": {
              "id": "v1",
              "epoch": 1,
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1?meta",
              "isdefault": true,
              "createdat": "YYYY-MM-DDTHH:MM:SSZ",
              "modifiedat": "YYYY-MM-DDTHH:MM:SSZ"
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
	xCheck(t, fMTime != f.Get("modifiedat"), "Should not be the same")

	// Close out any lingering tx
	xNoErr(t, reg.Commit())

	reg = NewRegistry("TestTimestampRegistry2")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "reg shouldn't be nil")

	xCheckHTTP(t, reg, &HTTPTest{
		URL:       "/",
		Method:    "GET",
		Code:      200,
		BodyMasks: []string{TSMASK},
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestTimestampRegistry2",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "createdat": "YYYY-MM-DDTHH:MM:SSZ",
  "modifiedat": "YYYY-MM-DDTHH:MM:SSZ"
}
`})
}
