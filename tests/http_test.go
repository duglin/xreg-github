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
	BodyMasks   []string // "PROPNAME" or "SEARCH|REPLACE"
	ResBody     string
}

func xHTTP(t *testing.T, reg *registry.Registry, verb, url, reqBody string, code int, resBody string) {
	t.Helper()
	xCheckHTTP(t, reg, &HTTPTest{
		URL:     url,
		Method:  verb,
		ReqBody: reqBody,
		Code:    code,
		ResBody: resBody,
	})
}

func xCheckHTTP(t *testing.T, reg *registry.Registry, test *HTTPTest) {
	t.Helper()
	xNoErr(t, reg.Commit())

	// t.Logf("Test: %s", test.Name)
	// t.Logf(">> %s %s  (%s)", test.Method, test.URL, registry.GetStack()[1])
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

	resBody := []byte{}
	res, err := client.Do(req)
	if res != nil {
		resBody, _ = io.ReadAll(res.Body)
	}

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
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "?html",
		URL:        "?html",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    "",

		Code:       200,
		ResHeaders: []string{"Content-Type:text/html"},
		ResBody: `<pre>
{
  "specversion": "` + registry.SPECVERSION + `",
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
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "?model",
		URL:        "?model",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    "",

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestHTTPModel",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "model": {
    "schemas": [
      "` + registry.XREGSCHEMA + "/" + registry.SPECVERSION + `"
    ],
    "attributes": {
      "specversion": {
        "name": "specversion",
        "type": "string",
        "readonly": true,
        "serverrequired": true
      },
      "id": {
        "name": "id",
        "type": "string",
        "serverrequired": true
      },
      "name": {
        "name": "name",
        "type": "string"
      },
      "epoch": {
        "name": "epoch",
        "type": "uinteger",
        "serverrequired": true
      },
      "self": {
        "name": "self",
        "type": "url",
        "readonly": true,
        "serverrequired": true
      },
      "description": {
        "name": "description",
        "type": "string"
      },
      "documentation": {
        "name": "documentation",
        "type": "url"
      },
      "labels": {
        "name": "labels",
        "type": "map",
        "item": {
          "type": "string"
        }
      },
      "createdby": {
        "name": "createdby",
        "type": "string",
        "readonly": true
      },
      "createdon": {
        "name": "createdon",
        "type": "timestamp",
        "readonly": true
      },
      "modifiedby": {
        "name": "modifiedby",
        "type": "string",
        "readonly": true
      },
      "modifiedon": {
        "name": "modifiedon",
        "type": "timestamp",
        "readonly": true
      }
    }
  }
}
`,
	})

	// Just model, no reg content
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "/model",
		URL:        "/model",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    "",

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "schemas": [
    "` + registry.XREGSCHEMA + "/" + registry.SPECVERSION + `"
  ],
  "attributes": {
    "specversion": {
      "name": "specversion",
      "type": "string",
      "readonly": true,
      "serverrequired": true
    },
    "id": {
      "name": "id",
      "type": "string",
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
    },
    "description": {
      "name": "description",
      "type": "string"
    },
    "documentation": {
      "name": "documentation",
      "type": "url"
    },
    "labels": {
      "name": "labels",
      "type": "map",
      "item": {
        "type": "string"
      }
    },
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
    }
  }
}
`,
	})

	// Create model tests
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "Create empty model",
		URL:        "/model",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{}`,

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "schemas": [
    "` + registry.XREGSCHEMA + "/" + registry.SPECVERSION + `"
  ],
  "attributes": {
    "specversion": {
      "name": "specversion",
      "type": "string",
      "readonly": true,
      "serverrequired": true
    },
    "id": {
      "name": "id",
      "type": "string",
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
    },
    "description": {
      "name": "description",
      "type": "string"
    },
    "documentation": {
      "name": "documentation",
      "type": "url"
    },
    "labels": {
      "name": "labels",
      "type": "map",
      "item": {
        "type": "string"
      }
    },
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
    }
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "Create model - just schema",
		URL:        "/model",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{"schemas":["schema1"]}`,

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "schemas": [
    "schema1",
    "` + registry.XREGSCHEMA + "/" + registry.SPECVERSION + `"
  ],
  "attributes": {
    "specversion": {
      "name": "specversion",
      "type": "string",
      "readonly": true,
      "serverrequired": true
    },
    "id": {
      "name": "id",
      "type": "string",
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
    },
    "description": {
      "name": "description",
      "type": "string"
    },
    "documentation": {
      "name": "documentation",
      "type": "url"
    },
    "labels": {
      "name": "labels",
      "type": "map",
      "item": {
        "type": "string"
      }
    },
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
    }
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "Create model - defaults",
		URL:        "/model",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
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
    "` + registry.XREGSCHEMA + "/" + registry.SPECVERSION + `"
  ],
  "attributes": {
    "specversion": {
      "name": "specversion",
      "type": "string",
      "readonly": true,
      "serverrequired": true
    },
    "id": {
      "name": "id",
      "type": "string",
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
    },
    "description": {
      "name": "description",
      "type": "string"
    },
    "documentation": {
      "name": "documentation",
      "type": "url"
    },
    "labels": {
      "name": "labels",
      "type": "map",
      "item": {
        "type": "string"
      }
    },
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
    }
  },
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "attributes": {
        "id": {
          "name": "id",
          "type": "string",
          "serverrequired": true
        },
        "name": {
          "name": "name",
          "type": "string"
        },
        "epoch": {
          "name": "epoch",
          "type": "uinteger",
          "serverrequired": true
        },
        "self": {
          "name": "self",
          "type": "url",
          "readonly": true,
          "serverrequired": true
        },
        "description": {
          "name": "description",
          "type": "string"
        },
        "documentation": {
          "name": "documentation",
          "type": "url"
        },
        "labels": {
          "name": "labels",
          "type": "map",
          "item": {
            "type": "string"
          }
        },
        "origin": {
          "name": "origin",
          "type": "uri"
        },
        "createdby": {
          "name": "createdby",
          "type": "string",
          "readonly": true
        },
        "createdon": {
          "name": "createdon",
          "type": "timestamp",
          "readonly": true
        },
        "modifiedby": {
          "name": "modifiedby",
          "type": "string",
          "readonly": true
        },
        "modifiedon": {
          "name": "modifiedon",
          "type": "timestamp",
          "readonly": true
        }
      },
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "versions": 0,
          "versionid": true,
          "latest": true,
          "hasdocument": true,
          "attributes": {
            "id": {
              "name": "id",
              "type": "string",
              "serverrequired": true
            },
            "name": {
              "name": "name",
              "type": "string"
            },
            "epoch": {
              "name": "epoch",
              "type": "uinteger",
              "serverrequired": true
            },
            "self": {
              "name": "self",
              "type": "url",
              "readonly": true,
              "serverrequired": true
            },
            "latest": {
              "name": "latest",
              "type": "boolean"
            },
            "latestversionid": {
              "name": "latestversionid",
              "type": "string",
              "readonly": true
            },
            "latestversionurl": {
              "name": "latestversionurl",
              "type": "url",
              "readonly": true
            },
            "description": {
              "name": "description",
              "type": "string"
            },
            "documentation": {
              "name": "documentation",
              "type": "url"
            },
            "labels": {
              "name": "labels",
              "type": "map",
              "item": {
                "type": "string"
              }
            },
            "origin": {
              "name": "origin",
              "type": "uri"
            },
            "createdby": {
              "name": "createdby",
              "type": "string",
              "readonly": true
            },
            "createdon": {
              "name": "createdon",
              "type": "timestamp",
              "readonly": true
            },
            "modifiedby": {
              "name": "modifiedby",
              "type": "string",
              "readonly": true
            },
            "modifiedon": {
              "name": "modifiedon",
              "type": "timestamp",
              "readonly": true
            },
            "contenttype": {
              "name": "contenttype",
              "type": "string"
            }
          }
        }
      }
    }
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
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
          "versions": 0,
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
    "schema1",
    "` + registry.XREGSCHEMA + "/" + registry.SPECVERSION + `"
  ],
  "attributes": {
    "specversion": {
      "name": "specversion",
      "type": "string",
      "readonly": true,
      "serverrequired": true
    },
    "id": {
      "name": "id",
      "type": "string",
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
    },
    "description": {
      "name": "description",
      "type": "string"
    },
    "documentation": {
      "name": "documentation",
      "type": "url"
    },
    "labels": {
      "name": "labels",
      "type": "map",
      "item": {
        "type": "string"
      }
    },
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
    }
  },
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "attributes": {
        "id": {
          "name": "id",
          "type": "string",
          "serverrequired": true
        },
        "name": {
          "name": "name",
          "type": "string"
        },
        "epoch": {
          "name": "epoch",
          "type": "uinteger",
          "serverrequired": true
        },
        "self": {
          "name": "self",
          "type": "url",
          "readonly": true,
          "serverrequired": true
        },
        "description": {
          "name": "description",
          "type": "string"
        },
        "documentation": {
          "name": "documentation",
          "type": "url"
        },
        "labels": {
          "name": "labels",
          "type": "map",
          "item": {
            "type": "string"
          }
        },
        "origin": {
          "name": "origin",
          "type": "uri"
        },
        "createdby": {
          "name": "createdby",
          "type": "string",
          "readonly": true
        },
        "createdon": {
          "name": "createdon",
          "type": "timestamp",
          "readonly": true
        },
        "modifiedby": {
          "name": "modifiedby",
          "type": "string",
          "readonly": true
        },
        "modifiedon": {
          "name": "modifiedon",
          "type": "timestamp",
          "readonly": true
        }
      },
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "versions": 0,
          "versionid": true,
          "latest": true,
          "hasdocument": false,
          "attributes": {
            "id": {
              "name": "id",
              "type": "string",
              "serverrequired": true
            },
            "name": {
              "name": "name",
              "type": "string"
            },
            "epoch": {
              "name": "epoch",
              "type": "uinteger",
              "serverrequired": true
            },
            "self": {
              "name": "self",
              "type": "url",
              "readonly": true,
              "serverrequired": true
            },
            "latest": {
              "name": "latest",
              "type": "boolean"
            },
            "latestversionid": {
              "name": "latestversionid",
              "type": "string",
              "readonly": true
            },
            "latestversionurl": {
              "name": "latestversionurl",
              "type": "url",
              "readonly": true
            },
            "description": {
              "name": "description",
              "type": "string"
            },
            "documentation": {
              "name": "documentation",
              "type": "url"
            },
            "labels": {
              "name": "labels",
              "type": "map",
              "item": {
                "type": "string"
              }
            },
            "origin": {
              "name": "origin",
              "type": "uri"
            },
            "createdby": {
              "name": "createdby",
              "type": "string",
              "readonly": true
            },
            "createdon": {
              "name": "createdon",
              "type": "timestamp",
              "readonly": true
            },
            "modifiedby": {
              "name": "modifiedby",
              "type": "string",
              "readonly": true
            },
            "modifiedon": {
              "name": "modifiedon",
              "type": "timestamp",
              "readonly": true
            },
            "contenttype": {
              "name": "contenttype",
              "type": "string"
            }
          }
        }
      }
    }
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "Modify description",
		URL:        "/model",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "attributes": {
    "description": {
      "name": "description",
      "type": "string",
      "enum": [ "one", "two" ]
    }
  },
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
    "` + registry.XREGSCHEMA + "/" + registry.SPECVERSION + `"
  ],
  "attributes": {
    "specversion": {
      "name": "specversion",
      "type": "string",
      "readonly": true,
      "serverrequired": true
    },
    "id": {
      "name": "id",
      "type": "string",
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
    },
    "description": {
      "name": "description",
      "type": "string",
      "enum": [
        "one",
        "two"
      ]
    },
    "documentation": {
      "name": "documentation",
      "type": "url"
    },
    "labels": {
      "name": "labels",
      "type": "map",
      "item": {
        "type": "string"
      }
    },
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
    }
  },
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "attributes": {
        "id": {
          "name": "id",
          "type": "string",
          "serverrequired": true
        },
        "name": {
          "name": "name",
          "type": "string"
        },
        "epoch": {
          "name": "epoch",
          "type": "uinteger",
          "serverrequired": true
        },
        "self": {
          "name": "self",
          "type": "url",
          "readonly": true,
          "serverrequired": true
        },
        "description": {
          "name": "description",
          "type": "string"
        },
        "documentation": {
          "name": "documentation",
          "type": "url"
        },
        "labels": {
          "name": "labels",
          "type": "map",
          "item": {
            "type": "string"
          }
        },
        "origin": {
          "name": "origin",
          "type": "uri"
        },
        "createdby": {
          "name": "createdby",
          "type": "string",
          "readonly": true
        },
        "createdon": {
          "name": "createdon",
          "type": "timestamp",
          "readonly": true
        },
        "modifiedby": {
          "name": "modifiedby",
          "type": "string",
          "readonly": true
        },
        "modifiedon": {
          "name": "modifiedon",
          "type": "timestamp",
          "readonly": true
        }
      },
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "versions": 0,
          "versionid": true,
          "latest": true,
          "hasdocument": true,
          "attributes": {
            "id": {
              "name": "id",
              "type": "string",
              "serverrequired": true
            },
            "name": {
              "name": "name",
              "type": "string"
            },
            "epoch": {
              "name": "epoch",
              "type": "uinteger",
              "serverrequired": true
            },
            "self": {
              "name": "self",
              "type": "url",
              "readonly": true,
              "serverrequired": true
            },
            "latest": {
              "name": "latest",
              "type": "boolean"
            },
            "latestversionid": {
              "name": "latestversionid",
              "type": "string",
              "readonly": true
            },
            "latestversionurl": {
              "name": "latestversionurl",
              "type": "url",
              "readonly": true
            },
            "description": {
              "name": "description",
              "type": "string"
            },
            "documentation": {
              "name": "documentation",
              "type": "url"
            },
            "labels": {
              "name": "labels",
              "type": "map",
              "item": {
                "type": "string"
              }
            },
            "origin": {
              "name": "origin",
              "type": "uri"
            },
            "createdby": {
              "name": "createdby",
              "type": "string",
              "readonly": true
            },
            "createdon": {
              "name": "createdon",
              "type": "timestamp",
              "readonly": true
            },
            "modifiedby": {
              "name": "modifiedby",
              "type": "string",
              "readonly": true
            },
            "modifiedon": {
              "name": "modifiedon",
              "type": "timestamp",
              "readonly": true
            },
            "contenttype": {
              "name": "contenttype",
              "type": "string"
            }
          }
        }
      }
    }
  }
}
`,
	})

	xHTTP(t, reg, "PUT", "/", `{"description": "testing"}`, 400,
		`Error processing registry: Attribute "description"(testing) must be one of the enum values: one, two`+"\n")

	xHTTP(t, reg, "PUT", "/", `{}`, 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestHTTPModel",
  "epoch": 2,
  "self": "http://localhost:8181/",

  "dirscount": 0,
  "dirsurl": "http://localhost:8181/dirs"
}
`)

	xHTTP(t, reg, "PUT", "/", `{"description": "two"}`, 200, `{
  "specversion": "0.5",
  "id": "TestHTTPModel",
  "epoch": 3,
  "self": "http://localhost:8181/",
  "description": "two",

  "dirscount": 0,
  "dirsurl": "http://localhost:8181/dirs"
}
`)
}

func TestHTTPRegistry(t *testing.T) {
	reg := NewRegistry("TestHTTPRegistry")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	_, err := reg.Model.AddAttr("myany", registry.ANY)
	xCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("mybool", registry.BOOLEAN)
	xCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("mydec", registry.DECIMAL)
	xCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("myint", registry.INTEGER)
	xCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("mystr", registry.STRING)
	xCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("mytime", registry.TIMESTAMP)
	xCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("myuint", registry.UINTEGER)
	xCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("myuri", registry.URI)
	xCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("myuriref", registry.URI_REFERENCE)
	xCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("myuritemplate", registry.URI_TEMPLATE)
	xCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("myurl", registry.URL)
	xCheckErr(t, err, "")

	attr, err := reg.Model.AddAttrObj("myobj1")
	xCheckErr(t, err, "")
	_, err = attr.AddAttr("mystr1", registry.STRING)
	xCheckErr(t, err, "")
	_, err = attr.AddAttr("myint1", registry.INTEGER)
	xCheckErr(t, err, "")
	_, err = attr.AddAttr("*", registry.ANY)
	xCheckErr(t, err, "")

	attr, _ = reg.Model.AddAttrObj("myobj2")
	attr.AddAttr("mystr2", registry.STRING)
	obj2, err := attr.AddAttrObj("myobj2_1")
	xCheckErr(t, err, "")
	_, err = obj2.AddAttr("*", registry.INTEGER)
	xCheckErr(t, err, "")

	item := registry.NewItemType(registry.ANY)
	attr, err = reg.Model.AddAttrArray("myarrayany", item)
	xCheckErr(t, err, "")
	attr, err = reg.Model.AddAttrMap("mymapany", item)
	xCheckErr(t, err, "")

	item = registry.NewItemType(registry.UINTEGER)
	attr, err = reg.Model.AddAttrArray("myarrayuint", item)
	xCheckErr(t, err, "")
	attr, err = reg.Model.AddAttrMap("mymapuint", item)
	xCheckErr(t, err, "")

	item = registry.NewItemObject()
	attr, err = reg.Model.AddAttrArray("myarrayemptyobj", item)
	xCheckErr(t, err, "")

	item = registry.NewItemObject()
	item.AddAttr("mapobj_int", registry.INTEGER)
	attr, err = reg.Model.AddAttrMap("mymapobj", item)
	xCheckErr(t, err, "")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST reg",
		URL:        "/",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    "",
		Code:       405,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "POST not allowed on the root of the registry\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - empty string id",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "{ \"id\": \"\" }",
		Code:       400,
		ResHeaders: []string{},
		ResBody:    "Error processing registry: ID can't be an empty string\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - empty",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "",
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestHTTPRegistry",
  "epoch": 2,
  "self": "http://localhost:8181/"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - empty json",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "{}",
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestHTTPRegistry",
  "epoch": 3,
  "self": "http://localhost:8181/"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
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
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestHTTPRegistry",
  "epoch": 4,
  "self": "http://localhost:8181/"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - bad epoch",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "epoch":33
}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "Error processing registry: Attribute \"epoch\"(33) doesn't match existing value (4)\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - full good",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "specversion": "` + registry.SPECVERSION + `",
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
  "specversion": "` + registry.SPECVERSION + `",
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

	xCheckHTTP(t, reg, &HTTPTest{
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
		ResBody: `Error processing registry: Attribute "mymapobj.mapobj_int" must be a map[string] or object
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - full empties",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "specversion": "` + registry.SPECVERSION + `",
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
  "specversion": "` + registry.SPECVERSION + `",
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
			response: `Attribute "epoch"(123) doesn't match existing value (6)`},
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
			response: `Attribute "myobj1" must be a map[string] or object`},
		{request: `{"myobj1": [ 123 ] }`,
			response: `Attribute "myobj1" must be a map[string] or object`},
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
			response: `Attribute "myarrayemptyobj[0]" must be a map[string] or object`},
		{request: `{"mymapobj": { "asd" : { "mapobj_int" : true } } }`,
			response: `Attribute "mymapobj.asd.mapobj_int" must be an integer`},
		{request: `{"mymapobj": { "asd" : { "qwe" : true } } }`,
			response: `Invalid extension(s) in "mymapobj.asd": qwe`},
		{request: `{"mymapobj": [ true ]}`,
			response: `Attribute "mymapobj" must be a map`},
	}

	for _, test := range typeTests {
		xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - bad self - ignored",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "self": 123
}`,
		Code:       400,
		ResHeaders: []string{"application/json"},
		ResBody: `Error processing registry: Attribute "self" must be a url
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestHTTPRegistry",
  "epoch": 7,
  "self": "http://localhost:8181/",
  "documentation": "docs"
}
`})

	xCheckHTTP(t, reg, &HTTPTest{
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
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestHTTPRegistry",
  "epoch": 8,
  "self": "http://localhost:8181/"
}
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - swap any - 1",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "myany": 5.5,
  "mymapany": {
    "any1": {
	  "foo": "bar"
	}
  }
}`,
		Code:       200,
		ResHeaders: []string{"application/json"},
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestHTTPRegistry",
  "epoch": 9,
  "self": "http://localhost:8181/",
  "myany": 5.5,
  "mymapany": {
    "any1": {
      "foo": "bar"
    }
  }
}
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - swap any - 2",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "myany": "foo",
  "mymapany": {
    "any1": 2.3
  }
}`,
		Code:       200,
		ResHeaders: []string{"application/json"},
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestHTTPRegistry",
  "epoch": 10,
  "self": "http://localhost:8181/",
  "myany": "foo",
  "mymapany": {
    "any1": 2.3
  }
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
	attr.AddAttr("foo", registry.STRING)
	attr.AddAttr("*", registry.ANY)

	item := registry.NewItemType(registry.ANY)
	attr, _ = gm.AddAttrArray("myarray", item)
	attr, _ = gm.AddAttrMap("mymap", item)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT groups - fail",
		URL:        "/dirs",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "",
		Code:       405,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "PUT not allowed on collections\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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
		ResBody:    "Error processing group: Attribute \"epoch\"(10) doesn't match existing value (3)\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT group - update - err id",
		URL:        "/dirs/dir1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{ "id":"dir2" }`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "Error processing group: Can't change the ID of an entity(dir1->dir2)\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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
		ResBody:    "Error processing group(dir2): Can't change the ID of an entity(dir2->dir3)\n",
	})

}

