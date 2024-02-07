package tests

import (
	"bytes"
	// "encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/duglin/xreg-github/registry"
)

type HTTPTest struct {
	Name       string
	URL        string
	Method     string
	ReqHeaders []string // name:value
	ReqBody    string

	Code        int
	HeaderMasks []string
	ResHeaders  []string // name:value
	BodyMasks   []string
	ResBody     string
}

func xCheckHTTP(t *testing.T, test *HTTPTest) {
	t.Logf("Test: %s", test.Name)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
	body := io.Reader(nil)
	if test.ReqBody != "" {
		body = bytes.NewReader([]byte(test.ReqBody))
	}
	req, err := http.NewRequest(test.Method, "http://localhost:8181/"+test.URL, body)
	xNoErr(t, err)
	for _, header := range test.ReqHeaders {
		name, value, _ := strings.Cut(header, ":")
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		req.Header.Add(name, value)
	}

	res, err := client.Do(req)
	resBody, _ := io.ReadAll(res.Body)

	xNoErr(t, err)
	xCheck(t, res.StatusCode == test.Code,
		fmt.Sprintf("Expected status %d, got %d\n%s", test.Code, res.StatusCode, string(resBody)))

	testHeaders := map[string]bool{}
	for _, header := range test.ResHeaders {
		name, value, _ := strings.Cut(header, ":")
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		testHeaders[strings.ToLower(name)] = true

		resValue := res.Header.Get(name)

		for _, mask := range test.HeaderMasks {
			var re *regexp.Regexp
			search, replace, _ := strings.Cut(mask, "|")
			if re = savedREs[search]; re == nil {
				re = regexp.MustCompile(search)
				savedREs[search] = re
			}

			value = re.ReplaceAllString(value, replace)
			resValue = re.ReplaceAllString(resValue, replace)
		}

		xCheckEqual(t, "Header:"+name+"\n", resValue, value)
	}

	// Make sure we don't have any extra xReg headers
	for k, _ := range res.Header {
		k = strings.ToLower(k)
		if !strings.HasPrefix(k, "xregistry-") {
			continue
		}
		if testHeaders[k] == true {
			continue
		}
		str := ""
		for k, v := range res.Header {
			str += fmt.Sprintf("%s:%s\n", k, v[0])
		}
		t.Errorf("%s:\nExtra header(%s)\nGot:\n%s", test.Name, k, str)
		t.FailNow()
	}

	testBody := test.ResBody

	for _, mask := range test.BodyMasks {
		var re *regexp.Regexp
		search, replace, found := strings.Cut(mask, "|")
		if !found {
			// Must be just a property name
			search = fmt.Sprintf(`("%s": ")(.*)(")`, search)
			replace = `${1}xxx${3}`
		}

		if re = savedREs[search]; re == nil {
			re = regexp.MustCompile(search)
			savedREs[search] = re
		}

		resBody = re.ReplaceAll(resBody, []byte(replace))
		testBody = re.ReplaceAllString(testBody, replace)
	}

	xCheckEqual(t, "Test: "+test.Name+"\nBody:\n",
		string(resBody), testBody)
	if t.Failed() {
		t.FailNow()
	}
}

var savedREs = map[string]*regexp.Regexp{}

func TestHTTPhtml(t *testing.T) {
	reg := NewRegistry("TestHTTPhtml")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	// Check as part of Reg request
	xCheckHTTP(t, &HTTPTest{
		Name:       "?html",
		URL:        "?html",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    "",

		Code:       200,
		ResHeaders: []string{"Content-Type:text/html"},
		ResBody: `<pre>
{
  "specversion": "0.5",
  "id": "TestHTTPhtml",
  "epoch": 1,
  "self": "<a href="http://localhost:8181/?html">http://localhost:8181/?html</a>"
}
`,
	})
}

