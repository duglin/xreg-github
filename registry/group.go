package registry

import (
	"fmt"
	// "io"
	// "strings"

	log "github.com/duglin/dlog"
)

type Group struct {
	Entity

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

func (g *Group) SetName(val string) error { return g.Set("name", val) }
func (g *Group) SetEpoch(val int) error   { return g.Set("epoch", val) }

func (g *Group) Set(name string, val any) error {
	return SetProp(g, name, val)
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
