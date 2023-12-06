package registry

import (
	"fmt"

	log "github.com/duglin/dlog"
)

type Group struct {
	Entity
	Registry *Registry
}

func (g *Group) Get(name string) any {
	return g.Entity.GetPropFromUI(name)
}

func (g *Group) Set(name string, val any) error {
	return g.Entity.SetFromUI(name, val)
}

func (g *Group) FindResource(rType string, id string) (*Resource, error) {
	log.VPrintf(3, ">Enter: FindResource(%s,%s)", rType, id)
	defer log.VPrintf(3, "<Exit: FindResource")

	results, err := Query(`
                SELECT r.SID, p.PropName, p.PropValue, p.PropType
                FROM Resources as r
                LEFT JOIN Props AS p ON (p.EntitySID=r.SID)
                WHERE r.GroupSID=? AND r.UID=?
                AND r.Abstract = CONCAT(?,'`+string(DB_IN)+`',?)`,
		g.DbSID, id, g.Plural, rType)
	defer results.Close()

	if err != nil {
		return nil, fmt.Errorf("Error finding Resource %q(%s): %s",
			id, rType, err)
	}

	r := (*Resource)(nil)
	for row := results.NextRow(); row != nil; row = results.NextRow() {
		if r == nil {
			r = &Resource{
				Entity: Entity{
					RegistrySID: g.RegistrySID,
					DbSID:       NotNilString(row[0]),
					Plural:      rType,
					UID:         id,

					Level:    2,
					Path:     g.Plural + "/" + g.UID + "/" + rType + "/" + id,
					Abstract: g.Plural + string(DB_IN) + rType,
				},
				Group: g,
			}
			log.VPrintf(3, "Found one: %s", r.DbSID)
		}
		if *row[1] != nil { // We have Props
			name := NotNilString(row[1])
			val := NotNilString(row[2])
			propType := NotNilString(row[3])
			r.Entity.SetPropFromString(name, &val, propType)
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
			RegistrySID: g.RegistrySID,
			DbSID:       NewUUID(),
			Plural:      rType,
			UID:         id,

			Level:    2,
			Path:     g.Plural + "/" + g.UID + "/" + rType + "/" + id,
			Abstract: g.Plural + string(DB_IN) + rType,
		},
		Group: g,
	}

	err = DoOne(`
        INSERT INTO Resources(SID, UID, GroupSID, ModelSID, Path, Abstract)
        SELECT ?,?,?,SID,?,?
        FROM ModelEntities
        WHERE RegistrySID=?
          AND ParentSID IN (
            SELECT SID FROM ModelEntities
            WHERE RegistrySID=?
            AND ParentSID IS NULL
            AND Plural=?)
            AND Plural=?`,
		r.DbSID, r.UID, g.DbSID,
		g.Plural+"/"+g.UID+"/"+rType+"/"+r.UID, g.Plural+string(DB_IN)+rType,
		g.RegistrySID,
		g.RegistrySID, g.Plural,
		rType)
	if err != nil {
		err = fmt.Errorf("Error adding Resource: %s", err)
		log.Print(err)
		return nil, err
	}
	r.Set(".id", r.UID)
	r.Set(".#nextVersionID", 1)

	_, err = r.AddVersion(vID)
	if err != nil {
		return nil, err
	}

	log.VPrintf(3, "Created new one - dbSID: %s", r.DbSID)
	return r, err
}

func (g *Group) Delete() error {
	log.VPrintf(3, ">Enter: Group.Delete(%s)", g.UID)
	defer log.VPrintf(3, "<Exit: Group.Delete")

	return DoOne(`DELETE FROM "Groups" WHERE SID=?`, g.DbSID)
}
