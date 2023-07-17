package tests

import (
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestSetResource(t *testing.T) {
	reg, _ := registry.NewRegistry("TestSetResource")
	defer reg.Delete()

	gm, _ := reg.AddGroupModel("dirs", "dir", "")
	gm.AddResourceModel("files", "file", 0, true, true)

	dir, _ := reg.AddGroup("dirs", "d1")
	file, _ := dir.AddResource("files", "f1", "v1")

	// /dirs/d1/f1/v1

	// Make sure setting it on the version is seen by res.Latest and res
	file.Set("name", "myName")
	ver, _ := file.FindVersion("v1")
	val := ver.Get("name")
	if val != "myName" {
		t.Errorf("ver.Name is %q, should be 'myName'", val)
	}

	name := file.Get("name").(string)
	xCheckEqual(t, "", name, "myName")

	// Verify that nil and "" are treated differently
	ver.Set("name", nil)
	ver2, _ := file.FindVersion(ver.ID)
	xJSONCheck(t, ver2, ver)
	val = ver.Get("name")
	xCheck(t, val == nil, "Setting to nil should return nil")

	ver.Set("name", "")
	ver2, _ = file.FindVersion(ver.ID)
	xJSONCheck(t, ver2, ver)
	val = ver.Get("name")
	xCheck(t, val == "", "Setting to '' should return ''")
}

func TestSetVersion(t *testing.T) {
	reg, _ := registry.NewRegistry("TestSetVersion")
	defer reg.Delete()

	gm, _ := reg.AddGroupModel("dirs", "dir", "")
	gm.AddResourceModel("files", "file", 0, true, true)

	dir, _ := reg.AddGroup("dirs", "d1")
	file, _ := dir.AddResource("files", "f1", "v1")
	ver, _ := file.FindVersion("v1")

	// /dirs/d1/f1/v1

	// Make sure setting it on the version is seen by res.Latest and res
	ver.Set("name", "myName")
	file, _ = dir.FindResource("files", "f1")
	l, err := file.GetLatest()
	xNoErr(t, err)
	xCheck(t, l != nil, "latest is nil")
	val := l.Get("name")
	if val != "myName" {
		t.Errorf("resource.latest.Name is %q, should be 'myName'", val)
	}
	val = file.Get("name")
	if val != "myName" {
		t.Errorf("resource.Name is %q, should be 'myName'", val)
	}

	// Make sure we can also still see it from the version itself
	ver, _ = file.FindVersion("v1")
	val = ver.Get("name")
	if val != "myName" {
		t.Errorf("version.Name is %q, should be 'myName'", val)
	}

}
