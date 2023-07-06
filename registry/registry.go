package registry

import (
	// "database/sql"
	"fmt"
	"io"
	// "net/http"
	// "os"
	// "reflect"
	"strings"

	log "github.com/duglin/dlog"
)

type RegistryFlags struct {
	BaseURL     string
	Indent      string
	InlineAll   bool
	InlinePaths []string
	Self        bool
	AsDoc       bool
	OrFilters   [][]string // [OLD][AND]string

	Filters []string // OLD
}

type Registry struct {
	BaseURL      string
	Model        *Model
	GenericModel *ModelElement

	ID          string
	Name        string
	Description string
	SpecVersion string
	Docs        string
	Tags        map[string]string
}

var Registries = map[string]*Registry{}

func NewRegistry(id string) (*Registry, error) {
	if id == "" {
		id = NewUUID()
	}

	if err := DoOne("INSERT INTO Registries(ID) VALUES(?)", id); err != nil {
		return nil, err
	}

	reg := &Registry{
		ID: id,
	}
	Registries[id] = reg

	return reg, nil
}

func Nullify(str string) any {
	if str == "" {
		return nil
	}
	return str
}

func NewRegistryFromStruct(reg *Registry) error {
	log.VPrintf(3, "In NewRegistryFromStruct: %#v", reg)
	Registries[reg.ID] = reg
	return reg.Save()
}

func (reg *Registry) Delete() error {
	log.VPrintf(3, ">Enter: Reg.Delete(%s)", reg.ID)
	defer log.VPrintf(3, "<Exit: Reg.Delete")

	return DoOne(`DELETE FROM Registries WHERE ID=?`, reg.ID)
}

func (reg *Registry) Refresh() error {
	log.VPrintf(3, ">Enter: Reg.Refresh(%s)", reg.ID)
	defer log.VPrintf(3, "<Exit: Reg.Refresh")

	if reg.ID == "" {
		log.Printf("Can't refresh a DB that hasn't been saved")
		return fmt.Errorf("Can't refresh a DB that hasn't been saved")
	}

	result, err := QueryRow(`
		SELECT COALESCE(Name,'') as Name,
			   COALESCE(BaseURL,'') as BaseURL,
		       COALESCE(Description,'') as Description,
			   COALESCE(SpecVersion,'') as SpecVersion,
			   COALESCE(Docs,'') as Docs
		FROM Registries WHERE ID=? `, reg.ID)

	if err != nil {
		log.Printf("Error refreshing reg: %s", err)
		return err
	}

	if result != nil {
		reg.Name = NotNilString(result.Data[0])
		reg.BaseURL = NotNilString(result.Data[1])
		reg.Description = NotNilString(result.Data[2])
		reg.SpecVersion = NotNilString(result.Data[3])
		reg.Docs = NotNilString(result.Data[4])
	}

	return nil
}

func (reg *Registry) Save() error {
	if reg.ID == "" {
		reg.ID = NewUUID()
	}

	err := DoOne(`
		REPLACE INTO Registries(ID,Name,BaseURL,Description,SpecVersion,Docs)
		VALUES(?,?,?,?,?,?)`,
		reg.ID, Nullify(reg.Name), Nullify(reg.BaseURL),
		Nullify(reg.Description), Nullify(reg.SpecVersion), Nullify(reg.Docs))
	if err != nil {
		return err
	}

	return reg.Refresh()
}

func (reg *Registry) AddGroupModel(plural string, singular string, schema string) (*GroupModel, error) {
	if plural == "" {
		return nil, fmt.Errorf("Can't add a group with an empty plural name")
	}
	if singular == "" {
		return nil, fmt.Errorf("Can't add a group with an empty sigular name")
	}
	mID := NewUUID()
	err := DoOne(`
		INSERT INTO ModelEntities(
			ID,
			RegistryID,
			ParentID,
			Plural,
			Singular,
			SchemaURL,
			Versions)
		VALUES(?,?,?,?,?,?,?) `,
		mID, reg.ID, nil, plural, singular, Nullify(schema), 0)
	if err != nil {
		log.Printf("Error inserting group(%s): %s", plural, err)
		return nil, err
	}
	g := &GroupModel{
		ID:       mID,
		Registry: reg,
		Singular: singular,
		Plural:   plural,
		Schema:   schema,

		Resources: map[string]*ResourceModel{},
	}

	if reg.Model == nil {
		reg.Model = &Model{
			Registry: reg,
			Groups:   map[string]*GroupModel{},
		}
	}

	reg.Model.Groups[plural] = g

	return g, nil
}

