package tests

import (
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestNoModel(t *testing.T) {
	reg, err := registry.NewRegistry("TestNoModel")
	defer reg.Delete()
	xNoErr(t, err)
	xCheck(t, reg != nil, "reg created didn't work")

	xCheckGet(t, reg, "/model", "{}\n")
	xCheckGet(t, reg, "?model", `{
  "id": "TestNoModel",
  "self": "http:///",
  "model": {}
}
`)

	xCheckGet(t, reg, "/model/foo", "404: Not found")
}

func TestGroupModelCreate(t *testing.T) {
	reg, err := registry.NewRegistry("TestGroupModelCreate")
	defer reg.Delete()
	xNoErr(t, err)
	xCheck(t, reg != nil, "reg created didn't work")

	gm, err := reg.AddGroupModel("dirs", "dir", "schema-url")
	xNoErr(t, err)
	xCheck(t, gm != nil, "gm created didn't work")
	xCheckGet(t, reg, "/model", `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "schema": "schema-url"
    }
  }
}
`)

	// Now error checking
	gm, err = reg.AddGroupModel("dirs1", "", "schema-url") // missing value
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.AddGroupModel("", "", "schema-url") // missing value
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.AddGroupModel("", "", "") // missing value
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.AddGroupModel("", "dir1", "") // missing value
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.AddGroupModel("dirs", "dir", "schema-url") // dup
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.AddGroupModel("dirs1", "dir", "") // dup
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.AddGroupModel("dirs", "dir1", "") // dup
	xCheck(t, gm == nil && err != nil, "gm should have failed")
}

// test multiple groups, multiple resources

// DUG continue
func TestResourceModelCreate(t *testing.T) {
	reg, err := registry.NewRegistry("TestGroupModels")
	defer reg.Delete()
	xNoErr(t, err)
	xCheck(t, reg != nil, "reg created didn't work")

	gm, err := reg.AddGroupModel("dirs", "dir", "dirs-url")
	xNoErr(t, err)
	xCheck(t, gm != nil, "gm should have worked")

	rm, err := gm.AddResourceModel("files", "file", 0, true, true)
	xNoErr(t, err)
	xCheck(t, rm != nil, "rm should have worked")
}
