package registry

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

type RegistryFlags struct {
	BaseURL     string
	Indent      string
	InlineAll   bool
	InlinePaths []string
	Self        bool
	AsDoc       bool
	Filters     []string
}

type Registry struct {
	BaseURL string
	Model   *Model `json:"-"`

	ID          string
	Name        string
	Description string
	SpecVersion string
	Tags        map[string]string
	Docs        string

	GroupCollections map[string]*GroupCollection // groupType
}

func (reg *Registry) FindGroupModel(gTypePlural string) *GroupModel {
	for _, gModel := range reg.Model.Groups {
		if strings.EqualFold(gModel.Plural, gTypePlural) {
			return gModel
		}
	}
	return nil
}

func (reg *Registry) FindOrAddGroupCollection(gType string) *GroupCollection {
	gc, _ := reg.GroupCollections[gType]
	if gc == nil {
		gm := reg.Model.Groups[gType]
		if gm == nil {
			panic(fmt.Sprintf("Can't find GroupModel %q", gType))
		}

		gc = &GroupCollection{
			Registry:   reg,
			GroupModel: gm,
			Groups:     map[string]*Group{},
		}
		if reg.GroupCollections == nil {
			reg.GroupCollections = map[string]*GroupCollection{}
		}
		reg.GroupCollections[gType] = gc
	}
	return gc
}

func (reg *Registry) AddGroup(gt string, id string) *Group {
	gc := reg.FindOrAddGroupCollection(gt)
	return gc.NewGroup(id)
}

func (reg *Registry) FindOrAddGroup(gt string, id string) *Group {
	gc := reg.FindOrAddGroupCollection(gt)
	g := gc.Groups[id]
	if g != nil {
		return g
	}
	return gc.NewGroup(id)
}

func (reg *Registry) ToObject(ctx *Context) *Object {
	obj := NewObject()
	if reg == nil {
		return obj
	}

	obj.AddProperty("id", reg.ID)
	obj.AddProperty("name", reg.Name)
	obj.AddProperty("description", reg.Description)
	obj.AddProperty("specVersion", reg.SpecVersion)
	obj.AddProperty("self", ctx.DataURL())

	tags := NewObject()
	for key, value := range reg.Tags {
		tags.AddProperty(key, value)
	}
	if len(tags.Children) != 0 {
		obj.AddProperty("tags", tags)
	}

	obj.AddProperty("docs", reg.Docs)
	obj.AddProperty("", "")

	if ctx.ShouldInline("model") {
		ctx.ModelPush("model")
		obj.AddProperty("model", reg.Model.ToObject(ctx))
		ctx.ModelPop()
		obj.AddProperty("", "")
	}

	for i, key := range SortedKeys(reg.Model.Groups) {
		gType := reg.Model.Groups[key]
		gCollection := reg.GroupCollections[gType.Plural]

		obj.AddProperty(gType.Plural+"URL",
			URLBuild(ctx.DataURL(), gType.Plural))

		ctx.DataPush(gType.Plural)
		ctx.ModelPush(gType.Plural)
		groupObj := NewObject()
		if gCollection != nil {
			groupObj = gCollection.ToObject(ctx)
		}
		ctx.ModelPop()
		ctx.DataPop()

		obj.AddProperty(gType.Plural+"Count", groupObj.Len())
		if ctx.ShouldInline(gType.Plural) {
			obj.AddProperty(gType.Plural, groupObj)
		}
		if i+1 != len(reg.Model.Groups) {
			obj.AddProperty("", "")
		}
	}

	return obj
}

