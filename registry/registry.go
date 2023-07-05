package registry

import (
	// "database/sql"
	"fmt"
	"io"
	"net/http"
	// "os"
	"reflect"
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
	Model        *Model `json:"-"`
	GenericModel *ModelElement

	ID          string
	Name        string
	Description string
	SpecVersion string
	Tags        map[string]string
	Docs        string

	GroupCollections map[string]*GroupCollection // groupType
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

func (reg *Registry) FindOrAddGroupCollection(gType string) *GroupCollection {
	gc, _ := reg.GroupCollections[gType]
	if gc == nil {
		gm := reg.Model.Groups[gType]
		if gm == nil {
			panic(fmt.Sprintf("Can't find GroupModel %q", gType))
		}

		gc = &GroupCollection{
			Registry:   reg,
			GroupModel: gm,
			Groups:     map[string]*Group{},
		}
		if reg.GroupCollections == nil {
			reg.GroupCollections = map[string]*GroupCollection{}
		}
		reg.GroupCollections[gType] = gc
	}
	return gc
}

func (reg *Registry) AddGroup(gt string, id string) *Group {
	gc := reg.FindOrAddGroupCollection(gt)
	return gc.NewGroup(id)
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

func (reg *Registry) FindOrAddGroup(gType string, gID string) *Group {
	log.VPrintf(3, ">Enter FindOrAddGroup(%s,%s)", gType, gID)
	defer log.VPrintf(3, "<Exit FindOrAddGroup")
	if gID == "" {
		gID = NewUUID()
	}

	g := reg.FindGroup(gType, gID)

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
		ID: gID,
	}

	err := DoOne(`
			INSERT INTO "Groups"(ID, RegistryID, GroupID, ModelID,Path,Abstract)
			SELECT ?,?,?,ID,?,? FROM ModelEntities WHERE Plural=?`,
		g.DbID, reg.ID, gID, gType+"/"+g.DbID, gType, gType)

	if err != nil {
		log.Printf("Error adding group: %s", err)
		return nil
	}
	g.Set("id", g.ID)

	log.VPrintf(3, "Created new one - DbID: %s", g.DbID)
	return g
}

func (reg *Registry) oldFindOrAddGroup(gt string, id string) *Group {
	gc := reg.FindOrAddGroupCollection(gt)
	g := gc.Groups[id]
	if g != nil {
		return g
	}
	return gc.NewGroup(id)
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
				Level:  lvl,
				Plural: plural,
				ID:     id,
				Values: map[string]any{},
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
	Level  int
	Plural string
	ID     string
	Values map[string]any
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

func (reg *Registry) NewToJSON(w io.Writer, jd *JSONData, path string) error {
	// w = io.MultiWriter(w, os.Stdout)

	path = strings.TrimRight(path, "/")

	//  /  GROUPs/  GROUPS/x  GROUPS/x/RESOURCES  GROUPS/x/RESOURCES/y
	results, err := NewQuery(`
		SELECT Level, Plural, ID, PropName, PropValue, PropType
		FROM FullTree WHERE RegID=?`, reg.ID)
	if err != nil {
		return err
	}

	rc := ResultsContext{
		results: results,
		pos:     0,
	}

	type depthInfo struct {
		Parent     *depthInfo
		Depth      int
		Groups     []string
		GroupsSave []string
		Prefix     string

		CurrGroup string
		Ending    string
		CollCount int
	}

	di := &depthInfo{
		Depth:  0,
		Groups: SortedKeys(reg.Model.Groups),
		Prefix: jd.Indent + "  ",

		CurrGroup: "",
		Ending:    "",
		CollCount: 0,
	}
	di.GroupsSave = di.Groups

	if path == "" {
		fmt.Fprintf(w, "%s{\n", jd.Prefix)
		OptF(w, `>%s  "id": "%s",`, jd.Indent, reg.ID)
		OptF(w, `>%s  "name": "%s",`, jd.Indent, reg.Name)
		OptF(w, `>%s  "description": "%s",`, jd.Indent, reg.Description)
		OptF(w, `>%s  "specVersion": "%s",`, jd.Indent, reg.SpecVersion)
		fmt.Fprintf(w, "%s  \"self\": \"%s\"", jd.Indent, "...")
		di.Ending = ","
	}

	obj := rc.NextObj()

	for {
		if obj == nil && di != nil && di.Parent == nil {
			break
		}

		if obj != nil {
			log.Printf("d:%d > Obj: L:%d %s %s", di.Depth, obj.Level, obj.Plural, obj.ID)
		}
		/*
			log.Printf("di: gs:%v pre:%d cg:%v end:%q #:%d",
				di.Groups, len(di.Prefix), di.CurrGroup, di.Ending, di.CollCount)
			for pdi := di.Parent; pdi != nil; pdi = pdi.Parent {
				log.Printf("-> pdi: gs:%v pre:%d cg:%v end:%q #:%d",
					pdi.Groups, len(pdi.Prefix), pdi.CurrGroup, pdi.Ending, pdi.CollCount)
			}
		*/

		// End any previous collection if we're exiting its scope
		if di.CollCount > 0 &&
			(obj == nil || obj.Level < di.Depth ||
				(obj.Level == di.Depth && di.CurrGroup != obj.Plural)) {

			if di.CollCount > 0 {
				// If we're in an Obj then close it

				// But first show any empty collections remaining
				if len(di.Groups) > 0 && di.Ending != "" {
					// Add extra \n if there are attrs
					fmt.Fprintf(w, "%s\n", di.Ending)
				}
				for i, g := range di.Groups {
					if i > 0 {
						fmt.Fprintf(w, ",")
					}
					fmt.Fprintf(w, "\n%s\"%sCount\": 0,\n", di.Prefix, g)
					fmt.Fprintf(w, "%s\"%sUrl\": \"...\"", di.Prefix, g)
					di.Groups = di.Groups[1:]
					di.Ending = ","
				}

				// Now reset Groups for next object
				if obj != nil {
					groups := []string{}
					if di.Depth == 1 {
						groups = SortedKeys(reg.Model.Groups[obj.Plural].Resources)
					} else if di.Depth == 2 {
						groups = []string{"versions"}
					}
					di.Groups = groups
				}

				di.Prefix = di.Prefix[:len(di.Prefix)-2]
				fmt.Fprintf(w, "\n%s}", di.Prefix)
			}

			di.Prefix = di.Prefix[:len(di.Prefix)-2]
			fmt.Fprintf(w, "\n%s},\n", di.Prefix)
			fmt.Fprintf(w, "%s\"%sCount\": %d,\n",
				di.Prefix, di.CurrGroup, di.CollCount)
			fmt.Fprintf(w, "%s\"%sUrl\": \"%s\"",
				di.Prefix, di.CurrGroup, "...")

			if obj == nil || obj.Level <= di.Depth {
				di = di.Parent
			}
			di.Ending = ","
			continue
		}

		// Start a new collection
		if obj != nil &&
			(obj.Level > di.Depth ||
				(di.CurrGroup != obj.Plural && obj.Level == di.Depth)) {

			if di.Ending != "" {
				// Add extra \n if there are attrs
				fmt.Fprintf(w, "%s\n", di.Ending)
			}

			// Before we start a new collection show any empty ones
			for len(di.Groups) > 0 && di.Groups[0] < obj.Plural {
				fmt.Fprintf(w, "\n%s\"%sCount\": 0,\n", di.Prefix, di.Groups[0])
				fmt.Fprintf(w, "%s\"%sUrl\": \"...\",", di.Prefix, di.Groups[0])
				di.Groups = di.Groups[1:]
			}
			if len(di.Groups) > 0 {
				di.Groups = di.Groups[1:] // Remove next group too
			}

			fmt.Fprintf(w, "\n%s%q: {", di.Prefix, obj.Plural)

			if obj.Level == di.Depth {
				// Resetting child groups
				di.Groups = di.GroupsSave // Copies items, not just ptr
				di.CurrGroup = obj.Plural
				di.Ending = ""
				di.CollCount = 0
			} else {
				groups := []string{}
				if di.Depth == 0 {
					groups = SortedKeys(reg.Model.Groups[obj.Plural].Resources)
				} else if di.Depth == 2 {
					groups = []string{"versions"}
				}
				di = &depthInfo{
					Parent:     di,
					Depth:      di.Depth + 1,
					Groups:     groups,
					GroupsSave: groups,
					Prefix:     di.Prefix + "  ",
					CurrGroup:  obj.Plural,
					Ending:     "",
					CollCount:  0,
				}
			}
		}

		// Print the obj
		if obj == nil || di.CollCount > 0 {
			// End previous object

			// But first show any empty collections remaining
			if len(di.Groups) > 0 && di.Ending != "" {
				// Add extra \n if there are attrs
				fmt.Fprintf(w, "%s\n", di.Ending)
			}
			for i, g := range di.Groups {
				if i > 0 {
					fmt.Fprintf(w, ",")
				}
				fmt.Fprintf(w, "\n%s\"%sCount\": 0,\n", di.Prefix, g)
				fmt.Fprintf(w, "%s\"%sUrl\": \"...\"", di.Prefix, g)
				di.Groups = di.Groups[1:]
				di.Ending = ","
			}

			di.Prefix = di.Prefix[:len(di.Prefix)-2]
			fmt.Fprintf(w, "\n%s},\n", di.Prefix)
			di.Ending = ","

			// Now reset Groups for next object
			di.Groups = di.GroupsSave
		} else {
			fmt.Fprintf(w, "%s\n", di.Ending)
		}

		if obj == nil {
			// continue
		}

		// Actually print it now
		fmt.Fprintf(w, "%s%q: {", di.Prefix, obj.Values["id"])
		di.Prefix += "  "
		di.Ending = ""

		// Print well-known attrs first
		keys := []string{"id", "name", "epoch"}
		for _, k := range keys {
			val, ok := obj.Values[k]
			if !ok {
				continue
			}
			if reflect.TypeOf(val).Kind() == reflect.String {
				fmt.Fprintf(w, "%s\n%s%q: %q", di.Ending, di.Prefix, k, val)
			} else {
				fmt.Fprintf(w, "%s\n%s%q: %v", di.Ending, di.Prefix, k, val)
			}
			di.Ending = ","
			delete(obj.Values, k)
		}

		// Now show everthing else
		for k, v := range obj.Values {
			if reflect.TypeOf(v).Kind() == reflect.String {
				fmt.Fprintf(w, "%s\n%s%q: %q", di.Ending, di.Prefix, k, v)
			} else {
				fmt.Fprintf(w, "%s\n%s%q: %v", di.Ending, di.Prefix, k, v)
			}
		}
		di.CollCount++

		obj = rc.NextObj()
	}

	// But first show any empty collections remaining
	if len(di.Groups) > 0 && di.Ending != "" {
		// Add extra \n if there are attrs
		fmt.Fprintf(w, "%s\n", di.Ending)
	}
	for i, g := range di.Groups {
		if i > 0 {
			fmt.Fprintf(w, ",")
		}
		fmt.Fprintf(w, "\n%s\"%sCount\": 0,\n", di.Prefix, g)
		fmt.Fprintf(w, "%s\"%sUrl\": \"...\"", di.Prefix, g)
		di.Groups = di.Groups[1:]
		di.Ending = ","
	}

	if path == "" {
		fmt.Fprintf(w, "\n%s}\n", jd.Prefix)
	}

	return nil
}

func (reg *Registry) ToJSON(w io.Writer, jd *JSONData) error {
	fmt.Fprintf(w, "%s{\n", jd.Prefix)
	fmt.Fprintf(w, "%s  \"id\": \"%s\",\n", jd.Indent, reg.ID)
	fmt.Fprintf(w, "%s  \"name\": \"%s\",\n", jd.Indent, reg.Name)
	fmt.Fprintf(w, "%s  \"description\": \"%s\",\n", jd.Indent, reg.Description)
	fmt.Fprintf(w, "%s  \"specVersion\": \"%s\",\n", jd.Indent, reg.SpecVersion)
	fmt.Fprintf(w, "%s  \"self\": \"%s\"", jd.Indent, "...")

	for _, gModel := range reg.Model.Groups {
		results, err := NewQuery(`
			SELECT ID, GroupID FROM "Groups" WHERE RegistryID=? AND ModelID=?`,
			reg.ID, gModel.ID)
		if err != nil {
			return err
		}

		gCount := 0
		for _, row := range results {
			g := reg.FindGroup(gModel.Plural, NotNilString(row[1])) // g.ID
			if g == nil {
				log.Printf("Can't find group %s/%s", gModel.Plural,
					NotNilString(row[1]))
				continue // should never happen, but just in case...
			}

			prefix := fmt.Sprintf(",\n")
			if gCount == 0 {
				prefix += fmt.Sprintf("\n  %q: {\n", gModel.Plural)
			}
			prefix += fmt.Sprintf("    %q: ", g.ID)
			shown, err := g.ToJSON(w, &JSONData{prefix, "    ", reg})
			if err != nil {
				return err
			}
			if shown {
				gCount++
			}
		}
		if gCount > 0 {
			fmt.Fprintf(w, "\n%s  },\n", jd.Indent)
		} else {
			fmt.Fprintf(w, ",\n\n")
		}

		fmt.Fprintf(w, "%s  \"%sCount\": %d,\n", jd.Indent, gModel.Plural,
			gCount)
		fmt.Fprintf(w, "%s  \"%sURL\": \"%s/%s\"", jd.Indent, gModel.Plural,
			"...", gModel.Plural)
	}

	fmt.Fprintf(w, "\n%s}\n", jd.Indent)
	return nil
}

func (reg *Registry) NewGet(w io.Writer) error {
	return reg.NewToJSON(w, &JSONData{"", "", reg}, "/")
}

func (reg *Registry) ToObject(ctx *Context) (*Object, error) {
	obj := NewObject()
	if reg == nil {
		return obj, nil
	}

	obj.AddProperty("id", reg.ID)
	obj.AddProperty("name", reg.Name)
	obj.AddProperty("description", reg.Description)
	obj.AddProperty("specVersion", reg.SpecVersion)
	obj.AddProperty("self", ctx.DataURL())

	tags := NewObject()
	for key, value := range reg.Tags {
		tags.AddProperty(key, value)
	}
	if len(tags.Children) != 0 {
		obj.AddProperty("tags", tags)
	}

	obj.AddProperty("docs", reg.Docs)

	if ctx.ShouldInline("model") {
		obj.AddProperty("", "")
		ctx.ModelPush("model")
		mod, err := reg.Model.ToObject(ctx)
		if err != nil {
			return nil, err
		}
		obj.AddProperty("model", mod)
		ctx.ModelPop()
	}

	for _, key := range SortedKeys(reg.Model.Groups) {
		gType := reg.Model.Groups[key]
		// gCollection := reg.GroupCollections[gType.Plural]
		gCollection := reg.QueryGroup(gType)

		var err error

		ctx.DataPush(gType.Plural)
		ctx.ModelPush(gType.Plural)
		ctx.FilterPush(gType.Plural)
		groupObj := NewObject()
		if gCollection != nil {
			groupObj, err = gCollection.ToObject(ctx)
		}
		ctx.FilterPop()
		ctx.ModelPop()
		ctx.DataPop()
		if err != nil {
			return nil, err
		}

		obj.AddProperty(gType.Plural, &Collection{
			Name:   gType.Plural,
			URL:    URLBuild(ctx.DataURL(), gType.Plural),
			Inline: ctx.ShouldInline(gType.Plural),
			Object: groupObj,
		})
	}

	return obj, nil
}

func (r *Registry) QueryGroup(gModel *GroupModel) *GroupCollection {
	gc := &GroupCollection{
		Registry:   r,
		GroupModel: gModel,
		Groups:     map[string]*Group{}, // id->*Group
	}

	gc.Groups["docker.com"] = r.FindGroup(gModel.Plural, "docker.com")

	return gc
}

func (r *Registry) Get(path string, rFlags *RegistryFlags) (string, error) {
	if r.GenericModel == nil {
		r.GenericModel = CreateGenericModel(r.Model)
	}

	paths := strings.Split(strings.Trim(path, "/"), "/")
	for len(paths) > 0 && paths[0] == "" {
		paths = paths[1:]
	}

	if rFlags == nil {
		rFlags = &RegistryFlags{}
	}

	filters, err := ParseFilterExprs(r, paths, rFlags.Filters)
	if err != nil {
		return "", err
	}

	ctx := &Context{
		Flags:   rFlags,
		BaseURL: r.BaseURL,
		Filters: filters,
	}

	if rFlags.BaseURL != "" {
		ctx.BaseURL = rFlags.BaseURL
	}
	ctx.BaseURL = strings.TrimRight(ctx.BaseURL, "/")

	if len(paths) == 0 {
		obj, err := r.ToObject(ctx)
		if err != nil {
			return "", err
		}
		obj.ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	if len(paths) == 1 && paths[0] == "model" {
		obj, err := r.Model.ToObject(ctx)
		if err != nil {
			return "", err
		}
		obj.ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	// GROUPs
	var gModel *GroupModel
	if gModel = r.FindGroupModel(paths[0]); gModel == nil {
		return "", fmt.Errorf("Unknown group %q", paths[0])
	}
	groupColl := r.GroupCollections[gModel.Plural]
	ctx.BaseURLPush(paths[0])

	if len(paths) == 1 {
		groupCollObj, err := groupColl.ToObject(ctx)
		if err != nil {
			return "", err
		}
		if groupCollObj == nil {
			return "{}", nil
		}
		groupCollObj.ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	// GROUPs/ID
	group := groupColl.Groups[paths[1]]
	if group == nil {
		return "", fmt.Errorf("Unknown group ID %q", paths[1])
	}
	ctx.BaseURLPush(paths[1])
	if len(paths) == 2 {
		groupObj, err := group.ToObject(ctx)
		if err != nil {
			return "", err
		}
		if groupObj == nil {
			return "{}", nil
		}
		groupObj.ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	// GROUPs/ID/RESOURCEs
	resColl := group.ResourceCollections[paths[2]]
	ctx.BaseURLPush(paths[2])
	if resColl == nil {
		return "", fmt.Errorf("Unknown rescource collection %q", paths[2])
	}
	if len(paths) == 3 {
		resCollObj, err := resColl.ToObject(ctx)
		if err != nil {
			return "", err
		}
		if resCollObj == nil {
			return "{}", nil
		}
		resCollObj.ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	// GROUPs/ID/RESOURCEs/ID
	res := resColl.Resources[paths[3]]
	ctx.BaseURLPush(paths[3])
	if res == nil {
		return "", fmt.Errorf("Unknown resource ID %q", paths[3])
	}

	if len(paths) == 4 {
		if ctx.Flags.Self {
			resObj, err := res.ToObject(ctx)
			if err != nil {
				return "", err
			}
			if resObj == nil {
				return "{}", nil
			}
			resObj.ToJson(&ctx.buffer, "", "  ")
			return ctx.buffer.String(), nil
		}
		latest := res.FindVersion(res.LatestId)
		if latest.ResourceContent != nil {
			return string(latest.ResourceContent), nil
		}

		if latest.ResourceProxyURL != "" {
			resp, err := http.Get(latest.ResourceProxyURL)
			if err != nil {
				return "", err
			}
			if resp.StatusCode/100 != 2 {
				return "", fmt.Errorf("%s ->%d",
					latest.ResourceProxyURL, resp.StatusCode)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", err
			}
			return string(body), nil
		}
	}

	// GROUPs/ID/RESOURCEs/ID/versions
	if paths[4] != "versions" {
		return "", fmt.Errorf("Unknown subresource %q", paths[4])
	}
	verColl := res.VersionCollection
	ctx.BaseURLPush(paths[4])
	if len(paths) == 5 {
		verCollObj, err := verColl.ToObject(ctx)
		if err != nil {
			return "", err
		}
		if verCollObj == nil {
			return "{}", nil
		}
		verCollObj.ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	// GROUPs/ID/RESOURCEs/ID/versions/ID
	ver := verColl.Versions[paths[5]]
	if ver == nil {
		return "", fmt.Errorf("Unknown version id %q", paths[5])
	}

	ctx.BaseURLPush(paths[5])
	if len(paths) == 6 {
		if ctx.Flags.Self {
			verObj, err := ver.ToObject(ctx)
			if err != nil {
				return "", err
			}
			if verObj == nil {
				return "{}", nil
			}
			verObj.ToJson(&ctx.buffer, "", "  ")
			return ctx.buffer.String(), nil
		}
		if ver.ResourceContent != nil {
			return string(ver.ResourceContent), nil
		}

		if ver.ResourceProxyURL != "" {
			resp, err := http.Get(ver.ResourceProxyURL)
			if err != nil {
				return "", err
			}
			if resp.StatusCode/100 != 2 {
				return "", fmt.Errorf("%s ->%d",
					ver.ResourceProxyURL, resp.StatusCode)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", err
			}
			return string(body), nil
		}
	}

	return "", fmt.Errorf("Can't figure out what to do with %q",
		strings.Join(paths, "/"))

	return ctx.buffer.String(), nil
}
