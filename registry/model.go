package registry

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/duglin/dlog"
)

type Model struct {
	Registry   *Registry              `json:"-"`
	Schemas    []string               `json:"schemas,omitempty"`
	Attributes map[string]*Attribute  `json:"attributes,omitempty"` // attrName
	Groups     map[string]*GroupModel `json:"groups,omitempty"`     // Plural
}

type Attribute struct {
	Name        string   `json:"name,omitempty"`
	Type        string   `json:"type,omitempty"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Strict      bool     `json:"strict,omitempty"`
	Required    bool     `json:"required,omitempty"`

	Item    *Item               `json:"item,omitempty"`
	IfValue map[string]*IfValue `json:"ifValue,omitempty"` // Value
}

type Item struct {
	Attributes map[string]*Attribute `json:"attributes,omitempty"` //attrName
	Type       string                `json:"type,omitempty"`
	Item       *Item                 `json:"item,omitempty"`
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

func (m *Model) Save() error {
	buf, _ := json.Marshal(m.Attributes)
	attrs := string(buf)

	err := DoZeroOne(`UPDATE Registries SET Attributes=? WHERE SID=?`,
		attrs, m.Registry.DbSID)
	if err != nil {
		log.Printf("Error updating model: %s", err)
	}

	for _, gm := range m.Groups {
		gm.Save()
	}
	return err
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

func (m *Model) AddAttr(name, daType string) *Attribute {
	return m.AddAttribute(&Attribute{
		Name: name,
		Type: daType,
	})
}

func (m *Model) AddAttrMap(name string, itemType string) *Attribute {
	return m.AddAttribute(&Attribute{
		Name: name,
		Type: MAP,
		Item: &Item{
			Type: itemType,
		},
	})
}

func (m *Model) AddAttribute(attr *Attribute) *Attribute {
	if attr == nil {
		return nil
	}

	if m.Attributes == nil {
		m.Attributes = map[string]*Attribute{}
	}

	m.Attributes[attr.Name] = attr
	m.Save()
	return attr
}

func (m *Model) DelAttribute(name string) {
	if m.Attributes == nil {
		return
	}

	delete(m.Attributes, name)
	m.Save()
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

	// Load Registry Attributes
	results, err := Query(`SELECT Attributes FROM Registries WHERE SID=?`,
		reg.DbSID)
	defer results.Close()
	if err != nil {
		log.Printf("Error loading registries(%s): %s", reg.UID, err)
		return nil
	}
	row := results.NextRow()
	if row == nil {
		log.Printf("Can't find registry: %s", reg.UID)
		return nil
	}

	if row[0] != nil {
		json.Unmarshal([]byte(NotNilString(row[0])), &model.Attributes)

	}

	// Load Schemas
	results, err = Query(`
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
            SID, RegistrySID, ParentSID, Plural, Singular, Attributes,
			Versions, VersionId, Latest, HasDocument
        FROM ModelEntities
        WHERE RegistrySID=?
        ORDER BY ParentSID ASC`, reg.DbSID)
	defer results.Close()

	if err != nil {
		log.Printf("Error loading model(%s): %s", reg.UID, err)
		return nil
	}

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		attrs := (map[string]*Attribute)(nil)
		if row[5] != nil {
			json.Unmarshal([]byte(NotNilString(row[5])), &attrs)
		}

		if *row[2] == nil { // ParentSID nil -> new Group
			g := &GroupModel{ // Plural
				SID:        NotNilString(row[0]), // SID
				Registry:   reg,
				Plural:     NotNilString(row[3]), // Plural
				Singular:   NotNilString(row[4]), // Singular
				Attributes: attrs,

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
					Attributes:  attrs,
					Versions:    NotNilIntDef(row[6], VERSIONS),
					VersionId:   NotNilBoolDef(row[7], VERSIONID),
					Latest:      NotNilBoolDef(row[8], LATEST),
					HasDocument: NotNilBoolDef(row[9], HASDOCUMENT),
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

	buf, _ := json.Marshal(gm.Attributes)
	attrs := string(buf)

	err := DoZeroTwo(`
        INSERT INTO ModelEntities(
            SID, RegistrySID,
			ParentSID, Plural, Singular, Attributes)
        VALUES(?,?,?,?,?,?)
        ON DUPLICATE KEY UPDATE
		    ParentSID=?,Plural=?,Singular=?,Attributes=?
		`,
		gm.SID, gm.Registry.DbSID,
		nil, gm.Plural, gm.Singular, attrs,
		nil, gm.Plural, gm.Singular, attrs)
	if err != nil {
		log.Printf("Error updating groupModel(%s): %s", gm.Plural, err)
	}

	for _, rm := range gm.Resources {
		rm.Save()
	}

	return err
}

func (gm *GroupModel) AddAttr(name, daType string) *Attribute {
	return gm.AddAttribute(&Attribute{
		Name: name,
		Type: daType,
	})
}

func (gm *GroupModel) AddAttrMap(name string, itemType string) *Attribute {
	return gm.AddAttribute(&Attribute{
		Name: name,
		Type: MAP,
		Item: &Item{
			Type: itemType,
		},
	})
}

func (gm *GroupModel) AddAttribute(attr *Attribute) *Attribute {
	if attr == nil {
		return nil
	}

	if gm.Attributes == nil {
		gm.Attributes = map[string]*Attribute{}
	}

	gm.Attributes[attr.Name] = attr
	gm.Save()
	return attr
}

func (gm *GroupModel) DelAttribute(name string) {
	if gm.Attributes == nil {
		return
	}

	delete(gm.Attributes, name)
	gm.Save()
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

func (rm *ResourceModel) AddAttr(name, daType string) *Attribute {
	return rm.AddAttribute(&Attribute{
		Name: name,
		Type: daType,
	})
}

func (rm *ResourceModel) AddAttrMap(name string, itemType string) *Attribute {
	return rm.AddAttribute(&Attribute{
		Name: name,
		Type: MAP,
		Item: &Item{
			Type: itemType,
		},
	})
}

func (rm *ResourceModel) AddAttribute(attr *Attribute) *Attribute {
	if attr == nil {
		return nil
	}

	if rm.Attributes == nil {
		rm.Attributes = map[string]*Attribute{}
	}

	rm.Attributes[attr.Name] = attr
	rm.Save()
	return attr
}

func (rm *ResourceModel) DelAttribute(name string) {
	if rm.Attributes == nil {
		return
	}

	delete(rm.Attributes, name)
	rm.Save()
}

func GetAttributes(rSID, abstractEntity string) map[string]*Attribute {
	reg, err := FindRegistryBySID(rSID)
	if reg == nil {
		log.Fatalf("Can't find registry(%s): %s", rSID, err)
	}

	var attrs map[string]*Attribute
	level := '0'

	paths := strings.Split(abstractEntity, string(DB_IN))
	if len(paths) == 0 || paths[0] == "" {
		attrs = reg.Model.Attributes
	} else {
		level = rune('0' + len(paths))
		gm := reg.Model.Groups[paths[0]]
		if gm == nil {
			panic(fmt.Sprintf("Can't find Group %q", paths[0]))
		}

		if len(paths) == 1 {
			attrs = gm.Attributes
		} else {
			rm := gm.Resources[paths[1]]
			if rm == nil {
				panic(fmt.Sprintf("Can't find Resource %q", paths[1]))
			}
			attrs = rm.Attributes
		}
	}

	// Now copy and add the xReg defined attributes
	res := map[string]*Attribute{}
	for key, value := range attrs {
		res[key] = value
	}

	for _, specProp := range OrderedSpecProps {
		if specProp.levels == "" || strings.ContainsRune(specProp.levels, level) {
			if specProp.modelAttribute != nil {
				res[specProp.name] = specProp.modelAttribute
			}
		}
	}

	return res
}

func IsScalar(daType string) bool {
	return daType == BOOLEAN || daType == DECIMAL || daType == INTEGER ||
		daType == STRING || daType == TIME || daType == UINTEGER ||
		daType == URI || daType == URI_REFERENCE || daType == URI_TEMPLATE ||
		daType == URL
}

func (a *Attribute) IsScalar() bool {
	return IsScalar(a.Type)
}

func (a *Attribute) AddAttr(name, daType string) *Attribute {
	return a.AddAttribute(&Attribute{
		Name: name,
		Type: daType,
	})
}

func (a *Attribute) AddAttribute(attr *Attribute) *Attribute {
	a.Item.Attributes[attr.Name] = attr
	return attr
}

func GetAttributeType(rSID, abstractEntity string, pp *PropPath) (string, error) {
	if pp.Len() == 0 {
		panic("PropPath can't be empty for GetAttributeType")
	}

	if pp.Top()[0] == '#' {
		return ANY, nil
	}

	savePP := pp.Clone()

	attrs := GetAttributes(rSID, abstractEntity)
	if attrs == nil {
		panic("Attributes can't be nil for: %s" + abstractEntity)
	}

	for {
		// We're on an object
		if pp.Len() == 0 {
			return OBJECT, nil
		}
		top := pp.Top()
		attr := attrs[top]

		if attr == nil {
			attr = attrs["*"]
			if attr == nil {
				return "", nil
			}
		}

		// Just a scalar, return it
		if attr.IsScalar() {
			if pp.Len() != 1 {
				panic(fmt.Sprintf("Trying to traverse into scalar %q: %s",
					attr.Name, top))
			}
			return attr.Type, nil
		}

		if attr.Type == ANY {
			return attr.Type, nil
		}

		// Is non-scalar (obj, map, array)
		if attr.Type == OBJECT || attr.Type == "" {
			attrs = attr.Item.Attributes
			attr = nil // let loop grab next one
			pp = pp.Next()
			continue
		}

		if attr.Type == ARRAY {
			saveTop := pp.Top()
			for {
				pp = pp.Next() // Next is the index of the array
				if pp.Len() == 0 {
					return ARRAY, nil // No index so just return the array itself
				}
				if pp.Parts[0].Index < 0 {
					return "", fmt.Errorf("Array index for %q isn't an integer: %v",
						saveTop, pp.Top())
				}
				if attr.Item.Type != ARRAY {
					break
				}
				attr = &Attribute{
					Type: attr.Item.Type,
					Item: attr.Item.Item,
				}
			}

			if IsScalar(attr.Item.Type) {
				if pp.Len() != 1 {
					return "", fmt.Errorf("Traversing into scalar "+
						"\"%s[%d]\": %s", saveTop, pp.Parts[0].Index,
						savePP.UI())
				}
				return attr.Item.Type, nil
			}
			// Skip key
			pp = pp.Next()
			if attr.Item.Type == OBJECT || attr.Type == "" {
				attrs = attr.Item.Attributes
			}
			continue
		}

		if attr.Type == MAP {
			pp = pp.Next() // Next is the key, if present
			if pp.Len() == 0 {
				return MAP, nil // No key so just return the map itself
			}
			if IsScalar(attr.Item.Type) {
				if pp.Len() != 1 {
					return "", fmt.Errorf("Traversing into scalar "+
						"%q: %s", pp.Top(), savePP.UI())
				}
				return attr.Item.Type, nil
			}
			// Skip key
			pp = pp.Next()
			if attr.Item.Type == OBJECT || attr.Item.Type == "" {
				attrs = attr.Item.Attributes
			}
			attr = nil // let loop grab the next one
			continue
		}

		panic(fmt.Sprintf("Can't deal with type: %s\npp:%v", attr.Type, savePP.UI()))
	}
}

func unused__GetAttributeValue(rsid, abstractEntity, name string) any {
	if name == "" {
		panic("Name can't be empty for GetAttributeValue")
	}

	attrs := GetAttributes(rsid, abstractEntity)
	if attrs == nil {
		panic("Attributes can't be nil for: %s" + abstractEntity)
	}

	parts := strings.Split(name, string(DB_IN))
	for {
		// We're on an object
		if len(parts) == 0 || parts[0] == "" {
			return nil
		}
		attr := attrs[parts[0]]
		if attr == nil {
			return ""
		}

		// Just a scalar, return it
		if attr.IsScalar() {
			if len(parts) != 1 {
				panic(fmt.Sprintf("Trying to traverse into scalar %q: %s",
					attr.Name, name))
			}
			return attr.Type
		}

		// Is non-scalar (obj, map, array)
		if attr.Type == OBJECT || attr.Type == "" {
			attrs = attr.Item.Attributes
			parts = parts[1:]
			continue
		}

		if attr.Type == ARRAY {
		}

		if attr.Type == MAP {
			parts = parts[1:] // Now parts[0] == key, if present
			if len(parts) == 0 {
				return MAP // No key so just return the map itself
			}
			if IsScalar(attr.Item.Type) {
				if len(parts) != 1 {
					panic(fmt.Sprintf("Traversing into scalar "+
						"%q: %s", parts[0], name))
				}
				return attr.Item.Type
			}
			// Skip key
			parts = parts[1:]
			if attr.Item.Type == OBJECT || attr.Item.Type == "" {
				attrs = attr.Item.Attributes
				continue
			}

		}

		panic(fmt.Sprintf("Can't deal with type: %s", attr.Type))
	}
}
