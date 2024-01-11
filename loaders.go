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

func ErrFatalf(err error, args ...any) {
	if err == nil {
		return
	}
	format := "%s"
	if len(args) > 0 {
		format = args[0].(string)
		args = args[1:]
	} else {
		args = []any{err}
	}
	log.Printf(format, args...)
	registry.ShowStack()
	os.Exit(1)
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

		ErrFatalf(reg.Set("#baseURL", "http://soaphub.org:8585/"))
		ErrFatalf(reg.Set("name", "APIs-guru Registry"))
		ErrFatalf(reg.Set("description", "xRegistry view of github.com/APIs-guru/openapi-directory"))
		ErrFatalf(reg.Set("documentation", "https://github.com/duglin/xreg-github"))
		ErrFatalf(reg.Refresh())
		// log.VPrintf(3, "New registry:\n%#v", reg)

		// TODO Support "model" being part of the Registry struct above
	}

	g, err := reg.Model.AddGroupModel("apiproviders", "apiprovider")
	ErrFatalf(err)
	_, err = g.AddResourceModel("apis", "api", 2, true, true, true)
	ErrFatalf(err)
	_, err = g.AddAttr("xxx", registry.INTEGER)
	ErrFatalf(err)
	_, err = g.AddAttr("yyy", registry.STRING)
	ErrFatalf(err)
	_, err = g.AddAttr("zzz", registry.STRING)
	ErrFatalf(err)

	g, err = reg.Model.AddGroupModel("schemagroups", "schemagroup")
	ErrFatalf(err)
	_, err = g.AddResourceModel("schemas", "schema", 1, true, true, true)
	ErrFatalf(err)
	// reg.Model.Save()

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

		group, err := reg.FindGroup("apiproviders", parts[0])
		ErrFatalf(err)

		if group == nil {
			group, err = reg.AddGroup("apiproviders", parts[0])
			ErrFatalf(err)
		}

		ErrFatalf(group.Set("name", group.UID))
		ErrFatalf(group.Set("modifiedby", "me"))
		ErrFatalf(group.Set("modifiedon", "2024-01-01T12:00:00Z"))
		ErrFatalf(group.Set("epoch", 5))
		ErrFatalf(group.Set("xxx", 5))
		ErrFatalf(group.Set("yyy", "6"))
		ErrFatalf(group.Set("zzz", "6"))

		ErrFatalf(group.Set("modifiedon", nil)) // delete prop
		ErrFatalf(group.Set("zzz", nil))        // delete prop

		// group2 := reg.FindGroup("apiproviders", parts[0])
		// log.Printf("Find Group:\n%s", registry.ToJSON(group2))

		resName := "core"
		verIndex := 1
		if len(parts) == 4 {
			resName = parts[1]
			verIndex++
		}

		res, err := group.AddResource("apis", resName, "v1")
		ErrFatalf(err)

		g2, err := reg.FindGroup("schemagroups", parts[0])
		ErrFatalf(err, "FindGroup(%s/%s): %s", "schemagroups", parts[0], err)

		if g2 == nil {
			g2, err = reg.AddGroup("schemagroups", parts[0])
			ErrFatalf(err, "AddGroup(%s/%s): %s", "schemagroups", parts[0], err)
		}
		ErrFatalf(g2.Set("name", group.Get("name")))
		/*
			r2,err := g2.AddResource("schemas", resName, parts[verIndex])
			ErrFatalf(err)
			v2,err := r2.FindVersion(parts[verIndex])
			ErrFatalf(err)
			v2.Name = parts[verIndex+1]
			v2.Format = "openapi/3.0.6"
		*/
		version, err := res.FindVersion(parts[verIndex])
		ErrFatalf(err)
		if version != nil {
			log.Fatalf("Have more than one file per version: %s\n", header.Name)
		}

		buf := &bytes.Buffer{}
		io.Copy(buf, reader)
		version, err = res.AddVersion(parts[verIndex])
		ErrFatalf(err)
		ErrFatalf(version.Set("name", parts[verIndex+1]))
		ErrFatalf(version.Set("format", "openapi/3.0.6"))

		// Don't upload the file contents into the registry. Instead just
		// give the registry a URL to it and ask it to server it via proxy.
		// We could have also just set the resourceURI to the file but
		// I wanted the URL to the file to be the registry and not github
		base := "https://raw.githubusercontent.com/APIs-guru/" +
			"openapi-directory/main/APIs/"
		// version.Set("resourceURL", base + header.Name[i+6:])
		// version.Set("resource", buf.Bytes())
		ErrFatalf(version.Set("#resourceProxyURL", base+header.Name[i+6:]))
	}

	return reg
}

