package tests

import (
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestCreateRegistry(t *testing.T) {
	reg := NewRegistry("TestCreateRegistry")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "reg shouldn't be nil")

	// Check basic GET first
	xCheckGet(t, reg, "/",
		`{
  "id": "TestCreateRegistry",
  "epoch": 1,
  "self": "http://localhost:8181/"
}
`)
	xCheckGet(t, reg, "/xxx", "Unknown Group type: \"xxx\"\n")
	xCheckGet(t, reg, "xxx", "Unknown Group type: \"xxx\"\n")
	xCheckGet(t, reg, "/xxx/yyy", "Unknown Group type: \"xxx\"\n")
	xCheckGet(t, reg, "xxx/yyy", "Unknown Group type: \"xxx\"\n")

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
	defer PassDeleteReg(t, reg3)
	xNoErr(t, err)
	xCheck(t, reg3 != nil, "reg3 shouldn't be nil")
	xCheck(t, reg3 != reg, "reg3 should be different from reg")

	xCheckGet(t, reg, "", `{
  "id": "TestCreateRegistry",
  "epoch": 1,
  "self": "http://localhost:8181/"
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
	defer PassDeleteReg(t, reg)
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
	defer PassDeleteReg(t, reg)
	xNoErr(t, err)

	reg2, err := registry.FindRegistry(reg.UID)
	xNoErr(t, err)
	xJSONCheck(t, reg2, reg)
}

func TestRegistryProps(t *testing.T) {
	reg := NewRegistry("TestRegistryProps")
	defer PassDeleteReg(t, reg)

	reg.Set("specVersion", "x.y")
	reg.Set("name", "nameIt")
	reg.Set("description", "a very cool reg")
	reg.Set("documentation", "https://docs.com")
	reg.Set("labels/stage", "dev")

	xCheckGet(t, reg, "", `{
  "specVersion": "x.y",
  "id": "TestRegistryProps",
  "name": "nameIt",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "description": "a very cool reg",
  "documentation": "https://docs.com",
  "labels": {
    "stage": "dev"
  }
}
`)
}