func TestHTTPModel(t *testing.T) {
	reg := NewRegistry("TestHTTPModel")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	// Check as part of Reg request
	xCheckHTTP(t, &HTTPTest{
		Name:       "?model",
		URL:        "?model",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    "",

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "0.5",
  "id": "TestHTTPModel",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "model": {}
}
`,
	})

	// Just model, no reg content
	xCheckHTTP(t, &HTTPTest{
		Name:       "/model",
		URL:        "/model",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    "",

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{}
`,
	})

	// Create model tests
	xCheckHTTP(t, &HTTPTest{
		Name:       "Create empty model",
		URL:        "/model",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{}`,

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "Create model - just schema",
		URL:        "/model",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{"schemas":["schema1"]}`,

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "schemas": [
    "schema1"
  ]
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "Create model - defaults",
		URL:        "/model",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "schemas": [
    "schema1"
  ],
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file"
        }
      }
    }
  }
}`,

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "schemas": [
    "schema1"
  ],
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "versions": 1,
          "versionid": true,
          "latest": true,
          "hasdocument": true
        }
      }
    }
  }
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "Create model - full",
		URL:        "/model",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "schemas": [
    "schema1"
  ],
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "versions": 1,
          "versionid": true,
          "latest": true,
          "hasdocument": false
        }
      }
    }
  }
}`,

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "schemas": [
    "schema1"
  ],
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "versions": 1,
          "versionid": true,
          "latest": true,
          "hasdocument": false
        }
      }
    }
  }
}
`,
	})
}

func TestHTTPRegistry(t *testing.T) {
	reg := NewRegistry("TestHTTPRegistry")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	reg.Model.AddAttr("myany", registry.ANY)
	reg.Model.AddAttr("mybool", registry.BOOLEAN)
	reg.Model.AddAttr("mydec", registry.DECIMAL)
	reg.Model.AddAttr("myint", registry.INTEGER)
	reg.Model.AddAttr("mystr", registry.STRING)
	reg.Model.AddAttr("mytime", registry.TIMESTAMP)
	reg.Model.AddAttr("myuint", registry.UINTEGER)
	reg.Model.AddAttr("myuri", registry.URI)
	reg.Model.AddAttr("myuriref", registry.URI_REFERENCE)
	reg.Model.AddAttr("myuritemplate", registry.URI_TEMPLATE)
	reg.Model.AddAttr("myurl", registry.URL)

	attr, _ := reg.Model.AddAttrObj("myobj1")
	attr.Item.AddAttr("mystr1", registry.STRING)
	attr.Item.AddAttr("myint1", registry.INTEGER)
	attr.Item.AddAttr("*", registry.ANY)

	attr, _ = reg.Model.AddAttrObj("myobj2")
	attr.Item.AddAttr("mystr2", registry.STRING)
	obj2, _ := attr.Item.AddAttrObj("myobj2_1")
	obj2.AddAttr("*", registry.INTEGER)

	item := registry.NewItem(registry.ANY)
	attr, _ = reg.Model.AddAttrArray("myarrayany", item)
	attr, _ = reg.Model.AddAttrMap("mymapany", item)

	item = registry.NewItem(registry.UINTEGER)
	attr, _ = reg.Model.AddAttrArray("myarrayuint", item)
	attr, _ = reg.Model.AddAttrMap("mymapuint", item)

	item = registry.NewItemObject()
	attr, _ = reg.Model.AddAttrArray("myarrayemptyobj", item)

	item = registry.NewItemObject()
	item.AddAttr("mapobj_int", registry.INTEGER)
	attr, _ = reg.Model.AddAttrMap("mymapobj", item)

	xCheckHTTP(t, &HTTPTest{
		Name:       "POST reg",
		URL:        "/",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    "",
		Code:       405,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "POST not allowed on the root of the registry\n",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT reg - empty",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "",
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "0.5",
  "id": "TestHTTPRegistry",
  "epoch": 2,
  "self": "http://localhost:8181/"
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT reg - empty json",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "{}",
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "0.5",
  "id": "TestHTTPRegistry",
  "epoch": 3,
  "self": "http://localhost:8181/"
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT reg - good epoch",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "epoch": 3
}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "0.5",
  "id": "TestHTTPRegistry",
  "epoch": 4,
  "self": "http://localhost:8181/"
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT reg - bad epoch",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "epoch":33
}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "Error processing registry: Attribute \"epoch\" doesn't match existing value (4)\n",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT reg - full good",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "specversion": "0.5",
  "id": "TestHTTPRegistry",
  "epoch": 4,

  "myany": 5.5,
  "mybool": true,
  "mydec": 2.4,
  "myint": -666,
  "mystr": "hello",
  "mytime": "2024-01-01T12:01:02Z",
  "myuint": 123,
  "myuri": "http://foo.com",
  "myuriref": "/foo",
  "myuritemplate": "...",
  "myurl": "http://someurl.com",
  "myobj1": {
    "mystr1": "str1",
    "myint1": 345,
    "myobj1_ext": 9.2
  },
  "myobj2": {
    "mystr2": "str2",
    "myobj2_1": {
      "myobj2_1_ext": 444
    }
  },
  "myarrayany": [
    { "any1": -333},
    "any2-str"
  ],
  "mymapany": {
    "key1": 1,
    "key2": "2"
  },
  "myarrayuint": [ 2, 999 ],
  "myarrayemptyobj": [],
  "mymapobj": {
    "mymapobj_k1": { "mapobj_int": 333 }
  }
}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "0.5",
  "id": "TestHTTPRegistry",
  "epoch": 5,
  "self": "http://localhost:8181/",
  "myany": 5.5,
  "myarrayany": [
    {
      "any1": -333
    },
    "any2-str"
  ],
  "myarrayemptyobj": [],
  "myarrayuint": [
    2,
    999
  ],
  "mybool": true,
  "mydec": 2.4,
  "myint": -666,
  "mymapany": {
    "key1": 1,
    "key2": "2"
  },
  "mymapobj": {
    "mymapobj_k1": {
      "mapobj_int": 333
    }
  },
  "myobj1": {
    "myint1": 345,
    "myobj1_ext": 9.2,
    "mystr1": "str1"
  },
  "myobj2": {
    "myobj2_1": {
      "myobj2_1_ext": 444
    },
    "mystr2": "str2"
  },
  "mystr": "hello",
  "mytime": "2024-01-01T12:01:02Z",
  "myuint": 123,
  "myuri": "http://foo.com",
  "myuriref": "/foo",
  "myuritemplate": "...",
  "myurl": "http://someurl.com"
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT reg - bad object",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "mymapobj": {
    "mapobj_int": 333
  }
}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody: `Error processing registry: Attribute "mymapobj.mapobj_int" must be an object
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT reg - full empties",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "specversion": "0.5",
  "id": "TestHTTPRegistry",
  "epoch": 5,

  "myany": 5.5,
  "myint": 4.0,
  "mybool": null,
  "myuri": null,
  "myobj1": {},
  "myobj2": null,
  "myarrayany": [],
  "mymapany": {},
  "myarrayuint": null,
  "myarrayemptyobj": [],
  "mymapobj": {
    "mymapobj_key1": {}
  },
  "mymapuint": {
    "asd": null
  }
}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "0.5",
  "id": "TestHTTPRegistry",
  "epoch": 6,
  "self": "http://localhost:8181/",
  "myany": 5.5,
  "myarrayany": [],
  "myarrayemptyobj": [],
  "myint": 4,
  "mymapany": {},
  "mymapobj": {
    "mymapobj_key1": {}
  },
  "mymapuint": {},
  "myobj1": {}
}
`,
	})

	type typeTest struct {
		request  string
		response string
	}

	typeTests := []typeTest{
		{request: `{"epoch":123}`,
			response: `Attribute "epoch" doesn't match existing value (6)`},
		{request: `{"epoch":-123}`,
			response: `Attribute "epoch" must be a uinteger`},
		{request: `{"epoch":"asd"}`,
			response: `Attribute "epoch" must be a uinteger`},
		{request: `{"mybool":123}`,
			response: `Attribute "mybool" must be a boolean`},
		{request: `{"mybool":"False"}`,
			response: `Attribute "mybool" must be a boolean`},
		{request: `{"mydec":[ 1 ]}`,
			response: `Attribute "mydec" must be a decimal`},
		{request: `{"mydec": "asd" }`,
			response: `Attribute "mydec" must be a decimal`},
		{request: `{"myint": 1.01 }`,
			response: `Attribute "myint" must be an integer`},
		{request: `{"myint": {} }`,
			response: `Attribute "myint" must be an integer`},
		{request: `{"mystr": {} }`,
			response: `Attribute "mystr" must be a string`},
		{request: `{"mystr": 123 }`,
			response: `Attribute "mystr" must be a string`},
		{request: `{"mystr": true }`,
			response: `Attribute "mystr" must be a string`},
		{request: `{"mytime": true }`,
			response: `Attribute "mytime" must be a timestamp`},
		{request: `{"mytime": "12-12-12" }`,
			response: `Attribute "mytime" is a malformed timestamp`},
		{request: `{"myuint": "str" }`,
			response: `Attribute "myuint" must be a uinteger`},
		{request: `{"myuint": "123" }`,
			response: `Attribute "myuint" must be a uinteger`},
		{request: `{"myuint": -123 }`,
			response: `Attribute "myuint" must be a uinteger`},
		{request: `{"myuri": 123 }`,
			response: `Attribute "myuri" must be a uri`},
		{request: `{"myuriref": 123 }`,
			response: `Attribute "myuriref" must be a uri-reference`},
		{request: `{"myuritemplate": 123 }`,
			response: `Attribute "myuritemplate" must be a uri-template`},
		{request: `{"myurl": 123 }`,
			response: `Attribute "myurl" must be a url`},
		{request: `{"myobj1": 123 }`,
			response: `Attribute "myobj1" must be an object`},
		{request: `{"myobj1": [ 123 ] }`,
			response: `Attribute "myobj1" must be an object`},
		{request: `{"myobj1": { "mystr1": 123 } }`,
			response: `Attribute "myobj1.mystr1" must be a string`},
		{request: `{"myobj2": { "ext": 123 } }`,
			response: `Invalid extension(s) in "myobj2": ext`},
		{request: `{"myobj2": { "myobj2_1": { "ext": "str" } } }`,
			response: `Attribute "myobj2.myobj2_1.ext" must be an integer`},
		{request: `{"myarrayuint": [ 123, -123 ] }`,
			response: `Attribute "myarrayuint[1]" must be a uinteger`},
		{request: `{"myarrayuint": [ "asd" ] }`,
			response: `Attribute "myarrayuint[0]" must be a uinteger`},
		{request: `{"myarrayuint": [ 123, null] }`,
			response: `Attribute "myarrayuint[1]" must be a uinteger`},
		{request: `{"myarrayuint": 123 }`,
			response: `Attribute "myarrayuint" must be an array`},
		{request: `{"mymapuint": 123 }`,
			response: `Attribute "mymapuint" must be a map`},
		{request: `{"mymapuint": { "asd" : -123 }}`,
			response: `Attribute "mymapuint.asd" must be a uinteger`},
		// {request: `{"mymapuint": { "asd" : null }}`,
		// response: `Attribute "mymapuint.asd" must be a uinteger`},
		{request: `{"myarrayemptyobj": [ { "asd": true } ] }`,
			response: `Invalid extension(s) in "myarrayemptyobj[0]": asd`},
		{request: `{"myarrayemptyobj": [ [ true ] ] }`,
			response: `Attribute "myarrayemptyobj[0]" must be an object`},
		{request: `{"mymapobj": { "asd" : { "mapobj_int" : true } } }`,
			response: `Attribute "mymapobj.asd.mapobj_int" must be an integer`},
		{request: `{"mymapobj": { "asd" : { "qwe" : true } } }`,
			response: `Invalid extension(s) in "mymapobj.asd": qwe`},
		{request: `{"mymapobj": [ true ]}`,
			response: `Attribute "mymapobj" must be a map`},
	}

	for _, test := range typeTests {
		xCheckHTTP(t, &HTTPTest{
			Name:       "PUT reg - bad type - request: " + test.request,
			URL:        "/",
			Method:     "PUT",
			ReqHeaders: []string{},
			ReqBody:    test.request,
			Code:       400,
			ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
			ResBody:    `Error processing registry: ` + test.response + "\n",
		})
	}

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT reg - bad self - ignored",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "self": 123
}`,
		Code:       400,
		ResHeaders: []string{"application/json"},
		ResBody: `Error processing registry: Attribute "self" must be a string
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT reg - bad id",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "id": 123
}`,
		Code:       400,
		ResHeaders: []string{"application/json"},
		ResBody:    "Error processing registry: Attribute \"id\" must be a string\n",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT reg - bad id",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "id": "foo"
}`,
		Code:       400,
		ResHeaders: []string{"application/json"},
		ResBody:    "Error processing registry: Can't change the ID of an entity(TestHTTPRegistry->foo)\n",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT reg - options",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "documentation": "docs"
}`,
		Code:       200,
		ResHeaders: []string{"application/json"},
		ResBody: `{
  "specversion": "0.5",
  "id": "TestHTTPRegistry",
  "epoch": 7,
  "self": "http://localhost:8181/",
  "documentation": "docs"
}
`})

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT reg - options - del",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "id": null,
  "self": null
}`,
		Code:       200,
		ResHeaders: []string{"application/json"},
		ResBody: `{
  "specversion": "0.5",
  "id": "TestHTTPRegistry",
  "epoch": 8,
  "self": "http://localhost:8181/"
}
`})

}

