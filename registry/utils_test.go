package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"testing"
)

// Test ResolvePath
func TestResolvePath(t *testing.T) {
	type ResolvePathTest struct {
		Base   string
		Next   string
		Result string
	}

	tests := []ResolvePathTest{
		{"", "", ""},

		// FILE base
		{"", "file.txt", "file.txt"},
		{"dir1/dir2/file1", "file.txt", "dir1/dir2/file.txt"},
		{"dir1/dir2/file1", "/file.txt", "/file.txt"},
		{"dir1/dir2/file1", "d:/file.txt", "d:/file.txt"},
		{"dir1/dir2/file1", "http://foo.com", "http://foo.com"},
		{"d1/d2/f1", "https://foo.com", "https://foo.com"},
		{"d1/d2/f1", "#abc", "d1/d2/f1#abc"},
		{"d1/d2/f1", "file2#abc", "d1/d2/file2#abc"},
		{"dir1/dir2/file1", "./file2#abc", "dir1/dir2/file2#abc"},
		{"dir1/dir2/file1", "/file2", "/file2"},
		{"dir1/dir2/file1", "/file2#abc", "/file2#abc"},
		{"dir1/dir2/file1/", "file2#abc", "dir1/dir2/file1/file2#abc"},
		{"/", "file2#abc", "/file2#abc"},
		{"", "file2#abc", "file2#abc"},
		{"#foo", "file2#abc", "file2#abc"},
		{"f1#foo", "file2#abc", "file2#abc"},
		{"d1/f1#foo", "file2#abc", "d1/file2#abc"},
		{"d1/#foo", "file2#abc", "d1/file2#abc"},
		{"d1/#foo", "/file2#abc", "/file2#abc"},
		{"d1/#foo", "./file2#abc", "d1/file2#abc"},
		{"/d1/#foo", "/file2#abc", "/file2#abc"},
		{"./d1/#foo", "/file2#abc", "/file2#abc"},
		{"./d1/#foo", "./file2#abc", "d1/file2#abc"},
		{"./d1#foo", "./file2#abc", "file2#abc"},
		{"/d1/d2/../f3", "./file2", "/d1/file2"},
		{"/d1/../f3", "./file2", "/file2"},
		{"d1/../f3", "./file2", "file2"},
		{"d1/d2/../d3/../../f3", "file2", "file2"},
		{"d1/d2/d3/../../f3", "../file2", "file2"},
		{"d1/d2/d3/././f3", "file2", "d1/d2/d3/file2"},
		{"d1/d2/d3/.././f3", "file2", "d1/d2/file2"},
		{"../d1/d2/d3/.././f3", "file2", "../d1/d2/file2"},
		{"../d1//d2///d3////.././f3", "file2", "../d1/d2/file2"},
		{"d1/d2/..", "file2", "d1/file2"},
		{"d1/d2/.", "file2", "d1/d2/file2"},
		{"d1/d2/...", "file2", "d1/d2/file2"},

		// HTTP base
		{"http://s1.com/dir1/file",
			"https://foo.com/dir2/file2",
			"https://foo.com/dir2/file2"},
		{"http://s1.com/dir1/file",
			"file2",
			"http://s1.com/dir1/file2"},
		{"http://s1.com/dir1/file",
			"./file2",
			"http://s1.com/dir1/file2"},
		{"http://s1.com/dir1/",
			"./file2",
			"http://s1.com/dir1/file2"},
		{"http://s1.com/dir1/file",
			"",
			"http://s1.com/dir1/file"},
		{"http://s1.com/dir1/file",
			"/file2",
			"http://s1.com/file2"},

		{"http://s1.com/dir1/file",
			"#abc",
			"http://s1.com/dir1/file#abc"},
		{"http://s1.com/dir1/",
			"#abc",
			"http://s1.com/dir1/#abc"},
		{"http://s1.com",
			"file#abc",
			"http://s1.com/file#abc"},
		{"http://s1.com#def",
			"file#abc",
			"http://s1.com/file#abc"},
		{"http://s1.com/d1#def",
			"file#abc",
			"http://s1.com/file#abc"},
		{"http://s1.com/d1/#def",
			"file#abc",
			"http://s1.com/d1/file#abc"},
		{"http://s1.com/d1/d2/..",
			"file#abc",
			"http://s1.com/d1/file#abc"},
		{"http://s1.com/d1/d2/.",
			"file#abc",
			"http://s1.com/d1/d2/file#abc"},
		{"http://s1.com/d1/d2/...",
			"file#abc",
			"http://s1.com/d1/d2/file#abc"},
	}

	for _, test := range tests {
		got := ResolvePath(test.Base, test.Next)
		if got != test.Result {
			t.Fatalf("\n%q + %q:\nExp: %s\nGot: %s\n", test.Base, test.Next,
				test.Result, got)
		}
	}
}

