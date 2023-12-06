package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"strings"

	log "github.com/duglin/dlog"
	"github.com/duglin/xreg-github/registry"
)

var Token string
var Secret string

func ErrFatalf(err error, format string, args ...any) {
	if err == nil {
		return
	}
	log.Fatalf(format, args...)
}

func init() {
	if tmp := os.Getenv("githubToken"); tmp != "" {
		Token = tmp
	} else {
		if buf, _ := os.ReadFile(".github"); len(buf) > 0 {
			Token = string(buf)
		}
	}
}

func LoadAPIGuru(reg *registry.Registry, orgName string, repoName string) *registry.Registry {
	var err error
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

	if reg == nil {
		reg, err = registry.NewRegistry("123-4567-3456")
		ErrFatalf(err, "Error creating new registry: %s", err)
		// log.VPrintf(3, "New registry:\n%#v", reg)

		reg.Set("#baseURL", "http://soaphub.org:8585/")
		reg.Set("name", "APIs-guru Registry")
		reg.Set("description", "xRegistry view of github.com/APIs-guru/openapi-directory")
		reg.Set("documentation", "https://github.com/duglin/xreg-github")
		err = reg.Refresh()
		ErrFatalf(err, "Error refeshing registry: %s", err)
		// log.VPrintf(3, "New registry:\n%#v", reg)

		// TODO Support "model" being part of the Registry struct above
	}

	g, _ := reg.Model.AddGroupModel("apiProviders", "apiProvider")
	_, err = g.AddResourceModel("apis", "api", 2, true, true, true)
	g.AddAttr("xxx", registry.INTEGER)
	g.AddAttr("yyy", registry.STRING)
	g.AddAttr("zzz", registry.STRING)

	g, _ = reg.Model.AddGroupModel("schemaGroups", "schemaGroup")
	_, err = g.AddResourceModel("schemas", "schema", 1, true, true, true)

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

		group, err := reg.FindGroup("apiProviders", parts[0])
		ErrFatalf(err, "FindGroup: %s", err)

		if group == nil {
			group, err = reg.AddGroup("apiProviders", parts[0])
			ErrFatalf(err, "AddGroup: %s", err)
		}

		group.Set("name", group.UID)
		group.Set("modifiedBy", "me")
		group.Set("modifiedOn", "noon")
		group.Set("epoch", 5)
		group.Set("xxx", 5)
		group.Set("yyy", "6")
		group.Set("zzz", "6")

		group.Set("modifiedOn", nil) // delete prop
		group.Set("zzz", nil)        // delete prop

		// group2 := reg.FindGroup("apiProviders", parts[0])
		// log.Printf("Find Group:\n%s", registry.ToJSON(group2))

		resName := "core"
		verIndex := 1
		if len(parts) == 4 {
			resName = parts[1]
			verIndex++
		}

		res, _ := group.AddResource("apis", resName, "v1")

		g2, err := reg.FindGroup("schemaGroups", parts[0])
		ErrFatalf(err, "FindGroup(%s/%s): %s", "schemaGroups", parts[0], err)

		if g2 == nil {
			g2, err = reg.AddGroup("schemaGroups", parts[0])
			ErrFatalf(err, "AddGroup(%s/%s): %s", "schemaGroups", parts[0], err)
		}
		g2.Set("name", group.Get("name"))
		/*
			r2,_ := g2.AddResource("schemas", resName, parts[verIndex])
			v2,_ := r2.FindVersion(parts[verIndex])
			v2.Name = parts[verIndex+1]
			v2.Format = "openapi/3.0.6"
		*/
		version, _ := res.FindVersion(parts[verIndex])
		if version != nil {
			log.Fatalf("Have more than one file per version: %s\n", header.Name)
		}

		buf := &bytes.Buffer{}
		io.Copy(buf, reader)
		version, _ = res.AddVersion(parts[verIndex])
		version.Set("name", parts[verIndex+1])
		version.Set("format", "openapi/3.0.6")

		// Don't upload the file contents into the registry. Instead just
		// give the registry a URL to it and ask it to server it via proxy.
		// We could have also just set the resourceURI to the file but
		// I wanted the URL to the file to be the registry and not github
		base := "https://raw.githubusercontent.com/APIs-guru/" +
			"openapi-directory/main/APIs/"
		// version.Set("resourceURL", base + header.Name[i+6:])
		// version.Set("resource", buf.Bytes())
		version.Set("#resourceProxyURL", base+header.Name[i+6:])
	}

	return reg
}