func TestHTTPGroups(t *testing.T) {
	reg := NewRegistry("TestHTTPGroups")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddAttr("format", registry.STRING)
	gm.AddResourceModel("files", "file", 0, true, true, true)

	attr, _ := gm.AddAttrObj("myobj")
	attr.Item.AddAttr("foo", registry.STRING)
	attr.Item.AddAttr("*", registry.ANY)

	item := registry.NewItem(registry.ANY)
	attr, _ = gm.AddAttrArray("myarray", item)
	attr, _ = gm.AddAttrMap("mymap", item)

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT groups - fail",
		URL:        "/dirs",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "",
		Code:       405,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "PUT not allowed on collections\n",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:        "Create group - empty",
		URL:         "/dirs",
		Method:      "POST",
		ReqHeaders:  []string{},
		ReqBody:     "",
		Code:        201,
		HeaderMasks: []string{"dirs/[a-zA-Z0-9]*|dirs/xxx"},
		ResHeaders: []string{
			"Content-Type:application/json",
			"Location:http://localhost:8181/dirs/xxx",
		},
		BodyMasks: []string{"id", "dirs/[a-zA-Z0-9]*|dirs/xxx"},
		ResBody: `{
  "id": "xxx",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/xxx",

  "filescount": 0,
  "filesurl": "http://localhost:8181/dirs/xxx/files"
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:        "Create group - {}",
		URL:         "/dirs",
		Method:      "POST",
		ReqHeaders:  []string{},
		ReqBody:     "{}",
		Code:        201,
		HeaderMasks: []string{"dirs/[a-zA-Z0-9]*|dirs/xxx"},
		ResHeaders: []string{
			"Content-Type:application/json",
			"Location:http://localhost:8181/dirs/xxx",
		},
		BodyMasks: []string{"id", "dirs/[a-zA-Z0-9]*|dirs/xxx"},
		ResBody: `{
  "id": "xxx",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/xxx",

  "filescount": 0,
  "filesurl": "http://localhost:8181/dirs/xxx/files"
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "POST group - full",
		URL:        "/dirs",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
  "id":"dir1",
  "name":"my group",
  "description":"desc",
  "documentation":"docs-url",
  "labels": {
    "label1": "value1",
    "label2": "5",
    "label3": "123.456",
    "label4": "",
    "label5": null
  },
  "format":"my group",
  "myarray": [ "hello", 5 ],
  "mymap": { "item1": 5.5 },
  "myobj": { "item2": [ "hi" ] }
}`,
		Code: 201,
		ResHeaders: []string{
			"Content-Type:application/json",
			"Location:http://localhost:8181/dirs/dir1",
		},
		ResBody: `{
  "id": "dir1",
  "name": "my group",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/dir1",
  "description": "desc",
  "documentation": "docs-url",
  "labels": {
    "label1": "value1",
    "label2": "5",
    "label3": "123.456",
    "label4": ""
  },
  "format": "my group",
  "myarray": [
    "hello",
    5
  ],
  "mymap": {
    "item1": 5.5
  },
  "myobj": {
    "item2": [
      "hi"
    ]
  },

  "filescount": 0,
  "filesurl": "http://localhost:8181/dirs/dir1/files"
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT group - update",
		URL:        "/dirs/dir1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "id":"dir1",
  "name":"my group new",
  "epoch": 1,
  "description":"desc new",
  "documentation":"docs-url-new",
  "labels": {
    "label.new": "new"
  },
  "format": "myformat/1",
  "myarray": [],
  "mymap": {},
  "myobj": {}
}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "id": "dir1",
  "name": "my group new",
  "epoch": 2,
  "self": "http://localhost:8181/dirs/dir1",
  "description": "desc new",
  "documentation": "docs-url-new",
  "labels": {
    "label.new": "new"
  },
  "format": "myformat/1",
  "myarray": [],
  "mymap": {},
  "myobj": {},

  "filescount": 0,
  "filesurl": "http://localhost:8181/dirs/dir1/files"
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT group - update - null",
		URL:        "/dirs/dir1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "id":"dir1",
  "name":"my group new",
  "epoch": 2,
  "description":"desc new",
  "documentation":"docs-url-new",
  "labels": {
    "label.new": "new"
  },
  "format": "myformat/1",
  "myarray": null,
  "mymap": null,
  "myobj": null
}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "id": "dir1",
  "name": "my group new",
  "epoch": 3,
  "self": "http://localhost:8181/dirs/dir1",
  "description": "desc new",
  "documentation": "docs-url-new",
  "labels": {
    "label.new": "new"
  },
  "format": "myformat/1",

  "filescount": 0,
  "filesurl": "http://localhost:8181/dirs/dir1/files"
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT group - update - err epoch",
		URL:        "/dirs/dir1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "id":"dir1",
  "name":"my group new",
  "epoch": 10,
  "description":"desc new",
  "documentation":"docs-url-new",
  "labels": {
    "label.new": "new"
  },
  "format":"myformat/1"
}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "Error processing group: Attribute \"epoch\" doesn't match existing value (3)\n",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT group - update - err id",
		URL:        "/dirs/dir1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{ "id":"dir2" }`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "Error processing group: Can't change the ID of an entity(dir1->dir2)\n",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT group - update - clear",
		URL:        "/dirs/dir1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "id": "dir1",
  "epoch": 4,
  "self": "http://localhost:8181/dirs/dir1",

  "filescount": 0,
  "filesurl": "http://localhost:8181/dirs/dir1/files"
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT group - create - error",
		URL:        "/dirs/dir2",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "id":"dir3",
  "name":"my group new",
  "epoch": 1,
  "description":"desc new",
  "documentation":"docs-url-new",
  "labels": {
    "label.new": "new"
  },
  "format": "myformat/1"
}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "Error processing group: Can't change the ID of an entity(dir2->dir3)\n",
	})

}

