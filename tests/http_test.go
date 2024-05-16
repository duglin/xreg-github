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
	BodyMasks   []string // "PROPNAME" or "SEARCH||REPLACE"
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

	// t.Logf("%v\n%s", res.Header, string(resBody))
	testHeaders := map[string]bool{}

	seenTS := map[string]string{}
	replaceFunc := func(input string) string {
		if val, ok := seenTS[input]; ok {
			return val
		}
		val := fmt.Sprintf("YYYY-MM-DDTHH:MM:%dZ", len(seenTS)+1)
		seenTS[input] = val
		return val
	}
	TSre := savedREs[TSREGEXP]

	// Do expected headers first
	for i, _ := range test.ResHeaders {
		val := test.ResHeaders[i]
		test.ResHeaders[i] = TSre.ReplaceAllStringFunc(val, replaceFunc)
	}
	// reset, and then do actual response headers
	seenTS = map[string]string{}
	for _, key := range registry.SortedKeys(res.Header) {
		if strings.HasSuffix(key, "at") {
			if resVal := res.Header.Get(key); resVal != "" {
				resVal = TSre.ReplaceAllStringFunc(resVal, replaceFunc)
				res.Header.Set(key, resVal)
			}
		}
	}

	for _, header := range test.ResHeaders {
		name, value, _ := strings.Cut(header, ":")
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		testHeaders[strings.ToLower(name)] = true

		resValue := res.Header.Get(name)

		for _, mask := range test.HeaderMasks {
			var re *regexp.Regexp
			search, replace, _ := strings.Cut(mask, "||")
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
		search, replace, found := strings.Cut(mask, "||")
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

var savedREs = map[string]*regexp.Regexp{
	TSREGEXP: regexp.MustCompile(TSREGEXP),
}

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
  "self": "<a href="http://localhost:8181/?html">http://localhost:8181/?html</a>",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z"
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
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "model": {
    "schemas": [
      "` + registry.XREGSCHEMA + "/" + registry.SPECVERSION + `"
    ],
    "attributes": {
      "specversion": {
        "name": "specversion",
        "type": "string",
        "readonly": true,
        "immutable": true,
        "serverrequired": true
      },
      "id": {
        "name": "id",
        "type": "string",
        "immutable": true,
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
      "createdat": {
        "name": "createdat",
        "type": "timestamp"
      },
      "modifiedat": {
        "name": "modifiedat",
        "type": "timestamp"
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
      "immutable": true,
      "serverrequired": true
    },
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
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
    "createdat": {
      "name": "createdat",
      "type": "timestamp"
    },
    "modifiedat": {
      "name": "modifiedat",
      "type": "timestamp"
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
      "immutable": true,
      "serverrequired": true
    },
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
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
    "createdat": {
      "name": "createdat",
      "type": "timestamp"
    },
    "modifiedat": {
      "name": "modifiedat",
      "type": "timestamp"
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
      "immutable": true,
      "serverrequired": true
    },
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
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
    "createdat": {
      "name": "createdat",
      "type": "timestamp"
    },
    "modifiedat": {
      "name": "modifiedat",
      "type": "timestamp"
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
      "immutable": true,
      "serverrequired": true
    },
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
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
    "createdat": {
      "name": "createdat",
      "type": "timestamp"
    },
    "modifiedat": {
      "name": "modifiedat",
      "type": "timestamp"
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
          "immutable": true,
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
        "createdat": {
          "name": "createdat",
          "type": "timestamp"
        },
        "modifiedat": {
          "name": "modifiedat",
          "type": "timestamp"
        }
      },
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "setstickydefaultversion": true,
          "hasdocument": true,
          "attributes": {
            "id": {
              "name": "id",
              "type": "string",
              "immutable": true,
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
            "isdefault": {
              "name": "isdefault",
              "type": "boolean",
              "readonly": true
            },
            "stickydefaultversion": {
              "name": "stickydefaultversion",
              "type": "boolean",
              "readonly": true
            },
            "defaultversionid": {
              "name": "defaultversionid",
              "type": "string",
              "readonly": true
            },
            "defaultversionurl": {
              "name": "defaultversionurl",
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
            "createdat": {
              "name": "createdat",
              "type": "timestamp"
            },
            "modifiedat": {
              "name": "modifiedat",
              "type": "timestamp"
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
          "maxversions": 0,
          "setversionid": true,
          "setstickydefaultversion": true,
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
      "immutable": true,
      "serverrequired": true
    },
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
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
    "createdat": {
      "name": "createdat",
      "type": "timestamp"
    },
    "modifiedat": {
      "name": "modifiedat",
      "type": "timestamp"
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
          "immutable": true,
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
        "createdat": {
          "name": "createdat",
          "type": "timestamp"
        },
        "modifiedat": {
          "name": "modifiedat",
          "type": "timestamp"
        }
      },
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "setstickydefaultversion": true,
          "hasdocument": false,
          "attributes": {
            "id": {
              "name": "id",
              "type": "string",
              "immutable": true,
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
            "isdefault": {
              "name": "isdefault",
              "type": "boolean",
              "readonly": true
            },
            "stickydefaultversion": {
              "name": "stickydefaultversion",
              "type": "boolean",
              "readonly": true
            },
            "defaultversionid": {
              "name": "defaultversionid",
              "type": "string",
              "readonly": true
            },
            "defaultversionurl": {
              "name": "defaultversionurl",
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
            "createdat": {
              "name": "createdat",
              "type": "timestamp"
            },
            "modifiedat": {
              "name": "modifiedat",
              "type": "timestamp"
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
      "immutable": true,
      "serverrequired": true
    },
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
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
    "createdat": {
      "name": "createdat",
      "type": "timestamp"
    },
    "modifiedat": {
      "name": "modifiedat",
      "type": "timestamp"
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
          "immutable": true,
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
        "createdat": {
          "name": "createdat",
          "type": "timestamp"
        },
        "modifiedat": {
          "name": "modifiedat",
          "type": "timestamp"
        }
      },
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "setstickydefaultversion": true,
          "hasdocument": true,
          "attributes": {
            "id": {
              "name": "id",
              "type": "string",
              "immutable": true,
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
            "isdefault": {
              "name": "isdefault",
              "type": "boolean",
              "readonly": true
            },
            "stickydefaultversion": {
              "name": "stickydefaultversion",
              "type": "boolean",
              "readonly": true
            },
            "defaultversionid": {
              "name": "defaultversionid",
              "type": "string",
              "readonly": true
            },
            "defaultversionurl": {
              "name": "defaultversionurl",
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
            "createdat": {
              "name": "createdat",
              "type": "timestamp"
            },
            "modifiedat": {
              "name": "modifiedat",
              "type": "timestamp"
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
		`Attribute "description"(testing) must be one of the enum values: one, two`+"\n")

	xHTTP(t, reg, "PUT", "/", `{}`, 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestHTTPModel",
  "epoch": 2,
  "self": "http://localhost:8181/",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirscount": 0,
  "dirsurl": "http://localhost:8181/dirs"
}
`)

	xHTTP(t, reg, "PUT", "/", `{"description": "two"}`, 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestHTTPModel",
  "epoch": 3,
  "self": "http://localhost:8181/",
  "description": "two",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

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
		ResBody:    "ID can't be an empty string\n",
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
  "self": "http://localhost:8181/",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z"
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
  "self": "http://localhost:8181/",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z"
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
  "self": "http://localhost:8181/",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z"
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
		ResBody:    "Attribute \"epoch\"(33) doesn't match existing value (4)\n",
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
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
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
		ResBody: `Attribute "mymapobj.mapobj_int" must be a map[string] or object
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
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
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
			ResBody:    test.response + "\n",
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
		ResBody: `Attribute "self" must be a url
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
		ResBody:    "Attribute \"id\" must be a string\n",
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
		ResBody:    "Can't change the ID of an entity(TestHTTPRegistry->foo)\n",
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
  "documentation": "docs",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z"
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
  "self": "http://localhost:8181/",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z"
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
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
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
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
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
		Name:       "Create group - empty",
		URL:        "/dirs",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    "",
		Code:       200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "Create group - {}",
		URL:        "/dirs",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    "{}",
		Code:       200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST group - full, single",
		URL:        "/dirs",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
  "dir1": {
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
  }
}`,
		Code: 200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{
  "dir1": {
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
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
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
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST group - full, multiple",
		URL:        "/dirs",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
  "dir2": {
    "id":"dir2",
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
  },
  "dir3": {}
}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "dir2": {
    "id": "dir2",
    "name": "my group",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/dir2",
    "description": "desc",
    "documentation": "docs-url",
    "labels": {
      "label1": "value1",
      "label2": "5",
      "label3": "123.456",
      "label4": ""
    },
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
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
    "filesurl": "http://localhost:8181/dirs/dir2/files"
  },
  "dir3": {
    "id": "dir3",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/dir3",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/dir3/files"
  }
}
`,
	})

	xHTTP(t, reg, "GET", "/dirs", "", 200, `{
  "dir1": {
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
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
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
  },
  "dir2": {
    "id": "dir2",
    "name": "my group",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/dir2",
    "description": "desc",
    "documentation": "docs-url",
    "labels": {
      "label1": "value1",
      "label2": "5",
      "label3": "123.456",
      "label4": ""
    },
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:02Z",
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
    "filesurl": "http://localhost:8181/dirs/dir2/files"
  },
  "dir3": {
    "id": "dir3",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/dir3",
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:02Z",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/dir3/files"
  }
}
`)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST group - full, multiple - clear",
		URL:        "/dirs",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
  "dir1": {},
  "dir2": {},
  "dir3": {
    "description": "hello"
  }
}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "dir1": {
    "id": "dir1",
    "epoch": 2,
    "self": "http://localhost:8181/dirs/dir1",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/dir1/files"
  },
  "dir2": {
    "id": "dir2",
    "epoch": 2,
    "self": "http://localhost:8181/dirs/dir2",
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:02Z",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/dir2/files"
  },
  "dir3": {
    "id": "dir3",
    "epoch": 2,
    "self": "http://localhost:8181/dirs/dir3",
    "description": "hello",
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:02Z",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/dir3/files"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST group - full, multiple - err",
		URL:        "/dirs",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
  "dir2": {
    "id":"dir2",
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
  },
  "dir3": {},
  "dir4": {
    "id": "dir44"
  }
}`,
		Code:       400,
		ResHeaders: []string{},
		ResBody:    `Can't change the ID of an entity(dir4->dir44)` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT group - update",
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
  "myarray": [],
  "mymap": {},
  "myobj": {}
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
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
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
  "epoch": 3,
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
  "epoch": 4,
  "self": "http://localhost:8181/dirs/dir1",
  "description": "desc new",
  "documentation": "docs-url-new",
  "labels": {
    "label.new": "new"
  },
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
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
		ResBody:    "Attribute \"epoch\"(10) doesn't match existing value (4)\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT group - update - err id",
		URL:        "/dirs/dir1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{ "id":"dir2" }`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "Can't change the ID of an entity(dir1->dir2)\n",
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
  "epoch": 5,
  "self": "http://localhost:8181/dirs/dir1",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

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
  "description":"desc new",
  "documentation":"docs-url-new",
  "labels": {
    "label.new": "new"
  },
  "format": "myformat/1"
}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "Can't change the ID of an entity(dir2->dir3)\n",
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
		Name:   "POST resources - empty",
		URL:    "/dirs/dir1/files",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-id: ff1",
		},
		ReqBody: "",
		Code:    201,
		HeaderMasks: []string{
			"^[a-z0-9]{8}$||xxx",
			"files/[^/]+||files/xxx",
		},
		ResHeaders: []string{
			"Content-Type: ",
			"xRegistry-id: ff1",
			"xRegistry-epoch: 1",
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/dir1/files/xxx/versions/1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/xxx",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/xxx/versions",
			"Location: http://localhost:8181/dirs/dir1/files/xxx",
			"Content-Location: http://localhost:8181/dirs/dir1/files/xxx/versions/1",
			"Content-Length: 0",
		},
		ResBody: ``,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST resources - w/doc",
		URL:    "/dirs/dir1/files",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-id: ff2",
		},
		ReqBody: "My cool doc",
		Code:    201,
		HeaderMasks: []string{
			"^[a-z0-9]{8}$||xxx",
			"files/[^/]+||files/xxx",
		},
		ResHeaders: []string{
			"Content-Type: text/plain; charset=utf-8",
			"xRegistry-id: ff2",
			"xRegistry-epoch: 1",
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/dir1/files/xxx/versions/1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/xxx",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
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
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/dir1/files/f1/versions/1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
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
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/dir1/files/f1/versions/1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f1/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f1/versions/1",
			"Content-Length: 17",
		},
		ResBody: `My cool doc - new`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources - w/doc - no content-type",
		URL:        "/dirs/dir1/files/f1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "My cool doc - new one",
		Code:       200,
		ResHeaders: []string{
			"Content-Type: my/format",
			"xRegistry-id: f1",
			"xRegistry-epoch: 3",
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/dir1/files/f1/versions/1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f1/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f1/versions/1",
			"Content-Length: 21",
		},
		ResBody: `My cool doc - new one`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - w/doc - revert content-type and body",
		URL:    "/dirs/dir1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"Content-Type: null",
		},
		ReqBody: "My cool doc - new x2",
		Code:    200,
		ResHeaders: []string{
			// "Content-Type: text/plain; charset=utf-8",
			"xRegistry-id: f1",
			"xRegistry-epoch: 4",
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/dir1/files/f1/versions/1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
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
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
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
		Name:        "PUT resources - update default - content",
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
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
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
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/dir1/files/f4/versions/1",
			"xRegistry-name: my doc",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-fileurl: http://example.com",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f4/versions",
			"Location: http://localhost:8181/dirs/dir1/files/f4",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f4/versions/1",
		},
		ResBody: "",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - update default - URL",
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
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
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
		Name:   "PUT resources - update default - URL + body - error",
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
		Name:   "PUT resources - update default - URL - null",
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
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-origin: foo.com",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
		},
		ResBody: "",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT resources - update default - w/body",
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
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-origin: foo.com",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
		},
		ResBody: "another body",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - update default - w/body - clear 1 prop",
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
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-origin: foo.com",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
		},
		ResBody: "another body",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - update default - w/body - edit 2 label",
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
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-documentation: my doc url",
			"xRegistry-origin: foo.com",
			"xRegistry-labels-l1: l1l1",
			"xRegistry-labels-l4: 4444",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
		},
		ResBody: "another body",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - update default - w/body - edit 1 label",
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
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-documentation: my doc url",
			"xRegistry-origin: foo.com",
			"xRegistry-labels-l3: 3333",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
		},
		ResBody: "another body",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - update default - w/body - delete labels",
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
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-documentation: my doc url",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-origin: foo.com",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
		},
		ResBody: "another body",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - update default - w/body - delete+add labels",
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
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-documentation: my doc url",
			"xRegistry-origin: foo.com",
			"xRegistry-labels-foo: foo",
			"xRegistry-labels-foo-bar: l-foo-bar",
			"xRegistry-labels-foo_bar: l-foo_bar",
			"xRegistry-labels-foo.bar: l-foo.bar",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
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
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"xRegistry-documentation: my doc url",
			"xRegistry-origin: foo.com",
			"xRegistry-labels-foo: foo",
			"xRegistry-labels-foo-bar: l-foo-bar",
			"xRegistry-labels-foo_bar: l-foo_bar",
			"xRegistry-labels-foo.bar: l-foo.bar",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
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

	// Change the modifiedat field since it'll change
	re := regexp.MustCompile(`"modifiedat": "[^"]*"`)
	body = re.ReplaceAll(body, []byte(`"modifiedat": "2024-01-01T12:12:12Z"`))

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
	f.SetSave(NewPP().P("#resource").UI(), "Hello world! v1")

	v, _ := f.AddVersion("v2")
	v.SetSave(NewPP().P("#resourceURL").UI(), "http://localhost:8181/EMPTY-URL")

	v, _ = f.AddVersion("v3")
	v.SetSave(NewPP().P("#resourceProxyURL").UI(), "http://localhost:8181/EMPTY-Proxy")

	// URL
	f, _ = d.AddResource("files", "f2-url", "v1")
	f.SetSave(NewPP().P("#resource").UI(), "Hello world! v1")

	v, _ = f.AddVersion("v2")
	v.SetSave(NewPP().P("#resourceProxyURL").UI(), "http://localhost:8181/EMPTY-Proxy")

	v, _ = f.AddVersion("v3")
	v.SetSave(NewPP().P("#resourceURL").UI(), "http://localhost:8181/EMPTY-URL")

	// Resource
	f, _ = d.AddResource("files", "f3-resource", "v1")
	f.SetSave(NewPP().P("#resourceProxyURL").UI(), "http://localhost:8181/EMPTY-Proxy")

	v, _ = f.AddVersion("v2")
	v.SetSave(NewPP().P("#resourceURL").UI(), "http://localhost:8181/EMPTY-URL")

	v, _ = f.AddVersion("v3")
	v.SetSave(NewPP().P("#resource").UI(), "Hello world! v3")

	// /dirs/d1/files/f1-proxy/v1 - resource
	//                        /v2 - URL
	//                        /v3 - ProxyURL  <- default
	// /dirs/d1/files/f2-url/v1 - resource
	//                      /v2 - ProxyURL
	//                      /v3 - URL  <- default
	// /dirs/d1/files/f3-resource/v1 - ProxyURL
	//                           /v2 - URL
	//                           /v3 - resource  <- default

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET resource - default - f1",
		URL:         "/dirs/d1/files/f1-proxy",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f1-proxy",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1-proxy",
			"xRegistry-defaultversionid: v3",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/d1/files/f1-proxy/versions/v3",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
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
		Name:        "GET resource - default - f1/v3",
		URL:         "/dirs/d1/files/f1-proxy/versions/v3",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: v3",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1-proxy/versions/v3",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
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
		Name:        "GET resource - default - f1/v2",
		URL:         "/dirs/d1/files/f1-proxy/versions/v2",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        303,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: v2",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1-proxy/versions/v2",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
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
		Name:        "GET resource - default - f2",
		URL:         "/dirs/d1/files/f2-url",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        303,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f2-url",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f2-url",
			"xRegistry-defaultversionid: v3",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/d1/files/f2-url/versions/v3",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
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
		Name:        "GET resource - default - f3",
		URL:         "/dirs/d1/files/f3-resource",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: f3-resource",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f3-resource",
			"xRegistry-defaultversionid: v3",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/d1/files/f3-resource/versions/v3",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
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
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "application/json",

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
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "application/json",
  "file": "Hello world! v1",

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
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1-proxy/versions/1",
			"Content-Length: 15",
			"Content-Type: application/json",
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
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "application/json",
  "file": "Hello world! v1",

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
		  "v2": {
		    "id": "v2",
            "file": "Hello world! v2"
		  }
		}`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "v2": {
    "id": "v2",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d1/files/f1-proxy/versions/v2?meta",
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z"
  }
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
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
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
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
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
			"xRegistry-defaultversionid:2",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/d1/files/f1-proxy/versions/2",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
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
			"xRegistry-defaultversionid:2",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/d1/files/f1-proxy/versions/2",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-versionscount:3",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f1-proxy/versions",
		},
		ResBody: `more data`,
	})

	// Update default with fileURL
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
  "defaultversionid": "2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions/2?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "fileurl": "http://localhost:8181/EMPTY-URL",

  "versionscount": 3,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions"
}
`,
	})

	// Update default - delete fileurl, notice no "id" either
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
  "defaultversionid": "2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions/2?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "versionscount": 3,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions"
}
`,
	})

	// Update default - set 'file' and 'fileurl' - error
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
		ResBody: `Only one of file,fileurl,filebase64 can be present at a time
