package registry

import (
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
