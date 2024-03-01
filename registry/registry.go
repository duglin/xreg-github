package registry

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	log "github.com/duglin/dlog"
)

type Registry struct {
	Entity
	Model *Model
}

var Registries = map[string]*Registry{}      // User UID->Reg
var RegistriesBySID = map[string]*Registry{} // SID->Reg

func NewRegistry(id string) (*Registry, error) {
	log.VPrintf(3, ">Enter: NewRegistry %q", id)
	defer log.VPrintf(3, "<Exit: NewRegistry")

	if id == "" {
		id = NewUUID()
	}

	if r, err := FindRegistry(id); r != nil {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("A registry with ID %q already exists", id)
	}

	dbSID := NewUUID()
	err := DoOne(`
		INSERT INTO Registries(SID, UID)
		VALUES(?,?)`, dbSID, id)
	if err != nil {
		return nil, err
	}

	reg := &Registry{
		Entity: Entity{
			RegistrySID: dbSID,
			DbSID:       dbSID,
			Plural:      "registries",
			UID:         id,

			Level:    0,
			Path:     "",
			Abstract: "",
		},
	}
	reg.Model = &Model{
		Registry: reg,
		Groups:   map[string]*GroupModel{},
	}

	err = DoOne(`
		INSERT INTO Models(RegistrySID)
		VALUES(?)`, dbSID)
	if err != nil {
		return nil, err
	}

	if err = reg.Set("specversion", SPECVERSION); err != nil {
		return nil, err
	}
	if err = reg.Set("id", reg.UID); err != nil {
		return nil, err
	}
	if err = reg.Set("epoch", 1); err != nil {
		return nil, err
	}

	Registries[id] = reg
	RegistriesBySID[reg.DbSID] = reg

	return reg, nil
}

func GetRegistryNames() []string {
	results, err := Query(` SELECT UID FROM Registries`)
	defer results.Close()

	if err != nil {
		panic(err.Error())
	}

	res := []string{}
	for row := results.NextRow(); row != nil; row = results.NextRow() {
		res = append(res, NotNilString(row[0]))
	}

	return res
}

func (reg *Registry) Get(name string) any {
	return reg.Entity.Get(name)
}

func (reg *Registry) Set(name string, val any) error {
	return reg.Entity.Set(name, val)
}

func (reg *Registry) Delete() error {
	log.VPrintf(3, ">Enter: Reg.Delete(%s)", reg.UID)
	defer log.VPrintf(3, "<Exit: Reg.Delete")

	err := DoOne(`DELETE FROM Registries WHERE SID=?`, reg.DbSID)
	if err == nil {
		delete(Registries, reg.UID)
		delete(RegistriesBySID, reg.DbSID)
	}
	return err
}

func FindRegistryBySID(sid string) (*Registry, error) {
	log.VPrintf(3, ">Enter: FindRegistrySID(%s)", sid)
	defer log.VPrintf(3, "<Exit: FindRegistrySID")

	if os.Getenv("NO_CACHE") == "" {
		if reg, ok := RegistriesBySID[sid]; ok {
			return reg, nil
		}
	}

	results, err := Query(`SELECT UID FROM Registries WHERE SID=?`, sid)
	defer results.Close()

	if err != nil {
		return nil, fmt.Errorf("Error finding Registry %q: %s", sid, err)
	}

	row := results.NextRow()
	if row == nil {
		return nil, fmt.Errorf("Error finding Registry %q: no match", sid)
	}

	uid := NotNilString(row[0])
	return FindRegistry(uid)
}

func FindRegistry(id string) (*Registry, error) {
	log.VPrintf(3, ">Enter: FindRegistry(%s)", id)
	defer log.VPrintf(3, "<Exit: FindRegistry")

	if os.Getenv("NO_CACHE") == "" {
		if reg, ok := Registries[id]; ok {
			return reg, nil
		}
	}

	results, err := Query(`
	   	SELECT SID
	   	FROM Registries
	   	WHERE UID=?`, id)
	defer results.Close()

	if err != nil {
		return nil, fmt.Errorf("Error finding Registry %q: %s", id, err)
	}

	row := results.NextRow()
	if row == nil {
		log.VPrintf(3, "None found")
		return nil, nil
	}
	id = NotNilString(row[0])

	ent, err := RawEntityFromPath(id, "")
	if err != nil {
		return nil, fmt.Errorf("Error finding Registry %q: %s", id, err)
	}
	PanicIf(ent == nil, "No entity but we found a reg")

	reg := &Registry{Entity: *ent}
	reg.LoadModel()

	Registries[reg.UID] = reg
	RegistriesBySID[reg.DbSID] = reg

	return reg, nil
}

