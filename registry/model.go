package registry

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"strings"
	"time"

	log "github.com/duglin/dlog"
)

var RegexpPropName = regexp.MustCompile("^[a-z_][a-z0-9_./]{0,62}$")
var RegexpMapKey = regexp.MustCompile("^[a-z0-9][a-z0-9_.\\-]{0,62}$")

func IsValidAttributeName(name string) bool {
	return RegexpPropName.MatchString(name)
}

func IsValidMapKey(key string) bool {
	return RegexpMapKey.MatchString(key)
}

type Model struct {
	Registry   *Registry              `json:"-"`
	Schemas    []string               `json:"schemas,omitempty"`
	Attributes Attributes             `json:"attributes,omitempty"`
	Groups     map[string]*GroupModel `json:"groups,omitempty"` // Plural
}

type Attributes map[string]*Attribute // AttrName->Attr

type Attribute struct {
	Registry    *Registry `json:"-"`
	Name        string    `json:"name,omitempty"`
	Type        string    `json:"type,omitempty"`
	Description string    `json:"description,omitempty"`
	Enum        []string  `json:"enum,omitempty"`
	Strict      bool      `json:"strict,omitempty"`
	Required    bool      `json:"required,omitempty"`
	ReadOnly    bool      `json:"readonly,omitempty"`

	Item    *Item               `json:"item,omitempty"`
	IfValue map[string]*IfValue `json:"ifValue,omitempty"` // Value

	// Internal fields
	checkFn  func(newObj map[string]any, oldObj map[string]any) error
	updateFn func(*UpdateFnArgs) error
}

