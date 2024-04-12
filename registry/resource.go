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

// These attributes are on the Resource not the Version
// We used to use a "." as a prefix to know - may still need to at some point
var specialResourceAttrs = map[string]bool{
	"id":               true,
	"defaultversionid": true,
	"setdefault":       true,
	"#nextversionid":   true,
}

func (r *Resource) Get(name string) any {
	log.VPrintf(4, "Get: r(%s).Get(%s)", r.UID, name)

	if specialResourceAttrs[name] {
		return r.Entity.Get(name)
	}

	v, err := r.GetDefault()
	if err != nil {
		panic(err)
	}
	return v.Entity.Get(name)
}

func (r *Resource) SetCommit(name string, val any) error {
	log.VPrintf(4, "Set: r(%s).SetCommit(%s,%v)", r.UID, name, val)
	if specialResourceAttrs[name] {
		return r.Entity.SetCommit(name, val)
	}

	v, err := r.GetDefault()
	if err != nil {
		panic(err)
	}

	return v.SetCommit(name, val)
}

func (r *Resource) JustSet(name string, val any) error {
	log.VPrintf(4, "JustSet: r(%s).JustSet(%s,%v)", r.UID, name, val)
	if specialResourceAttrs[name] {
		return r.Entity.JustSet(NewPPP(name), val)
	}

	v, err := r.GetDefault()
	if err != nil {
		panic(err)
	}

	return v.JustSet(name, val)
}

func (r *Resource) SetSave(name string, val any) error {
	log.VPrintf(4, "SetSave: r(%s).SetSave(%s,%v)", r.UID, name, val)
	if specialResourceAttrs[name] {
		return r.Entity.SetSave(name, val)
	}

	v, err := r.GetDefault()
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
func (r *Resource) GetDefault() (*Version, error) {
	val := r.Get("defaultversionid")
	if IsNil(val) {
		return nil, nil
		// panic("No default is set")
	}

	return r.FindVersion(val.(string))
}

// Only call this if you want things to be sticky (when not nil).
// Creating a new version should do this directly
func (r *Resource) SetDefault(newDefault *Version) error {
	// already set
	if newDefault != nil && r.Get("defaultversionid") == newDefault.UID {
		// But make sure we're sticky, could just be a coincidence
		if r.Get("setdefault") != true {
			return r.SetSave("setdefault", true)
		}
		return nil
	}

	if newDefault == nil {
		if err := r.JustSet("setdefault", nil); err != nil {
			return err
		}

		vIDs, err := r.GetVersionIDs()
		if err != nil {
			return err
		}
		newDefault, err = r.FindVersion(vIDs[len(vIDs)-1])
		if err != nil {
			return err
		}
		PanicIf(newDefault == nil, "No newest")
	} else {
		if err := r.JustSet("setdefault", true); err != nil {
			return err
		}
	}

	return r.SetSave("defaultversionid", newDefault.UID)
}

func (r *Resource) AddVersion(id string, objs ...Object) (*Version, error) {
	log.VPrintf(3, ">Enter: AddVersion%s)", id)
	defer log.VPrintf(3, "<Exit: AddVersion")

	var v *Version
	var err error

	if id == "" {
		// No versionID provided so grab the next available one
		tmp := r.Get("#nextversionid")
		nextID := NotNilInt(&tmp)
		for {
			id = strconv.Itoa(nextID)
			v, err = r.FindVersion(id)
			if err != nil {
				return nil, fmt.Errorf("Error checking for Version %q: %s",
					id, err)
			}

			// Increment no matter what since it's "next" not "default"
			nextID++

			if v == nil {
				r.JustSet("#nextversionid", nextID)
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

	if err = v.ValidateAndSave(); err != nil {
		return nil, err
	}

	// If we can only have one Version, then set the one we just created
	// as the default.
	// Also set it if we're not sticky w.r.t. default version
	_, rm := r.GetModels()
	if rm.MaxVersions == 1 || r.Get("setdefault") != true {
		err = r.SetSave("defaultversionid", v.UID)
		if err != nil {
			return nil, err
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

	tmp := r.Get("defaultversionid")
	defaultID := NotNilString(&tmp)

	// Starting with the oldest, keep deleting until we reach the max
	// number of Versions allowed. Technically, this should always just
	// delete 1, but ya never know. Also, skip the one that's tagged
	// as "default" since that one is special
	count := len(vIDs)
	for count > rm.MaxVersions {
		// Skip the "default" Version
		if vIDs[0] != defaultID {
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