func (reg *Registry) LoadModel() *Model {
	return LoadModel(reg)
}

func (reg *Registry) FindGroup(gType string, id string) (*Group, error) {
	log.VPrintf(3, ">Enter: FindGroup(%s/%s)", gType, id)
	defer log.VPrintf(3, "<Exit: FindGroup")

	ent, err := RawEntityFromPath(reg.DbSID, gType+"/"+id)
	if err != nil {
		return nil, fmt.Errorf("Error finding Group %q(%s): %s", id, gType, err)
	}
	if ent == nil {
		log.VPrintf(3, "None found")
		return nil, nil
	}

	return &Group{Entity: *ent, Registry: reg}, nil
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
			RegistrySID: reg.DbSID,
			DbSID:       NewUUID(),
			Plural:      gType,
			UID:         id,

			Level:    1,
			Path:     gType + "/" + id,
			Abstract: gType,
		},
		Registry: reg,
	}

	err = DoOne(`
			INSERT INTO "Groups"(SID,RegistrySID,UID,ModelSID,Path,Abstract)
			SELECT ?,?,?,SID,?,?
			FROM ModelEntities
			WHERE RegistrySID=? AND Plural=? AND ParentSID IS NULL`,
		g.DbSID, reg.DbSID, g.UID, gType+"/"+g.UID, gType, reg.DbSID, gType)

	if err != nil {
		err = fmt.Errorf("Error adding Group: %s", err)
		log.Print(err)
		return nil, err
	}

	if err = g.Set("id", g.UID); err != nil {
		return nil, err
	}
	if err = g.Set("epoch", 1); err != nil {
		return nil, err
	}

	log.VPrintf(3, "Created new one - DbSID: %s", g.DbSID)
	return g, nil
}

func (info *RequestInfo) AddInline(path string) error {
	// use "*" to inline all
	// path = strings.TrimLeft(path, "/.") // To be nice

	pp, err := PropPathFromUI(path)
	if err != nil {
		return err
	}

	for _, group := range info.Registry.Model.Groups {
		if pp.Equals(NewPPP(group.Plural)) {
			info.Inlines = append(info.Inlines, path)
			return nil
		}
		for _, res := range group.Resources {
			if pp.Equals(NewPPP(group.Plural).P(res.Plural)) ||
				pp.Equals(NewPPP(group.Plural).P(res.Plural).P(res.Singular)) ||
				pp.Equals(NewPPP(group.Plural).P(res.Plural).P("versions")) ||
				pp.Equals(NewPPP(group.Plural).P(res.Plural).P("versions").P(res.Singular)) {

				info.Inlines = append(info.Inlines, pp.DB())
				return nil
			}
		}
	}

	// Convert back to UI version for the error message
	path = pp.UI()

	// Remove Abstract value just to print a nicer error message
	if info.Abstract != "" && strings.HasPrefix(path, info.Abstract) {
		path = path[len(info.Abstract)+1:]
	}

	return fmt.Errorf("Invalid 'inline' value: %q", path)
}

func (info *RequestInfo) ShouldInline(entityPath string) bool {
	ePP, _ := PropPathFromDB(entityPath) // entity-PP
	for _, path := range info.Inlines {
		iPP, _ := PropPathFromDB(path) // Inline-PP
		if iPP.Top() == "*" || ePP.Equals(iPP) || iPP.HasPrefix(ePP) {
			log.VPrintf(4, "Inline match: %q in %q", entityPath, path)
			return true
		}
	}
	return false
}

type RequestInfo struct {
	Registry         *Registry
	BaseURL          string
	OriginalPath     string
	OriginalRequest  *http.Request       `json:"-"`
	OriginalResponse http.ResponseWriter `json:"-"`
	Parts            []string
	Root             string
	Abstract         string
	GroupType        string
	GroupUID         string
	ResourceType     string
	ResourceUID      string
	VersionUID       string
	What             string          // Registry, Coll, Entity
	Inlines          []string        // TODO store a PropPaths instead
	Filters          [][]*FilterExpr // [OR][AND] filter=e,e(and) &(or) filter=e
	ShowModel        bool
	ShowMeta         bool

	StatusCode int
	SentStatus bool
	HTTPWriter HTTPWriter `json:"-"`
}

func (ri *RequestInfo) Write(b []byte) (int, error) {
	return ri.HTTPWriter.Write(b)
}

