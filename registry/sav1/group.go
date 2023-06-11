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
		GroupCollection:     gc,
		ID:                  id,
		Name:                id,
		Epoch:               0,
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

		/*
			match := ctx.MatchFilters(gc.GroupModel.Singular, "", "", group)
			// log.Printf("  group match: %d\n", match)
			if match == -1 {
				continue
			}
		*/
		match, _, err := ctx.Filter(group)
		// log.Printf("%s(%s) match: %d", gc.GroupModel.Plural, group.ID, match)
		if err != nil {
			return nil, err
		}
		if match == -1 {
			continue
		}

		ctx.DataPush(group.ID)
		ctx.MatchPush(match)
		groupObj, err := group.ToObject(ctx)
		ctx.MatchPop()
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

func (gc *GroupCollection) ToJSON(ctx *Context) {
	ctx.Print("{\n")
	ctx.Indent()

	for gCount, key := range SortedKeys(gc.Groups) {
		group := gc.Groups[key]
		if gCount > 0 {
			ctx.Print(",\n")
		}
		ctx.DataPush(group.ID)
		ctx.Printf("\t\"%s\": ", group.ID)
		group.ToJSON(ctx)
		ctx.DataPop()
	}

	ctx.Print("\n")
	ctx.Outdent()
	ctx.Print("\t}")
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

	ID                  string
	Name                string
	Epoch               int
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
	// childFilter := false

	for i, key := range SortedKeys(g.GroupCollection.GroupModel.Resources) {
		rType := g.GroupCollection.GroupModel.Resources[key]
		rColl := g.ResourceCollections[rType.Plural]

		obj.AddProperty(rColl.ResourceModel.Plural+"URL",
			URLBuild(ctx.DataURL(), rColl.ResourceModel.Plural))

		/*
			// Only continue with resourceCollections that match the filters
			// -1 no match, 0 dunno, 1 for sure
			match, cf, err := ctx.Filter(rColl)
			// childFilter = childFilter || cf
			if err != nil {
				return nil, err
			}
			if match == -1 {
				continue
			}
		*/

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

	// new
	hasChildren := ctx.HasChildrenFilters()
	if hasChildren && rCount == 0 {
		return nil, nil
	}
	/*
		if childFilter && rCount == 0 {
			return nil, nil
		}
	*/

	/*
		if len(ctx.Filters) != 0 && rCount == 0 {
			return nil, nil
		}
	*/

	return obj, nil
}

func (g *Group) ToJSON(ctx *Context) {
	ctx.Print("{\n")
	ctx.Indent()

	ctx.Printf("\t\"id\": \"%s\",\n", g.ID)
	ctx.Printf("\t\"name\": \"%s\",\n", g.Name)
	ctx.Printf("\t\"epoch\": %d,\n", g.Epoch)
	ctx.Printf("\t\"self\": \"%s\",\n", ctx.DataURL())

	for rCount, key := range SortedKeys(g.GroupCollection.GroupModel.Resources) {
		rType := g.GroupCollection.GroupModel.Resources[key]
		rColl := g.ResourceCollections[rType.Plural]

		if rCount > 0 {
			ctx.Print(",\n")
		}
		ctx.Printf("\t\"%sURL\": \"%s\",\n",
			rColl.ResourceModel.Plural,
			URLBuild(ctx.DataURL(), rColl.ResourceModel.Plural))
		ctx.Printf("\t\"%sCount\": %d",
			rColl.ResourceModel.Plural,
			len(rColl.Resources))

		if ctx.ShouldInline(rColl.ResourceModel.Plural) && len(rColl.Resources) > 0 {
			ctx.ModelPush(rColl.ResourceModel.Plural)
			ctx.DataPush(rColl.ResourceModel.Plural)
			ctx.Print(",\n")
			ctx.Printf("\t\"%s\": ", rColl.ResourceModel.Plural)
			rColl.ToJSON(ctx)
			ctx.DataPop()
			ctx.ModelPop()
		}
	}

	ctx.Print("\n")
	ctx.Outdent()
	ctx.Print("\t}")
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