func GetRegistryByID(id string) (*Registry, error) {
	reg := &Registry{
		ID: id,
	}
	err := reg.Refresh()
	if err != nil {
		reg = nil
	}
	Registries[id] = reg
	return reg, err
}

func GetRegistryByName(name string) (*Registry, error) {
	result, err := QueryRow(`
		SELECT ID,
			   COALESCE(Name,'') as Name,
			   COALESCE(BaseURL,'') as BaseURL,
		       COALESCE(Description,'') as Description,
			   COALESCE(SpecVersion,'') as SpecVersion,
			   COALESCE(Docs,'') as Docs
		FROM Registries WHERE Name=? `, name)
	if err != nil {
		return nil, err
	}

	reg := &Registry{
		ID:          NotNilString(result.Data[0]),
		Name:        NotNilString(result.Data[1]),
		BaseURL:     NotNilString(result.Data[2]),
		Description: NotNilString(result.Data[3]),
		SpecVersion: NotNilString(result.Data[4]),
		Docs:        NotNilString(result.Data[5]),
	}
	Registries[reg.ID] = reg
	return reg, nil
}

func (reg *Registry) SetName(val string) error {
	err := DoOne("UPDATE Registries SET Name=? WHERE ID=?", val, reg.ID)
	if err == nil {
		reg.Name = val
	}
	return err
}

func (reg *Registry) SetBaseURL(val string) error {
	err := DoOne("UPDATE Registries SET BaseURL=? WHERE ID=?", val, reg.ID)
	if err == nil {
		reg.BaseURL = val
	}
	return err
}

func (reg *Registry) SetDescription(val string) error {
	err := DoOne("UPDATE Registries SET Description=? WHERE ID=?", val, reg.ID)
	if err == nil {
		reg.Description = val
	}
	return err
}

func (reg *Registry) SetSpecVersion(val string) error {
	err := DoOne("UPDATE Registries SET SpecVersion=? WHERE ID=?", val, reg.ID)
	if err == nil {
		reg.SpecVersion = val
	}
	return err
}

func (reg *Registry) SetDocs(val string) error {
	err := DoOne("UPDATE Registries SET Docs=? WHERE ID=?", val, reg.ID)
	if err == nil {
		reg.Docs = val
	}
	return err
}

func (reg *Registry) FindGroupModel(gTypePlural string) *GroupModel {
	for _, gModel := range reg.Model.Groups {
		if strings.EqualFold(gModel.Plural, gTypePlural) {
			return gModel
		}
	}
	return nil
}

func (reg *Registry) LoadModel() *Model {
	groups := map[string]*GroupModel{} // Model ID -> *GroupModel

	result, err := Query(`
		SELECT
			ID,
			RegistryID,
			ParentID,
			Plural,
			Singular,
			SchemaURL,
			Versions
		FROM ModelEntities
		WHERE RegistryID=?
		ORDER BY ParentID ASC`, reg.ID)
	defer result.Close()

	if err != nil {
		log.Printf("Error loading model(%s): %s", reg.ID, err)
		return nil
	}

	model := &Model{
		Registry: reg,
		Groups:   map[string]*GroupModel{},
	}

	for result.NextRow() {
		if *result.Data[2] == nil { // ParentID nil -> new Group
			g := &GroupModel{ // Plural
				ID:       NotNilString(result.Data[0]), // ID
				Registry: reg,
				Plural:   NotNilString(result.Data[3]), // Plural
				Singular: NotNilString(result.Data[4]), // Singular
				Schema:   NotNilString(result.Data[5]), // SchemaURL
				Versions: NotNilInt(result.Data[6]),    // Versions

				Resources: map[string]*ResourceModel{},
			}

			model.Groups[NotNilString(result.Data[3])] = g
			groups[NotNilString(result.Data[0])] = g

		} else { // New Resource
			g := groups[NotNilString(result.Data[2])] // Parent ID

			if g != nil { // should always be true, but...
				r := &ResourceModel{
					ID:         NotNilString(result.Data[0]), // ID
					GroupModel: g,
					Plural:     NotNilString(result.Data[3]), // Plural
					Singular:   NotNilString(result.Data[4]), // Singular
					Versions:   NotNilInt(result.Data[6]),    // Versions
				}

				g.Resources[r.Plural] = r
			}
		}
	}

	return model
}

