package registry

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	log "github.com/duglin/dlog"
)

type Registry struct {
	Entity
	Model *Model
}

func (r *Registry) Rollback() error {
	if r != nil {
		return r.tx.Rollback()
	}
	return nil
}

func (r *Registry) Commit() error {
	if r != nil {
		return r.tx.Commit()
	}
	return nil
}

type RegOpt string

func NewRegistry(tx *Tx, id string, regOpts ...RegOpt) (*Registry, error) {
	log.VPrintf(3, ">Enter: NewRegistry %q", id)
	defer log.VPrintf(3, "<Exit: NewRegistry")

	var err error // must be used for all error checking due to defer
	newTx := false

	defer func() {
		if newTx {
			// If created just for us, close it
			tx.Conditional(err)
		}
	}()

	if tx == nil {
		tx, err = NewTx()
		if err != nil {
			return nil, err
		}
		newTx = true
	}

	if id == "" {
		id = NewUUID()
	}

	r, err := FindRegistry(tx, id)
	if err != nil {
		return nil, err
	}
	if r != nil {
		return nil, fmt.Errorf("A registry with ID %q already exists", id)
	}

	dbSID := NewUUID()
	err = DoOne(tx, `
		INSERT INTO Registries(SID, UID)
		VALUES(?,?)`, dbSID, id)
	if err != nil {
		return nil, err
	}

	reg := &Registry{
		Entity: Entity{
			tx: tx,

			DbSID:    dbSID,
			Plural:   "registries",
			Singular: "registry",
			UID:      id,

			Type:     ENTITY_REGISTRY,
			Path:     "",
			Abstract: "",
		},
	}
	reg.Entity.Registry = reg
	reg.Model = &Model{
		Registry: reg,
		Schemas:  []string{XREGSCHEMA + "/" + SPECVERSION},
		Groups:   map[string]*GroupModel{},
	}

	tx.Registry = reg
	reg.tx = tx

	err = reg.Model.Verify()
	if err != nil {
		return nil, err
	}

	err = DoOne(tx, `
		INSERT INTO Models(RegistrySID)
		VALUES(?)`, dbSID)
	if err != nil {
		return nil, err
	}

	if err = reg.JustSet("specversion", SPECVERSION); err != nil {
		return nil, err
	}
	if err = reg.JustSet("registryid", reg.UID); err != nil {
		return nil, err
	}

	/*
		for _, regOpt := range regOpts {
			// if regOpts == RegOpt_STRING { ... }
		}
	*/

	if err = reg.SetSave("epoch", 1); err != nil {
		return nil, err
	}

	if err = reg.Model.VerifyAndSave(); err != nil {
		return nil, err
	}

	return reg, nil
}

