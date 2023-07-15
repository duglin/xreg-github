package registry

import (
	"fmt"

	log "github.com/duglin/dlog"
)

type Resource struct {
	Entity
	Group *Group
}

func (r *Resource) Get(name string) any {
	log.VPrintf(4, "Get: r(%s).Get(%s)", r.ID, name)
	if name[0] == '.' { // Force it to be on the Resource, not latest Version
		return r.Entity.Get(name[1:])
	}

	if name == "id" || name == "latestId" || name == "latestUrl" {
		return r.Entity.Get(name)
	}

	v := r.GetLatest()
	return v.Entity.Get(name)
}

func (r *Resource) Set(name string, val any) error {
	log.VPrintf(4, "Set: r(%s).Set(%s,%v)", r.ID, name, val)
	if name[0] == '.' { // Force it to be on the Resource, not latest Version
		return SetProp(r, name[1:], val)
	}

	if name == "id" || name == "latestId" || name == "latestUrl" {
		return SetProp(r, name, val)
	}

	v := r.GetLatest()
	return SetProp(v, name, val)
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
					ID:         id,
				},
				Resource: r,
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

func (r *Resource) GetLatest() *Version {
	val := r.Get("latestId")
	if val == nil {
		panic("No latest is set")
	}

	return r.FindVersion(val.(string))
}

func (r *Resource) AddVersion(id string) (*Version, error) {
	log.VPrintf(3, ">Enter: AddVersion%s)", id)
	defer log.VPrintf(3, "<Exit: AddVersion")

	v := &Version{
		Entity: Entity{
			RegistryID: r.RegistryID,
			DbID:       NewUUID(),
			ID:         id,
		},
		Resource: r,
	}

	err := DoOne(`
		INSERT INTO Versions(ID, VersionID, ResourceID, Path, Abstract)
		VALUES(?,?,?,?,?)`,
		v.DbID, id, r.DbID,
		r.Group.Plural+"/"+r.Group.ID+"/"+r.Plural+"/"+r.ID+"/versions/"+v.ID,
		r.Group.Plural+"/"+r.Plural+"/versions")
	if err != nil {
		err = fmt.Errorf("Error added version: %s", err)
		log.Print(err)
		return nil, err
	}
	v.Set("id", id)

	if r.Get("latestId") == nil {
		r.Set("latestId", id)
	}

	log.VPrintf(3, "Created new one - dbID: %s", v.DbID)
	return v, nil
}