func TestJSONPointer(t *testing.T) {
	data := map[string]any{
		"":    123,
		"str": "hello",
		"a~b": 234,
		"b/c": 345,
		"arr": []int{1, 2, 3},
		"obj": map[string]any{
			"":      666,
			"nil":   nil,
			"obj_a": 222,
			"obj_arr": []any{
				map[string]any{
					"o1": 333,
				},
				map[string]any{
					"o2": 444,
				},
			},
		},
	}

	type Test struct {
		Json   any
		Path   string
		Result any
	}

	tests := []Test{
		{data, ``, data},
		{data, `/`, data[""]},
		{data, `str`, data["str"]},
		{data, `/str`, data["str"]},
		{data, `/a~0b`, data["a~b"]},
		{data, `/b~1c`, data["b/c"]},
		{data, `/arr`, data["arr"]},
		{data, `/arr/0`, (data["arr"].([]int))[0]},
		{data, `/arr/2`, (data["arr"].([]int))[2]},
		{data, `/obj`, data["obj"]},
		{data, `/obj/`, 666},
		{data, `/obj/nil`, nil},
		{data, `/obj/obj_a`, 222},
		{data, `/obj/obj_arr`, ((data["obj"]).(map[string]any))["obj_arr"]},
		{data, `/obj/obj_arr/0`, (((data["obj"]).(map[string]any))["obj_arr"]).([]any)[0]},

		{data, `/obj/obj_arr/0/o1`, 333},
		{data, `/obj/obj_arr/1/o2`, 444},

		{data, `x`, `Attribute "x" not found`},
		{data, `/x`, `Attribute "x" not found`},
		{data, `/arr/`, `Index "/arr/" must be an integer`},
		{data, `/arr/foo`, `Index "/arr/foo" must be an integer`},
		{data, `/arr/-1`, `Index "/arr/-1" is out of bounds(0-2)`},
		{data, `/arr/3`, `Index "/arr/3" is out of bounds(0-2)`},
		{data, `/obj/obj_arr/`, `Index "/obj/obj_arr/" must be an integer`},
		{data, `/obj/obj_arr/o1/`, `Index "/obj/obj_arr/o1" must be an integer`},
		{data, `/obj/obj_arr/0/ox/`, `Attribute "obj/obj_arr/0/ox" not found`},
		{data, `/obj/obj_arr/1/o2/`, `Can't step into a type of "int", at: /obj/obj_arr/1/o2/`},
		{data, `/obj/obj_arr/2`, `Index "/obj/obj_arr/2" is out of bounds(0-1)`},
	}

	for _, test := range tests {
		res, err := GetJSONPointer(test.Json, test.Path)
		if err != nil {
			if err.Error() != test.Result {
				t.Fatalf("Test: %s\nExp: %s\nErr: %s", test.Path, test.Result,
					err)
			}
		} else if ToJSON(res) != ToJSON(test.Result) {
			t.Fatalf("Test: %s\nExp: %s\nGot: %s", test.Path,
				ToJSON(test.Result), ToJSON(res))
		}
	}
}

type FSHandler struct {
	Files map[string]string
}

func (h *FSHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// path, _ := strings.CutPrefix(req.URL.Path, "/")
	if file, ok := h.Files[req.URL.Path]; !ok {
		res.WriteHeader(404)
	} else {
		res.Write([]byte(file))
	}
}

