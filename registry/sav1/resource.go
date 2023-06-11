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

		match, _, err := ctx.Filter(&latest)
		// log.Printf("%s(%s) match: %d", rc.ResourceModel.Plural, latest.ID, match)
		if err != nil {
			return nil, err
		}
		if match == -1 {
			continue
		}
		/*
			match := ctx.MatchFilters(rc.Group.GroupCollection.GroupModel.Singular, rc.ResourceModel.Singular, "", &latest)
			// log.Printf("res.ID(%s) match: %v", resource.ID, match)
			if match == -1 {
				continue
			}
		*/

		ctx.DataPush(resource.ID)
		ctx.MatchPush(match)
		resObj, err := resource.ToObject(ctx)
		ctx.MatchPop()
		ctx.DataPop()

		if err != nil {
			return nil, err
		}

		// New
		if resObj == nil {
			continue
		}

		if match != 1 && resObj == nil {
			continue
		}

		// log.Printf("Adding %s.%s.%s", rc.Group.GroupCollection.GroupModel.Singular, rc.ResourceModel.Singular, resource.ID)
		obj.AddProperty(resource.ID, resObj)
	}

	return obj, nil
}

func (rc *ResourceCollection) ToJSON(ctx *Context) {
	ctx.Print("{\n")
	ctx.Indent()

	for rCount, key := range SortedKeys(rc.Resources) {
		resource := rc.Resources[key]
		if rCount > 0 {
			ctx.Print(",\n")
		}
		ctx.DataPush(resource.ID)
		ctx.Printf("\t\"%s\": ", resource.ID)
		resource.ToJSON(ctx)
		ctx.DataPop()
	}

	ctx.Print("\n")
	ctx.Outdent()
	ctx.Print("\t}")
}

func (rc *ResourceCollection) NewResource(id string) *Resource {
	res := &Resource{
		ResourceCollection: rc,
		ID:                 id,
		Latest:             "",
		VersionCollection:  &VersionCollection{},
	}

	res.VersionCollection.Resource = res

	rc.Resources[id] = res
	return res
}

type Resource struct {
	ResourceCollection *ResourceCollection

	ID                string
	Latest            string
	VersionCollection *VersionCollection // map[string]*Version // version
}

func (r *Resource) FindVersion(verString string) *Version {
	return r.VersionCollection.Versions[verString]
}

func (r *Resource) GetLatest() *Version {
	var latest *Version
	if r.Latest != "" {
		latest = r.VersionCollection.Versions[r.Latest]
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
		Version:  verStr,
		Data:     map[string]interface{}{},
	}

	if r.VersionCollection.Versions == nil {
		r.VersionCollection.Versions = map[string]*Version{}
	}

	r.VersionCollection.Versions[verStr] = ver

	if r.Latest == "" {
		r.Latest = verStr
	}
	return ver
}

func (r *Resource) ToObject(ctx *Context) (*Object, error) {
	obj := NewObject()
	if r == nil {
		return obj, nil
	}

	var latest *Version
	if r.Latest != "" {
		latest = r.VersionCollection.Versions[r.Latest]
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

	// new
	match, _, err := ctx.Filter(r.VersionCollection)
	// log.Printf("%s match: %d", "versions", match)
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

	// new
	if vers == nil {
		return nil, nil
	}

	hasChildren := ctx.HasChildrenFilters()
	if hasChildren && vers.Len() == 0 {
		return nil, nil
	}

	/*
		if len(ctx.Filters) != 0 && vers.Len() == 0 {
			return nil, nil
		}
	*/

	if ctx.ShouldInline("versions") && vers.Len() > 0 {
		obj.AddProperty("versions", vers)
	}

	return obj, nil
}

func (r *Resource) ToJSON(ctx *Context) {
	var latest *Version
	if r.Latest != "" {
		latest = r.VersionCollection.Versions[r.Latest]
	}
	if latest == nil {
		panic("Help")
		for _, latest = range r.VersionCollection.Versions {
			break
		}
	}

	ctx.Print("{\n")
	ctx.Indent()

	ctx.Printf("\t\"id\": \"%s\",\n", r.ID)
	latest.ToJSONInner(ctx)

	myURI := URLBuild(ctx.DataURL())
	mySelf := myURI
	if mySelf[0] != '#' {
		mySelf += "?self"
	}

	contentURI := latest.Data["resourceURI"]
	if contentURI == nil {
		contentURI = myURI
	}

	ctx.Printf("\t\"%sURI\": \"%s\",\n",
		r.ResourceCollection.ResourceModel.Singular,
		contentURI)
	ctx.Printf("\t\"self\": \"%s\"", mySelf)

	if ctx.ShouldInline("versions") && len(r.VersionCollection.Versions) > 0 {
		ctx.Print(",\n")
		ctx.ModelPush("versions")
		ctx.DataPush("versions")
		ctx.Print("\t\"versions\": ")
		r.VersionCollection.ToJSON(ctx)
		ctx.DataPop()
		ctx.ModelPop()
	}

	ctx.Print("\n")
	ctx.Outdent()
	ctx.Print("\t}")
}