func TestHTTPResourcesHeaders(t *testing.T) {
	reg := NewRegistry("TestHTTPResourcesHeaders")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)
	reg.AddGroup("dirs", "dir1")

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT resources - fail",
		URL:        "/dirs/dir1/files",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "",
		Code:       405,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "PUT not allowed on collections\n",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "POST resources - empty",
		URL:        "/dirs/dir1/files",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    "",
		Code:       201,
		HeaderMasks: []string{
			"^[a-z0-9]{8}$|xxx",
			"files/[^/]+|files/xxx",
		},
		ResHeaders: []string{
			"Content-Type: ",
			"xRegistry-id: xxx",
			"xRegistry-epoch: 1",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/dir1/files/xxx/versions/1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/xxx",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/xxx/versions",
			"Location: http://localhost:8181/dirs/dir1/files/xxx",
			"Content-Location: http://localhost:8181/dirs/dir1/files/xxx/versions/1",
			"Content-Length: 0",
		},
		ResBody: ``,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "POST resources - w/doc",
		URL:        "/dirs/dir1/files",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    "My cool doc",
		Code:       201,
		HeaderMasks: []string{
			"^[a-z0-9]{8}$|xxx",
			"files/[^/]+|files/xxx",
		},
		ResHeaders: []string{
			"Content-Type: text/plain; charset=utf-8",
			"xRegistry-id: xxx",
			"xRegistry-epoch: 1",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/dir1/files/xxx/versions/1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/xxx",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/xxx/versions",
			"Location: http://localhost:8181/dirs/dir1/files/xxx",
			"Content-Location: http://localhost:8181/dirs/dir1/files/xxx/versions/1",
			"Content-Length: 11",
		},
		ResBody: `My cool doc`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT resources - w/doc",
		URL:        "/dirs/dir1/files/f1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "My cool doc",
		Code:       201,
		ResHeaders: []string{
			"Content-Type: text/plain; charset=utf-8",
			"xRegistry-id: f1",
			"xRegistry-epoch: 1",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/dir1/files/f1/versions/1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f1",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f1/versions",
			"Location: http://localhost:8181/dirs/dir1/files/f1",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f1/versions/1",
			"Content-Length: 11",
		},
		ResBody: `My cool doc`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT resources - w/doc - bad id",
		URL:        "/dirs/dir1/files/f1",
		Method:     "PUT",
		ReqHeaders: []string{"xRegistry-id:f2"},
		ReqBody:    "My cool doc",
		Code:       400,
		ResHeaders: []string{
			"Content-Type: text/plain; charset=utf-8",
		},
		ResBody: "Metadata id(f2) doesn't match ID in URL(f1)\n",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "POST resources - w/doc + data",
		URL:    "/dirs/dir1/files",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-id: f3",
			"xRegistry-name: my doc",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-labels-l3: null",
			"xRegistry-origin: foo.com",
		},
		ReqBody:     "My cool doc",
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Type: text/plain; charset=utf-8",
			"xRegistry-id: f3",
			"xRegistry-name: my doc",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-origin: foo.com",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"Location: http://localhost:8181/dirs/dir1/files/f3",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"Content-Length: 11",
		},
		ResBody: `My cool doc`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:        "PUT resources - update latest - content",
		URL:         "/dirs/dir1/files/f3",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     "My cool doc - v2",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Type: text/plain; charset=utf-8",
			"xRegistry-id: f3",
			"xRegistry-name: my doc",
			"xRegistry-epoch: 2",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-origin: foo.com",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"Content-Length: 16",
		},
		ResBody: `My cool doc - v2`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "PUT resources - create - URL",
		URL:    "/dirs/dir1/files/f4",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-name: my doc",
			"xRegistry-fileurl: http://example.com",
		},
		ReqBody:     "",
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f4",
			"xRegistry-name: my doc",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f4",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/dir1/files/f4/versions/1",
			"xRegistry-name: my doc",
			"xRegistry-fileurl: http://example.com",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f4/versions",
			"Location: http://localhost:8181/dirs/dir1/files/f4",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f4/versions/1",
		},
		ResBody: "",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "PUT resources - update latest - URL",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-fileurl: http://example.com",
		},
		ReqBody:     "",
		Code:        303,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f3",
			"xRegistry-name: my doc",
			"xRegistry-epoch: 3",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-origin: foo.com",
			"xRegistry-fileurl: http://example.com",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"Location: http://example.com",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
		},
		ResBody: "",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "PUT resources - update latest - URL + body - error",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-fileurl: example.com",
		},
		ReqBody:     "My cool doc - v2",
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody:     "'xRegistry-fileurl' isn't allowed if there's a body\n",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "PUT resources - update latest - URL - null",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-fileurl: null",
		},
		ReqBody:     "",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f3",
			"xRegistry-name: my doc",
			"xRegistry-epoch: 4",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-origin: foo.com",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
		},
		ResBody: "",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:        "PUT resources - update latest - w/body",
		URL:         "/dirs/dir1/files/f3",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     "another body",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f3",
			"xRegistry-name: my doc",
			"xRegistry-epoch: 5",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-origin: foo.com",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
		},
		ResBody: "another body",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "PUT resources - update latest - w/body - clear 1 prop",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-description: null",
		},
		ReqBody:     "another body",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f3",
			"xRegistry-name: my doc",
			"xRegistry-epoch: 6",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-origin: foo.com",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
		},
		ResBody: "another body",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "PUT resources - update latest - w/body - edit 2 label",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-labels-l1: l1l1",
			"xRegistry-labels-l4: 4444",
		},
		ReqBody:     "another body",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f3",
			"xRegistry-name: my doc",
			"xRegistry-epoch: 7",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-documentation: my doc url",
			"xRegistry-origin: foo.com",
			"xRegistry-labels-l1: l1l1",
			"xRegistry-labels-l4: 4444",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
		},
		ResBody: "another body",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "PUT resources - update latest - w/body - edit 1 label",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-labels-l3: 3333",
		},
		ReqBody:     "another body",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f3",
			"xRegistry-name: my doc",
			"xRegistry-epoch: 8",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-documentation: my doc url",
			"xRegistry-origin: foo.com",
			"xRegistry-labels-l3: 3333",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
		},
		ResBody: "another body",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "PUT resources - update latest - w/body - delete labels",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-labels: null",
		},
		ReqBody:     "another body",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f3",
			"xRegistry-name: my doc",
			"xRegistry-epoch: 9",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-documentation: my doc url",
			"xRegistry-origin: foo.com",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
		},
		ResBody: "another body",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "PUT resources - update latest - w/body - delete+add labels",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-labels: null",
			"xRegistry-labels-foo: foo",
		},
		ReqBody:     "another body",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f3",
			"xRegistry-name: my doc",
			"xRegistry-epoch: 10",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-documentation: my doc url",
			"xRegistry-origin: foo.com",
			"xRegistry-labels-foo: foo",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
		},
		ResBody: "another body",
	})

	// Checking GET+PUTs

	// 1
	res, err := http.Get("http://localhost:8181/dirs/dir1/files/f3")
	xNoErr(t, err)
	body, err := io.ReadAll(res.Body)
	xNoErr(t, err)

	xCheckHTTP(t, &HTTPTest{
		Name:        "PUT resources - echo'ing resource GET",
		URL:         "/dirs/dir1/files/f3",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     string(body),
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f3",
			"xRegistry-name: my doc",
			"xRegistry-epoch: 11",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-documentation: my doc url",
			"xRegistry-origin: foo.com",
			"xRegistry-labels-foo: foo",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
		},
		ResBody: string(body),
	})

	// 2
	res, err = http.Get("http://localhost:8181/dirs/dir1/files/f3?meta")
	xNoErr(t, err)
	body, err = io.ReadAll(res.Body)
	xNoErr(t, err)

	resBody := strings.Replace(string(body), `"epoch": 11`, `"epoch": 12`, 1)
	xCheckHTTP(t, &HTTPTest{
		Name:        "PUT resources - echo'ing resource GET?meta",
		URL:         "/dirs/dir1/files/f3?meta",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     string(body),
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody:     resBody,
	})

	// 3
	res, err = http.Get("http://localhost:8181/")
	xNoErr(t, err)
	body, err = io.ReadAll(res.Body)
	xNoErr(t, err)

	resBody = strings.Replace(string(body), `"epoch": 1`, `"epoch": 2`, 1)
	xCheckHTTP(t, &HTTPTest{
		Name:        "PUT resources - echo'ing rergistry GET",
		URL:         "/",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     string(body),
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody:     resBody,
	})
}

