package registry

import (
	"fmt"
	"io"

	log "github.com/duglin/dlog"
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
	Entity
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

	ResourceURL      string // Send a redirect back to client
	ResourceProxyURL string // The URL to the data, but GET and return data
	ResourceContent  []byte // The raw data

	Data map[string]interface{}
}

func (v *Version) ToJSON(w io.Writer, jd *JSONData) (bool, error) {
	fmt.Fprintf(w, "%s{\n", jd.Prefix)
	fmt.Fprintf(w, "%s  \"id\": %q,\n", jd.Indent, v.ID)
	fmt.Fprintf(w, "%s  \"name\": %q,\n", jd.Indent, v.Name)
	fmt.Fprintf(w, "%s  \"epoch\": %d,\n", jd.Indent, v.Epoch)
	fmt.Fprintf(w, "%s  \"self\": %q", jd.Indent, "...")
	fmt.Fprintf(w, "\n%s}", jd.Indent)
	return true, nil
}

func (v *Version) Set(name string, val any) error {
	return SetProp(v, name, val)
}

func (v *Version) Refresh() error {
	log.VPrintf(3, ">Enter: version.Refresh(%s)", v.ID)
	defer log.VPrintf(3, "<Exit: version.Refresh")

	result, err := Query(`
		SELECT PropName, PropValue, PropType
		FROM Props WHERE EntityID=? `,
		v.DbID)
	defer result.Close()

	if err != nil {
		log.Printf("Error refreshing version(%s): %s", v.ID, err)
		return fmt.Errorf("Error refreshing version(%s): %s", v.ID, err)
	}

	*v = Version{ // Erase all existing properties
		Entity: Entity{
			RegistryID: v.RegistryID,
			DbID:       v.DbID,
		},
		ID: v.ID,
	}

	for result.NextRow() {
		name := NotNilString(result.Data[0])
		val := NotNilString(result.Data[1])
		propType := NotNilString(result.Data[2])
		SetField(v, name, &val, propType)
	}

	return nil
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

	contentURI := v.ResourceURL
	if contentURI == "" {
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