func (ri *RequestInfo) AddHeader(name, value string) {
	ri.HTTPWriter.AddHeader(name, value)
}

type FilterExpr struct {
	Path     string // endpoints.id  TODO store a PropPath?
	Value    string // myEndpoint
	HasEqual bool
}

func ParseRequest(w http.ResponseWriter, r *http.Request) (*RequestInfo, error) {
	path := strings.Trim(r.URL.Path, " /")
	info := &RequestInfo{
		OriginalPath:     path,
		OriginalRequest:  r,
		OriginalResponse: w,
		Registry:         DefaultReg,
		BaseURL:          "http://" + r.Host,
		ShowModel:        r.URL.Query().Has("model"),
		ShowMeta:         r.URL.Query().Has("meta"),
	}

	info.HTTPWriter = DefaultHTTPWriter(info)

	defer func() { log.VPrintf(3, "Info:\n%s\n", ToJSON(info)) }()

	err := info.ParseRequestURL()
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return info, err
	}

	if r.URL.Query().Has("inline") {
		stopInline := false
		for _, value := range r.URL.Query()["inline"] {
			for _, p := range strings.Split(value, ",") {
				if p == "" || p == "*" {
					info.Inlines = []string{"*"}
					stopInline = true
					break
				} else {
					// if we're not at the root then we need to twiddle
					// the inline path to add the HTTP Path as a prefix
					if info.Abstract != "" {
						// want: p = info.Abstract + "." + p  in UI format
						absPP, err := PropPathFromPath(info.Abstract)
						if err != nil {
							info.StatusCode = http.StatusBadRequest
							return info, err
						}
						pPP, err := PropPathFromUI(p)
						if err != nil {
							info.StatusCode = http.StatusBadRequest
							return info, err
						}
						p = absPP.Append(pPP).UI()
					}
					if err := info.AddInline(p); err != nil {
						info.StatusCode = http.StatusBadRequest
						return info, err
					}
				}
			}
			if stopInline {
				break
			}
		}
	}

	err = info.ParseFilters()
	if err != nil {
		info.StatusCode = http.StatusBadRequest
	}

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
			pp, err := PropPathFromUI(path)
			if err != nil {
				return err
			}
			path = pp.DB()

			/*
				if info.What != "Coll" && strings.Index(path, "/") < 0 {
					info.StatusCode = http.StatusBadRequest
					return fmt.Errorf("A filter with just an attribute name (%s) "+
						"isn't allowed in this context", path)
				}
			*/

			if info.Abstract != "" {
				// Want: path = abs + "," + path in DB format
				absPP, _ := PropPathFromPath(info.Abstract)
				absPP = absPP.Append(pp)
				path = absPP.DB()
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
		info.Parts = []string{}
	}

	if len(info.Parts) > 0 && strings.HasPrefix(info.Parts[0], "reg-") {
		info.BaseURL += "/" + info.Parts[0]
		name := info.Parts[0][4:]
		info.Parts = info.Parts[1:] // shift

		reg, err := FindRegistry(name)
		if reg == nil {
			extra := ""
			if err != nil {
				extra = err.Error()
			}
			return fmt.Errorf("Can't find registry %q%s", name, extra)
		}
		info.Registry = reg
	}

	if len(info.Parts) == 0 {
		info.Parts = nil
		info.What = "Registry"
		return nil
	}

	if len(info.Parts) > 0 && info.Parts[0] == "model" {
		return nil
	}

	gModel := (*GroupModel)(nil)
	if info.Registry.Model != nil && info.Registry.Model.Groups != nil {
		gModel = info.Registry.Model.Groups[info.Parts[0]]
	}
	if gModel == nil && (info.Parts[0] != "model" || len(info.Parts) > 1) {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Unknown Group type: %q", info.Parts[0])
	}
	info.GroupType = info.Parts[0]
	info.Root += info.Parts[0]
	info.Abstract += info.Parts[0]

	if len(info.Parts) == 1 {
		info.What = "Coll"
		return nil
	}

	info.GroupUID = info.Parts[1]
	info.Root += "/" + info.Parts[1]
	if len(info.Parts) == 2 {
		info.What = "Entity"
		return nil
	}

	rModel := (*ResourceModel)(nil)
	if gModel.Resources != nil {
		rModel = gModel.Resources[info.Parts[2]]
	}
	if rModel == nil {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Unknown Resource type: %q", info.Parts[2])
	}
	info.ResourceType = info.Parts[2]
	info.Root += "/" + info.Parts[2]
	info.Abstract += "/" + info.Parts[2]

	if len(info.Parts) == 3 {
		info.What = "Coll"
		return nil
	}

	info.ResourceUID = info.Parts[3]
	info.Root += "/" + info.Parts[3]
	if len(info.Parts) == 4 {
		info.What = "Entity"
		return nil
	}

	if info.Parts[4] != "versions" {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Expected \"versions\", got: %q", info.Parts[4])
	}
	info.Root += "/versions"
	info.Abstract += "/versions"
	if len(info.Parts) == 5 {
		info.What = "Coll"
		return nil
	}

	info.VersionUID = info.Parts[5]
	info.Root += "/" + info.Parts[5]

	if len(info.Parts) == 6 {
		info.What = "Entity"
		return nil
	}

	info.StatusCode = http.StatusNotFound
	return fmt.Errorf("Not found")
}

