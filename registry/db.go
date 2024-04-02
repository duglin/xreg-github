package registry

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"runtime/pprof"
	"strings"
	"time"

	log "github.com/duglin/dlog"
	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

var DBHOST = "localhost"
var DBPORT = "3306"

func init() {
	if tmp := os.Getenv("DBHOST"); tmp != "" {
		DBHOST = tmp
	}
	if tmp := os.Getenv("DBPORT"); tmp != "" {
		DBPORT = tmp
	}
}

// Active transaction - mainly for debugging and testing
var TXs = map[string]*Tx{}

func DumpTXs() {
	// Only show info if there are active Txs
	if len(TXs) == 0 {
		return
	}

	count := 1
	for _, t := range TXs {
		log.Printf("NewTx Stack %d:", count)
		for _, s := range t.stack {
			log.Printf("  %s", s)
		}
		count++
	}

	// Show threads/processes
	pprof.Lookup("goroutine").WriteTo(PProfFilter, 1)

	log.Printf("==========================")
	log.Printf("")
	PProfFilter.count = 0
	PProfFilter.inSection = false
	PProfFilter.buffer.Reset()
}

var PProfFilter = &FilterPProf{}

type FilterPProf struct {
	buffer    bytes.Buffer
	count     int
	inSection bool
}

// Extract func name and files info
var fpRE = regexp.MustCompile(`^#\s+[^\s]+\s+[^.]*.[^.]*\.([^+]*)\+[^/]*.*/(.*)$`)

// When dumping all processes, filter out the ones that aren't running our
// code and only show lines of interest to keep it small
func (fp *FilterPProf) Write(p []byte) (n int, err error) {
	for _, b := range p {
		if b == '\n' {
			line := fp.buffer.String()
			fp.buffer.Reset()

			if strings.Contains(line, "xreg-github") &&
				!strings.Contains(line, "(*Server).Serve+") &&
				!strings.Contains(line, "(*Server).Serve.") &&
				!strings.Contains(line, "TestMain") {

				if !fp.inSection {
					fp.inSection = true
					fp.count++
					log.Printf("Thread %d:", fp.count)
				}

				line = fpRE.ReplaceAllString(line, `  $1   $2`)
				log.Printf(line)
			} else {
				fp.inSection = false
			}
		} else {
			fp.buffer.WriteByte(b)
		}
	}
	return len(p), nil
}

// Holds info about the current transaction. In a lot of ways this is similar
// to golang's Context in that it holds other info related to the current
// changes that are going on. Maybe one day convert this to a Context where
// Tx is just as apsect of it.
type Tx struct {
	tx         *sql.Tx
	Registry   *Registry
	CreateTime string // use for entity timestamps too
	User       string

	// Cache of entities this Tx is dealing with. Things can get funky if
	// we have more than one instance of the same entity in memory.
	// TODO DUG expand this to save all types, not just Versions.
	// Also, consider having Commit() just automatically call ValidateAndSave
	// for all entities in the Tx - then people don't need to call save
	// explicitly
	// Registries map[string]*Registry // reg.UID
	// Groups     map[string]*Group    // reg.DbSID+g.UID
	// Resources  map[string]*Resource // reg.DbSID+g.DbSID+r.UID
	Versions map[string]*Version // reg.DbSID+g.DbSID+r.DbSID+v.UID

	// For debugging
	uuid  string   // just a unique ID for the TXs map key
	stack []string // Stack at time NewTX
}

func (tx *Tx) String() string {
	regStr := "<none>"
	if tx.Registry != nil {
		regStr = tx.Registry.DbSID
	}

	txStr := "<none>"
	if tx.tx != nil {
		txStr = "<set>"
	}
	return fmt.Sprintf("Tx: sql.tx: %s, Registry: %s", txStr, regStr)
}

func NewTx() *Tx {
	log.VPrintf(4, ">Enter: NewTx")
	defer log.VPrintf(4, "<exit: NewTx")

	tx := &Tx{}
	tx.NewTx()
	return tx
}

