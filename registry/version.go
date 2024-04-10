package registry

import (
	"fmt"

	log "github.com/duglin/dlog"
)

type Version struct {
	Entity
	Resource *Resource
}

func (v *Version) Get(name string) any {
	return v.Entity.Get(name)
}

func (v *Version) SetCommit(name string, val any) error {
	return v.Entity.SetCommit(name, val)
}

func (v *Version) JustSet(name string, val any) error {
	return v.Entity.JustSet(NewPPP(name), val)
}

func (v *Version) SetSave(name string, val any) error {
	return v.Entity.SetSave(name, val)
}

func (v *Version) Delete(nextVersionID string) error {
	log.VPrintf(3, ">Enter: Version.Delete(%s, %s)", v.UID, nextVersionID)
	defer log.VPrintf(3, "<Exit: Version.Delete")

	if nextVersionID == v.UID {
		return fmt.Errorf("Can't set defaultversionid to Version being deleted")
	}

	// Zero is ok if it's already been deleted
	err := DoZeroOne(v.tx, `DELETE FROM Versions WHERE SID=?`, v.DbSID)
	if err != nil {
		return fmt.Errorf("Error deleting Version %q: %s", v.UID, err)
	}

	// On zero, we'll continue and process the nextVersionID... should we?

	vIDs, err := v.Resource.GetVersionIDs()
	if err != nil {
		return fmt.Errorf("Error deleting Version %q: %s", v.UID, err)
	}

	if len(vIDs) == 0 {
		// If there are no more Versions left, delete the Resource
		// TODO: Could just do this instead of deleting the Version first?
		return v.Resource.Delete()
	}

	nextVersion := (*Version)(nil)
	currentDefault := v.Resource.Get("defaultversionid")
	mustChange := (v.UID == currentDefault)

	// If they explicitly told us to unset the default version or we're
	// deleting the current default w/o a new vID being given, then unstick it
	if (nextVersionID == "" && mustChange) || nextVersionID == "null" {
		v.Resource.SetDefault(nil)
	} else if nextVersionID != "" {
		nextVersion, err = v.Resource.FindVersion(nextVersionID)
		if err != nil {
			return err
		}
		if nextVersion == nil {
			return fmt.Errorf("Can't find next default Version %q",
				nextVersionID)
		}

		if err = v.Resource.SetDefault(nextVersion); err != nil {
			return err
		}
	}

	return nil
}

func (v *Version) SetDefault() error {
	return v.Resource.SetDefault(v)
}