`,
	})

	// Update default - set 'filebase64' and 'fileurl' - error
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
		ResBody: `Only one of file,fileurl,filebase64 can be present at a time
`,
	})

	// Update default - with 'filebase64'
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
  "defaultversionid": "2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions/2?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "versionscount": 3,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions"
}
`,
	})

	// Get default
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
			"xRegistry-defaultversionid: 2",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/d1/files/f1-proxy/versions/2",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
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
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
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
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
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
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-fileurl:http://localhost:8181/EMPTY-URL",
		},
		ResBody: "",
	})

	// testing of "isdefault" processing

	// Set up the following:
	// /dirs/d1/files/ff1-proxy/v1 - resource
	//                        /v2 - URL
	//                        /v3 - ProxyURL  <- default
	// /dirs/d1/files/ff2-url/v1 - resource
	//                      /v2 - ProxyURL
	//                      /v3 - URL  <- default
	// /dirs/d1/files/ff3-resource/v1 - ProxyURL
	//                           /v2 - URL
	//                           /v3 - resource  <- default

	// Now create the ff1-proxy variants
	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file ff1-proxy-v1 Resource",
		URL:    "/dirs/d1/files/ff1-proxy?meta",
		Method: "POST",
		ReqBody: `{
		  "v1": {
		    "id": "v1",
		    "file": "In resource ff1-proxy"
		  }
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "v1": {
    "id": "v1",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v1?meta",
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file ff1-proxy-v2 URL",
		URL:    "/dirs/d1/files/ff1-proxy?meta",
		Method: "POST",
		ReqBody: `{
		  "v2": {
		    "id": "v2",
		    "fileurl": "http://localhost:8181/EMPTY-URL"
		  }
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "v2": {
    "id": "v2",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v2?meta",
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "fileurl": "http://localhost:8181/EMPTY-URL"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file ff1-proxy-v3 ProxyURL",
		URL:    "/dirs/d1/files/ff1-proxy?meta",
		Method: "POST",
		ReqBody: `{
		  "v3": {
		    "id": "v3",
		    "fileproxyurl": "http://localhost:8181/EMPTY-Proxy"
		  }
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "v3": {
    "id": "v3",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v3?meta",
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z"
  }
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
		  "v1": {
		    "id": "v1",
		    "file": "In resource ff2-url"
		  }
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "v1": {
    "id": "v1",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d1/files/ff2-url/versions/v1?meta",
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file ff2-url-v2 ProxyURL",
		URL:    "/dirs/d1/files/ff2-url?meta",
		Method: "POST",
		ReqBody: `{
		  "v2": {
		    "id": "v2",
		    "fileproxyurl": "http://localhost:8181/EMPTY-Proxy"
		  }
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "v2": {
    "id": "v2",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d1/files/ff2-url/versions/v2?meta",
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file ff2-url-v2 URL",
		URL:    "/dirs/d1/files/ff2-url?meta",
		Method: "POST",
		ReqBody: `{
		  "v3": {
		    "id": "v3",
		    "fileurl": "http://localhost:8181/EMPTY-URL"
		  }
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "v3": {
    "id": "v3",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d1/files/ff2-url/versions/v3?meta",
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "fileurl": "http://localhost:8181/EMPTY-URL"
  }
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
		  "v1": {
		    "id": "v1",
		    "fileproxyurl": "http://localhost:8181/EMPTY-Proxy"
		  }
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "v1": {
    "id": "v1",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d1/files/ff3-resource/versions/v1?meta",
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file ff3-resource-v2 URL",
		URL:    "/dirs/d1/files/ff3-resource?meta",
		Method: "POST",
		ReqBody: `{
		  "v2": {
		    "id": "v2",
		    "fileurl": "http://localhost:8181/EMPTY-URL"
	      }
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "v2": {
    "id": "v2",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d1/files/ff3-resource/versions/v2?meta",
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "fileurl": "http://localhost:8181/EMPTY-URL"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file ff3-resource-v3 resource",
		URL:    "/dirs/d1/files/ff3-resource?meta",
		Method: "POST",
		ReqBody: `{
		  "v3": {
		    "id": "v3",
		    "file": "In resource ff3-resource"
		  }
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "v3": {
    "id": "v3",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d1/files/ff3-resource/versions/v3?meta",
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z"
  }
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
		Name:        "GET resource - default - ff1",
		URL:         "/dirs/d1/files/ff1-proxy",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: ff1-proxy",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/ff1-proxy",
			"xRegistry-defaultversionid: v3",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v3",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-versionscount: 3",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/ff1-proxy/versions",
			"Content-Location: http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v3",
		},
		ResBody: "hello-Proxy",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET resource - default - ff1/v3",
		URL:         "/dirs/d1/files/ff1-proxy/versions/v3",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: v3",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v3",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
		},
		ResBody: "hello-Proxy",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET resource - default - ff1/v2",
		URL:         "/dirs/d1/files/ff1-proxy/versions/v2",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        303,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: v2",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v2",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-fileurl: http://localhost:8181/EMPTY-URL",
			"Location: http://localhost:8181/EMPTY-URL",
		},
		ResBody: "",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET resource - default - ff2",
		URL:         "/dirs/d1/files/ff2-url",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        303,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: ff2-url",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/ff2-url",
			"xRegistry-defaultversionid: v3",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/d1/files/ff2-url/versions/v3",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-fileurl: http://localhost:8181/EMPTY-URL",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/ff2-url/versions",
			"xRegistry-versionscount: 3",
			"Location: http://localhost:8181/EMPTY-URL",
		},
		ResBody: "",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET resource - default - ff3",
		URL:         "/dirs/d1/files/ff3-resource",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-id: ff3-resource",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/ff3-resource",
			"xRegistry-defaultversionid: v3",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/d1/files/ff3-resource/versions/v3",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
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
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
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
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
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
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f5/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
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
			"xRegistry-defaultversionid:1",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/d1/files/f5/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
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
  "self": "http://localhost:8181/dirs/d1/files/f5/versions/v1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z"
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
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f5/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
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
  "self": "http://localhost:8181/",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z"
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
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
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
		ResBody: "Attribute \"myint\"(4) must be " +
			"one of the enum values: 1, 2, 3\n",
	})

	attr.Strict = registry.PtrBool(false)
	reg.Model.VerifyAndSave()

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
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
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
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
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
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
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
		ResBody: "Invalid extension(s): myext\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - ifvalue - required mystr",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint": 20
	   }`,
		Code:    400,
		ResBody: "Required property \"mystr\" is missing\n",
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
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
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
		ResBody: "Invalid extension(s): myext\n",
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
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
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
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
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
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
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
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
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
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
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
		ResBody: `Required property "myint7" is missing
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
		ResBody: `Invalid extension(s): myint7
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
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
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

	// "file" is ok this time because HasDocument=false
	rm.HasDocument = registry.PtrBool(false)
	xNoErr(t, reg.Model.VerifyAndSave())
	_, err = rm.AddAttribute(&registry.Attribute{
		Name: "mystring",
		Type: registry.STRING,
		IfValues: registry.IfValues{
			"foo": &registry.IfValue{
				SiblingAttributes: registry.Attributes{
					"file": &registry.Attribute{
						Name: "file",
						Type: registry.STRING,
					},
					"object": &registry.Attribute{
						Name: "object",
						Type: registry.OBJECT,
						Attributes: registry.Attributes{
							"objstr": &registry.Attribute{
								Name: "objstr",
								Type: registry.STRING,
								IfValues: registry.IfValues{
									"objval": {
										SiblingAttributes: registry.Attributes{
											"objint": &registry.Attribute{
												Name: "objint",
												Type: registry.INTEGER,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	xNoErr(t, err)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/vx", `{"id":"x"}`, 400,
		`Can't change the ID of an entity(vx->x)`+"\n")

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", "{}", 201, `{
  "id": "v1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "mystring": "hello"
}`, 200, `{
  "id": "v1",
  "epoch": 2,
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "mystring": "hello"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "file": "fff",
  "mystring": "hello",
  "object": {}
}`, 400, `Invalid extension(s): file,object
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "file": "fff",
  "mystring": "foo",
  "object": {}
}`, 200, `{
  "id": "v1",
  "epoch": 3,
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "file": "fff",
  "mystring": "foo",
  "object": {}
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "file": "fff",
  "mystring": "foo",
  "object": {
    "objstr": "ooo",
    "objint": 5
  }
}`, 400, `Invalid extension(s) in "object": objint
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "file": "fff",
  "mystring": "foo",
  "object": {
    "objstr": "objval",
    "objint": 5
  }
}`, 200, `{
  "id": "v1",
  "epoch": 4,
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "file": "fff",
  "mystring": "foo",
  "object": {
    "objint": 5,
    "objstr": "objval"
  }
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{
  "mystring": null
}`, 200, `{
  "id": "f1",
  "epoch": 5,
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "defaultversionid": "v1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "mystring": null
}`, 200, `{
  "id": "v1",
  "epoch": 6,
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z"
}
`)

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
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/d1/files/f1/versions/1",
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
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
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
		ResBody: `Attribute "mystr" must be a string
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
		ResBody: `Attribute "mystr" must be a string
`,
		ResHeaders: []string{},
	})
}

func TestHTTPDefault(t *testing.T) {
	reg := NewRegistry("TestHTTPDefault")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true /* L */, true, true)

	reg.AddGroup("dirs", "d1")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT file f1 - isdefault = true",
		URL:    "/dirs/d1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-isdefault: true",
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
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f1/versions",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "POST file f1 - no isdefault",
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
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT file f1/1 - setdefaultversionid = 1",
		URL:         "/dirs/d1/files/f1/versions/1?setdefaultversionid=1",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     `hello`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-id: 1",
			"xRegistry-epoch: 2",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT file f1/1 - setdefaultversionid = null, switches default",
		URL:         "/dirs/d1/files/f1/versions/1?setdefaultversionid=null",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     `hello`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-id: 1",
			"xRegistry-epoch: 3",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
		},
		ResBody: `hello`,
	})

	rm.SetSetStickyDefault(false)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT file f1/2 - setdefault=2 - diff server",
		URL:         "/dirs/d1/files/f1/versions/2?setdefaultversionid=2",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     `hello`,
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `Resource "files" doesn't allow setting of "defaultversionid"
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT file f1/1 - setdefault=1 - match server",
		URL:         "/dirs/d1/files/f1/versions/1?setdefaultversionid=1",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     `hello`,
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `Resource "files" doesn't allow setting of "defaultversionid"
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT file f1/1 - no setdefault",
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
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT file f1/2 - no setdefault",
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
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
		},
		ResBody: `hello`,
	})

	// Now test ?setdefaultversionid stuff
	///////////////////////////////

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1?setdefault= not allowed",
		URL:     "/dirs/d1/files/f1?setdefaultversionid",
		Method:  "POST",
		Code:    400,
		ResBody: `Resource "files" doesn't allow setting of "defaultversionid"` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1?meta&setdefault= not allowed",
		URL:     "/dirs/d1/files/f1?&meta&setdefaultversionid",
		Method:  "POST",
		Code:    400,
		ResBody: `Resource "files" doesn't allow setting of "defaultversionid"` + "\n",
	})

	// Enable client-side setting
	rm.SetSetStickyDefault(true)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1?setdefault - empty",
		URL:     "/dirs/d1/files/f1?setdefaultversionid",
		Method:  "POST",
		Code:    400,
		ResBody: `"setdefaultversionid" must not be empty` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1?setdefault= - empty",
		URL:     "/dirs/d1/files/f1?setdefaultversionid=",
		Method:  "POST",
		Code:    400,
		ResBody: `"setdefaultversionid" must not be empty` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1?meta&setdefault - empty",
		URL:     "/dirs/d1/files/f1?meta&setdefaultversionid",
		Method:  "POST",
		Code:    400,
		ResBody: `"setdefaultversionid" must not be empty` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1?meta&setdefault= - empty",
		URL:     "/dirs/d1/files/f1?meta&setdefaultversionid=",
		Method:  "POST",
		Code:    400,
		ResBody: `"setdefaultversionid" must not be empty` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file f1?setdefault=1 - no change",
		URL:    "/dirs/d1/files/f1?setdefaultversionid=1",
		Method: "POST",
		ReqHeaders: []string{
			`xRegistry-id: newone`,
		},
		ReqBody:     `pick me`,
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Type: text/plain; charset=utf-8",
			"xRegistry-id: newone",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/newone",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"Content-Length: 7",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/newone",
			"Location: http://localhost:8181/dirs/d1/files/f1/versions/newone",
		},
		ResBody: `pick me`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "POST file f1?meta&setdefault=1 - no change",
		URL:         "/dirs/d1/files/f1?meta&setdefaultversionid=1",
		Method:      "POST",
		ReqHeaders:  []string{},
		ReqBody:     ``,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Type: application/json",
		},
		ResBody: `{
  "id": "f1",
  "epoch": 4,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "stickydefaultversion": true,
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "versionscount": 3,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file f1?setdefault=1 - set to 2",
		URL:    "/dirs/d1/files/f1?setdefaultversionid=2",
		Method: "POST",
		ReqHeaders: []string{
			`xRegistry-id: bogus`,
		},
		ReqBody:     `ignore me`,
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Type: text/plain; charset=utf-8",
			"xRegistry-id: bogus",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/bogus",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"Content-Length: 9",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/bogus",
		},
		ResBody: `ignore me`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "POST file f1?setdefault=1 - set back to 1",
		URL:         "/dirs/d1/files/f1?meta&setdefaultversionid=1",
		Method:      "POST",
		ReqHeaders:  []string{},
		ReqBody:     ``, // must be empty for set to work
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Type: application/json",
		},
		ResBody: `{
  "id": "f1",
  "epoch": 4,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "stickydefaultversion": true,
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "versionscount": 4,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	// errors
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setdefault bad group type",
		URL:        "/badgroup/d1/files/f1?meta&setdefaultversionid=1",
		Method:     "POST",
		ReqHeaders: []string{`xRegistry-id: bogus`},
		Code:       404,
		ResHeaders: []string{"Content-Type: text/plain; charset=utf-8"},
		ResBody:    `Unknown Group type: badgroup` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setdefault bad group",
		URL:        "/dirs/dx/files/f1?meta&setdefaultversionid=1",
		Method:     "POST",
		ReqHeaders: []string{`xRegistry-id: bogus`},
		Code:       404,
		ResHeaders: []string{"Content-Type: text/plain; charset=utf-8"},
		ResBody:    `Group "dx" not found` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setdefault bad resource type",
		URL:        "/dirs/d1/badfiles/f1?meta&setdefaultversionid=1",
		Method:     "POST",
		ReqHeaders: []string{`xRegistry-id: bogus`},
		Code:       404,
		ResHeaders: []string{"Content-Type: text/plain; charset=utf-8"},
		ResBody:    `Unknown Resource type: badfiles` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setdefault bad resource",
		URL:        "/dirs/d1/files/xxf1?meta&setdefaultversionid=1",
		Method:     "POST",
		ReqHeaders: []string{`xRegistry-id: bogus`},
		Code:       404,
		ResHeaders: []string{"Content-Type: text/plain; charset=utf-8"},
		ResBody:    `Resource "xxf1" not found` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setdefault bad version",
		URL:        "/dirs/d1/files/f1?meta&setdefaultversionid=3",
		Method:     "POST",
		ReqHeaders: []string{`xRegistry-id: bogus`},
		Code:       400,
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
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/d1/files"
  },
  "d3": {
    "id": "d3",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d3",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/d3/files"
  },
  "d4": {
    "id": "d4",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d4",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

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
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/d1/files"
  },
  "d4": {
    "id": "d4",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d4",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

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
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

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
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/d2/files"
  },
  "d3": {
    "id": "d3",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d3",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/d3/files"
  },
  "d4": {
    "id": "d4",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d4",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

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
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filescount": 0,
    "filesurl": "http://localhost:8181/dirs/d2/files"
  },
  "d4": {
    "id": "d4",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d4",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

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
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

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
    "defaultversionid": "v6.1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f6/versions/v6.1?meta",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

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
	f1.AddVersion("v2")
	f1.AddVersion("v3")
	f1.AddVersion("v4")
	v5, _ := f1.AddVersion("v5")
	xNoErr(t, f1.SetDefault(v5))
	f1.AddVersion("v6")
	f1.AddVersion("v7")
	f1.AddVersion("v8")
	f1.AddVersion("v9")
	f1.AddVersion("v10")

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?meta", ``, 200,
		`{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "stickydefaultversion": true,
  "defaultversionid": "v5",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v5?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

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

	// Make sure we have some left, and default is still v5
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?meta&inline", "", 200, `{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "stickydefaultversion": true,
  "defaultversionid": "v5",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v5?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "versions": {
    "v10": {
      "id": "v10",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v10?meta",
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z"
    },
    "v3": {
      "id": "v3",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v3?meta",
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z"
    },
    "v5": {
      "id": "v5",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v5?meta",
      "isdefault": true,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z"
    },
    "v6": {
      "id": "v6",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v6?meta",
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z"
    },
    "v9": {
      "id": "v9",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v9?meta",
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z"
    }
  },
  "versionscount": 5,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v5?setdefaultversionid=v3",
		``, 204, "")

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v9?setdefaultversionid=v9",
		``, 400, "Can't set defaultversionid to Version being deleted\n")

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v9?setdefaultversionid=vx",
		``, 400, "Can't find next default Version \"vx\"\n")

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v9?setdefaultversionid=v3",
		``, 204, "")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v9?setdefaultversionid=vx",
		``, 404, "Version \"v9\" not found\n")

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?meta&inline", "", 200, `{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "stickydefaultversion": true,
  "defaultversionid": "v3",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "versions": {
    "v10": {
      "id": "v10",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v10?meta",
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z"
    },
    "v3": {
      "id": "v3",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v3?meta",
      "isdefault": true,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z"
    },
    "v6": {
      "id": "v6",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v6?meta",
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z"
    }
  },
  "versionscount": 3,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	f1.AddVersion("v1")
	// bad next
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions?setdefaultversionid=vx", `[{"id":"v6"}]`, 400, "Can't find next default Version \"vx\"\n")
	// next = being deleted
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions?setdefaultversionid=v6", `[{"id":"v6"}]`, 400, "Can't set defaultversionid to Version being deleted\n")

	// delete non-default, change default
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions?setdefaultversionid=v10", `[{"id":"v6"}]`, 204, "")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?meta&inline", "", 200, `{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "stickydefaultversion": true,
  "defaultversionid": "v10",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v10?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "versions": {
    "v1": {
      "id": "v1",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1?meta",
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z"
    },
    "v10": {
      "id": "v10",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v10?meta",
      "isdefault": true,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z"
    },
    "v3": {
      "id": "v3",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v3?meta",
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z"
    }
  },
  "versionscount": 3,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	// delete non-default, default not move
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions", `[{"id":"v3"}]`, 204, "")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?meta&inline", "", 200, `{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "stickydefaultversion": true,
  "defaultversionid": "v10",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v10?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "versions": {
    "v1": {
      "id": "v1",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1?meta",
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z"
    },
    "v10": {
      "id": "v10",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v10?meta",
      "isdefault": true,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z"
    }
  },
  "versionscount": 2,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions?setdefaultversionid=v1", `[]`, 204, "")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?meta&inline", "", 200, `{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "stickydefaultversion": true,
  "defaultversionid": "v1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "versions": {
    "v1": {
      "id": "v1",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1?meta",
      "isdefault": true,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z"
    },
    "v10": {
      "id": "v10",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v10?meta",
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z"
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
	err = reg.SetSave("description", "testing")
	xCheckErr(t, err, "Required property \"clireq1\" is missing")

	xNoErr(t, reg.JustSet("clireq1", "testing1"))
	xNoErr(t, reg.SetSave("description", "testing"))

	xHTTP(t, reg, "GET", "/", "", 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestHTTPRequiredFields",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "description": "testing",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "clireq1": "testing1",

  "dirscount": 0,
  "dirsurl": "http://localhost:8181/dirs"
}
`)

	// Groups
	xHTTP(t, reg, "PUT", "/dirs/d1", `{"description": "testing"}`, 400,
		`Required property "clireq2" is missing`+"\n")

	xHTTP(t, reg, "PUT", "/dirs/d1", `{
  "description": "testing",
  "clireq2": "testing2"
}`, 201, `{
  "id": "d1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1",
  "description": "testing",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "clireq2": "testing2",

  "filescount": 0,
  "filesurl": "http://localhost:8181/dirs/d1/files"
}
`)

	// Resources
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1?meta",
		`{"description": "testing"}`, 400,
		`Required property "clireq3" is missing`+"\n")

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1?meta", `{
  "description": "testingdesc3",
  "clireq3": "testing3"
}`, 201, `{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "description": "testingdesc3",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
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
		ResBody: `Required property "clireq3" is missing
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
		ResBody: `Required property "clireq3" is missing
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
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/d1/files/f2/versions/1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f2",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
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
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f2/versions/1?meta",
  "description": "desctesting3",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
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
		ResBody: "Required property \"clireq3\" is missing\n",
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
		ResBody: "Required property \"clireq3\" is missing\n",
	})
}