// Test ProcessImports
func TestProcessImports(t *testing.T) {
	// Setup HTTP server
	httpPaths := map[string]string{
		"/empty":        "",
		"/notjson":      "hello there",
		"/emptyjson":    "{}",
		"/onelevel":     `{"foo":"bar","foo6":666}`,
		"/twolevel":     `{"foo":"bar3","foo6":{"bar":666}}`,
		"/twoarray":     `{"foo":"bar","foo6":[{"x":"y"},{"bar":667}]}`,
		"/nonfoo":       `{"bar":"zzz","foo":"qqq"}`,
		"/nest1":        `{"foo":"bar1","$import":"onelevel"}`,
		"/nest2":        `{"foo":"bar1","$import":"twoarray#/foo6/1"}`,
		"/nest3":        `{"$import": "twoarray#/foo6/1","f3":"bar"}`,
		"/nest3.1":      `{"$import": "/onelevel"}`,
		"/nest3.1.f":    `{"$import": "./onelevel"}`,
		"/nest3.1.f2":   `{"$import": "onelevel"}`,
		"/nest/nest4":   `{"foo":"bar1","$import":"/onelevel"}`,
		"/nest/nest4.f": `{"foo":"bar1","$import":"../onelevel"}`,
		"/nest/nest5":   `{"foo":"bar2","$import":"/nest/nest4"}`,
		"/nest/nest5.f": `{"foo":"bar2","$import":"../nest/nest4.f"}`,
		"/nest/nest6":   `{"foo":"bar2","$import":"http://localhost:9999/nest/nest4"}`,

		"/err1": `{"$import": "empty"}`,
		"/err2": `{"$import": "notjson"}`,
		"/err3": `{"$import": "/err3"}`,
		"/err4": `{"$import": "twolevel/bar"}`,

		"/nest7":      `{"$imports": []}`,
		"/nest7.err1": `{"$import": []}`,
		"/nest7.err2": `{"$imports": [1,2,3]}`,
		"/nest7.err3": `{"$import": "foo", "$imports": []}`,

		"/nest8":  `{"$imports": [ "onelevel", "twolevel" ]}`,
		"/nest9":  `{"$imports": [ "onelevel", "twolevel" ], "foo":"xxx"}`,
		"/nest10": `{"$imports": [ "nonfoo", "onelevel" ], "foo":"xxx"}`,
	}
	server := &http.Server{Addr: ":9999", Handler: &FSHandler{httpPaths}}
	go server.ListenAndServe()

	// Setup our local dir structure
	/*
		files := map[string]string{
			"empty":     "",
			"emptyjson": "{}",
			"simple": `{"foo":"bar"}`,
		}
	*/
	dir, _ := os.MkdirTemp("", "xreg")
	defer func() {
		os.RemoveAll(dir)
	}()
	for file, data := range httpPaths {
		os.MkdirAll(dir+"/"+path.Dir(file), 0777)
		os.WriteFile(dir+"/"+file, []byte(data), 0666)
	}

	// Wait for server
	for {
		if _, err := http.Get("http://localhost:9999/"); err == nil {
			break
		}
	}

	type Test struct {
		Path   string // filename or http url to json file
		Result string // json or error msg
	}

	tests := []Test{
		{"empty", "Error parsing JSON: EOF"},
		{"emptyjson", "{}"},

		{"onelevel", httpPaths["/onelevel"]},
		{"http:/onelevel", httpPaths["/onelevel"]},

		{"nest1", `{"foo":"bar1","foo6":666}`},
		{"http:/nest1", `{"foo":"bar1","foo6":666}`},

		{"nest2", `{"bar":667,"foo":"bar1"}`},
		{"http:/nest2", `{"bar":667,"foo":"bar1"}`},

		{"nest3", `{"bar":667,"f3":"bar"}`},
		{"http:/nest3", `{"bar":667,"f3":"bar"}`},

		{"nest3.1.f", `{"foo":"bar","foo6":666}`},
		{"http:/nest3.1", `{"foo":"bar","foo6":666}`},
		{"http:/nest3.1.f2", `{"foo":"bar","foo6":666}`},

		{"nest/nest4.f", `{"foo":"bar1","foo6":666}`},
		{"http:/nest/nest4", `{"foo":"bar1","foo6":666}`},
		{"http:/nest/nest4.f", `{"foo":"bar1","foo6":666}`},

		{"nest/nest5.f", `{"foo":"bar2","foo6":666}`},
		{"http:/nest/nest5", `{"foo":"bar2","foo6":666}`},

		{"nest/nest6", `{"foo":"bar2","foo6":666}`},
		{"http:/nest/nest6", `{"foo":"bar2","foo6":666}`},

		{"nest7", `{}`},

		{"nest7.err1", `In "tmp/nest7.err1", $import isn't a string`},
		{"nest7.err2", `In "tmp/nest7.err2", $imports contains a non-string value (1)`},
		{"nest7.err3", `In "tmp/nest7.err3", both $import and $imports is not allowed`},
		{"http:/nest7.err1", `In "http://localhost:9999/nest7.err1", $import isn't a string`},

		{"nest8", `{"foo":"bar","foo6":666}`},
		{"nest9", `{"foo":"xxx","foo6":666}`},
		{"nest10", `{"bar":"zzz","foo":"xxx","foo6":666}`},
	}

	mask := regexp.MustCompile("/tmp/xreg[^/]*")

	for i, test := range tests {
		t.Logf("Test #: %d", i)
		t.Logf("  Path: %s", test.Path)
		var buf []byte
		var err error
		if strings.HasPrefix(test.Path, "http:") {
			test.Path = "http://localhost:9999" + test.Path[5:]
			var res *http.Response
			if res, err = http.Get(test.Path); err == nil {
				if res.StatusCode != 200 {
					err = fmt.Errorf("Err %q: %s", test.Path, res.Status)
				} else {
					buf, err = io.ReadAll(res.Body)
					res.Body.Close()
				}
			}
		} else {
			test.Path = dir + "/" + test.Path
			buf, err = os.ReadFile(test.Path)
		}
		if err != nil {
			t.Fatalf(err.Error())
		}
		buf, err = ProcessImports(test.Path, buf,
			!strings.HasPrefix(test.Path, "http"))
		if err != nil {
			buf = []byte(err.Error())
		}
		buf = mask.ReplaceAll(buf, []byte("tmp"))
		if string(buf) != test.Result {
			t.Fatalf("\nExp: %s\nGot: %s", test.Result, string(buf))
		}
	}

	server.Shutdown(context.Background())
}
