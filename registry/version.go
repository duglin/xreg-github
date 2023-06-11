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

		match, err := ctx.Filter(ver)
		// log.Printf("versions(%s) match: %d", ver.ID, match)
		if err != nil {
			return nil, err
		}
		if match == -1 {
			continue
		}

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
	obj.AddProperty("epoch", v.Epoch)
}

func (v *Version) ToJSONInner(ctx *Context) {
	ctx.Printf("\t\"name\": \"%s\",\n", v.Name)
	ctx.Printf("\t\"epoch\": %d,\n", v.Epoch)
	// ctx.Printf("\t\"self\": %s,\n", v.Self)
	ctx.Printf("\t\"description\": %s,\n", v.Description)
	ctx.Printf("\t\"docs\": %s,\n", v.Docs)
	ctx.Printf("\t\"tags\": %s,\n", "{}")
	ctx.Printf("\t\"createdBy\": %s,\n", v.CreatedBy)
	ctx.Printf("\t\"createdOn\": %s,\n", v.CreatedOn)
	ctx.Printf("\t\"modifiedBy\": %s,\n", v.ModifiedBy)
	ctx.Printf("\t\"modifiedOn\": %s,\n", v.ModifiedOn)
}