type Item struct {
	Registry   *Registry  `json:"-"`
	Attributes Attributes `json:"attributes,omitempty"` //attrName
	Type       string     `json:"type,omitempty"`
	Item       *Item      `json:"item,omitempty"`
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
	VersionId   bool                  `json:"versionid"`
	Latest      bool                  `json:"latest"`
	HasDocument bool                  `json:"hasdocument"`
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

// Save() should be called by these funcs automatically but there may be
// cases where someone would need to call it manually (e.g. setting an
// attribute's property - we should technically find a way to catch those
// cases so code above this shouldn't need to think about it
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

func (m *Model) AddAttr(name, daType string) (*Attribute, error) {
	return m.AddAttribute(&Attribute{Name: name, Type: daType})
}

func (m *Model) AddAttrMap(name string, item *Item) (*Attribute, error) {
	return m.AddAttribute(&Attribute{Name: name, Type: MAP, Item: item})
}

func (m *Model) AddAttrObj(name string) (*Attribute, error) {
	return m.AddAttribute(&Attribute{Name: name, Type: OBJECT, Item: &Item{}})
}

func (m *Model) AddAttrArray(name string, item *Item) (*Attribute, error) {
	return m.AddAttribute(&Attribute{Name: name, Type: ARRAY, Item: item})
}

func (m *Model) AddAttribute(attr *Attribute) (*Attribute, error) {
	if attr == nil {
		return nil, nil
	}

	if attr.Name != "*" && !IsValidAttributeName(attr.Name) {
		return nil, fmt.Errorf("Invalid attribute name: %s", attr.Name)
	}

	if m.Attributes == nil {
		m.Attributes = map[string]*Attribute{}
	}

	m.Attributes[attr.Name] = attr

	if attr.Registry == nil {
		attr.Registry = m.Registry
	}

	attr.Item.SetRegistry(m.Registry)

	m.Save()

	return attr, nil
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

	if !IsValidAttributeName(plural) {
		return nil, fmt.Errorf("GroupModel plural name is not valid")
	}

	if !IsValidAttributeName(singular) {
		return nil, fmt.Errorf("GroupModel singular name is not valid")
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

func NewItem(daType string) *Item {
	return &Item{
		Type: daType,
	}
}

func NewItemObject() *Item {
	return &Item{
		Type: OBJECT,
	}
}

func NewItemMap(item *Item) *Item {
	return &Item{
		Type: MAP,
		Item: item,
	}
}

func NewItemArray(item *Item) *Item {
	return &Item{
		Type: ARRAY,
		Item: item,
	}
}

func (i *Item) SetRegistry(reg *Registry) {
	if i == nil {
		return
	}

	i.Registry = reg
	i.Attributes.SetRegistry(reg)
}

func (i *Item) AddAttr(name, daType string) (*Attribute, error) {
	return i.AddAttribute(&Attribute{Name: name, Type: daType})
}

func (i *Item) AddAttrMap(name string, item *Item) (*Attribute, error) {
	return i.AddAttribute(&Attribute{Name: name, Type: MAP, Item: item})
}

func (i *Item) AddAttrObj(name string) (*Attribute, error) {
	return i.AddAttribute(&Attribute{Name: name, Type: OBJECT, Item: &Item{}})
}

func (i *Item) AddAttrArray(name string, item *Item) (*Attribute, error) {
	return i.AddAttribute(&Attribute{Name: name, Type: ARRAY, Item: item})
}

func (i *Item) AddAttribute(attr *Attribute) (*Attribute, error) {
	if attr == nil {
		return nil, nil
	}

	if attr.Name != "*" && !IsValidAttributeName(attr.Name) {
		return nil, fmt.Errorf("Invalid attribute name: %s", attr.Name)
	}

	if i.Attributes == nil {
		i.Attributes = map[string]*Attribute{}
	}

	i.Attributes[attr.Name] = attr

	if attr.Registry == nil {
		attr.Registry = i.Registry
	}
	if i.Registry != nil {
		i.Registry.Model.Save()
	}

	return attr, nil
}

func (i *Item) DelAttribute(name string) {
	if i.Attributes == nil {
		return
	}

	delete(i.Attributes, name)

	if i.Registry != nil {
		i.Registry.Model.Save()
	}
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

	model.Attributes.SetRegistry(reg)

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

func (gm *GroupModel) AddAttr(name, daType string) (*Attribute, error) {
	return gm.AddAttribute(&Attribute{Name: name, Type: daType})
}

func (gm *GroupModel) AddAttrMap(name string, item *Item) (*Attribute, error) {
	return gm.AddAttribute(&Attribute{Name: name, Type: MAP, Item: item})
}

func (gm *GroupModel) AddAttrObj(name string) (*Attribute, error) {
	return gm.AddAttribute(&Attribute{Name: name, Type: OBJECT, Item: &Item{}})
}

func (gm *GroupModel) AddAttrArray(name string, item *Item) (*Attribute, error) {
	return gm.AddAttribute(&Attribute{Name: name, Type: ARRAY, Item: item})
}

func (gm *GroupModel) AddAttribute(attr *Attribute) (*Attribute, error) {
	if attr == nil {
		return nil, nil
	}

	if attr.Name != "*" && !IsValidAttributeName(attr.Name) {
		return nil, fmt.Errorf("Invalid attribute name: %s", attr.Name)
	}

	if gm.Attributes == nil {
		gm.Attributes = map[string]*Attribute{}
	}

	gm.Attributes[attr.Name] = attr

	if attr.Registry == nil {
		attr.Registry = gm.Registry
	}
	gm.Registry.Model.Save()

	return attr, nil
}

func (gm *GroupModel) DelAttribute(name string) {
	if gm.Attributes == nil {
		return
	}

	delete(gm.Attributes, name)

	gm.Registry.Model.Save()
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
	if !IsValidAttributeName(plural) {
		return nil, fmt.Errorf("ResourceModel plural name is not valid")
	}
	if !IsValidAttributeName(singular) {
		return nil, fmt.Errorf("ResourceModel singular name is not valid")
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

func (rm *ResourceModel) AddAttr(name, daType string) (*Attribute, error) {
	return rm.AddAttribute(&Attribute{Name: name, Type: daType})
}

func (rm *ResourceModel) AddAttrMap(name string, item *Item) (*Attribute, error) {
	return rm.AddAttribute(&Attribute{Name: name, Type: MAP, Item: item})
}

func (rm *ResourceModel) AddAttrObj(name string) (*Attribute, error) {
	return rm.AddAttribute(&Attribute{Name: name, Type: OBJECT, Item: &Item{}})
}

func (rm *ResourceModel) AddAttrArray(name string, item *Item) (*Attribute, error) {
	return rm.AddAttribute(&Attribute{Name: name, Type: ARRAY, Item: item})
}

func (rm *ResourceModel) AddAttribute(attr *Attribute) (*Attribute, error) {
	if attr == nil {
		return nil, nil
	}

	if attr.Name != "*" && !IsValidAttributeName(attr.Name) {
		return nil, fmt.Errorf("Invalid attribute name: %s", attr.Name)
	}

	if rm.Attributes == nil {
		rm.Attributes = map[string]*Attribute{}
	}

	rm.Attributes[attr.Name] = attr

	if attr.Registry == nil {
		attr.Registry = rm.GroupModel.Registry
	}
	rm.GroupModel.Registry.Model.Save()

	return attr, nil
}

func (rm *ResourceModel) DelAttribute(name string) {
	if rm.Attributes == nil {
		return
	}

	delete(rm.Attributes, name)

	rm.GroupModel.Registry.Model.Save()
}

func (attrs *Attributes) SetRegistry(reg *Registry) {
	if attrs == nil {
		return
	}

	for _, attr := range *attrs {
		attr.Registry = reg
		attr.Item.SetRegistry(reg)
	}
}

func GetCollections(rSID, abstractEntity string) []string {
	reg, err := FindRegistryBySID(rSID)
	if reg == nil {
		log.Fatalf("Can't find registry(%s): %s", rSID, err)
	}

	paths := strings.Split(abstractEntity, string(DB_IN))
	if len(paths) == 0 || paths[0] == "" {
		return SortedKeys(reg.Model.Groups)
	} else {
		gm := reg.Model.Groups[paths[0]]
		if gm == nil {
			panic(fmt.Sprintf("Can't find Group %q", paths[0]))
		}

		if len(paths) == 1 {
			return SortedKeys(gm.Resources)
		} else if len(paths) == 2 {
			return []string{"versions"}
		}
	}

	return nil
}

func GetAttributes(rSID, abstractEntity string) map[string]*Attribute {
	reg, err := FindRegistryBySID(rSID)
	if reg == nil {
		log.Fatalf("Can't find registry(%s): %s", rSID, err)
	}

	var attrs map[string]*Attribute
	level := '0'
	singular := ""

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
			singular = rm.Singular
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

	if singular != "" {
		checkFn := func(newObj map[string]any, oldObj map[string]any) error {
			list := []string{singular, singular + "url", singular + "base64"}
			count := 0
			for _, name := range list {
				if v, ok := newObj[name]; ok && !IsNil(v) {
					count++
				}
			}
			if count > 1 {
				return fmt.Errorf("Only one of %s can be present at a time",
					strings.Join(list, ","))
			}
			return nil
		}

		// Add resource content attributes
		res[singular] = &Attribute{
			Name:    singular,
			Type:    ANY,
			checkFn: checkFn,
			updateFn: func(args *UpdateFnArgs) error {
				v, ok := args.NewObj[singular]
				if ok {
					args.NewObj["#resource"] = v
					// args.NewObj["#resourceURL"] = nil
					delete(args.NewObj, singular)
				}
				return nil
			},
		}
		res[singular+"url"] = &Attribute{
			Name:    singular + "url",
			Type:    URL,
			checkFn: checkFn,
			updateFn: func(args *UpdateFnArgs) error {
				v, ok := args.NewObj[singular+"url"]
				if !ok {
					return nil
				}
				args.NewObj["#resource"] = nil
				args.NewObj["#resourceURL"] = v
				delete(args.NewObj, singular+"url")
				return nil
			},
		}
		res[singular+"base64"] = &Attribute{
			Name:    singular + "base64",
			Type:    STRING,
			checkFn: checkFn,
			updateFn: func(args *UpdateFnArgs) error {
				v, ok := args.NewObj[singular+"base64"]
				if !ok {
					return nil
				}
				if !IsNil(v) {
					data := v.(string)
					content, err := base64.StdEncoding.DecodeString(data)
					if err != nil {
						return fmt.Errorf("Error decoding \"%sbase64\" "+
							"attribute: "+"%s", singular, err)
					}
					v = any(content)
				}
				args.NewObj["#resource"] = v
				// args.NewObj["#resourceURL"] = nil
				delete(args.NewObj, singular+"base64")
				return nil
			},
		}
	}

	return res
}

func KindIsScalar(k reflect.Kind) bool {
	// SOOOO risky :-)
	return k < reflect.Array || k == reflect.String
}

func IsScalar(daType string) bool {
	return daType == BOOLEAN || daType == DECIMAL || daType == INTEGER ||
		daType == STRING || daType == TIMESTAMP || daType == UINTEGER ||
		daType == URI || daType == URI_REFERENCE || daType == URI_TEMPLATE ||
		daType == URL
}

func (a *Attribute) IsScalar() bool {
	return IsScalar(a.Type)
}

func (a *Attribute) AddAttr(name, daType string) (*Attribute, error) {
	return a.AddAttribute(&Attribute{
		Registry: a.Registry,
		Name:     name,
		Type:     daType,
	})
}

func (a *Attribute) AddAttrMap(name string, item *Item) (*Attribute, error) {
	return a.AddAttribute(&Attribute{Name: name, Type: MAP, Item: item})
}

func (a *Attribute) AddAttrObj(name string) (*Attribute, error) {
	return a.AddAttribute(&Attribute{Name: name, Type: OBJECT, Item: &Item{}})
}

func (a *Attribute) AddAttrArray(name string, item *Item) (*Attribute, error) {
	return a.AddAttribute(&Attribute{Name: name, Type: ARRAY, Item: item})
}
func (a *Attribute) AddAttribute(attr *Attribute) (*Attribute, error) {
	if attr.Name != "*" && !IsValidAttributeName(attr.Name) {
		return nil, fmt.Errorf("Invalid attribute name: %s", attr.Name)
	}

	if a.Item.Attributes == nil {
		a.Item.Attributes = map[string]*Attribute{}
	}

	a.Item.Attributes[attr.Name] = attr
	attr.Registry = a.Registry
	a.Registry.Model.Save()
	return attr, nil
}

// Note this will also validate that the names used to build up the path
// of the attribute are valid (e.g. wrt case and valid chars)
func GetAttributeType(rSID, abstractEntity string, pp *PropPath) (string, error) {
	log.VPrintf(3, ">Enter: GetAttributeType: %s / %s", abstractEntity, pp.UI())
	defer log.VPrintf(3, "<Exit: GetAttributeType")

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

	item := &Item{ // model definition of current thing we're processing
		Attributes: attrs,
		Type:       OBJECT,
		Item:       nil,
	}
	prevTop := ""
	top := ""

	for {
		log.VPrintf(3, "PP: %q Item: %#q", pp.UI(), item)
		// Nothing to do so just return current item
		if pp.Len() == 0 {
			return item.Type, nil
		}

		prevTop = top

		top = pp.Top()

		if IsScalar(item.Type) {
			if pp.Len() != 0 {
				sub := ""
				if pp.Parts[0].Index >= 0 {
					sub = fmt.Sprintf("[%d]", pp.Parts[0].Index)
				}
				return "", fmt.Errorf("Traversing into scalar "+
					"\"%s%s\": %s", prevTop, sub, savePP.UI())
			}
			return item.Type, nil
		}

		if item.Type == ANY {
			return ANY, nil
		}

		if item.Type == OBJECT {
			// We're looking at the attribute name in the object
			if !IsValidAttributeName(top) {
				return "", fmt.Errorf("Attribute name %q isn't valid", top)
			}

			attr := item.Attributes[top]
			if attr == nil {
				attr = item.Attributes["*"]
				if attr == nil {
					return "", nil
				}
			}

			if attr.Type == ANY {
				return attr.Type, nil
			}

			item = &Item{
				Attributes: nil,
				Type:       attr.Type,
				Item:       attr.Item,
			}

			if attr.Item != nil {
				item.Attributes = attr.Item.Attributes
			}
		} else if item.Type == ARRAY {
			// We're looking at the index of the array
			if pp.Parts[0].Index < 0 {
				return "", fmt.Errorf("Array index %q isn't an integer", top)
			}

			item = item.Item
		} else if item.Type == MAP {
			// We're looking at the map key name
			if !IsValidMapKey(top) {
				return "", fmt.Errorf("Map key %q isn't valid", top)
			}
			item = item.Item
		} else {
			panic(fmt.Sprintf("Can't deal with type: %s\npp:%v",
				item.Type, savePP.UI()))
		}

		pp = pp.Next()
	}
}

// This will check to ensure that the entity adheres to the model
func (e *Entity) Validate(currentEntity *Entity) error {
	obj := e.Materialize(nil)
	reg, err := FindRegistryBySID(e.RegistrySID)
	Must(err)

	currentObj := (map[string]any)(nil)
	if currentEntity != nil {
		currentObj = currentEntity.Materialize(nil)
	}

	err = ValidateEntity(reg, obj, currentObj, e.Abstract)
	if err != nil {
		fmt.Printf("Obj:\n%s\n", ToJSON(obj))
	}
	return err
}

// Doesn't fully validate in the sense that it'll assume read-only fields
// are not worth cheching since the server generated them.
// This is mainly used for validating input from a client
func ValidateEntity(reg *Registry, newObj map[string]any,
	oldObj map[string]any, abstract string) error {

	// Don't touch what was passed in
	obj := map[string]any{}
	maps.Copy(obj, newObj)

	for _, coll := range GetCollections(reg.RegistrySID, abstract) {
		log.VPrintf(3, "Deleting collection: %q", coll)
		delete(obj, coll)
		delete(obj, coll+"count")
		delete(obj, coll+"url")
	}

	attrs := GetAttributes(reg.RegistrySID, abstract)
	return ValidateObject(obj, oldObj, attrs, NewPP())
}

func PrepUpdateEntity(reg *Registry, args *UpdateFnArgs) error {
	attrs := GetAttributes(reg.RegistrySID, args.Abstract)

	for key, _ := range attrs {
		attr := attrs[key]
		if attr != nil && attr.updateFn != nil {
			if err := attr.updateFn(args); err != nil {
				return err
			}
		}
	}

	return nil
}

// This should be called after all level-specific calculated properties have
// been removed - such as collections
func ValidateObject(val any, oldObj map[string]any,
	origAttrs Attributes, path *PropPath) error {

	log.VPrintf(3, ">Enter: ValidateObject(%s)", path)
	defer log.VPrintf(3, "<Exit: ValidateObject")
	log.VPrintf(3, "NewObj:\n%s", ToJSON(val))
	log.VPrintf(3, "OrigAttrs:\n%s", ToJSON(origAttrs))

	valValue := reflect.ValueOf(val)
	if valValue.Kind() != reflect.Map ||
		valValue.Type().Key().Kind() != reflect.String {

		return fmt.Errorf("Attribute %q must be an object", path.UI())
	}
	newObj := val.(map[string]any)

	// Convert origAttrs to a slice of *Attribute where "*" is first, if there
	attrs := make([]*Attribute, len(origAttrs))
	count := 1
	for _, attr := range origAttrs {
		if attr.Name == "*" {
			attrs[0] = attr // "*" must appear first in the slice
		} else if count == len(attrs) {
			attrs[0] = attr // at last one and no "*" so use [0]
		} else {
			attrs[count] = attr
			count++
		}
	}

	// Don't touch what was passed in
	objKeys := map[string]bool{}
	for k, _ := range newObj {
		objKeys[k] = true
	}

	attr := (*Attribute)(nil)
	key := ""
	for len(attrs) > 0 {
		l := len(attrs)
		attr = attrs[l-1] // grab last one & remove it
		attrs = attrs[:l-1]

		// Keys are all of the attribute names in newObj we need to check.
		// Normally there's just one (attr.Name) but if attr.Name is "*"
		// then we'll have a list of all remaining attribute names in newObj to
		// check, hence it's a slice not a single string
		keys := []string{}
		if attr.Name != "*" {
			keys = []string{attr.Name}
		} else {
			keys = SortedKeys(objKeys) // no need to be sorted, just grab keys
		}

		// For each attribute (key) in newObj, check its type
		for _, key = range keys {
			val, ok := newObj[key]

			// Based on the attribute's type check the incoming 'val'.
			// This will check for adherence to the model (eg type),
			// the next section (checkFn) will allow for more detailed
			// checking, like for valid values
			if !IsNil(val) {
				err := ValidateAttribute(val, attr, path.P(key))
				if err != nil {
					return err
				}
			}

			// We normally skip read-only attrs, but if it has a checkFn
			// then allow for that to be called
			if attr.ReadOnly {
				// Call the attr's checkFn if there
				if attr.checkFn != nil {
					if err := attr.checkFn(newObj, oldObj); err != nil {
						return err
					}
				}

				delete(objKeys, key)
				continue
			}

			if attr.Required && !ok { // Required but not present
				return fmt.Errorf("Required property %q is missing",
					path.P(key).UI())
			}

			if !attr.Required && (!ok || IsNil(val)) { // treat nil as absent
				delete(objKeys, key)
				continue
			}

			// Call the attr's checkFn if there - for more refined checks
			if attr.checkFn != nil {
				if err := attr.checkFn(newObj, oldObj); err != nil {
					return err
				}
			}

			// Everything is good, so remove it
			delete(objKeys, key)
		}
	}

	// See if we have any extra keys and if so, generate an error
	del := []string{}
	for k, _ := range objKeys {
		if k[0] == '#' {
			del = append(del, k)
		}
	}
	for _, k := range del {
		delete(objKeys, k)
	}
	if len(objKeys) != 0 {
		where := path.UI()
		if where != "" {
			where = " in \"" + where + "\""
		}
		return fmt.Errorf("Invalid extension(s)%s: %s", where,
			strings.Join(SortedKeys(objKeys), ","))
	}

	return nil
}

func ValidateAttribute(val any, attr *Attribute, path *PropPath) error {
	log.VPrintf(3, ">Enter: ValidateAttribute(%s)", path.UI())
	defer log.VPrintf(3, "<Exit: ValidateAttribute")

	log.VPrintf(3, "ValidateAttribute:")
	log.VPrintf(3, " val: %v", val)
	log.VPrintf(3, " path: %v", path.UI())
	log.VPrintf(3, " attr: %v", ToJSON(attr))

	if attr.Type == ANY {
		// All good - let it thru
		return nil
	} else if IsScalar(attr.Type) {
		return ValidateScalar(val, attr, path)
	} else if attr.Type == MAP {
		return ValidateMap(val, attr.Item, path)
	} else if attr.Type == ARRAY {
		return ValidateArray(val, attr.Item, path)
	} else if attr.Type == OBJECT {
		return ValidateObject(val, nil, attr.Item.Attributes, path)
	}

	ShowStack()
	panic(fmt.Sprintf("Unknown type(%s): %s", path.UI(), attr.Type))
}

func ValidateMap(val any, item *Item, path *PropPath) error {
	log.VPrintf(3, ">Enter: ValidateMap(%s)", path.UI())
	defer log.VPrintf(3, "<Exit: ValidateMap")

	log.VPrintf(3, "ValidateMap:")
	log.VPrintf(3, " val: %v", val)
	log.VPrintf(3, " path: %v", path.UI())
	log.VPrintf(3, " item: %v", ToJSON(item))

	valValue := reflect.ValueOf(val)
	if valValue.Kind() != reflect.Map {
		return fmt.Errorf("Attribute %q must be a map", path.UI())
	}

	// All values in the map must be of the same type
	attr := &Attribute{
		Type: item.Type,
		Item: item,
	}

	for _, k := range valValue.MapKeys() {
		keyName := k.Interface().(string)
		v := valValue.MapIndex(k).Interface()
		if IsNil(v) {
			continue
		}
		if err := ValidateAttribute(v, attr, path.P(keyName)); err != nil {
			return err
		}
	}

	return nil
}

func ValidateArray(val any, item *Item, path *PropPath) error {
	log.VPrintf(3, ">Enter: ValidateArray(%s)", path.UI())
	defer log.VPrintf(3, "<Exit: ValidateArray")

	valValue := reflect.ValueOf(val)
	if valValue.Kind() != reflect.Slice {
		return fmt.Errorf("Attribute %q must be an array", path.UI())
	}

	// All values in the array must be of the same type
	attr := &Attribute{
		Type: item.Type,
		Item: item,
	}

	for i := 0; i < valValue.Len(); i++ {
		v := valValue.Index(i).Interface()
		if err := ValidateAttribute(v, attr, path.I(i)); err != nil {
			return err
		}
	}

	return nil
}

func ValidateScalar(val any, attr *Attribute, path *PropPath) error {
	log.VPrintf(3, ">Enter: ValidateScalar(%s:%v)", path, val)
	defer log.VPrintf(3, "<Exit: ValidateScalar")

	valKind := reflect.ValueOf(val).Kind()

	switch attr.Type {
	case BOOLEAN:
		if valKind != reflect.Bool {
			return fmt.Errorf("Attribute %q must be a boolean", path.UI())
		}
	case DECIMAL:
		if valKind != reflect.Int && valKind != reflect.Float64 {
			return fmt.Errorf("Attribute %q must be a decimal", path.UI())
		}
	case INTEGER:
		if valKind == reflect.Float64 {
			f := val.(float64)
			if f != float64(int(f)) {
				return fmt.Errorf("Attribute %q must be an integer", path.UI())
			}
			return nil
		}
		if valKind != reflect.Int {
			return fmt.Errorf("Attribute %q must be an integer", path.UI())
		}
	case UINTEGER:
		i := 0
		if valKind == reflect.Float64 {
			f := val.(float64)
			i = int(f)
			if f != float64(i) {
				return fmt.Errorf("Attribute %q must be a uinteger", path.UI())
			}
		} else if valKind != reflect.Int {
			return fmt.Errorf("Attribute %q must be a uinteger", path.UI())
		} else {
			i = val.(int)
			if valKind != reflect.Int {
				return fmt.Errorf("Attribute %q must be a uinteger", path.UI())
			}
		}
		if i < 0 {
			return fmt.Errorf("Attribute %q must be a uinteger", path.UI())
		}
	case STRING:
		if valKind != reflect.String {
			return fmt.Errorf("Attribute %q must be a string", path.UI())
		}
	case URI:
		if valKind != reflect.String {
			return fmt.Errorf("Attribute %q must be a uri", path.UI())
		}
	case URI_REFERENCE:
		if valKind != reflect.String {
			return fmt.Errorf("Attribute %q must be a uri-reference", path.UI())
		}
	case URI_TEMPLATE:
		if valKind != reflect.String {
			return fmt.Errorf("Attribute %q must be a uri-template", path.UI())
		}
	case URL: // cheat
		if valKind != reflect.String {
			return fmt.Errorf("Attribute %q must be a url", path.UI())
		}
	case TIMESTAMP:
		if valKind != reflect.String {
			return fmt.Errorf("Attribute %q must be a timestamp", path.UI())
		}
		str := val.(string)
		_, err := time.Parse(time.RFC3339, str)
		if err != nil {
			return fmt.Errorf("Attribute %q is a malformed timestamp",
				path.UI())
		}
	}

	return nil
}

func AbstractToSingular(reg *Registry, abs string) string {
	absParts := strings.Split(abs, string(DB_IN))

	if len(absParts) == 0 {
		panic("help")
	}
	gm := reg.Model.Groups[absParts[0]]
	PanicIf(gm == nil, "no gm")

	if len(absParts) == 1 {
		return gm.Singular
	}

	rm := gm.Resources[absParts[1]]
	PanicIf(rm == nil, "no rm")
	return rm.Singular
}