func TestHTTPResourcesHeaders(t *testing.T) {
	reg := NewRegistry("TestHTTPResourcesHeaders")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)
	reg.AddGroup("dirs", "dir1")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources - fail",
		URL:        "/dirs/dir1/files",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "",
		Code:       405,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "PUT not allowed on collections\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - w/doc - new content-type",
		URL:    "/dirs/dir1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"Content-Type: my/format",
		},
		ReqBody: "My cool doc - new",
		Code:    200,
		ResHeaders: []string{
			"Content-Type: my/format",
			"xRegistry-id: f1",
			"xRegistry-epoch: 2",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/dir1/files/f1/versions/1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f1",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f1/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f1/versions/1",
			"Content-Length: 17",
		},
		ResBody: `My cool doc - new`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources - w/doc - revert content-type and body",
		URL:        "/dirs/dir1/files/f1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "My cool doc - new x2",
		Code:       200,
		ResHeaders: []string{
			"Content-Type: text/plain; charset=utf-8",
			"xRegistry-id: f1",
			"xRegistry-epoch: 3",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/dir1/files/f1/versions/1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f1",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f1/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f1/versions/1",
			"Content-Length: 20",
		},
		ResBody: `My cool doc - new x2`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - update latest - w/body - delete+add labels",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-labels: null",
			"xRegistry-labels-foo: foo",
			"xRegistry-labels-foo-bar: l-foo-bar",
			"xRegistry-labels-foo_bar: l-foo_bar",
			"xRegistry-labels-foo.bar: l-foo.bar",
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
			"xRegistry-labels-foo-bar: l-foo-bar",
			"xRegistry-labels-foo_bar: l-foo_bar",
			"xRegistry-labels-foo.bar: l-foo.bar",
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

	xCheckHTTP(t, reg, &HTTPTest{
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
			"xRegistry-labels-foo-bar: l-foo-bar",
			"xRegistry-labels-foo_bar: l-foo_bar",
			"xRegistry-labels-foo.bar: l-foo.bar",
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
	xCheckHTTP(t, reg, &HTTPTest{
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
	xCheckHTTP(t, reg, &HTTPTest{
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

	v, _ := f.AddVersion("v2", true)
	v.Set(NewPP().P("#resourceURL").UI(), "http://localhost:8181/EMPTY-URL")

	v, _ = f.AddVersion("v3", true)
	v.Set(NewPP().P("#resourceProxyURL").UI(), "http://localhost:8181/EMPTY-Proxy")

	// URL
	f, _ = d.AddResource("files", "f2-url", "v1")
	f.Set(NewPP().P("#resource").UI(), "Hello world! v1")

	v, _ = f.AddVersion("v2", true)
	v.Set(NewPP().P("#resourceProxyURL").UI(), "http://localhost:8181/EMPTY-Proxy")

	v, _ = f.AddVersion("v3", true)
	v.Set(NewPP().P("#resourceURL").UI(), "http://localhost:8181/EMPTY-URL")

	// Resource
	f, _ = d.AddResource("files", "f3-resource", "v1")
	f.Set(NewPP().P("#resourceProxyURL").UI(), "http://localhost:8181/EMPTY-Proxy")

	v, _ = f.AddVersion("v2", true)
	v.Set(NewPP().P("#resourceURL").UI(), "http://localhost:8181/EMPTY-URL")

	v, _ = f.AddVersion("v3", true)
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

	xCheckHTTP(t, reg, &HTTPTest{
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
	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1-proxy",
		Body:    "hello-Proxy",
		Headers: nil,
	})

	xCheckHTTP(t, reg, &HTTPTest{
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
	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1-proxy/versions/v3",
		Body:    "hello-Proxy",
		Headers: nil,
	})

	xCheckHTTP(t, reg, &HTTPTest{
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
	CompareContentMeta(t, reg, &Test{
		Code: 303,
		URL:  "dirs/d1/files/f1-proxy/versions/v2",
		Headers: []string{
			"Location: http://localhost:8181/EMPTY-URL",
		},
	})

	xCheckHTTP(t, reg, &HTTPTest{
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
	CompareContentMeta(t, reg, &Test{
		Code: 303,
		URL:  "dirs/d1/files/f2-url",
		Headers: []string{
			"Location: http://localhost:8181/EMPTY-URL",
		},
	})

	xCheckHTTP(t, reg, &HTTPTest{
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
	CompareContentMeta(t, reg, &Test{
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
	xCheckHTTP(t, reg, &HTTPTest{
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
	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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
	xCheckHTTP(t, reg, &HTTPTest{
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
	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "POST file f1-proxy - create 2 - no meta",
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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
	xCheckHTTP(t, reg, &HTTPTest{
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
	xCheckHTTP(t, reg, &HTTPTest{
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
	xCheckHTTP(t, reg, &HTTPTest{
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
	xCheckHTTP(t, reg, &HTTPTest{
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
	xCheckHTTP(t, reg, &HTTPTest{
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
	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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
	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff1-proxy",
		Headers: []string{},
		Body:    "hello-Proxy",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff1-proxy/versions/v1",
		Headers: []string{},
		Body:    "In resource ff1-proxy",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    303,
		URL:     "dirs/d1/files/ff1-proxy/versions/v2",
		Headers: []string{},
		Body:    "",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff1-proxy/versions/v3",
		Headers: []string{},
		Body:    "hello-Proxy",
	})

	// Now create the ff2-url variants
	// ///////////////////////////////
	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	CompareContentMeta(t, reg, &Test{
		Code:    303,
		URL:     "dirs/d1/files/ff2-url",
		Headers: []string{},
		Body:    "",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff2-url/versions/v1",
		Headers: []string{},
		Body:    "In resource ff2-url",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff2-url/versions/v2",
		Headers: []string{},
		Body:    "hello-Proxy",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    303,
		URL:     "dirs/d1/files/ff2-url/versions/v3",
		Headers: []string{},
		Body:    "",
	})

	// Now create the ff3-resource variants
	// ///////////////////////////////
	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff3-resource",
		Headers: []string{},
		Body:    "In resource ff3-resource",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff3-resource/versions/v1",
		Headers: []string{},
		Body:    "hello-Proxy",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    303,
		URL:     "dirs/d1/files/ff3-resource/versions/v2",
		Headers: []string{},
		Body:    "",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff3-resource/versions/v3",
		Headers: []string{},
		Body:    "In resource ff3-resource",
	})

	// Now do some testing

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	xCheckHTTP(t, reg, &HTTPTest{
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

	// Test content-type
	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT files/f5/versions/v1 - content-type",
		URL:    "/dirs/d1/files/f5/versions/v1",
		Method: "PUT",
		ReqHeaders: []string{
			"Content-Type: my/format",
		},
		ReqBody:     "Hello world - v1",
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/f5/versions/v1",
			"Content-Length:16",
			"Content-Type:my/format",
			"Content-Location:http://localhost:8181/dirs/d1/files/f5/versions/v1",
			"xRegistry-id:v1",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f5/versions/v1",
			"xRegistry-latest:true",
		},
		ResBody: "Hello world - v1",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST files/f5 - add version - content-type",
		URL:    "/dirs/d1/files/f5",
		Method: "POST",
		ReqHeaders: []string{
			// Notice no "ID" - so this is also testing "POST Resource w/o id"
			"Content-Type: my/format2",
		},
		ReqBody:     "Hello world - v2",
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/f5/versions/1",
			"Content-Length:16",
			"Content-Type:my/format2",
			"Content-Location:http://localhost:8181/dirs/d1/files/f5/versions/1",
			"xRegistry-id:1",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f5/versions/1",
			"xRegistry-latest:true",
		},
		ResBody: "Hello world - v2",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET files/f5?meta - content-type",
		URL:         "/dirs/d1/files/f5?meta",
		Method:      "GET",
		ReqHeaders:  []string{},
		ReqBody:     "",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "id": "f5",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f5?meta",
  "latestversionid": "1",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f5/versions/1?meta",
  "contenttype": "my/format2",

  "versionscount": 2,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f5/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET files/f5 - content-type",
		URL:         "/dirs/d1/files/f5",
		Method:      "GET",
		ReqHeaders:  []string{},
		ReqBody:     "",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Length:16",
			"Content-Type:my/format2",
			"Content-Location:http://localhost:8181/dirs/d1/files/f5/versions/1",
			"xRegistry-id:f5",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f5",
			"xRegistry-latestversionid:1",
			"xRegistry-latestversionurl:http://localhost:8181/dirs/d1/files/f5/versions/1",
			"xRegistry-versionscount:2",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f5/versions",
		},
		ResBody: "Hello world - v2",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT files/f5/v1?meta - revert content-type",
		URL:        "/dirs/d1/files/f5/versions/v1?meta",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "id": "v1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f5/versions/xxx?meta"
}`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "id": "v1",
  "epoch": 2,
  "self": "http://localhost:8181/dirs/d1/files/f5/versions/v1?meta"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET files/f5?meta - content-type - again",
		URL:         "/dirs/d1/files/f5?meta",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "id": "f5",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f5?meta",
  "latestversionid": "1",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f5/versions/1?meta",
  "contenttype": "my/format2",

  "versionscount": 2,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f5/versions"
}
`,
	})

}

func TestHTTPEnum(t *testing.T) {
	reg := NewRegistry("TestHTTPEnum")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	attr, _ := reg.Model.AddAttribute(&registry.Attribute{
		Name:   "myint",
		Type:   registry.INTEGER,
		Enum:   []any{1, 2, 3},
		Strict: registry.PtrBool(true),
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - baseline",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
}`,
		Code: 200,
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestHTTPEnum",
  "epoch": 2,
  "self": "http://localhost:8181/"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - int valid",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
  "myint": 2
}`,
		Code: 200,
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestHTTPEnum",
  "epoch": 3,
  "self": "http://localhost:8181/",
  "myint": 2
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - int invalid",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
  "myint": 4
}`,
		Code: 400,
		ResBody: "Error processing registry: Attribute \"myint\"(4) must be " +
			"one of the enum values: 1, 2, 3\n",
	})

	attr.Strict = registry.PtrBool(false)
	reg.Model.Save()

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - int valid - no-strict",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
  "myint": 4
}`,
		Code: 200,
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestHTTPEnum",
  "epoch": 4,
  "self": "http://localhost:8181/",
  "myint": 4
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - int valid - no-strict - valid enum",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
  "myint": 1
}`,
		Code: 200,
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestHTTPEnum",
  "epoch": 5,
  "self": "http://localhost:8181/",
  "myint": 1
}
`,
	})

	// TODO test other enum types and test in Groups and Resources
}

func TestHTTPIfValue(t *testing.T) {
	reg := NewRegistry("TestHTTPIfValues")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	_, err := reg.Model.AddAttribute(&registry.Attribute{
		Name: "myint",
		Type: registry.INTEGER,
		IfValues: registry.IfValues{
			"10": &registry.IfValue{
				SiblingAttributes: registry.Attributes{
					"mystr": &registry.Attribute{
						Name: "mystr",
						Type: registry.STRING,
					},
					"myobj": &registry.Attribute{
						Name: "myobj",
						Type: registry.OBJECT,
						Attributes: registry.Attributes{
							"subint": &registry.Attribute{
								Name: "subint",
								Type: registry.INTEGER,
							},
							"subobj": &registry.Attribute{
								Name: "subobj",
								Type: registry.OBJECT,
								Attributes: registry.Attributes{
									"subsubint": &registry.Attribute{
										Name: "subsubint",
										Type: registry.INTEGER,
									},
									"*": &registry.Attribute{
										Name: "*",
										Type: registry.ANY,
									},
								},
							},
						},
					},
				},
			},
			"20": &registry.IfValue{
				SiblingAttributes: registry.Attributes{
					"mystr": &registry.Attribute{
						Name:           "mystr",
						Type:           registry.STRING,
						ClientRequired: true,
						ServerRequired: true,
					},
					"*": &registry.Attribute{
						Name: "*",
						Type: registry.ANY,
					},
				},
			},
		},
	})
	xCheckErr(t, err, "")

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name: "myobj",
		Type: registry.OBJECT,
	})
	// Test empty obj and name conflict with IfValue above
	xCheckErr(t, err,
		`Duplicate attribute name (myobj) at: model.myint.ifvalues.10`)

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name: "myobj2",
		Type: registry.OBJECT,
		Attributes: registry.Attributes{
			"subint1": &registry.Attribute{
				Name: "subint1",
				Type: registry.INTEGER,
				IfValues: registry.IfValues{
					"666": &registry.IfValue{
						SiblingAttributes: registry.Attributes{
							"reqint": &registry.Attribute{
								Name: "reqint",
								Type: registry.INTEGER,
							},
						},
					},
				},
			},
		},
	})
	xCheckErr(t, err, "")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - ifvalue - 1",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint": 10
	   }`,
		Code: 200,
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestHTTPIfValues",
  "epoch": 2,
  "self": "http://localhost:8181/",
  "myint": 10
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - ifvalue - verify ext isn't allowed",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint": 10,
	     "myext": 5.5
	   }`,
		Code:    400,
		ResBody: "Error processing registry: Invalid extension(s): myext\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - ifvalue - required mystr",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint": 20
	   }`,
		Code:    400,
		ResBody: "Error processing registry: Required property \"mystr\" is missing\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - ifvalue - required mystr, allow ext",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint": 20,
	     "mystr": "hello",
	     "myext": 5.5
	   }`,
		Code: 200,
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestHTTPIfValues",
  "epoch": 3,
  "self": "http://localhost:8181/",
  "myext": 5.5,
  "myint": 20,
  "mystr": "hello"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - ifvalue - myext isn't allow any more",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint": 10,
	     "mystr": "hello",
	     "myext": 5.5
	   }`,
		Code:    400,
		ResBody: "Error processing registry: Invalid extension(s): myext\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - ifvalue - 3 levels - valid",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint": 10,
	     "mystr": "hello",
		 "myobj": {
		   "subint": 123,
		   "subobj": {
		     "subsubint": 432
		   }
		 }
	   }`,
		Code: 200,
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestHTTPIfValues",
  "epoch": 4,
  "self": "http://localhost:8181/",
  "myint": 10,
  "myobj": {
    "subint": 123,
    "subobj": {
      "subsubint": 432
    }
  },
  "mystr": "hello"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - ifvalue - 3 levels - valid, unknown 3 level",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint": 10,
	     "mystr": "hello",
		 "myobj": {
		   "subint": 123,
		   "subobj": {
		     "subsubint": 432,
			 "okext": "hello"
		   }
		 }
	   }`,
		Code: 200,
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestHTTPIfValues",
  "epoch": 5,
  "self": "http://localhost:8181/",
  "myint": 10,
  "myobj": {
    "subint": 123,
    "subobj": {
      "okext": "hello",
      "subsubint": 432
    }
  },
  "mystr": "hello"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - ifvalue - down a level - valid",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint": 10,
	     "mystr": "hello",
		 "myobj2": {
		   "subint1": 666,
		   "reqint": 777
		 }
	   }`,
		Code: 200,
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestHTTPIfValues",
  "epoch": 6,
  "self": "http://localhost:8181/",
  "myint": 10,
  "myobj2": {
    "reqint": 777,
    "subint1": 666
  },
  "mystr": "hello"
}
`,
	})

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name: "myint5",
		Type: registry.INTEGER,
		IfValues: registry.IfValues{
			"1": &registry.IfValue{
				SiblingAttributes: registry.Attributes{
					"myint6": &registry.Attribute{
						Name: "myint6",
						Type: registry.INTEGER,
						IfValues: registry.IfValues{
							"2": &registry.IfValue{
								SiblingAttributes: registry.Attributes{
									"myint7": {
										Name:           "myint7",
										Type:           registry.INTEGER,
										ClientRequired: true,
										ServerRequired: true,
									},
								},
							},
						},
					},
				},
			},
		},
	})
	xCheckErr(t, err, "")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - nested ifValues - 1",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint5": 1
	   }`,
		Code: 200,
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestHTTPIfValues",
  "epoch": 7,
  "self": "http://localhost:8181/",
  "myint5": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - nested ifValues - 2",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint5": 1,
	     "myint6": 1
	   }`,
		Code: 200,
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestHTTPIfValues",
  "epoch": 8,
  "self": "http://localhost:8181/",
  "myint5": 1,
  "myint6": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - nested ifValues - 3",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint5": 1,
	     "myint6": 2
	   }`,
		Code: 400,
		ResBody: `Error processing registry: Required property "myint7" is missing
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - nested ifValues - 4",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint5": 1,
	     "myint6": 1,
		 "myint7": 5
	   }`,
		Code: 400,
		ResBody: `Error processing registry: Invalid extension(s): myint7
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - nested ifValues - 5",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint5": 1,
	     "myint6": 2,
		 "myint7": 5
	   }`,
		Code: 200,
		ResBody: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestHTTPIfValues",
  "epoch": 9,
  "self": "http://localhost:8181/",
  "myint5": 1,
  "myint6": 2,
  "myint7": 5
}
`,
	})

}

func TestHTTPResources(t *testing.T) {
	reg := NewRegistry("TestHTTPResources")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)
	rm, err := gm.AddResourceModelSimple("files", "file")
	xNoErr(t, err)

	/*
		_, err := rm.AddAttribute(&registry.Attribute{
			Name: "files",
			Type: registry.INTEGER,
		})
		xCheckErr(t, err, "Attribute name is reserved: files")
	*/

	_, err = rm.AddAttribute(&registry.Attribute{
		Name: "file",
		Type: registry.INTEGER,
	})
	xCheckErr(t, err, "Attribute name is reserved: file")

	_, err = rm.AddAttribute(&registry.Attribute{
		Name: "filebase64",
		Type: registry.INTEGER,
	})
	xCheckErr(t, err, "Attribute name is reserved: filebase64")

	_, err = rm.AddAttribute(&registry.Attribute{
		Name: "fileproxyurl",
		Type: registry.INTEGER,
	})
	xCheckErr(t, err, "Attribute name is reserved: fileproxyurl")

	_, err = rm.AddAttribute(&registry.Attribute{
		Name: "mystring",
		Type: registry.STRING,
		IfValues: registry.IfValues{
			"foo": &registry.IfValue{
				SiblingAttributes: registry.Attributes{
					"file": &registry.Attribute{
						Name:     "file",
						Type:     registry.INTEGER,
						IfValues: registry.IfValues{},
					},
				},
			},
		},
	})
	xCheckErr(t, err, "Duplicate attribute name (file) at: resources.files.mystring.ifvalues.foo")
	gm = reg.Model.Groups["dirs"]
	rm = gm.Resources["files"]

	_, err = rm.AddAttribute(&registry.Attribute{
		Name: "mystring",
		Type: registry.STRING,
		IfValues: registry.IfValues{
			"foo": &registry.IfValue{
				SiblingAttributes: registry.Attributes{
					"xxx": &registry.Attribute{
						Name: "xxx",
						Type: registry.INTEGER,
						IfValues: registry.IfValues{
							"5": &registry.IfValue{
								SiblingAttributes: registry.Attributes{
									"xxx": &registry.Attribute{
										Name: "xxx",
										Type: registry.STRING,
									},
								},
							},
						},
					},
				},
			},
		},
	})
	xCheckErr(t, err, "Duplicate attribute name (xxx) at: resources.files.mystring.ifvalues.foo.xxx.ifvalues.5")
	gm = reg.Model.Groups["dirs"]
	rm = gm.Resources["files"]

	_, err = rm.AddAttribute(&registry.Attribute{
		Name: "mystring",
		Type: registry.STRING,
		IfValues: registry.IfValues{
			"foo": &registry.IfValue{
				SiblingAttributes: registry.Attributes{
					"xxx": &registry.Attribute{
						Name: "xxx",
						Type: registry.INTEGER,
						IfValues: registry.IfValues{
							"5": &registry.IfValue{
								SiblingAttributes: registry.Attributes{
									"file": &registry.Attribute{
										Name: "file",
										Type: registry.STRING,
									},
								},
							},
						},
					},
				},
			},
		},
	})
	xCheckErr(t, err, "Duplicate attribute name (file) at: resources.files.mystring.ifvalues.foo.xxx.ifvalues.5")
	gm = reg.Model.Groups["dirs"]
	rm = gm.Resources["files"]

	// "file" is ok this time because HasDocument=false
	rm.HasDocument = false
	xNoErr(t, reg.Model.Save())
	_, err = rm.AddAttribute(&registry.Attribute{
		Name: "mystring",
		Type: registry.STRING,
		IfValues: registry.IfValues{
			"foo": &registry.IfValue{
				SiblingAttributes: registry.Attributes{
					"file": &registry.Attribute{
						Name:     "file",
						Type:     registry.INTEGER,
						IfValues: registry.IfValues{},
					},
				},
			},
		},
	})
	xNoErr(t, err)
	gm = reg.Model.Groups["dirs"]
	rm = gm.Resources["files"]
}

func TestHTTPNonStrings(t *testing.T) {
	reg := NewRegistry("TestHTTPNonStrings")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true /* L */, true, true)

	// rm.AddAttr("myint", registry.INTEGER)
	attr, _ := rm.AddAttr("myint", registry.INTEGER)
	attr.IfValues = registry.IfValues{
		"-5": &registry.IfValue{
			SiblingAttributes: registry.Attributes{
				"ifext": {
					Name: "ifext",
					Type: registry.INTEGER,
				},
			},
		},
	}

	rm.AddAttr("mydec", registry.DECIMAL)
	rm.AddAttr("mybool", registry.BOOLEAN)
	rm.AddAttr("myuint", registry.UINTEGER)
	rm.AddAttr("mystr", registry.STRING)
	rm.AddAttr("*", registry.BOOLEAN)
	rm.AddAttrMap("mymapint", registry.NewItemType(registry.INTEGER))
	rm.AddAttrMap("mymapdec", registry.NewItemType(registry.DECIMAL))
	rm.AddAttrMap("mymapbool", registry.NewItemType(registry.BOOLEAN))
	rm.AddAttrMap("mymapuint", registry.NewItemType(registry.UINTEGER))

	reg.AddGroup("dirs", "d1")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT file f1",
		URL:    "/dirs/d1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-myint: -5",
			"xRegistry-mydec: 5.4",
			"xRegistry-mybool: true",
			"xRegistry-myuint: 5",
			"xRegistry-mymapint-k1: -6",
			"xRegistry-mymapdec-k2: -6.5",
			"xRegistry-mymapbool-k3: false",
			"xRegistry-mymapuint-k4: 6",
			"xRegistry-ext: true",
			"xRegistry-ifext: 666",
		},
		ReqBody: `hello`,
		Code:    201,
		ResBody: `hello`,
		ResHeaders: []string{
			"xRegistry-myint: -5",
			"xRegistry-mydec: 5.4",
			"xRegistry-mybool: true",
			"xRegistry-myuint: 5",
			"xRegistry-mymapint-k1: -6",
			"xRegistry-mymapdec-k2: -6.5",
			"xRegistry-mymapbool-k3: false",
			"xRegistry-mymapuint-k4: 6",
			"xRegistry-ext: true",
			"xRegistry-ifext: 666",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-id: f1",
			"xRegistry-epoch: 1",
			"Content-Type:text/plain; charset=utf-8",
			"Content-Location:http://localhost:8181/dirs/d1/files/f1/versions/1",
			"Content-Length:5",
			"Location:http://localhost:8181/dirs/d1/files/f1",
		},
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET file f1",
		URL:         "/dirs/d1/files/f1?meta",
		Method:      "GET",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "latestversionid": "1",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "ext": true,
  "ifext": 666,
  "mybool": true,
  "mydec": 5.4,
  "myint": -5,
  "mymapbool": {
    "k3": false
  },
  "mymapdec": {
    "k2": -6.5
  },
  "mymapint": {
    "k1": -6
  },
  "mymapuint": {
    "k4": 6
  },
  "myuint": 5,

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT file f1",
		URL:    "/dirs/d1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-mystr-foo: hello",
		},
		ReqBody: `hello`,
		Code:    400,
		ResBody: `Error processing resource: Attribute "mystr" must be a string
`,
		ResHeaders: []string{},
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT file f1",
		URL:    "/dirs/d1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-mystr-foo-bar: hello",
		},
		ReqBody: `hello`,
		Code:    400,
		ResBody: `Error processing resource: Attribute "mystr" must be a string
`,
		ResHeaders: []string{},
	})
}

func TestHTTPLatest(t *testing.T) {
	reg := NewRegistry("TestHTTPLatest")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true /* L */, true, true)

	reg.AddGroup("dirs", "d1")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT file f1 - latest = false",
		URL:    "/dirs/d1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-latest: false",
		},
		ReqBody:     `hello`,
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `"latest" can not be "false" since there is only one version, so it must be the latest
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT file f1 - latest = true",
		URL:    "/dirs/d1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-latest: true",
		},
		ReqBody:     `hello`,
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"Location: http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-id: f1",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f1/versions",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "POST file f1 - no latest",
		URL:         "/dirs/d1/files/f1",
		Method:      "POST",
		ReqHeaders:  []string{},
		ReqBody:     `hello`,
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/2",
			"Location: http://localhost:8181/dirs/d1/files/f1/versions/2",
			"xRegistry-id: 2",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/2",
			"xRegistry-latest: true",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT file f1/1 - latest = true",
		URL:    "/dirs/d1/files/f1/versions/1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-latest: true",
		},
		ReqBody:     `hello`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-id: 1",
			"xRegistry-epoch: 2",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-latest: true",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT file f1/1 - latest = false",
		URL:    "/dirs/d1/files/f1/versions/1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-latest: false",
		},
		ReqBody:     `hello`,
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody:     `"latest" can not be "false" since doing so would result in no latest version` + "\n",
	})

	rm.Latest = false
	rm.Save()

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT file f1/2 - latest = true - diff server",
		URL:    "/dirs/d1/files/f1/versions/2",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-latest: true",
		},
		ReqBody:     `hello`,
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `"latest" can not be "true", it is controlled by the server
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT file f1/1 - latest = true - match server",
		URL:    "/dirs/d1/files/f1/versions/1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-latest: true",
		},
		ReqBody:     `hello`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-id: 1",
			"xRegistry-epoch: 3",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-latest: true",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT file f1/1 - no latest",
		URL:         "/dirs/d1/files/f1/versions/1",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     `hello`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-id: 1",
			"xRegistry-epoch: 4",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-latest: true",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT file f1/2 - no latest",
		URL:         "/dirs/d1/files/f1/versions/2",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     `hello`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/2",
			"xRegistry-id: 2",
			"xRegistry-epoch: 2",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/2",
		},
		ResBody: `hello`,
	})

	// Now test ?setlatestversid stuff
	///////////////////////////////

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1?setlatest= not allowed",
		URL:     "/dirs/d1/files/f1?setlatestversionid",
		Method:  "POST",
		Code:    400,
		ResBody: `Resource "files" doesn't allow setting of "latestversionid"` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1?meta&setlatest= not allowed",
		URL:     "/dirs/d1/files/f1?&meta&setlatestversionid",
		Method:  "POST",
		Code:    400,
		ResBody: `Resource "files" doesn't allow setting of "latestversionid"` + "\n",
	})

	// Enable client-side setting
	rm.Latest = true
	rm.Save()

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1?setlatest - empty",
		URL:     "/dirs/d1/files/f1?setlatestversionid",
		Method:  "POST",
		Code:    400,
		ResBody: `"setlatestversionid" must not be empty` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1?setlatest= - empty",
		URL:     "/dirs/d1/files/f1?setlatestversionid=",
		Method:  "POST",
		Code:    400,
		ResBody: `"setlatestversionid" must not be empty` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1?meta&setlatest - empty",
		URL:     "/dirs/d1/files/f1?meta&setlatestversionid",
		Method:  "POST",
		Code:    400,
		ResBody: `"setlatestversionid" must not be empty` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1?meta&setlatest= - empty",
		URL:     "/dirs/d1/files/f1?meta&setlatestversionid=",
		Method:  "POST",
		Code:    400,
		ResBody: `"setlatestversionid" must not be empty` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file f1?setlatest=1 - no change",
		URL:    "/dirs/d1/files/f1?setlatestversionid=1",
		Method: "POST",
		ReqHeaders: []string{
			`xRegistry-id: bogus`,
		},
		ReqBody:     `ignore me`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Type: text/plain; charset=utf-8",
			"xRegistry-id: f1",
			"xRegistry-epoch: 4",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount: 2",
			"Content-Length: 5",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/1",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file f1?meta&setlatest=1 - no change",
		URL:    "/dirs/d1/files/f1?meta&setlatestversionid=1",
		Method: "POST",
		ReqHeaders: []string{
			`xRegistry-id: bogus`,
		},
		ReqBody:     `ignore me`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Type: application/json",
		},
		ResBody: `{
  "id": "f1",
  "epoch": 4,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "latestversionid": "1",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",

  "versionscount": 2,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file f1?setlatest=1 - set to 2",
		URL:    "/dirs/d1/files/f1?setlatestversionid=2",
		Method: "POST",
		ReqHeaders: []string{
			`xRegistry-id: bogus`,
		},
		ReqBody:     `ignore me`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Type: text/plain; charset=utf-8",
			"xRegistry-id: f1",
			"xRegistry-epoch: 2",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-latestversionid: 2",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/d1/files/f1/versions/2",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount: 2",
			"Content-Length: 5",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/2",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file f1?setlatest=1 - set back to 1",
		URL:    "/dirs/d1/files/f1?meta&setlatestversionid=1",
		Method: "POST",
		ReqHeaders: []string{
			`xRegistry-id: bogus`,
		},
		ReqBody:     `ignore me`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Type: application/json",
		},
		ResBody: `{
  "id": "f1",
  "epoch": 4,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "latestversionid": "1",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",

  "versionscount": 2,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	// errors
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setlatest bad group type",
		URL:        "/badgroup/d1/files/f1?meta&setlatestversionid=1",
		Method:     "POST",
		ReqHeaders: []string{`xRegistry-id: bogus`},
		Code:       404,
		ResHeaders: []string{"Content-Type: text/plain; charset=utf-8"},
		ResBody:    `Unknown Group type: badgroup` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setlatest bad group",
		URL:        "/dirs/dx/files/f1?meta&setlatestversionid=1",
		Method:     "POST",
		ReqHeaders: []string{`xRegistry-id: bogus`},
		Code:       404,
		ResHeaders: []string{"Content-Type: text/plain; charset=utf-8"},
		ResBody:    `Group "dx" not found` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setlatest bad resource type",
		URL:        "/dirs/d1/badfiles/f1?meta&setlatestversionid=1",
		Method:     "POST",
		ReqHeaders: []string{`xRegistry-id: bogus`},
		Code:       404,
		ResHeaders: []string{"Content-Type: text/plain; charset=utf-8"},
		ResBody:    `Unknown Resource type: badfiles` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setlatest bad resource",
		URL:        "/dirs/d1/files/xxf1?meta&setlatestversionid=1",
		Method:     "POST",
		ReqHeaders: []string{`xRegistry-id: bogus`},
		Code:       404,
		ResHeaders: []string{"Content-Type: text/plain; charset=utf-8"},
		ResBody:    `Resource "xxf1" not found` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setlatest bad version",
		URL:        "/dirs/d1/files/f1?meta&setlatestversionid=3",
		Method:     "POST",
		ReqHeaders: []string{`xRegistry-id: bogus`},
		Code:       404,
		ResHeaders: []string{"Content-Type: text/plain; charset=utf-8"},
		ResBody:    `Version "3" not found` + "\n",
	})

}

