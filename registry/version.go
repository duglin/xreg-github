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

func (v *Version) Set(name string, val any) error {
	return v.Entity.Set(name, val)
}

func (v *Version) JustSet(name string, val any) error {
	return v.Entity.JustSet(NewPPP(name), val)
}

func (v *Version) Delete(nextVersionID string) error {
	log.VPrintf(3, ">Enter: Version.Delete(%s, %s)", v.UID, nextVersionID)
	defer log.VPrintf(3, "<Exit: Version.Delete")

	var err error
	nextVersion := (*Version)(nil)
	if nextVersionID == v.UID {
		return fmt.Errorf("Can't set latestversionid to Version being deleted")
	}

	currentLatest := v.Resource.Get("latestversionid")
	mustChange := (v.UID == currentLatest)

	if nextVersionID == "" && mustChange {
		results, err := Query(`
        SELECT UID FROM Versions
        WHERE ResourceSID=? AND UID<>?
        ORDER BY Counter DESC LIMIT 1`,
			v.Resource.DbSID, v.UID)
		defer results.Close()

		if err != nil {
			return fmt.Errorf("Error finding next latestVersionID for "+
				"Resource "+"%q: %s",
				v.Resource.UID, err)
		}
		row := results.NextRow()
		if row != nil {
			nextVersionID = NotNilString(row[0])
		}
	}

	if nextVersionID != "" && nextVersionID != currentLatest {
		nextVersion, err = v.Resource.FindVersion(nextVersionID)
		if err != nil {
			return err
		}
		if nextVersion == nil {
			return fmt.Errorf("Can't find next latest Version %q",
				nextVersionID)
		}

		if err = v.Resource.SetLatest(nextVersion); err != nil {
			return err
		}
	}

	err = DoOne(`DELETE FROM Versions WHERE SID=?`, v.DbSID)
	if err != nil {
		return fmt.Errorf("Error deleting Version %q: %s", v.UID, err)
	}

	// If there is no next version AND we need to change it, then there
	// must not be any more versions to use, so delete the Resource
	if nextVersion == nil && mustChange {
		// No more versions so delete the Resource
		// TODO: Could just do this instead of deleting the Version first?
		return v.Resource.Delete()
	}

	return nil
}

func (v *Version) SetLatest() error {
	return v.Resource.SetLatest(v)
}
