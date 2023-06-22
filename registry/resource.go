package registry

import (
	"fmt"
	// "log"
)

type ResourceCollection struct {
	Group         *Group
	ResourceModel *ResourceModel
	Resources     map[string]*Resource // id
}

func (rc *ResourceCollection) ToObject(ctx *Context) (*Object, error) {
	obj := NewObject()
	if rc == nil {
		return obj, nil
	}

	for _, key := range SortedKeys(rc.Resources) {
		resource := rc.Resources[key]
		latest := *resource.GetLatest()
		latest.ID = resource.ID

		match, err := ctx.Filter(&latest)
		if err != nil {
			return nil, err
		}
		if match == -1 {
			continue
		}

		ctx.DataPush(resource.ID)
		resObj, err := resource.ToObject(ctx)
		ctx.DataPop()

		if err != nil {
			return nil, err
		}

		if resObj == nil {
			continue
		}

		obj.AddProperty(resource.ID, resObj)
	}

	return obj, nil
}

func (rc *ResourceCollection) NewResource(id string) *Resource {
	res := &Resource{
		ResourceCollection: rc,
		ID:                 id,
		LatestId:           "",
		VersionCollection:  &VersionCollection{},
	}

	res.VersionCollection.Resource = res

	rc.Resources[id] = res
	return res
}

type Resource struct {
	ResourceCollection *ResourceCollection

	ID          string
	Name        string
	Epoch       int
	Self        string
	LatestId    string
	LatestUrl   string
	Description string
	Docs        string
	Tags        map[string]string
	Format      string
	CreatedBy   string
	CreatedOn   string
	ModifiedBy  string
	ModifiedOn  string

	VersionCollection *VersionCollection // map[string]*Version // version
}

func (r *Resource) FindVersion(verString string) *Version {
	return r.VersionCollection.Versions[verString]
}

func (r *Resource) GetLatest() *Version {
	var latest *Version
	if r.LatestId != "" {
		latest = r.VersionCollection.Versions[r.LatestId]
	}
	if latest == nil {
		panic(fmt.Sprintf("Help cant determine latest for %#v", r))
	}
	return latest
}

func (r *Resource) FindOrAddVersion(verStr string) *Version {
	ver := r.VersionCollection.Versions[verStr]
	if ver != nil {
		return ver
	}
	ver = &Version{
		Resource: r,
		ID:       verStr,
		Data:     map[string]interface{}{},
	}

	if r.VersionCollection.Versions == nil {
		r.VersionCollection.Versions = map[string]*Version{}
	}

	r.VersionCollection.Versions[verStr] = ver

	if r.LatestId == "" {
		r.LatestId = verStr
	}
	return ver
}

func (r *Resource) ToObject(ctx *Context) (*Object, error) {
	obj := NewObject()
	if r == nil {
		return obj, nil
	}

	var latest *Version
	if r.LatestId != "" {
		latest = r.VersionCollection.Versions[r.LatestId]
	}
	if latest == nil {
		panic("Help")
		for _, latest = range r.VersionCollection.Versions {
			break
		}
	}

	obj.AddProperty("id", r.ID)
	latest.AddToJsonInner(ctx, obj)

	myURI := URLBuild(ctx.DataURL())
	mySelf := myURI
	if mySelf[0] != '#' {
		mySelf += "?self"
	}

	contentURI := latest.Data["resourceURI"]
	if contentURI == nil {
		contentURI = myURI
	}

	obj.AddProperty(r.ResourceCollection.ResourceModel.Singular+"URI",
		contentURI)
	obj.AddProperty("self", mySelf)

	match, err := ctx.Filter(r.VersionCollection)
	if err != nil {
		return nil, err
	}
	if match == -1 {
		return nil, nil
	}

	ctx.ModelPush("versions")
	ctx.DataPush("versions")
	ctx.FilterPush("versions")
	vers, err := r.VersionCollection.ToObject(ctx)
	ctx.FilterPop()
	ctx.DataPop()
	ctx.ModelPop()
	if err != nil {
		return nil, err
	}

	if vers == nil {
		return nil, nil
	}

	if ctx.HasChildrenFilters() && vers.Len() == 0 {
		return nil, nil
	}

	if ctx.ShouldInline("versions") && vers.Len() > 0 {
		obj.AddProperty("versions", vers)
	}

	return obj, nil
}
