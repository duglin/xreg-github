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

	dir := reg.FindOrAddGroup("dirs", "d1")
	file := dir.AddResource("files", "f1", "v1")

	// /dirs/d1/f1/v1

	// Make sure setting it on the version is seen by res.Latest and res
	file.Set("name", "myName")
	ver := file.FindVersion("v1")
	val := ver.Get("name")
	if val != "myName" {
		t.Errorf("ver.Name is %q, should be 'myName'", val)
	}
}

func TestSetVersion(t *testing.T) {
	reg, _ := registry.NewRegistry("TestSetVersion")
	defer reg.Delete()

	gm, _ := reg.AddGroupModel("dirs", "dir", "")
	gm.AddResourceModel("files", "file", 0, true, true)

	dir := reg.FindOrAddGroup("dirs", "d1")
	file := dir.AddResource("files", "f1", "v1")
	ver := file.FindVersion("v1")

	// /dirs/d1/f1/v1

	// Make sure setting it on the version is seen by res.Latest and res
	ver.Set("name", "myName")
	file = dir.FindResource("file", "f1")
	val := file.GetLatest().Get("name")
	if val != "myName" {
		t.Errorf("resource.latest.Name is %q, should be 'myName'", val)
	}
	val = file.Get("name")
	if val != "myName" {
		t.Errorf("resource.Name is %q, should be 'myName'", val)
	}

	// Make sure we can also still see it from the version itself
	ver = file.FindVersion("v1")
	val = ver.Get("name")
	if val != "myName" {
		t.Errorf("version.Name is %q, should be 'myName'", val)
	}

}
