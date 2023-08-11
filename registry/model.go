package registry

import (
	"fmt"
	"strings"

	log "github.com/duglin/dlog"
)

type GroupModel struct {
	SID      string    `json:"-"`
	Registry *Registry `json:"-"`

	Plural   string `json:"plural"`
	Singular string `json:"singular"`
	Schema   string `json:"schema,omitempty"`

	Resources map[string]*ResourceModel `json:"resources,omitempty"` // Plural
}

type ResourceModel struct {
	SID        string      `json:"-"`
	GroupModel *GroupModel `json:"-"`

	Plural    string `json:"plural"`
	Singular  string `json:"singular"`
	Versions  int    `json:"versions"`
	VersionId bool   `json:"versionId"`
	Latest    bool   `json:"latest"`
}

type Model struct {
	Registry *Registry              `json:"-"`
	Schema   string                 `json:"schema,omitempty"`
	Groups   map[string]*GroupModel `json:"groups,omitempty"` // Plural
}

func (m *Model) SetSchema(schema string) error {
	var val any

	// "" -> nil
	if schema != "" {
		val = &schema
	}

	err := DoOne(`
        INSERT INTO Models(RegistrySID, "Schema")
		VALUES(?,?) ON DUPLICATE KEY UPDATE "Schema"=?`,
		m.Registry.DbSID, val, val)
	if err != nil {
		log.Printf("Error updating modelSchema(%s): %s", m.Registry.DbSID, err)
	}

	m.Schema = schema
	return err
}

func (m *Model) AddGroupModel(plural string, singular string, schema string) (*GroupModel, error) {
	if plural == "" {
		return nil, fmt.Errorf("Can't add a GroupModel with an empty plural name")
	}
	if singular == "" {
		return nil, fmt.Errorf("Can't add a GroupModel with an empty sigular name")
	}

	for _, gm := range m.Groups {
		if gm.Plural == plural {
			return nil, fmt.Errorf("GroupModel plural %q already exists",
				plural)
		}
		if gm.Singular == singular {
			return nil, fmt.Errorf("GroupModel singular %q already exists",
				singular)
		}
	}

	mSID := NewUUID()
	err := DoOne(`
        INSERT INTO ModelEntities(
            SID, RegistrySID, ParentSID, Plural, Singular, SchemaURL, Versions)
        VALUES(?,?,?,?,?,?,?) `,
		mSID, m.Registry.DbSID, nil, plural, singular, schema, 0)
	if err != nil {
		log.Printf("Error inserting groupModel(%s): %s", plural, err)
		return nil, err
	}
	gm := &GroupModel{
		SID:      mSID,
		Registry: m.Registry,
		Singular: singular,
		Plural:   plural,
		Schema:   schema,

		Resources: map[string]*ResourceModel{},
	}

	m.Groups[plural] = gm

	return gm, nil
}

