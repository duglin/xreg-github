package tests

import (
	"os/exec"
	"strings"
	"testing"
)

var RepoBase = "https://raw.githubusercontent.com/xregistry/spec/main/"

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
		RepoBase + "endpoint/model.json",
		RepoBase + "message/model.json",
		RepoBase + "schema/model.json",
	}

	for _, file := range files {
		cmd = exec.Command("../xr", "model", "verify", file)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("File: %s\nOut: %s\nErr: %s", file, string(out), err)
		}
		xCheckEqual(t, "", string(out), "")
	}
}