func TestHTTPHasDocumentFalse(t *testing.T) {
	reg := NewRegistry("TestHTTPHasDocumentFalse")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)

	// plural, singular, versions, verId bool, isDefault bool, hasDocument bool
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
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/bars/b1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/bars/b1/versions"
}
`)

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files",
		Method: "POST",
		ReqBody: `{
		  "ff1": {
		    "test":"foo"
		  }
		}`,
		Code: 200,
		ResBody: `{
  "ff1": {
    "id": "ff1",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d1/files/ff1",
    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/ff1/versions/1",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "test": "foo",

    "versionscount": 1,
    "versionsurl": "http://localhost:8181/dirs/d1/files/ff1/versions"
  }
}
`})

	// Make sure that each type of Resource (w/ and w/o hasdoc) has the
	// correct self/defaultversionurl URL (meaing w/ and w/o ?meta)
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1?inline",
		Method: "GET",

		Code: 200,
		ResBody: `{
  "id": "d1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "bars": {
    "b1": {
      "id": "b1",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/bars/b1?meta",
      "defaultversionid": "1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/bars/b1/versions/1?meta",
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",

      "versions": {
        "1": {
          "id": "1",
          "epoch": 1,
          "self": "http://localhost:8181/dirs/d1/bars/b1/versions/1?meta",
          "isdefault": true,
          "createdat": "2024-01-01T12:00:01Z",
          "modifiedat": "2024-01-01T12:00:01Z"
        }
      },
      "versionscount": 1,
      "versionsurl": "http://localhost:8181/dirs/d1/bars/b1/versions"
    }
  },
  "barscount": 1,
  "barsurl": "http://localhost:8181/dirs/d1/bars",
  "files": {
    "ff1": {
      "id": "ff1",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1/files/ff1",
      "defaultversionid": "1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/ff1/versions/1",
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",
      "test": "foo",

      "versions": {
        "1": {
          "id": "1",
          "epoch": 1,
          "self": "http://localhost:8181/dirs/d1/files/ff1/versions/1",
          "isdefault": true,
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",
          "test": "foo"
        }
      },
      "versionscount": 1,
      "versionsurl": "http://localhost:8181/dirs/d1/files/ff1/versions"
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
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
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
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "foo2": "test2",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1/files/f1", `{"2":{"id":"2","foo2":"test2"}}`, 200,
		`{
  "2": {
    "id": "2",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/2",
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "foo2": "test2"
  }
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{"3":{"id":"3","foo3":"test3"}}`, 200,
		`{
  "3": {
    "id": "3",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/3",
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "foo3": "test3"
  }
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/3", `{"foo3.1":"test3.1"}`, 200,
		`{
  "id": "3",
  "epoch": 2,
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/3",
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "foo3.1": "test3.1"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/4", `{"foo4":"test4"}`, 201,
		`{
  "id": "4",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/4",
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
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
      "immutable": true,
      "serverrequired": true
    },
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
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
    "createdat": {
      "name": "createdat",
      "type": "timestamp"
    },
    "modifiedat": {
      "name": "modifiedat",
      "type": "timestamp"
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
      "immutable": true,
      "serverrequired": true
    },
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
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
    "createdat": {
      "name": "createdat",
      "type": "timestamp"
    },
    "modifiedat": {
      "name": "modifiedat",
      "type": "timestamp"
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
      "immutable": true,
      "serverrequired": true
    },
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
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
    "createdat": {
      "name": "createdat",
      "type": "timestamp"
    },
    "modifiedat": {
      "name": "modifiedat",
      "type": "timestamp"
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
		Plural:           "files",
		Singular:         "file",
		MaxVersions:      0,
		SetVersionId:     registry.PtrBool(true),
		SetStickyDefault: registry.PtrBool(true),
		HasDocument:      registry.PtrBool(true),
		ReadOnly:         true,
	})
	xNoErr(t, err)

	xHTTP(t, reg, "PUT", "/dirs/dir1", "{}", 201, `{
  "id": "dir1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/dir1",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

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
    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "versionscount": 1,
    "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions"
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/dir1/files/f1/versions/v1?meta", "", 200, `{
  "id": "v1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z"
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

func TestDefaultVersionThis(t *testing.T) {
	reg := NewRegistry("TestDefaultVersionThis")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "create res?setdefault=this",
		URL:    "/dirs/d1/files/f1?setdefaultversionid=this",
		Method: "PUT",
		Code:   201,
		ResHeaders: []string{
			"xRegistry-id: f1",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-stickydefaultversion: true",
			"xRegistry-defaultversionid: 1",
			"xRegistry-defaultversionurl: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-versionscount: 1",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f1/versions",
			"Location: http://localhost:8181/dirs/d1/files/f1",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/1",
		},
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "create v2 - no setdef flag",
		URL:    "/dirs/d1/files/f1/versions/2",
		Method: "PUT",
		Code:   201,
		ResHeaders: []string{
			"xRegistry-id: 2",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/2",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"Location: http://localhost:8181/dirs/d1/files/f1/versions/2",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/2",
		},
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "check v1",
		URL:    "/dirs/d1/files/f1/versions/1",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-id: 1",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/1",
		},
	})

	xHTTP(t, reg, "POST", "/dirs/d1/files/f1?meta&setdefaultversionid", "", 400, `"setdefaultversionid" must not be empty`+"\n")
	xHTTP(t, reg, "POST", "/dirs/d1/files/f1?meta&setdefaultversionid=", "", 400, `"setdefaultversionid" must not be empty`+"\n")
	xHTTP(t, reg, "POST", "/dirs/d1/files/f1?meta&setdefaultversionid=this", "", 400, `Can't use 'this' if a version wasn't processed`+"\n")

	xHTTP(t, reg, "POST", "/dirs/d1/files/f1?setdefaultversionid", "", 400, `"setdefaultversionid" must not be empty`+"\n")
	xHTTP(t, reg, "POST", "/dirs/d1/files/f1?setdefaultversionid=", "", 400, `"setdefaultversionid" must not be empty`+"\n")

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?setdefaultversionid=this",
		Method: "POST",
		Code:   201,
		ResHeaders: []string{
			"xRegistry-id: 3",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/3",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/3",
		},
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1",
		Method: "POST",
		Code:   201,
		ResHeaders: []string{
			"xRegistry-id: 4",
			"xRegistry-epoch: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/4",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/4",
		},
	})

	// Just move sticky ptr
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?meta&setdefaultversionid=1",
		Method: "POST",
		Code:   200,
		ResBody: `{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "stickydefaultversion": true,
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "versionscount": 4,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	// delete version that's default
	xCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/versions/1",
		Method:  "DELETE",
		Code:    204,
		ResBody: "",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "check v1",
		URL:    "/dirs/d1/files/f1?meta",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "4",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/4?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "versionscount": 3,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/versions/4?setdefaultversionid=2",
		Method:  "DELETE",
		Code:    204,
		ResBody: "",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "check v1",
		URL:    "/dirs/d1/files/f1?meta",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "stickydefaultversion": true,
  "defaultversionid": "2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/2?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "versionscount": 2,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})
}

func TestHTTPContent(t *testing.T) {
	reg := NewRegistry("TestHTTPContent")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	reg.AddGroup("dirs", "d1")

	// Simple string
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?meta",
		Method: "PUT",
		ReqBody: `{
	"file": "hello"
}
`,
		Code: 201,
		ResBody: `{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "application/json",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-id:f1",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-defaultversionid:1",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount:1",
			"Content-Type:application/json",
			"Content-Length:5",
			"Content-Location:http://localhost:8181/dirs/d1/files/f1/versions/1",
		},
		ResBody: `hello`,
	})

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?meta&inline=file", "", 200, `{
  "id": "f1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "application/json",
  "file": "hello",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	// Escaped string
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?meta",
		Method: "PUT",
		ReqBody: `{
	"file": "\"hel\nlo"
}
`,
		Code: 200,
		ResBody: `{
  "id": "f1",
  "epoch": 2,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "application/json",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-id:f1",
			"xRegistry-epoch:2",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-defaultversionid:1",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount:1",
			"Content-Length:7",
			"Content-Location:http://localhost:8181/dirs/d1/files/f1/versions/1",
			"Content-Type:application/json",
		},
		ResBody: "\"hel\nlo",
	})

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?meta&inline=file", "", 200, `{
  "id": "f1",
  "epoch": 2,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "application/json",
  "file": "\"hel\nlo",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	// Pure JSON - map
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?meta",
		Method: "PUT",
		ReqBody: `{
	"contenttype": "application/json",
	"file": { "foo": "bar" }
}
`,
		Code: 200,
		ResBody: `{
  "id": "f1",
  "epoch": 3,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "application/json",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-id:f1",
			"xRegistry-epoch:3",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-defaultversionid:1",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount:1",
			"Content-Type:application/json",
			"Content-Length:13",
			"Content-Location:http://localhost:8181/dirs/d1/files/f1/versions/1",
		},
		ResBody: `{"foo":"bar"}`,
	})
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?meta&inline=file", "", 200, `{
  "id": "f1",
  "epoch": 3,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "application/json",
  "file": {
    "foo": "bar"
  },

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	// Pure JSON - array
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?meta",
		Method: "PUT",
		ReqBody: `{
	"contenttype": "application/json",
	"file": [ "hello", null, 5 ]
}
`,
		Code: 200,
		ResBody: `{
  "id": "f1",
  "epoch": 4,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "application/json",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-id:f1",
			"xRegistry-epoch:4",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-defaultversionid:1",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount:1",
			"Content-Type:application/json",
			"Content-Length:16",
			"Content-Location:http://localhost:8181/dirs/d1/files/f1/versions/1",
		},
		ResBody: `["hello",null,5]`,
	})
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?meta&inline=file", "", 200, `{
  "id": "f1",
  "epoch": 4,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "application/json",
  "file": [
    "hello",
    null,
    5
  ],

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	// Pure JSON - numeric
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?meta",
		Method: "PUT",
		ReqBody: `{
	"contenttype": "application/json",
	"file": 123
}
`,
		Code: 200,
		ResBody: `{
  "id": "f1",
  "epoch": 5,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "application/json",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-id:f1",
			"xRegistry-epoch:5",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-defaultversionid:1",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount:1",
			"Content-Type:application/json",
			"Content-Length:3",
			"Content-Location:http://localhost:8181/dirs/d1/files/f1/versions/1",
		},
		ResBody: `123`,
	})
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?meta&inline=file", "", 200, `{
  "id": "f1",
  "epoch": 5,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "application/json",
  "file": 123,

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	// base64 - string - with quotes
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?meta",
		Method: "PUT",
		ReqBody: `{
	"filebase64": "ImhlbGxvIgo="
}
`,
		Code: 200,
		ResBody: `{
  "id": "f1",
  "epoch": 6,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?meta&inline=file",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "id": "f1",
  "epoch": 6,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "filebase64": "ImhlbGxvIgo=",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	// base64 - string - w/o quotes
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?meta",
		Method: "PUT",
		ReqBody: `{
	"filebase64": "aGVsbG8K"
}
`,
		Code: 200,
		ResBody: `{
  "id": "f1",
  "epoch": 7,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?meta&inline=file",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "id": "f1",
  "epoch": 7,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "filebase64": "aGVsbG8K",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	// base64 - json
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?meta",
		Method: "PUT",
		ReqBody: `{
	"filebase64": "eyAiZm9vIjoiYmFyIjogfQo="
}
`,
		Code: 200,
		ResBody: `{
  "id": "f1",
  "epoch": 8,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?meta&inline=file",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "id": "f1",
  "epoch": 8,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "filebase64": "eyAiZm9vIjoiYmFyIjogfQo=",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	// base64 - empty
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?meta",
		Method: "PUT",
		ReqBody: `{
	"filebase64": ""
}
`,
		Code: 200,
		ResBody: `{
  "id": "f1",
  "epoch": 9,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?meta&inline=file",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "id": "f1",
  "epoch": 9,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	// base64 - null
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?meta",
		Method: "PUT",
		ReqBody: `{
	"filebase64": null
}
`,
		Code: 200,
		ResBody: `{
  "id": "f1",
  "epoch": 10,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?meta&inline=file",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "id": "f1",
  "epoch": 10,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	// file - null
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?meta",
		Method: "PUT",
		ReqBody: `{
	"file": null
}
`,
		Code: 200,
		ResBody: `{
  "id": "f1",
  "epoch": 11,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "application/json",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?meta&inline=file",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "id": "f1",
  "epoch": 11,
  "self": "http://localhost:8181/dirs/d1/files/f1?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "application/json",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`,
	})

	// Pure JSON - error
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?meta",
		Method: "PUT",
		ReqBody: `{
	"file": { bad bad json }
`,
		Code: 400,
		ResBody: `Syntax error at line 2: invalid character 'b' looking for beginning of object key string
`,
	})

	// New implied json - empty string
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f11?meta",
		Method: "PUT",
		ReqBody: `{
	"file": ""
}
`,
		Code: 201,
		ResBody: `{
  "id": "f11",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f11?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f11/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "application/json",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f11/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f11",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-id:f11",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f11",
			"xRegistry-defaultversionid:1",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/d1/files/f11/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f11/versions",
			"xRegistry-versionscount:1",
			"Content-Type:application/json",
			"Content-Length:0",
			"Content-Location:http://localhost:8181/dirs/d1/files/f11/versions/1",
		},
		ResBody: ``,
	})

	// New implied json - obj
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f12?meta",
		Method: "PUT",
		ReqBody: `{
	"file": { "foo": "bar" }
}
`,
		Code: 201,
		ResBody: `{
  "id": "f12",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f12?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f12/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "application/json",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f12/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f12",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-id:f12",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f12",
			"xRegistry-defaultversionid:1",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/d1/files/f12/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f12/versions",
			"xRegistry-versionscount:1",
			"Content-Type:application/json",
			"Content-Length:13",
			"Content-Location:http://localhost:8181/dirs/d1/files/f12/versions/1",
		},
		ResBody: `{"foo":"bar"}`,
	})

	// New implied json - numeric
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f13?meta",
		Method: "PUT",
		ReqBody: `{
	"file": 123
}
`,
		Code: 201,
		ResBody: `{
  "id": "f13",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f13?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f13/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "application/json",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f13/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f13",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-id:f13",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f13",
			"xRegistry-defaultversionid:1",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/d1/files/f13/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f13/versions",
			"xRegistry-versionscount:1",
			"Content-Type:application/json",
			"Content-Length:3",
			"Content-Location:http://localhost:8181/dirs/d1/files/f13/versions/1",
		},
		ResBody: `123`,
	})

	// New implied json - array
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f14?meta",
		Method: "PUT",
		ReqBody: `{
	"file": [ 123, 0 ]
}
`,
		Code: 201,
		ResBody: `{
  "id": "f14",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f14?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f14/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "application/json",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f14/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f14",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-id:f14",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f14",
			"xRegistry-defaultversionid:1",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/d1/files/f14/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f14/versions",
			"xRegistry-versionscount:1",
			"Content-Type:application/json",
			"Content-Length:7",
			"Content-Location:http://localhost:8181/dirs/d1/files/f14/versions/1",
		},
		ResBody: `[123,0]`,
	})

	// New implied json - bool
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f15?meta",
		Method: "PUT",
		ReqBody: `{
	"file": true
}
`,
		Code: 201,
		ResBody: `{
  "id": "f15",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f15?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f15/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "application/json",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f15/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f15",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-id:f15",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f15",
			"xRegistry-defaultversionid:1",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/d1/files/f15/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f15/versions",
			"xRegistry-versionscount:1",
			"Content-Type:application/json",
			"Content-Length:4",
			"Content-Location:http://localhost:8181/dirs/d1/files/f15/versions/1",
		},
		ResBody: `true`,
	})

	// New implied json - string
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f16?meta",
		Method: "PUT",
		ReqBody: `{
	"file": "he\tllo"
}
`,
		Code: 201,
		ResBody: `{
  "id": "f16",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f16?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f16/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "application/json",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f16/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f16",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-id:f16",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f16",
			"xRegistry-defaultversionid:1",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/d1/files/f16/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f16/versions",
			"xRegistry-versionscount:1",
			"Content-Type:application/json",
			"Content-Length:6",
			"Content-Location:http://localhost:8181/dirs/d1/files/f16/versions/1",
		},
		ResBody: "he\tllo",
	})

	// New unknown type
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f17?meta",
		Method: "PUT",
		ReqBody: `{
	"contenttype": "foo/bar",
	"file": "he\tllo"
}
`,
		Code: 201,
		ResBody: `{
  "id": "f17",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f17?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f17/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "foo/bar",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f17/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f17",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-id:f17",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f17",
			"xRegistry-defaultversionid:1",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/d1/files/f17/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f17/versions",
			"xRegistry-versionscount:1",
			"Content-Type:foo/bar",
			"Content-Length:6",
			"Content-Location:http://localhost:8181/dirs/d1/files/f17/versions/1",
		},
		ResBody: "he\tllo",
	})

	xHTTP(t, reg, "GET", "/dirs/d1/files/f17?meta&inline=file", ``, 200, `{
  "id": "f17",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f17?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f17/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "foo/bar",
  "filebase64": "aGUJbGxv",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f17/versions"
}
`)

	// New unknown type - contenttype:null
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f18?meta",
		Method: "PUT",
		ReqBody: `{
	"contenttype": null,
	"file": "he\tllo"
}
`,
		Code: 201,
		ResBody: `{
  "id": "f18",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f18?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f18/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f18/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f18",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-id:f18",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f18",
			"xRegistry-defaultversionid:1",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/d1/files/f18/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f18/versions",
			"xRegistry-versionscount:1",
			"Content-Length:6",
			"Content-Location:http://localhost:8181/dirs/d1/files/f18/versions/1",
		},
		ResBody: "he\tllo",
	})

	xHTTP(t, reg, "GET", "/dirs/d1/files/f18?meta&inline=file", ``, 200, `{
  "id": "f18",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/d1/files/f18?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f18/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "filebase64": "aGUJbGxv",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f18/versions"
}
`)

	// patch - contenttype:null
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f18?meta",
		Method: "PATCH",
		ReqBody: `{
	"contenttype": null,
	"file": "foo"
}
`,
		Code: 200,
		ResBody: `{
  "id": "f18",
  "epoch": 2,
  "self": "http://localhost:8181/dirs/d1/files/f18?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f18/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f18/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f18",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-id:f18",
			"xRegistry-epoch:2",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f18",
			"xRegistry-defaultversionid:1",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/d1/files/f18/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f18/versions",
			"xRegistry-versionscount:1",
			"Content-Length:3",
			"Content-Location:http://localhost:8181/dirs/d1/files/f18/versions/1",
		},
		ResBody: "foo",
	})

	xHTTP(t, reg, "GET", "/dirs/d1/files/f18?meta&inline=file", ``, 200, `{
  "id": "f18",
  "epoch": 2,
  "self": "http://localhost:8181/dirs/d1/files/f18?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f18/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "filebase64": "Zm9v",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f18/versions"
}
`)

	// patch - no ct saved, implied json, set ct
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f18?meta",
		Method: "PATCH",
		ReqBody: `{
	"file": "foo"
}
`,
		Code: 200,
		ResBody: `{
  "id": "f18",
  "epoch": 3,
  "self": "http://localhost:8181/dirs/d1/files/f18?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f18/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "application/json",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f18/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f18",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-id:f18",
			"xRegistry-epoch:3",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f18",
			"xRegistry-defaultversionid:1",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/d1/files/f18/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f18/versions",
			"xRegistry-versionscount:1",
			"Content-Length:3",
			"Content-Location:http://localhost:8181/dirs/d1/files/f18/versions/1",
			"Content-Type:application/json",
		},
		ResBody: "foo",
	})

	xHTTP(t, reg, "GET", "/dirs/d1/files/f18?meta&inline=file", ``, 200, `{
  "id": "f18",
  "epoch": 3,
  "self": "http://localhost:8181/dirs/d1/files/f18?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f18/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "application/json",
  "file": "foo",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f18/versions"
}
`)

	// patch - include odd ct
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f18?meta",
		Method: "PATCH",
		ReqBody: `{
	"contenttype": "foo/bar",
	"file": "bar"
}
`,
		Code: 200,
		ResBody: `{
  "id": "f18",
  "epoch": 4,
  "self": "http://localhost:8181/dirs/d1/files/f18?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f18/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "foo/bar",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f18/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f18",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-id:f18",
			"xRegistry-epoch:4",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f18",
			"xRegistry-defaultversionid:1",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/d1/files/f18/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f18/versions",
			"xRegistry-versionscount:1",
			"Content-Length:3",
			"Content-Location:http://localhost:8181/dirs/d1/files/f18/versions/1",
			"Content-Type:foo/bar",
		},
		ResBody: "bar",
	})

	xHTTP(t, reg, "GET", "/dirs/d1/files/f18?meta&inline=file", ``, 200, `{
  "id": "f18",
  "epoch": 4,
  "self": "http://localhost:8181/dirs/d1/files/f18?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f18/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "foo/bar",
  "filebase64": "YmFy",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f18/versions"
}
`)

	// patch - has ct, set ct to null
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f18?meta",
		Method: "PATCH",
		ReqBody: `{
	"contenttype": null
}
`,
		Code: 200,
		ResBody: `{
  "id": "f18",
  "epoch": 5,
  "self": "http://localhost:8181/dirs/d1/files/f18?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f18/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f18/versions"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f18",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-id:f18",
			"xRegistry-epoch:5",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f18",
			"xRegistry-defaultversionid:1",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/d1/files/f18/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f18/versions",
			"xRegistry-versionscount:1",
			"Content-Length:3",
			"Content-Location:http://localhost:8181/dirs/d1/files/f18/versions/1",
		},
		ResBody: "bar",
	})

	xHTTP(t, reg, "GET", "/dirs/d1/files/f18?meta&inline=file", ``, 200, `{
  "id": "f18",
  "epoch": 5,
  "self": "http://localhost:8181/dirs/d1/files/f18?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f18/versions/1?meta",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "filebase64": "YmFy",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f18/versions"
}
`)

}