func LoadModel(reg *Registry) *Model {
	groups := map[string]*GroupModel{} // Model SID -> *GroupModel

	model := &Model{
		Registry: reg,
		Groups:   map[string]*GroupModel{},
	}

	results, err := Query(`SELECT "Schema" FROM Models WHERE RegistrySID=?`,
		reg.DbSID)

	if err != nil {
		log.Printf("Error loading model(%s): %s", reg.UID, err)
		return nil
	}

	row := results.NextRow()
	if row == nil {
		log.Printf("Error loading model(%s): results are empty", reg.UID)
		return nil
	}
	model.Schema = NotNilString(row[0])
	results.Close()

	results, err = Query(`
        SELECT
            SID, RegistrySID, ParentSID, Plural, Singular, SchemaURL, Versions,
            VersionId, Latest
        FROM ModelEntities
        WHERE RegistrySID=?
        ORDER BY ParentSID ASC`, reg.DbSID)
	defer results.Close()

	if err != nil {
		log.Printf("Error loading model(%s): %s", reg.UID, err)
		return nil
	}

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		if *row[2] == nil { // ParentSID nil -> new Group
			g := &GroupModel{ // Plural
				SID:      NotNilString(row[0]), // SID
				Registry: reg,
				Plural:   NotNilString(row[3]), // Plural
				Singular: NotNilString(row[4]), // Singular
				Schema:   NotNilString(row[5]), // SchemaURL

				Resources: map[string]*ResourceModel{},
			}

			model.Groups[NotNilString(row[3])] = g
			groups[NotNilString(row[0])] = g

		} else { // New Resource
			g := groups[NotNilString(row[2])] // Parent SID

			if g != nil { // should always be true, but...
				r := &ResourceModel{
					SID:        NotNilString(row[0]),
					GroupModel: g,
					Plural:     NotNilString(row[3]),
					Singular:   NotNilString(row[4]),
					Versions:   NotNilInt(row[6]),
					VersionId:  NotNilBool(row[7]),
					Latest:     NotNilBool(row[8]),
				}

				g.Resources[r.Plural] = r
			}
		}
	}

	reg.Model = model
	return model
}

func (m *Model) FindGroupModel(gTypePlural string) *GroupModel {
	for _, gModel := range m.Groups {
		if strings.EqualFold(gModel.Plural, gTypePlural) {
			return gModel
		}
	}
	return nil
}

