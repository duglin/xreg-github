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

const RegOpt_TrackTimestamps = RegOpt("TRACK_TIMESTAMPS")

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
		tx = NewTx()
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

			DbSID:  dbSID,
			Plural: "registries",
			UID:    id,

			Level:    0,
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
	if err = reg.JustSet("id", reg.UID); err != nil {
		return nil, err
	}

	for _, regOpt := range regOpts {
		if regOpt == RegOpt_TrackTimestamps {
			if err = reg.TrackTimestamps(true); err != nil {
				return nil, err
			}
		}
	}

	if err = reg.SetSave("epoch", 1); err != nil {
		return nil, err
	}

	if err = reg.Model.VerifyAndSave(); err != nil {
		return nil, err
	}

	return reg, nil
}

func GetRegistryNames() []string {
	tx := NewTx()
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

func (reg *Registry) TrackTimestamps(val bool) error {
	if val {
		return reg.SetSave("#tracktimestamps", true)
	} else {
		return reg.SetSave("#tracktimestamps", nil)
	}
}

func (reg *Registry) Get(name string) any {
	return reg.Entity.Get(name)
}

// Technically this should be called SetValidateSave
func (reg *Registry) SetCommit(name string, val any) error {
	return reg.Entity.SetCommit(name, val)
}

func (reg *Registry) JustSet(name string, val any) error {
	return reg.Entity.JustSet(NewPPP(name), val)
}

func (reg *Registry) SetSave(name string, val any) error {
	return reg.Entity.SetSave(name, val)
}

func (reg *Registry) Delete() error {
	log.VPrintf(3, ">Enter: Reg.Delete(%s)", reg.UID)
	defer log.VPrintf(3, "<Exit: Reg.Delete")

	return DoOne(reg.tx, `DELETE FROM Registries WHERE SID=?`, reg.DbSID)
}

func FindRegistryBySID(tx *Tx, sid string) (*Registry, error) {
	log.VPrintf(3, ">Enter: FindRegistrySID(%s)", sid)
	defer log.VPrintf(3, "<Exit: FindRegistrySID")

	results, err := Query(tx, `SELECT UID FROM Registries WHERE SID=?`, sid)
	defer results.Close()

	if err != nil {
		return nil, fmt.Errorf("Error finding Registry %q: %s", sid, err)
	}

	row := results.NextRow()
	if row == nil {
		return nil, fmt.Errorf("Error finding Registry %q: no match", sid)
	}
	results.Close()

	uid := NotNilString(row[0])
	return FindRegistry(tx, uid)
}

// BY UID
func FindRegistry(tx *Tx, id string) (*Registry, error) {
	log.VPrintf(3, ">Enter: FindRegistry(%s)", id)
	defer log.VPrintf(3, "<Exit: FindRegistry")

	newTx := false
	if tx == nil {
		tx = NewTx()
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

	ent, err := RawEntityFromPath(tx, id, "")
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
	defer log.VPrintf(3, "<Exit:FindGroup")

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
		return err
	}

	buf, err = ProcessImports(file, buf, true)
	if err != nil {
		return err
	}

	model := &Model{
		Registry: reg,
	}

	if err := Unmarshal(buf, model); err != nil {
		return err
	}

	model.SetPointers()

	if err := model.Verify(); err != nil {
		return err
	}

	reg.Model.ApplyNewModel(model)

	// reg.Model = model
	// reg.Model.VerifyAndSave()
	return nil
}

func (reg *Registry) FindGroup(gType string, id string) (*Group, error) {
	log.VPrintf(3, ">Enter: FindGroup(%s/%s)", gType, id)
	defer log.VPrintf(3, "<Exit: FindGroup")

	ent, err := RawEntityFromPath(reg.tx, reg.DbSID, gType+"/"+id)
	if err != nil {
		return nil, fmt.Errorf("Error finding Group %q(%s): %s", id, gType, err)
	}
	if ent == nil {
		log.VPrintf(3, "None found")
		return nil, nil
	}

	return &Group{Entity: *ent, Registry: reg}, nil
}

func (reg *Registry) AddGroup(gType string, id string, objs ...Object) (*Group, error) {
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
			tx: reg.tx,

			Registry: reg,
			DbSID:    NewUUID(),
			Plural:   gType,
			UID:      id,

			Level:    1,
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
		return nil, err
	}

	if err = g.JustSet("id", g.UID); err != nil {
		return nil, err
	}

	for _, obj := range objs {
		for k, v := range obj {
			if err = g.JustSet(k, v); err != nil {
				return nil, err
			}
		}
	}

	if err = g.SetSave("epoch", 1); err != nil {
		return nil, err
	}

	log.VPrintf(3, "Created new one - DbSID: %s", g.DbSID)
	return g, nil
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
