package registry

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

type Prep struct {
	Stmt *sql.Stmt
	Cmd  string
}

func Prepare(str string) (*Prep, error) {
	ps, err := DB.Prepare(str)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error DB.Prepare(%s)->%s\n", str, err)
		return nil, err
	}
	return &Prep{ps, str}, nil
}

func (p *Prep) Exec(args ...interface{}) error {
	_, err := p.Stmt.Exec(args...)
	if err != nil {
		query := SubQuery(p.Cmd, args)
		fmt.Fprintf(os.Stderr, "Exec:Error DB(%s)->%s\n", query, err)
	}
	return err
}

type Result struct {
	sqlRows  *sql.Rows
	colTypes []reflect.Type
	Data     []*any
}

func (r *Result) Close() {
	if r.sqlRows != nil {
		r.sqlRows.Close()
	}
	r.sqlRows = nil
}

func (r *Result) NextRow() bool {
	if r.sqlRows == nil {
		// Should never be here
		return false
	}
	if r.sqlRows.Next() == false {
		return false
	}

	d := []any{}
	for range r.colTypes {
		d = append(d, new(any))
	}

	err := r.sqlRows.Scan(d...)
	if err != nil {
		log.Printf("Error scanning DB row: %s", err)
		return false
	}

	// Can't explain this but it works
	for i, _ := range d {
		r.Data[i] = d[i].(*any)
	}

	return true
}

func NewQuery(cmd string, args ...interface{}) ([][]*any, error) {
	result, err := Query(cmd, args...)
	if err != nil {
		return nil, err
	}

	data := [][]*any{}
	for result.NextRow() {
		newRow := make([]*any, len(result.Data))
		copy(newRow, result.Data)
		data = append(data, newRow)
	}

	return data, nil
}

func Query(cmd string, args ...interface{}) (*Result, error) {
	ps, err := DB.Prepare(cmd)
	if err != nil {
		log.Printf("Error Prepping query (%s)->%s\n", cmd, err)
		return nil, fmt.Errorf("Error Prepping query (%s)->%s\n", cmd, err)
	}
	rows, err := ps.Query(args...)

	if err != nil {
		log.Printf("Error Querying DB(%s)->%s\n", cmd, err)
		return nil, fmt.Errorf("Error Querying DB(%s)->%s\n", cmd, err)
	}

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		log.Printf("Error Querying DB(%s)->%s\n", cmd, err)
		return nil, fmt.Errorf("Error Querying DB(%s)->%s\n", cmd, err)
	}

	r := &Result{
		sqlRows:  rows,
		colTypes: []reflect.Type{},
	}

	for _, col := range colTypes {
		r.colTypes = append(r.colTypes, col.ScanType())
		r.Data = append(r.Data, new(any))
	}

	return r, nil
}

func Do(cmd string, args ...interface{}) error {
	ps, err := DB.Prepare(cmd)
	if err != nil {
		return err
	}
	_, err = ps.Exec(args...)

	if err != nil {
		query := SubQuery(cmd, args)
		log.Printf("Do:Error DB(%s)->%s\n", query, err)
	}
	return err
}

func DoOne(cmd string, args ...interface{}) error {
	ps, err := DB.Prepare(cmd)
	if err != nil {
		return err
	}
	result, err := ps.Exec(args...)

	if err != nil {
		query := SubQuery(cmd, args)
		log.Printf("DoOne:Error DB(%s)->%s\n", query, err)
		return err
	}

	if count, _ := result.RowsAffected(); count != 1 {
		query := SubQuery(cmd, args)
		log.Printf("DoOne:Error DB(%s) didn't change any rows", query)
	}

	return err
}

func MustDo(cmd string) {
	_, err := DB.Exec(cmd)
	if err != nil {
		log.Printf("Error on: %s", cmd)
		log.Fatalf("%s", err)
	}
}

func IgnoreDo(cmd string) {
	_, err := DB.Exec(cmd)
	if err != nil {
		log.Printf("Ignoring error on: %s", cmd)
		log.Printf("%s", err)
	}
}

//go:embed init.sql
var initDB string

