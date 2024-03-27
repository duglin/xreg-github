package tests

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestBasicInline(t *testing.T) {
	reg := NewRegistry("TestBasicInline")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	d, _ := reg.AddGroup("dirs", "d1")
	f, _ := d.AddResource("files", "f1", "v1")
	f.AddVersion("v2", true)
	d, _ = reg.AddGroup("dirs", "d2")
	f, _ = d.AddResource("files", "f2", "v1")
	f.AddVersion("v1.1", true)

	gm2, _ := reg.Model.AddGroupModel("dirs2", "dir2")
	gm2.AddResourceModel("files", "file", 0, true, true, true)
	d2, _ := reg.AddGroup("dirs2", "d2")
	d2.AddResource("files", "f2", "v1")

	// /dirs/d1/files/f1/v1
	//                  /v2
	//      /d2/files/f2/v1
	//                  /v1.1
	// /dirs2/d2/files/f2/v1

	tests := []struct {
		Name string
		URL  string
		Exp  string
	}{
		{
			Name: "No Inline",
			URL:  "?",
			Exp: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestBasicInline",
  "epoch": 1,
  "self": "http://localhost:8181/",

  "dirscount": 2,
  "dirsurl": "http://localhost:8181/dirs",
  "dirs2count": 1,
  "dirs2url": "http://localhost:8181/dirs2"
}
`,
		},
		{
			Name: "Inline - No Filter - full",
			URL:  "?inline",
			Exp: `{
  "specversion": "` + registry.SPECVERSION + `",
  "id": "TestBasicInline",
  "epoch": 1,
  "self": "http://localhost:8181/",

  "dirs": {
    "d1": {
      "id": "d1",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1",

      "files": {
        "f1": {
          "id": "f1",
          "epoch": 1,
          "self": "http://localhost:8181/dirs/d1/files/f1?meta",
          "latestversionid": "v2",
          "latestversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2?meta",

          "versions": {
            "v1": {
              "id": "v1",
              "epoch": 1,
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1?meta"
            },
            "v2": {
              "id": "v2",
              "epoch": 1,
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2?meta",
              "latest": true
            }
          },
          "versionscount": 2,
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
        }
      },
      "filescount": 1,
      "filesurl": "http://localhost:8181/dirs/d1/files"
    },
    "d2": {
      "id": "d2",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d2",

      "files": {
        "f2": {
          "id": "f2",
          "epoch": 1,
          "self": "http://localhost:8181/dirs/d2/files/f2?meta",
          "latestversionid": "v1.1",
          "latestversionurl": "http://localhost:8181/dirs/d2/files/f2/versions/v1.1?meta",

          "versions": {
            "v1": {
              "id": "v1",
              "epoch": 1,
              "self": "http://localhost:8181/dirs/d2/files/f2/versions/v1?meta"
            },
            "v1.1": {
              "id": "v1.1",
              "epoch": 1,
              "self": "http://localhost:8181/dirs/d2/files/f2/versions/v1.1?meta",
              "latest": true
            }
          },
          "versionscount": 2,
          "versionsurl": "http://localhost:8181/dirs/d2/files/f2/versions"
        }
      },
      "filescount": 1,
      "filesurl": "http://localhost:8181/dirs/d2/files"
    }
  },
  "dirscount": 2,
  "dirsurl": "http://localhost:8181/dirs",
  "dirs2": {
    "d2": {
      "id": "d2",
      "epoch": 1,
      "self": "http://localhost:8181/dirs2/d2",

      "files": {
        "f2": {
          "id": "f2",
          "epoch": 1,
          "self": "http://localhost:8181/dirs2/d2/files/f2?meta",
          "latestversionid": "v1",
          "latestversionurl": "http://localhost:8181/dirs2/d2/files/f2/versions/v1?meta",

          "versions": {
            "v1": {
              "id": "v1",
              "epoch": 1,
              "self": "http://localhost:8181/dirs2/d2/files/f2/versions/v1?meta",
              "latest": true
            }
          },
          "versionscount": 1,
          "versionsurl": "http://localhost:8181/dirs2/d2/files/f2/versions"
        }
      },
      "filescount": 1,
      "filesurl": "http://localhost:8181/dirs2/d2/files"
    }
  },
  "dirs2count": 1,
  "dirs2url": "http://localhost:8181/dirs2"
}
`,
		},
		{
			Name: "Inline - No Filter",
			URL:  "?inline&oneline",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}},"dirs2":{"d2":{"files":{"f2":{"versions":{"v1":{}}}}}}}`,
		},
		{
			Name: "Inline * - * Filter",
			URL:  "?inline=*&oneline",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}},"dirs2":{"d2":{"files":{"f2":{"versions":{"v1":{}}}}}}}`,
		},
		{
			Name: "Inline * - * Filter - not first",
			URL:  "?inline=dirs2,*&oneline",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}},"dirs2":{"d2":{"files":{"f2":{"versions":{"v1":{}}}}}}}`,
		},
		{
			Name: "inline one level",
			URL:  "?inline=dirs&oneline",
			Exp:  `{"dirs":{"d1":{},"d2":{}}}`,
		},
		{
			Name: "inline one level - invalid",
			URL:  "?inline=xxx&oneline",
			Exp:  `Invalid 'inline' value: xxx`,
		},
		{
			Name: "inline one level - invalid - bad case",
			URL:  "?inline=Dirs&oneline",
			Exp:  `Invalid 'inline' value: Dirs`,
		},
		{
			Name: "inline two levels - invalid first",
			URL:  "?inline=xxx.files&oneline",
			Exp:  `Invalid 'inline' value: xxx.files`,
		},
		{
			Name: "inline two levels - invalid second",
			URL:  "?inline=dirs.xxx&oneline",
			Exp:  `Invalid 'inline' value: dirs.xxx`,
		},
		{
			Name: "inline two levels",
			URL:  "?inline=dirs.files&oneline",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{}}},"d2":{"files":{"f2":{}}}}}`,
		},
		{
			Name: "inline three levels",
			URL:  "?inline=dirs.files.versions&oneline",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "get one level, inline one level - invalid",
			URL:  "dirs?inline=dirs&oneline",
			Exp:  `Invalid 'inline' value: dirs`,
		},
		{
			Name: "get one level, inline one level",
			URL:  "dirs?inline=files&oneline",
			Exp:  `{"d1":{"files":{"f1":{}}},"d2":{"files":{"f2":{}}}}`,
		},
		{
			Name: "get one level, inline two levels",
			URL:  "dirs?inline=files.versions&oneline",
			Exp:  `{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}`,
		},
		{
			Name: "get one level, inline three levels",
			URL:  "dirs?inline=files.versions.xxx&oneline",
			Exp:  `Invalid 'inline' value: files.versions.xxx`,
		},
		{
			Name: "get one level, inline one level",
			URL:  "dirs/d1?inline=files&oneline",
			Exp:  `{"files":{"f1":{}}}`,
		},
		{
			Name: "get one level, inline two levels",
			URL:  "dirs/d1?inline=files.versions&oneline",
			Exp:  `{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}}`,
		},
		{
			Name: "get one level, inline all",
			URL:  "dirs/d1?inline=&oneline",
			Exp:  `{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}}`,
		},

		{
			Name: "inline 2 top levels",
			URL:  "?inline=dirs,dirs2&oneline",
			Exp:  `{"dirs":{"d1":{},"d2":{}},"dirs2":{"d2":{}}}`,
		},
		{
			Name: "inline 2 top, 1 and 2 levels",
			URL:  "?inline=dirs,dirs2.files&oneline",
			Exp:  `{"dirs":{"d1":{},"d2":{}},"dirs2":{"d2":{"files":{"f2":{}}}}}`,
		},
		{
			Name: "inline 2 top, 1 and 2 levels - one err",
			URL:  "?inline=dirs,dirs2.files.xxx&oneline",
			Exp:  `Invalid 'inline' value: dirs2.files.xxx`,
		},
		{
			Name: "get one level, inline 2, 1 and 2 levels same top",
			URL:  "dirs?inline=files,files.versions&oneline",
			Exp:  `{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}`,
		},

		{
			Name: "get one level, inline all",
			URL:  "dirs?inline&oneline",
			Exp:  `{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}`,
		},
		{
			Name: "get one level/res, inline all",
			URL:  "dirs/d2?inline&oneline",
			Exp:  `{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}`,
		},
	}

	for _, test := range tests {
		t.Logf("Testing: %s", test.Name)
		xCheckGet(t, reg, test.URL, test.Exp)
	}
}

func TestResourceInline(t *testing.T) {
	reg := NewRegistry("TestResourceInline")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	d, _ := reg.AddGroup("dirs", "d1")

	// ProxyURL
	f, _ := d.AddResource("files", "f1-proxy", "v1")
	f.SetSave(NewPP().P("#resource").UI(), "Hello world! v1")

	v, _ := f.AddVersion("v2", true)
	v.SetSave(NewPP().P("#resourceURL").UI(), "http://localhost:8181/EMPTY-URL")

	v, _ = f.AddVersion("v3", true)
	v.SetSave(NewPP().P("#resourceProxyURL").UI(), "http://localhost:8181/EMPTY-Proxy")

	// URL
	f, _ = d.AddResource("files", "f2-url", "v1")
	f.SetSave(NewPP().P("#resource").UI(), "Hello world! v1")

	v, _ = f.AddVersion("v2", true)
	v.SetSave(NewPP().P("#resourceProxyURL").UI(), "http://localhost:8181/EMPTY-Proxy")

	v, _ = f.AddVersion("v3", true)
	v.SetSave(NewPP().P("#resourceURL").UI(), "http://localhost:8181/EMPTY-URL")

	// Resource
	f, _ = d.AddResource("files", "f3-resource", "v1")
	f.SetSave(NewPP().P("#resourceProxyURL").UI(), "http://localhost:8181/EMPTY-Proxy")

	v, _ = f.AddVersion("v2", true)
	v.SetSave(NewPP().P("#resourceURL").UI(), "http://localhost:8181/EMPTY-URL")

	v, _ = f.AddVersion("v3", true)
	xNoErr(t, v.SetSave(NewPP().P("#resource").UI(), "Hello world! v3"))

	// /dirs/d1/files/f1-proxy/v1 - resource
	//                        /v2 - URL
	//                        /v3 - ProxyURL  <- latest
	// /dirs/d1/files/f2-url/v1 - resource
	//                      /v2 - ProxyURL
	//                      /v3 - URL  <- latest
	// /dirs/d1/files/f3-resource/v1 - ProxyURL
	//                           /v2 - URL
	//                           /v3 - resource  <- latest

	tests := []struct {
		Name string
		URL  string
		Exp  string
	}{
		{
			Name: "No Inline",
			URL:  "?",
			Exp: `{
  "dirscount": 1,
}
`,
		},
		{
			Name: "Inline - No Filter - full",
			URL:  "?inline",
			Exp: `{
  "dirs": {
    "d1": {
      "files": {
        "f1-proxy": {
          "latestversionid": "v3",
          "filebase64": "aGVsbG8tUHJveHk=",
          "versions": {
            "v1": {
              "filebase64": "SGVsbG8gd29ybGQhIHYx"
            },
            "v2": {
              "fileurl": "http://localhost:8181/EMPTY-URL"
            },
            "v3": {
              "filebase64": "aGVsbG8tUHJveHk="
            }
          },
          "versionscount": 3,
        },
        "f2-url": {
          "latestversionid": "v3",
          "fileurl": "http://localhost:8181/EMPTY-URL",
          "versions": {
            "v1": {
              "filebase64": "SGVsbG8gd29ybGQhIHYx"
            },
            "v2": {
              "filebase64": "aGVsbG8tUHJveHk="
            },
            "v3": {
              "fileurl": "http://localhost:8181/EMPTY-URL"
            }
          },
          "versionscount": 3,
        },
        "f3-resource": {
          "latestversionid": "v3",
          "filebase64": "SGVsbG8gd29ybGQhIHYz",
          "versions": {
            "v1": {
              "filebase64": "aGVsbG8tUHJveHk="
            },
            "v2": {
              "fileurl": "http://localhost:8181/EMPTY-URL"
            },
            "v3": {
              "filebase64": "SGVsbG8gd29ybGQhIHYz"
            }
          },
          "versionscount": 3,
        }
      },
      "filescount": 3,
      "filesurl": "http://localhost:8181/dirs/d1/files"
    }
  },
  "dirscount": 1,
}
`,
		},
		{
			Name: "Inline - filter + inline file",
			URL:  "?filter=dirs.files.id=f1-proxy&inline=dirs.files.file",
			Exp: `{
  "dirs": {
    "d1": {
      "files": {
        "f1-proxy": {
          "latestversionid": "v3",
          "filebase64": "aGVsbG8tUHJveHk=",
          "versionscount": 3,
        }
      },
      "filescount": 1,
      "filesurl": "http://localhost:8181/dirs/d1/files"
    }
  },
  "dirscount": 1,
}
`,
		},
		{
			Name: "Inline - filter + inline vers.file",
			URL:  "?filter=dirs.files.id=f1-proxy&inline=dirs.files.versions.file",
			Exp: `{
  "dirs": {
    "d1": {
      "files": {
        "f1-proxy": {
          "latestversionid": "v3",
          "versions": {
            "v1": {
              "filebase64": "SGVsbG8gd29ybGQhIHYx"
            },
            "v2": {
              "fileurl": "http://localhost:8181/EMPTY-URL"
            },
            "v3": {
              "filebase64": "aGVsbG8tUHJveHk="
            }
          },
          "versionscount": 3,
        }
      },
      "filescount": 1,
      "filesurl": "http://localhost:8181/dirs/d1/files"
    }
  },
  "dirscount": 1,
}
`,
		},
		{
			Name: "file-proxy",
			URL:  "/dirs/d1/files/f1-proxy",
			Exp:  `hello-Proxy`,
		},
		{
			Name: "file-url",
			URL:  "/dirs/d1/files/f2-url",
			Exp:  `hello-URL`,
		},
		{
			Name: "file-resource",
			URL:  "/dirs/d1/files/f3-resource",
			Exp:  `Hello world! v3`,
		},
		{
			Name: "Inline - at file + inline file",
			URL:  "/dirs/d1/files/f1-proxy?meta&inline=file",
			Exp: `{
  "latestversionid": "v3",
  "filebase64": "aGVsbG8tUHJveHk=",
  "versionscount": 3,
}
`,
		},
		{
			Name: "Inline - at file + inline file",
			URL:  "/dirs/d1/files/f1-proxy?meta&inline=versions.file",
			Exp: `{
  "latestversionid": "v3",
  "versions": {
    "v1": {
      "filebase64": "SGVsbG8gd29ybGQhIHYx"
    },
    "v2": {
      "fileurl": "http://localhost:8181/EMPTY-URL"
    },
    "v3": {
      "filebase64": "aGVsbG8tUHJveHk="
    }
  },
  "versionscount": 3,
}
`,
		},
		{
			Name: "Bad inline xx",
			URL:  "/dirs/d1/files/f1-proxy?meta&inline=XXversions.file",
			Exp:  "Invalid 'inline' value: dirs.files.XXversions.file\n",
		},
		{
			Name: "Bad inline yy",
			URL:  "/?inline=dirs.files.yy",
			Exp:  "Invalid 'inline' value: dirs.files.yy\n",
		},
		{
			Name: "Bad inline vers.yy",
			URL:  "/?inline=dirs.files.version.yy",
			Exp:  "Invalid 'inline' value: dirs.files.version.yy\n",
		},
	}

	// Save everythign to the DB
	xNoErr(t, reg.Commit())

	for _, test := range tests {
		t.Logf("Testing: %s", test.Name)

		remove := []string{
			`"specversion"`,
			`"id"`,
			`"epoch"`,
			`"self"`,
			`"latest"`,
			`"latestversionurl"`,
			`"versionsurl"`,
			`"dirsurl"`,
		}

		res, err := http.Get("http://localhost:8181/" + test.URL)
		xCheck(t, err == nil, fmt.Sprintf("%s", err))

		body, _ := io.ReadAll(res.Body)

		for _, str := range remove {
			str = fmt.Sprintf(`(?m)^ *%s.*$\n`, str)
			re := regexp.MustCompile(str)
			body = re.ReplaceAll(body, []byte{})
		}
		body = regexp.MustCompile(`(?m)^ *$\n`).ReplaceAll(body, []byte{})

		xCheckEqual(t, "Test: "+test.Name+"\n", string(body), test.Exp)
	}
}
