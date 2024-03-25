package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	gourl "net/url"
	"os"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	log "github.com/duglin/dlog"
	"github.com/duglin/xreg-github/registry"
)

func TestMain(m *testing.M) {
	if tmp := os.Getenv("VERBOSE"); tmp != "" {
		if tmpInt, err := strconv.Atoi(tmp); err == nil {
			log.SetVerbose(tmpInt)
		}
	}

	// call flag.Parse() here if TestMain uses flags
	registry.DeleteDB("testreg")
	registry.CreateDB("testreg")
	registry.OpenDB("testreg")

	// DBName := "registry"
	// if !registry.DBExists(DBName) {
	// registry.CreateDB(DBName)
	// }
	// registry.OpenDB(DBName)

	// Start HTTP server

	server := registry.NewServer(8181).Start()

	// Run the tests
	rc := m.Run()

	// Shutdown HTTP server
	server.Close()

	if rc == 0 {
		// registry.DeleteDB("testreg")
	}
	os.Exit(rc)
}

func NewRegistry(name string) *registry.Registry {
	var err error

	reg, _ := registry.FindRegistry(nil, name)
	if reg != nil {
		reg.Delete()
		reg.Commit()
	}

	reg, err = registry.NewRegistry(nil, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating registry %q: %s", name, err)
		os.Exit(1)
	}
	reg.Commit()

	registry.DefaultRegDbSID = reg.DbSID

	/*
		// Now find it again and start a new Tx
		reg, err = registry.FindRegistry(nil, name)
		if err != nil {
			panic(err.Error())
		}
		if reg == nil {
			panic("nil")
		}
	*/

	return reg
}

func PassDeleteReg(t *testing.T, reg *registry.Registry) {
	if !t.Failed() {
		if os.Getenv("NO_DELETE_REGISTRY") == "" {
			// We do this to make sure that we can support more than
			// one registry in the DB at a time
			reg.Delete()
		}
		registry.DefaultRegDbSID = ""
	}
	reg.Commit() // should this be Rollback() ?
}

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

func Fail(t *testing.T, str string, args ...any) {
	t.Helper()
	text := strings.TrimSpace(fmt.Sprintf(str, args...))
	t.Fatalf("%s\n\n", text)
}

func xCheckErr(t *testing.T, err error, errStr string) {
	t.Helper()
	if err == nil {
		if errStr == "" {
			return
		}
		t.Fatalf("\nGot:<no err>\nExp: %s", errStr)
	}

	if errStr == "" {
		t.Fatalf("Test failed: %s", err)
	}

	if err.Error() != errStr {
		t.Fatalf("\nGot: %s\nExp: %s", err.Error(), errStr)
	}
}

func xCheck(t *testing.T, b bool, errStr string, args ...any) {
	t.Helper()
	if !b {
		t.Fatalf(errStr, args...)
	}
}

func ToJSON(obj interface{}) string {
	if obj != nil && reflect.TypeOf(obj).String() == "*errors.errorString" {
		obj = obj.(error).Error()
	}
	buf, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error Marshaling: %s", err)
	}
	return string(buf)
}

func xNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
}

func xCheckGet(t *testing.T, reg *registry.Registry, url string, expected string) {
	t.Helper()
	xNoErr(t, reg.Commit())

	res, err := http.Get("http://localhost:8181/" + url)
	xNoErr(t, err)

	body, err := io.ReadAll(res.Body)
	buf := bytes.NewBuffer(body)
	daURL, _ := gourl.Parse(url)

	if daURL.Query().Has("noprops") {
		buf = bytes.NewBuffer(RemoveProps(buf.Bytes()))
		// expected = string(RemoveProps([]byte(expected)))
	}
	if daURL.Query().Has("oneline") {
		buf = bytes.NewBuffer(OneLine(buf.Bytes()))
		expected = string(OneLine([]byte(expected)))
	}

	xCheckEqual(t, "URL: "+url+"\n", buf.String(), expected)
}

func xCheckEqual(t *testing.T, extra string, got string, exp string) {
	t.Helper()
	pos := 0
	for pos < len(got) && pos < len(exp) && got[pos] == exp[pos] {
		pos++
	}
	if pos == len(got) && pos == len(exp) {
		return
	}

	if pos == len(got) {
		t.Fatalf("%s"+
			"Expected:\n%s\nGot:\n%s\nGot ended early at(%d)[%02X]:\n%q",
			extra, exp, got, pos, exp[pos], got[pos:])
	}

	if pos == len(exp) {
		t.Fatalf("%s"+
			"Expected:\n%s\nGot:\n%s\nExp ended early at(%d)[%02X]:\n%q",
			extra, exp, got, pos, got[pos], got[pos:])
	}

	expMax := pos + 90
	if expMax > len(exp) {
		expMax = len(exp)
	}
	t.Fatalf( /* Caller()+"\n*/ "%s"+
		"\nExpected:\n%s\nGot:\n%s\n"+
		"Diff at(%d)[%x/%x]:\n"+
		"Exp subset:\n%s\nGot:\n%s",
		extra, exp, got, pos, exp[pos], got[pos],
		exp[pos:expMax], got[pos:])
}

func xJSONCheck(t *testing.T, gotObj any, expObj any) {
	got := ToJSON(gotObj)
	exp := ToJSON(expObj)
	xCheckEqual(t, "", got, exp)
}

func OneLine(buf []byte) []byte {
	buf = RemoveProps(buf)

	re := regexp.MustCompile(`[\r\n]*`)
	buf = re.ReplaceAll(buf, []byte(""))
	re = regexp.MustCompile(`([^a-zA-Z])\s+([^a-zA-Z])`)
	buf = re.ReplaceAll(buf, []byte(`$1$2`))
	re = regexp.MustCompile(`([^a-zA-Z])\s+([^a-zA-Z])`)
	buf = re.ReplaceAll(buf, []byte(`$1$2`))

	return buf
}

func RemoveProps(buf []byte) []byte {
	re := regexp.MustCompile(`\n[^{}]*\n`)
	buf = re.ReplaceAll(buf, []byte("\n"))

	re = regexp.MustCompile(`\s"labels": {\s*},*`)
	buf = re.ReplaceAll(buf, []byte(""))

	re = regexp.MustCompile(`\n *\n`)
	buf = re.ReplaceAll(buf, []byte("\n"))

	re = regexp.MustCompile(`\n *}\n`)
	buf = re.ReplaceAll(buf, []byte("}\n"))

	re = regexp.MustCompile(`}[\s,]+}`)
	buf = re.ReplaceAll(buf, []byte("}}"))
	buf = re.ReplaceAll(buf, []byte("}}"))

	return buf
}

func HTMLify(r *http.Request, buf []byte) []byte {
	str := fmt.Sprintf(`"(https?://%s[^"\n]*?)"`, r.Host)
	re := regexp.MustCompile(str)
	repl := fmt.Sprintf(`"<a href="$1?%s">$1?%s</a>"`,
		r.URL.RawQuery, r.URL.RawQuery)

	return re.ReplaceAll(buf, []byte(repl))
}

func NotNilString(val any) string {
	b := (val).([]byte)
	return string(b)
}

func NewPP() *registry.PropPath {
	return registry.NewPP()
}

func ShowStack() {
	fmt.Printf("-----\n")
	for i := 1; i < 20; i++ {
		pc, file, line, _ := runtime.Caller(i)
		fmt.Printf("Caller: %s:%d\n", path.Base(runtime.FuncForPC(pc).Name()), line)
		if strings.Contains(file, "main") || strings.Contains(file, "testing") {
			break
		}
	}
}
