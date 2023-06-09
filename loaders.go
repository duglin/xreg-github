package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	_ "embed"
	"io"
	"io/ioutil"
	"strings"

	log "github.com/duglin/dlog"
	"github.com/duglin/xreg-github/registry"
)

//go:embed .github
var Token string
var Secret string

func ErrFatalf(err error, format string, args ...any) {
	if err == nil {
		return
	}
	log.Fatalf(format, args...)
}

func LoadGitRepo(orgName string, repoName string) *registry.Registry {
	log.VPrintf(1, "Loading registry '%s/%s'", orgName, repoName)
	Token = strings.TrimSpace(Token)

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
	ErrFatalf(err, "Error creating new registry: %s", err)
	// log.VPrintf(3, "New registry:\n%#v", reg)

	reg.Set("baseURL", "http://soaphub.org:8585/")
	reg.Set("name", "APIs-guru Registry")
	reg.Set("description", "xRegistry view of github.com/APIs-guru/openapi-directory")
	reg.Set("specVersion", "0.5")
	reg.Set("docs", "https://github.com/duglin/xreg-github")
	err = reg.Refresh()
	ErrFatalf(err, "Error refeshing registry: %s", err)
	// log.VPrintf(3, "New registry:\n%#v", reg)

	// TODO Support "model" being part of the Registry struct above

	g, _ := reg.AddGroupModel("apiProviders", "apiProvider", "")
	_, err = g.AddResourceModel("apis", "api", 2, true, true)

	g, _ = reg.AddGroupModel("schemaGroups", "schemaGroup", "")
	_, err = g.AddResourceModel("schemas", "schema", 1, true, true)

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
		// version.Set("resourceURL", base + header.Name[i+6:])
		// version.Set("resourceContent", buf.Bytes())
		version.Set("resourceProxyURL", base+header.Name[i+6:])
	}

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
	_, err = gm.AddResourceModel("ress", "res", 2, true, true)

	gm, _ = reg.AddGroupModel("zgroups", "group", "")
	_, err = gm.AddResourceModel("ress", "res", 2, true, true)

	gm, _ = reg.AddGroupModel("endpoints", "endpoint", "")
	_, err = gm.AddResourceModel("defs", "def", 2, true, true)
	_, err = gm.AddResourceModel("adefs", "def", 2, true, true)
	_, err = gm.AddResourceModel("zdefs", "def", 2, true, true)

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