func TestHTTPResourcesBulk(t *testing.T) {
	reg := NewRegistry("TestHTTPResourcesBulk")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)
	reg.AddGroup("dirs", "dir1")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources - missing id",
		URL:        "/dirs/dir1/files",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    ``,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody: `A "xRegistry-id" header must be provided
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST resources - with id",
		URL:    "/dirs/dir1/files",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-id: f1",
		},
		ReqBody: ``,
		Code:    201,
		ResHeaders: []string{
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/dir1/files/f1/versions/1",
			"xRegistry-id:f1",
			"xRegistry-versionscount:1",
			"xRegistry-versionsurl:http://localhost:8181/dirs/dir1/files/f1/versions",
			"xRegistry-defaultversionid:1",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f1/versions/1",
			"Location:http://localhost:8181/dirs/dir1/files/f1",
			"Content-Length:0",
		},
		ResBody: ``,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources?meta - empty",
		URL:        "/dirs/dir1/files?meta",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources?meta - {}",
		URL:        "/dirs/dir1/files?meta",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    `{}`,
		Code:       200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources?meta - one, empty",
		URL:        "/dirs/dir1/files?meta",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
		  "f2": {}
        }`,
		Code: 200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{
  "f2": {
    "id": "f2",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/dir1/files/f2?meta",
    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/dir1/files/f2/versions/1?meta",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "versionscount": 1,
    "versionsurl": "http://localhost:8181/dirs/dir1/files/f2/versions"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources?meta - one, update",
		URL:        "/dirs/dir1/files?meta",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
		  "f2": {
            "description": "foo"
          }
        }`,
		Code: 200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{
  "f2": {
    "id": "f2",
    "epoch": 2,
    "self": "http://localhost:8181/dirs/dir1/files/f2?meta",
    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/dir1/files/f2/versions/1?meta",
    "description": "foo",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",

    "versionscount": 1,
    "versionsurl": "http://localhost:8181/dirs/dir1/files/f2/versions"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources?meta - two, update+create, bad ext",
		URL:        "/dirs/dir1/files?meta",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
		  "f2": {
            "description": "hello"
          },
          "f3": {
            "foo": "bar"
          }
        }`,
		Code:       400,
		ResHeaders: []string{},
		ResBody: `Invalid extension(s): foo
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources?meta - two, update+create",
		URL:        "/dirs/dir1/files?meta",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
		  "f2": {
            "description": "foo"
          },
		  "f3": {
		    "labels": {
			  "l1": "hello"
			}
          }
        }`,
		Code: 200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{
  "f2": {
    "id": "f2",
    "epoch": 3,
    "self": "http://localhost:8181/dirs/dir1/files/f2?meta",
    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/dir1/files/f2/versions/1?meta",
    "description": "foo",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",

    "versionscount": 1,
    "versionsurl": "http://localhost:8181/dirs/dir1/files/f2/versions"
  },
  "f3": {
    "id": "f3",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/dir1/files/f3?meta",
    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/dir1/files/f3/versions/1?meta",
    "labels": {
      "l1": "hello"
    },
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:02Z",

    "versionscount": 1,
    "versionsurl": "http://localhost:8181/dirs/dir1/files/f3/versions"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources/f1?meta - two, bad metadata",
		URL:        "/dirs/dir1/files/f1?meta",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
		  "f2": {
            "description": "foo"
          },
		  "f3": {
		    "labels": {
			  "l1": "hello"
			}
          }
        }`,
		Code: 400,
		ResHeaders: []string{
			"Content-Type:text/plain; charset=utf-8",
		},
		ResBody: `Invalid extension(s): f2,f3
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources/f4?meta - new resource - bad id",
		URL:        "/dirs/dir1/files/f4?meta",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
          "id": "f5",
          "description": "my f5"
        }`,
		Code: 400,
		ResHeaders: []string{
			"Content-Type:text/plain; charset=utf-8",
		},
		ResBody: `Metadata id(f5) doesn't match ID in URL(f4)
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources/f4?meta - new resource",
		URL:        "/dirs/dir1/files/f4?meta",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
          "id": "f4",
          "description": "my f4",
          "file": "hello"
        }`,
		Code: 201,
		ResHeaders: []string{
			"Content-Type:application/json",
			"Location:http://localhost:8181/dirs/dir1/files/f4?meta",
		},
		ResBody: `{
  "id": "f4",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/dir1/files/f4?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/dir1/files/f4/versions/1?meta",
  "description": "my f4",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "application/json",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f4/versions"
}
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET resources/f4 - check doc",
		URL:        "/dirs/dir1/files/f4",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{
			"xRegistry-id:f4",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f4",
			"xRegistry-description:my f4",
			"xRegistry-defaultversionid:1",
			"xRegistry-defaultversionurl:http://localhost:8181/dirs/dir1/files/f4/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-versionscount:1",
			"xRegistry-versionsurl:http://localhost:8181/dirs/dir1/files/f4/versions",
			"Content-Length:5",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f4/versions/1",
			"Content-Type:application/json",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources/f4?meta - new resource - no id",
		URL:        "/dirs/dir1/files/f5?meta",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
          "description": "my f5"
        }`,
		Code: 201,
		ResHeaders: []string{
			"Content-Type:application/json",
			"Location:http://localhost:8181/dirs/dir1/files/f5?meta",
		},
		ResBody: `{
  "id": "f5",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/dir1/files/f5?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/dir1/files/f5/versions/1?meta",
  "description": "my f5",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f5/versions"
}
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST resources/f6 - new res,version - no id",
		URL:    "/dirs/dir1/files/f6",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-description: my f6",
		},
		ReqBody: `hello`,
		Code:    201,
		ResHeaders: []string{
			"xRegistry-id:1",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f6/versions/1",
			"xRegistry-description:my f6",
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"Content-Length:5",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f6/versions/1",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST resources/f61 - new res,version - no id+setdef=null",
		URL:    "/dirs/dir1/files/f61?setdefaultversionid=null",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-description: my f61",
		},
		ReqBody: `hello`,
		Code:    201,
		ResHeaders: []string{
			"xRegistry-id:1",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f61/versions/1",
			"xRegistry-description:my f61",
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"Content-Length:5",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f61/versions/1",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST resources/f62 - new res,version - no id+setdef=this",
		URL:    "/dirs/dir1/files/f62?setdefaultversionid=this",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-description: my f62",
		},
		ReqBody: `hello`,
		Code:    201,
		ResHeaders: []string{
			"xRegistry-id:1",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f62/versions/1",
			"xRegistry-description:my f62",
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"Content-Length:5",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f62/versions/1",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST resources/f63 - new res,version - no id+setdef=1",
		URL:    "/dirs/dir1/files/f63?setdefaultversionid=1",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-description: my f63",
		},
		ReqBody: `hello`,
		Code:    201,
		ResHeaders: []string{
			"xRegistry-id:1",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f63/versions/1",
			"xRegistry-description:my f63",
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"Content-Length:5",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f63/versions/1",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST resources/f7 - new res,version - with id",
		URL:    "/dirs/dir1/files/f7",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-description: my f7",
			"xRegistry-id: v1",
		},
		ReqBody: `hello`,
		Code:    201,
		ResHeaders: []string{
			"xRegistry-id:v1",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f7/versions/v1",
			"xRegistry-description:my f7",
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"Content-Length:5",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f7/versions/v1",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST resources/f8?meta - new res,version - extra headers",
		URL:    "/dirs/dir1/files/f8?meta",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-description: my f8",
		},
		ReqBody:    `hello`,
		Code:       400,
		ResHeaders: []string{},
		ResBody: `Including "xRegistry" headers when "?meta" is used is invalid
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f9?meta - new res,version - empty",
		URL:        "/dirs/dir1/files/f9?meta",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    ``,
		Code:       400,
		ResHeaders: []string{
			"Content-Type:text/plain; charset=utf-8",
		},
		ResBody: `Set of Versions to add can't be empty
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f9?meta - new res,version - empty",
		URL:        "/dirs/dir1/files/f9?meta",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    `{}`,
		Code:       400,
		ResHeaders: []string{
			"Content-Type:text/plain; charset=utf-8",
		},
		ResBody: `Set of Versions to add can't be empty
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f9/versions?meta - new res,version - empty",
		URL:        "/dirs/dir1/files/f9/versions?meta",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    ``,
		Code:       400,
		ResHeaders: []string{
			"Content-Type:text/plain; charset=utf-8",
		},
		ResBody: `Set of Versions to add can't be empty
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f9/versions?meta - new res,version - empty",
		URL:        "/dirs/dir1/files/f9/versions?meta",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    `{}`,
		Code:       400,
		ResHeaders: []string{
			"Content-Type:text/plain; charset=utf-8",
		},
		ResBody: `Set of Versions to add can't be empty
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f9/versions?meta - new res,version-1v",
		URL:        "/dirs/dir1/files/f9/versions?meta",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v1": {}
        }`,
		Code: 200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{
  "v1": {
    "id": "v1",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/dir1/files/f9/versions/v1?meta",
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f10/versions?meta - new res,version-2v,err",
		URL:        "/dirs/dir1/files/f10/versions?meta&setdefaultversionid=null",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v1": {},
          "v2": {}
        }`,
		Code: 400,
		ResHeaders: []string{
			"Content-Type:text/plain; charset=utf-8",
		},
		ResBody: `?setdefaultversionid can not be 'null'
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f10/versions?meta - new res,version-2v,err",
		URL:        "/dirs/dir1/files/f10/versions?meta&setdefaultversionid=this",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v1": {},
          "v2": {}
        }`,
		Code: 400,
		ResHeaders: []string{
			"Content-Type:text/plain; charset=utf-8",
		},
		ResBody: `?setdefaultversionid can not be 'this'
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f10a/versions?meta - new res,version-2v,err",
		URL:        "/dirs/dir1/files/f10a/versions?meta&setdefaultversionid=this",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v1": {}
        }`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "v1": {
    "id": "v1",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/dir1/files/f10a/versions/v1?meta",
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f10/versions?meta - new res,version-2v",
		URL:        "/dirs/dir1/files/f10/versions?meta",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v1": {},
          "v2": {}
        }`,
		Code: 400,
		ResHeaders: []string{
			"Content-Type:text/plain; charset=utf-8",
		},
		ResBody: `?setdefaultversionid is required
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f10/versions?meta - new res,version-1v",
		URL:        "/dirs/dir1/files/f10/versions?meta&setdefaultversionid=v2",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v1": {}
        }`,
		Code: 400,
		ResHeaders: []string{
			"Content-Type:text/plain; charset=utf-8",
		},
		ResBody: `Version "v2" not found
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f10/versions?meta - new res,version-1v",
		URL:        "/dirs/dir1/files/f10/versions?meta&setdefaultversionid=v1",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v1": {}
        }`,
		Code: 200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{
  "v1": {
    "id": "v1",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/dir1/files/f10/versions/v1?meta",
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f11/versions?meta - new res,version-2v",
		URL:        "/dirs/dir1/files/f11/versions?meta&setdefaultversionid=v1",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v1": {},
          "v2": {}
        }`,
		Code: 200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{
  "v1": {
    "id": "v1",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/dir1/files/f11/versions/v1?meta",
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z"
  },
  "v2": {
    "id": "v2",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/dir1/files/f11/versions/v2?meta",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f12/versions?meta - new res,version-2v",
		URL:        "/dirs/dir1/files/f12/versions?meta&setdefaultversionid=v2",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v1": {},
          "v2": {}
        }`,
		Code: 200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{
  "v1": {
    "id": "v1",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/dir1/files/f12/versions/v1?meta",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z"
  },
  "v2": {
    "id": "v2",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/dir1/files/f12/versions/v2?meta",
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f12/versions?meta - update,add v",
		URL:        "/dirs/dir1/files/f12/versions?meta&setdefaultversionid=v1",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v3": { "description": "my v3"},
          "v1": { "description": "my v1"},
          "v2": { "description": "my v2"}
        }`,
		Code: 200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{
  "v1": {
    "id": "v1",
    "epoch": 2,
    "self": "http://localhost:8181/dirs/dir1/files/f12/versions/v1?meta",
    "isdefault": true,
    "description": "my v1",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z"
  },
  "v2": {
    "id": "v2",
    "epoch": 2,
    "self": "http://localhost:8181/dirs/dir1/files/f12/versions/v2?meta",
    "description": "my v2",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z"
  },
  "v3": {
    "id": "v3",
    "epoch": 1,
    "self": "http://localhost:8181/dirs/dir1/files/f12/versions/v3?meta",
    "description": "my v3",
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:02Z"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f13/versions - new+doc",
		URL:        "/dirs/dir1/files/f13/versions",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    `hello`,
		Code:       201,
		ResHeaders: []string{
			"xRegistry-id:1",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f13/versions/1",
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"Content-Length:5",
			"Location:http://localhost:8181/dirs/dir1/files/f13/versions/1",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f13/versions/1",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f13/versions - new v+doc",
		URL:        "/dirs/dir1/files/f13/versions",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    `hello2`,
		Code:       201,
		ResHeaders: []string{
			"xRegistry-id:2",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f13/versions/2",
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"Content-Length:6",
			"Location:http://localhost:8181/dirs/dir1/files/f13/versions/2",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f13/versions/2",
		},
		ResBody: `hello2`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST resources/f13/versions - update+doc",
		URL:    "/dirs/dir1/files/f13/versions",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-id: 1",
		},
		ReqBody: `bye`,
		Code:    200,
		ResHeaders: []string{
			"xRegistry-id:1",
			"xRegistry-epoch:2",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f13/versions/1",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"Content-Length:3",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f13/versions/1",
		},
		ResBody: `bye`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources/f13/versions - new v+doc, err",
		URL:    "/dirs/dir1/files/f13/versions/3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-id: 33",
		},
		ReqBody:    `v3`,
		Code:       400,
		ResHeaders: []string{},
		ResBody: `Can't change the ID of an entity(3->33)
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources/f13/versions - new v+doc+id",
		URL:    "/dirs/dir1/files/f13/versions/3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-id: 3",
		},
		ReqBody: `v3`,
		Code:    201,
		ResHeaders: []string{
			"xRegistry-id:3",
			"xRegistry-epoch:1",
			"xRegistry-isdefault:true",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f13/versions/3",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"Content-Length:2",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f13/versions/3",
		},
		ResBody: `v3`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources/f13/versions - new v+doc+no id",
		URL:        "/dirs/dir1/files/f13/versions/4",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `v4`,
		Code:       201,
		ResHeaders: []string{
			"xRegistry-id:4",
			"xRegistry-epoch:1",
			"xRegistry-isdefault:true",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f13/versions/4",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"Content-Length:2",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f13/versions/4",
		},
		ResBody: `v4`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources/f13/versions + meta - empty",
		URL:        "/dirs/dir1/files/f13/versions/5?meta",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    ``,
		Code:       201,
		ResHeaders: []string{},
		ResBody: `{
  "id": "5",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/dir1/files/f13/versions/5?meta",
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources/f13/versions + meta - {}",
		URL:        "/dirs/dir1/files/f13/versions/6?meta",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{}`,
		Code:       201,
		ResHeaders: []string{},
		ResBody: `{
  "id": "6",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/dir1/files/f13/versions/6?meta",
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources/f13/versions + meta - {} again",
		URL:        "/dirs/dir1/files/f13/versions/6?meta",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "id": "6",
  "epoch": 2,
  "self": "http://localhost:8181/dirs/dir1/files/f13/versions/6?meta",
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources/f13/versions + meta - bad id update",
		URL:        "/dirs/dir1/files/f13/versions/7?meta",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{ "id": "77" }`,
		Code:       400,
		ResHeaders: []string{},
		ResBody: `Can't change the ID of an entity(7->77)
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f14/versions - new+doc+setdef=null",
		URL:        "/dirs/dir1/files/f14/versions?setdefaultversionid=null",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    `hello`,
		Code:       201,
		ResHeaders: []string{
			"xRegistry-id:1",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f14/versions/1",
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"Content-Length:5",
			"Location:http://localhost:8181/dirs/dir1/files/f14/versions/1",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f14/versions/1",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f141/versions - new+doc+setdef=this",
		URL:        "/dirs/dir1/files/f141/versions?setdefaultversionid=this",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    `hello`,
		Code:       201,
		ResHeaders: []string{
			"xRegistry-id:1",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f141/versions/1",
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"Content-Length:5",
			"Location:http://localhost:8181/dirs/dir1/files/f141/versions/1",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f141/versions/1",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f142/versions - new+doc+setdef=1",
		URL:        "/dirs/dir1/files/f142/versions?setdefaultversionid=1",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    `hello`,
		Code:       201,
		ResHeaders: []string{
			"xRegistry-id:1",
			"xRegistry-epoch:1",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f142/versions/1",
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"Content-Length:5",
			"Location:http://localhost:8181/dirs/dir1/files/f142/versions/1",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f142/versions/1",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f143/versions - new+doc+setdef=bad",
		URL:        "/dirs/dir1/files/f143/versions?setdefaultversionid=2",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    `hello`,
		Code:       400,
		ResHeaders: []string{
			"Content-Type: text/plain; charset=utf-8",
		},
		ResBody: `Version "2" not found
`,
	})

}

func TestHTTPRegistryPatch(t *testing.T) {
	reg := NewRegistry("TestHTTPRegistryPatch")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	reg.Model.AddAttr("regext", registry.STRING)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddAttr("gext", registry.STRING)

	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, true)
	rm.AddAttr("rext", registry.STRING)

	g, _ := reg.AddGroup("dirs", "dir1")
	f, err := g.AddResource("files", "f1", "v1")

	xNoErr(t, err)

	reg.Commit()
	reg.Refresh()
	regMod := reg.GetAsString("modifiedat")

	// Test PATCHing the Registry

	// skip timestamp masking (the "--")
	xHTTP(t, reg, "GET", "/", ``, 200, `--{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestHTTPRegistryPatch",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "createdat": "`+regMod+`",
  "modifiedat": "`+regMod+`",

  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
}
`)

	xHTTP(t, reg, "PATCH", "/", `{}`, 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestHTTPRegistryPatch",
  "epoch": 2,
  "self": "http://localhost:8181/",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
}
`)

	reg.Refresh()
	xCheck(t, reg.GetAsString("modifiedat") != regMod, "Should be diff")

	xHTTP(t, reg, "PATCH", "/", `{
	  "description": "testing"
	}`, 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestHTTPRegistryPatch",
  "epoch": 3,
  "self": "http://localhost:8181/",
  "description": "testing",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
}
`)

	xHTTP(t, reg, "PATCH", "/", `{
	  "description": null
	}`, 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestHTTPRegistryPatch",
  "epoch": 4,
  "self": "http://localhost:8181/",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
}
`)

	xHTTP(t, reg, "PATCH", "/", `{
	  "labels": {
	    "foo": "bar"
	  },
	  "createdat": null
	}`, 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestHTTPRegistryPatch",
  "epoch": 5,
  "self": "http://localhost:8181/",
  "labels": {
    "foo": "bar"
  },
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",

  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
}
`)

	xHTTP(t, reg, "PATCH", "/", `{
	  "labels": {}
	}`, 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestHTTPRegistryPatch",
  "epoch": 6,
  "self": "http://localhost:8181/",
  "labels": {},
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
}
`)

	xHTTP(t, reg, "PATCH", "/", `{
	  "labels": null
	}`, 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestHTTPRegistryPatch",
  "epoch": 7,
  "self": "http://localhost:8181/",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
}
`)

	xHTTP(t, reg, "PATCH", "/", `{
	  "regext": "str"
	}`, 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestHTTPRegistryPatch",
  "epoch": 8,
  "self": "http://localhost:8181/",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "regext": "str",

  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
}
`)

	xHTTP(t, reg, "PATCH", "/", `{
	  "regext": null
	}`, 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "id": "TestHTTPRegistryPatch",
  "epoch": 9,
  "self": "http://localhost:8181/",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
}
`)

	xHTTP(t, reg, "PATCH", "/", `{
	  "badext": "str"
	}`, 400, `Invalid extension(s): badext
