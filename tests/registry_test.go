package tests

import (
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestCreateRegistry(t *testing.T) {
	reg, err := registry.NewRegistry("TestCreateRegistry")
	defer reg.Delete()
	xNoErr(t, err)
	xCheck(t, reg != nil, "reg shouldn't be nil")

	// Check basic GET first
	xCheckGet(t, reg, "/",
		`{
  "id": "TestCreateRegistry",
  "self": "http:///"
}
`)
	xCheckGet(t, reg, "/xxx", "Unknown Group type: \"xxx\"")
	xCheckGet(t, reg, "xxx", "Unknown Group type: \"xxx\"")
	xCheckGet(t, reg, "/xxx/yyy", "Unknown Group type: \"xxx\"")
	xCheckGet(t, reg, "xxx/yyy", "Unknown Group type: \"xxx\"")

	// make sure dups generate an error
	reg2, err := registry.NewRegistry("TestCreateRegistry")
	if err == nil || reg2 != nil {
		t.Errorf("Creating same named registry worked!")
	}

	// make sure it was really created
	reg3, err := registry.FindRegistry("TestCreateRegistry")
	xCheck(t, err == nil && reg3 != nil,
		"Finding TestCreateRegistry should have worked")

	reg3, err = registry.NewRegistry("")
	defer reg3.Delete()
	xNoErr(t, err)
	xCheck(t, reg3 != nil, "reg3 shouldn't be nil")
	xCheck(t, reg3 != reg, "reg3 should be different from reg")

	xCheckGet(t, reg, "", `{
  "id": "TestCreateRegistry",
  "self": "http:///"
}
`)
}

func TestDeleteRefistry(t *testing.T) {
	reg, err := registry.NewRegistry("TestDeleteRegistry")
	xNoErr(t, err)
	xCheck(t, reg != nil, "reg shouldn't be nil")

	err = reg.Delete()
	xNoErr(t, err)

	reg, err = registry.FindRegistry("TestDeleteRegistry")
	xCheck(t, reg == nil && err == nil,
		"Finding TestCreateRegistry found one but shouldn't")
}

func TestRefreshRegistry(t *testing.T) {
	reg, err := registry.NewRegistry("TestRefreshRegistry")
	defer reg.Delete()
	xNoErr(t, err)

	reg.Props["xxx"] = "yyy"

	err = reg.Refresh()
	xNoErr(t, err)

	xCheck(t, reg.Props["xxx"] == nil, "xxx should not be there")
}

func TestFindRegistry(t *testing.T) {
	reg, err := registry.FindRegistry("TestFindRegistry")
	xCheck(t, reg == nil && err == nil,
		"Shouldn't have found TestFindRegistry")

	reg, err = registry.NewRegistry("TestFindRegistry")
	defer reg.Delete()
	xNoErr(t, err)

	reg2, err := registry.FindRegistry(reg.UID)
	xNoErr(t, err)
	xJSONCheck(t, reg2, reg)
}

func TestRegistryProps(t *testing.T) {
	reg, _ := registry.NewRegistry("TestCreateRegistry")
	defer reg.Delete()

	reg.Set("specVersion", "x.y")
	reg.Set("name", "nameIt")
	reg.Set("description", "a very cool reg")
	reg.Set("docs", "https://docs.com")
	reg.Set("tags.stage", "dev")

	xCheckGet(t, reg, "", `{
  "specVersion": "x.y",
  "id": "TestCreateRegistry",
  "name": "nameIt",
  "self": "http:///",
  "description": "a very cool reg",
  "docs": "https://docs.com",
  "tags": {
    "stage": "dev"
  }
}
`)
}
