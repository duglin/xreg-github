package registry

import (
	"fmt"

	log "github.com/duglin/dlog"
)

type Version struct {
	Entity
	Resource *Resource
}

func (v *Version) Set(name string, val any) error {
	return SetProp(v, name, val)
}

func (v *Version) Delete() error {
	log.VPrintf(3, ">Enter: Version.Delete(%s)", v.UID)
	defer log.VPrintf(3, "<Exit: Version.Delete")

	err := DoOne(`DELETE FROM Versions WHERE SID=?`, v.DbSID)
	if err != nil {
		return fmt.Errorf("Error deleting Version %q: %s", v.UID, err)
	}

	results, err := Query(`
        SELECT UID FROM Versions
        WHERE ResourceSID=?
        ORDER BY Counter DESC LIMIT 1`,
		v.Resource.DbSID)
	defer results.Close()

	if err != nil {
		return fmt.Errorf("Error finding next latestVersionID for Resource "+
			"%q: %s", v.Resource.UID, err)
	}

	row := results.NextRow()

	if row == nil {
		// No more versions so delete the Resource
		return v.Resource.Delete()
	}

	latestID := NotNilString(row[0])

	v, err = v.Resource.FindVersion(latestID)
	if err != nil {
		return err
	}
	return v.Resource.SetLatest(v)
}