`)

	// Test PATCHing a Group
	// //////////////////////////////////////////////////////

	gmod := g.GetAsString("modifiedat")

	xHTTP(t, reg, "PATCH", "/dirs", `{}`, 405,
		`PATCH not allowed on collections
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1", `{}`, 200, `{
  "id": "dir1",
  "epoch": 2,
  "self": "http://localhost:8181/dirs/dir1",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filescount": 1,
  "filesurl": "http://localhost:8181/dirs/dir1/files"
}
`)

	g.Refresh()
	xCheck(t, g.GetAsString("modifiedat") != gmod, "Should be diff")

	xHTTP(t, reg, "PATCH", "/dirs/dir1", `{
	  "description": "testing"
	}`, 200, `{
  "id": "dir1",
  "epoch": 3,
  "self": "http://localhost:8181/dirs/dir1",
  "description": "testing",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filescount": 1,
  "filesurl": "http://localhost:8181/dirs/dir1/files"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1", `{
	  "description": null
	}`, 200, `{
  "id": "dir1",
  "epoch": 4,
  "self": "http://localhost:8181/dirs/dir1",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filescount": 1,
  "filesurl": "http://localhost:8181/dirs/dir1/files"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1", `{
	  "labels": {
	    "foo": "bar"
	  },
	  "createdat": null
	}`, 200, `{
  "id": "dir1",
  "epoch": 5,
  "self": "http://localhost:8181/dirs/dir1",
  "labels": {
    "foo": "bar"
  },
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",

  "filescount": 1,
  "filesurl": "http://localhost:8181/dirs/dir1/files"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1", `{
	  "labels": {}
	}`, 200, `{
  "id": "dir1",
  "epoch": 6,
  "self": "http://localhost:8181/dirs/dir1",
  "labels": {},
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filescount": 1,
  "filesurl": "http://localhost:8181/dirs/dir1/files"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1", `{
	  "labels": null
	}`, 200, `{
  "id": "dir1",
  "epoch": 7,
  "self": "http://localhost:8181/dirs/dir1",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filescount": 1,
  "filesurl": "http://localhost:8181/dirs/dir1/files"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1", `{
	  "gext": "str"
	}`, 200, `{
  "id": "dir1",
  "epoch": 8,
  "self": "http://localhost:8181/dirs/dir1",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "gext": "str",

  "filescount": 1,
  "filesurl": "http://localhost:8181/dirs/dir1/files"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1", `{
	  "gext": null
	}`, 200, `{
  "id": "dir1",
  "epoch": 9,
  "self": "http://localhost:8181/dirs/dir1",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filescount": 1,
  "filesurl": "http://localhost:8181/dirs/dir1/files"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1", `{
	  "badext": "str"
	}`, 400, `Invalid extension(s): badext
