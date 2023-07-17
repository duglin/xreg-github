package registry

import (
	"database/sql"
	_ "embed"
	"fmt"
	"reflect"
	"strings"

	log "github.com/duglin/dlog"
	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

type Result struct {
	sqlRows  *sql.Rows
	colTypes []reflect.Type
	Data     []*any
	TempData []any
	Reuse    bool
}

func (r *Result) Close() {
	if r.sqlRows != nil {
		r.sqlRows.Close()
		r.sqlRows = nil
	}
	r.Data = nil
	r.TempData = nil
}

func (r *Result) Push() {
	if r.Reuse {
		panic("Already pushed")
	}
	r.Reuse = true
}

func (r *Result) NextRow() []*any {
	if r.Data == nil {
		return nil
	}

	if r.Reuse {
		r.Reuse = false
	} else {
		// check for error from PullNextRow
		r.PullNextRow()
	}

	return r.Data
}

func (r *Result) PullNextRow() {
	if r.sqlRows == nil {
		panic("sqlRows is nil")
	}
	if r.sqlRows.Next() == false {
		r.Close()
		return
	}

	err := r.sqlRows.Scan(r.TempData...) // Can't pass r.Data directly
	if err != nil {
		panic(fmt.Sprintf("Error scanning DB row: %s", err))
		// should return err.  r.Data = nil ; return err..
	}

	// Move data from TempData to Data
	for i, _ := range r.Data {
		r.Data[i] = r.TempData[i].(*any)
	}

	if log.GetVerbose() > 3 {
		dd := []string{}
		for _, d := range r.Data {
			if reflect.ValueOf(*d).Type().String() == "[]uint8" {
				dd = append(dd, string((*d).([]byte)))
			} else {
				dd = append(dd, fmt.Sprintf("%v", *d))
			}
		}
		log.VPrintf(4, "row: %v", dd)
	}
}

func Query(cmd string, args ...interface{}) (*Result, error) {
	if log.GetVerbose() > 3 {
		log.VPrintf(4, "Query: %s", SubQuery(cmd, args))
	}

	ps, err := DB.Prepare(cmd)
	if err != nil {
		log.Printf("Error Prepping query (%s)->%s\n", cmd, err)
		return nil, fmt.Errorf("Error Prepping query (%s)->%s\n", cmd, err)
	}

	rows, err := ps.Query(args...)
	if err != nil {
		log.Printf("Error querying DB(%s)->%s\n", cmd, err)
		return nil, fmt.Errorf("Error querying DB(%s)->%s\n", cmd, err)
	}

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		log.Printf("Error querying DB(%s)->%s\n", cmd, err)
		return nil, fmt.Errorf("Error querying DB(%s)->%s\n", cmd, err)
	}

	result := &Result{
		sqlRows:  rows,
		colTypes: []reflect.Type{},
	}

	for _, col := range colTypes {
		result.colTypes = append(result.colTypes, col.ScanType())
		result.Data = append(result.Data, new(any))
		result.TempData = append(result.TempData, new(any))
	}

	return result, nil
}

func Do(cmd string, args ...interface{}) error {
	log.VPrintf(4, "Do: %q arg: %v", cmd, args)
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
		log.Printf("DoOne:Error DB(%s) didn't change exactly 1 row", query)
	}

	return err
}

func DBExists(name string) bool {
	log.VPrintf(3, ">Enter: DBExists %q", name)
	db, err := sql.Open("mysql", "root:password@/")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT SCHEMA_NAME
		FROM INFORMATION_SCHEMA.SCHEMATA
		WHERE SCHEMA_NAME=?`, name)
	if err != nil {
		panic(err)
	}
	found := rows.Next()
	log.VPrintf(3, "<Exit: found: %v", found)
	return found
}

//go:embed init.sql
var initDB string

func OpenDB(name string) {
	log.VPrintf(3, ">Enter: OpenDB %q", name)
	defer log.VPrintf(3, "<Exit: OpenDB")

	// DB, err := sql.Open("mysql", "root:password@tcp(localhost:3306)/")
	var err error

	DB, err = sql.Open("mysql", "root:password@/"+name)
	if err != nil {
		err = fmt.Errorf("Error talking to SQL: %s\n", err)
		log.Print(err)
		panic(err)
		// return err
	}

	DB.SetMaxOpenConns(5)
	DB.SetMaxIdleConns(5)
}

func CreateDB(name string) error {
	log.VPrintf(3, ">Enter: CreateDB %q", name)
	defer log.VPrintf(3, "<Exit: CreateDB")

	db, err := sql.Open("mysql", "root:password@/")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if _, err = db.Exec("CREATE DATABASE " + name); err != nil {
		panic(err)
	}

	if _, err = db.Exec("USE " + name); err != nil {
		panic(err)
	}

	log.VPrintf(3, "Creating DB")

	for _, cmd := range strings.Split(initDB, ";") {
		cmd = strings.TrimSpace(cmd)
		cmd = strings.Replace(cmd, "@", ";", -1) // Can't use ; in file
		if cmd == "" {
			continue
		}

		log.VPrintf(4, "CMD: %s", cmd)
		if _, err := db.Exec(cmd); err != nil {
			panic(fmt.Sprintf("Error on: %s\n%s", cmd, err))
		}
	}

	return nil
}

func DeleteDB(name string) error {
	log.VPrintf(3, "Deleting DB %q", name)

	db, err := sql.Open("mysql", "root:password@/")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	_, err = db.Exec("DROP DATABASE IF EXISTS " + name)
	if err != nil {
		panic(err)
	}
	return nil
}

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
