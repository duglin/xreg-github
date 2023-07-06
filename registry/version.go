package registry

import (
	"fmt"
	// "io"

	log "github.com/duglin/dlog"
)

type Version struct {
	Entity
	Resource *Resource

	ID          string
	Name        string
	Epoch       int
	Self        string
	Description string
	Docs        string
	Tags        map[string]string
	Format      string
	CreatedBy   string
	CreatedOn   string
	ModifiedBy  string
	ModifiedOn  string

	ResourceURL      string // Send a redirect back to client
	ResourceProxyURL string // The URL to the data, but GET and return data
	ResourceContent  []byte // The raw data

	Data map[string]interface{}
}

func (v *Version) Set(name string, val any) error {
	return SetProp(v, name, val)
}

func (v *Version) Refresh() error {
	log.VPrintf(3, ">Enter: version.Refresh(%s)", v.ID)
	defer log.VPrintf(3, "<Exit: version.Refresh")

	result, err := Query(`
		SELECT PropName, PropValue, PropType
		FROM Props WHERE EntityID=? `,
		v.DbID)
	defer result.Close()

	if err != nil {
		log.Printf("Error refreshing version(%s): %s", v.ID, err)
		return fmt.Errorf("Error refreshing version(%s): %s", v.ID, err)
	}

	*v = Version{ // Erase all existing properties
		Entity: Entity{
			RegistryID: v.RegistryID,
			DbID:       v.DbID,
		},
		ID: v.ID,
	}

	for result.NextRow() {
		name := NotNilString(result.Data[0])
		val := NotNilString(result.Data[1])
		propType := NotNilString(result.Data[2])
		SetField(v, name, &val, propType)
	}

	return nil
}