func TestHTTPResourcesContentHeaders(t *testing.T) {
	reg := NewRegistry("TestHTTPResourcesContentHeaders")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	d, _ := reg.AddGroup("dirs", "d1")

	// ProxyURL
	f, _ := d.AddResource("files", "f1-proxy", "v1")
	f.Set(NewPP().P("#resource").UI(), "Hello world! v1")

	v, _ := f.AddVersion("v2")
	v.Set(NewPP().P("#resourceURL").UI(), "http://localhost:8181/EMPTY-URL")

	v, _ = f.AddVersion("v3")
	v.Set(NewPP().P("#resourceProxyURL").UI(), "http://localhost:8181/EMPTY-Proxy")

	// URL
	f, _ = d.AddResource("files", "f2-url", "v1")
	f.Set(NewPP().P("#resource").UI(), "Hello world! v1")

	v, _ = f.AddVersion("v2")
	v.Set(NewPP().P("#resourceProxyURL").UI(), "http://localhost:8181/EMPTY-Proxy")

	v, _ = f.AddVersion("v3")
	v.Set(NewPP().P("#resourceURL").UI(), "http://localhost:8181/EMPTY-URL")

	// Resource
	f, _ = d.AddResource("files", "f3-resource", "v1")
	f.Set(NewPP().P("#resourceProxyURL").UI(), "http://localhost:8181/EMPTY-Proxy")

	v, _ = f.AddVersion("v2")
	v.Set(NewPP().P("#resourceURL").UI(), "http://localhost:8181/EMPTY-URL")

	v, _ = f.AddVersion("v3")
	v.Set(NewPP().P("#resource").UI(), "Hello world! v3")

	// /dirs/d1/files/f1-proxy/v1 - resource
	//                        /v2 - URL
	//                        /v3 - ProxyURL  <- latest
	// /dirs/d1/files/f2-url/v1 - resource
	//                      /v2 - ProxyURL
	//                      /v3 - URL  <- latest
	// /dirs/d1/files/f3-resource/v1 - ProxyURL
	//                           /v2 - URL
	//                           /v3 - resource  <- latest

	xCheckHTTP(t, &HTTPTest{
		Name:        "GET resource - latest - f1",
		URL:         "/dirs/d1/files/f1-proxy",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f1-proxy",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1-proxy",
			"xRegistry-latestversionid: v3",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/d1/files/f1-proxy/versions/v3",
			"xRegistry-versionscount: 3",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f1-proxy/versions",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1-proxy/versions/v3",
		},
		ResBody: "hello-Proxy",
	})
	CompareContentMeta(t, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1-proxy",
		Body:    "hello-Proxy",
		Headers: nil,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:        "GET resource - latest - f1/v3",
		URL:         "/dirs/d1/files/f1-proxy/versions/v3",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: v3",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1-proxy/versions/v3",
			"xRegistry-latest: true",
		},
		ResBody: "hello-Proxy",
	})
	CompareContentMeta(t, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1-proxy/versions/v3",
		Body:    "hello-Proxy",
		Headers: nil,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:        "GET resource - latest - f1/v2",
		URL:         "/dirs/d1/files/f1-proxy/versions/v2",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        303,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: v2",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1-proxy/versions/v2",
			"xRegistry-fileurl: http://localhost:8181/EMPTY-URL",
			"Location: http://localhost:8181/EMPTY-URL",
		},
		ResBody: "",
	})
	CompareContentMeta(t, &Test{
		Code: 303,
		URL:  "dirs/d1/files/f1-proxy/versions/v2",
		Headers: []string{
			"Location: http://localhost:8181/EMPTY-URL",
		},
	})

	xCheckHTTP(t, &HTTPTest{
		Name:        "GET resource - latest - f2",
		URL:         "/dirs/d1/files/f2-url",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        303,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f2-url",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f2-url",
			"xRegistry-latestversionid: v3",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/d1/files/f2-url/versions/v3",
			"xRegistry-fileurl: http://localhost:8181/EMPTY-URL",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f2-url/versions",
			"xRegistry-versionscount: 3",
			"Location: http://localhost:8181/EMPTY-URL",
		},
		ResBody: "",
	})
	CompareContentMeta(t, &Test{
		Code: 303,
		URL:  "dirs/d1/files/f2-url",
		Headers: []string{
			"Location: http://localhost:8181/EMPTY-URL",
		},
	})

	xCheckHTTP(t, &HTTPTest{
		Name:        "GET resource - latest - f3",
		URL:         "/dirs/d1/files/f3-resource",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f3-resource",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f3-resource",
			"xRegistry-latestversionid: v3",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/d1/files/f3-resource/versions/v3",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f3-resource/versions",
			"xRegistry-versionscount: 3",
		},
		ResBody: "Hello world! v3",
	})
	CompareContentMeta(t, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f3-resource/versions/v3",
		Headers: []string{},
		Body:    "Hello world! v3",
	})
}

