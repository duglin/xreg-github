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

func (r *Resource) SetCommit(name string, val any) error {
	log.VPrintf(4, "Set: r(%s).SetCommit(%s,%v)", r.UID, name, val)
	if name[0] == '.' { // Force it to be on the Resource, not latest Version
		if name == ".latestVersionId" {
			log.Printf("Shouldn't be setting .latestVersionId directly-1")
			panic("can't set .latestversionid directly")
		}
		return r.Entity.SetCommit(name[1:], val)
	}

	if name == "id" || name == "latestversionid" || name == "latestversionurl" {
		if name == "latestversionid" {
			log.Printf("Shouldn't be setting .latestVersionId directly-2")
			panic("can't set .latestversionid directly")
		}
		return r.Entity.SetCommit(name, val)
	}

	v, err := r.GetLatest()
	if err != nil {
		panic(err)
	}

	return v.SetCommit(name, val)
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

	return v.SetSave(name, val)
}

// Maybe replace error with a panic? same for other finds??
func (r *Resource) FindVersion(id string) (*Version, error) {
	log.VPrintf(3, ">Enter: FindVersion(%s)", id)
	defer log.VPrintf(3, "<Exit: FindVersion")

	if v := r.tx.GetVersion(r, id); v != nil {
		return v, nil
	}

	ent, err := RawEntityFromPath(r.tx, r.Group.Registry.DbSID,
		r.Group.Plural+"/"+r.Group.UID+"/"+r.Plural+"/"+r.UID+"/versions/"+id)
	if err != nil {
		return nil, fmt.Errorf("Error finding Version %q: %s", id, err)
	}
	if ent == nil {
		log.VPrintf(3, "None found")
		return nil, nil
	}

	v := &Version{Entity: *ent, Resource: r}
	v.tx.AddVersion(v)
	return v, nil
}

// Maybe replace error with a panic?
func (r *Resource) GetLatest() (*Version, error) {
	val := r.Get("latestversionid")
	if IsNil(val) {
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

func (r *Resource) AddVersion(id string, latest bool, objs ...Object) (*Version, error) {
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
				r.JustSet(".#nextVersionID", nextID)
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

	v.tx.AddVersion(v)

	if err = v.JustSet("id", id); err != nil {
		return nil, err
	}

	for _, obj := range objs {
		v.ConvertStrings(obj)
		for key, val := range obj {
			if key == "id" && val != id {
				return nil, fmt.Errorf("The IDs must match(%q vs %q)", id, val)
			}
			if err = v.JustSet(key, val); err != nil {
				return nil, err
			}
		}
	}
	l, ok := v.NewObject["latest"]
	if !ok {
		l = latest
	} else {
		// if "latest" was part of the data then it's from a user, make
		// sure they're allowed to set it
		_, rm := r.GetModels()
		if l != latest && rm.SetLatest == false {
			return nil, fmt.Errorf(`"latest" can not be "%v", it is `+
				`controlled by the server`, latest)
		}
	}

	vIDs, err := r.GetVersionIDs()
	if err != nil {
		return nil, err
	}
	if len(vIDs) == 1 && l == false {
		return nil, fmt.Errorf(`"latest" can not be "false" since ` +
			`doing so would result in no latest version`)
	}

	if err = v.ValidateAndSave(); err != nil {
		return nil, err
	}

	// If we can only have one Version, then set the one we just created
	// as the latest. Basically the "latest" flag is pointless in this case
	_, rm := r.GetModels()
	if rm.MaxVersions == 1 {
		r.SetLatest(v)
	} else if latest {
		err = r.SetLatest(v)
		if err != nil {
			err = fmt.Errorf("Error setting latestVersionId: %s", err)
			return v, err
		}
	}

	// If we've reached the maximum # of Versions, then delete oldest
	if err = r.EnsureMaxVersions(); err != nil {
		return nil, err
	}

	log.VPrintf(3, "Created new one - dbSID: %s", v.DbSID)
	return v, nil
}

func (r *Resource) GetVersionIDs() ([]string, error) {
	// Get the list of Version IDs for this Resource (oldest first)
	results, err := Query(r.tx, `
			SELECT UID,Counter FROM Versions
			WHERE ResourceSID=? ORDER BY Counter ASC`,
		r.DbSID)
	defer results.Close()

	if err != nil {
		return nil, fmt.Errorf("Error counting Versions: %s", err)
	}

	vIDs := []string{}
	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		vIDs = append(vIDs, NotNilString(row[0]))
	}
	results.Close()
	return vIDs, nil
}

func (r *Resource) EnsureMaxVersions() error {
	_, rm := r.GetModels()
	if rm.MaxVersions == 0 {
		// No limit, so just exit
		return nil
	}

	vIDs, err := r.GetVersionIDs()
	if err != nil {
		return err
	}
	PanicIf(len(vIDs) == 0, "Query can't be empty")

	tmp := r.Get("latestversionid")
	latestID := NotNilString(&tmp)

	// Starting with the oldest, keep deleting until we reach the max
	// number of Versions allowed. Technically, this should always just
	// delete 1, but ya never know. Also, skip the one that's tagged
	// as "latest" since that one is special
	count := len(vIDs)
	for count > rm.MaxVersions {
		// Skip the "latest" Version
		if vIDs[0] != latestID {
			err = DoOne(r.tx, `DELETE FROM Versions
					WHERE ResourceSID=? AND UID=?`, r.DbSID, vIDs[0])
			if err != nil {
				return fmt.Errorf("Error deleting Version %q: %s", vIDs[0], err)
			}
			count--
		}
		vIDs = vIDs[1:]
	}
	return nil
}

func (r *Resource) Delete() error {
	log.VPrintf(3, ">Enter: Resource.Delete(%s)", r.UID)
	defer log.VPrintf(3, "<Exit: Resource.Delete")

	return DoOne(r.tx, `DELETE FROM Resources WHERE SID=?`, r.DbSID)
}

func (r *Resource) GetVersions() ([]*Version, error) {
	list := []*Version{}

	entities, err := RawEntitiesFromQuery(r.tx, r.Registry.DbSID,
		`ParentSID=?`, r.DbSID)
	if err != nil {
		return nil, err
	}

	for _, e := range entities {
		v := r.tx.GetVersion(r, e.UID)
		if v == nil {
			v = &Version{Entity: *e, Resource: r}
			v.tx.AddVersion(v)
		}
		list = append(list, v)
	}

	return list, nil
}
