package registry

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/duglin/dlog"
)

const VERSIONS = 1
const VERSIONID = true
const LATEST = true
const HASDOCUMENT = true

type Model struct {
	Registry   *Registry              `json:"-"`
	Schemas    []string               `json:"schemas,omitempty"`
	Attributes map[string]*Attribute  `json:"attributes,omitempty"` // attrName
	Groups     map[string]*GroupModel `json:"groups,omitempty"`     // Plural
}

type Attribute struct {
	Name        string                `json:"name,omitempty"`
	Type        string                `json:"type,omitempty"`
	Description string                `json:"description,omitempty"`
	Enum        []string              `json:"enum,omitempty"`
	Strict      bool                  `json:"strict,omitempty"`
	Required    bool                  `json:"required,omitempty"`
	Attributes  map[string]*Attribute `json:"attributes,omitempty"`
	KeyType     string                `json:"keyType,omitempty"`
	ItemType    string                `json:"itemType,omitempty"`
	IfValue     map[string]*IfValue   `json:"ifValue,omitempty"` // Value
}

type IfValue struct {
	SiblingAttributes map[string]*Attribute `json:"siblingAttributes,omitempty"`
}

type GroupModel struct {
	SID      string    `json:"-"`
	Registry *Registry `json:"-"`

	Plural     string                `json:"plural"`
	Singular   string                `json:"singular"`
	Attributes map[string]*Attribute `json:"attributes,omitempty"`

	Resources map[string]*ResourceModel `json:"resources,omitempty"` // Plural
}

type ResourceModel struct {
	SID        string      `json:"-"`
	GroupModel *GroupModel `json:"-"`

	Plural      string                `json:"plural"`
	Singular    string                `json:"singular"`
	Versions    int                   `json:"versions"`
	VersionId   bool                  `json:"versionId"`
	Latest      bool                  `json:"latest"`
	HasDocument bool                  `json:"hasDocument"`
	Attributes  map[string]*Attribute `json:"attributes,omitempty"`
}

func (r *ResourceModel) UnmarshalJSON(data []byte) error {
	// Set the default values
	r.Versions = VERSIONS
	r.VersionId = VERSIONID
	r.Latest = LATEST
	r.HasDocument = HASDOCUMENT

	type tmpResourceModel ResourceModel
	return json.Unmarshal(data, (*tmpResourceModel)(r))
}

func (m *Model) AddSchema(schema string) error {
	err := Do(`INSERT INTO "Schemas" (RegistrySID, "Schema") VALUES(?,?)`,
		m.Registry.DbSID, schema)
	if err != nil {
		err = fmt.Errorf("Error inserting schema(%s): %s", schema, err)
		log.Print(err)
		return err
	}

	for _, s := range m.Schemas {
		if s == schema {
			// already there
			return nil
		}
	}

	m.Schemas = append(m.Schemas, schema)
	return nil
}