// It's ok for this to be called multiple times for the same Tx just to
// make sure we have an active transaction - it's a no-op at that point
func (tx *Tx) NewTx() {
	log.VPrintf(4, ">Enter: tx.NewTx")
	defer log.VPrintf(4, "<Exit: tx.NewTx")

	if tx.tx != nil {
		return
	}

	t, err := DB.BeginTx(context.Background(),
		&sql.TxOptions{sql.LevelReadCommitted, false})
	Must(err)

	tx.tx = t
	// if TESTING {
	tx.CreateTime = time.Now().Format(time.RFC3339Nano)
	// } else {
	// tx.CreateTime = time.Now().Format(time.RFC3339)
	// }
	tx.Versions = map[string]*Version{}
	tx.uuid = NewUUID()
	tx.stack = GetStack()
	TXs[tx.uuid] = tx
}

func (tx *Tx) Commit() error {
	if tx.tx == nil {
		return nil
	}
	err := tx.tx.Commit()
	Must(err)
	if err != nil {
		return err
	}

	delete(TXs, tx.uuid)
	tx.tx = nil
	tx.CreateTime = ""
	tx.Versions = nil // force a NPE if someone tries to use it outside of a tx
	tx.uuid = ""

	return nil
}

func (tx *Tx) Rollback() error {
	if tx.tx == nil {
		return nil
	}
	err := tx.tx.Rollback()
	Must(err)
	if err != nil {
		return err
	}

	delete(TXs, tx.uuid)
	tx.tx = nil
	tx.CreateTime = ""
	tx.Versions = nil // force a NPE if someone tries to use it outside of a tx
	tx.uuid = ""

	return nil
}

func (tx *Tx) Conditional(err error) error {
	if err == nil {
		return tx.Commit()
	}
	return tx.Rollback()
}

func (tx *Tx) Prepare(query string) (*sql.Stmt, error) {
	// If the current Tx is closed, create a new one
	if tx.tx == nil {
		tx.NewTx()
	}
	ps, err := tx.tx.Prepare(query)

	return ps, err
}

func (tx *Tx) AddVersion(v *Version) {
	if tx.Versions == nil {
		tx.Versions = map[string]*Version{}
	}
	tx.Versions[v.Resource.Group.Registry.DbSID+
		v.Resource.Group.DbSID+
		v.Resource.DbSID+
		v.UID] = v
}

func (tx *Tx) GetVersion(r *Resource, vID string) *Version {
	key := r.Group.Registry.DbSID + r.Group.DbSID + r.DbSID + vID
	return tx.Versions[key]
}

type Result struct {
	tx       *Tx
	sqlRows  *sql.Rows
	colTypes []reflect.Type
	Data     []*any // One row
	TempData []any
	Reuse    bool

	AllRows [][]*any
}

func (r *Result) Close() {
	if r == nil {
		return
	}

	if r.Data == nil {
		// Already done
		return
	}

	if r.tx != nil {
		r.tx = nil
	}

	if r.sqlRows != nil {
		r.sqlRows.Close()
		r.sqlRows = nil
	}

	r.Data = nil
	r.TempData = nil
	r.AllRows = nil
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
	if r.AllRows == nil || len(r.AllRows) == 0 {
		r.Close()
		return
	}

	r.Data = r.AllRows[0]
	r.AllRows = r.AllRows[1:]

	if log.GetVerbose() > 3 {
		dd := []string{}
		for _, d := range r.Data {
			dVal := reflect.ValueOf(*d)
			if !IsNil(*d) && dVal.Type().String() == "[]uint8" {
				// if reflect.ValueOf(*d).Type().String() == "[]uint8"
				dd = append(dd, string((*d).([]byte)))
			} else {
				dd = append(dd, fmt.Sprintf("%v", *d))
			}
		}
		log.VPrintf(4, "row: %v", dd)
	}
}

func (r *Result) RetrieveAllRowsFromDB() {
	for {
		if r.RetrieveNextRowFromDB() == false {
			break
		}
		r.AllRows = append(r.AllRows, r.Data)
	}
	// When done, technically r.Data contains the last item from the query
	// but it'll be overwritten on the first call to PullNextRow

	// Close the MYSQL query and prepare stmt
	if r.sqlRows != nil {
		r.sqlRows.Close()
		r.sqlRows = nil
	}
}

