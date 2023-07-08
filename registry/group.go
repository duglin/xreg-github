package registry

import (
	// "fmt"

	log "github.com/duglin/dlog"
)

type Group struct {
	Entity
}

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
					ID:         id,
				},
				Group: g,
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
			ID:         id,
		},
		Group: g,
	}

	err := DoOne(`
		INSERT INTO Resources(ID, ResourceID, GroupID, ModelID, Path, Abstract)
		SELECT ?,?,?,ID,?,? FROM ModelEntities WHERE Plural=?`,
		r.DbID, r.ID, g.DbID,
		g.Plural+"/"+g.ID+"/"+rType+"/"+r.ID,
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

func (g *Group) AddResource(rType string, rID string, vID string) *Resource {
	r := g.FindOrAddResource(rType, rID)
	if r != nil {
		r.FindOrAddVersion("v0")
	}
	return r
}
