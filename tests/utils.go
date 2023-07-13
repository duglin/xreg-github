package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"runtime"
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func Caller() string {
	_, me, _, _ := runtime.Caller(0)

	for depth := 1; ; depth++ {
		_, file, line, ok := runtime.Caller(depth)
		if !ok {
			break
		}
		if file != me {
			return fmt.Sprintf("%s:%d", path.Base(file), line)
		}

	}
	return "unknownFile"
}

func xCheck(t *testing.T, b bool, errStr string) {
	if !b {
		t.Errorf("%s: %s", Caller(), errStr)
	}
}

func ToJSON(obj interface{}) string {
	buf, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error Marshaling: %s", err)
	}
	return string(buf)
}

func xNoErr(t *testing.T, err error) bool {
	if err != nil {
		t.Errorf("%s: Unexpected error: %s", Caller(), err)
		return true
	}
	return false
}

func xCheckGet(t *testing.T, reg *registry.Registry, url string, expected string) {
	buf := &bytes.Buffer{}
	out := io.Writer(buf)

	req, err := http.NewRequest("GET", url, nil)
	if xNoErr(t, err) {
		return
	}
	info, err := reg.ParseRequest(req)
	if err != nil {
		xCheckEqual(t, "URL: "+url+"\n", err.Error(), expected)
		return
	}
	Check(info.ErrCode == 0, Caller()+":info.ec != 0")
	if err = reg.NewGet(out, info); err != nil {
		errMsg := err.Error()
		if req.URL.Query().Has("oneline") {
			errMsg = string(OneLine([]byte(errMsg)))
		}
		xCheckEqual(t, "URL: "+url+"\n", errMsg, expected)
		return
	}
	if xNoErr(t, err) {
		return
	}

	if req.URL.Query().Has("noprops") {
		buf = bytes.NewBuffer(RemoveProps(buf.Bytes()))
		// expected = string(RemoveProps([]byte(expected)))
	}
	if req.URL.Query().Has("oneline") {
		buf = bytes.NewBuffer(OneLine(buf.Bytes()))
		// expected = string(OneLine([]byte(expected)))
	}

	xCheckEqual(t, "URL: "+url+"\n", buf.String(), expected)
}

func xCheckEqual(t *testing.T, extra string, got string, exp string) {
	if got != exp {
		pos := 0
		for pos < len(got) && pos < len(exp) && got[pos] == exp[pos] {
			pos++
		}

		if pos == len(got) {
			t.Errorf(Caller()+"\n%s"+
				"Expected:\n%s\nGot:\n%s\nGot ended early at(%d)[%02X]:\n%q",
				extra, exp, got, pos, exp[pos], got[pos:])
			return
		}

		if pos == len(exp) {
			t.Errorf(Caller()+"\n%s"+
				"Expected:\n%s\nGot:\n%s\nExp ended early at(%d)[%02X]:\n%q",
				extra, exp, got, pos, got[pos], got[pos:])
			return
		}

		t.Errorf(Caller()+"\n%s"+
			"Expected:\n%s\nGot:\n%s\nDiff at(%d)[%x/%x]:\n%q",
			extra, exp, got, pos, exp[pos], got[pos], got[pos:])
	}
}

func xJSONCheck(t *testing.T, gotObj any, expObj any) {
	got := ToJSON(gotObj)
	exp := ToJSON(expObj)
	xCheckEqual(t, "", got, exp)
}