func TestHTTPDelete(t *testing.T) {
	reg := NewRegistry("TestHTTPDelete")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	reg.AddGroup("dirs", "d1")
	reg.AddGroup("dirs", "d2")
	reg.AddGroup("dirs", "d3")
	reg.AddGroup("dirs", "d4")

	// DELETE /GROUPs
	xHTTP(t, reg, "DELETE", "/", "", 405, "Can't delete an entire registry\n")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d2",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `[{"id":"d2"}]`,
		Code:    204,
		ResBody: ``,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d2",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `[]`, // should be a no-op, not delete everything
		Code:    204,
		ResBody: ``,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "GET /dirs - 1",
		URL:    "/dirs",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "d1": {
    "id": "d1",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d1",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/d1/files"
  },
  "d3": {
    "id": "d3",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d3",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/d3/files"
  },
  "d4": {
    "id": "d4",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d4",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/d4/files"
  }
}
`,
	})

	xHTTP(t, reg, "DELETE", "/dirs/d3?epoch=2x", "", 400,
		"Epoch value \"2x\" must be an UINTEGER\n")
	xHTTP(t, reg, "DELETE", "/dirs/d3?epoch=2", "", 400,
		"Epoch value for \"d3\" must be 1\n")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d3 err",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `[{"id":"d3","epoch":2}]`,
		Code:    400,
		ResBody: `Epoch value for "d3" must be 1