func (r *Result) RetrieveNextRowFromDB() bool {
	if r.sqlRows == nil {
		panic("sqlRows is nil")
	}
	if r.sqlRows.Next() == false {
		// r.Close()
		return false
	}

	r.TempData = make([]any, len(r.TempData))
	r.Data = make([]*any, len(r.Data))
	for i, _ := range r.TempData {
		r.TempData[i] = new(any)
		r.Data[i] = r.TempData[i].(*any)
	}

	err := r.sqlRows.Scan(r.TempData...) // Can't pass r.Data directly
	if err != nil {
		panic(fmt.Sprintf("Error scanning DB row: %s", err))
		// should return err.  r.Data = nil ; return err..
	}

	// Move data from TempData to Data

	if log.GetVerbose() >= 4 {
		dd := []string{}
		for _, d := range r.Data {
			dVal := reflect.ValueOf(*d)
			if !IsNil(*d) && dVal.Type().String() == "[]uint8" {
				// if reflect.ValueOf(*d).Type().String() == "[]uint8"
				dd = append(dd, string((*d).([]byte)))
			} else {
				dd = append(dd, fmt.Sprintf("%v", *d))
			}
		}
		log.Printf("row: %v", dd)
	}
	return true
}

func Query(tx *Tx, cmd string, args ...interface{}) (*Result, error) {
	if log.GetVerbose() >= 4 {
		log.Printf("Query: %s", SubQuery(cmd, args))
	}

	ps, err := tx.Prepare(cmd)
	if err != nil {
		log.Printf("Error Prepping query (%s)->%s\n", cmd, err)
		return nil, fmt.Errorf("Error Prepping query (%s)->%s\n", cmd, err)
	}
	defer ps.Close()

	rows, err := ps.Query(args...)
	if err != nil {
		log.Printf("Error querying DB(%s)(%v)->%s\n", cmd, args, err)
		return nil, fmt.Errorf("Error querying DB(%s)->%s\n", cmd, err)
	}

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		log.Printf("Error querying DB(%s)(%v)->%s\n", cmd, args, err)
		return nil, fmt.Errorf("Error querying DB(%s)->%s\n", cmd, err)
	}

	result := &Result{
		tx:       tx,
		sqlRows:  rows,
		colTypes: []reflect.Type{},
	}

	for _, col := range colTypes {
		result.colTypes = append(result.colTypes, col.ScanType())
		result.Data = append(result.Data, new(any))
		result.TempData = append(result.TempData, new(any))
	}

	// Download all data. We used to pull from DB on each PullNextRow
	// but mysql doesn't support multiple queries being active in the same Tx
	result.RetrieveAllRowsFromDB()

	return result, nil
}

func doCount(tx *Tx, cmd string, args ...interface{}) (int, error) {
	log.VPrintf(4, "doCount: %q arg: %v", cmd, args)
	ps, err := tx.Prepare(cmd)
	if err != nil {
		ShowStack()
		return 0, err
	}
	defer ps.Close()

	result, err := ps.Exec(args...)
	if err != nil {
		query := SubQuery(cmd, args)
		log.Printf("doCount:Error DB(%s)->%s\n", query, err)
		ShowStack()
		return 0, err
	}

	count, _ := result.RowsAffected()
	return int(count), err
}

func Do(tx *Tx, cmd string, args ...interface{}) error {
	_, err := doCount(tx, cmd, args...)
	return err
}

func DoOne(tx *Tx, cmd string, args ...interface{}) error {
	count, err := doCount(tx, cmd, args...)
	if err != nil {
		return err
	}

	if count != 1 {
		query := SubQuery(cmd, args)
		ShowStack()
		log.Printf("DoOne:Error DB(%s) didn't change exactly 1 row(%d)",
			query, count)
		return fmt.Errorf("DoOne:Error DB(%s) didn't change exactly 1 row(%d)",
			query, count)
	}

	return nil
}

func DoZeroOne(tx *Tx, cmd string, args ...interface{}) error {
	count, err := doCount(tx, cmd, args...)
	if err != nil {
		return err
	}

	if count != 0 && count != 1 {
		query := SubQuery(cmd, args)
		ShowStack()
		log.Printf("DoZeroOne:Error DB(%s) didn't change exactly 0/1 rows(%d)",
			query, count)
		return fmt.Errorf("DoZeroOne:Error DB(%s) didn't change exactly 0/1 rows(%d)",
			query, count)
	}

	return nil
}

