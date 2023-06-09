package tests

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"

	log "github.com/duglin/dlog"
	"github.com/duglin/xreg-github/registry"
)

func TestAll() *registry.Registry {
	reg := DoTests()
	log.VPrintf(1, "ALL TESTS PASSED")
	return reg
}

func RemoveProps(buf []byte) []byte {
	re := regexp.MustCompile(`\n[^{}]*\n`)
	buf = re.ReplaceAll(buf, []byte("\n"))

	re = regexp.MustCompile(`\s"tags": {\s*},*`)
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

func NoErr(name string, err error) {
	if err == nil {
		return
	}
	log.Fatalf("%s: %s", name, err)
}

func Check(b bool, errStr string) {
	if !b {
		log.Fatal(errStr)
	}
}

func CheckGet(reg *registry.Registry, name string, URL string, expected string) {
	buf := &bytes.Buffer{}
	out := io.Writer(buf)

	req, err := http.NewRequest("GET", URL, nil)
	NoErr(name, err)
	info, err := reg.ParseRequest(req)
	if err != nil {
		CheckEqual(err.Error(), expected, name)
		return
	}
	Check(info.ErrCode == 0, name+":info.ec != 0")
	err = reg.NewGet(out, info)

	if err != nil {
		CheckEqual(err.Error(), expected, name)
		return
	}
	NoErr(name, err)

	if req.URL.Query().Has("noprops") {
		buf = bytes.NewBuffer(RemoveProps(buf.Bytes()))
	}

	CheckEqual(buf.String(), expected, name)
}

func CheckEqual(str1 string, str2 string, desc string) {
	if str1 != str2 {
		pos := 0
		for pos < len(str1) && pos < len(str2) && str1[pos] == str2[pos] {
			pos++
		}

		log.Fatalf("%s - Output mismatch:\n"+
			"Expected:\n%s\nGot:\n%s\n\nAt: %s",
			desc, str2, str1, str1[pos:])
	}
}

func DoTests() *registry.Registry {
	// Registry stuff
	reg, err := registry.NewRegistry("666-1234-1234")
	NoErr("new reg", err)
	NoErr("reg refresh", reg.Refresh())

	// reg.Set("baseURL", "http://soaphub.org:8585")
	reg.Set("name", "testReg")
	reg.Set("description", "A test Reg")
	reg.Set("specVersion", "0.5")
	reg.Set("docs", "docs-url")

	reg1, err := registry.FindRegistry(reg.ID)
	NoErr("didn't find reg1", err)
	Check(reg1 != nil, "reg1 is nil")
	NoErr("reg1 refresh", reg1.Refresh())
	if registry.ToJSON(reg) != registry.ToJSON(reg1) {
		log.Fatalf("\nreg : %v\n!=\nreg1: %v", reg, reg1)
	}

	reg2, err := registry.FindRegistry(reg.ID)
	NoErr("get reg2", err)
	CheckEqual(registry.ToJSON(reg2), registry.ToJSON(reg), "reg2!=reg")

	CheckGet(reg, "minimal reg", "http://example.com", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "name": "testReg",
  "self": "http://example.com/",
  "description": "A test Reg",
  "docs": "docs-url"
}
`)
	reg.Set("description", nil)
	reg.Set("name", nil)
	reg.Set("docs", nil)

	CheckGet(reg, "reg del props", "http://example.com", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/"
}
`)

	// Model stuff
	gm1, err := reg.AddGroupModel("dirs", "dir", "schema-url")
	NoErr("add groups2", err)

	CheckGet(reg, "one group model", "http://example.com?inline", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "dirs": {},
  "dirsCount": 0,
  "dirsUrl": "http://example.com/dirs"
}
`)

	CheckGet(reg, "inline *", "http://example.com?inline=*", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "dirs": {},
  "dirsCount": 0,
  "dirsUrl": "http://example.com/dirs"
}
`)

	CheckGet(reg, "inline by name", "http://example.com?inline=dirs", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "dirs": {},
  "dirsCount": 0,
  "dirsUrl": "http://example.com/dirs"
}
`)

	CheckGet(reg, "no inline", "http://example.com", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "dirsCount": 0,
  "dirsUrl": "http://example.com/dirs"
}
`)

	CheckGet(reg, "bad inline", "http://example.com?inline=foo",
		`Invalid 'inline' value: "foo"`)

	_, err = gm1.AddResourceModel("files", "file", 5, true, true)
	NoErr("add files", err)

	CheckGet(reg, "check model", "http://example.com?model", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",
  "model": {
    "groups": {
      "dirs": {
        "plural": "dirs",
        "singular": "dir",
        "schema": "schema-url",
        "resources": {
          "files": {
            "plural": "files",
            "singular": "file",
            "versions": 5,
            "versionId": true,
            "latest": true
          }
        }
      }
    }
  },

  "dirsCount": 0,
  "dirsUrl": "http://example.com/dirs"
}
`)

	CheckGet(reg, "just model", "http://example.com/model", `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "schema": "schema-url",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "versions": 5,
          "versionId": true,
          "latest": true
        }
      }
    }
  }
}
`)

	_, err = gm1.AddResourceModel("file2s", "file2", 4, false, false)
	NoErr("add files", err)

	CheckGet(reg, "model with false", "http://example.com?model", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",
  "model": {
    "groups": {
      "dirs": {
        "plural": "dirs",
        "singular": "dir",
        "schema": "schema-url",
        "resources": {
          "file2s": {
            "plural": "file2s",
            "singular": "file2",
            "versions": 4,
            "versionId": false,
            "latest": false
          },
          "files": {
            "plural": "files",
            "singular": "file",
            "versions": 5,
            "versionId": true,
            "latest": true
          }
        }
      }
    }
  },

  "dirsCount": 0,
  "dirsUrl": "http://example.com/dirs"
}
`)

	m1 := reg.LoadModel()
	Check(m1.Groups["dirs"].Singular == "dir", "dirs.Singular")
	Check(m1.Groups["dirs"].Resources["files"].Versions == 5, "files.Vers")

	// Group stuff
	g1 := reg.FindGroup("dirs", "g1")
	Check(g1 == nil, "g1 should be nil")
	g1 = reg.FindOrAddGroup("dirs", "g1")
	Check(g1 != nil, "g1 should not be nil")
	g1.Set("name", g1.ID)
	g1.Set("epoch", 5)
	g1.Set("ext1", "extvalue")
	g1.Set("ext2", 666)
	Check(g1.Extensions["ext2"] == 666, "g1.Ext isn't an int")
	g2 := reg.FindGroup("dirs", "g1")
	CheckEqual(registry.ToJSON(g2), registry.ToJSON(g1), "g2 != g1")
	g2.Set("ext2", nil)
	g2.Set("epoch", nil)
	g1.Refresh()
	CheckEqual(registry.ToJSON(g2), registry.ToJSON(g1), "g1.refresh")

	CheckGet(reg, "one group", "http://example.com?inline", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "dirs": {
    "g1": {
      "id": "g1",
      "name": "g1",
      "self": "http://example.com/dirs/g1",
      "ext1": "extvalue",

      "file2s": {},
      "file2sCount": 0,
      "file2sUrl": "http://example.com/dirs/g1/file2s",
      "files": {},
      "filesCount": 0,
      "filesUrl": "http://example.com/dirs/g1/files"
    }
  },
  "dirsCount": 1,
  "dirsUrl": "http://example.com/dirs"
}
`)

	CheckGet(reg, "one group no inline", "http://example.com", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "dirsCount": 1,
  "dirsUrl": "http://example.com/dirs"
}
`)

	// Resource stuff
	r1 := g1.FindResource("files", "r1")
	Check(r1 == nil, "r1 should be nil")

	// Technical this is wrong - we need to create a version at the
	// same time - TODO
	// use g.AddResource() instead
	r1 = g1.FindOrAddResource("files", "r1")
	Check(r1 != nil, "r1 should not be nil")

	CheckGet(reg, "one res no inline", "http://example.com?inline", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "dirs": {
    "g1": {
      "id": "g1",
      "name": "g1",
      "self": "http://example.com/dirs/g1",
      "ext1": "extvalue",

      "file2s": {},
      "file2sCount": 0,
      "file2sUrl": "http://example.com/dirs/g1/file2s",
      "files": {
        "r1": {
          "id": "r1",
          "self": "http://example.com/dirs/g1/files/r1",

          "versions": {},
          "versionsCount": 0,
          "versionsUrl": "http://example.com/dirs/g1/files/r1/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http://example.com/dirs/g1/files"
    }
  },
  "dirsCount": 1,
  "dirsUrl": "http://example.com/dirs"
}
`)

	CheckGet(reg, "1 res,inline 3 level", "http://example.com?inline=dirs.files.versions", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "dirs": {
    "g1": {
      "id": "g1",
      "name": "g1",
      "self": "http://example.com/dirs/g1",
      "ext1": "extvalue",

      "file2sCount": 0,
      "file2sUrl": "http://example.com/dirs/g1/file2s",
      "files": {
        "r1": {
          "id": "r1",
          "self": "http://example.com/dirs/g1/files/r1",

          "versions": {},
          "versionsCount": 0,
          "versionsUrl": "http://example.com/dirs/g1/files/r1/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http://example.com/dirs/g1/files"
    }
  },
  "dirsCount": 1,
  "dirsUrl": "http://example.com/dirs"
}
`)

	CheckGet(reg, "1 res,inline 2 level", "http://example.com?inline=dirs.files", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "dirs": {
    "g1": {
      "id": "g1",
      "name": "g1",
      "self": "http://example.com/dirs/g1",
      "ext1": "extvalue",

      "file2sCount": 0,
      "file2sUrl": "http://example.com/dirs/g1/file2s",
      "files": {
        "r1": {
          "id": "r1",
          "self": "http://example.com/dirs/g1/files/r1",

          "versionsCount": 0,
          "versionsUrl": "http://example.com/dirs/g1/files/r1/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http://example.com/dirs/g1/files"
    }
  },
  "dirsCount": 1,
  "dirsUrl": "http://example.com/dirs"
}
`)

	CheckGet(reg, "1 res,inline 1 level", "http://example.com?inline=dirs", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "dirs": {
    "g1": {
      "id": "g1",
      "name": "g1",
      "self": "http://example.com/dirs/g1",
      "ext1": "extvalue",

      "file2sCount": 0,
      "file2sUrl": "http://example.com/dirs/g1/file2s",
      "filesCount": 1,
      "filesUrl": "http://example.com/dirs/g1/files"
    }
  },
  "dirsCount": 1,
  "dirsUrl": "http://example.com/dirs"
}
`)

	CheckGet(reg, "1 deep", "http://example.com/dirs?inline", `{
  "g1": {
    "id": "g1",
    "name": "g1",
    "self": "http://example.com/dirs/g1",
    "ext1": "extvalue",

    "file2s": {},
    "file2sCount": 0,
    "file2sUrl": "http://example.com/dirs/g1/file2s",
    "files": {
      "r1": {
        "id": "r1",
        "self": "http://example.com/dirs/g1/files/r1",

        "versions": {},
        "versionsCount": 0,
        "versionsUrl": "http://example.com/dirs/g1/files/r1/versions"
      }
    },
    "filesCount": 1,
    "filesUrl": "http://example.com/dirs/g1/files"
  }
}
`)

	CheckGet(reg, "1 deep+2 level", "http://example.com/dirs?inline=files.versions", `{
  "g1": {
    "id": "g1",
    "name": "g1",
    "self": "http://example.com/dirs/g1",
    "ext1": "extvalue",

    "file2sCount": 0,
    "file2sUrl": "http://example.com/dirs/g1/file2s",
    "files": {
      "r1": {
        "id": "r1",
        "self": "http://example.com/dirs/g1/files/r1",

        "versions": {},
        "versionsCount": 0,
        "versionsUrl": "http://example.com/dirs/g1/files/r1/versions"
      }
    },
    "filesCount": 1,
    "filesUrl": "http://example.com/dirs/g1/files"
  }
}
`)

	CheckGet(reg, "1 deep+1 level", "http://example.com/dirs?inline=files", `{
  "g1": {
    "id": "g1",
    "name": "g1",
    "self": "http://example.com/dirs/g1",
    "ext1": "extvalue",

    "file2sCount": 0,
    "file2sUrl": "http://example.com/dirs/g1/file2s",
    "files": {
      "r1": {
        "id": "r1",
        "self": "http://example.com/dirs/g1/files/r1",

        "versionsCount": 0,
        "versionsUrl": "http://example.com/dirs/g1/files/r1/versions"
      }
    },
    "filesCount": 1,
    "filesUrl": "http://example.com/dirs/g1/files"
  }
}
`)

	CheckGet(reg, "1 deep+bad", "http://example.com/dirs?inline=foo",
		`Invalid 'inline' value: "foo"`)

	// Test setting Resource stuff, not Latest version stuff
	r1.Set(".name", "unique")
	Check(r1.Extensions["name"] == "unique", "r1.Name != unique")
	r1.Set(".Int", 345)
	r1.Set(".Float", 3.14)
	r1.Set(".BoolT", true)
	r1.Set(".BoolF", false)
	Check(r1.Extensions["Int"] == 345, "r1.Int != 345")
	Check(r1.Extensions["Float"] == 3.14, "r1.Float != 3.14")
	Check(r1.Extensions["BoolT"] == true, "r1.BoolT != true")
	Check(r1.Extensions["BoolF"] == false, "r1.BoolF != false")
	r3 := g1.FindResource("files", "r1")
	CheckEqual(registry.ToJSON(r3), registry.ToJSON(r1), "r3 != r1")
	Check(r3.Extensions["Int"] == 345, "r3.Int != 345")
	Check(r3.Extensions["Float"] == 3.14, "r3.Float != 3.14")
	Check(r3.Extensions["BoolT"] == true, "r3.BoolT != true")
	Check(r3.Extensions["BoolF"] == false, "r3.BoolF != false")

	CheckGet(reg, "r1 props", "http://example.com/dirs?inline", `{
  "g1": {
    "id": "g1",
    "name": "g1",
    "self": "http://example.com/dirs/g1",
    "ext1": "extvalue",

    "file2s": {},
    "file2sCount": 0,
    "file2sUrl": "http://example.com/dirs/g1/file2s",
    "files": {
      "r1": {
        "id": "r1",
        "name": "unique",
        "self": "http://example.com/dirs/g1/files/r1",
        "BoolF": false,
        "BoolT": true,
        "Float": 3.14,
        "Int": 345,

        "versions": {},
        "versionsCount": 0,
        "versionsUrl": "http://example.com/dirs/g1/files/r1/versions"
      }
    },
    "filesCount": 1,
    "filesUrl": "http://example.com/dirs/g1/files"
  }
}
`)

	// Version stuff
	v1 := r1.FindVersion("v1")
	Check(v1 == nil, "v1 should be nil")
	v1 = r1.FindOrAddVersion("v1")
	Check(v1 != nil, "v1 should not be nil")
	Check(registry.ToJSON(v1) == registry.ToJSON(r1.GetLatest()), "not latest")

	v1.Set("name", v1.ID)
	v1.Set("epoch", 42)
	v1.Set("ext1", "someext")
	v1.Set("ext2", 234)
	Check(v1.Extensions["ext2"] == 234, "v1.Ext isn't an int")
	v2 := r1.FindVersion("v1")
	Check(registry.ToJSON(v1) == registry.ToJSON(v2), "v2 != v1")
	vlatest := r1.GetLatest()
	Check(registry.ToJSON(v1) == registry.ToJSON(vlatest), "vlatest != v1")

	CheckGet(reg, "r1 props", "http://example.com/dirs?inline", `{
  "g1": {
    "id": "g1",
    "name": "g1",
    "self": "http://example.com/dirs/g1",
    "ext1": "extvalue",

    "file2s": {},
    "file2sCount": 0,
    "file2sUrl": "http://example.com/dirs/g1/file2s",
    "files": {
      "r1": {
        "id": "r1",
        "name": "v1",
        "epoch": 42,
        "self": "http://example.com/dirs/g1/files/r1",
        "latestId": "v1",
        "latestUrl": "http://example.com/dirs/g1/files/r1/versions/v1",
        "BoolF": false,
        "BoolT": true,
        "Float": 3.14,
        "Int": 345,
        "ext1": "someext",
        "ext2": 234,

        "versions": {
          "v1": {
            "id": "v1",
            "name": "v1",
            "epoch": 42,
            "self": "http://example.com/dirs/g1/files/r1/versions/v1",
            "ext1": "someext",
            "ext2": 234
          }
        },
        "versionsCount": 1,
        "versionsUrl": "http://example.com/dirs/g1/files/r1/versions"
      }
    },
    "filesCount": 1,
    "filesUrl": "http://example.com/dirs/g1/files"
  }
}
`)

	// Test Latest version stuff
	r1.Set("name", r1.ID)
	r1.Set("epoch", 68)
	r1.Set("ext1", "someext")
	r1.Set("ext2", 123)
	Check(r1.GetLatest().Extensions["ext2"] == 123, "r1.Ext isn't an int")
	r2 := g1.FindResource("files", "r1")
	Check(registry.ToJSON(r1) == registry.ToJSON(r2), "r2 != r1")
	Check(r1.FindVersion("v3") == nil, "v3 should be nil")
	Check(r2.FindVersion("v3") == nil, "v3 should be nil")

	CheckGet(reg, "v3 missing",
		"http://example.com/dirs/g1/files/r1/versions/v3",
		"not found\n")

	// Test tags
	v1.Set("tags.stage", "dev")
	v1.Set("tags.stale", "true")
	v1.Set("tags.int", 3)

	CheckGet(reg, "v2.tags",
		"http://example.com/dirs/g1/files/r1/versions/v1", `{
  "id": "v1",
  "name": "r1",
  "epoch": 68,
  "self": "http://example.com/dirs/g1/files/r1/versions/v1",
  "tags": {
    "tags.int": "3",
    "tags.stage": "dev",
    "tags.stale": "true"
  },
  "ext1": "someext",
  "ext2": 123
}
`)

	// Some filtering
	g2 = reg.FindOrAddGroup("dirs", "g2")
	r2 = g2.FindOrAddResource("files", "r2")
	v2 = r2.FindOrAddVersion("v1")
	g2.Set("tags.stage", "dev")
	r1.Set("tags.stale", "true")
	v2.Set("tags.v2", "true")

	CheckGet(reg, "filter id",
		"http://example.com/?filter=dirs.id=g2", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "dirsCount": 1,
  "dirsUrl": "http://example.com/dirs"
}
`)

	CheckGet(reg, "filter id inline",
		"http://example.com/?inline&filter=dirs.id=g2", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "dirs": {
    "g2": {
      "id": "g2",
      "self": "http://example.com/dirs/g2",
      "tags": {
        "tags.stage": "dev"
      },

      "file2s": {},
      "file2sCount": 0,
      "file2sUrl": "http://example.com/dirs/g2/file2s",
      "files": {
        "r2": {
          "id": "r2",
          "self": "http://example.com/dirs/g2/files/r2",
          "latestId": "v1",
          "latestUrl": "http://example.com/dirs/g2/files/r2/versions/v1",
          "tags": {
            "tags.v2": "true"
          },

          "versions": {
            "v1": {
              "id": "v1",
              "self": "http://example.com/dirs/g2/files/r2/versions/v1",
              "tags": {
                "tags.v2": "true"
              }
            }
          },
          "versionsCount": 1,
          "versionsUrl": "http://example.com/dirs/g2/files/r2/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http://example.com/dirs/g2/files"
    }
  },
  "dirsCount": 1,
  "dirsUrl": "http://example.com/dirs"
}
`)

	CheckGet(reg, "filter tag level 1",
		"http://example.com/?inline&noprops&filter=dirs.tags.stage=dev", `{
  "dirs": {
    "g2": {
      "file2s": {},
      "files": {
        "r2": {
          "versions": {
            "v1": {}}}}}}}
`)

	CheckGet(reg, "filter AND same obj",
		"http://example.com/?inline&noprops&filter=dirs.id=g1,dirs.name=g1", `{
  "dirs": {
    "g1": {
      "file2s": {},
      "files": {
        "r1": {
          "versions": {
            "v1": {}}}}}}}
`)

	CheckGet(reg, "filter id OR same obj",
		"http://example.com/?inline&noprops&filter=dirs.id=g1&filter=dirs.name=g1", `{
  "dirs": {
    "g1": {
      "file2s": {},
      "files": {
        "r1": {
          "versions": {
            "v1": {}}}}}}}
`)

	CheckGet(reg, "filter id OR no 2nd match",
		"http://example.com/?inline&noprops&filter=dirs.id=g1&filter=dirs.name=g3", `{
  "dirs": {
    "g1": {
      "file2s": {},
      "files": {
        "r1": {
          "versions": {
            "v1": {}}}}}}}
`)

	CheckGet(reg, "filter id AND no 2nd match",
		"http://example.com/?inline&noprops&filter=dirs.id=g1,dirs.name=g3", `{
  "dirs": {}}
`)

	CheckGet(reg, "filter tags level 2",
		"http://example.com/?inline&noprops&filter=dirs.files.tags.v2=true", `{
  "dirs": {
    "g2": {
      "file2s": {},
      "files": {
        "r2": {
          "versions": {
            "v1": {}}}}}}}
`)

	CheckGet(reg, "filter multi result level 2",
		"http://example.com/?inline&noprops&filter=dirs.files.latestId=v1", `{
  "dirs": {
    "g1": {
      "file2s": {},
      "files": {
        "r1": {
          "versions": {
            "v1": {}}}}},
    "g2": {
      "file2s": {},
      "files": {
        "r2": {
          "versions": {
            "v1": {}}}}}}}
`)

	CheckGet(reg, "filter group in filter and path - bad",
		"http://example.com/dirs?inline&noprops&filter=dirs.files.latestId=v1", `{}
`)
	CheckGet(reg, "filter path+level 1",
		"http://example.com/dirs?inline&noprops&filter=files.latestId=v1", `{
  "g1": {
    "file2s": {},
    "files": {
      "r1": {
        "versions": {
          "v1": {}}}}},
  "g2": {
    "file2s": {},
    "files": {
      "r2": {
        "versions": {
          "v1": {}}}}}}
`)

	// reg.Delete()
	return reg
}