`,
	})

	// TODO add a delete of 2 with bad epoch in 2nd one and verify that
	// the first one isn't deleted due to the transaction rollback

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d3",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `[{"id":"d3","epoch":1}]`,
		Code:    204,
		ResBody: ``,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d3",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `[{"id":"d3","epoch":"1x"}]`,
		Code:    400,
		ResBody: `Can't parse "string" as a(n) "int" at line 1
`,
	})
	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - dx",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `[{"id":"dx","epoch":1}]`,
		Code:    204,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d3",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `[{"id":"d3","epoch":1}]`,
		Code:    204,
		ResBody: ``,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "GET /dirs - 2",
		URL:    "/dirs",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "d1": {
    "id": "d1",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d1",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/d1/files"
  },
  "d4": {
    "id": "d4",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d4",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/d4/files"
  }
}
`,
	})

	xHTTP(t, reg, "DELETE", "/dirs/d4?epoch=1", "", 204, "")
	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "GET /dirs - 2",
		URL:    "/dirs",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "d1": {
    "id": "d1",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d1",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/d1/files"
  }
}
`,
	})

	xHTTP(t, reg, "DELETE", "/dirs", "", 204, "")
	xHTTP(t, reg, "DELETE", "/dirs", "", 204, "")
	xHTTP(t, reg, "DELETE", "/dirsx", "", 404, "Unknown Group type: dirsx\n")
	xHTTP(t, reg, "GET", "/dirs", "", 200, "{}\n")

	// Reset
	reg.AddGroup("dirs", "d1")
	reg.AddGroup("dirs", "d2")
	reg.AddGroup("dirs", "d3")
	reg.AddGroup("dirs", "d4")

	// DELETE /GROUPs/gID
	xHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "GET /dirs - 4",
		URL:    "/dirs",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "d2": {
    "id": "d2",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d2",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/d2/files"
  },
  "d3": {
    "id": "d3",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d3",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/d3/files"
  },
  "d4": {
    "id": "d4",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d4",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/d4/files"
  }
}
`,
	})

	xHTTP(t, reg, "DELETE", "/dirs/d3", "", 204, ``)
	xHTTP(t, reg, "DELETE", "/dirs/dx", "", 404, `Group "dx" not found`+"\n")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "GET /dirs - 5",
		URL:    "/dirs",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "d2": {
    "id": "d2",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d2",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/d2/files"
  },
  "d4": {
    "id": "d4",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d4",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/d4/files"
  }
}
`,
	})

	xHTTP(t, reg, "DELETE", "/dirs", "", 204, "")
	xHTTP(t, reg, "GET", "/dirs", "", 200, "{}\n")

	// Reset
	d1, _ := reg.AddGroup("dirs", "d1")
	d1.AddResource("files", "f1", "v1.1")
	d1.AddResource("files", "f2", "v2.1")
	d1.AddResource("files", "f3", "v3.1")
	d1.AddResource("files", "f4", "v4.1")
	d1.AddResource("files", "f5", "v5.1")
	d1.AddResource("files", "f6", "v6.1")

	// DELETE Resources
	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "GET /dirs/d1 - 7",
		URL:    "/dirs/d1",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "id": "d1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1",

  "filescount": 6,
  "filesurl": "http://localhost:8181/dirs/d1/files"
}
`,
	})

	// DELETE /dirs/d1/files/f1
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1", "", 204, "")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/fx", "", 404,
		"Resource \"fx\" not found\n")

	// DELETE /dirs/d1/files/f1?epoch=...
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f3?epoch=2x", "", 400,
		"Epoch value \"2x\" must be an UINTEGER\n")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f3?epoch=2", "", 400,
		"Epoch value for \"f3\" must be 1\n")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f3?epoch=1", "", 204, "")

	// DELETE /dirs/d1/files/f3 - bad epoch in body
	xHTTP(t, reg, "DELETE", "/dirs/d1/files", `[{"id":"f2","epoch":"1x"}]`, 400,
		"Can't parse \"string\" as a(n) \"int\" at line 1\n")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files", `[{"id":"f2","epoch":2}]`, 400,
		"Epoch value for \"f2\" must be 1\n")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files", `[{"id":"f2","epoch":1}]`, 204, "")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files", `[{"id":"fx","epoch":1}]`, 204, "")

	xHTTP(t, reg, "DELETE", "/dirs/d1/files", `[{"id":"f2"},{"id":"f4","epoch":3}]`,
		400, "Epoch value for \"f4\" must be 1\n")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files", `[{"id":"f4"},{"id":"f5","epoch":1}]`,
		204, "")

	xHTTP(t, reg, "DELETE", "/dirs/d1/files", `[]`, 204, "") // no-op

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "GET /dirs/d1 - 7",
		URL:    "/dirs/d1/files",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "f6": {
    "id": "f6",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d1/files/f6?meta",
    "latestversionid": "v6.1",
    "latestversionurl": "http://localhost:8181/dirs/d1/files/f6/versions/v6.1?meta",

    "versionscount": 1,
    "versionsurl": "http://localhost:8181/dirs/d1/files/f6/versions"
  }
}
`,
	})

	xHTTP(t, reg, "DELETE", "/dirs/d1/files", ``, 204, "")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "GET /dirs/d1 - 7",
		URL:     "/dirs/d1/files",
		Method:  "GET",
		Code:    200,
		ResBody: "{}\n",
	})

	// TODO
	// DEL /dirs/d1/files [ f2,f4 ] - bad epoch on 2nd,verify f2 is still there

	// DELETE Versions
	f1, err := d1.AddResource("files", "f1", "v1")
	xNoErr(t, err)
	f1.AddVersion("v2", true)
	f1.AddVersion("v3", true)
	f1.AddVersion("v4", true)
	f1.AddVersion("v5", true)
	f1.AddVersion("v6", false)
	f1.AddVersion("v7", false)
	f1.AddVersion("v8", false)
	f1.AddVersion("v9", false)
	f1.AddVersion("v10", false)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?meta", ``, 200,
		`{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "latestversionid": "v5",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v5?meta",

  "versionscount": 10,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	// DELETE v1
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/vx", "", 404,
		"Version \"vx\" not found\n")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v1", "", 204, "")

	// DELETE /dirs/d1/files/f1?epoch=...
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v2?epoch=2x", "", 400,
		"Epoch value \"2x\" must be an UINTEGER\n")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v2?epoch=2", "", 400,
		"Epoch value for Version \"v2\" must be 1\n")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v2?epoch=1", "", 204, "")

	// DELETE /dirs/d1/files/f1/versions/v4 - bad epoch in body
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions",
		`[{"id":"v4","epoch":"1x"}]`, 400,
		"Can't parse \"string\" as a(n) \"int\" at line 1\n")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions",
		`[{"id":"v4","epoch":2}]`, 400,
		"Epoch value for \"v4\" must be 1\n")

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions", `[{"id":"v4","epoch":1}]`, 204, "")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions", `[{"id":"v4","epoch":1}]`, 204, "")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions", `[{"id":"vx","epoch":1}]`, 204, "")

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions",
		`[{"id":"v6"},{"id":"v7","epoch":3}]`, // v6 will still be around
		400, "Epoch value for \"v7\" must be 1\n")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions",
		`[{"id":"v7"},{"id":"v8","epoch":1}]`,
		204, "")

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions", `[]`, 204, "") // No-op

	// Make sure we have some left, and latest is still v5
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?meta&inline", "", 200, `{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "latestversionid": "v5",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v5?meta",

  "versions": {
    "v10": {
      "id": "v10",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v10?meta"
    },
    "v3": {
      "id": "v3",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v3?meta"
    },
    "v5": {
      "id": "v5",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v5?meta",
      "latest": true
    },
    "v6": {
      "id": "v6",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v6?meta"
    },
    "v9": {
      "id": "v9",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v9?meta"
    }
  },
  "versionscount": 5,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v5?setlatestversionid=v3",
		``, 204, "")

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v9?setlatestversionid=v9",
		``, 400, "Can't set latestversionid to Version being deleted\n")

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v9?setlatestversionid=vx",
		``, 400, "Can't find next latest Version \"vx\"\n")

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v9?setlatestversionid=v3",
		``, 204, "")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v9?setlatestversionid=vx",
		``, 404, "Version \"v9\" not found\n")

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?meta&inline", "", 200, `{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "latestversionid": "v3",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3?meta",

  "versions": {
    "v10": {
      "id": "v10",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v10?meta"
    },
    "v3": {
      "id": "v3",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v3?meta",
      "latest": true
    },
    "v6": {
      "id": "v6",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v6?meta"
    }
  },
  "versionscount": 3,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	f1.AddVersion("v1", false)
	// bad next
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions?setlatestversionid=vx", `[{"id":"v6"}]`, 400, "Can't find next latest Version \"vx\"\n")
	// next = being deleted
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions?setlatestversionid=v6", `[{"id":"v6"}]`, 400, "Can't set latestversionid to Version being deleted\n")

	// delete non-latest, change latest
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions?setlatestversionid=v10", `[{"id":"v6"}]`, 204, "")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?meta&inline", "", 200, `{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "latestversionid": "v10",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v10?meta",

  "versions": {
    "v1": {
      "id": "v1",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1?meta"
    },
    "v10": {
      "id": "v10",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v10?meta",
      "latest": true
    },
    "v3": {
      "id": "v3",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v3?meta"
    }
  },
  "versionscount": 3,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	// delete non-latest, latest not move
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions", `[{"id":"v3"}]`, 204, "")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?meta&inline", "", 200, `{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "latestversionid": "v10",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v10?meta",

  "versions": {
    "v1": {
      "id": "v1",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1?meta"
    },
    "v10": {
      "id": "v10",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v10?meta",
      "latest": true
    }
  },
  "versionscount": 2,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions?setlatestversionid=v1", `[]`, 204, "")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?meta&inline", "", 200, `{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "latestversionid": "v1",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1?meta",

  "versions": {
    "v1": {
      "id": "v1",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1?meta",
      "latest": true
    },
    "v10": {
      "id": "v10",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v10?meta"
    }
  },
  "versionscount": 2,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions", ``, 204, "")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions", "", 200, "{}\n")

	// TODO
	// DEL /..versions/ [ v2,v4 ] - bad epoch on 2nd,verify v2 is still there
}

func TestHTTPRequiredFields(t *testing.T) {
	reg := NewRegistry("TestHTTPRequiredFields")
	defer PassDeleteReg(t, reg)

	_, err := reg.Model.AddAttribute(&registry.Attribute{
		Name:           "clireq1",
		Type:           registry.STRING,
		ClientRequired: true,
		ServerRequired: true,
	})
	xNoErr(t, err)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	_, err = gm.AddAttribute(&registry.Attribute{
		Name:           "clireq2",
		Type:           registry.STRING,
		ClientRequired: true,
		ServerRequired: true,
	})
	xNoErr(t, err)

	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, true)
	_, err = rm.AddAttribute(&registry.Attribute{
		Name:           "clireq3",
		Type:           registry.STRING,
		ClientRequired: true,
		ServerRequired: true,
	})
	xNoErr(t, err)

	// Must commit before we call Set below otherwise the transaction will
	// be rolled back
	reg.Commit()

	// Registry itself
	err = reg.Set("description", "testing")
	xCheckErr(t, err, "Required property \"clireq1\" is missing")

	xNoErr(t, reg.JustSet("clireq1", "testing1"))
	xNoErr(t, reg.Set("description", "testing"))

	xHTTP(t, reg, "GET", "/", "", 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestHTTPRequiredFields",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "description": "testing",
  "clireq1": "testing1",

  "dirscount": 0,
  "dirsurl": "http://localhost:8181/dirs"
}
`)

	// Groups
	xHTTP(t, reg, "PUT", "/dirs/d1", `{"description": "testing"}`, 400,
		`Error processing group(d1): `+
			`Required property "clireq2" is missing`+"\n")

	xHTTP(t, reg, "PUT", "/dirs/d1", `{
  "description": "testing",
  "clireq2": "testing2"
}`, 201, `{
  "id": "d1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1",
  "description": "testing",
  "clireq2": "testing2",

  "filescount": 0,
  "filesurl": "http://localhost:8181/dirs/d1/files"
}
`)

	// Resources
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1?meta",
		`{"description": "testing"}`, 400,
		`Error processing resource(f1): `+
			`Required property "clireq3" is missing`+"\n")

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1?meta", `{
  "description": "testingdesc3",
  "clireq3": "testing3"
}`, 201, `{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "latestversionid": "1",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "description": "testingdesc3",
  "clireq3": "testing3",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "",
		URL:    "/dirs/d1/files/f2",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-description: testingdesc",
		},

		Code:       400,
		ResHeaders: []string{},
		ResBody: `Error processing resource(f2): Required property "clireq3" is missing
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "",
		URL:    "/dirs/d1/files/f2?meta",
		Method: "PUT",
		ReqBody: `{
  "description": "testingdesc2"
}`,

		Code:       400,
		ResHeaders: []string{},
		ResBody: `Error processing resource(f2): Required property "clireq3" is missing
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "",
		URL:    "/dirs/d1/files/f2",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-description: desctesting",
			"xRegistry-clireq3: testing3",
		},

		Code: 201,
		ResHeaders: []string{
			"xRegistry-clireq3: testing3",
			"xRegistry-description: desctesting",
			"xRegistry-epoch: 1",
			"xRegistry-id: f2",
			"xRegistry-latestversionid: 1",
			"xRegistry-latestversionurl: http://localhost:8181/dirs/d1/files/f2/versions/1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f2",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f2/versions",

			"Content-Length: 0",
			"Content-Location: http://localhost:8181/dirs/d1/files/f2/versions/1",
			"Location: http://localhost:8181/dirs/d1/files/f2",
		},
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "",
		URL:    "/dirs/d1/files/f2?meta",
		Method: "PUT",
		ReqBody: `{
  "description": "desctesting3",
  "clireq3": "testing4"
}`,

		Code: 200,
		ResBody: `{
  "id": "f2",
  "epoch": 2,
  "self": "http://localhost:8181/dirs/d1/files/f2?meta",
  "latestversionid": "1",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f2/versions/1?meta",
  "description": "desctesting3",
  "clireq3": "testing4",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "",
		URL:    "/dirs/d1/files/f2/versions/1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-id: 1",
			"xRegistry-description: desctesting",
			"xRegistry-clireq3: null",
		},

		Code:    400,
		ResBody: "Error processing resource: Required property \"clireq3\" is missing\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "",
		URL:    "/dirs/d1/files/f2/versions/1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-id: 1",
			"xRegistry-description: desctesting",
			"xRegistry-clireq3: null",
		},

		Code:    400,
		ResBody: "Error processing resource: Required property \"clireq3\" is missing\n",
	})
}

func TestHTTPHasDocumentFalse(t *testing.T) {
	reg := NewRegistry("TestHTTPHasDocumentFalse")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)

	// plural, singular, versions, verId bool, latest bool, hasDocument bool
	_, err = gm.AddResourceModel("bars", "bar", 0, true, true, true)
	rm, err := gm.AddResourceModel("files", "file", 0, true, true, false)
	xNoErr(t, err)
	_, err = rm.AddAttr("*", registry.STRING)
	xNoErr(t, err)

	xHTTP(t, reg, "POST", "/dirs/d1/files?meta", `{}`, 400,
		"Specifying \"?meta\" for a Resource that has the model "+
			"\"hasdocument\" value set to \"false\" is invalid\n")
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1?meta", `{}`, 400,
		"Specifying \"?meta\" for a Resource that has the model "+
			"\"hasdocument\" value set to \"false\" is invalid\n")
	xHTTP(t, reg, "POST", "/dirs/d1/files/f1?meta", `{}`, 400,
		"Specifying \"?meta\" for a Resource that has the model "+
			"\"hasdocument\" value set to \"false\" is invalid\n")
	xHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions?meta", `{}`, 400,
		"Specifying \"?meta\" for a Resource that has the model "+
			"\"hasdocument\" value set to \"false\" is invalid\n")
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1?meta", `{}`, 400,
		"Specifying \"?meta\" for a Resource that has the model "+
			"\"hasdocument\" value set to \"false\" is invalid\n")

	// Not really a "hasdoc" test, but it has to go someplace :-)
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/bars?meta",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-id: 123",
		},
		ReqBody: `{}`,

		Code: 400,
		ResBody: `Including "xRegistry" headers when "?meta" is used is invalid