func (reg *Registry) ToJSON(ctx *Context) {
	ctx.Print("{\n")

	ctx.Indent()

	ctx.Printf("\t\"id\": \"%s\",\n", reg.ID)
	if reg.Name != "" {
		ctx.Printf("\t\"name\": \"%s\",\n", reg.Name)
	}
	if reg.Description != "" {
		ctx.Printf("\t\"description\": \"%s\",\n", reg.Description)
	}
	ctx.Printf("\t\"specVersion\": \"%s\",\n", reg.SpecVersion)
	ctx.Printf("\t\"self\": \"%s\",\n", ctx.DataURL())
	if len(reg.Tags) > 0 {
		ctx.Print("\t\"tags\": {\n")
		ctx.Indent()
		count := 0
		for key, value := range reg.Tags {
			count++
			ctx.Printf("\t\"%s\": \"%s\"", key, value)
			if count != len(reg.Tags) {
				ctx.Print(",")
			}
			ctx.Print("\n")
		}
		ctx.Outdent()
		ctx.Print("\t},\n")
	}
	if reg.Docs != "" {
		ctx.Printf("\t\"docs\": \"%s\",\n", reg.Docs)
	}

	// Add the Registry model
	if ctx.ShouldInline("model") {
		ctx.ModelPush("model")
		ctx.Print("\n")
		ctx.Print("\t\"model\": ")
		reg.Model.ToJSON(ctx)
		ctx.Print(",\n")
		ctx.ModelPop()
	}

	for gCount, key := range SortedKeys(reg.Model.Groups) {
		gType := reg.Model.Groups[key]
		gCollection := reg.GroupCollections[gType.Plural]

		if gCount > 0 {
			ctx.Print(",\n")
		}
		ctx.Print("\n")
		ctx.Printf("\t\"%sURL\": \"%s\",\n",
			gType.Plural,
			URLBuild(ctx.DataURL(), gType.Plural))

		l := 0
		if gCollection != nil {
			l = len(gCollection.Groups)
		}
		ctx.Printf("\t\"%sCount\": %d", gType.Plural, l)

		if ctx.ShouldInline(gType.Plural) && l > 0 {
			ctx.DataPush(gType.Plural)
			ctx.ModelPush(gType.Plural)
			ctx.Print(",\n")
			ctx.Printf("\t\"%s\": ", gType.Plural)
			gCollection.ToJSON(ctx)
			ctx.ModelPop()
			ctx.DataPop()
		}
	}
	ctx.Print("\n}")
}

