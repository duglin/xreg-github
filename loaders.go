package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

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
		reg, err = registry.FindRegistry("API-Guru")
		ErrFatalf(err)
		if reg != nil {
			return reg
		}

		reg, err = registry.NewRegistry("API-Guru")
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

	iter := 0

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

		// Just a subset for now
		if strings.Index(header.Name, "/docker.com/") < 0 &&
			strings.Index(header.Name, "/adobe.com/") < 0 &&
			strings.Index(header.Name, "/fec.gov/") < 0 &&
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
		ErrFatalf(group.Set("modifiedon", time.Now().Format(time.RFC3339)))
		ErrFatalf(group.Set("epoch", 5))

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
		switch iter % 3 {
		case 0:
			ErrFatalf(version.Set("#resource", buf.Bytes()))
		case 1:
			ErrFatalf(version.Set("#resourceURL", base+header.Name[i+6:]))
		case 2:
			ErrFatalf(version.Set("#resourceProxyURL", base+header.Name[i+6:]))
		}
		iter++
	}

	return reg
}

func LoadDirsSample(reg *registry.Registry) *registry.Registry {
	var err error
	log.VPrintf(1, "Loading registry '%s'", "TestRegistry")
	if reg == nil {
		reg, err = registry.FindRegistry("TestRegistry")
		ErrFatalf(err)
		if reg != nil {
			return reg
		}

		reg, err = registry.NewRegistry("TestRegistry")
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

		item := registry.NewItemObject()
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
	log.VPrintf(1, "Loading registry '%s'", "Endpoints")
	if reg == nil {
		reg, err = registry.FindRegistry("Endpoints")
		ErrFatalf(err)
		if reg != nil {
			return reg
		}

		reg, err = registry.NewRegistry("Endpoints")
		ErrFatalf(err, "Error creating new registry: %s", err)

		ErrFatalf(reg.Set("#baseURL", "http://soaphub.org:8585/"))
		ErrFatalf(reg.Set("name", "Endpoints Registry"))
		ErrFatalf(reg.Set("description", "An impl of the endpoints spec"))
		ErrFatalf(reg.Set("documentation", "https://github.com/duglin/xreg-github"))
	}

	ep, err := reg.Model.AddGroupModel("endpoints", "endpoint")
	ErrFatalf(err)
	attr, err := ep.AddAttr("usage", registry.STRING)
	ErrFatalf(err)
	attr.Required = true
	_, err = ep.AddAttr("origin", registry.URI)
	ErrFatalf(err)
	_, err = ep.AddAttr("channel", registry.STRING)
	ErrFatalf(err)
	attr, err = ep.AddAttrObj("deprecated")
	ErrFatalf(err)
	_, err = attr.AddAttr("effective", registry.TIMESTAMP)
	ErrFatalf(err)
	_, err = attr.AddAttr("removal", registry.TIMESTAMP)
	ErrFatalf(err)
	_, err = attr.AddAttr("alternative", registry.URL)
	ErrFatalf(err)
	_, err = attr.AddAttr("docs", registry.URL)
	ErrFatalf(err)

	config, err := attr.AddAttrObj("config")
	ErrFatalf(err)
	_, err = config.AddAttr("protocol", registry.STRING)
	ErrFatalf(err)
	obj, err := config.AddAttrMap("endpoints", registry.NewItemObject())
	ErrFatalf(err)
	_, err = obj.Item.AddAttr("*", registry.ANY)
	ErrFatalf(err)

	auth, err := config.AddAttrObj("authorization")
	ErrFatalf(err)
	attr, err = auth.AddAttr("type", registry.STRING)
	ErrFatalf(err)
	attr, err = auth.AddAttr("resourceurl", registry.STRING)
	ErrFatalf(err)
	attr, err = auth.AddAttr("authorityurl", registry.STRING)
	ErrFatalf(err)
	attr, err = auth.AddAttrArray("grant_types", registry.NewItem(registry.STRING))
	ErrFatalf(err)

	_, err = config.AddAttr("strict", registry.BOOLEAN)
	ErrFatalf(err)

	_, err = config.AddAttrMap("options", registry.NewItem(registry.ANY))
	ErrFatalf(err)

	_, err = ep.AddResourceModel("definitions", "definition", 2, true, true, true)
	ErrFatalf(err)

	// End of model

	g, err := reg.AddGroup("endpoints", "e1")
	ErrFatalf(err)
	ErrFatalf(g.Set("name", "end1"))
	ErrFatalf(g.Set("epoch", 1))
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

	return reg
}

func LoadMessagesSample(reg *registry.Registry) *registry.Registry {
	var err error
	log.VPrintf(1, "Loading registry '%s'", "Messages")
	if reg == nil {
		reg, err = registry.FindRegistry("Messages")
		ErrFatalf(err)
		if reg != nil {
			return reg
		}

		reg, err = registry.NewRegistry("Messages")
		ErrFatalf(err, "Error creating new registry: %s", err)

		reg.Set("#baseURL", "http://soaphub.org:8585/")
		reg.Set("name", "Messages Registry")
		reg.Set("description", "An impl of the messages spec")
		reg.Set("documentation", "https://github.com/duglin/xreg-github")
	}

	msgs, _ := reg.Model.AddGroupModel("messagegroups", "messagegroup")
	msgs.AddAttr("binding", registry.STRING)

	msg, _ := msgs.AddResourceModel("messages", "message", 1, true, true, false)

	// Modify core attribute
	attr, _ := msg.AddAttr("format", registry.STRING)
	attr.Required = true

	msg.AddAttr("basedefinitionurl", registry.URL)

	meta, _ := msg.AddAttrObj("metadata")
	meta.AddAttr("required", registry.BOOLEAN)
	meta.AddAttr("description", registry.STRING)
	meta.AddAttr("value", registry.ANY)
	meta.AddAttr("type", registry.STRING)
	meta.AddAttr("specurl", registry.URL)

	obj := registry.NewItemObject()
	meta.AddAttrMap("attributes", obj)
	obj.AddAttr("type", registry.STRING)
	obj.AddAttr("value", registry.ANY)
	obj.AddAttr("required", registry.BOOLEAN)

	meta.AddAttr("binding", registry.STRING)
	meta.AddAttrMap("message", registry.NewItem(registry.ANY))

	meta.AddAttr("schemaformat", registry.STRING)
	meta.AddAttr("schema", registry.ANY)
	meta.AddAttr("schemaurl", registry.URL)

	// End of model

	return reg
}

func LoadSchemasSample(reg *registry.Registry) *registry.Registry {
	var err error
	log.VPrintf(1, "Loading registry '%s'", "Schemas")
	if reg == nil {
		reg, err = registry.FindRegistry("Schemas")
		ErrFatalf(err)
		if reg != nil {
			return reg
		}

		reg, err = registry.NewRegistry("Schemas")
		ErrFatalf(err, "Error creating new registry: %s", err)

		reg.Set("#baseURL", "http://soaphub.org:8585/")
		reg.Set("name", "Schemas Registry")
		reg.Set("description", "An impl of the schemas spec")
		reg.Set("documentation", "https://github.com/duglin/xreg-github")
	}

	msgs, _ := reg.Model.AddGroupModel("schemagroups", "schemagroup")
	msgs.AddResourceModel("schemas", "schema", 0, true, true, true)

	// End of model

	return reg
}