/*
select * from FullTree
where
  eID in (
    select gID from FullTree
	where PropName='Name' and PropValue='docker.com' and Path='apiProviders/7fbc05b2'
    union select rID from FullTree
	where PropName='Name' and PropValue='docker.com' and Path='apiProviders/7fbc05b2'
	union select vID from FullTree
	where PropName='Name' and PropValue='docker.com' and Path='apiProviders/7fbc05b2'
  )
  order by Path ;


Children:
select ft.* from FullTree as ft where ft.Path like concat((select Path from FullTree where PropValue=4 and LevelNum=2),'/%') order by ft.Path ;

Node+Children:
select ft.* from FullTree as ft where ft.Path like concat((select Path from FullTree where PropValue=4 and LevelNum=2),'%') order by ft.Path ;

Parents:
select ft.* from FullTree as ft where (select Path from FullTree where PropValue=4 and LevelNum=2) like concat(ft.Path, '/%') order by ft.Path;

Node+Parents:
select ft.* from FullTree as ft where (select Path from FullTree where PropValue=4 and LevelNum=2) like concat(ft.Path, '%') order by ft.Path;



NODES + Children:
select ft2.* from FullTree as ft right JOIN FullTree as ft2 on(ft2.Path like concat(ft.Path, '%')) where (ft.PropValue=3 and ft.LevelNum=2) or (ft.PropValue=4 and ft.LevelNum=3) group by ft2.eID,ft2.PropName Order by ft2.Path;

PARENTS (not NODES):
select ft2.* from FullTree as ft right JOIN FullTree as ft2 on(ft.Path like concat(ft2.Path,'/%')) where (ft.PropValue=3 and ft.LevelNum=2) or (ft.PropValue=4 and ft.LevelNum=3) group by ft2.eID,ft2.PropName Order by ft2.Path;

( ( exp1 AND expr2 ...) or ( expr3 AND expr4 ) )
Find IDs that match expr1 OR expr2
SELECT eID FROM FullTree WHERE ( (expr1) OR (expr2) );
SELECT eID FROM FullTree WHERE (Level=2 AND PropName='epoch' && PropValue='4');

Given an ID find all Parents (include original ID)
WITH RECURSIVE cte(eID,ParentID,Path) AS (
  SELECT eID,ParentID,Path FROM Entities
  WHERE eID in (
    -- below find IDs of interes
	SELECT eID FROM FullTree
	  WHERE (PropName='tags.int' AND PropValue=3 AND Level=2)
    -- end of ID selection
  )
  UNION ALL SELECT e.eID,e.ParentID,e.Path FROM Entities AS e
  INNER JOIN cte ON e.eID=cte.ParentID)
SELECT * FROM cte ;

Given an ID find all Leaves (with recursion)
WITH RECURSIVE cte(eID,ParentID,Path) AS (
  SELECT eID,ParentID,Path FROM Entities
  WHERE eID='f91a4ec9'
  UNION ALL SELECT e.eID,e.ParentID,e.Path FROM Entities AS e
    INNER JOIN cte ON e.ParentID=cte.eID)
SELECT eID,ParentID,Path FROM cte
WHERE eID IN (SELECT * FROM Leaves);

Given an ID find all Leaves (w/o recursion)
  Should use IDs instead of Path to pick-up the Registry itself
SELECT e2.eID,e2.ParentID,e2.Path FROM Entities AS e1
RIGHT JOIN Entities AS e2 ON (e2.Path=e1.Path OR e2.Path LIKE
CONCAT(e1.Path,'%')) WHERE e1.eID in (
  -- below finds IDs of interest
  SELECT eID FROM FullTree
  WHERE (PropName='tags.int' AND PropValue=3 AND Level=2)
  -- end of ID selection
  )
AND e2.eID IN (SELECT * from Leaves);

Given an ID, find all leaves, and then find all Parents
-- Finding all parents
WITH RECURSIVE cte(eID,ParentID,Path) AS (
  SELECT eID,ParentID,Path FROM Entities
  WHERE eID in (
    -- below find IDs of interest (finding all leaves)
	SELECT e2.eID FROM Entities AS e1
	RIGHT JOIN Entities AS e2 ON (
	  e2.RegID=e1.RegID AND
	  (e2.Path=e1.Path OR e2.Path LIKE CONCAT(e1.Path,'%'))
	)
	WHERE e1.eID in (
	  -- below finds SeachNodes/IDs of interest
	  -- Add regID into the search
	    SELECT eID FROM FullTree
		WHERE (PropName='tags.int' AND PropValue=3 AND Level=2)
	  -- end of ID selection
	)
	AND e2.eID IN (SELECT * from Leaves)
    -- end of Leaves/ID selection
  )
  UNION ALL SELECT e.eID,e.ParentID,e.Path FROM Entities AS e
  INNER JOIN cte ON e.eID=cte.ParentID)
SELECT * FROM cte ;

(expr1 AND expr2)
WITH RECURSIVE cte(eID,ParentID,Path) AS (
  SELECT eID,ParentID,Path FROM Entities
  WHERE eID in (
    -- below find IDs of interest (finding all leaves)
	-- start of (expr1 and expr2 and expr3)
	SELECT list.eID FROM (
	  SELECT count(*) as cnt,e2.eID,e2.Path FROM Entities AS e1
	  RIGHT JOIN (
	    -- below finds SeachNodes/IDs of interest
	    -- Add regID into the search
	      SELECT eID,Path FROM FullTree
		  WHERE (CONCAT(Abstract,'.',PropName)='myGroups/ress.tags.int')
		  UNION ALL
	      SELECT eID,Path FROM FullTree
		  WHERE (PropName='tags.int' AND PropValue=3 AND Level=2)
		  UNION ALL
		  SELECT eID,Path from FullTree
		  WHERE (PropName='id' AND PropValue='g1' AND Level=1)
	    -- end of ID selection
	  ) as res ON ( res.eID=e1.eID )
	  JOIN Entities AS e2 ON (
	    (e2.Path=res.Path OR e2.Path LIKE CONCAT(res.Path,'%'))
	    AND e2.eID IN (SELECT * from Leaves)
	  ) GROUP BY e2.eID
      -- end of Leaves/ID selection
    ) as list
    WHERE list.cnt=3
	-- end of (expr1 and expr2 and expr3)

	-- ADD the next OR expr here
	UNION
	-- start of expr4
    SELECT list.eID FROM (
      SELECT count(*) as cnt,e2.eID,e2.Path FROM Entities AS e1
      RIGHT JOIN (
        -- below finds SeachNodes/IDs of interest
        -- Add regID into the search
          SELECT eID,Path FROM FullTree
          WHERE (PropName='latestId' AND PropValue='v1.0' AND Level=2)
        -- end of ID selection
      ) as res ON ( res.eID=e1.eID )
      JOIN Entities AS e2 ON (
        (e2.Path=res.Path OR e2.Path LIKE CONCAT(res.Path,'%'))
        AND e2.eID IN (SELECT * from Leaves)
      ) GROUP BY e2.eID
      -- end of Leaves/ID selection
    ) as list
    WHERE list.cnt=1
  )
  UNION ALL SELECT e.eID,e.ParentID,e.Path FROM Entities AS e
  INNER JOIN cte ON e.eID=cte.ParentID)
SELECT * FROM cte ;


*/