func (r *Registry) Get(path string, rFlags *RegistryFlags) (string, error) {
	paths := strings.Split(strings.Trim(path, "/"), "/")
	for len(paths) > 0 && paths[0] == "" {
		paths = paths[1:]
	}

	ctx := &Context{
		Flags:         rFlags,
		BaseURL:       r.BaseURL,
		DataPath:      "",
		ModelPath:     "",
		currentIndent: "",
		indent:        rFlags.Indent,
		Filters:       ParseFilterExprs(rFlags.Filters),
	}

	if rFlags.BaseURL != "" {
		ctx.BaseURL = rFlags.BaseURL
	}
	ctx.BaseURL = strings.TrimRight(ctx.BaseURL, "/")

	if len(paths) == 0 {
		r.ToObject(ctx).ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	if len(paths) == 1 && paths[0] == "model" {
		r.Model.ToObject(ctx).ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	// GROUPs
	var gModel *GroupModel
	if gModel = r.FindGroupModel(paths[0]); gModel == nil {
		return "", fmt.Errorf("Unknown group %q", paths[0])
	}
	groupColl := r.GroupCollections[gModel.Plural]
	ctx.BaseURLPush(paths[0])

	if len(paths) == 1 {
		groupColl.ToObject(ctx).ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	// GROUPs/ID
	group := groupColl.Groups[paths[1]]
	if group == nil {
		return "", fmt.Errorf("Unknown group ID %q", paths[1])
	}
	ctx.BaseURLPush(paths[1])
	if len(paths) == 2 {
		group.ToObject(ctx).ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	// GROUPs/ID/RESOURCEs
	resColl := group.ResourceCollections[paths[2]]
	ctx.BaseURLPush(paths[2])
	if resColl == nil {
		return "", fmt.Errorf("Unknown rescource collection %q", paths[2])
	}
	if len(paths) == 3 {
		resColl.ToObject(ctx).ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	// GROUPs/ID/RESOURCEs/ID
	res := resColl.Resources[paths[3]]
	ctx.BaseURLPush(paths[3])
	if res == nil {
		return "", fmt.Errorf("Unknown resource ID %q", paths[3])
	}

	if len(paths) == 4 {
		if ctx.Flags.Self {
			res.ToObject(ctx).ToJson(&ctx.buffer, "", "  ")
			return ctx.buffer.String(), nil
		}
		latest := res.FindVersion(res.Latest)
		data, ok := latest.Data["resourceContent"]
		if ok {
			str, _ := data.(string)
			return str, nil
		}

		uri, ok := latest.Data["resourceProxyURI"]
		if ok {
			resp, err := http.Get(uri.(string))
			if err != nil {
				return "", err
			}
			if resp.StatusCode/100 != 2 {
				return "", fmt.Errorf("%s ->%d",
					uri, resp.StatusCode)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", err
			}
			return string(body), nil
		}
	}

	// GROUPs/ID/RESOURCEs/ID/versions
	if paths[4] != "versions" {
		return "", fmt.Errorf("Unknown subresource %q", paths[4])
	}
	verColl := res.VersionCollection
	ctx.BaseURLPush(paths[4])
	if len(paths) == 5 {
		verColl.ToObject(ctx).ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	// GROUPs/ID/RESOURCEs/ID/versions/ID
	ver := verColl.Versions[paths[5]]
	if ver == nil {
		return "", fmt.Errorf("Unknown version id %q", paths[5])
	}

	ctx.BaseURLPush(paths[5])
	if len(paths) == 6 {
		if ctx.Flags.Self {
			ver.ToObject(ctx).ToJson(&ctx.buffer, "", "  ")
			return ctx.buffer.String(), nil
		}
		data, ok := ver.Data["resourceContent"]
		if ok {
			str, _ := data.(string)
			return str, nil
		}

		uri, ok := ver.Data["resourceProxyURI"]
		if ok {
			resp, err := http.Get(uri.(string))
			if err != nil {
				return "", err
			}
			if resp.StatusCode/100 != 2 {
				return "", fmt.Errorf("%s ->%d",
					uri, resp.StatusCode)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", err
			}
			return string(body), nil
		}
	}

	return "", fmt.Errorf("Can't figure out what to do with %q",
		strings.Join(paths, "/"))

	return ctx.buffer.String(), nil

	/*
	   	for _, groupColl := range r.GroupCollections {
	   		if strings.ToLower(groupColl.GroupModel.Plural) != paths[0] {
	   			continue
	   		}
	   		ctx.BaseURLPush(paths[0])
	   		if len(paths) == 1 {
	   			groupColl.ToJSON(ctx)
	   			return ctx.Result(), nil
	   		}

	   		// GROUPs/ID
	   		for _, group := range groupColl.Groups {
	   			if group.ID != paths[1] {
	   				continue
	   			}
	   			ctx.BaseURLPush(paths[1])
	   			if len(paths) == 2 {
	   				group.ToJSON(ctx)
	   				return ctx.Result(), nil
	   			}

	   			// GROUPs/ID/RESOURCEs
	   			for _, resColl := range group.ResourceCollections {
	   				if strings.ToLower(resColl.ResourceModel.Plural) != paths[2] {
	   					continue
	   				}
	   				ctx.BaseURLPush(paths[2])
	   				if len(paths) == 3 {
	   					resColl.ToJSON(ctx)
	   					return ctx.Result(), nil
	   				}

	   				// GROUPs/ID/RESOURCEs/ID
	   				for _, res := range resColl.Resources {
	   					if res.ID != paths[3] {
	   						continue
	   					}
	   					ctx.BaseURLPush(paths[3])

	   					if len(paths) == 4 {
	   						if ctx.Flags.Self {
	   							res.ToJSON(ctx)
	   							return ctx.Result(), nil
	   						}
	   						latest := res.FindVersion(res.Latest)
	   						data, ok := latest.Data["resourceContent"]
	   						if ok {
	   							str, _ := data.(string)
	   							return str, nil
	   						}

	   						uri, ok := latest.Data["resourceProxyURI"]
	   						if ok {
	   							resp, err := http.Get(uri.(string))
	   							if err != nil {
	   								return "", err
	   							}
	   							if resp.StatusCode/100 != 2 {
	   								return "", fmt.Errorf("%s ->%d",
	   									uri, resp.StatusCode)
	   							}
	   							body, err := io.ReadAll(resp.Body)
	   							if err != nil {
	   								return "", err
	   							}
	   							return string(body), nil
	   						}
	   					}

	   					// GROUPs/ID/RESOURCEs/ID/versions
	   					if paths[4] == "versions" {
	   						ctx.BaseURLPush(paths[4])
	   						if len(paths) == 5 {
	   							res.VersionCollection.ToJSON(ctx)
	   							return ctx.Result(), nil
	   						}

	   						// GROUPs/ID/RESOURCEs/ID/versions/ID
	   						for _, ver := range res.VersionCollection.Versions {
	   							if ver.ID == paths[5] {
	   								ctx.BaseURLPush(paths[5])
	   								if len(paths) == 6 {
	   									if ctx.Flags.Self {
	   										ver.ToJSON(ctx)
	   										return ctx.Result(), nil
	   									}
	   									data, ok := ver.Data["resourceContent"]
	   									if ok {
	   										str, _ := data.(string)
	   										return str, nil
	   									}

	   									uri, ok := ver.Data["resourceProxyURI"]
	   									if ok {
	   										resp, err := http.Get(uri.(string))
	   										if err != nil {
	   											return "", err
	   										}
	   										if resp.StatusCode/100 != 2 {
	   											return "", fmt.Errorf("%s ->%d",
	   												uri, resp.StatusCode)
	   										}
	   										body, err := io.ReadAll(resp.Body)
	   										if err != nil {
	   											return "", err
	   										}
	   										return string(body), nil
	   									}
	   								}
	   							}
	   						}
	   					}
	   				}
	   			}
	   		}
	   	}

	   return "", fmt.Errorf("Can't figure out what to do with %q",

	   	strings.Join(paths, "/"))
	*/
}