`)

	// Test PATCHing a Resource
	// //////////////////////////////////////////////////////

	f.Refresh()
	v, _ := f.GetDefault()
	vmod := v.GetAsString("modifiedat")

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files", `{}`, 405,
		`PATCH not allowed on collections
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1", ``, 400,
		`PATCH is not allowed on Resource documents
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1?meta", `{}`, 200, `{
  "id": "f1",
  "epoch": 2,
  "self": "http://localhost:8181/dirs/dir1/files/f1?meta",
  "defaultversionid": "v1",
  "defaultversionurl": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions"
}
`)

	v.Refresh()
	xCheck(t, v.GetAsString("modifiedat") != vmod, "Should be diff")

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1?meta", `{
	  "description": "testing"
	}`, 200, `{
  "id": "f1",
  "epoch": 3,
  "self": "http://localhost:8181/dirs/dir1/files/f1?meta",
  "defaultversionid": "v1",
  "defaultversionurl": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "description": "testing",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1?meta", `{
	  "description": null
	}`, 200, `{
  "id": "f1",
  "epoch": 4,
  "self": "http://localhost:8181/dirs/dir1/files/f1?meta",
  "defaultversionid": "v1",
  "defaultversionurl": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1?meta", `{
	  "labels": {
	    "foo": "bar"
	  },
	  "createdat": null
	}`, 200, `{
  "id": "f1",
  "epoch": 5,
  "self": "http://localhost:8181/dirs/dir1/files/f1?meta",
  "defaultversionid": "v1",
  "defaultversionurl": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "labels": {
    "foo": "bar"
  },
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1?meta", `{
	  "labels": {}
	}`, 200, `{
  "id": "f1",
  "epoch": 6,
  "self": "http://localhost:8181/dirs/dir1/files/f1?meta",
  "defaultversionid": "v1",
  "defaultversionurl": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "labels": {},
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1?meta", `{
	  "labels": null
	}`, 200, `{
  "id": "f1",
  "epoch": 7,
  "self": "http://localhost:8181/dirs/dir1/files/f1?meta",
  "defaultversionid": "v1",
  "defaultversionurl": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1?meta", `{
	  "rext": "str"
	}`, 200, `{
  "id": "f1",
  "epoch": 8,
  "self": "http://localhost:8181/dirs/dir1/files/f1?meta",
  "defaultversionid": "v1",
  "defaultversionurl": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "rext": "str",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1?meta", `{
	  "rext": null
	}`, 200, `{
  "id": "f1",
  "epoch": 9,
  "self": "http://localhost:8181/dirs/dir1/files/f1?meta",
  "defaultversionid": "v1",
  "defaultversionurl": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1?meta", `{
	  "badext": "str"
	}`, 400, `Invalid extension(s): badext
