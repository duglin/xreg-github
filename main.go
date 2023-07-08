package main

import (
	"archive/tar"
	"regexp"
	// "bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	log "github.com/duglin/dlog"
	"github.com/duglin/xreg-github/registry"
)

func init() {
	log.SetVerbose(2)
}

//go:embed .github
var Token string
var Secret = ""
var Port = "8080"
var Reg = (*registry.Registry)(nil)

func LoadGitRepo(orgName string, repoName string) *registry.Registry {
	log.Printf("Loading registry '%s/%s'", orgName, repoName)
	/*
		gh := github.NewGitHubClient("api.github.com", Token, Secret)
		repo, err := gh.GetRepository(orgName, repoName)
		if err != nil {
			log.Fatalf("Error finding repo %s/%s: %s", orgName, repoName, err)
		}

		tarStream, err := repo.GetTar()
		if err != nil {
			log.Fatalf("Error getting tar from repo %s/%s: %s",
				orgName, repoName, err)
		}
		defer tarStream.Close()
	*/

	buf, _ := ioutil.ReadFile("repo.tar")
	tarStream := bytes.NewReader(buf)

	gzf, _ := gzip.NewReader(tarStream)
	reader := tar.NewReader(gzf)

	reg, err := registry.NewRegistry("123-4567-3456")
	registry.ErrFatalf(err, "Error creating new registry: %s", err)
	// log.VPrintf(3, "New registry:\n%#v", reg)

	reg.Set("BaseURL", "http://soaphub.org:8585/")
	reg.Set("name", "APIs-guru Registry")
	reg.Set("description", "xRegistry view of github.com/APIs-guru/openapi-directory")
	reg.Set("specVersion", "0.5")
	reg.Set("docs", "https://github.com/duglin/xreg-github")
	err = reg.Refresh()
	registry.ErrFatalf(err, "Error refeshing registry: %s", err)
	// log.VPrintf(3, "New registry:\n%#v", reg)

	// TODO Support "model" being part of the Registry struct above

	g, _ := reg.AddGroupModel("apiProviders", "apiProvider", "")
	_, err = g.AddResourceModel("apis", "api", 2)

	g, _ = reg.AddGroupModel("schemaGroups", "schemaGroup", "")
	_, err = g.AddResourceModel("schemas", "schema", 1)

	m := reg.LoadModel()
	log.VPrintf(3, "Model: %#v\n", m)

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Error getting next tar entry: %s", err)
		}

		// Skip non-regular files (and dirs)
		if header.Typeflag > '9' || header.Typeflag == tar.TypeDir {
			continue
		}

		i := 0
		// Skip files not under the APIs dir
		if i = strings.Index(header.Name, "/APIs/"); i < 0 {
			continue
		}

		if strings.Index(header.Name, "/docker.com/") < 0 &&
			strings.Index(header.Name, "/apiz.ebay.com/") < 0 {
			continue
		}

		parts := strings.Split(strings.Trim(header.Name[i+6:], "/"), "/")
		// org/service/version/file
		// org/version/file

		group := reg.FindOrAddGroup("apiProviders", parts[0])
		group.Set("name", group.ID)
		group.Set("modifiedBy", "me")
		group.Set("modifiedAt", "noon")
		group.Set("epoch", 5)
		group.Set("xxx", 5)
		group.Set("yyy", "6")
		group.Set("zzz", "6")

		group.Set("modifiedAt", nil) // delete prop
		group.Set("zzz", nil)        // delete prop

		// group2 := reg.FindGroup("apiProviders", parts[0])
		// log.Printf("Find Group:\n%s", registry.ToJSON(group2))

		resName := "core"
		verIndex := 1
		if len(parts) == 4 {
			resName = parts[1]
			verIndex++
		}

		res := group.FindOrAddResource("apis", resName)

		g2 := reg.FindOrAddGroup("schemaGroups", parts[0])
		g2.Set("name", group.Get("name"))
		/*
			r2 := g2.FindOrAddResource("schemas", resName)
			v2 := r2.FindOrAddVersion(parts[verIndex])
			v2.Name = parts[verIndex+1]
			v2.Format = "openapi/3.0.6"
		*/
		version := res.FindVersion(parts[verIndex])
		if version != nil {
			log.Fatalf("Have more than one file per version: %s\n", header.Name)
		}

		buf := &bytes.Buffer{}
		io.Copy(buf, reader)
		version = res.FindOrAddVersion(parts[verIndex])
		version.Set("name", parts[verIndex+1])
		version.Set("format", "openapi/3.0.6")

		// Don't upload the file contents into the registry. Instead just
		// give the registry a URL to it and ask it to server it via proxy.
		// We could have also just set the resourceURI to the file but
		// I wanted the URL to the file to be the registry and not github
		base := "https://raw.githubusercontent.com/APIs-guru/" +
			"openapi-directory/main/APIs/"
		// version.ResourceURL = base + header.Name[i+6:]
		// version.ResourceContent = buf.Bytes()
		version.ResourceProxyURL = base + header.Name[i+6:]
	}

	return reg
}