func init() {
	// DB, err := sql.Open("mysql", "root:password@tcp(localhost:3306)/")
	var err error

	DB, err = sql.Open("mysql", "root:password@/")
	if err != nil {
		log.Fatalf("Error talking to SQL: %s\n", err)
	}
	DB.SetMaxOpenConns(5)
	DB.SetMaxIdleConns(5)

	for _, cmd := range strings.Split(initDB, ";") {
		cmd = strings.TrimSpace(cmd)
		cmd = strings.Replace(cmd, "@", ";", -1)
		if cmd == "" {
			continue
		}
		log.VPrintf(4, "CMD: %s", cmd)
		if cmd[0] == '-' {
			IgnoreDo(cmd[1:])
		} else {
			MustDo(cmd)
		}
	}
	log.VPrintf(2, "Done init'ing DB")
}

type Entity struct {
	RegistryID string
	DbID       string
	Plural     string
	ID         string
	Extensions map[string]any
}

func (e *Entity) Get(name string) any {
	val, _ := e.Extensions[name]
	log.VPrintf(4, "%s(%s).Get(%s) -> %v", e.Plural, e.ID, name, val)
	return val
}

func (e *Entity) Find() (bool, error) {
	log.VPrintf(3, ">Enter: Find(%s)", e.ID)
	log.VPrintf(3, "<Exit: Find")

	results, err := NewQuery(`
		SELECT
			p.RegistryID AS RegistryID,
			p.EntityID AS DbID,
			e.Plural AS Plural,
			e.ID AS ID,
			p.PropName AS PropName,
			p.PropValue AS PropValue,
			p.PropType AS PropType
		FROM Props AS p
		LEFT JOIN Entities AS e ON (e.eID=p.EntityID)
		WHERE e.ID=?`, e.ID)

	if err != nil {
		return false, err
	}

	first := true
	for _, row := range results {
		if first {
			e.RegistryID = NotNilString(row[0])
			e.DbID = NotNilString(row[1])
			e.Plural = NotNilString(row[2])
			e.ID = NotNilString(row[3])
			first = false
		}
	}

	return !first, nil
}

func (e *Entity) Refresh() error {
	log.VPrintf(3, ">Enter: Refresh(%s)", e.DbID)
	defer log.VPrintf(3, "<Exit: Refresh")

	result, err := Query(`
        SELECT PropName, PropValue, PropType
        FROM Props WHERE EntityID=? `, e.DbID)
	defer result.Close()

	if err != nil {
		log.Printf("Error refreshing props(%s): %s", e.DbID, err)
		return fmt.Errorf("Error refreshing props(%s): %s", e.DbID, err)
	}

	e.Extensions = map[string]any{}

	for result.NextRow() {
		name := NotNilString(result.Data[0])
		val := NotNilString(result.Data[1])
		propType := NotNilString(result.Data[2])

		if propType == "s" {
			e.Extensions[name] = val
		} else if propType == "b" {
			e.Extensions[name] = (val == "true")
		} else if propType == "i" {
			tmpInt, err := strconv.Atoi(val)
			if err != nil {
				panic(fmt.Sprintf("error parsing int: %s", val))
			}
			e.Extensions[name] = tmpInt
		} else if propType == "f" {
			tmpFloat, err := strconv.ParseFloat(val, 64)
			if err != nil {
				panic(fmt.Sprintf("error parsing float: %s", val))
			}
			e.Extensions[name] = tmpFloat
		} else {
			panic(fmt.Sprintf("bad type: %v", propType))
		}
	}
	return nil
}

