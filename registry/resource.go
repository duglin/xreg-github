package registry

import (
	"fmt"
	// "io"
	"strings"

	log "github.com/duglin/dlog"
)

type Resource struct {
	Entity
	Group *Group

	ID        string
	LatestId  string
	LatestUrl string
	Self      string
}

func (r *Resource) Get(name string) any {
	v, _ := r.Extensions[name]
	return v
}

func (r *Resource) Set(name string, val any) error {
	if name[0] == '.' {
		return SetProp(r, name[1:], val)
	}

	lName := strings.ToLower(name)
	if lName == "id" || lName == "latestid" || lName == "latesturl" ||
		lName == "self" {
		return SetProp(r, name, val)
	}

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
