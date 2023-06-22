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
		verObj, err := ver.ToObject(ctx)
		ctx.DataPop()
		if err != nil {
			return nil, err
		}
		obj.AddProperty(ver.ID, verObj)
	}

	return obj, nil
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

func (v *Version) AddToJsonInner(ctx *Context, obj *Object) {
	obj.AddProperty("name", v.Name)
	obj.AddProperty("epoch", v.Epoch)
}