func TestHTTPVersions(t *testing.T) {
	reg := NewRegistry("TestHTTPVersions")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	reg.AddGroup("dirs", "d1")

	// ProxyURL
	// f, _ := d.AddResource("files", "f1-proxy", "v1")
	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT file f1-proxy",
		URL:        "/dirs/d1/files/f1-proxy?meta",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "file": "Hello world! v1"
}`,
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/f1-proxy?meta",
		},
		ResBody: `{
  "id": "f1-proxy",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1-proxy?meta",
  "latestversionid": "1",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions/1?meta",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions"
}
`,
	})

	// Now inline "file"
	xCheckHTTP(t, &HTTPTest{
		Name:        "GET file f1-proxy + inline",
		URL:         "/dirs/d1/files/f1-proxy?meta&inline=file",
		Method:      "GET",
		ReqHeaders:  []string{},
		ReqBody:     ``,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "id": "f1-proxy",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1-proxy?meta",
  "latestversionid": "1",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions/1?meta",
  "filebase64": "SGVsbG8gd29ybGQhIHYx",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions"
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:        "GET file f1-proxy/v/1",
		URL:         "/dirs/d1/files/f1-proxy/versions/1",
		Method:      "GET",
		ReqHeaders:  []string{},
		ReqBody:     ``,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: 1",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1-proxy/versions/1",
			"xRegistry-latest: true",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1-proxy/versions/1",
			"Content-Length: 15",
		},
		ResBody: "Hello world! v1",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:        "GET file f1-proxy/v/1+inline",
		URL:         "/dirs/d1/files/f1-proxy?meta&inline=file",
		Method:      "GET",
		ReqHeaders:  []string{},
		ReqBody:     "",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "id": "f1-proxy",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1-proxy?meta",
  "latestversionid": "1",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions/1?meta",
  "filebase64": "SGVsbG8gd29ybGQhIHYx",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions"
}
`,
	})

	// add new version via POST to "versions" collection
	xCheckHTTP(t, &HTTPTest{
		Name:       "POST file f1-proxy - create v2?meta",
		URL:        "/dirs/d1/files/f1-proxy/versions?meta",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
		  "id": "v2",
          "file": "Hello world! v2"
		}`,
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/f1-proxy/versions/v2?meta",
		},
		ResBody: `{
  "id": "v2",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1-proxy/versions/v2?meta",
  "latest": true
}
`,
	})

	// add new version via POST to "versions" collection
	xCheckHTTP(t, &HTTPTest{
		Name:        "POST file f1-proxy - create v2 - no meta",
		URL:         "/dirs/d1/files/f1-proxy/versions",
		Method:      "POST",
		ReqHeaders:  []string{},
		ReqBody:     `this is v3`,
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id:2",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f1-proxy/versions/2",
			"xRegistry-latest:true",
			"Location:http://localhost:8181/dirs/d1/files/f1-proxy/versions/2",
			"Content-Location:http://localhost:8181/dirs/d1/files/f1-proxy/versions/2",
			"Content-Length:10",
		},
		ResBody: `this is v3`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:        "GET file f1-proxy - v2 + inline",
		URL:         "/dirs/d1/files/f1-proxy/versions/v2?meta&inline=file",
		Method:      "GET",
		ReqHeaders:  []string{},
		ReqBody:     ``,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "id": "v2",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1-proxy/versions/v2?meta",
  "filebase64": "SGVsbG8gd29ybGQhIHYy"
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:        "PUT file f1-proxy - update contents",
		URL:         "/dirs/d1/files/f1-proxy",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     `more data`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id:f1-proxy",
			"xRegistry-epoch:2",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f1-proxy",
			"xRegistry-latestversionid:2",
			"xRegistry-latestversionurl:http://localhost:8181/dirs/d1/files/f1-proxy/versions/2",
			"xRegistry-versionscount:3",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f1-proxy/versions",
		},
		ResBody: `more data`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:        "GET file f1-proxy - check update",
		URL:         "/dirs/d1/files/f1-proxy",
		Method:      "GET",
		ReqHeaders:  []string{},
		ReqBody:     ``,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id:f1-proxy",
			"xRegistry-epoch:2",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f1-proxy",
			"xRegistry-latestversionid:2",
			"xRegistry-latestversionurl:http://localhost:8181/dirs/d1/files/f1-proxy/versions/2",
			"xRegistry-versionscount:3",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f1-proxy/versions",
		},
		ResBody: `more data`,
	})

	// Update latest with fileURL
	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT file f1-proxy - use fileurl",
		URL:        "/dirs/d1/files/f1-proxy?meta",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
		  "id": "f1-proxy",
		  "fileurl": "http://localhost:8181/EMPTY-URL"
		}`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "id": "f1-proxy",
  "epoch": 3,
  "self": "http://localhost:8181/dirs/d1/files/f1-proxy?meta",
  "latestversionid": "2",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions/2?meta",
  "fileurl": "http://localhost:8181/EMPTY-URL",

  "versionscount": 3,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions"
}
`,
	})

	// Update latest - delete fileurl, notice no "id" either
	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT file f1-proxy - del fileurl",
		URL:        "/dirs/d1/files/f1-proxy?meta",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
		}`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "id": "f1-proxy",
  "epoch": 4,
  "self": "http://localhost:8181/dirs/d1/files/f1-proxy?meta",
  "latestversionid": "2",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions/2?meta",

  "versionscount": 3,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions"
}
`,
	})

	// Update latest - set 'file' and 'fileurl' - error
	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT file f1-proxy - dup files",
		URL:        "/dirs/d1/files/f1-proxy?meta",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
		  "file": "hello world",
		  "fileurl": "http://example.com"
		}`,
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `Error processing resource: Only one of file,fileurl,filebase64 can be present at a time
`,
	})

	// Update latest - set 'filebase64' and 'fileurl' - error
	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT file f1-proxy - dup files base64",
		URL:        "/dirs/d1/files/f1-proxy?meta",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
		  "filebase64": "aGVsbG8K",
		  "fileurl": "http://example.com"
		}`,
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `Error processing resource: Only one of file,fileurl,filebase64 can be present at a time
`,
	})

	// Update latest - with 'filebase64'
	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT file f1-proxy - use base64",
		URL:        "/dirs/d1/files/f1-proxy?meta",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
		  "filebase64": "aGVsbG8K"
		}`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "id": "f1-proxy",
  "epoch": 5,
  "self": "http://localhost:8181/dirs/d1/files/f1-proxy?meta",
  "latestversionid": "2",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions/2?meta",

  "versionscount": 3,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions"
}
`,
	})

	// Get latest
	xCheckHTTP(t, &HTTPTest{
		Name:        "GET file f1-proxy - use base64",
		URL:         "/dirs/d1/files/f1-proxy",
		Method:      "GET",
		ReqHeaders:  []string{},
		ReqBody:     "",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f1-proxy",
			"xRegistry-epoch: 5",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1-proxy",
			"xRegistry-latestversionid: 2",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/d1/files/f1-proxy/versions/2",
			"xRegistry-versionscount: 3",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f1-proxy/versions",
		},
		ResBody: `hello
