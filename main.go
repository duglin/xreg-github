package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/duglin/xreg-github/registry"
)

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

	reg := &registry.Registry{
		// BaseURL: "#",
		// BaseURL: "https://example.com/",
		BaseURL: "http://soaphub.org:8585/",

		Model: &registry.Model{
			Groups: map[string]*registry.GroupModel{
				"apiProviders": &registry.GroupModel{
					Singular: "apiProvider",
					Plural:   "apiProviders",

					Resources: map[string]*registry.ResourceModel{
						"apis": &registry.ResourceModel{
							Singular: "api",
							Plural:   "apis",
							Versions: -1,
						},
					},
				},
				"schemaGroups": &registry.GroupModel{
					Singular: "schemaGroup",
					Plural:   "schemaGroups",

					Resources: map[string]*registry.ResourceModel{
						"schema": &registry.ResourceModel{
							Singular: "schema",
							Plural:   "schemas",
							Versions: -1,
						},
					},
				},
			},
		},

		ID:          "123-1234-1234",
		Name:        "APIs-guru Registry",
		Description: "xRegistry view of github.com/APIs-guru/openapi-directory",
		SpecVersion: "0.4",
		Docs:        "https://github.com/duglin/xreg-github",
	}

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

		/*
			if strings.Index(header.Name, "/docker.com/") < 0 &&
				strings.Index(header.Name, "/apiz.ebay.com/") < 0 {
				continue
			}
		*/

		parts := strings.Split(strings.Trim(header.Name[i+6:], "/"), "/")
		// org/service/version/file
		// org/version/file

		group := reg.FindOrAddGroup("apiProviders", parts[0])

		resName := "core"
		verIndex := 1
		if len(parts) == 4 {
			resName = parts[1]
			verIndex++
		}

		res := group.FindOrAddResource("apis", resName)

		version := res.FindVersion(parts[verIndex])
		if version != nil {
			fmt.Printf("group: %s\nresource: %s\n", parts[0], resName)
			log.Fatalf("Have more than one file per version: %s\n", header.Name)
		}

		buf := &bytes.Buffer{}
		io.Copy(buf, reader)
		version = res.FindOrAddVersion(parts[verIndex])
		version.Name = parts[verIndex+1]
		version.Type = "openapi/3.0.6"

		// Don't upload the file contents into the registry. Instead just
		// give the registry a URL to it and ask it to server it via proxy.
		// We could have also just set the resourceURI to the file but
		// I wanted the URL to the file to be the registry and not github
		base := "https://raw.githubusercontent.com/APIs-guru/" +
			"openapi-directory/main/APIs/"
		// version.Data["resourceURI"] = base + header.Name[i+6:]
		// version.Data["resourceContent"] = string(buf.Bytes())
		version.Data["resourceProxyURI"] = base + header.Name[i+6:]
	}

	return reg
}

func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL)

	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	baseURL := fmt.Sprintf("http://%s", r.Host)

	rFlags := &registry.RegistryFlags{
		Indent:      "  ",
		InlineAll:   false,
		InlinePaths: []string(nil),
		Self:        r.URL.Query().Has("self"),
		AsDoc:       r.URL.Query().Has("doc"),
		BaseURL:     baseURL,
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
}

func main() {
	Token = strings.TrimSpace(Token)
	orgName := "APIs-guru"
	repoName := "openapi-directory"

	Reg = LoadGitRepo(orgName, repoName)

	if tmp := os.Getenv("PORT"); tmp != "" {
		Port = tmp
	}

	http.HandleFunc("/", handler)
	log.Printf("Listening on %s", Port)
	http.ListenAndServe(":"+Port, nil)
}