func handler(w http.ResponseWriter, r *http.Request) {
	log.VPrintf(2, "%s %s", r.Method, r.URL.Path)

	info, err := Reg.ParseRequest(r)
	if err != nil {
		w.WriteHeader(info.ErrCode)
		w.Write([]byte(err.Error()))
		return
	}

	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var out = io.Writer(w)
	buf := bytes.Buffer{}

	if r.URL.Query().Has("html") {
		out = io.Writer(&buf) // bufio.NewWriter(&buf)
	}

	err = Reg.NewGet(out, info)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	if r.URL.Query().Has("html") {
		w.Header().Add("Content-Type", "text/html")

		re := regexp.MustCompile(`"(http[^"\n]*?)"`)
		repl := fmt.Sprintf(`"<a href="$1?%s">$1?%s</a>"`,
			r.URL.RawQuery, r.URL.RawQuery)

		w.Write([]byte("<pre>\n"))
		w.Write(re.ReplaceAll(buf.Bytes(), []byte(repl)))
		w.Write([]byte("</pre>\n"))
	}

	return

	/*
		baseURL := fmt.Sprintf("http://%s", r.Host)

		rFlags := &registry.RegistryFlags{
			Indent:      "  ",
			InlineAll:   false,
			InlinePaths: []string(nil),
			Self:        r.URL.Query().Has("self"),
			AsDoc:       r.URL.Query().Has("doc"),
			BaseURL:     baseURL,
			Filters:     r.URL.Query()["filter"],
		}

		if r.URL.Query().Has("inline") {
			for _, value := range r.URL.Query()["inline"] {
				paths := strings.Split(value, ",")
				for _, p := range paths {
					p = strings.TrimSpace(p)
					if p != "" {
						rFlags.InlinePaths = append(rFlags.InlinePaths, p)
					}
				}
			}
			if rFlags.InlinePaths == nil {
				rFlags.InlineAll = true
			}
		}

		res, err := Reg.Get(r.URL.Path, rFlags)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		if r.URL.Query().Has("html") {
			w.Header().Add("Content-Type", "text/html")
		}

		w.WriteHeader(http.StatusOK)

		if r.URL.Query().Has("html") {
			start := 0
			w.Write([]byte("<pre>"))
			for pos := 0; pos+5 < len(res); pos++ {
				if res[pos] != '"' || res[pos:pos+5] != `"http` {
					continue
				}
				w.Write([]byte(res[start : pos+1]))
				end := pos + 1
				for ; end < len(res) && res[end] != '"'; end++ {
				}
				start = end
				url := res[pos+1 : end]
				org := url
				if strings.Index(url, "?") < 0 {
					url += "?" + r.URL.RawQuery
				} else {
					url += "&" + r.URL.RawQuery
				}
				//repl := fmt.Sprintf(`<a href="%s">%s</a>`, url, org)
				//w.Write([]byte(repl))
				fmt.Fprintf(w, `<a href="%s">%s</a>`, url, org)
			}
			w.Write([]byte(res[start:]))
		} else {
			w.Write([]byte(res))
		}
	*/
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
	buf := bytes.Buffer{}
	out := io.Writer(&buf)

	req, err := http.NewRequest("GET", URL, nil)
	NoErr(name, err)
	info, err := reg.ParseRequest(req)
	if err != nil {
		CheckEqual(err.Error(), expected, name)
		return
	}
	Check(info.ErrCode == 0, name+":info.ec != 0")
	err = reg.NewGet(out, info)
	NoErr(name, err)
	CheckEqual(buf.String(), expected, name)
}