`,
	})

	// test the variants of how to store a resource

	xCheckHTTP(t, &HTTPTest{
		Name:        "PUT files/f2/versions/v1 - resource",
		URL:         "/dirs/d1/files/f2/versions/v1",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     "Hello world - v1",
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/f2/versions/v1",
			"Content-Location:http://localhost:8181/dirs/d1/files/f2/versions/v1",
			"xRegistry-id:v1",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f2/versions/v1",
			"xRegistry-latest:true",
		},
		ResBody: "Hello world - v1",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "PUT files/f2/versions/v2 - resourceProxyURL",
		URL:    "/dirs/d1/files/f2/versions/v2",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-fileproxyurl:http://localhost:8181/EMPTY-Proxy",
		},
		ReqBody:     "",
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/f2/versions/v2",
			"Content-Location:http://localhost:8181/dirs/d1/files/f2/versions/v2",
			"xRegistry-id:v2",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f2/versions/v2",
			"xRegistry-latest:true",
		},
		ResBody: "hello-Proxy",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "PUT files/f2/versions/v3 - resourceURL",
		URL:    "/dirs/d1/files/f2/versions/v3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-fileurl:http://localhost:8181/EMPTY-URL",
		},
		ReqBody:     "",
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/f2/versions/v3",
			"Content-Location:http://localhost:8181/dirs/d1/files/f2/versions/v3",
			"xRegistry-id:v3",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f2/versions/v3",
			"xRegistry-latest:true",
			"xRegistry-fileurl:http://localhost:8181/EMPTY-URL",
		},
		ResBody: "",
	})

	// testing of "latest" processing

	// Set up the following:
	// /dirs/d1/files/ff1-proxy/v1 - resource
	//                        /v2 - URL
	//                        /v3 - ProxyURL  <- latest
	// /dirs/d1/files/ff2-url/v1 - resource
	//                      /v2 - ProxyURL
	//                      /v3 - URL  <- latest
	// /dirs/d1/files/ff3-resource/v1 - ProxyURL
	//                           /v2 - URL
	//                           /v3 - resource  <- latest

	// Now create the ff1-proxy variants
	xCheckHTTP(t, &HTTPTest{
		Name:   "POST file ff1-proxy-v1 Resource",
		URL:    "/dirs/d1/files/ff1-proxy?meta",
		Method: "POST",
		ReqBody: `{
		  "id": "v1",
		  "file": "In resource ff1-proxy"
		}`,
		Code: 201,
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v1?meta",
		},
		ResBody: `{
  "id": "v1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v1?meta",
  "latest": true
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "POST file ff1-proxy-v2 URL",
		URL:    "/dirs/d1/files/ff1-proxy?meta",
		Method: "POST",
		ReqBody: `{
		  "id": "v2",
		  "fileurl": "http://localhost:8181/EMPTY-URL"
		}`,
		Code: 201,
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v2?meta",
		},
		ResBody: `{
  "id": "v2",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v2?meta",
  "latest": true,
  "fileurl": "http://localhost:8181/EMPTY-URL"
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "POST file ff1-proxy-v3 ProxyURL",
		URL:    "/dirs/d1/files/ff1-proxy?meta",
		Method: "POST",
		ReqBody: `{
		  "id": "v3",
		  "fileproxyurl": "http://localhost:8181/EMPTY-Proxy"
		}`,
		Code: 201,
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v3?meta",
		},
		ResBody: `{
  "id": "v3",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v3?meta",
  "latest": true
}
`,
	})

	CompareContentMeta(t, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff1-proxy",
		Headers: []string{},
		Body:    "hello-Proxy",
	})
	CompareContentMeta(t, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff1-proxy/versions/v1",
		Headers: []string{},
		Body:    "In resource ff1-proxy",
	})
	CompareContentMeta(t, &Test{
		Code:    303,
		URL:     "dirs/d1/files/ff1-proxy/versions/v2",
		Headers: []string{},
		Body:    "",
	})
	CompareContentMeta(t, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff1-proxy/versions/v3",
		Headers: []string{},
		Body:    "hello-Proxy",
	})

	// Now create the ff2-url variants
	// ///////////////////////////////
	xCheckHTTP(t, &HTTPTest{
		Name:   "POST file ff2-url-v1 resource",
		URL:    "/dirs/d1/files/ff2-url?meta",
		Method: "POST",
		ReqBody: `{
		  "id": "v1",
		  "file": "In resource ff2-url"
		}`,
		Code: 201,
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/ff2-url/versions/v1?meta",
		},
		ResBody: `{
  "id": "v1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/ff2-url/versions/v1?meta",
  "latest": true
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "POST file ff2-url-v2 ProxyURL",
		URL:    "/dirs/d1/files/ff2-url?meta",
		Method: "POST",
		ReqBody: `{
		  "id": "v2",
		  "fileproxyurl": "http://localhost:8181/EMPTY-Proxy"
		}`,
		Code: 201,
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/ff2-url/versions/v2?meta",
		},
		ResBody: `{
  "id": "v2",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/ff2-url/versions/v2?meta",
  "latest": true
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "POST file ff2-url-v2 URL",
		URL:    "/dirs/d1/files/ff2-url?meta",
		Method: "POST",
		ReqBody: `{
		  "id": "v3",
		  "fileurl": "http://localhost:8181/EMPTY-URL"
		}`,
		Code: 201,
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/ff2-url/versions/v3?meta",
		},
		ResBody: `{
  "id": "v3",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/ff2-url/versions/v3?meta",
  "latest": true,
  "fileurl": "http://localhost:8181/EMPTY-URL"
}
`,
	})

	CompareContentMeta(t, &Test{
		Code:    303,
		URL:     "dirs/d1/files/ff2-url",
		Headers: []string{},
		Body:    "",
	})
	CompareContentMeta(t, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff2-url/versions/v1",
		Headers: []string{},
		Body:    "In resource ff2-url",
	})
	CompareContentMeta(t, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff2-url/versions/v2",
		Headers: []string{},
		Body:    "hello-Proxy",
	})
	CompareContentMeta(t, &Test{
		Code:    303,
		URL:     "dirs/d1/files/ff2-url/versions/v3",
		Headers: []string{},
		Body:    "",
	})

	// Now create the ff3-resource variants
	// ///////////////////////////////
	xCheckHTTP(t, &HTTPTest{
		Name:   "POST file ff3-resource-v1 ProxyURL",
		URL:    "/dirs/d1/files/ff3-resource?meta",
		Method: "POST",
		ReqBody: `{
		  "id": "v1",
		  "fileproxyurl": "http://localhost:8181/EMPTY-Proxy"
		}`,
		Code: 201,
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/ff3-resource/versions/v1?meta",
		},
		ResBody: `{
  "id": "v1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/ff3-resource/versions/v1?meta",
  "latest": true
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "POST file ff3-resource-v2 URL",
		URL:    "/dirs/d1/files/ff3-resource?meta",
		Method: "POST",
		ReqBody: `{
		  "id": "v2",
		  "fileurl": "http://localhost:8181/EMPTY-URL"
		}`,
		Code: 201,
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/ff3-resource/versions/v2?meta",
		},
		ResBody: `{
  "id": "v2",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/ff3-resource/versions/v2?meta",
  "latest": true,
  "fileurl": "http://localhost:8181/EMPTY-URL"
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "POST file ff3-resource-v3 resource",
		URL:    "/dirs/d1/files/ff3-resource?meta",
		Method: "POST",
		ReqBody: `{
		  "id": "v3",
		  "file": "In resource ff3-resource"
		}`,
		Code: 201,
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/ff3-resource/versions/v3?meta",
		},
		ResBody: `{
  "id": "v3",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/ff3-resource/versions/v3?meta",
  "latest": true
}
`,
	})

	CompareContentMeta(t, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff3-resource",
		Headers: []string{},
		Body:    "In resource ff3-resource",
	})
	CompareContentMeta(t, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff3-resource/versions/v1",
		Headers: []string{},
		Body:    "hello-Proxy",
	})
	CompareContentMeta(t, &Test{
		Code:    303,
		URL:     "dirs/d1/files/ff3-resource/versions/v2",
		Headers: []string{},
		Body:    "",
	})
	CompareContentMeta(t, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff3-resource/versions/v3",
		Headers: []string{},
		Body:    "In resource ff3-resource",
	})

	// Now do some testing

	xCheckHTTP(t, &HTTPTest{
		Name:        "GET resource - latest - ff1",
		URL:         "/dirs/d1/files/ff1-proxy",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: ff1-proxy",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/ff1-proxy",
			"xRegistry-latestversionid: v3",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v3",
			"xRegistry-versionscount: 3",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/ff1-proxy/versions",
			"Content-Location: http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v3",
		},
		ResBody: "hello-Proxy",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:        "GET resource - latest - ff1/v3",
		URL:         "/dirs/d1/files/ff1-proxy/versions/v3",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: v3",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v3",
			"xRegistry-latest: true",
		},
		ResBody: "hello-Proxy",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:        "GET resource - latest - ff1/v2",
		URL:         "/dirs/d1/files/ff1-proxy/versions/v2",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        303,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: v2",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v2",
			"xRegistry-fileurl: http://localhost:8181/EMPTY-URL",
			"Location: http://localhost:8181/EMPTY-URL",
		},
		ResBody: "",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:        "GET resource - latest - ff2",
		URL:         "/dirs/d1/files/ff2-url",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        303,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: ff2-url",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/ff2-url",
			"xRegistry-latestversionid: v3",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/d1/files/ff2-url/versions/v3",
			"xRegistry-fileurl: http://localhost:8181/EMPTY-URL",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/ff2-url/versions",
			"xRegistry-versionscount: 3",
			"Location: http://localhost:8181/EMPTY-URL",
		},
		ResBody: "",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:        "GET resource - latest - ff3",
		URL:         "/dirs/d1/files/ff3-resource",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: ff3-resource",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/ff3-resource",
			"xRegistry-latestversionid: v3",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/d1/files/ff3-resource/versions/v3",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/ff3-resource/versions",
			"xRegistry-versionscount: 3",
		},
		ResBody: "In resource ff3-resource",
	})
}