func DoOneTwo(tx *Tx, cmd string, args ...interface{}) error {
	count, err := doCount(tx, cmd, args...)
	if err != nil {
		return err
	}

	if count != 1 && count != 2 {
		query := SubQuery(cmd, args)
		ShowStack()
		log.Printf("DoOneTwo:Error DB(%s) didn't change exactly 1/2 rows(%d)",
			query, count)
		return fmt.Errorf("DoOneTwo:Error DB(%s) didn't change exactly 1/2 rows(%d)",
			query, count)
	}

	return nil
}

func DoZeroTwo(tx *Tx, cmd string, args ...interface{}) error {
	count, err := doCount(tx, cmd, args...)
	if err != nil {
		return err
	}

	if count != 0 && count != 2 {
		query := SubQuery(cmd, args)
		ShowStack()
		log.Printf("DoZeroTwo:Error DB(%s) didn't change exactly 0/2 rows(%d)",
			query, count)
		return fmt.Errorf("DoZeroTwo:Error DB(%s) didn't change exactly 0/2 rows(%d)",
			query, count)
	}

	return nil
}

func DoCount(tx *Tx, num int, cmd string, args ...interface{}) error {
	log.VPrintf(4, "DoCount: %s", cmd)
	count, err := doCount(tx, cmd, args...)
	if err != nil {
		return err
	}

	if count != num {
		query := SubQuery(cmd, args)
		ShowStack()
		log.Printf("DoCount:Error DB(%s) didn't change exactly %d rows(%d)",
			query, num, count)
		return fmt.Errorf("DoCount:Error DB(%s) didn't change exactly %d rows(%d)",
			query, num, count)
	}

	return nil
}

func DBExists(name string) bool {
	log.VPrintf(3, ">Enter: DBExists %q", name)
	defer log.VPrintf(3, "<Exit: DBExists")
	db, err := sql.Open("mysql", "root:password@tcp("+DBHOST+":"+DBPORT+")/")
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
var firstTime = true

func OpenDB(name string) {
	if firstTime {
		log.VPrintf(1, "DB: %s:%s", DBHOST, DBPORT)
		firstTime = false
	}

	log.VPrintf(3, ">Enter: OpenDB %q", name)
	defer log.VPrintf(3, "<Exit: OpenDB")

	// DB, err := sql.Open("mysql", "root:password@tcp(localhost:3306)/")
	var err error

	DB, err = sql.Open("mysql", "root:password@tcp("+DBHOST+":"+DBPORT+")/"+name)
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

	db, err := sql.Open("mysql", "root:password@tcp("+DBHOST+":"+DBPORT+")/")
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

	db, err := sql.Open("mysql", "root:password@tcp("+DBHOST+":"+DBPORT+")/")
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

func SubQuery(query string, args []interface{}) string {
	argNum := 0

	for pos := 0; pos < len(query); pos++ {
		if ch := query[pos]; ch != '?' {
			continue
		}
		if argNum >= len(args) {
			panic(fmt.Sprintf("Extra ? in query at %q", query[pos:]))
		}

		val := fmt.Sprintf("%v", args[argNum])
		query = fmt.Sprintf("%s'%s'%s", query[:pos], val, query[pos+1:])
		pos += len(val) + 1 // one more will be added due to pos++
		argNum++
	}
	if argNum != len(args) {
		panic(fmt.Sprintf("Too many args passed into %q", query))
	}
	return query
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
	  WHERE (PropName='labels.int' AND PropValue=3 AND Level=2)
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
  WHERE (PropName='labels.int' AND PropValue=3 AND Level=2)
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
		WHERE (PropName='labels.int' AND PropValue=3 AND Level=2)
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
		  WHERE (CONCAT(Abstract,'.',PropName)='myGroups/ress.labels.int')
		  UNION ALL
	      SELECT eID,Path FROM FullTree
		  WHERE (PropName='labels.int' AND PropValue=3 AND Level=2)
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
          WHERE (PropName='latestVersionId' AND PropValue='v1.0' AND Level=2)
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
