package registry

import (
	"fmt"

	log "github.com/duglin/dlog"
)

type Version struct {
	Entity
	Resource *Resource
}

var _ EntitySetter = &Version{}

func (v *Version) Get(name string) any {
	return v.Entity.Get(name)
}

func (v *Version) SetCommit(name string, val any) error {
	return v.Entity.eSetCommit(name, val)
}

func (v *Version) JustSet(name string, val any) error {
	return v.Entity.eJustSet(NewPPP(name), val)
}

func (v *Version) SetSave(name string, val any) error {
	return v.Entity.eSetSave(name, val)
}

func (v *Version) Delete() error {
	panic("Should never call this directly - try DeleteSetNextVersion")
}

// JustDelete will delete the Version w/o any additional logic like
// "defaultversionid" manipulation.
// Used when xref on the Resource is set.
func (v *Version) JustDelete() error {
	if err := v.Resource.Touch(); err != nil {
		return err
	}

	// Zero is ok if it's already been deleted
	err := DoZeroOne(v.tx, `DELETE FROM Versions WHERE SID=?`, v.DbSID)
	if err != nil {
		return fmt.Errorf("Error deleting Version %q: %s", v.UID, err)
	}
	return nil
}

func (v *Version) DeleteSetNextVersion(nextVersionID string) error {
	log.VPrintf(3, ">Enter: Version.Delete(%s, %s)", v.UID, nextVersionID)
	defer log.VPrintf(3, "<Exit: Version.Delete")

	if nextVersionID == v.UID {
		return fmt.Errorf("Can't set defaultversionid to Version being deleted")
	}

	// delete it!
	if err := v.JustDelete(); err != nil {
		return fmt.Errorf("Error deleting Version %q: %s", v.UID, err)
	}

	// If it was already gone we'll continue and process the nextVersionID...
	// should we?

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
		nextVersion, err = v.Resource.FindVersion(nextVersionID, false)
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