`)

	// Test PATCHing a Version
	// //////////////////////////////////////////////////////

	f.Refresh()
	v, _ = f.GetDefault()
	vmod = v.GetAsString("modifiedat")

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions", `{}`, 405,
		`PATCH not allowed on collections
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1", ``, 400,
		`PATCH is not allowed on Resource documents
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1?meta", `{}`, 200, `{
  "id": "v1",
  "epoch": 10,
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`)

	v.Refresh()
	xCheck(t, v.GetAsString("modifiedat") != vmod, "Should be diff")

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1?meta", `{
	  "description": "testing"
	}`, 200, `{
  "id": "v1",
  "epoch": 11,
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "isdefault": true,
  "description": "testing",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1?meta", `{
	  "description": null
	}`, 200, `{
  "id": "v1",
  "epoch": 12,
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1?meta", `{
	  "labels": {
	    "foo": "bar"
	  },
	  "createdat": null
	}`, 200, `{
  "id": "v1",
  "epoch": 13,
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "isdefault": true,
  "labels": {
    "foo": "bar"
  },
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1?meta", `{
	  "labels": {}
	}`, 200, `{
  "id": "v1",
  "epoch": 14,
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "isdefault": true,
  "labels": {},
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1?meta", `{
	  "labels": null
	}`, 200, `{
  "id": "v1",
  "epoch": 15,
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1?meta", `{
	  "rext": "str"
	}`, 200, `{
  "id": "v1",
  "epoch": 16,
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "rext": "str"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1?meta", `{
	  "rext": null
	}`, 200, `{
  "id": "v1",
  "epoch": 17,
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1?meta", `{
	  "badext": "str"
	}`, 400, `Invalid extension(s): badext
`)

	// Test that PATCH can be used to create stuff too

	xHTTP(t, reg, "PATCH", "/dirs/dir2", `{}`, 201, `{
  "id": "dir2",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/dir2",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",

  "filescount": 0,
  "filesurl": "http://localhost:8181/dirs/dir2/files"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir2/files/f2?meta", `{}`, 201, `{
  "id": "f2",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/dir2/files/f2?meta",
  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/dir2/files/f2/versions/1?meta",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/dir2/files/f2/versions"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir2/files/f2/versions/v2?meta", `{}`, 201, `{
  "id": "v2",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/dir2/files/f2/versions/v2?meta",
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir3/files/f3/versions/v3?meta", `{}`, 201, `{
  "id": "v3",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/dir3/files/f3/versions/v3?meta",
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z"
}
`)

}

func TestHTTPEpoch(t *testing.T) {
	reg := NewRegistry("TestHTTPRegistryPatch")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1?meta", `{}`,
		201, `{
  "id": "v1",
  "epoch": 1,
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1",
		`{"epoch":null}`, 200, `{
  "id": "dir1",
  "epoch": 2,
  "self": "http://localhost:8181/dirs/dir1",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filescount": 1,
  "filesurl": "http://localhost:8181/dirs/dir1/files"
}
`)
	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1?meta",
		`{"epoch":null}`, 200, `{
  "id": "f1",
  "epoch": 2,
  "self": "http://localhost:8181/dirs/dir1/files/f1?meta",
  "defaultversionid": "v1",
  "defaultversionurl": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions"
}
`)
	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1?meta",
		`{"epoch":null}`, 200, `{
  "id": "v1",
  "epoch": 3,
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1?meta",
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`)

}

