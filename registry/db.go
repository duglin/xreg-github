package registry

import (
	"database/sql"
	"fmt"
	"os"
	"reflect"
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
		fmt.Fprintf(os.Stderr, "Error DB(%s)->%s\n", p.Cmd, err)
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

func QueryRow(cmd string, args ...interface{}) (*Result, error) {
	res, err := Query(cmd, args...)
	defer res.Close()
	if err != nil {
		return nil, err
	}

	// Read the first row of data and then stop
	if !res.NextRow() {
		return nil, nil
	}

	return res, nil
}

func Do(cmd string, args ...interface{}) error {
	ps, err := DB.Prepare(cmd)
	if err != nil {
		return err
	}
	_, err = ps.Exec(args...)

	if err != nil {
		log.Printf("Error DB(%s)->%s\n", cmd, err)
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
		log.Printf("Error DB(%s)->%s\n", cmd, err)
	}

	if count, _ := result.RowsAffected(); count != 1 {
		log.Printf("Error DB(%s) didn't change any rows", cmd)
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

var initDB = `
	SET GLOBAL sql_mode = 'ANSI_QUOTES' ;

	-DROP DATABASE registry ;
	CREATE DATABASE registry ;
	USE registry ;

	CREATE TABLE Registries (
		ID          VARCHAR(255) NOT NULL,	// User defined
		"Name"      VARCHAR(64),
		BaseURL     VARCHAR(255),
		Description VARCHAR(255),
		SpecVersion VARCHAR(64),
		Docs		VARCHAR(255),

		PRIMARY KEY (ID),
		INDEX(NAME)
	);

	CREATE TABLE Tags (
		RegistryID  VARCHAR(255) NOT NULL,
		EntityType	VARCHAR(64) NOT NULL,	// Registry, Group, Resource
		EntityName	VARCHAR(64) NOT NULL,
		TagKey		VARCHAR(64) NOT NULL,
		TagValue	VARCHAR(255),

		PRIMARY KEY (RegistryID, EntityType, EntityName, TagKey ),
		FOREIGN KEY (RegistryID) REFERENCES Registries(ID)
			ON UPDATE CASCADE ON DELETE CASCADE
	);

	CREATE TABLE ModelEntities (		// Group or Resource (no parent->Group)
		ID     		VARCHAR(64),		// my System ID
		RegistryID  VARCHAR(64),
		ParentID	VARCHAR(64),		// ID of parent ModelEntity

		Plural		VARCHAR(64),
		Singular	VARCHAR(64),
		SchemaURL	VARCHAR(255),		// For Groups
		Versions    INT NOT NULL,		// For Resources

		PRIMARY KEY(ID),
		INDEX (RegistryID, ParentID, Plural),
		FOREIGN KEY (RegistryID) REFERENCES Registries(ID)
			ON UPDATE CASCADE ON DELETE CASCADE
	);

	CREATE TABLE "Groups" (
		ID				VARCHAR(64) NOT NULL,	// System ID
		RegistryID		VARCHAR(64) NOT NULL,
		GroupID			VARCHAR(64) NOT NULL,	// User defined
		ModelID			VARCHAR(64) NOT NULL,
		Path			VARCHAR(255) NOT NULL,
		Abstract		VARCHAR(255) NOT NULL,

		PRIMARY KEY (ID),
		INDEX(GroupID),
		FOREIGN KEY (ModelID) REFERENCES ModelEntities(ID) ON UPDATE CASCADE,
		FOREIGN KEY (RegistryID) REFERENCES Registries(ID)
			ON UPDATE CASCADE ON DELETE CASCADE
	);

	CREATE TABLE Resources (
		ID				VARCHAR(64) NOT NULL,	// System ID
		ResourceID      VARCHAR(64) NOT NULL,	// User defined
		GroupID			VARCHAR(64) NOT NULL,	// System ID
		ModelID         VARCHAR(64) NOT NULL,
		Path			VARCHAR(255) NOT NULL,
		Abstract		VARCHAR(255) NOT NULL,

		PRIMARY KEY (ID),
		INDEX(ResourceID),
		FOREIGN KEY (GroupID) REFERENCES "Groups"(ID)
			ON UPDATE CASCADE ON DELETE CASCADE,
		FOREIGN KEY (ModelID) REFERENCES ModelEntities(ID)
			ON UPDATE CASCADE ON DELETE CASCADE
	);

	CREATE TABLE Versions (
		ID					VARCHAR(64) NOT NULL,	// System ID
		VersionID			VARCHAR(64) NOT NULL,	// User defined
		ResourceID			VARCHAR(64) NOT NULL,	// System ID
		Path				VARCHAR(255) NOT NULL,
		Abstract			VARCHAR(255) NOT NULL,

		ResourceURL     	VARCHAR(255),
		ResourceProxyURL	VARCHAR(255),
		ResourceContent		MEDIUMBLOB,

		PRIMARY KEY (ID),
		INDEX(VersionID),
		FOREIGN KEY (ResourceID) REFERENCES Resources(ID) 
			ON UPDATE CASCADE ON DELETE CASCADE
	);

	CREATE TABLE Props (
		RegistryID  VARCHAR(64) NOT NULL,
		EntityID	VARCHAR(64) NOT NULL,		// Group, Res or Ver System ID
		PropName	VARCHAR(64) NOT NULL,
		PropValue	VARCHAR(255),
		PropType	VARCHAR(64) NOT NULL,

		PRIMARY KEY (EntityID, PropName),
		INDEX (EntityID),
		FOREIGN KEY (RegistryID) REFERENCES Registries(ID) 
			ON UPDATE CASCADE ON DELETE CASCADE
	);

	CREATE VIEW LatestProps AS
	SELECT
		p.RegistryID,
		r.ID AS EntityID,
		p.PropName,
		p.PropValue,
		p.PropType
	FROM Props AS p
	JOIN Versions AS v ON (p.EntityID=v.ID)
	JOIN Resources AS r ON (r.ID=v.ResourceID)
	JOIN Props AS p1 ON (p1.EntityID=r.ID)
	WHERE p1.PropName='LatestId' AND v.VersionID=p1.PropValue AND
		  p.PropName<>'id';		// Don't overwrite 'id'

	CREATE VIEW AllProps AS
	SELECT * FROM Props
	UNION SELECT * FROM LatestProps ;


	CREATE VIEW Entities AS
	SELECT							// Gather Groups
		g.RegistryID AS RegID,
		1 AS Level,
		m.Plural AS Plural,
		NULL AS ParentID,
		g.ID AS eID,
		g.GroupID AS ID,
		g.Abstract,
		g.Path
	FROM "Groups" AS g
	JOIN ModelEntities AS m ON (m.ID=g.ModelID)

	UNION SELECT					// Add Resources
		m.RegistryID AS RegID,
		2 AS Level,
		m.Plural AS Plural,
		r.GroupID AS ParentID,
		r.ID AS eID,
		r.ResourceID AS ID,
		r.Abstract,
		r.Path
	FROM Resources AS r
	JOIN ModelEntities AS m ON (m.ID=r.ModelID)

	UNION SELECT					// Add Versions
		rm.RegistryID AS RegID,
		3 AS Level,
		'versions' AS Plural,
		r.ID AS ParentID,
		v.ID AS eID,
		v.VersionID AS ID,
		v.Abstract,
		v.Path
	FROM Versions AS v
	JOIN Resources AS r ON (r.ID=v.ResourceID)
	JOIN ModelEntities AS rm ON (rm.ID=r.ModelID) ;

	CREATE VIEW FullTree AS
	SELECT
		RegID,
		Level,
		Plural,
		ParentID,
		eID,
		ID,
		Path,
		PropName,
		PropValue,
		PropType,
		Abstract
	FROM Entities
	LEFT JOIN AllProps ON (AllProps.EntityID=Entities.eID)
	ORDER by Path, PropName;

	CREATE VIEW Leaves AS
	SELECT eID FROM Entities
	WHERE eID NOT IN (
		SELECT DISTINCT ParentID FROM Entities WHERE ParentID IS NOT NULL
	);

`

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


*/

func init() {
	// DB, err := sql.Open("mysql", "root:password@tcp(localhost:3306)/")
	var err error

	DB, err = sql.Open("mysql", "root:password@/")
	if err != nil {
		log.Fatalf("Error talking to SQL: %s\n", err)
	}
	cmd := ""
	DB.SetMaxOpenConns(5)
	DB.SetMaxIdleConns(5)

	for _, line := range strings.Split(initDB, "\n") {
		line, _, _ = strings.Cut(line, "//")
		line = strings.TrimSpace(line)
		if cmd != "" {
			cmd += " "
		}
		cmd += line
		if strings.HasSuffix(line, ";") {
			log.VPrintf(4, "CMD: %s", cmd)
			if cmd[0] == '-' {
				IgnoreDo(cmd[1:])
			} else {
				MustDo(cmd)
			}
			cmd = ""
		}
	}
	log.VPrintf(2, "Done init'ing DB")
}

type Entity struct {
	RegistryID string
	DbID       string
	Plural     string
	Extensions map[string]any
}

func (e *Entity) sSet(name string, value any) error {
	return SetProp(e, name, value)
}

func SetProp(entity any, name string, val any) error {
	log.VPrintf(3, ">Enter: SetProp(%s=%v)", name, val)
	defer log.VPrintf(3, "<Exit SetProp")

	eField := reflect.ValueOf(entity).Elem().FieldByName("Entity")
	e := (*Entity)(nil)
	if !eField.IsValid() {
		log.Fatalf("Passing a non-entity to SetProp: %#v", entity)
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
		err = Do(`
			REPLACE INTO Props( 
				RegistryID, EntityID, PropName, PropValue, PropType)
			VALUES( ?,?,?,?,? )`,
			e.RegistryID, e.DbID, name, val,
			reflect.ValueOf(val).Type().Kind())
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
