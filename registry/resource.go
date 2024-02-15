package registry

import (
	"fmt"
	"strconv"

	log "github.com/duglin/dlog"
)

type Resource struct {
	Entity
	Group *Group
}

func (r *Resource) Get(name string) any {
	log.VPrintf(4, "Get: r(%s).Get(%s)", r.UID, name)
	if name[0] == '.' { // Force it to be on the Resource, not latest Version
		return r.Entity.GetPropFromUI(name[1:])
	}

	if name == "id" || name == "latestversionid" || name == "latestversionurl" || name == "#nextVersionID" {
		return r.Entity.GetPropFromUI(name)
	}

	v, err := r.GetLatest()
	if err != nil {
		panic(err)
	}
	return v.Entity.GetPropFromUI(name)
}

func (r *Resource) Set(name string, val any) error {
	log.VPrintf(4, "Set: r(%s).Set(%s,%v)", r.UID, name, val)
	if name[0] == '.' { // Force it to be on the Resource, not latest Version
		if name == ".latestVersionId" {
			log.Printf("Shouldn't be setting .latestVersionId directly-1")
			panic("can't set .latestversionid directly")
		}
		return r.Entity.SetFromUI(name[1:], val)
	}

	if name == "id" || name == "latestversionid" || name == "latestversionurl" {
		if name == "latestversionid" {
			log.Printf("Shouldn't be setting .latestVersionId directly-2")
			panic("can't set .latestversionid directly")
		}
		return r.Entity.SetFromUI(name, val)
	}

	v, err := r.GetLatest()
	if err != nil {
		panic(err)
	}
	v.SkipEpoch = r.SkipEpoch
	return v.Set(name, val)
}

// Maybe replace error with a panic? same for other finds??
func (r *Resource) FindVersion(id string) (*Version, error) {
	log.VPrintf(3, ">Enter: FindVersion(%s)", id)
	defer log.VPrintf(3, "<Exit: FindVersion")

	ent, err := RawEntityFromPath(r.Group.Registry.DbSID,
		r.Group.Plural+"/"+r.Group.UID+"/"+r.Plural+"/"+r.UID+"/versions/"+id)
	if err != nil {
		return nil, fmt.Errorf("Error finding Version %q: %s", id, err)
	}
	if ent == nil {
		log.VPrintf(3, "None found")
		return nil, nil
	}

	return &Version{Entity: *ent, Resource: r}, nil
}

// Maybe replace error with a panic?
func (r *Resource) GetLatest() (*Version, error) {
	val := r.GetPropFromUI("latestversionid")
	if val == nil {
		return nil, nil
		// panic("No latest is set")
	}

	return r.FindVersion(val.(string))
}

func (r *Resource) SetLatest(newLatest *Version) error {
	oldLatest, err := r.GetLatest()
	if err != nil {
		panic("Error getting latest: " + err.Error())
	}
	if oldLatest != nil {
		oldLatest.Set("latest", nil)
	}

	// TODO: do both of these in one transaction to make it atomic
	r.Entity.SetFromUI("latestversionid", newLatest.UID)
	newLatest.SkipEpoch = true
	err = newLatest.Set("latest", true)
	newLatest.SkipEpoch = false
	return err
}

func (r *Resource) AddVersion(id string) (*Version, error) {
	log.VPrintf(3, ">Enter: AddVersion%s)", id)
	defer log.VPrintf(3, "<Exit: AddVersion")

	var v *Version
	var err error

	if id == "" {
		// No versionID provided so grab the next available one
		tmp := r.Get("#nextVersionID")
		nextID := NotNilInt(&tmp)
		for {
			id = strconv.Itoa(nextID)
			v, err = r.FindVersion(id)
			if err != nil {
				return nil, fmt.Errorf("Error checking for Version %q: %s", id, err)
			}

			// Increment no matter what since it's "next" not "latest"
			nextID++

			if v == nil {
				r.Set(".#nextVersionID", nextID)
				break
			}
		}
	} else {
		v, err = r.FindVersion(id)

		if err != nil {
			return nil, fmt.Errorf("Error checking for Version %q: %s", id, err)
		}
		if v != nil {
			return nil, fmt.Errorf("Version %q already exists", id)
		}
	}

	v = &Version{
		Entity: Entity{
			RegistrySID: r.RegistrySID,
			DbSID:       NewUUID(),
			Plural:      "versions",
			UID:         id,

			Level:    3,
			Path:     r.Group.Plural + "/" + r.Group.UID + "/" + r.Plural + "/" + r.UID + "/versions/" + id,
			Abstract: r.Group.Plural + string(DB_IN) + r.Plural + string(DB_IN) + "versions",
		},
		Resource: r,
	}

	err = DoOne(`
        INSERT INTO Versions(SID, UID, ResourceSID, Path, Abstract)
        VALUES(?,?,?,?,?)`,
		v.DbSID, id, r.DbSID,
		r.Group.Plural+"/"+r.Group.UID+"/"+r.Plural+"/"+r.UID+"/versions/"+v.UID,
		r.Group.Plural+string(DB_IN)+r.Plural+string(DB_IN)+"versions")
	if err != nil {
		err = fmt.Errorf("Error adding Version: %s", err)
		log.Print(err)
		return nil, err
	}
	v.SkipEpoch = true
	v.Set("id", id)
	v.Set("epoch", 1)
	v.SkipEpoch = false

	err = r.SetLatest(v)
	if err != nil {
		// v.Delete()
		err = fmt.Errorf("Error setting latestVersionId: %s", err)
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