func (reg *Registry) FindGroup(gt string, id string) *Group {
	log.VPrintf(3, ">Enter: FindGroup(%s/%s)", gt, id)
	defer log.VPrintf(3, "<Exit: FindGroup")

	results, _ := NewQuery(`
		SELECT g.ID, p.PropName, p.PropValue, p.PropType
		FROM "Groups" AS g
		JOIN ModelEntities AS m ON (m.ID=g.ModelID)
		LEFT JOIN Props AS p ON (p.EntityID=g.ID)
		WHERE g.GroupID=? AND m.Plural=?`, id, gt)

	g := (*Group)(nil)
	for _, row := range results {
		if g == nil {
			g = &Group{
				Entity: Entity{
					RegistryID: reg.ID,
					DbID:       NotNilString(row[0]),
					Plural:     gt,
				},
				ID: id,
			}
			log.VPrintf(3, "Found one: %s", g.DbID)
		}
		if *row[1] != nil { // We have Props
			name := NotNilString(row[1])
			val := NotNilString(row[2])
			propType := NotNilString(row[3])
			SetField(g, name, &val, propType)
		}
	}

	if g == nil {
		log.VPrintf(3, "None found")
	}

	return g
}

func (reg *Registry) FindOrAddGroup(gType string, id string) *Group {
	log.VPrintf(3, ">Enter FindOrAddGroup(%s,%s)", gType, id)
	defer log.VPrintf(3, "<Exit FindOrAddGroup")
	if id == "" {
		id = NewUUID()
	}

	g := reg.FindGroup(gType, id)

	if g != nil {
		log.VPrintf(3, "Found one")
		return g
	}

	g = &Group{
		Entity: Entity{
			RegistryID: reg.ID,
			DbID:       NewUUID(),
			Plural:     gType,
		},
		ID: id,
	}

	err := DoOne(`
			INSERT INTO "Groups"(ID, RegistryID, GroupID, ModelID,Path,Abstract)
			SELECT ?,?,?,ID,?,? FROM ModelEntities WHERE Plural=?`,
		g.DbID, reg.ID, g.ID, gType+"/"+g.ID, gType, gType)

	if err != nil {
		log.Printf("Error adding group: %s", err)
		return nil
	}
	g.Set("id", g.ID)

	log.VPrintf(3, "Created new one - DbID: %s", g.DbID)
	return g
}

func OptF(w io.Writer, f string, prefix string, arg string) {
	if arg == "" {
		return
	}
	if f[0] == '>' {
		f = f[1:] + "\n"
	}
	fmt.Fprintf(w, f, prefix, arg)
}

func readObj(results [][]*any, index int) (*Obj, int) {
	obj := (*Obj)(nil)

	for index < len(results) {
		row := results[index]

		lvl := int((*row[0]).(int64))
		plural := NotNilString(row[1])
		id := NotNilString(row[2])

		if obj == nil {
			obj = &Obj{
				Level:    lvl,
				Plural:   plural,
				ID:       id,
				Abstract: NotNilString(row[6]),
				Values:   map[string]any{},
			}
		} else {
			if obj.Level != lvl || obj.Plural != plural || obj.ID != id {
				break
			}
		}
		val := NotNilString(row[4])
		vType := NotNilString(row[5])

		obj.Values[NotNilString(row[3])] = Convert(val, vType)
		index++
	}

	return obj, index
}

type Obj struct {
	Level    int
	Plural   string
	ID       string
	Abstract string
	Values   map[string]any
}

type ResultsContext struct {
	results [][]*any
	pos     int
}

func (rc *ResultsContext) NextObj() *Obj {
	obj, nextPos := readObj(rc.results, rc.pos)
	rc.pos = nextPos
	return obj
}

func (info *RequestInfo) AddInline(path string) error {
	// use "*" to inline all
	path = strings.Trim(path, "/.") // To be nice

	for _, group := range info.Registry.Model.Groups {
		if path == group.Plural {
			info.Inlines = append(info.Inlines, path)
			return nil
		}
		for _, res := range group.Resources {
			if path == group.Plural+"."+res.Plural ||
				path == group.Plural+"."+res.Plural+"."+res.Singular ||
				path == group.Plural+"."+res.Plural+".versions" ||
				path == group.Plural+"."+res.Plural+".versions."+res.Singular {

				info.Inlines = append(info.Inlines, path)
				return nil
			}
		}
	}

	return fmt.Errorf("Bad inline path: %q", path)
}

