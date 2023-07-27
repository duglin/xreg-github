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

	// These were added just for convinience and so we can use the same
	// struct for traversing the SQL results
	Level    int
	Path     string
	Abstract string
}

func (e *Entity) Get(name string) any {
	if name == "#resource" {
		results, err := Query(`
            SELECT Content
            FROM ResourceContents
            WHERE VersionSID=?`, e.DbSID)
		defer results.Close()

		if err != nil {
			return fmt.Errorf("Error finding contents %q: %s", e.DbSID, err)
		}

		row := results.NextRow()
		if row == nil {
			// No data so just return
			return nil
		}

		if results.NextRow() != nil {
			panic("too many results")
		}

		return (*(row[0])).([]byte)
	}

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

	// Only allow "." in the name if it's "labels.xxx"
	preDot, _, found := strings.Cut(name, ".")
	if found {
		if preDot != "labels" {
			return fmt.Errorf("Can't use '.' in a property name except for "+
				"labels: %s", name)
		}
	} else if name == "labels" {
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

	// #resource is special and is saved in it's own table
	if name == "#resource" {
		// The actual contents
		err := DoOne(`
            INSERT INTO ResourceContents(VersionSID, Content)
            VALUES(?,?)`, e.DbSID, val)
		return err
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

	// Technically this is old, we should just assume everything is in Props
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

func readNextEntity(results *Result) *Entity {
	entity := (*Entity)(nil)

	// RegSID,Level,Plural,eSID,UID,PropName,PropValue,PropType,Path,Abstract
	//   0     1      2     3    4     5         6         7     8      9
	for row := results.NextRow(); row != nil; row = results.NextRow() {
		// log.Printf("Row(%d): %#v", len(row), row)
		level := int((*row[1]).(int64))
		plural := NotNilString(row[2])
		uid := NotNilString(row[4])

		if entity == nil {
			entity = &Entity{
				RegistrySID: NotNilString(row[0]),
				DbSID:       NotNilString(row[3]),
				Plural:      plural,
				UID:         uid,
				Props:       map[string]any{},

				Level:    level,
				Path:     NotNilString(row[8]),
				Abstract: NotNilString(row[9]),
			}
		} else {
			// If the next row isn't part of the current Entity then
			// push it back into the result set so we'll grab it the next time
			// we're called. And exit.
			if entity.Level != level || entity.Plural != plural || entity.UID != uid {
				results.Push()
				break
			}
		}

		propName := NotNilString(row[5])
		propVal := NotNilString(row[6])
		propType := NotNilString(row[7])

		if propType == "s" {
			entity.Props[propName] = propVal
		} else if propType == "b" {
			entity.Props[propName] = (propVal == "true")
		} else if propType == "i" {
			tmpInt, err := strconv.Atoi(propVal)
			if err != nil {
				panic(fmt.Sprintf("error parsing int: %s", propVal))
			}
			entity.Props[propName] = tmpInt
		} else if propType == "f" {
			tmpFloat, err := strconv.ParseFloat(propVal, 64)
			if err != nil {
				panic(fmt.Sprintf("error parsing float: %s", propVal))
			}
			entity.Props[propName] = tmpFloat
		} else {
			panic(fmt.Sprintf("bad type: %v", propType))
		}
	}

	return entity
}

// This allows for us to choose the order and define custom logic per prop
var orderedProps = []struct {
	key    string                          // prop name
	levels string                          // only show for these levels
	fn     func(*Entity, *RequestInfo) any // caller will Marshal the 'any'
}{
	{"specVersion", "", nil},
	{"id", "", nil},
	{"name", "", nil},
	{"epoch", "23", nil},
	{"self", "", func(e *Entity, info *RequestInfo) any {
		return info.BaseURL + "/" + e.Path
	}},
	{"latest", "3", nil},
	{"latestId", "2", nil},
	{"latestUrl", "2", func(e *Entity, info *RequestInfo) any {
		val := e.Props["latestId"]
		if IsNil(val) {
			return nil
		}
		return info.BaseURL + "/" + e.Path + "/versions/" + val.(string)
	}},
	{"description", "", nil},
	{"docs", "", nil},
	{"labels", "", func(e *Entity, info *RequestInfo) any {
		var res map[string]string

		for _, key := range SortedKeys(e.Props) {
			if key[0] > 't' {
				break
			}

			if strings.HasPrefix(key, "labels.") {
				val, _ := e.Props[key]
				if res == nil {
					res = map[string]string{}
				}
				// Convert it to a string per the spec
				res[key[7:]] = fmt.Sprintf("%v", val)
			}
		}
		return res
	}},
	{"createdBy", "", nil},
	{"createdOn", "", nil},
	{"modifiedBy", "", nil},
	{"modifiedOn", "", nil},
	{"model", "0", func(e *Entity, info *RequestInfo) any {
		if info.ShowModel {
			model := info.Registry.Model
			if model == nil {
				model = &Model{}
			}
			httpModel := ModelToHTTPModel(model)
			return httpModel
		}
		return nil
	}},
}

// This is used to serialize Prop regardless of the format.
func (e *Entity) SerializeProps(info *RequestInfo,
	fn func(*Entity, *RequestInfo, string, any) error) error {

	usedProps := map[string]bool{}

	for _, prop := range orderedProps {
		usedProps[prop.key] = true

		// Only show props that are for this level
		ch := rune('0' + byte(e.Level))
		if prop.levels != "" && !strings.ContainsRune(prop.levels, ch) {
			continue
		}

		// Even if it has a func, if there's a val in Values let it override
		val, ok := e.Props[prop.key]
		if !ok && prop.fn != nil {
			val = prop.fn(e, info)
		}

		// Only write it if we have a value
		if !IsNil(val) {
			err := fn(e, info, prop.key, val)
			if err != nil {
				log.Printf("Error serializing %q(%v): %s", prop.key, val, err)
				return err
			}
		}
	}

	// Now write the remaining properties (sorted)
	for _, key := range SortedKeys(e.Props) {
		// Keys that start with '#' are for internal use only
		if key[0] == '#' {
			continue
		}
		// "labels." is special and we know we did it above
		if usedProps[key] || strings.HasPrefix(key, "labels.") {
			continue
		}
		val, _ := e.Props[key]
		err := fn(e, info, key, val)
		if err != nil {
			log.Printf("Error serializing %q(%v): %s", key, val, err)
			return err
		}
	}

	return nil
}
