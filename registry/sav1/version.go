package registry

import (
// "log"
)

type VersionCollection struct {
	Resource *Resource
	Versions map[string]*Version // version
}

func (vc *VersionCollection) ToObject(ctx *Context) (*Object, error) {
	obj := NewObject()
	if vc == nil {
		return obj, nil
	}

	for _, key := range SortedKeys(vc.Versions) {
		ver := vc.Versions[key]

		match, _, err := ctx.Filter(ver)
		// log.Printf("versions(%s) match: %d", ver.ID, match)
		if err != nil {
			return nil, err
		}
		if match == -1 {
			continue
		}
		/*
			match := ctx.MatchFilters(vc.Resource.ResourceCollection.Group.GroupCollection.GroupModel.Singular, vc.Resource.ResourceCollection.ResourceModel.Singular, "version", ver)
			if match == -1 {
				continue
			}
		*/

		ctx.DataPush(ver.ID)
		ctx.MatchPush(match)
		verObj, err := ver.ToObject(ctx)
		ctx.MatchPop()
		ctx.DataPop()
		if err != nil {
			return nil, err
		}
		obj.AddProperty(ver.ID, verObj)
	}

	return obj, nil
}

func (vc *VersionCollection) ToJSON(ctx *Context) {
	ctx.Print("{\n")
	ctx.Indent()

	for vCount, key := range SortedKeys(vc.Versions) {
		ver := vc.Versions[key]
		if vCount > 0 {
			ctx.Print(",\n")
		}
		ctx.DataPush(ver.ID)
		ctx.Printf("\t\"%s\": ", ver.ID)
		ver.ToJSON(ctx)
		ctx.DataPop()
	}

	ctx.Print("\n")
	ctx.Outdent()
	ctx.Print("\t}")
}

type Version struct {
	Resource *Resource

	ID      string
	Name    string
	Type    string
	Version string
	Epoch   int

	Data map[string]interface{}
}

func (v *Version) ToObject(ctx *Context) (*Object, error) {
	obj := NewObject()
	if v == nil {
		return obj, nil
	}

	obj.AddProperty("id", v.ID)
	v.AddToJsonInner(ctx, obj)

	myURI := ctx.DataURL()
	mySelf := myURI
	if mySelf[0] != '#' {
		mySelf += "?self"
	}

	contentURI := v.Data["resourceURI"]
	if contentURI == nil {
		contentURI = myURI
	}

	obj.AddProperty(v.Resource.ResourceCollection.ResourceModel.Singular+"URI",
		contentURI)
	obj.AddProperty("self", mySelf)

	return obj, nil
}

func (v *Version) ToJSON(ctx *Context) {
	ctx.Print("{\n")
	ctx.Indent()

	ctx.Printf("\t\"id\": \"%s\",\n", v.ID)
	v.ToJSONInner(ctx)

	myURI := ctx.DataURL()
	mySelf := myURI
	if mySelf[0] != '#' {
		mySelf += "?self"
	}

	contentURI := v.Data["resourceURI"]
	if contentURI == nil {
		contentURI = myURI
	}

	ctx.Printf("\t\"%sURI\": \"%s\",\n",
		v.Resource.ResourceCollection.ResourceModel.Singular,
		contentURI)
	ctx.Printf("\t\"self\": \"%s\"\n", mySelf)

	ctx.Outdent()
	ctx.Print("\t}")
}

func (v *Version) AddToJsonInner(ctx *Context, obj *Object) {
	obj.AddProperty("name", v.Name)
	obj.AddProperty("type", v.Type)
	obj.AddProperty("version", v.Version)
	obj.AddProperty("epoch", v.Epoch)
}

func (v *Version) ToJSONInner(ctx *Context) {
	ctx.Printf("\t\"name\": \"%s\",\n", v.Name)
	ctx.Printf("\t\"type\": \"%s\",\n", v.Type)
	ctx.Printf("\t\"version\": \"%s\",\n", v.Version)
	ctx.Printf("\t\"epoch\": %d,\n", v.Epoch)
}
