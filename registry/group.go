package registry

import (
	"fmt"

	log "github.com/duglin/dlog"
)

type Group struct {
	Entity
	Registry *Registry
}

func (g *Group) Set(name string, val any) error {
	return SetProp(g, name, val)
}

func (g *Group) FindResource(rType string, id string) (*Resource, error) {
	log.VPrintf(3, ">Enter: FindResource(%s,%s)", rType, id)
	defer log.VPrintf(3, "<Exit: FindResource")

	results, err := Query(`
                SELECT r.ID, p.PropName, p.PropValue, p.PropType
                FROM Resources as r
                LEFT JOIN Props AS p ON (p.EntityID=r.ID)
                WHERE r.GroupID=? AND r.ResourceID=?
                AND r.Abstract = CONCAT(?,'/',?)`,
		g.DbID, id, g.Plural, rType)
	defer results.Close()

	if err != nil {
		return nil, fmt.Errorf("Error finding resource %q(%s): %s",
			id, rType, err)
	}

	r := (*Resource)(nil)
	for row := results.NextRow(); row != nil; row = results.NextRow() {
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
	return r, nil
}

func (g *Group) AddResource(rType string, id string, vID string) (*Resource, error) {
	log.VPrintf(3, ">Enter: AddResource(%s,%s)", rType, id)
	defer log.VPrintf(3, "<Exit: AddResource")

	rModel := g.Registry.Model.Groups[g.Plural].Resources[rType]
	if rModel == nil {
		return nil, fmt.Errorf("Unknown Resource type (%s) for Group %q",
			rType, g.Plural)
	}

	r, err := g.FindResource(rType, id)
	if err != nil {
		return nil, fmt.Errorf("Error checking for Resource(%s) %q: %s",
			rType, id, err)
	}
	if r != nil {
		return nil, fmt.Errorf("Resource %q of type %q already exists",
			id, rType)
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

	err = DoOne(`
        INSERT INTO Resources(ID, ResourceID, GroupID, ModelID, Path, Abstract)
        SELECT ?,?,?,ID,?,?
        FROM ModelEntities
        WHERE RegistryID=?
          AND ParentID IN (
            SELECT ID FROM ModelEntities
            WHERE RegistryID=?
            AND ParentID IS NULL
            AND Plural=?)
            AND Plural=?`,
		r.DbID, r.ID, g.DbID,
		g.Plural+"/"+g.ID+"/"+rType+"/"+r.ID, g.Plural+"/"+rType,
		g.RegistryID,
		g.RegistryID, g.Plural,
		rType)
	if err != nil {
		err = fmt.Errorf("Error adding resource: %s", err)
		log.Print(err)
		return nil, err
	}
	r.Set(".id", r.ID)

	_, err = r.AddVersion(vID)
	if err != nil {
		return nil, err
	}

	log.VPrintf(3, "Created new one - dbID: %s", r.DbID)
	return r, err
}

func (g *Group) Delete() error {
	log.VPrintf(3, ">Enter: Group.Delete(%s)", g.ID)
	defer log.VPrintf(3, "<Exit: Group.Delete")

	return DoOne(`DELETE FROM "Groups" WHERE ID=?`, g.DbID)
}
