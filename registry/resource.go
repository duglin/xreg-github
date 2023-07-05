package registry

import (
	"fmt"
	"io"

	log "github.com/duglin/dlog"
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
		Group:              rc.Group,
		ID:                 id,
		LatestId:           "",
		VersionCollection:  &VersionCollection{},
	}

	res.VersionCollection.Resource = res

	rc.Resources[id] = res
	return res
}

type Resource struct {
	Entity
	ResourceCollection *ResourceCollection
	Group              *Group

	ID        string
	LatestId  string
	LatestUrl string
	Self      string

	VersionCollection *VersionCollection // map[string]*Version // version
}

func (r *Resource) ToJSON(w io.Writer, jd *JSONData) (bool, error) {
	ver := r.GetLatest()

	fmt.Fprintf(w, "%s{\n", jd.Prefix)
	fmt.Fprintf(w, "%s  \"id\": %q,\n", jd.Indent, r.ID)
	fmt.Fprintf(w, "%s  \"name\": %q,\n", jd.Indent, ver.Name)
	fmt.Fprintf(w, "%s  \"epoch\": %d,\n", jd.Indent, ver.Epoch)
	fmt.Fprintf(w, "%s  \"self\": %q,\n", jd.Indent, "...")
	fmt.Fprintf(w, "%s  \"latestId\": %q,\n", jd.Indent, r.LatestId)
	fmt.Fprintf(w, "%s  \"latestUrl\": %q", jd.Indent, r.LatestUrl)

	// TODO add Resource stuff

	vCount := 0
	results, err := NewQuery(`
			SELECT VersionID from Versions
			WHERE ResourceID=?`, r.DbID)
	if err != nil {
		return false, err
	}

	for _, row := range results {
		v := r.FindVersion(NotNilString(row[0])) // v.ID
		if v == nil {
			log.Printf("Can't find version %s", NotNilString(row[0]))
			continue // Should never happen
		}

		prefix := fmt.Sprintf(",\n")
		if vCount == 0 {
			prefix += fmt.Sprintf("\n%s  \"versions\": {\n", jd.Indent)
		}
		prefix += fmt.Sprintf("%s    %q: ", jd.Indent, v.ID)
		shown, err := v.ToJSON(w,
			&JSONData{prefix, jd.Indent + "    ", jd.Registry})
		if err != nil {
			return false, err
		}
		if shown {
			vCount++
		}
	}
	if vCount > 0 {
		fmt.Fprintf(w, "\n%s  },\n", jd.Indent)
	} else {
		fmt.Fprintf(w, ",\n\n")
	}

	fmt.Fprintf(w, "%s  \"versionsCount\": %d,\n", jd.Indent, vCount)
	fmt.Fprintf(w, "%s  \"versionsUrl\": \"%s/versions\"", jd.Indent, "...")

	fmt.Fprintf(w, "\n%s}", jd.Indent)
	return true, nil
}

func (r *Resource) Set(name string, val any) error {
	if name[0] == '.' {
		return SetProp(r, name[1:], val)
	}

	if name == "ID" || name == "LatestId" || name == "LatestUrl" ||
		name == "Self" {
		// return SetProp(r, name, val)
		return SetProp(r, name, val)
	}

	// return SetProp(r.GetLatest(), name, val)
	v := r.GetLatest()
	return SetProp(v, name, val)
}

func (r *Resource) Refresh() error {
	log.VPrintf(3, ">Enter: resource.Refresh(%s)", r.ID)
	defer log.VPrintf(3, "<Exit: resource.Refresh")

	result, err := Query(`
	        SELECT PropName, PropValue, PropType
	        FROM Props WHERE EntityID=? `,
		r.DbID)
	defer result.Close()

	if err != nil {
		log.Printf("Error refreshing Resource(%s): %s", r.ID, err)
		return fmt.Errorf("Error refreshing Resource(%s): %s", r.ID, err)
	}

	*r = Resource{ // Erase all existing properties
		Entity: Entity{
			RegistryID: r.RegistryID,
			DbID:       r.DbID,
		},
		ID: r.ID,
	}

	for result.NextRow() {
		name := NotNilString(result.Data[0])
		val := NotNilString(result.Data[1])
		propType := NotNilString(result.Data[2])
		SetField(r, name, &val, propType)
	}

	return nil
}

func (r *Resource) FindVersion(id string) *Version {
	log.VPrintf(3, ">Enter: FindVersion(%s)", id)
	defer log.VPrintf(3, "<Exit: FindVersion")

	results, _ := NewQuery(`
		SELECT v.ID, p.PropName, p.PropValue, p.PropType
		FROM Versions as v LEFT JOIN Props AS p ON (p.EntityID=v.ID)
		WHERE v.VersionID=? AND v.ResourceID=?`, id, r.DbID)

	v := (*Version)(nil)
	for _, row := range results {
		if v == nil {
			v = &Version{
				Entity: Entity{
					RegistryID: r.RegistryID,
					DbID:       NotNilString(row[0]),
				},
				Resource: r,
				ID:       id,
			}
			log.VPrintf(3, "Found one: %s", v.DbID)
		}
		if *row[1] != nil { // We have Props
			name := NotNilString(row[1])
			val := NotNilString(row[2])
			propType := NotNilString(row[3])
			SetField(v, name, &val, propType)
		}
	}

	if v == nil {
		log.VPrintf(3, "None found")
	}

	return v
}

func (r *Resource) OldFindVersion(verString string) *Version {
	return r.VersionCollection.Versions[verString]
}

func (r *Resource) GetLatest() *Version {
	if r.LatestId == "" {
		panic("Latest ID is missing")
	}

	return r.FindVersion(r.LatestId)
}

func (r *Resource) FindOrAddVersion(id string) *Version {
	log.VPrintf(3, ">Enter: FindOrAddVersion%s)", id)
	defer log.VPrintf(3, "<Exit: FindOrAddVersion")

	v := r.FindVersion(id)
	if v != nil {
		log.VPrintf(3, "Found one")
		return v
	}

	v = &Version{
		Entity: Entity{
			RegistryID: r.RegistryID,
			DbID:       NewUUID(),
		},
		Resource: r,
		ID:       id,
	}

	err := DoOne(`
		INSERT INTO Versions(ID, VersionID, ResourceID, Path, Abstract)
		VALUES(?,?,?,?,?)`,
		v.DbID, id, r.DbID,
		r.Group.Plural+"/"+r.Group.DbID+"/"+r.Plural+"/"+r.DbID+"/versions/"+id,
		r.Group.Plural+"/"+r.Plural+"/versions")
	if err != nil {
		log.Printf("Error adding version: %s", err)
		return nil
	}

	v.Set("id", id)

	if r.LatestId == "" {
		r.Set("LatestId", id)
	}

	log.VPrintf(3, "Created new one - dbID: %s", v.DbID)
	return v
}

func (r *Resource) OldFindOrAddVersion(verStr string) *Version {
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
