package registry

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
	_ "github.com/go-sql-driver/mysql"
)

type Entity struct {
	RegistrySID string
	DbSID       string // Entity's SID
	Plural      string
	UID         string // Entity's UID
	Props       map[string]any
}

func (e *Entity) Get(name string) any {
	val, _ := e.Props[name]
	log.VPrintf(4, "%s(%s).Get(%s) -> %v", e.Plural, e.UID, name, val)
	return val
}

func (e *Entity) Find() (bool, error) {
	log.VPrintf(3, ">Enter: Find(%s)", e.UID)
	log.VPrintf(3, "<Exit: Find")

	results, err := Query(`
		SELECT
			p.RegistrySID AS RegistrySID,
			p.EntitySID AS DbSID,
			e.Plural AS Plural,
			e.UID AS UID,
			p.PropName AS PropName,
			p.PropValue AS PropValue,
			p.PropType AS PropType
		FROM Props AS p
		LEFT JOIN Entities AS e ON (e.eSID=p.EntitySID)
		WHERE e.UID=?`, e.UID)
	defer results.Close()

	if err != nil {
		return false, err
	}

	first := true
	for row := results.NextRow(); row != nil; row = results.NextRow() {
		if first {
			e.RegistrySID = NotNilString(row[0])
			e.DbSID = NotNilString(row[1])
			e.Plural = NotNilString(row[2])
			e.UID = NotNilString(row[3])
			first = false
		}
	}

	return !first, nil
}

func (e *Entity) Refresh() error {
	log.VPrintf(3, ">Enter: Refresh(%s)", e.DbSID)
	defer log.VPrintf(3, "<Exit: Refresh")

	results, err := Query(`
        SELECT PropName, PropValue, PropType
        FROM Props WHERE EntitySID=? `, e.DbSID)
	defer results.Close()

	if err != nil {
		log.Printf("Error refreshing props(%s): %s", e.DbSID, err)
		return fmt.Errorf("Error refreshing props(%s): %s", e.DbSID, err)
	}

	e.Props = map[string]any{}

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		name := NotNilString(row[0])
		val := NotNilString(row[1])
		propType := NotNilString(row[2])

		if propType == "s" {
			e.Props[name] = val
		} else if propType == "b" {
			e.Props[name] = (val == "true")
		} else if propType == "i" {
			tmpInt, err := strconv.Atoi(val)
			if err != nil {
				panic(fmt.Sprintf("error parsing int: %s", val))
			}
			e.Props[name] = tmpInt
		} else if propType == "f" {
			tmpFloat, err := strconv.ParseFloat(val, 64)
			if err != nil {
				panic(fmt.Sprintf("error parsing float: %s", val))
			}
			e.Props[name] = tmpFloat
		} else {
			panic(fmt.Sprintf("bad type: %v", propType))
		}
	}
	return nil
}