func GenerateQuery(info *RequestInfo) (string, []interface{}, error) {
	query := ""
	args := []any{}

	// Make sure we include the root entity even if the filter excludes it
	rootEntityQuery := func() string {
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

	args = []interface{}{info.Registry.DbSID}
	query = `
SELECT
  RegSID,Level,Plural,eSID,UID,PropName,PropValue,PropType,Path,Abstract
FROM FullTree WHERE RegSID=?`

	// Remove entities that are higher than the GET PATH specified
	if info.What != "Registry" {
		p := strings.Join(info.Parts, "/")
		query += "\nAND "
		if info.What == "Coll" {
			query += "Path LIKE ?"
			args = append(args, p+"/%")
		} else if info.What == "Entity" {
			query += "(Path=? OR Path LIKE ?)"
			args = append(args, p, p+"/%")
		}
	}

	if len(info.Filters) != 0 {
		query += `
AND
(
` + rootEntityQuery() + `
eSID IN ( -- eSID from query
  WITH RECURSIVE cte(eSID,ParentSID,Path) AS (
    SELECT eSID,ParentSID,Path FROM Entities
    WHERE eSID in ( -- start of the OR Filter groupings`
		firstOr := true
		for _, OrFilters := range info.Filters {
			if !firstOr {
				query += `
      UNION -- Adding another OR`
			}
			firstOr = false
			query += `
      -- start of one Filter AND grouping (expre1 AND expr2)
      -- below find SIDs of interest (then find their leaves)
      SELECT list.eSID FROM (
        SELECT count(*) as cnt,e2.eSID,e2.Path FROM Entities AS e1
        RIGHT JOIN (
          -- start of expr1 - below finds SearchNodes/SIDs of interest`
			firstAnd := true
			andCount := 0
			for _, filter := range OrFilters { // AndFilters
				andCount++
				if !firstAnd {
					query += `
          UNION ALL`
				}
				firstAnd = false
				check := ""
				args = append(args, info.Registry.DbSID, filter.Path)
				if filter.HasEqual {
					args = append(args, filter.Value)
					check = "PropValue=?"
				} else {
					check = "PropValue IS NOT NULL"
				}
				// BINARY means case-sensitive for that operand
				query += `
          SELECT eSID,Path FROM FullTree
          WHERE
            RegSID=? AND
            (BINARY CONCAT(IF(Abstract<>'',CONCAT(Abstract,'` + string(DB_IN) + `'),''),PropName)=? AND
               ` + check + `)`
			} // end of AndFilter
			query += `
          -- end of expr1
        ) AS res ON ( res.eSID=e1.eSID )
        JOIN Entities AS e2 ON (
          (e2.Path=res.Path OR e2.Path LIKE
             CONCAT(IF(res.Path<>'',CONCAT(res.Path,'/'),''),'%'))
          AND e2.eSID IN (SELECT * from Leaves)
        ) GROUP BY e2.eSID
        -- end of RIGHT JOIN
      ) as list
      WHERE list.cnt=?
      -- end of one Filter AND grouping (expr1 AND expr2 ...)`
			args = append(args, andCount)
		} // end of OrFilter

		query += `
    ) -- end of all OR Filter groupings
    UNION ALL SELECT e.eSID,e.ParentSID,e.Path FROM Entities AS e
    INNER JOIN cte ON e.eSID=cte.ParentSID)
  SELECT DISTINCT eSID FROM cte )
)
ORDER BY Path ;
`
	}

	log.VPrintf(3, "Query:\n%s\n\n", SubQuery(query, args))
	return query, args, nil
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
- can we set the registry's path to "" instead of NULL ?? already did, test it
- add support for boolean types (set/get/filter)

*/
