package tests

import (
	"testing"

	// log "github.com/duglin/dlog"
	"github.com/duglin/xreg-github/registry"
)

func TestCreateGroup(t *testing.T) {
	reg, err := registry.NewRegistry("TestCreateGroup")
	defer reg.Delete()
	xCheck(t, reg != nil && err == nil, "can't create reg")

	gm, _ := reg.AddGroupModel("dirs", "dir", "")
	gm.AddResourceModel("files", "file", 0, true, true)

	d1, err := reg.AddGroup("dirs", "d1")
	xNoErr(t, err)
	xCheck(t, d1 != nil, "D1 is nil")

	dt, err := reg.AddGroup("dirs", "d1")
	xCheck(t, dt == nil && err != nil, "Dup should fail")

	f1, err := d1.AddResource("files", "f1", "v1")
	xNoErr(t, err)
	ft, err := d1.AddResource("files", "f1", "v1")
	xCheck(t, ft == nil && err != nil, "Dup files should have failed - 1")
	ft, err = d1.AddResource("files", "f1", "v2")
	xCheck(t, ft == nil && err != nil, "Dup files should have failed - 2")

	f1.AddVersion("v2")
	d2, _ := reg.AddGroup("dirs", "d2")
	f2, _ := d2.AddResource("files", "f2", "v1")
	f2.AddVersion("v1.1")

	// /dirs/d1/f1/v1
	//            /v2
	//      /d2/f2/v1
	//             v1.1

	g, err := reg.FindGroup("dirs", "d1")
	xNoErr(t, err)
	xJSONCheck(t, g, d1)

	g, err = reg.FindGroup("xxx", "d1")
	xCheck(t, err == nil && g == nil, "Finding xxx/d1 should have failed")

	g, err = reg.FindGroup("dirs", "xx")
	xCheck(t, err == nil && g == nil, "Finding dirs/xxx should have failed")

	r, err := d1.FindResource("files", "f1")
	xCheck(t, err == nil && r != nil, "Finding resource failed")
	xJSONCheck(t, r, f1)

	r2, err := d1.FindResource("files", "xxx")
	xCheck(t, err == nil && r2 == nil, "Finding files/xxx didn't work")

	r2, err = d1.FindResource("xxx", "f1")
	t.Logf("r2: %v  err: %v", r2, err)
	xCheck(t, err == nil && r2 == nil, "Finding xxx/f1 didn't work")

	err = d1.Delete()
	xNoErr(t, err)

	g, err = reg.FindGroup("dirs", "d1")
	xCheck(t, err == nil && g == nil, "Finding delete group failed")
}
