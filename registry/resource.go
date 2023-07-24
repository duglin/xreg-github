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
	log.VPrintf(4, "Get: r(%s).Get(%s)", r.UID, name)
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
	log.VPrintf(4, "Set: r(%s).Set(%s,%v)", r.UID, name, val)
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
	return v.Set(name, val)
}

// Maybe replace error with a panic? same for other finds??
func (r *Resource) FindVersion(id string) (*Version, error) {
	log.VPrintf(3, ">Enter: FindVersion(%s)", id)
	defer log.VPrintf(3, "<Exit: FindVersion")

	results, err := Query(`
        SELECT v.SID, p.PropName, p.PropValue, p.PropType
        FROM Versions as v LEFT JOIN Props AS p ON (p.EntitySID=v.SID)
        WHERE v.UID=? AND v.ResourceSID=?`, id, r.DbSID)
	defer results.Close()

	if err != nil {
		return nil, fmt.Errorf("Error finding Version %q: %s", id, err)
	}

	v := (*Version)(nil)
	for row := results.NextRow(); row != nil; row = results.NextRow() {
		if v == nil {
			v = &Version{
				Entity: Entity{
					RegistrySID: r.RegistrySID,
					DbSID:       NotNilString(row[0]),
					UID:         id,

					Level:    3,
					Path:     r.Group.Plural + "/" + r.Group.UID + "/" + r.Plural + "/" + r.UID + "/versions/" + id,
					Abstract: r.Group.Plural + "/" + r.Plural + "/versions",
				},
				Resource: r,
			}
			log.VPrintf(3, "Found one: %s", v.DbSID)
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
			RegistrySID: r.RegistrySID,
			DbSID:       NewUUID(),
			UID:         id,

			Level:    3,
			Path:     r.Group.Plural + "/" + r.Group.UID + "/" + r.Plural + "/" + r.UID + "/versions/" + id,
			Abstract: r.Group.Plural + "/" + r.Plural + "/versions",
		},
		Resource: r,
	}

	err = DoOne(`
        INSERT INTO Versions(SID, UID, ResourceSID, Path, Abstract)
        VALUES(?,?,?,?,?)`,
		v.DbSID, id, r.DbSID,
		r.Group.Plural+"/"+r.Group.UID+"/"+r.Plural+"/"+r.UID+"/versions/"+v.UID,
		r.Group.Plural+"/"+r.Plural+"/versions")
	if err != nil {
		err = fmt.Errorf("Error adding Version: %s", err)
		log.Print(err)
		return nil, err
	}
	v.Set("id", id)
	// v.Set("epoch", 1)

	err = r.Set("latestId", id)
	if err != nil {
		// v.Delete()
		err = fmt.Errorf("Error setting latestId: %s", err)
		return v, err
	}

	log.VPrintf(3, "Created new one - dbSID: %s", v.DbSID)
	return v, nil
}

func (r *Resource) Delete() error {
	log.VPrintf(3, ">Enter: Resource.Delete(%s)", r.UID)
	defer log.VPrintf(3, "<Exit: Resource.Delete")

	return DoOne(`DELETE FROM Resources WHERE SID=?`, r.DbSID)
}

// This neesd to match the merging algorithm in the 'LatestProps' view in SQL
func (r *Resource) MergeProps(entity *Entity) {
	for k, v := range entity.Props {
		// Grab all of the Version's Props except 'id'
		if k == "id" {
			continue
		}
		r.Props[k] = v
	}
}