func LoadDirsSample(reg *registry.Registry) *registry.Registry {
	var err error
	log.VPrintf(1, "Loading registry '%s'", "SampleRegistry")
	if reg == nil {
		reg, err = registry.NewRegistry("SampleRegistry")
		ErrFatalf(err, "Error creating new registry: %s", err)

		reg.Set("#baseURL", "http://soaphub.org:8585/")
		reg.Set("name", "Test Registry")
		reg.Set("description", "A test reg")
		reg.Set("documentation", "https://github.com/duglin/xreg-github")

		reg.Set("labels.stage", "prod")

		reg.Model.AddAttribute(&registry.Attribute{Name: "bool1",
			Type: registry.BOOLEAN})
		reg.Model.AddAttribute(&registry.Attribute{Name: "int1",
			Type: registry.INTEGER})
		reg.Model.AddAttribute(&registry.Attribute{Name: "dec1",
			Type: registry.DECIMAL})
		reg.Model.AddAttribute(&registry.Attribute{Name: "str1",
			Type: registry.STRING})
		reg.Model.AddAttribute(&registry.Attribute{Name: "map1",
			Type:    registry.MAP,
			KeyType: registry.STRING, ItemType: registry.STRING})

		reg.Set("bool1", true)
		reg.Set("int1", 1)
		reg.Set("dec1", 1.1)
		reg.Set("str1", "hi")
		reg.Set("map1.k1", "v1")
	}

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	_, err = gm.AddResourceModel("files", "file", 2, true, true, true)

	g, _ := reg.AddGroup("dirs", "dir1")
	g.Set("labels.private", "true")
	r, _ := g.AddResource("files", "f1", "v1")
	r.AddVersion("v2")
	r.Set("labels.stage", "dev")
	r.Set("labels.none", "")

	return reg
}

func LoadEndpointsSample(reg *registry.Registry) *registry.Registry {
	var err error
	log.VPrintf(1, "Loading registry '%s'", "EndpointsRegistry")
	if reg == nil {
		reg, _ = registry.FindRegistry("EndpointsRegistry")
		if reg != nil {
			return reg
		}

		reg, err = registry.NewRegistry("EndpointsRegistry")
		ErrFatalf(err, "Error creating new registry: %s", err)

		reg.Set("#baseURL", "http://soaphub.org:8585/")
		reg.Set("name", "Test Registry")
		reg.Set("description", "A test reg")
		reg.Set("documentation", "https://github.com/duglin/xreg-github")
	}

	gm, _ := reg.Model.AddGroupModel("endpoints", "endpoint")
	gm.AddAttribute(&registry.Attribute{Name: "ext", Type: "string"})

	_, err = gm.AddResourceModel("definitions", "definition", 2, true, true, true)

	g, _ := reg.AddGroup("endpoints", "e1")
	g.Set("name", "end1")
	g.Set("epoch", 1)
	g.Set("ext", "ext1")
	g.Set("labels.stage", "dev")
	g.Set("labels.stale", "true")

	r, _ := g.AddResource("definitions", "created", "v1")
	v, _ := r.FindVersion("v1")
	v.Set("name", "blobCreated")
	v.Set("epoch", 2)

	v, _ = r.AddVersion("v2")
	v.Set("name", "blobCreated")
	v.Set("epoch", 4)
	r.SetLatest(v)

	r, _ = g.AddResource("definitions", "deleted", "v1.0")
	v, _ = r.FindVersion("v1.0")
	v.Set("name", "blobDeleted")
	v.Set("epoch", 3)

	g, _ = reg.AddGroup("endpoints", "e2")
	g.Set("name", "end1")
	g.Set("epoch", 1)
	g.Set("ext", "ext1")

	return reg
}