func (e *Entity) sSet(name string, val any) error {
	log.VPrintf(3, ">Enter: SetProp(%s=%v)", name, val)
	defer log.VPrintf(3, "<Exit SetProp")

	if e.DbSID == "" {
		log.Fatalf("DbSID should not be empty")
	}
	if e.RegistrySID == "" {
		log.Fatalf("RegistrySID should not be empty")
	}

	var err error
	if val == nil {
		err = Do(`DELETE FROM Props WHERE EntitySID=? and PropName=?`,
			e.DbSID, name)
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
				RegistrySID, EntitySID, PropName, PropValue, PropType)
			VALUES( ?,?,?,?,? )`,
			e.RegistrySID, e.DbSID, name, val, propType)
	}

	if err != nil {
		log.Printf("Error updating prop(%s/%v): %s", name, val, err)
		return fmt.Errorf("Error updating prop(%s/%v): %s", name, val, err)
	}
	return nil
	// return SetProp(e, name, val)
}

// Maybe replace error with a panic?
func SetProp(entity any, name string, val any) error {
	log.VPrintf(3, ">Enter: SetProp(%s=%v)", name, val)
	defer log.VPrintf(3, "<Exit SetProp")

	// Only allow "." in the name if it's "tags.xxx"
	preDot, tagName, found := strings.Cut(name, ".")
	if found {
		if preDot != "tags" {
			return fmt.Errorf("Can't use '.' in a property name except for "+
				"tags: %s", name)
		}
		if strings.Index(tagName, ".") >= 0 {
			return fmt.Errorf("Can't use '.' in a tag name: %s", name)
		}
	} else if name == "tags" {
		return fmt.Errorf("Invalid propery name: %s", name)
	}

	eField := reflect.ValueOf(entity).Elem().FieldByName("Entity")
	e := (*Entity)(nil)
	if !eField.IsValid() {
		panic(fmt.Sprintf("Passing a non-entity to SetProp: %#v", entity))
		// e = entity.(*Entity)
	} else {
		e = eField.Addr().Interface().(*Entity)
	}

	if e.DbSID == "" {
		log.Fatalf("DbSID should not be empty")
	}
	if e.RegistrySID == "" {
		log.Fatalf("RegistrySID should not be empty")
	}

	var err error
	if val == nil {
		err = Do(`DELETE FROM Props WHERE EntitySID=? and PropName=?`,
			e.DbSID, name)
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
				RegistrySID, EntitySID, PropName, PropValue, PropType)
			VALUES( ?,?,?,?,? )`,
			e.RegistrySID, e.DbSID, name, dbVal, propType)
	}

	if err != nil {
		log.Printf("Error updating prop(%s/%v): %s", name, val, err)
		return fmt.Errorf("Error updating prop(%s/%v): %s", name, val, err)
	}

	field := reflect.ValueOf(entity).Elem().FieldByName(name)
	if !field.IsValid() {
		field := reflect.ValueOf(e).Elem().FieldByName("Props")
		if !field.IsValid() {
			log.VPrintf(2, "Can't Set unknown field(%s/%s)", e.DbSID, name)
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

type Obj struct {
	Level    int
	Plural   string
	UID      string
	Path     string
	Abstract string
	Values   map[string]any
}

func readObj(results *Result) *Obj {
	obj := (*Obj)(nil)

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		level := int((*row[0]).(int64))
		plural := NotNilString(row[1])
		uid := NotNilString(row[2])

		if obj == nil {
			obj = &Obj{
				Level:    level,
				Plural:   plural,
				UID:      uid,
				Path:     NotNilString(row[6]),
				Abstract: NotNilString(row[7]),
				Values:   map[string]any{},
			}
		} else {
			if obj.Level != level || obj.Plural != plural || obj.UID != uid {
				results.Push()
				break
			}
		}

		propName := NotNilString(row[3])
		propVal := NotNilString(row[4])
		propType := NotNilString(row[5])

		if propType == "s" {
			obj.Values[propName] = propVal
		} else if propType == "b" {
			obj.Values[propName] = (propVal == "true")
		} else if propType == "i" {
			tmpInt, err := strconv.Atoi(propVal)
			if err != nil {
				panic(fmt.Sprintf("error parsing int: %s", propVal))
			}
			obj.Values[propName] = tmpInt
		} else if propType == "f" {
			tmpFloat, err := strconv.ParseFloat(propVal, 64)
			if err != nil {
				panic(fmt.Sprintf("error parsing float: %s", propVal))
			}
			obj.Values[propName] = tmpFloat
		} else {
			panic(fmt.Sprintf("bad type: %v", propType))
		}
	}

	return obj
}

/*
type ResultsContext struct {
	results [][]*any
	pos     int
}

func (rc *ResultsContext) NextObj() *Obj {
	obj, nextPos := readObj(rc.results, rc.pos)
	rc.pos = nextPos
	return obj
}
*/
