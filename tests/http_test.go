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
)

type HTTPTest struct {
	Name       string
	URL        string
	Method     string
	ReqHeaders []string // name:value
	ReqBody    string

	Code       int
	ResHeaders []string // name:value
	Masks      []string
	ResBody    string
}

func xCheckHTTP(t *testing.T, test *HTTPTest) {
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
	xNoErr(t, err)
	xCheck(t, res.StatusCode == test.Code,
		fmt.Sprintf("Expected status %d, got %d", test.Code, res.StatusCode))

	for _, header := range test.ResHeaders {
		name, value, _ := strings.Cut(header, ":")
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		xCheckEqual(t, "Header:"+name+"\n", res.Header.Get(name), value)
	}

	resBody, _ := io.ReadAll(res.Body)
	testBody := test.ResBody

	fmt.Printf("Res before:\n%s\n\n", string(resBody))
	for _, mask := range test.Masks {
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

		fmt.Printf("Res:\n%s\n\n", string(resBody))
	}

	xCheckEqual(t, "Test: "+test.Name+"\nBody:\n",
		string(resBody), testBody)
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
  "specVersion": "0.5",
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
  "specVersion": "0.5",
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
		ReqBody: `{
  "schema": "model.schema"
}`,

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "schema": "model.schema"
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "Create model - full",
		URL:        "/model",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "groups": [
    {
      "plural": "dirs",
      "singular": "dir",
      "resources": [
        {
          "plural": "files",
          "singular": "file",
          "versions": 1,
          "versionId": true,
          "latest": true
        }
      ]
    }
  ]
}`,

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "groups": [
    {
      "plural": "dirs",
      "singular": "dir",
      "resources": [
        {
          "plural": "files",
          "singular": "file",
          "versions": 1,
          "versionId": true,
          "latest": true
        }
      ]
    }
  ]
}
`,
	})
}

func TestHTTPRegistry(t *testing.T) {
	reg := NewRegistry("TestHTTPRegistry")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

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
  "specVersion": "0.5",
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
  "specVersion": "0.5",
  "id": "TestHTTPRegistry",
  "epoch": 3,
  "self": "http://localhost:8181/"
}
`,
	})
}

func TestHTTPGroups(t *testing.T) {
	reg := NewRegistry("TestHTTPGroups")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir", "")
	gm.AddResourceModel("files", "file", 0, true, true)

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT groups",
		URL:        "/dirs",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "",
		Code:       405,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "PUT not allowed on collections\n",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "Create group - empty",
		URL:        "/dirs",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    "",
		Code:       201,
		ResHeaders: []string{"Content-Type:application/json"},
		Masks:      []string{"id", "dirs/[a-zA-Z0-9]*|dirs/xxx"},
		ResBody: `{
  "id": "xxx",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/xxx",

  "filesCount": 0,
  "filesUrl": "http://localhost:8181/dirs/xxx/files"
}
`,
	})

}