func TestHTTPRegistryPatchNoDoc(t *testing.T) {
	reg := NewRegistry("TestHTTPRegistryPatchNoDoc")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, false)

	g, _ := reg.AddGroup("dirs", "dir1")
	_, err := g.AddResource("files", "f1", "v1")

	xNoErr(t, err)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1?meta",
		`{}`, 400, `Specifying "?meta" for a Resource that has the model "hasdocument" value set to "false" is invalid
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1",
		`{"description": "desc"}`, 200, `{
  "id": "f1",
  "epoch": 2,
  "self": "http://localhost:8181/dirs/dir1/files/f1",
  "defaultversionid": "v1",
  "defaultversionurl": "http://localhost:8181/dirs/dir1/files/f1/versions/v1",
  "description": "desc",
  "createdat": "YYYY-MM-DDTHH:MM:1Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:2Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1",
		`{"description": null}`, 200, `{
  "id": "f1",
  "epoch": 3,
  "self": "http://localhost:8181/dirs/dir1/files/f1",
  "defaultversionid": "v1",
  "defaultversionurl": "http://localhost:8181/dirs/dir1/files/f1/versions/v1",
  "createdat": "YYYY-MM-DDTHH:MM:1Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:2Z",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1?meta",
		`{}`, 400, `Specifying "?meta" for a Resource that has the model "hasdocument" value set to "false" is invalid
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1",
		`{"description": "desc"}`, 200, `{
  "id": "v1",
  "epoch": 4,
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1",
  "isdefault": true,
  "description": "desc",
  "createdat": "YYYY-MM-DDTHH:MM:1Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:2Z"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1",
		`{"description": null}`, 200, `{
  "id": "v1",
  "epoch": 5,
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1",
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:1Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:2Z"
}
`)

}