func LoadDirsSample(reg *registry.Registry) *registry.Registry {
	var err error
	log.VPrintf(1, "Loading registry '%s'", "SampleRegistry")
	if reg == nil {
		reg, err = registry.NewRegistry("SampleRegistry")
		ErrFatalf(err, "Error creating new registry: %s", err)

		ErrFatalf(reg.Set("#baseURL", "http://soaphub.org:8585/"))
		ErrFatalf(reg.Set("name", "Test Registry"))
		ErrFatalf(reg.Set("description", "A test reg"))
		ErrFatalf(reg.Set("documentation", "https://github.com/duglin/xreg-github"))

		ErrFatalf(reg.Set("labels.stage", "prod"))

		_, err = reg.Model.AddAttr("bool1", registry.BOOLEAN)
		ErrFatalf(err)
		_, err = reg.Model.AddAttr("int1", registry.INTEGER)
		ErrFatalf(err)
		_, err = reg.Model.AddAttr("dec1", registry.DECIMAL)
		ErrFatalf(err)
		_, err = reg.Model.AddAttr("str1", registry.STRING)
		ErrFatalf(err)
		_, err = reg.Model.AddAttrMap("map1", registry.NewItem(registry.STRING))
		ErrFatalf(err)
		_, err = reg.Model.AddAttrArray("arr1", registry.NewItem(registry.STRING))
		ErrFatalf(err)

		_, err = reg.Model.AddAttrMap("emptymap", registry.NewItem(registry.STRING))
		ErrFatalf(err)
		_, err = reg.Model.AddAttrArray("emptyarr", registry.NewItem(registry.STRING))
		ErrFatalf(err)
		_, err = reg.Model.AddAttrObj("emptyobj")
		ErrFatalf(err)

		item := registry.NewItemObj()
		item.AddAttr("inint", registry.INTEGER)
		_, err = reg.Model.AddAttrMap("mapobj", item)
		ErrFatalf(err)

		_, err = reg.Model.AddAttrArray("arrmap",
			registry.NewItemMap(
				registry.NewItem(registry.STRING)))
		ErrFatalf(err)

		ErrFatalf(reg.Set("bool1", true))
		ErrFatalf(reg.Set("int1", 1))
		ErrFatalf(reg.Set("dec1", 1.1))
		ErrFatalf(reg.Set("str1", "hi"))
		ErrFatalf(reg.Set("map1.k1", "v1"))

		ErrFatalf(reg.Set("emptymap", map[string]int{}))
		ErrFatalf(reg.Set("emptyarr", []int{}))
		ErrFatalf(reg.Set("emptyobj", struct{}{}))

		ErrFatalf(reg.Set("arr1[1]", "arr1-value"))
		ErrFatalf(reg.Set("mapobj.mapkey.inint", 5))
		ErrFatalf(reg.Set("mapobj['cool.key'].inint", 666))
		ErrFatalf(reg.Set("arrmap[1].key1", "arrmapk1-value"))
	}

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	ErrFatalf(err)
	_, err = gm.AddResourceModel("files", "file", 2, true, true, true)
	ErrFatalf(err)

	g, err := reg.AddGroup("dirs", "dir1")
	ErrFatalf(err)
	ErrFatalf(g.Set("labels.private", "true"))
	r, err := g.AddResource("files", "f1", "v1")
	ErrFatalf(g.Set("labels.private", "true"))
	_, err = r.AddVersion("v2")
	ErrFatalf(r.Set("labels.stage", "dev"))
	ErrFatalf(r.Set("labels.none", ""))

	return reg
}

func LoadEndpointsSample(reg *registry.Registry) *registry.Registry {
	var err error
	log.VPrintf(1, "Loading registry '%s'", "EndpointsRegistry")
	if reg == nil {
		reg, err = registry.FindRegistry("EndpointsRegistry")
		ErrFatalf(err)
		if reg != nil {
			return reg
		}

		reg, err = registry.NewRegistry("EndpointsRegistry")
		ErrFatalf(err, "Error creating new registry: %s", err)

		ErrFatalf(reg.Set("#baseURL", "http://soaphub.org:8585/"))
		ErrFatalf(reg.Set("name", "Test Registry"))
		ErrFatalf(reg.Set("description", "A test reg"))
		ErrFatalf(reg.Set("documentation", "https://github.com/duglin/xreg-github"))
	}

	gm, err := reg.Model.AddGroupModel("endpoints", "endpoint")
	ErrFatalf(err)
	_, err = gm.AddAttr("ext", registry.STRING)
	ErrFatalf(err)

	_, err = gm.AddResourceModel("definitions", "definition", 2, true, true, true)
	ErrFatalf(err)

	g, err := reg.AddGroup("endpoints", "e1")
	ErrFatalf(err)
	ErrFatalf(g.Set("name", "end1"))
	ErrFatalf(g.Set("epoch", 1))
	ErrFatalf(g.Set("ext", "ext1"))
	ErrFatalf(g.Set("labels.stage", "dev"))
	ErrFatalf(g.Set("labels.stale", "true"))

	r, err := g.AddResource("definitions", "created", "v1")
	ErrFatalf(err)
	v, err := r.FindVersion("v1")
	ErrFatalf(err)
	ErrFatalf(v.Set("name", "blobCreated"))
	ErrFatalf(v.Set("epoch", 2))

	v, err = r.AddVersion("v2")
	ErrFatalf(err)
	ErrFatalf(v.Set("name", "blobCreated"))
	ErrFatalf(v.Set("epoch", 4))
	ErrFatalf(r.SetLatest(v))

	r, err = g.AddResource("definitions", "deleted", "v1.0")
	ErrFatalf(err)
	v, err = r.FindVersion("v1.0")
	ErrFatalf(err)
	ErrFatalf(v.Set("name", "blobDeleted"))
	ErrFatalf(v.Set("epoch", 3))

	g, err = reg.AddGroup("endpoints", "e2")
	ErrFatalf(err)
	ErrFatalf(g.Set("name", "end1"))
	ErrFatalf(g.Set("epoch", 1))
	ErrFatalf(g.Set("ext", "ext1"))

	return reg
}
