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
	log.VPrintf(3, ">Enter: Version.Delete(%s)", v.ID)
	defer log.VPrintf(3, "<Exit: Version.Delete")

	err := DoOne(`DELETE FROM Versions WHERE ID=?`, v.DbID)
	if err != nil {
		return fmt.Errorf("Error deleting version %q: %s", v.ID, err)
	}

	results, err := Query(`
        SELECT VersionID FROM Versions
        WHERE ResourceID=?
        ORDER BY CreatedIndex DESC LIMIT 1`,
		v.Resource.DbID)
	if err != nil {
		return fmt.Errorf("Error finding next latestID for Resource %q: %s",
			v.Resource.ID, err)
	}

	if len(results) == 0 {
		// No more versions so delete the Resource
		return v.Resource.Delete()
	}

	latestID := NotNilString(results[0][0])
	return v.Resource.Set("latestId", latestID)
}
