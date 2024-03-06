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
	"strings"
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestMain(m *testing.M) {
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

	reg, _ := registry.FindRegistry(name)
	if reg != nil {
		reg.Delete()
	}

	registry.DefaultReg, err = registry.NewRegistry(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating registry %q: %s", name, err)
		os.Exit(1)
	}
	return registry.DefaultReg
}

func PassDeleteReg(t *testing.T, reg *registry.Registry) {
	if !t.Failed() {
		if os.Getenv("NO_DELETE_REGISTRY") == "" {
			// We do this to make sure that we can support more than
			// one registry in the DB at a time
			reg.Delete()
		}
		registry.DefaultReg = nil
	}
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

func xCheckErr(t *testing.T, err error, errStr string) bool {
	t.Helper()
	if err == nil {
		if errStr == "" {
			return true
		}
		Fail(t, "Test was supposed to generate an error: %s", errStr)
		return false
	}
	if errStr == "" {
		Fail(t, "Test failed: %s", err)
		return false
	}
	if err.Error() != errStr {
		Fail(t, "Got: %s\nExp; %s", err.Error(), errStr)
		return false
	}
	return true
}

func xCheck(t *testing.T, b bool, errStr string) bool {
	t.Helper()
	if !b {
		t.Fatalf("%s", errStr)
	}
	return b
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

func xNoErr(t *testing.T, err error) bool {
	t.Helper()
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
		return false
	}
	return true
}

func xCheckGet(t *testing.T, reg *registry.Registry, url string, expected string) bool {
	t.Helper()
	res, err := http.Get("http://localhost:8181/" + url)
	if !xNoErr(t, err) {
		return false
	}

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

	return xCheckEqual(t, "URL: "+url+"\n", buf.String(), expected)
}

func xCheckEqual(t *testing.T, extra string, got string, exp string) bool {
	t.Helper()
	pos := 0
	for pos < len(got) && pos < len(exp) && got[pos] == exp[pos] {
		pos++
	}
	if pos == len(got) && pos == len(exp) {
		return true
	}

	if pos == len(got) {
		t.Errorf("%s"+
			"Expected:\n%s\nGot:\n%s\nGot ended early at(%d)[%02X]:\n%q",
			extra, exp, got, pos, exp[pos], got[pos:])
		return false
	}

	if pos == len(exp) {
		t.Errorf("%s"+
			"Expected:\n%s\nGot:\n%s\nExp ended early at(%d)[%02X]:\n%q",
			extra, exp, got, pos, got[pos], got[pos:])
		return false
	}

	expMax := pos + 90
	if expMax > len(exp) {
		expMax = len(exp)
	}
	t.Fatalf( /* Caller()+"\n%s"+ */
		"\nExpected:\n%s\nGot:\n%s\n"+
			"Diff at(%d)[%x/%x]:\n"+
			"Exp subset:\n%s\nGot:\n%s",
		/*extra, */ exp, got, pos, exp[pos], got[pos],
		exp[pos:expMax], got[pos:])
	return false
}

func xJSONCheck(t *testing.T, gotObj any, expObj any) bool {
	got := ToJSON(gotObj)
	exp := ToJSON(expObj)
	return xCheckEqual(t, "", got, exp)
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
