package tests

import (
	"testing"

	"github.com/duglin/xreg-github/registry"
)

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
  "self": "http://localhost:8181/"
}
`)

	xNoErr(t, reg.TrackTimestamps(false))
	xCheckGet(t, reg, "/",
		`{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestTimestampRegistry",
  "epoch": 1,
  "self": "http://localhost:8181/"
}
`)

	// Note that turning on Tracking will set the timetamps immediately
	// on any newly touched entity
	xNoErr(t, reg.TrackTimestamps(true))

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/",
		Method: "GET",
		Code:   200,
		BodyMasks: []string{
			`[0-4]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]+)?Z|YYYY-MM-DDTHH:MM:SSZ`,
		},
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestTimestampRegistry",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "modifiedon": "2024-01-01T12:00:00Z"
}
`})

	regCreate := reg.Get("createdon")
	regMod := reg.Get("modifiedon")

	xCheck(t, regCreate == nil, "Should be nil")
	xCheck(t, regMod != nil, "Should not be nil")

	xNoErr(t, reg.SetSave("description", "my docs"))
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/",
		Method: "GET",
		Code:   200,
		BodyMasks: []string{
			`[0-4]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]+)?Z|YYYY-MM-DDTHH:MM:SSZ`,
		},
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestTimestampRegistry",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "description": "my docs",
  "modifiedon": "2024-01-01T12:00:00Z"
}
`})

	reg.Refresh()

	xCheckEqual(t, "", reg.Get("createdon"), regCreate)
	xCheck(t, regMod != reg.Get("modifiedon"), "should be new time")

	regCreate = reg.Get("createdon")
	regMod = reg.Get("modifiedon")

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
		BodyMasks: []string{
			`[0-4]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]+)?Z|YYYY-MM-DDTHH:MM:SSZ`,
		},
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestTimestampRegistry",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "description": "my docs",
  "modifiedon": "2024-01-01T12:00:00Z",

  "dirs": {
    "d1": {
      "id": "d1",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1",
      "createdon": "YYYY-MM-DDTHH:MM:SSZ",
      "modifiedon": "YYYY-MM-DDTHH:MM:SSZ",

      "files": {
        "f1": {
          "id": "f1",
          "epoch": 1,
          "self": "http://localhost:8181/dirs/d1/files/f1?meta",
          "latestversionid": "v1",
          "latestversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1?meta",
          "createdon": "YYYY-MM-DDTHH:MM:SSZ",
          "modifiedon": "YYYY-MM-DDTHH:MM:SSZ",

          "versions": {
            "v1": {
              "id": "v1",
              "epoch": 1,
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1?meta",
              "latest": true,
              "createdon": "YYYY-MM-DDTHH:MM:SSZ",
              "modifiedon": "YYYY-MM-DDTHH:MM:SSZ"
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
	dCTime := d.Get("createdon")
	dMTime := d.Get("modifiedon")

	fCTime := f.Get("createdon")
	fMTime := f.Get("modifiedon")

	xCheckEqual(t, "", reg.Get("createdon"), regCreate)
	xCheckEqual(t, "", reg.Get("modifiedon"), regMod)

	xNoErr(t, f.SetSave("description", "myfile"))
	xCheckEqual(t, "", dCTime, d.Get("createdon"))
	xCheckEqual(t, "", dMTime, d.Get("modifiedon"))
	xCheckEqual(t, "", fCTime, f.Get("createdon"))
	xCheck(t, fMTime != f.Get("modifiedon"), "Should not be the same")

	// Close out any lingering tx
	xNoErr(t, reg.Commit())

	reg = NewRegistry("TestTimestampRegistry2",
		registry.RegOpt_TrackTimestamps)
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "reg shouldn't be nil")

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/",
		Method: "GET",
		Code:   200,
		BodyMasks: []string{
			`[0-4]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]+)?Z|YYYY-MM-DDTHH:MM:SSZ`,
		},
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestTimestampRegistry2",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "createdon": "YYYY-MM-DDTHH:MM:SSZ",
  "modifiedon": "YYYY-MM-DDTHH:MM:SSZ"
}
`})
}