func (info *RequestInfo) ShouldInline(objPath string) bool {
	objPath = strings.Replace(objPath, "/", ".", -1)
	for _, path := range info.Inlines {
		log.VPrintf(3, "Inline check: %q in %q ?", objPath, path)
		if path == "*" || objPath == path || strings.HasPrefix(path, objPath) {
			return true
		}
	}
	return false
}

func (reg *Registry) NewGet(w io.Writer, info *RequestInfo) error {
	info.Root = strings.Trim(info.Root, "/")

	query := "SELECT " +
		"Level, Plural, ID, PropName, PropValue, PropType, Abstract " +
		"FROM FullTree WHERE RegID=? "

	args := []interface{}{reg.ID}

	if info.What != "Registry" {
		p := strings.Join(info.Parts, "/")
		query += "AND "

		if info.What == "Coll" {
			query += "Path LIKE ?"
			args = append(args, p+"/%")
		} else if info.What == "Entity" {
			query += "(Path=? OR Path LIKE ?)"
			args = append(args, p, p+"/%")
		} else {
			panic("what!")
		}
	}

	results, err := NewQuery(query, args...)
	if err != nil {
		return err
	}

	jw := NewJsonWriter(w, info, results)

	if info.What == "Registry" {
		err = jw.WriteRegistry()
	} else if info.What == "Coll" {
		jw.NextObj()
		if jw.Obj == nil {
			return fmt.Errorf("not found\n")
		}
		_, err = jw.WriteCollection()
	} else if info.What == "Entity" {
		jw.NextObj()
		if jw.Obj == nil {
			return fmt.Errorf("not found\n")
		}
		err = jw.WriteObject()
	}

	return err
}

type RequestInfo struct {
	Registry     *Registry
	OriginalPath string
	Parts        []string
	Root         string
	Abstract     string
	GroupType    string
	GroupID      string
	ResourceType string
	ResourceID   string
	VersionID    string
	What         string // Registry, Coll, Entity
	Inlines      []string
	ErrCode      int
}

func (reg *Registry) ParseRequestPath(path string) (*RequestInfo, error) {
	path = strings.Trim(path, " /")
	info := &RequestInfo{
		OriginalPath: path,
	}

	info.Parts = strings.Split(path, "/")

	if len(info.Parts) == 1 && info.Parts[0] == "" {
		info.Parts = nil
		info.What = "Registry"
		return info, nil
	}

	group := reg.Model.Groups[info.Parts[0]]
	if group == nil {
		info.ErrCode = 404
		return info, fmt.Errorf("Unknown Group type: %q", info.Parts[0])
	}
	info.GroupType = info.Parts[0]
	info.Root += info.Parts[0]
	info.Abstract += info.Parts[0]

	if len(info.Parts) == 1 {
		info.What = "Coll"
		return info, nil
	}

	info.GroupID = info.Parts[1]
	info.Root += "/" + info.Parts[1]
	if len(info.Parts) == 2 {
		info.What = "Entity"
		return info, nil
	}

	res := group.Resources[info.Parts[2]]
	if res == nil {
		info.ErrCode = 404
		return info, fmt.Errorf("Unknown Resource type: %q", info.Parts[0])
	}
	info.ResourceType = info.Parts[2]
	info.Root += "/" + info.Parts[2]
	info.Abstract += "/" + info.Parts[2]

	if len(info.Parts) == 3 {
		info.What = "Coll"
		return info, nil
	}

	info.ResourceID = info.Parts[3]
	info.Root += "/" + info.Parts[3]
	if len(info.Parts) == 4 {
		info.What = "Entity"
		return info, nil
	}

	if info.Parts[4] != "versions" {
		info.ErrCode = 404
		return info, fmt.Errorf("Expected \"versions\", got: %q", info.Parts[4])
	}
	info.Root += "/versions"
	info.Abstract += "/versions"
	if len(info.Parts) == 5 {
		info.What = "Coll"
		return info, nil
	}

	info.VersionID = info.Parts[5]
	info.Root += "/" + info.Parts[5]

	if len(info.Parts) == 6 {
		info.What = "Entity"
		return info, nil
	}

	info.ErrCode = 404
	return info, fmt.Errorf("Uknown resource path: %q", path)
}