`,
	})

	// Load up one that has hasdocument=true
	xHTTP(t, reg, "PUT", "/dirs/d1/bars/b1?meta", "", 201, `{
  "id": "b1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/bars/b1?meta",
  "latestversionid": "1",
  "latestversionurl": "http://localhost:8181/dirs/d1/bars/b1/versions/1?meta",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/bars/b1/versions"
}
`)

	xCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files",
		Method:  "POST",
		ReqBody: `{"test":"foo"}`,

		Code:      201,
		BodyMasks: []string{"id", "files/[a-zA-Z0-9]*|files/xxx"},
		ResBody: `{
  "id": "5bd549c7",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/5bd549c7",
  "latestversionid": "1",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/5bd549c7/versions/1",
  "test": "foo",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/5bd549c7/versions"
}
`})

	// Make sure that each type of Resource (w/ and w/o hasdoc) has the
	// correct self/latestversionurl URL (meaing w/ and w/o ?meta)
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1?inline",
		Method: "GET",

		Code: 200,
		BodyMasks: []string{
			`"id": "[a-z0-9]{8}"|"id": "xxx"`,
			`"[a-z0-9]{8}": {|"xxx": {"`,
			`files/[a-z0-9]{8}|files/xxx`,
		},
		ResBody: `{
  "id": "d1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1",

  "bars": {
    "b1": {
      "id": "b1",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/bars/b1?meta",
      "latestversionid": "1",
      "latestversionurl": "http://localhost:8181/dirs/d1/bars/b1/versions/1?meta",

      "versions": {
        "1": {
          "id": "1",
          "epoch": 1,
          "self": "http://localhost:8181/dirs/d1/bars/b1/versions/1?meta",
          "latest": true
        }
      },
      "versionscount": 1,
      "versionsurl": "http://localhost:8181/dirs/d1/bars/b1/versions"
    }
  },
  "barscount": 1,
  "barsurl": "http://localhost:8181/dirs/d1/bars",
  "files": {
    "de0fe3c9": {
      "id": "de0fe3c9",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/de0fe3c9",
      "latestversionid": "1",
      "latestversionurl": "http://localhost:8181/dirs/d1/files/de0fe3c9/versions/1",
      "test": "foo",

      "versions": {
        "1": {
          "id": "1",
          "epoch": 1,
          "self": "http://localhost:8181/dirs/d1/files/de0fe3c9/versions/1",
          "latest": true,
          "test": "foo"
        }
      },
      "versionscount": 1,
      "versionsurl": "http://localhost:8181/dirs/d1/files/de0fe3c9/versions"
    }
  },
  "filescount": 1,
  "filesurl": "http://localhost:8181/dirs/d1/files"
}
`,
	})

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{"foo":"test"}`, 201,
		`{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "latestversionid": "1",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "foo": "test",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{"foo2":"test2"}`, 200,
		`{
  "id": "f1",
  "epoch": 2,
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "latestversionid": "1",
  "latestversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "foo2": "test2",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1/files/f1", `{"foo2":"test2"}`, 201,
		`{
  "id": "2",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/2",
  "latest": true,
  "foo2": "test2"
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{"foo3":"test3"}`, 201,
		`{
  "id": "3",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/3",
  "latest": true,
  "foo3": "test3"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/3", `{"foo3.1":"test3.1"}`, 200,
		`{
  "id": "3",
  "epoch": 2,
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/3",
  "latest": true,
  "foo3.1": "test3.1"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/4", `{"foo4":"test4"}`, 201,
		`{
  "id": "4",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/4",
  "latest": true,
  "foo4": "test4"
}
`)

}

func TestHTTPModelSchema(t *testing.T) {
	reg := NewRegistry("TestHTTPModelSchema")
	defer PassDeleteReg(t, reg)

	xHTTP(t, reg, "GET", "/model", "", 200, `{
  "schemas": [
    "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
  ],
  "attributes": {
    "specversion": {
      "name": "specversion",
      "type": "string",
      "readonly": true,
      "serverrequired": true
    },
    "id": {
      "name": "id",
      "type": "string",
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
    },
    "description": {
      "name": "description",
      "type": "string"
    },
    "documentation": {
      "name": "documentation",
      "type": "url"
    },
    "labels": {
      "name": "labels",
      "type": "map",
      "item": {
        "type": "string"
      }
    },
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
    }
  }
}
`)

	xHTTP(t, reg, "GET", "/model?schema="+registry.XREGSCHEMA, "", 200, `{
  "schemas": [
    "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
  ],
  "attributes": {
    "specversion": {
      "name": "specversion",
      "type": "string",
      "readonly": true,
      "serverrequired": true
    },
    "id": {
      "name": "id",
      "type": "string",
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
    },
    "description": {
      "name": "description",
      "type": "string"
    },
    "documentation": {
      "name": "documentation",
      "type": "url"
    },
    "labels": {
      "name": "labels",
      "type": "map",
      "item": {
        "type": "string"
      }
    },
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
    }
  }
}
`)

	xHTTP(t, reg, "GET", "/model?schema="+registry.XREGSCHEMA+"/"+registry.SPECVERSION, "", 200, `{
  "schemas": [
    "`+registry.XREGSCHEMA+"/"+registry.SPECVERSION+`"
  ],
  "attributes": {
    "specversion": {
      "name": "specversion",
      "type": "string",
      "readonly": true,
      "serverrequired": true
    },
    "id": {
      "name": "id",
      "type": "string",
      "serverrequired": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "serverrequired": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "serverrequired": true
    },
    "description": {
      "name": "description",
      "type": "string"
    },
    "documentation": {
      "name": "documentation",
      "type": "url"
    },
    "labels": {
      "name": "labels",
      "type": "map",
      "item": {
        "type": "string"
      }
    },
    "createdby": {
      "name": "createdby",
      "type": "string",
      "readonly": true
    },
    "createdon": {
      "name": "createdon",
      "type": "timestamp",
      "readonly": true
    },
    "modifiedby": {
      "name": "modifiedby",
      "type": "string",
      "readonly": true
    },
    "modifiedon": {
      "name": "modifiedon",
      "type": "timestamp",
      "readonly": true
    }
  }
}
`)

	xHTTP(t, reg, "GET", "/model?schema="+registry.XREGSCHEMA+"/bad", "", 400,
		`Unsupported schema format: xRegistry-json/bad
`)

	xHTTP(t, reg, "GET", "/model?schema=bad", "", 400,
		`Unsupported schema format: bad
`)

}

func TestHTTPReadOnlyResource(t *testing.T) {
	reg := NewRegistry("TestHTTPReadOnlyResource")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)

	_, err = gm.AddResourceModelFull(&registry.ResourceModel{
		Plural:      "files",
		Singular:    "file",
		Versions:    0,
		VersionId:   true,
		Latest:      true,
		HasDocument: true,
		ReadOnly:    true,
	})
	xNoErr(t, err)

	xHTTP(t, reg, "PUT", "/dirs/dir1", "{}", 201, `{
  "id": "dir1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/dir1",

  "filescount": 0,
  "filesurl": "http://localhost:8181/dirs/dir1/files"
}
`)

	d1, err := reg.FindGroup("dirs", "dir1")
	xNoErr(t, err)
	xCheck(t, d1 != nil, "d1 should not be nil")

	f1, err := d1.AddResource("files", "f1", "v1")
	xNoErr(t, err)
	xCheck(t, f1 != nil, "f1 should not be nil")

	xHTTP(t, reg, "GET", "/dirs/dir1/files", "", 200, `{
  "f1": {
    "id": "f1",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/dir1/files/f1?meta",
    "latestversionid": "v1",
    "latestversionurl": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",

    "versionscount": 1,
    "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions"
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/dir1/files/f1/versions/v1?meta", "", 200, `{
  "id": "v1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "latest": true
}
`)

	xHTTP(t, reg, "POST", "/dirs/dir1/files", "", 405,
		"Write operations to read-only resources are not allowed\n")
	xHTTP(t, reg, "PUT", "/dirs/dir1/files/f1", "", 405,
		"Write operations to read-only resources are not allowed\n")
	xHTTP(t, reg, "POST", "/dirs/dir1/files/f1", "", 405,
		"Write operations to read-only resources are not allowed\n")
	xHTTP(t, reg, "POST", "/dirs/dir1/files/f1/versions", "", 405,
		"Write operations to read-only resources are not allowed\n")
	xHTTP(t, reg, "PUT", "/dirs/dir1/files/f1/versions/v1", "", 405,
		"Write operations to read-only resources are not allowed\n")
}
