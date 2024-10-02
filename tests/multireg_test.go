package tests

import (
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestMultiReg(t *testing.T) {
	reg := NewRegistry("TestMultiReg")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)
	_, err = gm.AddResourceModel("files", "file", 0, true, true, true)
	xNoErr(t, err)
	reg.Commit()

	reg2, err := registry.NewRegistry(nil, "reg2")
	defer PassDeleteReg(t, reg2)
	xNoErr(t, err)
	gm, err = reg2.Model.AddGroupModel("reg2_dirs", "reg2_dir")
	xNoErr(t, err)
	_, err = gm.AddResourceModel("reg2_files", "reg2_file", 0, true, true,
		true)
	xNoErr(t, err)
	reg2.Commit()

	// reg
	xHTTP(t, reg, "GET", "/", "", 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestMultiReg",
  "self": "http://localhost:8181/",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "dirscount": 0,
  "dirsurl": "http://localhost:8181/dirs"
}
`)

	// reg2
	xHTTP(t, reg2, "GET", "/reg-reg2", "", 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "reg2",
  "self": "http://localhost:8181/reg-reg2/",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "reg2_dirscount": 0,
  "reg2_dirsurl": "http://localhost:8181/reg-reg2/reg2_dirs"
}
`)

	xHTTP(t, reg2, "GET", "/reg-reg2/reg2_dirs", "", 200, "{}\n")

	xHTTP(t, reg2, "PUT", "/reg-reg2/reg2_dirs/d2", "", 201, `{
  "reg2_dirid": "d2",
  "self": "http://localhost:8181/reg-reg2/reg2_dirs/d2",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "reg2_filescount": 0,
  "reg2_filesurl": "http://localhost:8181/reg-reg2/reg2_dirs/d2/reg2_files"
}
`)

	xHTTP(t, reg2, "PUT", "/reg-reg2/reg2_dirs/d2/reg2_files/f2$meta", "", 201, `{
  "reg2_fileid": "f2",
  "self": "http://localhost:8181/reg-reg2/reg2_dirs/d2/reg2_files/f2$meta",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/reg-reg2/reg2_dirs/d2/reg2_files/f2/versions/1$meta",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/reg-reg2/reg2_dirs/d2/reg2_files/f2/versions"
}
`)

}
