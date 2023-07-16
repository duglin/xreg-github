package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	// "reflect"
	"strings"

	log "github.com/duglin/dlog"
)

type Registry struct {
	Entity
	Model *Model
}

var Registries = map[string]*Registry{} // User ID->Reg

func NewRegistry(id string) (*Registry, error) {
	if id == "" {
		id = NewUUID()
	}

	if r, err := FindRegistry(id); r != nil {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("A registry with ID %q alredy exists", id)
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

func (reg *Registry) Delete() error {
	log.VPrintf(3, ">Enter: Reg.Delete(%s)", reg.ID)
	defer log.VPrintf(3, "<Exit: Reg.Delete")

	return DoOne(`DELETE FROM Registries WHERE ID=?`, reg.DbID)
}

func (reg *Registry) AddGroupModel(plural string, singular string, schema string) (*GroupModel, error) {
	if plural == "" {
		return nil, fmt.Errorf("Can't add a group with an empty plural name")
	}
	if singular == "" {
		return nil, fmt.Errorf("Can't add a group with an empty sigular name")
	}

	if reg.Model != nil {
		for _, gm := range reg.Model.Groups {
			if gm.Plural == plural {
				return nil, fmt.Errorf("GroupModel plural %q already exists",
					plural)
			}
			if gm.Singular == singular {
				return nil, fmt.Errorf("GroupModel singular %q already exists",
					singular)
			}
		}
	}

	mID := NewUUID()
	err := DoOne(`
		INSERT INTO ModelEntities(
			ID, RegistryID, ParentID, Plural, Singular, SchemaURL, Versions)
		VALUES(?,?,?,?,?,?,?) `,
		mID, reg.DbID, nil, plural, singular, schema, 0)
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

func FindRegistry(id string) (*Registry, error) {
	log.VPrintf(3, ">Enter: FindRegistry(%s)", id)
	defer log.VPrintf(3, "<Exit: FindRegistry")

	results, err := Query(`
		SELECT r.ID, p.PropName, p.PropValue, p.PropType
		FROM Registries as r LEFT JOIN Props AS p ON (p.EntityID=r.ID)
		WHERE r.RegistryID=?`, id)

	if err != nil {
		return nil, fmt.Errorf("Error finding registry %q: %s", id, err)
	}

	reg := (*Registry)(nil)
	for row := results.NextRow(); row != nil; row = results.NextRow() {
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

	results, err := Query(`
		SELECT
			ID,
			RegistryID,
			ParentID,
			Plural,
			Singular,
			SchemaURL,
			Versions,
			VersionId,
			Latest
		FROM ModelEntities
		WHERE RegistryID=?
		ORDER BY ParentID ASC`, reg.DbID)

	if err != nil {
		log.Printf("Error loading model(%s): %s", reg.ID, err)
		return nil
	}

	model := &Model{
		Registry: reg,
		Groups:   map[string]*GroupModel{},
	}

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		if *row[2] == nil { // ParentID nil -> new Group
			g := &GroupModel{ // Plural
				ID:       NotNilString(row[0]), // ID
				Registry: reg,
				Plural:   NotNilString(row[3]), // Plural
				Singular: NotNilString(row[4]), // Singular
				Schema:   NotNilString(row[5]), // SchemaURL

				Resources: map[string]*ResourceModel{},
			}

			model.Groups[NotNilString(row[3])] = g
			groups[NotNilString(row[0])] = g

		} else { // New Resource
			g := groups[NotNilString(row[2])] // Parent ID

			if g != nil { // should always be true, but...
				r := &ResourceModel{
					ID:         NotNilString(row[0]),
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

	return model
}

func (reg *Registry) FindGroup(gt string, id string) (*Group, error) {
	log.VPrintf(3, ">Enter: FindGroup(%s/%s)", gt, id)
	defer log.VPrintf(3, "<Exit: FindGroup")

	results, err := Query(`
		SELECT g.ID, p.PropName, p.PropValue, p.PropType
		FROM "Groups" AS g
		JOIN ModelEntities AS m ON (m.ID=g.ModelID)
		LEFT JOIN Props AS p ON (p.EntityID=g.ID)
		WHERE g.RegistryID=? AND g.GroupID=? AND m.Plural=?`, reg.DbID, id, gt)
	if err != nil {
		return nil, fmt.Errorf("Error finding group %q(%s): %s", id, gt, err)
	}

	g := (*Group)(nil)
	for row := results.NextRow(); row != nil; row = results.NextRow() {
		if g == nil {
			g = &Group{
				Entity: Entity{
					RegistryID: reg.DbID,
					DbID:       NotNilString(row[0]),
					Plural:     gt,
					ID:         id,
				},
				Registry: reg,
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

	return g, nil
}

func (reg *Registry) AddGroup(gType string, id string) (*Group, error) {
	log.VPrintf(3, ">Enter AddGroup(%s,%s)", gType, id)
	defer log.VPrintf(3, "<Exit AddGroup")

	if reg.Model.Groups[gType] == nil {
		return nil, fmt.Errorf("Error adding Group, unknown type: %s", gType)
	}

	if id == "" {
		id = NewUUID()
	}

	g, err := reg.FindGroup(gType, id)
	if err != nil {
		return nil, fmt.Errorf("Error checking for Group(%s) %q: %s",
			gType, id, err)
	}
	if g != nil {
		return nil, fmt.Errorf("Group %q of type %q already exists", id, gType)
	}

	g = &Group{
		Entity: Entity{
			RegistryID: reg.DbID,
			DbID:       NewUUID(),
			Plural:     gType,
			ID:         id,
		},
		Registry: reg,
	}

	err = DoOne(`
			INSERT INTO "Groups"(ID,RegistryID,GroupID,ModelID,Path,Abstract)
			SELECT ?,?,?,ID,?,?
			FROM ModelEntities
			WHERE RegistryID=? AND Plural=? AND ParentID IS NULL`,
		g.DbID, reg.DbID, g.ID, gType+"/"+g.ID, gType, reg.DbID, gType)

	if err != nil {
		err = fmt.Errorf("Error adding group: %s", err)
		log.Print(err)
		return nil, err
	}
	g.Set("id", g.ID)

	log.VPrintf(3, "Created new one - DbID: %s", g.DbID)
	return g, nil
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

	// Remove Abstract value just to print a nicer error message
	if info.Abstract != "" && strings.HasPrefix(path, info.Abstract) {
		path = path[len(info.Abstract)+1:]
	}

	return fmt.Errorf("Invalid 'inline' value: %q", path)
}

func (info *RequestInfo) ShouldInline(objPath string) bool {
	objPath = strings.Replace(objPath, "/", ".", -1)
	for _, path := range info.Inlines {
		log.VPrintf(4, "Inline check: %q in %q ?", objPath, path)
		if path == "*" || objPath == path || strings.HasPrefix(path, objPath) {
			return true
		}
	}
	return false
}

func (reg *Registry) NewGet(w io.Writer, info *RequestInfo) error {
	info.Root = strings.Trim(info.Root, "/")

	if len(info.Parts) > 0 && info.Parts[0] == "model" {
		if len(info.Parts) > 1 {
			info.ErrCode = http.StatusNotFound
			return fmt.Errorf("404: Not found")
		}
		model := info.Registry.Model
		if model == nil {
			model = &Model{}
		}
		if info.Registry.Model == nil {
			fmt.Fprint(w, "{}\n")
			return nil
		}
		buf, err := json.MarshalIndent(model, "", "  ")
		if err != nil {
			info.ErrCode = http.StatusInternalServerError
			return fmt.Errorf("500: " + err.Error())
		}
		fmt.Fprintf(w, "%s\n", string(buf))
		return nil
	}

	query, args, err := GenerateQuery(info)

	results, err := Query(query, args...)
	if err != nil {
		info.ErrCode = http.StatusInternalServerError
		return fmt.Errorf("500: " + err.Error())
	}

	jw := NewJsonWriter(w, info, results)

	jw.NextObj()

	if info.What == "Coll" {
		_, err = jw.WriteCollection()
	} else {
		if jw.Obj == nil {
			info.ErrCode = http.StatusNotFound
			return fmt.Errorf("404: Not found\n")
		}
		err = jw.WriteObject()
	}

	if err == nil {
		jw.Print("\n")
	} else {
		info.ErrCode = http.StatusInternalServerError
		err = fmt.Errorf("500: " + err.Error())
	}

	return err
}

type RequestInfo struct {
	Registry        *Registry
	BaseURL         string
	OriginalPath    string
	OriginalRequest *http.Request `json:"-"`
	Parts           []string
	Root            string
	Abstract        string
	GroupType       string
	GroupID         string
	ResourceType    string
	ResourceID      string
	VersionID       string
	What            string // Registry, Coll, Entity
	Inlines         []string
	Filters         [][]*FilterExpr // [OR][AND] filter=e,e(and) &(or) filter=e
	ShowModel       bool
	HideProps       bool // Hide props - for less verbose testing
	ErrCode         int
}

type FilterExpr struct {
	Path     string // endpoints.id
	Value    string // myEndpoint
	HasEqual bool
}

func (reg *Registry) ParseRequest(r *http.Request) (*RequestInfo, error) {
	path := strings.Trim(r.URL.Path, " /")
	info := &RequestInfo{
		OriginalPath:    path,
		OriginalRequest: r,
		Registry:        reg,
		BaseURL:         "http://" + r.Host,
		ShowModel:       r.URL.Query().Has("model"),
	}

	defer func() { log.VPrintf(3, "Info:\n%s\n", ToJSON(info)) }()

	err := info.ParseRequestURL()
	if err != nil {
		info.ErrCode = http.StatusBadRequest
		return info, err
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

	err = info.ParseFilters()

	return info, err
}

func (info *RequestInfo) ParseFilters() error {
	for _, filterQ := range info.OriginalRequest.URL.Query()["filter"] {
		// ?filter=path.to.attribute[=value],* & filter=...

		filterQ = strings.TrimSpace(filterQ)
		exprs := strings.Split(filterQ, ",")
		AndFilters := ([]*FilterExpr)(nil)
		for _, expr := range exprs {
			expr = strings.TrimSpace(expr)
			if expr == "" {
				continue
			}
			path, value, found := strings.Cut(expr, "=")

			/*
				if info.What != "Coll" && strings.Index(path, ".") < 0 {
					info.ErrCode = http.StatusBadRequest
					return fmt.Errorf("A filter with just an attribute name (%s) "+
						"isn't allowed in this context", path)
				}
			*/

			if info.Abstract != "" {
				path = strings.Replace(info.Abstract, "/", ".", -1) + "." + path
			}
			filter := &FilterExpr{
				Path:     path,
				Value:    value,
				HasEqual: found,
			}

			if AndFilters == nil {
				AndFilters = []*FilterExpr{}
			}
			AndFilters = append(AndFilters, filter)
		}

		if AndFilters != nil {
			if info.Filters == nil {
				info.Filters = [][]*FilterExpr{}
			}
			info.Filters = append(info.Filters, AndFilters)
		}
	}
	return nil
}

func (info *RequestInfo) ParseRequestURL() error {
	path := strings.Trim(info.OriginalPath, " /")
	info.Parts = strings.Split(path, "/")

	if len(info.Parts) == 1 && info.Parts[0] == "" {
		info.Parts = nil
		info.What = "Registry"
		return nil
	}

	if len(info.Parts) > 0 && info.Parts[0] == "model" {
		return nil
	}

	group := info.Registry.Model.Groups[info.Parts[0]]
	if group == nil && (info.Parts[0] != "model" || len(info.Parts) > 1) {
		info.ErrCode = 404
		return fmt.Errorf("Unknown Group type: %q", info.Parts[0])
	}
	info.GroupType = info.Parts[0]
	info.Root += info.Parts[0]
	info.Abstract += info.Parts[0]

	if len(info.Parts) == 1 {
		info.What = "Coll"
		return nil
	}

	info.GroupID = info.Parts[1]
	info.Root += "/" + info.Parts[1]
	if len(info.Parts) == 2 {
		info.What = "Entity"
		return nil
	}

	res := group.Resources[info.Parts[2]]
	if res == nil {
		info.ErrCode = 404
		return fmt.Errorf("Unknown Resource type: %q", info.Parts[0])
	}
	info.ResourceType = info.Parts[2]
	info.Root += "/" + info.Parts[2]
	info.Abstract += "/" + info.Parts[2]

	if len(info.Parts) == 3 {
		info.What = "Coll"
		return nil
	}

	info.ResourceID = info.Parts[3]
	info.Root += "/" + info.Parts[3]
	if len(info.Parts) == 4 {
		info.What = "Entity"
		return nil
	}

	if info.Parts[4] != "versions" {
		info.ErrCode = 404
		return fmt.Errorf("Expected \"versions\", got: %q", info.Parts[4])
	}
	info.Root += "/versions"
	info.Abstract += "/versions"
	if len(info.Parts) == 5 {
		info.What = "Coll"
		return nil
	}

	info.VersionID = info.Parts[5]
	info.Root += "/" + info.Parts[5]

	if len(info.Parts) == 6 {
		info.What = "Entity"
		return nil
	}

	info.ErrCode = 404
	return fmt.Errorf("Uknown resource path: %q", path)
}

func GenerateQuery(info *RequestInfo) (string, []interface{}, error) {
	q := ""
	args := []any{}

	// Remove entities that are higher than the GET PATH specified
	addPath := func() string {
		subsetPath := ""
		if info.What != "Registry" {
			p := strings.Join(info.Parts, "/")
			subsetPath += "\nAND "
			if info.What == "Coll" {
				subsetPath += "Path LIKE ?"
				args = append(args, p+"/%")
			} else if info.What == "Entity" {
				subsetPath += "(Path=? OR Path LIKE ?)"
				args = append(args, p, p+"/%")
			} else {
				panic("what!")
			}
		}
		return subsetPath
	}

	// Make sure we include the root object even if the filter excludes it
	rootObjQuery := func() string {
		return ""
		res := ""

		/*
			if info.What != "Coll" {
				args = append(args, strings.Join(info.Parts, "/"))
				res = "Path=?\nOR  "
			}
		*/

		return res
	}

	if len(info.Filters) == 0 {
		args = []interface{}{info.Registry.DbID}

		q = fmt.Sprintf("SELECT " +
			"Level,Plural,ID,PropName,PropValue,PropType,Path,Abstract " +
			"FROM FullTree WHERE RegID=?" + addPath())
	} else {
		args = []interface{}{info.Registry.DbID}
		q = fmt.Sprintf(`
SELECT
  Level,Plural,ID,PropName,PropValue,PropType,Path,Abstract
FROM FullTree WHERE
RegID=?` + addPath() + `
AND
(
` + rootObjQuery() + `
eID IN ( -- eID from query
  WITH RECURSIVE cte(eID,ParentID,Path) AS (
    SELECT eID,ParentID,Path FROM Entities
    WHERE eID in ( -- start of the OR Filter groupings`)
		firstOr := true
		for _, OrFilters := range info.Filters {
			if !firstOr {
				q += `
      UNION -- Adding another OR`
			}
			firstOr = false
			q += `
      -- start of one Filter AND grouping (expre1 AND expr2)
      -- below find IDs of interest (then find their leaves)
      SELECT list.eID FROM (
        SELECT count(*) as cnt,e2.eID,e2.Path FROM Entities AS e1
        RIGHT JOIN (
          -- start of expr1 - below finds SeachNodes/IDs of interest`
			firstAnd := true
			andCount := 0
			for _, filter := range OrFilters { // AndFilters
				andCount++
				if !firstAnd {
					q += `
          UNION ALL`
				}
				firstAnd = false
				check := ""
				args = append(args, filter.Path)
				if filter.HasEqual {
					args = append(args, filter.Value)
					check = "PropValue=?"
				} else {
					check = "PropValue IS NOT NULL"
				}
				q += `
          SELECT eID,Path FROM FullTree
          WHERE (CONCAT(IF(Abstract<>'',CONCAT(REPLACE(Abstract,'/','.'),'.'),''),PropName)=? AND
               ` + check + `)`
			} // end of AndFilter
			q += `
          -- end of expr1
        ) AS res ON ( res.eID=e1.eID )
        JOIN Entities AS e2 ON (
          (e2.Path=res.Path OR e2.Path LIKE
             CONCAT(IF(res.Path<>'',CONCAT(res.Path,'/'),''),'%'))
          AND e2.eID IN (SELECT * from Leaves)
        ) GROUP BY e2.eID
        -- end of RIGHT JOIN
      ) as list
      WHERE list.cnt=?
      -- end of one Filter AND grouping (expr1 AND expr2 ...)`
			args = append(args, andCount)
		} // end of OrFilter

		q += `
    ) -- end of all OR Filter groupings
    UNION ALL SELECT e.eID,e.ParentID,e.Path FROM Entities AS e
    INNER JOIN cte ON e.eID=cte.ParentID)
  SELECT DISTINCT eID FROM cte )
)
ORDER BY Path ;
`
	}

	log.VPrintf(3, "Query:\n%s\n\n", SubQuery(q, args))
	return q, args, nil
}

func SubQuery(query string, args []interface{}) string {
	for i, arg := range args {
		before, after, found := strings.Cut(query, "?")
		if !found {
			panic(fmt.Sprintf("Too few ? in query - missing number %d", i+1))
		}
		query = fmt.Sprintf("%s'%v'%s", before, arg, after)
	}
	if i := strings.Index(query, "?"); i >= 0 {
		panic(fmt.Sprintf("Extra ? in query at '%s'", query[i:]))
	}
	return query
}

/*
TODO:
- Move the logic that takes the Path into account for the query into
  GenerateQuery
- Make sure that the Path entity is always in the result set when filtering
- twiddle the self and XXXUrls to include proper filter and inline stuff
- see if we can get rid of the recursion stuff
- should we add "/" to then end of the Path for non-collections, then
  we can just look for PATH/%  and not PATH + PATH/%
- add RegID checks to the filtering query
- can we set the registry's path to "" instead of NULL ?? already did, test it
- add support for boolean types (set/get/filter)

*/
