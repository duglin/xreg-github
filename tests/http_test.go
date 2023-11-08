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
	xNoErr(t, err)
	xCheck(t, res.StatusCode == test.Code,
		fmt.Sprintf("Expected status %d, got %d", test.Code, res.StatusCode))

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
		t.Errorf("Extra header(%s)", k)
	}

	resBody, _ := io.ReadAll(res.Body)
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
          "versionId": true,
          "latest": true,
          "hasDocument": true
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
          "versionId": true,
          "latest": true,
          "hasDocument": false
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
          "versionId": true,
          "latest": true,
          "hasDocument": false
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

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

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
		Name:       "Create group - empty",
		URL:        "/dirs",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    "",
		Code:       201,
		ResHeaders: []string{"Content-Type:application/json"},
		BodyMasks:  []string{"id", "dirs/[a-zA-Z0-9]*|dirs/xxx"},
		ResBody: `{
  "id": "xxx",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/xxx",

  "filesCount": 0,
  "filesUrl": "http://localhost:8181/dirs/xxx/files"
}
`,
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "Create group - {}",
		URL:        "/dirs",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    "{}",
		Code:       201,
		ResHeaders: []string{"Content-Type:application/json"},
		BodyMasks:  []string{"id", "dirs/[a-zA-Z0-9]*|dirs/xxx"},
		ResBody: `{
  "id": "xxx",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/xxx",

  "filesCount": 0,
  "filesUrl": "http://localhost:8181/dirs/xxx/files"
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
  "epoch": 5,
  "description":"desc",
  "documentation":"docs-url",
  "labels": {
    "label1": "value1",
    "label2": 5,
    "label3": 123.456,
    "label4": "",
    "label5": null
  },
  "format":"my group"
}`,
		Code:       201,
		ResHeaders: []string{"Content-Type:application/json"},
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

  "filesCount": 0,
  "filesUrl": "http://localhost:8181/dirs/dir1/files"
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
  "format":"myformat/1"
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

  "filesCount": 0,
  "filesUrl": "http://localhost:8181/dirs/dir1/files"
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
		ResBody:    "Error processing group(dir1): Incoming epoch(10) doesn't match existing epoch(2)\n",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:       "PUT group - update - err id",
		URL:        "/dirs/dir1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{ "id":"dir2" }`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "Error processing group(dir1): Metadata id(dir2) doesn't match ID in URL(dir1)\n",
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
  "epoch": 3,
  "self": "http://localhost:8181/dirs/dir1",

  "filesCount": 0,
  "filesUrl": "http://localhost:8181/dirs/dir1/files"
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
  "format":"myformat/1"
}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "Error processing group(dir2): Metadata id(dir3) doesn't match ID in URL(dir2)\n",
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
			"Location: http://localhost:8181/dirs/dir1/files/xxx",
			"xRegistry-id: xxx",
			"xRegistry-epoch: 1",
			"xRegistry-latestVersionId: 1",
			"xRegistry-latestVersionUrl: http://localhost:8181/dirs/dir1/files/xxx/versions/1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/xxx",
			"xRegistry-versionsCount: 1",
			"xRegistry-versionsUrl: http://localhost:8181/dirs/dir1/files/xxx/versions",
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
			"Location: http://localhost:8181/dirs/dir1/files/xxx",
			"xRegistry-id: xxx",
			"xRegistry-epoch: 1",
			"xRegistry-latestVersionId: 1",
			"xRegistry-latestVersionUrl: http://localhost:8181/dirs/dir1/files/xxx/versions/1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/xxx",
			"xRegistry-versionsCount: 1",
			"xRegistry-versionsUrl: http://localhost:8181/dirs/dir1/files/xxx/versions",
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
			"Location: http://localhost:8181/dirs/dir1/files/f1",
			"xRegistry-id: f1",
			"xRegistry-epoch: 1",
			"xRegistry-latestVersionId: 1",
			"xRegistry-latestVersionUrl: http://localhost:8181/dirs/dir1/files/f1/versions/1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f1",
			"xRegistry-versionsCount: 1",
			"xRegistry-versionsUrl: http://localhost:8181/dirs/dir1/files/f1/versions",
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
			"xRegistry-format: ce/1.0",
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
			"xRegistry-latestVersionId: 1",
			"xRegistry-latestVersionUrl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-format: ce/1.0",
			"xRegistry-versionsCount: 1",
			"xRegistry-versionsUrl: http://localhost:8181/dirs/dir1/files/f3/versions",
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
			"xRegistry-latestVersionId: 1",
			"xRegistry-latestVersionUrl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-format: ce/1.0",
			"xRegistry-versionsCount: 1",
			"xRegistry-versionsUrl: http://localhost:8181/dirs/dir1/files/f3/versions",
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
			"xRegistry-fileUrl: http://example.com",
		},
		ReqBody:     "",
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f4",
			"xRegistry-name: my doc",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f4",
			"xRegistry-latestVersionId: 1",
			"xRegistry-latestVersionUrl: http://localhost:8181/dirs/dir1/files/f4/versions/1",
			"xRegistry-name: my doc",
			"xRegistry-fileUrl: http://example.com",
			"xRegistry-versionsCount: 1",
			"xRegistry-versionsUrl: http://localhost:8181/dirs/dir1/files/f4/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f4/versions/1",
			"Location: http://example.com",
		},
		ResBody: "",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "PUT resources - update latest - URL",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-fileUrl: http://example.com",
		},
		ReqBody:     "",
		Code:        303,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f3",
			"xRegistry-name: my doc",
			"xRegistry-epoch: 3",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-latestVersionId: 1",
			"xRegistry-latestVersionUrl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-format: ce/1.0",
			"xRegistry-fileUrl: http://example.com",
			"xRegistry-versionsCount: 1",
			"xRegistry-versionsUrl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"Location: http://example.com",
		},
		ResBody: "",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "PUT resources - update latest - URL + body - error",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-fileUrl: example.com",
		},
		ReqBody:     "My cool doc - v2",
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody:     "'xRegistry-fileUrl' isn't allowed if there's a body\n",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "PUT resources - update latest - URL - null",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-fileUrl: null",
		},
		ReqBody:     "",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f3",
			"xRegistry-name: my doc",
			"xRegistry-epoch: 4",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-latestVersionId: 1",
			"xRegistry-latestVersionUrl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-format: ce/1.0",
			"xRegistry-versionsCount: 1",
			"xRegistry-versionsUrl: http://localhost:8181/dirs/dir1/files/f3/versions",
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
			"xRegistry-latestVersionId: 1",
			"xRegistry-latestVersionUrl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-format: ce/1.0",
			"xRegistry-versionsCount: 1",
			"xRegistry-versionsUrl: http://localhost:8181/dirs/dir1/files/f3/versions",
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
			"xRegistry-latestVersionId: 1",
			"xRegistry-latestVersionUrl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-format: ce/1.0",
			"xRegistry-versionsCount: 1",
			"xRegistry-versionsUrl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
		},
		ResBody: "another body",
	})

	xCheckHTTP(t, &HTTPTest{
		Name:   "PUT resources - update latest - w/body - clear 1 label",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-labels-l1: null",
		},
		ReqBody:     "another body",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f3",
			"xRegistry-name: my doc",
			"xRegistry-epoch: 7",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-latestVersionId: 1",
			"xRegistry-latestVersionUrl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-documentation: my doc url",
			"xRegistry-format: ce/1.0",
			"xRegistry-versionsCount: 1",
			"xRegistry-versionsUrl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
		},
		ResBody: "another body",
	})

}
