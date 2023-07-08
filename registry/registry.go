package registry

import (
	// "database/sql"
	"fmt"
	"io"
	"net/http"
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
	Entity
	BaseURL      string
	Model        *Model
	GenericModel *ModelElement
}

var Registries = map[string]*Registry{}

func NewRegistry(id string) (*Registry, error) {
	if id == "" {
		id = NewUUID()
	}

	dbID := NewUUID()
	err := DoOne(`
		INSERT INTO Registries(ID, RegistryID)
		VALUES(?,?)`, dbID, id)
	if err != nil {
		return nil, err
	}

	reg := &Registry{
		Entity: Entity{
			RegistryID: dbID,
			DbID:       dbID,
			Plural:     "registries",
			ID:         id,
		},
	}
	reg.Set("id", reg.ID)
	Registries[id] = reg

	return reg, nil
}

func (reg *Registry) Set(name string, val any) error {
	return SetProp(reg, name, val)
}

func Nullify(str string) any {
	if str == "" {
		return nil
	}
	return str
}

func (reg *Registry) Delete() error {
	log.VPrintf(3, ">Enter: Reg.Delete(%s)", reg.ID)
	defer log.VPrintf(3, "<Exit: Reg.Delete")

	return DoOne(`DELETE FROM Registries WHERE ID=?`, reg.DbID)
}

func (reg *Registry) Refresh() error {
	log.VPrintf(3, ">Enter: Reg.Refresh(%s)", reg.ID)
	defer log.VPrintf(3, "<Exit: Reg.Refresh")

	if reg.DbID == "" {
		log.Printf("Can't refresh a DB that hasn't been saved")
		return fmt.Errorf("Can't refresh a DB that hasn't been saved")
	}

	result, err := Query(`
		SELECT PropName, PropValue, PropType
		FROM Props WHERE EntityID=? `,
		reg.DbID)
	defer result.Close()

	if err != nil {
		log.Printf("Error refreshing Registry(%s): %s", reg.ID, err)
		return fmt.Errorf("Error refreshing Registry(%s): %s", reg.ID, err)
	}

	*reg = Registry{ // Erase all existing properties
		Entity: Entity{
			RegistryID: reg.DbID,
			DbID:       reg.DbID,
			Plural:     reg.Plural,
			ID:         reg.ID,
		},
	}

	for result.NextRow() {
		name := NotNilString(result.Data[0])
		val := NotNilString(result.Data[1])
		propType := NotNilString(result.Data[2])
		SetField(reg, name, &val, propType)
	}

	return nil
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
		mID, reg.DbID, nil, plural, singular, Nullify(schema), 0)
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
		Entity: Entity{
			ID: id,
		},
	}
	err := reg.Refresh()
	if err != nil {
		reg = nil
	}
	Registries[id] = reg
	return reg, err
}

func FindRegistry(id string) (*Registry, error) {
	log.VPrintf(3, ">Enter: FindRegistry(%s)", id)
	defer log.VPrintf(3, "<Exit: FindRegistry")

	results, err := NewQuery(`
		SELECT r.ID, p.PropName, p.PropValue, p.PropType
		FROM Registries as r LEFT JOIN Props AS p ON (p.EntityID=r.ID)
		WHERE r.RegistryID=?`, id)

	if err != nil {
		return nil, err
	}

	reg := (*Registry)(nil)
	for _, row := range results {
		if reg == nil {
			reg = &Registry{
				Entity: Entity{
					RegistryID: NotNilString(row[0]),
					DbID:       NotNilString(row[0]),
					Plural:     "registries",
					ID:         id,
				},
			}
			log.VPrintf(3, "Found one: %s", reg.DbID)
		}
		if *row[1] != nil { // We have Props
			name := NotNilString(row[1])
			val := NotNilString(row[2])
			propType := NotNilString(row[3])
			SetField(reg, name, &val, propType)
		}
	}

	if reg == nil {
		log.VPrintf(3, "None found")
	}
	return reg, nil
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
		ORDER BY ParentID ASC`, reg.DbID)
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
					RegistryID: reg.DbID,
					DbID:       NotNilString(row[0]),
					Plural:     gt,
					ID:         id,
				},
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
			RegistryID: reg.DbID,
			DbID:       NewUUID(),
			Plural:     gType,
			ID:         id,
		},
	}

	err := DoOne(`
			INSERT INTO "Groups"(ID, RegistryID, GroupID, ModelID,Path,Abstract)
			SELECT ?,?,?,ID,?,? FROM ModelEntities WHERE Plural=?`,
		g.DbID, reg.DbID, g.ID, gType+"/"+g.ID, gType, gType)

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
				Path:     NotNilString(row[6]),
				Abstract: NotNilString(row[7]),
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
	Path     string
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

	return fmt.Errorf("Bad inline - path: %q", path)
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
		"Level, Plural, ID, PropName, PropValue, PropType, Path, Abstract " +
		"FROM FullTree WHERE RegID=? "

	args := []interface{}{reg.DbID}

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
		/*
			if jw.Obj == nil {
				jw.Print("{}\n")
				// return fmt.Errorf("not found\n")
			}
		*/
		_, err = jw.WriteCollection()
	} else if info.What == "Entity" {
		jw.NextObj()
		if jw.Obj == nil {
			return fmt.Errorf("not found\n")
		}
		err = jw.WriteObject()
	}
	if err == nil {
		jw.Print("\n")
	}

	return err
}

type RequestInfo struct {
	Registry     *Registry
	BaseURL      string
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

func (reg *Registry) ParseRequest(r *http.Request) (*RequestInfo, error) {
	path := strings.Trim(r.URL.Path, " /")
	info := &RequestInfo{
		OriginalPath: path,
		Registry:     reg,
		BaseURL:      "http://" + r.Host,
	}

	if r.URL.Query().Has("inline") {
		for _, value := range r.URL.Query()["inline"] {
			for _, p := range strings.Split(value, ",") {
				if p == "" || p == "*" {
					info.Inlines = []string{"*"}
				} else {
					// if we're not at the root then we need to twiddle
					// the inline path to add the HTTP Path as a prefix
					if info.Abstract != "" {
						p = info.Abstract + "." + p
					}
					if err := info.AddInline(p); err != nil {
						info.ErrCode = http.StatusBadRequest
						return info, err
					}
				}
			}
		}
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
