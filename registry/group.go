package registry

import (
	"fmt"
	"io"
	"strings"

	log "github.com/duglin/dlog"
)

type GroupCollection struct {
	Registry   *Registry
	GroupModel *GroupModel
	Groups     map[string]*Group // id->*Group
}

func (gc *GroupCollection) NewGroup(id string) *Group {
	group := &Group{
		GroupCollection: gc,
		Entity: Entity{
			Plural: gc.GroupModel.Plural,
		},

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
	Entity
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

func (g *Group) ToJSON(w io.Writer, jd *JSONData) (bool, error) {
	// Prefix is the first line indent
	// Indent is the indent for all subsequent lines

	fmt.Fprintf(w, "%s{\n", jd.Prefix)
	fmt.Fprintf(w, "%s  \"id\": %q,\n", jd.Indent, g.ID)
	fmt.Fprintf(w, "%s  \"name\": %q,\n", jd.Indent, g.Name)
	fmt.Fprintf(w, "%s  \"epoch\": %d,\n", jd.Indent, g.Epoch)
	fmt.Fprintf(w, "%s  \"self\": %q", jd.Indent, "...")

	for _, rModel := range jd.Registry.Model.Groups[g.Plural].Resources {
		results, err := NewQuery(`
			SELECT ResourceID FROM Resources
			WHERE GroupID=? AND ModelID=?`, g.DbID, rModel.ID)
		if err != nil {
			return false, err
		}

		rCount := 0
		for _, row := range results {
			r := g.FindResource(rModel.Plural, NotNilString(row[0])) // r.ID
			if r == nil {
				log.Printf("Can't find resource %s/%s", rModel.Plural,
					NotNilString(row[0]))
				continue // Should never happen
			}

			prefix := fmt.Sprintf(",\n")
			if rCount == 0 {
				prefix += fmt.Sprintf("\n%s  %q: {\n", jd.Indent, rModel.Plural)
			}
			prefix += fmt.Sprintf("%s    %q: ", jd.Indent, r.ID)
			shown, err := r.ToJSON(w,
				&JSONData{prefix, jd.Indent + "    ", jd.Registry})
			if err != nil {
				return false, err
			}
			if shown {
				rCount++
			}
		}
		if rCount > 0 {
			fmt.Fprintf(w, "\n%s  },\n", jd.Indent)
		} else {
			fmt.Fprintf(w, ",\n\n")
		}

		fmt.Fprintf(w, "%s  \"%sCount\": %d,\n", jd.Indent, rModel.Plural,
			rCount)
		fmt.Fprintf(w, "%s  \"%sURL\": \"%s/%s\"", jd.Indent, rModel.Plural,
			"...", rModel.Plural)

	}

	fmt.Fprintf(w, "\n%s}", jd.Indent)

	return true, nil
}

func (g *Group) Refresh() error {
	log.VPrintf(3, ">Enter: group.Refresh(%s)", g.ID)
	defer log.VPrintf(3, "<Exit: group.Refresh")

	result, err := Query(`
		SELECT PropName, PropValue, PropType
		FROM Props WHERE EntityID=? `,
		g.DbID)
	defer result.Close()

	if err != nil {
		log.Printf("Error refreshing Group(%s): %s", g.ID, err)
		return fmt.Errorf("Error refreshing group(%s): %s", g.ID, err)
	}

	*g = Group{ // Erase all existing properties
		Entity: Entity{
			RegistryID: g.RegistryID,
			DbID:       g.DbID,
			Plural:     g.Plural,
		},
		ID: g.ID,
	}

	for result.NextRow() {
		name := NotNilString(result.Data[0])
		val := NotNilString(result.Data[1])
		propType := NotNilString(result.Data[2])
		SetField(g, name, &val, propType)
	}

	return nil
}

func (g *Group) SetName(val string) error { return g.Set("Name", val) }
func (g *Group) SetEpoch(val int) error   { return g.Set("Epoch", val) }

func (g *Group) Set(name string, val any) error {
	return SetProp(g, name, val)
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

	// for i, key := range SortedKeys(g.GroupCollection.GroupModel.Resources) {
	gm := Registries[g.RegistryID].Model.Groups[g.Plural]
	for i, key := range SortedKeys(gm.Resources) {
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

func (g *Group) FindResource(rType string, id string) *Resource {
	log.VPrintf(3, ">Enter: FindResource(%s,%s)", rType, id)
	defer log.VPrintf(3, "<Exit: FindResource")

	results, _ := NewQuery(`
	        SELECT r.ID, p.PropName, p.PropValue, p.PropType
			FROM Resources as r LEFT JOIN Props AS p ON (p.EntityID=r.ID)
			WHERE r.GroupID=? AND r.ResourceID=?`, g.DbID, id)

	r := (*Resource)(nil)
	for _, row := range results {
		if r == nil {
			r = &Resource{
				Entity: Entity{
					RegistryID: g.RegistryID,
					DbID:       NotNilString(row[0]),
					Plural:     rType,
				},
				Group: g,
				ID:    id,
			}
			log.VPrintf(3, "Found one: %s", r.DbID)
		}
		if *row[1] != nil { // We have Props
			name := NotNilString(row[1])
			val := NotNilString(row[2])
			propType := NotNilString(row[3])
			SetField(r, name, &val, propType)
		}
	}

	if r == nil {
		log.VPrintf(3, "None found")
	}
	return r
}

func (g *Group) FindOrAddResource(rType string, id string) *Resource {
	log.VPrintf(3, ">Enter: FindOrAddResource(%s,%s)", rType, id)
	defer log.VPrintf(3, "<Exit: FindOrAddResource")

	r := g.FindResource(rType, id)
	if r != nil {
		log.VPrintf(3, "Found one")
		return r
	}

	r = &Resource{
		Entity: Entity{
			RegistryID: g.RegistryID,
			DbID:       NewUUID(),
			Plural:     rType,
		},
		Group: g,
		ID:    id,
	}

	err := DoOne(`
		INSERT INTO Resources(ID, ResourceID, GroupID, ModelID, Path, Abstract)
		SELECT ?,?,?,ID,?,? FROM ModelEntities WHERE Plural=?`,
		r.DbID, id, g.DbID,
		g.Plural+"/"+g.DbID+"/"+rType+"/"+r.DbID,
		g.Plural+"/"+rType,
		rType)
	if err != nil {
		log.Printf("Error adding resource: %s", err)
		return nil
	}
	r.Set(".id", r.ID)

	log.VPrintf(3, "Created new one - dbID: %s", r.DbID)
	return r
}

func (g *Group) OldFindOrAddResource(rType string, id string) *Resource {
	rc := g.FindOrAddResourceCollection(rType)
	res := rc.Resources[id]
	if res != nil {
		return res
	}
	return rc.NewResource(id)
}
