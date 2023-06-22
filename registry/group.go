package registry

import (
	"fmt"
	// "log"
	"strings"
)

type GroupCollection struct {
	Registry   *Registry
	GroupModel *GroupModel
	Groups     map[string]*Group // id->*Group
}

func (gc *GroupCollection) NewGroup(id string) *Group {
	group := &Group{
		GroupCollection: gc,

		ID:    id,
		Name:  id,
		Epoch: 0,

		ResourceCollections: map[string]*ResourceCollection{}, // id
	}
	gc.Groups[id] = group
	return group
}

func (gc *GroupCollection) FindByID(id string) {
}

func (gc *GroupCollection) ToObject(ctx *Context) (*Object, error) {
	obj := NewObject()
	if gc == nil {
		return obj, nil
	}
	for _, key := range SortedKeys(gc.Groups) {
		group := gc.Groups[key]

		match, err := ctx.Filter(group)
		if err != nil {
			return nil, err
		}
		if match == -1 {
			continue
		}

		ctx.DataPush(group.ID)
		groupObj, err := group.ToObject(ctx)
		ctx.DataPop()
		if err != nil {
			return nil, err
		}

		// new
		if groupObj == nil {
			continue
		}

		if match != 1 && groupObj == nil {
			continue
		}
		// 	if groupObj != nil {
		obj.AddProperty(group.ID, groupObj)
		// }
	}

	return obj, nil
}

func (gc *GroupCollection) FindGroup(gID string) *Group {
	for _, group := range gc.Groups {
		if strings.EqualFold(group.ID, gID) {
			return group
		}
	}
	return nil
}

type Group struct {
	GroupCollection *GroupCollection

	ID          string
	Name        string
	Epoch       int
	Self        string
	Description string
	Docs        string
	Tags        map[string]string
	Format      string
	CreatedBy   string
	CreatedOn   string
	ModifiedBy  string
	ModifiedOn  string

	ResourceCollections map[string]*ResourceCollection
}

func (g *Group) ToObject(ctx *Context) (*Object, error) {
	obj := NewObject()
	if g == nil {
		return obj, nil
	}
	obj.AddProperty("id", g.ID)
	obj.AddProperty("name", g.Name)
	obj.AddProperty("epoch", g.Epoch)
	obj.AddProperty("self", ctx.DataURL())

	rCount := 0

	for i, key := range SortedKeys(g.GroupCollection.GroupModel.Resources) {
		rType := g.GroupCollection.GroupModel.Resources[key]
		rColl := g.ResourceCollections[rType.Plural]

		obj.AddProperty(rColl.ResourceModel.Plural+"Url",
			URLBuild(ctx.DataURL(), rColl.ResourceModel.Plural))

		ctx.ModelPush(rColl.ResourceModel.Plural)
		ctx.DataPush(rColl.ResourceModel.Plural)
		ctx.FilterPush(rColl.ResourceModel.Plural)
		resObj, err := rColl.ToObject(ctx)
		ctx.FilterPop()
		ctx.DataPop()
		ctx.ModelPop()
		if err != nil {
			return nil, err
		}

		obj.AddProperty(rColl.ResourceModel.Plural+"Count", resObj.Len())
		rCount += resObj.Len()

		if ctx.ShouldInline(rColl.ResourceModel.Plural) {
			obj.AddProperty(rColl.ResourceModel.Plural, resObj)
			if i+1 != len(g.GroupCollection.GroupModel.Resources) {
				obj.AddProperty("", "")
			}
		}
	}

	if ctx.HasChildrenFilters() && rCount == 0 {
		return nil, nil
	}

	return obj, nil
}

func (g *Group) FindOrAddResourceCollection(rType string) *ResourceCollection {
	rc, _ := g.ResourceCollections[rType]
	if rc == nil {
		rm := g.GroupCollection.GroupModel.Resources[rType]
		if rm == nil {
			panic(fmt.Sprintf("Can't find ResourceModel %q", rType))
		}

		rc = &ResourceCollection{
			Group:         g,
			ResourceModel: rm,
			Resources:     map[string]*Resource{},
		}
		g.ResourceCollections[rType] = rc
	}
	return rc
}

func (g *Group) FindOrAddResource(rType string, id string) *Resource {
	rc := g.FindOrAddResourceCollection(rType)
	res := rc.Resources[id]
	if res != nil {
		return res
	}
	return rc.NewResource(id)
}
