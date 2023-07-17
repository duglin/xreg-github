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

	v, err := r.GetLatest()
	if err != nil {
		panic(err)
	}
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

	v, err := r.GetLatest()
	if err != nil {
		panic(err)
	}
	return SetProp(v, name, val)
}

// Maybe replace error with a panic? same for other finds??
func (r *Resource) FindVersion(id string) (*Version, error) {
	log.VPrintf(3, ">Enter: FindVersion(%s)", id)
	defer log.VPrintf(3, "<Exit: FindVersion")

	results, err := Query(`
        SELECT v.ID, p.PropName, p.PropValue, p.PropType
        FROM Versions as v LEFT JOIN Props AS p ON (p.EntityID=v.ID)
        WHERE v.VersionID=? AND v.ResourceID=?`, id, r.DbID)
	defer results.Close()

	if err != nil {
		return nil, fmt.Errorf("Error finding Version %q: %s", id, err)
	}

	v := (*Version)(nil)
	for row := results.NextRow(); row != nil; row = results.NextRow() {
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

	return v, nil
}

// Maybe replace error with a panic?
func (r *Resource) GetLatest() (*Version, error) {
	val := r.Get("latestId")
	if val == nil {
		panic("No latest is set")
	}

	return r.FindVersion(val.(string))
}

func (r *Resource) AddVersion(id string) (*Version, error) {
	log.VPrintf(3, ">Enter: AddVersion%s)", id)
	defer log.VPrintf(3, "<Exit: AddVersion")

	v, err := r.FindVersion(id)
	if err != nil {
		return nil, fmt.Errorf("Error checking for Version %q: %s", id, err)
	}
	if v != nil {
		return nil, fmt.Errorf("Version %q already exists", id)
	}

	v = &Version{
		Entity: Entity{
			RegistryID: r.RegistryID,
			DbID:       NewUUID(),
			ID:         id,
		},
		Resource: r,
	}

	err = DoOne(`
        INSERT INTO Versions(ID, VersionID, ResourceID, Path, Abstract)
        VALUES(?,?,?,?,?)`,
		v.DbID, id, r.DbID,
		r.Group.Plural+"/"+r.Group.ID+"/"+r.Plural+"/"+r.ID+"/versions/"+v.ID,
		r.Group.Plural+"/"+r.Plural+"/versions")
	if err != nil {
		err = fmt.Errorf("Error adding Version: %s", err)
		log.Print(err)
		return nil, err
	}
	v.Set("id", id)

	err = r.Set("latestId", id)
	if err != nil {
		// v.Delete()
		err = fmt.Errorf("Error setting latestId: %s", err)
		return v, err
	}

	log.VPrintf(3, "Created new one - dbID: %s", v.DbID)
	return v, nil
}

func (r *Resource) Delete() error {
	log.VPrintf(3, ">Enter: Resource.Delete(%s)", r.ID)
	defer log.VPrintf(3, "<Exit: Resource.Delete")

	return DoOne(`DELETE FROM Resources WHERE ID=?`, r.DbID)
}