func (e *Entity) sSet(name string, val any) error {
	log.VPrintf(3, ">Enter: SetProp(%s=%v)", name, val)
	defer log.VPrintf(3, "<Exit SetProp")

	if e.DbID == "" {
		log.Fatalf("DbID should not be empty")
	}
	if e.RegistryID == "" {
		log.Fatalf("RegistryID should not be empty")
	}

	var err error
	if val == nil {
		err = Do(`DELETE FROM Props WHERE EntityID=? and PropName=?`,
			e.DbID, name)
	} else {
		propType := ""
		k := reflect.ValueOf(val).Type().Kind()
		if k == reflect.Bool {
			propType = "b" // boolean
		} else if k == reflect.String {
			propType = "s" // string
		} else if k == reflect.Int {
			propType = "i" // int
		} else if k == reflect.Float64 {
			propType = "f" // float
		} else {
			panic(fmt.Sprintf("Bad property kind: %s", k.String()))
		}
		err = Do(`
			REPLACE INTO Props( 
				RegistryID, EntityID, PropName, PropValue, PropType)
			VALUES( ?,?,?,?,? )`,
			e.RegistryID, e.DbID, name, val, propType)
	}

	if err != nil {
		log.Printf("Error updating prop(%s/%v): %s", name, val, err)
		return fmt.Errorf("Error updating prop(%s/%v): %s", name, val, err)
	}
	return nil
	// return SetProp(e, name, val)
}

func SetProp(entity any, name string, val any) error {
	log.VPrintf(3, ">Enter: SetProp(%s=%v)", name, val)
	defer log.VPrintf(3, "<Exit SetProp")

	eField := reflect.ValueOf(entity).Elem().FieldByName("Entity")
	e := (*Entity)(nil)
	if !eField.IsValid() {
		panic(fmt.Sprintf("Passing a non-entity to SetProp: %#v", entity))
		// e = entity.(*Entity)
	} else {
		e = eField.Addr().Interface().(*Entity)
	}

	if e.DbID == "" {
		log.Fatalf("DbID should not be empty")
	}
	if e.RegistryID == "" {
		log.Fatalf("RegistryID should not be empty")
	}

	var err error
	if val == nil {
		err = Do(`DELETE FROM Props WHERE EntityID=? and PropName=?`,
			e.DbID, name)
	} else {
		dbVal := val
		propType := ""
		k := reflect.ValueOf(val).Type().Kind()
		if k == reflect.Bool {
			propType = "b" // boolean
			if val.(bool) {
				dbVal = "true"
			} else {
				dbVal = "false"
			}
		} else if k == reflect.String {
			propType = "s" // string
		} else if k == reflect.Int {
			propType = "i" // int
		} else if k == reflect.Float64 {
			propType = "f" // float
		} else {
			panic(fmt.Sprintf("Bad property kind: %s", k.String()))
		}
		err = Do(`
			REPLACE INTO Props( 
				RegistryID, EntityID, PropName, PropValue, PropType)
			VALUES( ?,?,?,?,? )`,
			e.RegistryID, e.DbID, name, dbVal, propType)
	}

	if err != nil {
		log.Printf("Error updating prop(%s/%v): %s", name, val, err)
		return fmt.Errorf("Error updating prop(%s/%v): %s", name, val, err)
	}

	field := reflect.ValueOf(entity).Elem().FieldByName(name)
	if !field.IsValid() {
		field := reflect.ValueOf(e).Elem().FieldByName("Extensions")
		if !field.IsValid() {
			log.VPrintf(2, "Can't Set unknown field(%s/%s)", e.DbID, name)
		} else {
			if val == nil {
				if field.IsNil() {
					return nil
				}
				field.SetMapIndex(reflect.ValueOf(name), reflect.Value{})
			} else {
				if field.IsNil() {
					field.Set(reflect.ValueOf(map[string]any{}))
				}
				//tmp := fmt.Sprint(val)
				//field.SetMapIndex(reflect.ValueOf(name), reflect.ValueOf(tmp))
				field.SetMapIndex(reflect.ValueOf(name), reflect.ValueOf(val))
			}
		}
	} else {
		if val == nil {
			field.SetZero()
		} else {
			field.Set(reflect.ValueOf(val))
		}
	}

	return nil
}
