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

	// Names that starts with a dot(.) means it's a resource prop, not ver prop
	if name[0] == '.' {
		return r.Entity.Get(name[1:])
	}

	// These are also resource properties, not vesion properties
	if name == "id" || name == "latestversionid" ||
		name == "latestversionurl" || name == "#nextVersionID" {

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
		if name == ".latestVersionId" {
			log.Printf("Shouldn't be setting .latestVersionId directly-1")
			panic("can't set .latestversionid directly")
		}
		return r.Entity.Set(name[1:], val)
	}

	if name == "id" || name == "latestversionid" || name == "latestversionurl" {
		if name == "latestversionid" {
			log.Printf("Shouldn't be setting .latestVersionId directly-2")
			panic("can't set .latestversionid directly")
		}
		return r.Entity.Set(name, val)
	}

	v, err := r.GetLatest()
	if err != nil {
		panic(err)
	}

	v.SkipEpoch = r.SkipEpoch
	return v.Set(name, val)
}

func (r *Resource) JustSet(name string, val any) error {
	log.VPrintf(4, "JustSet: r(%s).JustSet(%s,%v)", r.UID, name, val)
	if name[0] == '.' { // Force it to be on the Resource, not latest Version
		if name == ".latestVersionId" {
			log.Printf("Shouldn't be setting .latestVersionId directly-1")
			panic("can't set .latestversionid directly")
		}
		return r.Entity.JustSet(NewPPP(name[1:]), val)
	}

	if name == "id" || name == "latestversionid" || name == "latestversionurl" {
		if name == "latestversionid" {
			log.Printf("Shouldn't be setting .latestVersionId directly-2")
			panic("can't set .latestversionid directly")
		}
		return r.Entity.JustSet(NewPPP(name), val)
	}

	v, err := r.GetLatest()
	if err != nil {
		panic(err)
	}

	v.SkipEpoch = r.SkipEpoch
	return v.JustSet(name, val)
}

func (r *Resource) SetSave(name string, val any) error {
	log.VPrintf(4, "SetSave: r(%s).SetSave(%s,%v)", r.UID, name, val)
	if name[0] == '.' { // Force it to be on the Resource, not latest Version
		if name == ".latestVersionId" {
			log.Printf("Shouldn't be setting .latestVersionId directly-1")
			panic("can't set .latestversionid directly")
		}
		return r.Entity.SetSave(name[1:], val)
	}

	if name == "id" || name == "latestversionid" || name == "latestversionurl" {
		if name == "latestversionid" {
			log.Printf("Shouldn't be setting .latestVersionId directly-2")
			panic("can't set .latestversionid directly")
		}
		return r.Entity.SetSave(name, val)
	}

	v, err := r.GetLatest()
	if err != nil {
		panic(err)
	}

	v.SkipEpoch = r.SkipEpoch
	return v.SetSave(name, val)
}

// Maybe replace error with a panic? same for other finds??
func (r *Resource) FindVersion(id string) (*Version, error) {
	log.VPrintf(3, ">Enter: FindVersion(%s)", id)
	defer log.VPrintf(3, "<Exit: FindVersion")

	ent, err := RawEntityFromPath(r.tx, r.Group.Registry.DbSID,
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
	val := r.Get("latestversionid")
	if val == nil {
		return nil, nil
		// panic("No latest is set")
	}

	return r.FindVersion(val.(string))
}

func (r *Resource) SetLatest(newLatest *Version) error {
	// already set
	if r.Get("latestversionid") == newLatest.UID {
		return nil
	}
	return r.Entity.SetSave("latestversionid", newLatest.UID)
}

func (r *Resource) AddVersion(id string, latest bool) (*Version, error) {
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
				return nil, fmt.Errorf("Error checking for Version %q: %s",
					id, err)
			}

			// Increment no matter what since it's "next" not "latest"
			nextID++

			if v == nil {
				r.SetSave(".#nextVersionID", nextID)
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
			tx: r.tx,

			Registry: r.Registry,
			DbSID:    NewUUID(),
			Plural:   "versions",
			UID:      id,

			Level:    3,
			Path:     r.Group.Plural + "/" + r.Group.UID + "/" + r.Plural + "/" + r.UID + "/versions/" + id,
			Abstract: r.Group.Plural + string(DB_IN) + r.Plural + string(DB_IN) + "versions",
		},
		Resource: r,
	}

	err = DoOne(r.tx, `
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
	if err = v.SetSave("id", id); err != nil {
		return nil, err
	}

	if err = v.SetSave("epoch", 1); err != nil {
		return nil, err
	}

	v.SkipEpoch = false

	if latest {
		err = r.SetLatest(v)
		if err != nil {
			err = fmt.Errorf("Error setting latestVersionId: %s", err)
			return v, err
		}
	}

	log.VPrintf(3, "Created new one - dbSID: %s", v.DbSID)
	return v, nil
}

func (r *Resource) Delete() error {
	log.VPrintf(3, ">Enter: Resource.Delete(%s)", r.UID)
	defer log.VPrintf(3, "<Exit: Resource.Delete")

	return DoOne(r.tx, `DELETE FROM Resources WHERE SID=?`, r.DbSID)
}
