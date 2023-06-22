package registry

import (
	"fmt"
	"io"
	// "log"
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
	OrFilters   [][]string // [OLD][AND]string

	Filters []string // OLD
}

type Registry struct {
	BaseURL      string
	Model        *Model `json:"-"`
	GenericModel *ModelElement

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

func (reg *Registry) ToObject(ctx *Context) (*Object, error) {
	obj := NewObject()
	if reg == nil {
		return obj, nil
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

	if ctx.ShouldInline("model") {
		obj.AddProperty("", "")
		ctx.ModelPush("model")
		mod, err := reg.Model.ToObject(ctx)
		if err != nil {
			return nil, err
		}
		obj.AddProperty("model", mod)
		ctx.ModelPop()
	}

	for _, key := range SortedKeys(reg.Model.Groups) {
		gType := reg.Model.Groups[key]
		gCollection := reg.GroupCollections[gType.Plural]

		var err error

		ctx.DataPush(gType.Plural)
		ctx.ModelPush(gType.Plural)
		ctx.FilterPush(gType.Plural)
		groupObj := NewObject()
		if gCollection != nil {
			groupObj, err = gCollection.ToObject(ctx)
		}
		ctx.FilterPop()
		ctx.ModelPop()
		ctx.DataPop()
		if err != nil {
			return nil, err
		}

		obj.AddProperty(gType.Plural, &Collection{
			Name:   gType.Plural,
			URL:    URLBuild(ctx.DataURL(), gType.Plural),
			Inline: ctx.ShouldInline(gType.Plural),
			Object: groupObj,
		})
	}

	return obj, nil
}

func (r *Registry) Get(path string, rFlags *RegistryFlags) (string, error) {
	if r.GenericModel == nil {
		r.GenericModel = CreateGenericModel(r.Model)
	}

	paths := strings.Split(strings.Trim(path, "/"), "/")
	for len(paths) > 0 && paths[0] == "" {
		paths = paths[1:]
	}

	if rFlags == nil {
		rFlags = &RegistryFlags{}
	}

	filters, err := ParseFilterExprs(r, paths, rFlags.Filters)
	if err != nil {
		return "", err
	}

	ctx := &Context{
		Flags:   rFlags,
		BaseURL: r.BaseURL,
		Filters: filters,
	}

	if rFlags.BaseURL != "" {
		ctx.BaseURL = rFlags.BaseURL
	}
	ctx.BaseURL = strings.TrimRight(ctx.BaseURL, "/")

	if len(paths) == 0 {
		obj, err := r.ToObject(ctx)
		if err != nil {
			return "", err
		}
		obj.ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	if len(paths) == 1 && paths[0] == "model" {
		obj, err := r.Model.ToObject(ctx)
		if err != nil {
			return "", err
		}
		obj.ToJson(&ctx.buffer, "", "  ")
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
		groupCollObj, err := groupColl.ToObject(ctx)
		if err != nil {
			return "", err
		}
		if groupCollObj == nil {
			return "{}", nil
		}
		groupCollObj.ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	// GROUPs/ID
	group := groupColl.Groups[paths[1]]
	if group == nil {
		return "", fmt.Errorf("Unknown group ID %q", paths[1])
	}
	ctx.BaseURLPush(paths[1])
	if len(paths) == 2 {
		groupObj, err := group.ToObject(ctx)
		if err != nil {
			return "", err
		}
		if groupObj == nil {
			return "{}", nil
		}
		groupObj.ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	// GROUPs/ID/RESOURCEs
	resColl := group.ResourceCollections[paths[2]]
	ctx.BaseURLPush(paths[2])
	if resColl == nil {
		return "", fmt.Errorf("Unknown rescource collection %q", paths[2])
	}
	if len(paths) == 3 {
		resCollObj, err := resColl.ToObject(ctx)
		if err != nil {
			return "", err
		}
		if resCollObj == nil {
			return "{}", nil
		}
		resCollObj.ToJson(&ctx.buffer, "", "  ")
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
			resObj, err := res.ToObject(ctx)
			if err != nil {
				return "", err
			}
			if resObj == nil {
				return "{}", nil
			}
			resObj.ToJson(&ctx.buffer, "", "  ")
			return ctx.buffer.String(), nil
		}
		latest := res.FindVersion(res.LatestId)
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
		verCollObj, err := verColl.ToObject(ctx)
		if err != nil {
			return "", err
		}
		if verCollObj == nil {
			return "{}", nil
		}
		verCollObj.ToJson(&ctx.buffer, "", "  ")
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
			verObj, err := ver.ToObject(ctx)
			if err != nil {
				return "", err
			}
			if verObj == nil {
				return "{}", nil
			}
			verObj.ToJson(&ctx.buffer, "", "  ")
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
}