func (m *Model) ApplyNewModel(newM *Model) error {
	if newM.Schema != m.Schema {
		m.SetSchema(newM.Schema)
	}

	// Find all old groups that need to be deleted
	for gmPlural, gm := range m.Groups {
		if newGM, ok := newM.Groups[gmPlural]; !ok {
			if err := gm.Delete(); err != nil {
				return err
			}
		} else {
			for rmPlural, rm := range gm.Resources {
				if _, ok := newGM.Resources[rmPlural]; !ok {
					if err := rm.Delete(); err != nil {
						return err
					}
				}
			}
		}
	}

	// Apply new stuff
	var err error
	newM.Registry = m.Registry
	for _, newGM := range newM.Groups {
		newGM.Registry = m.Registry
		oldGM := m.Groups[newGM.Plural]
		if oldGM == nil {
			oldGM, err = m.AddGroupModel(newGM.Plural, newGM.Singular,
				newGM.Schema)
			if err != nil {
				return err
			}
		} else {
			oldGM.Singular = newGM.Singular
			oldGM.Schema = newGM.Schema
			if err = oldGM.Save(); err != nil {
				return err
			}
		}

		for _, newRM := range newGM.Resources {
			oldRM := oldGM.Resources[newRM.Plural]
			if oldRM == nil {
				oldRM, err = oldGM.AddResourceModel(newRM.Plural,
					newRM.Singular, newRM.Versions, newRM.VersionId,
					newRM.Latest)
			} else {
				oldRM.Singular = newRM.Singular
				oldRM.Versions = newRM.Versions
				oldRM.VersionId = newRM.VersionId
				oldRM.Latest = newRM.Latest
				if err = oldRM.Save(); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (gm *GroupModel) Delete() error {
	log.VPrintf(3, ">Enter: Delete.GroupModel: %s", gm.Plural)
	defer log.VPrintf(3, "<Exit: Delete.GroupModel")
	err := DoOne(`
        DELETE FROM ModelEntities
		WHERE RegistrySID=? AND SID=?`, // SID should be enough, but ok
		gm.Registry.DbSID, gm.SID)
	if err != nil {
		log.Printf("Error deleting groupModel(%s): %s", gm.Plural, err)
		return err
	}

	delete(gm.Registry.Model.Groups, gm.Plural)

	return nil
}

func (gm *GroupModel) Save() error {
	// Just updates this GroupModel, not any Resources
	// DO NOT use this to insert a new one
	err := DoOne(`
        INSERT INTO ModelEntities(
            SID, RegistrySID,
			ParentSID, Plural, Singular, SchemaURL)
        VALUES(?,?,?,?,?,?)
        ON DUPLICATE KEY UPDATE
		    ParentSID=?,Plural=?,Singular=?,SchemaURL=?
		`,
		gm.SID, gm.Registry.DbSID,
		nil, gm.Plural, gm.Singular, gm.Schema,

		nil, gm.Plural, gm.Singular, gm.Schema)
	if err != nil {
		log.Printf("Error updating groupModel(%s): %s", gm.Plural, err)
	}
	return err
}

func (gm *GroupModel) AddResourceModel(plural string, singular string, versions int, verId bool, latest bool) (*ResourceModel, error) {
	if plural == "" {
		return nil, fmt.Errorf("Can't add a group with an empty plural name")
	}
	if singular == "" {
		return nil, fmt.Errorf("Can't add a group with an empty sigular name")
	}
	if versions < 0 {
		return nil, fmt.Errorf("'versions'(%d) must be >= 0", versions)
	}

	for _, r := range gm.Resources {
		if r.Plural == plural {
			return nil, fmt.Errorf("Resoucre model plural %q already "+
				"exists for group %q", plural, gm.Plural)
		}
		if r.Singular == singular {
			return nil,
				fmt.Errorf("Resoucre model singular %q already "+
					"exists for group %q", singular, gm.Plural)
		}
	}

	mSID := NewUUID()

	err := DoOne(`
		INSERT INTO ModelEntities(
			SID, RegistrySID, ParentSID, Plural, Singular, SchemaURL, Versions,
			VersionId, Latest)
		VALUES(?,?,?,?,?,?,?,?,?) `,
		mSID, gm.Registry.DbSID, gm.SID, plural, singular, nil, versions,
		verId, latest)
	if err != nil {
		log.Printf("Error inserting resourceModel(%s): %s", plural, err)
		return nil, err
	}
	r := &ResourceModel{
		SID:        mSID,
		GroupModel: gm,
		Singular:   singular,
		Plural:     plural,
		Versions:   versions,
		VersionId:  verId,
		Latest:     latest,
	}

	gm.Resources[plural] = r

	return r, nil
}

func (rm *ResourceModel) Delete() error {
	log.VPrintf(3, ">Enter: Delete.ResourceModel: %s", rm.Plural)
	defer log.VPrintf(3, "<Exit: Delete.ResourceModel")
	err := DoOne(`
        DELETE FROM ModelEntities
		WHERE RegistrySID=? AND SID=?`, // SID should be enough, but ok
		rm.GroupModel.Registry.DbSID, rm.SID)
	if err != nil {
		log.Printf("Error deleting resourceModel(%s): %s", rm.Plural, err)
		return err
	}

	delete(rm.GroupModel.Resources, rm.Plural)

	return nil
}

func (rm *ResourceModel) Save() error {
	// Just updates this GroupModel, not any Resources
	// DO NOT use this to insert a new one
	err := DoOne(`
        INSERT INTO ModelEntities(
            SID, RegistrySID,
			ParentSID, Plural, Singular, SchemaURL, Versions,
			VersionId, Latest)
        VALUES(?,?,?,?,?,?,?,?,?)
        ON DUPLICATE KEY UPDATE
            ParentSID=?, Plural=?, Singular=?, SchemaURL=?,
            Versions=?, VersionId=?, Latest=?`,
		rm.SID, rm.GroupModel.Registry.DbSID,
		rm.GroupModel.SID, rm.Plural, rm.Singular, nil, rm.Versions,
		rm.VersionId, rm.Latest,

		rm.GroupModel.SID, rm.Plural, rm.Singular, nil, rm.Versions,
		rm.VersionId, rm.Latest)
	if err != nil {
		log.Printf("Error updating resourceModel(%s): %s", rm.Plural, err)
		return err
	}
	return err
}
