package tests

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/xregistry/server/registry"
)

var RepoBase = "https://raw.githubusercontent.com/xregistry/spec/main"

func TestXRBasic(t *testing.T) {
	cmd := exec.Command("../xr")
	out, err := cmd.CombinedOutput()
	xNoErr(t, err)
	lines, _, _ := strings.Cut(string(out), ":")

	// Just look for the first 3 lines
	xCheckEqual(t, "", lines, "xRegistry CLI\n\nUsage")

	// Make sure we can validate the various spec owned model files
	files := []string{
		"sample-model.json",
		"/endpoint/model.json",
		"/message/model.json",
		"/schema/model.json",
	}
	paths := os.Getenv("XR_MODEL_PATH")
	os.Setenv("XR_MODEL_PATH", paths+":.:"+RepoBase)

	for _, file := range files {
		fn := file
		if !strings.HasPrefix(fn, "http") {
			if fn, err = registry.FindModelFile(file); err == nil {
				break
			}
			if fn == "" {
				t.Errorf("Can't find %q in %q", file, paths)
				t.FailNow()
			}
		}

		cmd = exec.Command("../xr", "model", "verify", fn)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("File: %s\nOut: %s\nErr: %s", file, string(out), err)
		}
		xCheckEqual(t, "", string(out), "")
	}
}