func CheckEqual(str1 string, str2 string, desc string) {
	if str1 != str2 {
		pos := 0
		for pos < len(str1) && pos < len(str2) && str1[pos] == str2[pos] {
			pos++
		}

		log.Fatalf("%s - Output mismatch:\n"+
			"Expected:\n%s\nGot:\n%s\n\nAt: %#v",
			desc, str2, str1, str1[:pos])
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
	Check(registry.ToJSON(reg) == registry.ToJSON(reg2), "reg2!=reg")

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
	gm1, err := reg.AddGroupModel("myGroups", "myGroup", "schema-url")
	NoErr("add myGroups2", err)

	CheckGet(reg, "one group model", "http://example.com?inline", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "myGroups": {},
  "myGroupsCount": 0,
  "myGroupsUrl": "http://example.com/myGroups"
}
`)

	CheckGet(reg, "inline *", "http://example.com?inline=*", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "myGroups": {},
  "myGroupsCount": 0,
  "myGroupsUrl": "http://example.com/myGroups"
}
`)

	CheckGet(reg, "inline by name", "http://example.com?inline=myGroups", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "myGroups": {},
  "myGroupsCount": 0,
  "myGroupsUrl": "http://example.com/myGroups"
}
`)

	CheckGet(reg, "no inline", "http://example.com", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "myGroupsCount": 0,
  "myGroupsUrl": "http://example.com/myGroups"
}
`)

	CheckGet(reg, "bad inline", "http://example.com?inline=foo",
		`Bad inline - path: "foo"`)

	_, err = gm1.AddResourceModel("ress", "res", 5)
	NoErr("add ress", err)

	m1 := reg.LoadModel()
	Check(m1.Groups["myGroups"].Singular == "myGroup", "myGroups.Singular")
	Check(m1.Groups["myGroups"].Resources["ress"].Versions == 5, "ress.Vers")

	// Group stuff
	g1 := reg.FindGroup("myGroups", "g1")
	Check(g1 == nil, "g1 should be nil")
	g1 = reg.FindOrAddGroup("myGroups", "g1")
	Check(g1 != nil, "g1 should not be nil")
	g1.Set("name", g1.ID)
	g1.Set("epoch", 5)
	g1.Set("ext1", "extvalue")
	g1.Set("ext2", 666)
	Check(g1.Extensions["ext2"] == 666, "g1.Ext isn't an int")
	g2 := reg.FindGroup("myGroups", "g1")
	Check(registry.ToJSON(g1) == registry.ToJSON(g2), "g2 != g1")
	g2.Set("ext2", nil)
	g2.Set("epoch", nil)
	g1.Refresh()
	Check(registry.ToJSON(g1) == registry.ToJSON(g2), "g1.refresh")

	CheckGet(reg, "one group", "http://example.com?inline", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "myGroups": {
    "g1": {
      "id": "g1",
      "name": "g1",
      "self": "http://example.com/myGroups/g1",
      "ext1": "extvalue",

      "ress": {},
      "ressCount": 0,
      "ressUrl": "http://example.com/myGroups/g1/ress"
    }
  },
  "myGroupsCount": 1,
  "myGroupsUrl": "http://example.com/myGroups"
}
`)

	CheckGet(reg, "one group no inline", "http://example.com", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "myGroupsCount": 1,
  "myGroupsUrl": "http://example.com/myGroups"
}
`)

	// Resource stuff
	r1 := g1.FindResource("ress", "r1")
	Check(r1 == nil, "r1 should be nil")

	// Technical this is wrong - we need to create a version at the
	// same time - TODO
	// use g.AddResource() instead
	r1 = g1.FindOrAddResource("ress", "r1")
	Check(r1 != nil, "r1 should not be nil")

	CheckGet(reg, "one res no inline", "http://example.com?inline", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "myGroups": {
    "g1": {
      "id": "g1",
      "name": "g1",
      "self": "http://example.com/myGroups/g1",
      "ext1": "extvalue",

      "ress": {
        "r1": {
          "id": "r1",
          "self": "http://example.com/myGroups/g1/ress/r1",

          "versions": {},
          "versionsCount": 0,
          "versionsUrl": "http://example.com/myGroups/g1/ress/r1/versions"
        }
      },
      "ressCount": 1,
      "ressUrl": "http://example.com/myGroups/g1/ress"
    }
  },
  "myGroupsCount": 1,
  "myGroupsUrl": "http://example.com/myGroups"
}
`)

	CheckGet(reg, "1 res,inline 3 level", "http://example.com?inline=myGroups.ress.versions", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "myGroups": {
    "g1": {
      "id": "g1",
      "name": "g1",
      "self": "http://example.com/myGroups/g1",
      "ext1": "extvalue",

      "ress": {
        "r1": {
          "id": "r1",
          "self": "http://example.com/myGroups/g1/ress/r1",

          "versions": {},
          "versionsCount": 0,
          "versionsUrl": "http://example.com/myGroups/g1/ress/r1/versions"
        }
      },
      "ressCount": 1,
      "ressUrl": "http://example.com/myGroups/g1/ress"
    }
  },
  "myGroupsCount": 1,
  "myGroupsUrl": "http://example.com/myGroups"
}
`)

	CheckGet(reg, "1 res,inline 1 level", "http://example.com?inline=myGroups.ress", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "myGroups": {
    "g1": {
      "id": "g1",
      "name": "g1",
      "self": "http://example.com/myGroups/g1",
      "ext1": "extvalue",

      "ress": {
        "r1": {
          "id": "r1",
          "self": "http://example.com/myGroups/g1/ress/r1",

          "versionsCount": 0,
          "versionsUrl": "http://example.com/myGroups/g1/ress/r1/versions"
        }
      },
      "ressCount": 1,
      "ressUrl": "http://example.com/myGroups/g1/ress"
    }
  },
  "myGroupsCount": 1,
  "myGroupsUrl": "http://example.com/myGroups"
}
`)

	CheckGet(reg, "1 res,inline 1 level", "http://example.com?inline=myGroups", `{
  "specVersion": "0.5",
  "id": "666-1234-1234",
  "self": "http://example.com/",

  "myGroups": {
    "g1": {
      "id": "g1",
      "name": "g1",
      "self": "http://example.com/myGroups/g1",
      "ext1": "extvalue",

      "ressCount": 1,
      "ressUrl": "http://example.com/myGroups/g1/ress"
    }
  },
  "myGroupsCount": 1,
  "myGroupsUrl": "http://example.com/myGroups"
}
`)

	CheckGet(reg, "1 deep", "http://example.com/myGroups?inline", `{
  "g1": {
    "id": "g1",
    "name": "g1",
    "self": "http://example.com/myGroups/g1",
    "ext1": "extvalue",

    "ress": {
      "r1": {
        "id": "r1",
        "self": "http://example.com/myGroups/g1/ress/r1",

        "versions": {},
        "versionsCount": 0,
        "versionsUrl": "http://example.com/myGroups/g1/ress/r1/versions"
      }
    },
    "ressCount": 1,
    "ressUrl": "http://example.com/myGroups/g1/ress"
  }
}
`)

	// Test setting Resource stuff, not Latest version stuff
	r1.Set(".name", "unique")
	Check(r1.Extensions["name"] == "unique", "r1.Name != unique")
	r1.Set(".Int", 345)
	Check(r1.Extensions["Int"] == 345, "r1.Int != 345")
	r3 := g1.FindResource("ress", "r1")
	Check(registry.ToJSON(r1) == registry.ToJSON(r3), "r3 != r1")
	Check(r3.Extensions["Int"] == 345, "r3.Int != 345")

	CheckGet(reg, "r1 props", "http://example.com/myGroups?inline", `{
  "g1": {
    "id": "g1",
    "name": "g1",
    "self": "http://example.com/myGroups/g1",
    "ext1": "extvalue",

    "ress": {
      "r1": {
        "id": "r1",
        "name": "unique",
        "self": "http://example.com/myGroups/g1/ress/r1",
        "Int": 345,

        "versions": {},
        "versionsCount": 0,
        "versionsUrl": "http://example.com/myGroups/g1/ress/r1/versions"
      }
    },
    "ressCount": 1,
    "ressUrl": "http://example.com/myGroups/g1/ress"
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

	CheckGet(reg, "r1 props", "http://example.com/myGroups?inline", `{
  "g1": {
    "id": "g1",
    "name": "g1",
    "self": "http://example.com/myGroups/g1",
    "ext1": "extvalue",

    "ress": {
      "r1": {
        "id": "r1",
        "name": "v1",
        "epoch": 42,
        "self": "http://example.com/myGroups/g1/ress/r1",
        "latestId": "v1",
        "Int": 345,
        "ext1": "someext",
        "ext2": 234,

        "versions": {
          "v1": {
            "id": "v1",
            "name": "v1",
            "self": "http://example.com/myGroups/g1/ress/r1/versions/v1",
            "epoch": 42,
            "ext1": "someext",
            "ext2": 234
          }
        },
        "versionsCount": 1,
        "versionsUrl": "http://example.com/myGroups/g1/ress/r1/versions"
      }
    },
    "ressCount": 1,
    "ressUrl": "http://example.com/myGroups/g1/ress"
  }
}
`)

	// Test Latest version stuff
	r1.Set("name", r1.ID)
	r1.Set("epoch", 42)
	r1.Set("ext1", "someext")
	r1.Set("ext2", 123)
	Check(r1.GetLatest().Extensions["ext2"] == 123, "r1.Ext isn't an int")
	r2 := g1.FindResource("ress", "r1")
	Check(registry.ToJSON(r1) == registry.ToJSON(r2), "r2 != r1")
	Check(r1.FindVersion("v3") == nil, "v3 should be nil")
	Check(r2.FindVersion("v3") == nil, "v3 should be nil")

	log.Printf("ALL TESTS PASSED")
	// reg.Delete()
	return reg
}

func LoadSample() *registry.Registry {
	reg, err := registry.NewRegistry("987")
	ErrFatalf(err, "Error creating new registry: %s", err)

	reg.Set("BaseURL", "http://soaphub.org:8585/")
	reg.Set("name", "Test Registry")
	reg.Set("description", "A test reg")
	reg.Set("specVersion", "0.5")
	reg.Set("docs", "https://github.com/duglin/xreg-github")

	gm, _ := reg.AddGroupModel("agroups", "group", "")
	_, err = gm.AddResourceModel("ress", "res", 2)

	gm, _ = reg.AddGroupModel("zgroups", "group", "")
	_, err = gm.AddResourceModel("ress", "res", 2)

	gm, _ = reg.AddGroupModel("endpoints", "endpoint", "")
	_, err = gm.AddResourceModel("defs", "def", 2)
	_, err = gm.AddResourceModel("adefs", "def", 2)
	_, err = gm.AddResourceModel("zdefs", "def", 2)

	g := reg.FindOrAddGroup("endpoints", "e1")
	g.Set("name", "end1")
	g.Set("epoch", 1)
	g.Set("ext", "ext1")
	g.Set("tags.stage", "dev")
	g.Set("tags.stale", "true")

	r := g.FindOrAddResource("defs", "created")
	v := r.FindOrAddVersion("v1")
	v.Set("name", "blobCreated")
	v.Set("epoch", 2)

	v = r.FindOrAddVersion("v2")
	v.Set("name", "blobCreated")
	v.Set("epoch", 4)
	r.Set(".latestId", "v2")

	r = g.FindOrAddResource("defs", "deleted")
	v = r.FindOrAddVersion("v1.0")
	v.Set("name", "blobDeleted")
	v.Set("epoch", 3)

	g = reg.FindOrAddGroup("endpoints", "e2")
	g.Set("name", "end1")
	g.Set("epoch", 1)
	g.Set("ext", "ext1")

	return reg
}

func ErrFatalf(err error, str string, args ...interface{}) {
	if err == nil {
		return
	}
	log.Fatalf(str, args...)
}

func main() {
	Token = strings.TrimSpace(Token)

	Reg = DoTests()
	// Reg.Delete()

	Reg = LoadGitRepo("APIs-guru", "openapi-directory")
	// Reg = LoadSample()

	if tmp := os.Getenv("PORT"); tmp != "" {
		Port = tmp
	}

	http.HandleFunc("/", handler)
	log.Printf("Listening on %s", Port)
	http.ListenAndServe(":"+Port, nil)
}