func GetRegistryNames() []string {
	tx, err := NewTx()
	if err != nil {
		return []string{} // error!
	}
	defer tx.Rollback()

	results, err := Query(tx, ` SELECT UID FROM Registries`)
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

var _ EntitySetter = &Registry{}

func (reg *Registry) Get(name string) any {
	return reg.Entity.Get(name)
}

// Technically this should be called SetValidateSave
func (reg *Registry) SetCommit(name string, val any) error {
	return reg.Entity.eSetCommit(name, val)
}

func (reg *Registry) JustSet(name string, val any) error {
	return reg.Entity.eJustSet(NewPPP(name), val)
}

func (reg *Registry) SetSave(name string, val any) error {
	return reg.Entity.eSetSave(name, val)
}

func (reg *Registry) Delete() error {
	log.VPrintf(3, ">Enter: Reg.Delete(%s)", reg.UID)
	defer log.VPrintf(3, "<Exit: Reg.Delete")

	return DoOne(reg.tx, `DELETE FROM Registries WHERE SID=?`, reg.DbSID)
}

func FindRegistryBySID(tx *Tx, sid string) (*Registry, error) {
	log.VPrintf(3, ">Enter: FindRegistrySID(%s)", sid)
	defer log.VPrintf(3, "<Exit: FindRegistrySID")

	if tx.Registry != nil && tx.Registry.DbSID == sid {
		return tx.Registry, nil
	}

	ent, err := RawEntityFromPath(tx, sid, "", false)
	if err != nil {
		return nil, fmt.Errorf("Error finding Registry %q: %s", sid, err)
	}
	if ent == nil {
		return nil, nil
	}

	reg := &Registry{Entity: *ent}
	if tx.Registry == nil {
		tx.Registry = reg
	}
	reg.Entity.Registry = reg
	reg.tx = tx

	reg.LoadModel()
	return reg, nil
}

// BY UID
func FindRegistry(tx *Tx, id string) (*Registry, error) {
	log.VPrintf(3, ">Enter: FindRegistry(%s)", id)
	defer log.VPrintf(3, "<Exit: FindRegistry")

	if tx != nil && tx.Registry != nil && tx.Registry.UID == id {
		return tx.Registry, nil
	}

	newTx := false
	if tx == nil {
		var err error
		tx, err = NewTx()
		if err != nil {
			return nil, err
		}
		newTx = true

		defer func() {
			// If we created a new Tx then assume someone is just looking
			// for the Registry and may not actually want to edit stuff, so
			// go ahead and close the Tx. It'll be reopened later if needed.
			// If a Tx was passed in then don't close it, the caller will
			if newTx { // not needed?
				tx.Rollback()
			}
		}()
	}

	results, err := Query(tx, `
	   	SELECT SID
	   	FROM Registries
	   	WHERE UID=?`, id)

	defer results.Close()

	if err != nil {
		if newTx {
			tx.Rollback()
		}
		return nil, fmt.Errorf("Error finding Registry %q: %s", id, err)
	}

	row := results.NextRow()

	if row == nil {
		log.VPrintf(3, "None found")
		return nil, nil
	}

	id = NotNilString(row[0])
	results.Close()

	ent, err := RawEntityFromPath(tx, id, "", false)

	if err != nil {
		if newTx {
			tx.Rollback()
		}
		return nil, fmt.Errorf("Error finding Registry %q: %s", id, err)
	}

	PanicIf(ent == nil, "No entity but we found a reg")

	reg := &Registry{Entity: *ent}

	if tx.Registry == nil {
		tx.Registry = reg
	}

	reg.Entity.Registry = reg
	reg.tx = tx

	reg.LoadModel()

	return reg, nil
}

func (reg *Registry) LoadModel() *Model {
	return LoadModel(reg)
}

func (reg *Registry) LoadModelFromFile(file string) error {
	log.VPrintf(3, ">Enter: LoadModelFromFile: %s", file)
	defer log.VPrintf(3, "<Exit:LoadModelFromFile")

	var err error
	buf := []byte{}
	if strings.HasPrefix(file, "http") {
		res, err := http.Get(file)
		if err == nil {
			buf, err = io.ReadAll(res.Body)
			res.Body.Close()

			if res.StatusCode/100 != 2 {
				err = fmt.Errorf("Error getting model: %s\n%s",
					res.Status, string(buf))
			}
		}
	} else {
		buf, err = os.ReadFile(file)
	}
	if err != nil {
		return fmt.Errorf("Processing %q: %s", file, err)
	}

	buf, err = ProcessIncludes(file, buf, true)
	if err != nil {
		return fmt.Errorf("Processing %q: %s", file, err)
	}

	model := &Model{
		Registry: reg,
	}

	if err := Unmarshal(buf, model); err != nil {
		return fmt.Errorf("Processing %q: %s", file, err)
	}

	model.SetPointers()

	if err := model.Verify(); err != nil {
		return fmt.Errorf("Processing %q: %s", file, err)
	}

	if err := reg.Model.ApplyNewModel(model); err != nil {
		return fmt.Errorf("Processing %q: %s", file, err)
	}

	// reg.Model = model
	// reg.Model.VerifyAndSave()
	return nil
}

func (reg *Registry) Update(obj Object, addType AddType, doChildren bool) error {
	reg.NewObject = obj

	if doChildren {
		colls := reg.GetCollections()
		for _, coll := range colls {
			plural := coll[0]
			singular := coll[1]

			collVal := obj[plural]
			if IsNil(collVal) {
				continue
			}
			collMap, ok := collVal.(map[string]any)
			if !ok {
				return fmt.Errorf("Attribute %q doesn't appear to be of a "+
					"map of %q", plural, plural)
			}
			for key, val := range collMap {
				valObj, ok := val.(map[string]any)
				if !ok {
					return fmt.Errorf("Key %q in attribute %q doesn't "+
						"appear to be of type %q", key, plural, singular)
				}
				_, _, err := reg.UpsertGroupWithObject(plural, key, valObj,
					addType, doChildren)
				if err != nil {
					return err
				}
			}
		}
	}

	if addType == ADD_PATCH {
		// Copy existing props over if the incoming obj doesn't set them
		for k, val := range reg.Object {
			if _, ok := reg.NewObject[k]; !ok {
				reg.NewObject[k] = val
			}
		}
	}

	// Make sure we always have an ID
	if IsNil(reg.NewObject["registryid"]) {
		reg.NewObject["registryid"] = reg.UID
	}

	return reg.ValidateAndSave()
}

func (reg *Registry) FindGroup(gType string, id string, anyCase bool) (*Group, error) {
	log.VPrintf(3, ">Enter: FindGroup(%s,%s,%v)", gType, id, anyCase)
	defer log.VPrintf(3, "<Exit: FindGroup")

	ent, err := RawEntityFromPath(reg.tx, reg.DbSID, gType+"/"+id, anyCase)
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
	g, _, err := reg.UpsertGroupWithObject(gType, id, nil, ADD_ADD, false)
	return g, err
}

func (reg *Registry) AddGroupWithObject(gType string, id string, obj Object, doChildren bool) (*Group, error) {
	g, _, err := reg.UpsertGroupWithObject(gType, id, obj, ADD_ADD, doChildren)
	return g, err
}

// *Group, isNew, error
func (reg *Registry) UpsertGroup(gType string, id string) (*Group, bool, error) {
	return reg.UpsertGroupWithObject(gType, id, nil, ADD_UPSERT, false)
}

func (reg *Registry) UpsertGroupWithObject(gType string, id string, obj Object, addType AddType, doChildren bool) (*Group, bool, error) {
	log.VPrintf(3, ">Enter UpsertGroupWithObject(%s,%s)", gType, id)
	defer log.VPrintf(3, "<Exit UpsertGroupWithObject")

	if reg.Model.Groups[gType] == nil {
		return nil, false, fmt.Errorf("Error adding Group, unknown type: %s",
			gType)
	}

	if id == "" {
		id = NewUUID()
	}

	g, err := reg.FindGroup(gType, id, true)
	if err != nil {
		return nil, false, fmt.Errorf("Error finding Group(%s) %q: %s",
			gType, id, err)
	}

	if g != nil && g.UID != id {
		return nil, false, fmt.Errorf("Attempting to create a Group "+
			"with a \"%sid\" of %q, when one already exists as %q",
			reg.Model.Groups[gType].Singular, id, g.UID)
	}
	if addType == ADD_ADD && g != nil {
		return nil, false, fmt.Errorf("Group %q of type %q already exists",
			id, gType)
	}

	isNew := (g == nil)
	if g == nil {
		// Not found, so create a new one
		g = &Group{
			Entity: Entity{
				tx: reg.tx,

				Registry: reg,
				DbSID:    NewUUID(),
				Plural:   gType,
				Singular: reg.Model.Groups[gType].Singular,
				UID:      id,

				Type:     ENTITY_GROUP,
				Path:     gType + "/" + id,
				Abstract: gType,
			},
			Registry: reg,
		}

		err = DoOne(reg.tx, `
			INSERT INTO "Groups"(SID,RegistrySID,UID,ModelSID,Path,Abstract)
			SELECT ?,?,?,SID,?,?
			FROM ModelEntities
			WHERE RegistrySID=? AND Plural=? AND ParentSID IS NULL`,
			g.DbSID, g.Registry.DbSID, g.UID, g.Path, g.Abstract,
			g.Registry.DbSID, g.Plural)

		if err != nil {
			err = fmt.Errorf("Error adding Group: %s", err)
			log.Print(err)
			return nil, false, err
		}

		// Use the ID passed as an arg, not from the metadata, as the true
		// ID. If the one in the metadata differs we'll flag it down below
		if err = g.JustSet(g.Singular+"id", g.UID); err != nil {
			return nil, false, err
		}
	}

	if isNew || obj != nil {
		if obj != nil {
			g.NewObject = obj
		}

		if addType == ADD_PATCH {
			// Copy existing props over if the incoming obj doesn't set them
			for k, v := range g.Object {
				if _, ok := g.NewObject[k]; !ok {
					g.NewObject[k] = v
				}
			}
		}

		// Make sure we always have an ID
		if IsNil(g.NewObject[g.Singular+"id"]) {
			g.NewObject[g.Singular+"id"] = g.UID
		}

		if err = g.ValidateAndSave(); err != nil {
			return nil, false, err
		}
	}

	if doChildren {
		colls := g.GetCollections()
		for _, coll := range colls {
			plural := coll[0]
			singular := coll[1]

			collVal := obj[plural]
			if IsNil(collVal) {
				continue
			}
			collMap, ok := collVal.(map[string]any)
			if !ok {
				return nil, false,
					fmt.Errorf("Attribute %q doesn't appear to be of a "+
						"map of %q", plural, plural)
			}
			for key, val := range collMap {
				valObj, ok := val.(map[string]any)
				if !ok {
					return nil, false,
						fmt.Errorf("Key %q in attribute %q doesn't "+
							"appear to be of type %q", key, plural, singular)
				}
				_, _, err := g.UpsertResourceWithObject(plural, key, "",
					valObj, addType, doChildren, false)
				if err != nil {
					return nil, false, err
				}
			}
		}
	}

	return g, isNew, nil
}

func GenerateQuery(reg *Registry, what string, paths []string, filters [][]*FilterExpr) (string, []interface{}, error) {
	query := ""
	args := []any{}

	args = []interface{}{reg.DbSID}
	query = `
SELECT
  RegSID,Type,Plural,Singular,eSID,UID,PropName,PropValue,PropType,Path,Abstract
FROM FullTree WHERE RegSID=?`

	// Remove entities that are higher than the GET PATH specified
	if what != "Registry" && len(paths) > 0 {
		query += "\nAND ("
		for i, p := range paths {
			if i > 0 {
				query += " OR "
			}
			query += "Path=? OR Path LIKE ?"
			args = append(args, p, p+"/%")
		}
		query += ")"

	}

	if len(filters) != 0 {
		query += `
AND
(
eSID IN ( -- eSID from query
  WITH RECURSIVE cte(eSID,ParentSID,Path) AS (
    SELECT eSID,ParentSID,Path FROM Entities
    WHERE eSID in ( -- start of the OR Filter groupings`
		firstOr := true
		for _, OrFilters := range filters {
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
				args = append(args, reg.DbSID, filter.Path)
				if filter.HasEqual {
					value, wildcard := WildcardIt(filter.Value)
					args = append(args, value)
					if !wildcard {
						check = "PropValue=?"
					} else {
						args = append(args, value)
						check = "((PropType<>'string' AND PropValue=?) OR (PropType='string' AND PropValue LIKE ?))"
					}
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

func WildcardIt(str string) (string, bool) {
	wild := false
	res := strings.Builder{}

	prevch := '\000'
	for _, ch := range str {
		if ch == '*' && prevch != '\\' {
			res.WriteRune('%')
			wild = true
		} else {
			res.WriteRune(ch)
		}
		prevch = ch
	}

	return res.String(), wild
}