func (m *Model) DelSchema(schema string) error {
	err := Do(`DELETE FROM "Schemas" WHERE RegistrySID=? AND "Schema"=?`,
		m.Registry.DbSID, schema)
	if err != nil {
		err = fmt.Errorf("Error deleting schema(%s): %s", schema, err)
		log.Print(err)
		return err
	}

	for i, s := range m.Schemas {
		if s == schema {
			m.Schemas = append(m.Schemas[:i], m.Schemas[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *Model) SetSchemas(schemas []string) error {
	err := Do(`DELETE FROM "Schemas" WHERE RegistrySID=?`, m.Registry.DbSID)
	if err != nil {
		err = fmt.Errorf("Error deleting schemas: %s", err)
		log.Print(err)
		return err
	}
	m.Schemas = nil

	for _, s := range schemas {
		err = m.AddSchema(s)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Model) AddGroupModel(plural string, singular string) (*GroupModel, error) {
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
            SID, RegistrySID, ParentSID, Plural, Singular, Versions)
        VALUES(?,?,?,?,?,?) `,
		mSID, m.Registry.DbSID, nil, plural, singular, 0)
	if err != nil {
		log.Printf("Error inserting groupModel(%s): %s", plural, err)
		return nil, err
	}
	gm := &GroupModel{
		SID:      mSID,
		Registry: m.Registry,
		Singular: singular,
		Plural:   plural,

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

	// Load Schemas
	results, err := Query(`
        SELECT RegistrySID, "Schema" FROM "Schemas"
        WHERE RegistrySID=?
        ORDER BY "Schema" ASC`, reg.DbSID)
	defer results.Close()

	if err != nil {
		log.Printf("Error loading schemas(%s): %s", reg.UID, err)
		return nil
	}

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		model.Schemas = append(model.Schemas, NotNilString(row[1]))
	}

	// Load Groups & Resources
	results, err = Query(`
        SELECT
            SID, RegistrySID, ParentSID, Plural, Singular, Versions,
            VersionId, Latest, HasDocument
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

				Resources: map[string]*ResourceModel{},
			}

			model.Groups[NotNilString(row[3])] = g
			groups[NotNilString(row[0])] = g

		} else { // New Resource
			g := groups[NotNilString(row[2])] // Parent SID

			if g != nil { // should always be true, but...
				r := &ResourceModel{
					SID:         NotNilString(row[0]),
					GroupModel:  g,
					Plural:      NotNilString(row[3]),
					Singular:    NotNilString(row[4]),
					Versions:    NotNilIntDef(row[5], VERSIONS),
					VersionId:   NotNilBoolDef(row[6], VERSIONID),
					Latest:      NotNilBoolDef(row[7], LATEST),
					HasDocument: NotNilBoolDef(row[8], HASDOCUMENT),
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
	// Delete old Schemas, then add new ones
	err := Do(`DELETE FROM "Schemas" WHERE RegistrySID=?`, m.Registry.DbSID)
	if err != nil {
		return err
	}
	for _, schema := range newM.Schemas {
		if err = m.AddSchema(schema); err != nil {
			return err
		}
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
	newM.Registry = m.Registry
	for _, newGM := range newM.Groups {
		newGM.Registry = m.Registry
		oldGM := m.Groups[newGM.Plural]
		if oldGM == nil {
			oldGM, err = m.AddGroupModel(newGM.Plural, newGM.Singular)
			if err != nil {
				return err
			}
		} else {
			oldGM.Singular = newGM.Singular
			if err = oldGM.Save(); err != nil {
				return err
			}
		}

		for _, newRM := range newGM.Resources {
			oldRM := oldGM.Resources[newRM.Plural]
			if oldRM == nil {
				oldRM, err = oldGM.AddResourceModel(newRM.Plural,
					newRM.Singular, newRM.Versions, newRM.VersionId,
					newRM.Latest, newRM.HasDocument)
			} else {
				oldRM.Singular = newRM.Singular
				oldRM.Versions = newRM.Versions
				oldRM.VersionId = newRM.VersionId
				oldRM.Latest = newRM.Latest
				oldRM.HasDocument = newRM.HasDocument
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
	err := DoZeroTwo(`
        INSERT INTO ModelEntities(
            SID, RegistrySID,
			ParentSID, Plural, Singular)
        VALUES(?,?,?,?,?)
        ON DUPLICATE KEY UPDATE
		    ParentSID=?,Plural=?,Singular=?
		`,
		gm.SID, gm.Registry.DbSID,
		nil, gm.Plural, gm.Singular,
		nil, gm.Plural, gm.Singular)
	if err != nil {
		log.Printf("Error updating groupModel(%s): %s", gm.Plural, err)
	}
	return err
}

func (gm *GroupModel) AddResourceModel(plural string, singular string, versions int, verId bool, latest bool, hasDocument bool) (*ResourceModel, error) {
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
			SID, RegistrySID, ParentSID, Plural, Singular, Versions,
			VersionId, Latest, HasDocument)
		VALUES(?,?,?,?,?,?,?,?,?)`,
		mSID, gm.Registry.DbSID, gm.SID, plural, singular, versions,
		verId, latest, hasDocument)
	if err != nil {
		log.Printf("Error inserting resourceModel(%s): %s", plural, err)
		return nil, err
	}
	r := &ResourceModel{
		SID:         mSID,
		GroupModel:  gm,
		Singular:    singular,
		Plural:      plural,
		Versions:    versions,
		VersionId:   verId,
		Latest:      latest,
		HasDocument: hasDocument,
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
	err := DoZeroTwo(`
        INSERT INTO ModelEntities(
            SID, RegistrySID,
			ParentSID, Plural, Singular, Versions,
			VersionId, Latest, HasDocument)
        VALUES(?,?,?,?,?,?,?,?,?)
        ON DUPLICATE KEY UPDATE
            ParentSID=?, Plural=?, Singular=?,
            Versions=?, VersionId=?, Latest=?, HasDocument=?`,
		rm.SID, rm.GroupModel.Registry.DbSID,
		rm.GroupModel.SID, rm.Plural, rm.Singular, rm.Versions,
		rm.VersionId, rm.Latest, rm.HasDocument,

		rm.GroupModel.SID, rm.Plural, rm.Singular, rm.Versions,
		rm.VersionId, rm.Latest, rm.HasDocument)
	if err != nil {
		log.Printf("Error updating resourceModel(%s): %s", rm.Plural, err)
		return err
	}
	return err
}
