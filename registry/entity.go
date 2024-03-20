package registry

import (
	"encoding/base64"
	"fmt"
	"maps"
	"reflect"
	"strconv"
	"strings"
	"time"

	log "github.com/duglin/dlog"
	_ "github.com/go-sql-driver/mysql"
)

type Object map[string]any

// type Map map[string]any
// type Array []any

type Entity struct {
	tx *Tx

	Registry  *Registry `json:"-"`
	DbSID     string    // Entity's SID
	Plural    string
	UID       string         // Entity's UID
	Object    map[string]any `json:"-"`
	NewObject map[string]any `json:"-"` // updated version, save() will store

	// These were added just for convenience and so we can use the same
	// struct for traversing the SQL results
	Level     int // 0=registry, 1=group, 2=resource, 3=version
	Path      string
	Abstract  string
	SkipEpoch bool `json:"-"` // Should we skip epoch-specific processing?
}

type EntitySetter interface {
	Get(name string) any
	Set(name string, val any) error
}

func GoToOurType(val any) string {
	switch reflect.ValueOf(val).Kind() {
	case reflect.Bool:
		return BOOLEAN
	case reflect.Int:
		return INTEGER
	case reflect.Interface:
		return ANY
	case reflect.Float64:
		return DECIMAL
	case reflect.String:
		return STRING
	case reflect.Uint64:
		return UINTEGER
	case reflect.Slice:
		return ARRAY
	case reflect.Map:
		return MAP
	case reflect.Struct:
		return OBJECT
	}
	panic(fmt.Sprintf("Bad Kind: %v", reflect.ValueOf(val).Kind()))
}

func (e *Entity) Get(path string) any {
	pp, err := PropPathFromUI(path)
	PanicIf(err != nil, fmt.Sprintf("%s", err))
	return e.GetPP(pp)
}

func (e *Entity) GetPP(pp *PropPath) any {
	name := pp.DB()
	if pp.Len() == 1 && pp.Top() == "#resource" {
		results, err := Query(e.tx, `
            SELECT Content
            FROM ResourceContents
            WHERE VersionSID=? OR
			      VersionSID=(SELECT eSID FROM FullTree WHERE ParentSID=? AND
				  PropName=? and PropValue='true')
			`, e.DbSID, e.DbSID, NewPPP("latest").DB())
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

	/*
		// At some point we may decide we need to check the NewObject map
		// for the value - eg. if the resource hasn't been saved yet.
		var val any
		if e.NewObject != nil {
			// TODO check this - what if it got nil'd out??  DUG
			val, _ = ObjectGetProp(e.NewObject, pp)
			if IsNil(val) {
				val, _ = ObjectGetProp(e.Object, pp)
			}
		} else {
			val, _ = ObjectGetProp(e.Object, pp)
		}
	*/

	// An error from ObjectGetProp is ignored because if they tried to
	// go into something incorrect/bad we should just return 'nil'.
	// This may not be the best choice in the long-run - which in case we
	// should return the 'error'
	val, _ := ObjectGetProp(e.Object, pp)
	log.VPrintf(4, "%s(%s).Get(%s) -> %v", e.Plural, e.UID, name, val)
	return val
}

func ObjectGetProp(obj any, pp *PropPath) (any, error) {
	return NestedGetProp(obj, pp, NewPP())
}

func NestedGetProp(obj any, pp *PropPath, prev *PropPath) (any, error) {
	log.VPrintf(3, "ObjectGetProp: %q\nobj:\n%s", pp.UI(), ToJSON(obj))
	if pp == nil || pp.Len() == 0 {
		return obj, nil
	}
	if IsNil(obj) {
		return nil, fmt.Errorf("Can't traverse into nothing: %s", prev.UI())
	}

	objValue := reflect.ValueOf(obj)
	part := pp.Parts[0]
	if index := part.Index; index >= 0 {
		// Is an array
		if objValue.Kind() != reflect.Slice {
			return nil, fmt.Errorf("Can't index into non-array: %s", prev.UI())
		}
		if index < 0 || index >= objValue.Len() {
			return nil, fmt.Errorf("Array reference %q out of bounds: "+
				"(max:%d-1)", prev.Append(pp.First()).UI(), objValue.Len())
		}
		objValue = objValue.Index(index)
		if objValue.IsValid() {
			obj = objValue.Interface()
		} else {
			obj = nil
		}
		return NestedGetProp(obj, pp.Next(), prev.Append(pp.First()))
	}

	// Is map/object
	if objValue.Kind() != reflect.Map {
		return nil, fmt.Errorf("Can't reference a non-map/object: %s",
			prev.UI())
	}
	if objValue.Type().Key().Kind() != reflect.String {
		return nil, fmt.Errorf("Key of %q must be a string, not %s",
			prev.UI(), objValue.Type().Key().Kind())
	}

	objValue = objValue.MapIndex(reflect.ValueOf(pp.Top()))
	if objValue.IsValid() {
		obj = objValue.Interface()
	} else {
		obj = nil
	}
	return NestedGetProp(obj, pp.Next(), prev.Append(pp.First()))
}

func RawEntityFromPath(tx *Tx, regID string, path string) (*Entity, error) {
	log.VPrintf(3, ">Enter: RawEntityFromPath(%s)", path)
	defer log.VPrintf(3, "<Exit: RawEntityFromPath")

	// RegSID,Level,Plural,eSID,UID,PropName,PropValue,PropType,Path,Abstract
	//   0     1      2     3    4     5         6         7     8      9

	results, err := Query(tx, `
		SELECT
            e.RegSID as RegSID,
            e.Level as Level,
            e.Plural as Plural,
            e.eSID as eSID,
            e.UID as UID,
            p.PropName as PropName,
            p.PropValue as PropValue,
            p.PropType as PropType,
            e.Path as Path,
            e.Abstract as Abstract
        FROM Entities AS e
        LEFT JOIN Props AS p ON (e.eSID=p.EntitySID)
        WHERE e.RegSID=? AND e.Path=? ORDER BY Path`, regID, path)
	defer results.Close()

	if err != nil {
		return nil, err
	}

	return readNextEntity(tx, results)
}

// Update the entity's Object - not the other props in Entity. Similar to
// RawEntityFromPath
func (e *Entity) Refresh() error {
	log.VPrintf(3, ">Enter: Refresh(%s)", e.DbSID)
	defer log.VPrintf(3, "<Exit: Refresh")

	results, err := Query(e.tx, `
        SELECT PropName, PropValue, PropType
        FROM Props WHERE EntitySID=? `, e.DbSID)
	defer results.Close()

	if err != nil {
		log.Printf("Error refreshing props(%s): %s", e.DbSID, err)
		return fmt.Errorf("Error refreshing props(%s): %s", e.DbSID, err)
	}

	// Erase all old props first
	e.Object = map[string]any{}
	e.NewObject = nil

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		name := NotNilString(row[0])
		val := NotNilString(row[1])
		propType := NotNilString(row[2])

		if err = e.SetFromDBName(name, &val, propType); err != nil {
			return err
		}
	}
	return nil
}

// All in one: Set, Validate, Save to DB and Commit (or Rollback on error)
func (e *Entity) Set(path string, val any) error {
	log.VPrintf(3, ">Enter: Set(%s=%v)", path, val)
	defer log.VPrintf(3, "<Exit Set")

	err := e.SetSave(path, val)
	Must(e.tx.Conditional(err))

	return err
}

// Set, Validate and Save to DB but not Commit
func (e *Entity) SetSave(path string, val any) error {
	log.VPrintf(3, ">Enter: Set(%s=%v)", path, val)
	defer log.VPrintf(3, "<Exit Set")

	pp, err := PropPathFromUI(path)
	if err == nil {
		// Set, Validate and Save
		err = e.SetPP(pp, val)
	}

	return err
}

// Set the prop in the Entity but don't Validate or Save to the DB
func (e *Entity) JustSet(pp *PropPath, val any) error {
	log.VPrintf(3, ">Enter: JustSet(%s=%v)", pp.UI(), val)
	defer log.VPrintf(3, "<Exit: JustSet")

	// Assume no other edits are pending
	// e.Refresh() // trying not to have this here
	if e.Object == nil {
		e.Object = map[string]any{}
	}

	if e.NewObject == nil {
		// If we don't have a NewObject yet then this is our first update
		// so clone the current values before adding the new prop/val
		e.NewObject = maps.Clone(e.Object)
	}

	// Cheat a little just to make caller's life easier by converting
	// empty structs and maps to be of the type we like (meaning 'any's)
	if !IsNil(val) {
		if val == struct{}{} {
			val = map[string]any{}
		}
		valValue := reflect.ValueOf(val)
		if valValue.Kind() == reflect.Slice && valValue.Len() == 0 {
			val = []any{}
		}
		if valValue.Kind() == reflect.Map && valValue.Len() == 0 {
			val = map[string]any{}
		}
	}
	// end of cheat

	if pp.Top() == "epoch" {
		save := e.SkipEpoch
		e.SkipEpoch = true
		defer func() {
			e.SkipEpoch = save
		}()
	}

	log.VPrintf(3, "Abstract/ID: %s/%s", e.Abstract, e.UID)
	log.VPrintf(3, "e.Object:\n%s", ToJSON(e.Object))
	log.VPrintf(3, "e.NewObject:\n%s", ToJSON(e.NewObject))

	return ObjectSetProp(e.NewObject, pp, val)
}

func (e *Entity) ValidateAndSave(isNew bool) error {
	log.VPrintf(3, ">Enter: ValidateAndSave")
	defer log.VPrintf(3, "<Exit: ValidateAndSave")

	// If nothing changed, just exit
	// if e.NewObject == nil {
	// return nil
	// }

	log.VPrintf(3, "Validating e.NewObject:\n%s", ToJSON(e.NewObject))

	if err := e.Validate(); err != nil {
		return err
	}

	if err := PrepUpdateEntity(e, isNew); err != nil {
		return err
	}

	return e.Save()
}

// This is really just an internal Setter used for testing.
// It'sll set a property and then validate and save the entity in the DB
func (e *Entity) SetPP(pp *PropPath, val any) error {
	log.VPrintf(3, ">Enter: SetPP(%s: %s=%v)", e.DbSID, pp.UI(), val)
	defer log.VPrintf(3, "<Exit SetPP")
	defer func() {
		log.VPrintf(3, "SetPP exit: e.Object:\n%s", ToJSON(e.Object))
	}()

	if err := e.JustSet(pp, val); err != nil {
		return err
	}

	// Make the bold assumption that we we're setting and saving all in one
	// that a user who is explicitly setting 'epoc' via an interenal
	// set() knows what they're doing
	save := e.SkipEpoch
	e.SkipEpoch = true
	defer func() { e.SkipEpoch = save }()

	err := e.ValidateAndSave(false)
	if err != nil {
		// If there's an error, and we're making the assumption that we're
		// setting and saving all in one shot (and there are no other edits
		// pending), go ahead and undo the changes since they're wrong.
		// Otherwise the caller would need to call Refresh themselves.

		// Not sure why setting it to nil isn't sufficient (todo)
		// e.NewObject = nil
		e.Refresh()
	}

	return err
}

// This will save a single property/value in the DB. This assumes
// the caller is traversing the Object and splitting it into individual props
func (e *Entity) SetDBProperty(pp *PropPath, val any) error {
	log.VPrintf(3, ">Enter: SetDBProperty(%s=%v)", pp.UI(), val)
	defer log.VPrintf(3, "<Exit SetDBProperty")

	PanicIf(pp.UI() == "", "pp is empty")

	var err error
	name := pp.DB()

	// Any prop with "dontStore"=true we skip
	if sp, ok := SpecProps[pp.Top()]; ok && sp.dontStore {
		return nil
	}

	PanicIf(e.DbSID == "", "DbSID should not be empty")
	PanicIf(e.Registry == nil, "Registry should not be nil")

	// #resource is special and is saved in it's own table
	// Need to explicitly set #resoure to nil to delete it.
	if pp.Len() == 1 && pp.Top() == "#resource" {
		if IsNil(val) {
			err = Do(e.tx, `DELETE FROM ResourceContents WHERE VersionSID=?`,
				e.DbSID)
			return err
		} else {
			if val == "" {
				return nil
			}
			// The actual contents
			err = DoOneTwo(e.tx, `
                REPLACE INTO ResourceContents(VersionSID, Content)
            	VALUES(?,?)`, e.DbSID, val)
			if err != nil {
				return err
			}
			val = ""
			// Fall thru to normal processing so we save a placeholder
			// attribute in the resource
		}
	}

	if IsNil(val) {
		// Should never use this but keeping it just in case
		err = Do(e.tx, `DELETE FROM Props WHERE EntitySID=? and PropName=?`,
			e.DbSID, name)
	} else {
		propType := GoToOurType(val)

		// Convert booleans to true/false instead of 1/0 so filter works
		// ...=true and not ...=1
		dbVal := val
		if propType == BOOLEAN {
			if val == true {
				dbVal = "true"
			} else {
				dbVal = "false"
			}
		}

		switch reflect.ValueOf(val).Kind() {
		case reflect.Slice:
			if reflect.ValueOf(val).Len() > 0 {
				return fmt.Errorf("Can't set non-empty arrays")
			}
			dbVal = ""
		case reflect.Map:
			if reflect.ValueOf(val).Len() > 0 {
				return fmt.Errorf("Can't set non-empty maps")
			}
			dbVal = ""
		case reflect.Struct:
			if reflect.ValueOf(val).NumField() > 0 {
				return fmt.Errorf("Can't set non-empty objects")
			}
			dbVal = ""
		}

		err = DoOneTwo(e.tx, `
            REPLACE INTO Props(
              RegistrySID, EntitySID, PropName, PropValue, PropType)
            VALUES( ?,?,?,?,? )`,
			e.Registry.DbSID, e.DbSID, name, dbVal, propType)
	}

	if err != nil {
		log.Printf("Error updating prop(%s/%v): %s", pp.UI(), val, err)
		return fmt.Errorf("Error updating prop(%s/%v): %s", pp.UI(), val, err)
	}

	return nil
}

// This is used to take a DB entry and update the current Entity's Object
func (e *Entity) SetFromDBName(name string, val *string, propType string) error {
	pp := MustPropPathFromDB(name)

	if val == nil {
		return ObjectSetProp(e.Object, pp, val)
	}
	if e.Object == nil {
		e.Object = map[string]any{}
	}

	if propType == STRING || propType == URI || propType == URI_REFERENCE ||
		propType == URI_TEMPLATE || propType == URL || propType == TIMESTAMP {
		return ObjectSetProp(e.Object, pp, *val)
	} else if propType == BOOLEAN {
		// Technically the "1" check shouldn't be needed, but just in case
		return ObjectSetProp(e.Object, pp, (*val == "1" || (*val == "true")))
	} else if propType == INTEGER || propType == UINTEGER {
		tmpInt, err := strconv.Atoi(*val)
		if err != nil {
			panic(fmt.Sprintf("error parsing int: %s", *val))
		}
		return ObjectSetProp(e.Object, pp, tmpInt)
	} else if propType == DECIMAL {
		tmpFloat, err := strconv.ParseFloat(*val, 64)
		if err != nil {
			panic(fmt.Sprintf("error parsing float: %s", *val))
		}
		return ObjectSetProp(e.Object, pp, tmpFloat)
	} else if propType == MAP {
		if *val != "" {
			panic(fmt.Sprintf("MAP value should be empty string"))
		}
		return ObjectSetProp(e.Object, pp, map[string]any{})
	} else if propType == ARRAY {
		if *val != "" {
			panic(fmt.Sprintf("MAP value should be empty string"))
		}
		return ObjectSetProp(e.Object, pp, []any{})
	} else if propType == OBJECT {
		if *val != "" {
			panic(fmt.Sprintf("MAP value should be empty string"))
		}
		return ObjectSetProp(e.Object, pp, map[string]any{})
	} else {
		panic(fmt.Sprintf("bad type(%s): %v", propType, name))
	}
}

// Create a new Entity based on what's in the DB. Similar to Refresh()
func readNextEntity(tx *Tx, results *Result) (*Entity, error) {
	entity := (*Entity)(nil)

	// RegSID,Level,Plural,eSID,UID,PropName,PropValue,PropType,Path,Abstract
	//   0     1      2     3    4     5         6         7     8      9
	for row := results.NextRow(); row != nil; row = results.NextRow() {
		// log.Printf("Row(%d): %#v", len(row), row)
		if log.GetVerbose() >= 4 {
			str := "("
			for _, c := range row {
				if IsNil(c) || IsNil(*c) {
					str += "nil,"
				} else {
					str += fmt.Sprintf("%s,", *c)
				}
			}
			log.Printf("Row: %s)", str)
		}
		level := int((*row[1]).(int64))
		plural := NotNilString(row[2])
		uid := NotNilString(row[4])

		if entity == nil {
			entity = &Entity{
				tx: tx,

				Registry: tx.Registry,
				DbSID:    NotNilString(row[3]),
				Plural:   plural,
				UID:      uid,

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

		// Edge case - no props but entity is there
		if propName == "" && propVal == "" && propType == "" {
			continue
		}

		if err := entity.SetFromDBName(propName, &propVal, propType); err != nil {
			return nil, err
		}
	}

	return entity, nil
}

// This allows for us to choose the order and define custom logic per prop
var OrderedSpecProps = []*Attribute{
	{
		Name:           "specversion",
		Type:           STRING,
		ServerRequired: true,
		ReadOnly:       true,

		levels:    "0",
		immutable: true,
		dontStore: false,
		getFn: func(e *Entity, info *RequestInfo) any {
			return SPECVERSION
		},
		checkFn: func(e *Entity) error {
			tmp := e.NewObject["specversion"]
			if !IsNil(tmp) && tmp != "" && tmp != SPECVERSION {
				return fmt.Errorf("Invalid 'specversion': %s", tmp)
			}
			return nil
		},
		updateFn: nil,
	},
	{
		Name:           "id",
		Type:           STRING,
		ServerRequired: true,

		levels:    "",
		immutable: true,
		dontStore: false,
		getFn:     nil,
		checkFn: func(e *Entity) error {
			if e.Object != nil {
				oldID := any(e.UID)
				newID := any(e.NewObject["id"])

				if !IsNil(newID) && newID == "" {
					return fmt.Errorf("ID can't be an empty string")
				}
				if IsNil(newID) {
					newID = ""
				}

				if newID != "" && oldID != "" && newID != oldID {
					return fmt.Errorf("Can't change the ID of an "+
						"entity(%s->%s)", oldID, newID)
				}
			}
			return nil
		},
		updateFn: func(e *Entity, isNew bool) error {
			if e.Object != nil {
				if IsNil(e.NewObject["id"]) && !IsNil(e.Object["id"]) {
					e.NewObject["id"] = e.Object["id"]
					return nil
				}
			}
			return nil
		},
	},
	{
		Name: "name",
		Type: STRING,

		levels:    "",
		immutable: false,
		dontStore: false,
		getFn:     nil,
		checkFn:   nil,
		updateFn:  nil,
	},
	{
		Name:           "epoch",
		Type:           UINTEGER,
		ServerRequired: true,

		levels:    "",
		immutable: true,
		dontStore: false,
		getFn:     nil,
		checkFn: func(e *Entity) error {
			if e.SkipEpoch {
				return nil
			}

			val := e.NewObject["epoch"]
			if IsNil(val) {
				return nil
			}

			tmp := e.Object["epoch"]
			oldEpoch := NotNilInt(&tmp)
			if oldEpoch < 0 {
				oldEpoch = 0
			}

			newEpoch, err := AnyToUInt(val)
			if err != nil {
				return fmt.Errorf("Attribute \"epoch\" must be a uinteger")
			}

			if oldEpoch != 0 && newEpoch != oldEpoch {
				return fmt.Errorf("Attribute %q(%d) doesn't match existing "+
					"value (%d)", "epoch", newEpoch, oldEpoch)
			}
			return nil
		},
		updateFn: func(e *Entity, isNew bool) error {
			if e.SkipEpoch {
				return nil
			}
			tmp := e.Object["epoch"]
			if IsNil(tmp) {
				return nil
			}
			epoch := NotNilInt(&tmp)
			if epoch < 0 {
				epoch = 0
			}
			if isNew {
				epoch = 0
			}
			e.NewObject["epoch"] = epoch + 1
			return nil
		},
	},
	{
		Name:           "self",
		Type:           URL,
		ReadOnly:       true,
		ServerRequired: true,

		levels:    "",
		immutable: true,
		dontStore: false,
		getFn: func(e *Entity, info *RequestInfo) any {
			base := ""
			if info != nil {
				base = info.BaseURL
			}
			if e.Level > 1 {
				meta := info != nil && (info.ShowMeta || info.ResourceUID == "")
				absParts := strings.Split(e.Abstract, string(DB_IN))
				gm := e.Registry.Model.Groups[absParts[0]]
				rm := gm.Resources[absParts[1]]
				if rm.HasDocument == false {
					meta = false
				}

				if meta {
					return base + "/" + e.Path + "?meta"
				} else {
					return base + "/" + e.Path
				}
			}
			return base + "/" + e.Path
		},
		checkFn:  nil,
		updateFn: nil,
	},
	{
		Name: "latest",
		Type: BOOLEAN,

		levels:    "3",
		immutable: true,
		dontStore: true,
		getFn:     nil,
		checkFn:   nil,
		updateFn: func(e *Entity, isNew bool) error {
			// TODO if set, set latestvesionid in the resource to this
			// guy's UID

			return nil
		},
	},
	{
		Name:     "latestversionid",
		Type:     STRING,
		ReadOnly: true,
		// ServerRequired: true,

		levels:    "2",
		immutable: true,
		dontStore: false,
		getFn:     nil,
		checkFn:   nil,
		updateFn:  nil,
	},
	{
		Name:     "latestversionurl",
		Type:     URL,
		ReadOnly: true,
		// ServerRequired: true,

		levels:    "2",
		immutable: true,
		dontStore: false,
		getFn: func(e *Entity, info *RequestInfo) any {
			val := e.Object["latestversionid"]
			if IsNil(val) {
				return nil
			}
			base := ""
			if info != nil {
				base = info.BaseURL
			}

			tmp := base + "/" + e.Path + "/versions/" + val.(string)

			meta := info != nil && (info.ShowMeta || info.ResourceUID == "")
			absParts := strings.Split(e.Abstract, string(DB_IN))
			gm := e.Registry.Model.Groups[absParts[0]]
			rm := gm.Resources[absParts[1]]
			if rm.HasDocument == false {
				meta = false
			}

			if meta {
				tmp += "?meta"
			}
			return tmp
		},
		checkFn:  nil,
		updateFn: nil,
	},
	{
		Name: "description",
		Type: STRING,

		levels:    "",
		immutable: false,
		dontStore: false,
		getFn:     nil,
		checkFn:   nil,
		updateFn:  nil,
	},
	{
		Name: "documentation",
		Type: URL,

		levels:    "",
		immutable: false,
		dontStore: false,
		getFn:     nil,
		checkFn:   nil,
		updateFn:  nil,
	},
	{
		Name: "labels",
		Type: MAP,
		Item: &Item{
			Type: STRING,
		},

		levels:    "",
		immutable: false,
		dontStore: false,
		getFn:     nil,
		checkFn:   nil,
		updateFn:  nil,
	},
	{
		Name: "origin",
		Type: URI,

		levels:    "123",
		immutable: false,
		dontStore: false,
		getFn:     nil,
		checkFn:   nil,
		updateFn:  nil,
	},
	{
		Name:     "createdby",
		Type:     STRING,
		ReadOnly: true,

		levels:    "",
		immutable: true,
		dontStore: false,
		getFn:     nil,
		checkFn:   nil,
		updateFn:  nil,
	},
	{
		Name:     "createdon",
		Type:     TIMESTAMP,
		ReadOnly: true,

		levels:    "",
		immutable: true,
		dontStore: false,
		getFn:     nil,
		checkFn:   nil,
		updateFn:  nil,
	},
	{
		Name:     "modifiedby",
		Type:     STRING,
		ReadOnly: true,

		levels:    "",
		immutable: true,
		dontStore: false,
		getFn:     nil,
		checkFn:   nil,
		updateFn:  nil,
	},
	{
		Name:     "modifiedon",
		Type:     TIMESTAMP,
		ReadOnly: true,

		levels:    "",
		immutable: true,
		dontStore: false,
		getFn:     nil,
		checkFn:   nil,
		updateFn:  nil,
	},
	{
		Name: "contenttype",
		Type: STRING,

		levels:     "23",
		immutable:  false,
		dontStore:  false,
		httpHeader: "Content-Type",
		getFn:      nil,
		checkFn:    nil,
		updateFn:   nil,
	},
	{
		Name:     "model",
		Type:     ANY, // OBJECT
		ReadOnly: true,

		levels:       "0",
		immutable:    true,
		dontStore:    false,
		modelExclude: true,
		getFn: func(e *Entity, info *RequestInfo) any {
			if info != nil && info.ShowModel {
				model := info.Registry.Model
				if model == nil {
					model = &Model{}
				}
				httpModel := model // ModelToHTTPModel(model)
				return httpModel
			}
			return nil
		},
		checkFn:  nil,
		updateFn: nil,
	},
}

var SpecProps = map[string]*Attribute{}

func init() {
	// Load map via lower-case version of prop name
	for _, sp := range OrderedSpecProps {
		SpecProps[sp.Name] = sp
	}
}

// This is used to serialize Prop regardless of the format.
func (e *Entity) SerializeProps(info *RequestInfo,
	fn func(*Entity, *RequestInfo, string, any, *Attribute) error) error {

	daObj := e.Materialize(info)
	attrs := e.GetAttributes(false)

	// Do spec defined props first, in order
	for _, prop := range OrderedSpecProps {
		attr, ok := attrs[prop.Name]
		if !ok {
			delete(daObj, prop.Name)
			continue // not allowed at this level so skip it
		}

		if val, ok := daObj[prop.Name]; ok {
			if err := fn(e, info, prop.Name, val, attr); err != nil {
				return err
			}
			delete(daObj, prop.Name)
		}
	}

	// Now do all other props (extensions) alphabetically
	for _, key := range SortedKeys(daObj) {
		val, _ := daObj[key]
		attr := attrs[key]
		if attr == nil {
			attr = attrs["*"]
			PanicIf(key[0] != '#' && attr == nil, "Can't find attr for %q", key)
		}

		if err := fn(e, info, key, val, attr); err != nil {
			return err
		}
	}

	return nil
}

func (e *Entity) Save() error {
	log.VPrintf(3, ">Enter: Save(%s/%s)", e.Plural, e.UID)
	defer log.VPrintf(3, "<Exit: Save")

	log.VPrintf(3, "Saving - %s (id:%s):\n%s\n", e.Abstract, e.UID,
		ToJSON(e.NewObject))

	// make a dup so we can delete some attributes
	newObj := maps.Clone(e.NewObject)

	// TODO calculate which to delete based on attr properties
	delete(newObj, "self")

	for _, coll := range e.GetCollections() {
		delete(newObj, coll)
		delete(newObj, coll+"count")
		delete(newObj, coll+"url")
	}

	err := Do(e.tx, `DELETE FROM Props WHERE EntitySID=?`, e.DbSID)
	if err != nil {
		log.Printf("Error deleting all props %s", err)
		return fmt.Errorf("Error deleting all prop: %s", err)
	}

	var traverse func(pp *PropPath, val any, obj map[string]any) error
	traverse = func(pp *PropPath, val any, obj map[string]any) error {
		if IsNil(val) { // Skip empty attributes
			return nil
		}

		switch reflect.ValueOf(val).Kind() {
		case reflect.Map:
			vMap := val.(map[string]any)
			count := 0
			for k, v := range vMap {
				if k[0] == '#' {
					if err := e.SetDBProperty(pp.P(k), v); err != nil {
						return err
					}
				} else {
					if IsNil(v) {
						continue
					}
					if err := traverse(pp.P(k), v, obj); err != nil {
						return err
					}
				}
				count++
			}
			if count == 0 && pp.Len() != 0 {
				return e.SetDBProperty(pp, map[string]any{})
			}

		case reflect.Slice:
			vArray := val.([]any)
			if len(vArray) == 0 {
				return e.SetDBProperty(pp, []any{})
			}
			for i, v := range vArray {
				if err := traverse(pp.I(i), v, obj); err != nil {
					return err
				}
			}

		case reflect.Struct:
			vMap := val.(map[string]any)
			count := 0
			for k, v := range vMap {
				if IsNil(v) {
					continue
				}
				if err := traverse(pp.P(k), v, obj); err != nil {
					return err
				}
				count++
			}
			if count == 0 {
				return e.SetDBProperty(pp, struct{}{})
			}
		default:
			// must be scalar so add it
			return e.SetDBProperty(pp, val)
		}
		return nil
	}

	err = traverse(NewPP(), newObj, e.NewObject)
	if err == nil {
		e.Object = newObj
		e.NewObject = nil
	}
	return err
}

// Note that this will copy the latest version props to the resource.
// This is mainly used for end-user facing serialization of the entity
func (e *Entity) Materialize(info *RequestInfo) map[string]any {
	mat := maps.Clone(e.Object)

	// Copy all Version props into the Resource (except for a few)
	if e.Level == 2 {
		// On Resource grab latest Version attrs
		paths := strings.Split(e.Path, "/")
		group, _ := e.Registry.FindGroup(paths[0], paths[1])
		resource, _ := group.FindResource(paths[2], paths[3])
		ver, _ := resource.GetLatest()

		if ver != nil { // can be nil during resource.create()
			// Copy version specific attributes not found in Resources
			for k, v := range ver.Object {
				if k == "id" { // Retain Resource ID
					continue
				}
				// exclude props that only appear in vers, eg. ver.latest
				if prop, ok := SpecProps[k]; ok {
					if prop.InLevel(3) && !prop.InLevel(2) {
						continue
					}
				}

				mat[k] = v
			}
		}
	}

	// Regardless of the type of entity, set the generated properties
	for _, prop := range OrderedSpecProps {
		// Only generate props that are for this level, and have a Fn
		if prop.getFn == nil || !prop.InLevel(e.Level) {
			continue
		}

		// Only generate/set the value if it's not already set
		if _, ok := mat[prop.Name]; !ok {
			if val := prop.getFn(e, info); !IsNil(val) {
				// Only write it if we have a value
				mat[prop.Name] = val
			}
		}
	}

	return mat
}

func (e *Entity) GetCollections() []string {
	paths := strings.Split(e.Abstract, string(DB_IN))
	if len(paths) == 0 || paths[0] == "" {
		return SortedKeys(e.Registry.Model.Groups)
	} else {
		gm := e.Registry.Model.Groups[paths[0]]
		PanicIf(gm == nil, "Can't find Group %q", paths[0])

		if len(paths) == 1 {
			return SortedKeys(gm.Resources)
		} else if len(paths) == 2 {
			return []string{"versions"}
		}
	}

	return nil
}

func (e *Entity) GetAttributes(useNew bool) Attributes {
	/*
		var attrs Attributes
		parts := strings.Split(e.Abstract, string(DB_IN))
		if len(parts) == 0 || parts[0] == "" {
			attrs = maps.Clone(e.Registry.Model.Attributes)
		} else if len(parts) == 1 {
			gm := e.Registry.Model.Groups[parts[0]]
			attrs = maps.Clone(gm.Attributes)
		} else {
			gm := e.Registry.Model.Groups[parts[0]]
			rm := gm.Resources[parts[1]]
			attrs = maps.Clone(rm.Attributes)
		}
	*/

	attrs := e.GetBaseAttributes()
	if useNew {
		attrs.AddIfValuesAttributes(e.NewObject)
	} else {
		attrs.AddIfValuesAttributes(e.Object)
	}
	return attrs
}

// Used to convert top level (or one level maps) from strings to the right
// scalar types. Used in cases like values coming in a HTTP headers and
// we're assuming they're all strings, at first.
// Assume that if anything is wrong that it'll be flagged later by the
// verfication checks
func (e *Entity) ConvertStrings() {
	attrs := e.GetAttributes(true) // Use e.NewObject

	for key, val := range e.NewObject {
		attr := attrs[key]
		if attr == nil {
			attr = attrs["*"]
			if attr == nil {
				// Can't find it, so it must be an error.
				// Assume we'll catch it during the normal verification checks
				continue
			}
		}

		// We'll only try to convert strings and one-level-scalar maps
		valValue := reflect.ValueOf(val)
		if valValue.Kind() != reflect.String && valValue.Kind() != reflect.Map {
			continue
		}
		valStr := fmt.Sprintf("%v", val)

		// If not one of these, just skip it
		switch attr.Type {
		case BOOLEAN, DECIMAL, INTEGER, UINTEGER:
			if newVal, ok := ConvertString(valStr, attr.Type); ok {
				// Replace the string with the non-string value
				e.NewObject[key] = newVal
			}
		case MAP:
			if valValue.Kind() == reflect.Map {
				valMap := val.(map[string]any)
				for k, v := range valMap {
					vStr := fmt.Sprintf("%v", v)
					// Only saved the converted string if we did a conversion
					if nV, ok := ConvertString(vStr, attr.Item.Type); ok {
						valMap[k] = nV
					}
				}
			}
		}
	}
}

func ConvertString(val string, toType string) (any, bool) {
	switch toType {
	case BOOLEAN:
		if val == "true" {
			return true, true
		} else if val == "false" {
			return false, true
		}
	case DECIMAL:
		tmpFloat, err := strconv.ParseFloat(val, 64)
		if err == nil {
			return tmpFloat, true
		}
	case INTEGER, UINTEGER:
		tmpInt, err := strconv.Atoi(val)
		if err == nil {
			return tmpInt, true
		}
	}
	return nil, false
}

// Returns the initial set of attributes defined for the entity. So
// no IfValues attributes yet as we need the current set of properties
// to calculate that
func (e *Entity) GetBaseAttributes() Attributes {
	attrs := Attributes{}
	// level := 0
	singular := ""

	// Add user-defined attributes
	// TODO check for conflicts with xReg defined ones
	paths := strings.Split(e.Abstract, string(DB_IN))
	if len(paths) == 0 || paths[0] == "" {
		maps.Copy(attrs, e.Registry.Model.Attributes)
	} else {
		// level = len(paths)
		gm := e.Registry.Model.Groups[paths[0]]
		PanicIf(gm == nil, "Can't find Group %q", paths[0])
		if len(paths) == 1 {
			maps.Copy(attrs, gm.Attributes)
		} else {
			rm := gm.Resources[paths[1]]
			PanicIf(rm == nil, "Cant find Resource %q", paths[1])
			maps.Copy(attrs, rm.Attributes)
			singular = rm.Singular
		}
	}

	// Add xReg defied attributes
	// TODO Check for conflicts
	/*
		for _, specProp := range OrderedSpecProps {
			if specProp.InLevel(level) {
				attrs[specProp.Name] = specProp
			}
		}
	*/

	// Add the RESOURCExxx attributes (for resources and versions)
	if singular != "" {
		checkFn := func(e *Entity) error {
			list := []string{
				singular,
				singular + "url",
				singular + "base64",
				singular + "proxyurl",
			}
			count := 0
			for _, name := range list {
				if v, ok := e.NewObject[name]; ok && !IsNil(v) {
					count++
				}
			}
			if count > 1 {
				return fmt.Errorf("Only one of %s can be present at a time",
					strings.Join(list[:3], ",")) // exclude proxy
			}
			return nil
		}

		// Add resource content attributes
		attrs[singular] = &Attribute{
			Name:    singular,
			Type:    ANY,
			checkFn: checkFn,
			updateFn: func(e *Entity, isNew bool) error {
				v, ok := e.NewObject[singular]
				if ok {
					e.NewObject["#resource"] = v
					// e.NewObject["#resourceURL"] = nil
					delete(e.NewObject, singular)
				}
				return nil
			},
		}
		attrs[singular+"url"] = &Attribute{
			Name:    singular + "url",
			Type:    URL,
			checkFn: checkFn,
			updateFn: func(e *Entity, isNew bool) error {
				v, ok := e.NewObject[singular+"url"]
				if !ok {
					return nil
				}
				e.NewObject["#resource"] = nil
				e.NewObject["#resourceURL"] = v
				delete(e.NewObject, singular+"url")
				return nil
			},
		}
		attrs[singular+"proxyurl"] = &Attribute{
			Name:    singular + "proxyurl",
			Type:    URL,
			checkFn: checkFn,
			updateFn: func(e *Entity, isNew bool) error {
				v, ok := e.NewObject[singular+"proxyurl"]
				if !ok {
					return nil
				}
				e.NewObject["#resource"] = nil
				e.NewObject["#resourceProxyURL"] = v
				delete(e.NewObject, singular+"proxyurl")
				return nil
			},
		}
		attrs[singular+"base64"] = &Attribute{
			Name:    singular + "base64",
			Type:    STRING,
			checkFn: checkFn,
			updateFn: func(e *Entity, isNew bool) error {
				v, ok := e.NewObject[singular+"base64"]
				if !ok {
					return nil
				}
				if !IsNil(v) {
					data := v.(string)
					content, err := base64.StdEncoding.DecodeString(data)
					if err != nil {
						return fmt.Errorf("Error decoding \"%sbase64\" "+
							"attribute: "+"%s", singular, err)
					}
					v = any(content)
				}
				e.NewObject["#resource"] = v
				// e.NewObject["#resourceURL"] = nil
				delete(e.NewObject, singular+"base64")
				return nil
			},
		}
	}

	return attrs
}

func ObjectSetProp(obj map[string]any, pp *PropPath, val any) error {
	// TODO see if we can move this into MaterializeProp
	if pp.Len() == 0 && IsNil(val) {
		// A bit of a special case, not 100% sure if this is ok
		for k, _ := range obj {
			delete(obj, k)
		}
		return nil
	}
	PanicIf(pp.Len() == 0, "Can't be zero w/non-nil val")

	_, err := MaterializeProp(obj, pp, val, NewPP())
	if err != nil {
		return err
	}
	return nil
}

func MaterializeProp(current any, pp *PropPath, val any, prev *PropPath) (any, error) {
	log.VPrintf(4, ">Enter: MaterializeProp(%s)", pp.UI())
	log.VPrintf(4, "<Exit: MaterializeProp")

	// current is existing value, used for adding to maps/arrays
	if pp == nil {
		return val, nil
	}

	var ok bool
	var err error

	part := pp.Parts[0]
	if index := part.Index; index >= 0 {
		// Is an array
		// TODO look for cases where Kind(val) == array too - maybe?
		var daArray []any

		if current != nil {
			daArray, ok = current.([]any)
			if !ok {
				return nil, fmt.Errorf("Attribute %q isn't an array",
					prev.Append(pp.First()).UI())
			}
		}

		// Resize if needed
		if diff := (1 + index - len(daArray)); diff > 0 {
			daArray = append(daArray, make([]any, diff)...)
		}

		// Trim the end of the array if there are nil's
		daArray[index], err = MaterializeProp(daArray[index], pp.Next(), val,
			prev.Append(pp.First()))
		for len(daArray) > 0 && daArray[len(daArray)-1] == nil {
			daArray = daArray[:len(daArray)-1]
		}
		return daArray, err
	}

	// Is a map/object
	// TODO look for cases where Kind(val) == obj/map too - maybe?

	daMap := map[string]any{}
	if !IsNil(current) {
		daMap, ok = current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("Current isn't a map: %T", current)
		}
	}

	res, err := MaterializeProp(daMap[pp.Top()], pp.Next(), val,
		prev.Append(pp.First()))
	if err != nil {
		return nil, err
	}
	if IsNil(res) {
		delete(daMap, pp.Top())
	} else {
		daMap[pp.Top()], err = MaterializeProp(daMap[pp.Top()], pp.Next(), val,
			prev.Append(pp.First()))
	}

	return daMap, err
}

// Doesn't fully validate in the sense that it'll assume read-only fields
// are not worth checking since the server generated them.
// This is mainly used for validating input from a client.
// NOTE!!! This isn't a read-only operation. Normally it would be, but to
// avoid traversing the entity more than once, we will tweak things if needed.
// For example, if a missing attribute has a Default value then we'll add it.
func (e *Entity) Validate() error {
	if e.Level == 2 {
		// Skip Resources // TODO DUG - would prefer to not do this
		return nil
	}

	// Don't touch what was passed in

	attrs := e.GetAttributes(true)

	log.VPrintf(3, "========")
	log.VPrintf(3, "Validating:\n%s", ToJSON(e.NewObject))
	return e.ValidateObject(e.NewObject, attrs, NewPP())
}

// This should be called after all level-specific calculated properties have
// been removed - such as collections
func (e *Entity) ValidateObject(val any, origAttrs Attributes, path *PropPath) error {

	log.VPrintf(3, ">Enter: ValidateObject(path: %s)", path.UI())
	defer log.VPrintf(3, "<Exit: ValidateObject")

	log.VPrintf(3, "Check Obj:\n%s", ToJSON(val))
	log.VPrintf(3, "OrigAttrs:\n%s", ToJSON(SortedKeys(origAttrs)))

	valValue := reflect.ValueOf(val)
	if valValue.Kind() != reflect.Map ||
		valValue.Type().Key().Kind() != reflect.String {

		return fmt.Errorf("Attribute %q must be a map[string] or object",
			path.UI())
	}
	newObj := val.(map[string]any)

	// Convert origAttrs to a slice of *Attribute where "*" is first, if there
	attrs := make([]*Attribute, len(origAttrs))
	allAttrNames := map[string]bool{}
	count := 1
	for _, attr := range origAttrs {
		allAttrNames[attr.Name] = true
		if attr.Name == "*" {
			attrs[0] = attr // "*" must appear first in the slice
		} else if count == len(attrs) {
			attrs[0] = attr // at last one and no "*" so use [0]
		} else {
			attrs[count] = attr
			count++
		}
	}

	// For top-level entities, get the list of possible collections
	collections := []string{}
	if path.Len() == 0 {
		collections = e.GetCollections()
	}

	// Don't touch what was passed in
	objKeys := map[string]bool{}
	for k, _ := range newObj {
		// Skip collection related attributes
		isColl := false
		for _, coll := range collections {
			if k == coll || k == coll+"count" || k == coll+"url" {
				isColl = true
				break
			}
		}
		if !isColl {
			objKeys[k] = true
		}
	}

	attr := (*Attribute)(nil)
	key := ""
	for len(attrs) > 0 {
		l := len(attrs)
		attr = attrs[l-1] // grab last one & remove it
		attrs = attrs[:l-1]

		// Keys are all of the attribute names in newObj we need to check.
		// Normally there's just one (attr.Name) but if attr.Name is "*"
		// then we'll have a list of all remaining attribute names in newObj to
		// check, hence it's a slice not a single string
		keys := []string{}
		if attr.Name != "*" {
			keys = []string{attr.Name}
		} else {
			keys = SortedKeys(objKeys) // no need to be sorted, just grab keys
		}

		// For each attribute (key) in newObj, check its type
		for _, key = range keys {
			if key[0] == '#' {
				continue
			}

			val, ok := newObj[key]

			// A Default value is defined but there's no value, so set it
			// and then let normal processing continue
			if !IsNil(attr.Default) && (!ok || IsNil(val)) {
				newObj[key] = attr.Default
			}

			// Based on the attribute's type check the incoming 'val'.
			// This will check for adherence to the model (eg type),
			// the next section (checkFn) will allow for more detailed
			// checking, like for valid values
			if !IsNil(val) {
				err := e.ValidateAttribute(val, attr, path.P(key))
				if err != nil {
					return err
				}
			}

			// GetAttributes already added IfValues for top-level attributes
			if path.Len() >= 1 && len(attr.IfValues) > 0 {
				valStr := fmt.Sprintf("%v", val)
				for ifValStr, ifValueData := range attr.IfValues {
					if valStr != ifValStr {
						continue
					}

					for _, newAttr := range ifValueData.SiblingAttributes {
						if _, ok := allAttrNames[newAttr.Name]; ok {
							return fmt.Errorf(`Attribute %q has an "ifvalues"`+
								`(%s) that defines a conflictng `+
								`siblingattribute: %s`, path.P(key).UI(),
								valStr, newAttr.Name)
						}
						if newAttr.Name == "*" {
							attrs = append([]*Attribute{newAttr}, attrs...)
						} else {
							attrs = append(attrs, newAttr)
						}
						allAttrNames[newAttr.Name] = true
					}
				}
			}

			// We normally skip read-only attrs, but if it has a checkFn
			// then allow for that to be called
			if attr.ReadOnly {
				// Call the attr's checkFn if there
				if attr.checkFn != nil {
					if err := attr.checkFn(e); err != nil {
						return err
					}
				}

				delete(objKeys, key)
				continue
			}

			// Required but not present - note that nil means will be deleted
			if attr.ClientRequired && (!ok || IsNil(val)) {
				return fmt.Errorf("Required property %q is missing",
					path.P(key).UI())
			}

			// Not ClientRequired && no there (or being deleted)
			if !attr.ClientRequired && (!ok || IsNil(val)) {
				delete(objKeys, key)
				continue
			}

			// Call the attr's checkFn if there - for more refined checks
			if attr.checkFn != nil {
				if err := attr.checkFn(e); err != nil {
					return err
				}
			}

			// Everything is good, so remove it
			delete(objKeys, key)
		}
	}

	// See if we have any extra keys and if so, generate an error
	del := []string{}
	for k, _ := range objKeys {
		if k[0] == '#' {
			del = append(del, k)
		}
	}
	for _, k := range del {
		delete(objKeys, k)
	}
	if len(objKeys) != 0 {
		where := path.UI()
		if where != "" {
			where = " in \"" + where + "\""
		}
		return fmt.Errorf("Invalid extension(s)%s: %s", where,
			strings.Join(SortedKeys(objKeys), ","))
	}

	return nil
}

func (e *Entity) ValidateAttribute(val any, attr *Attribute, path *PropPath) error {
	log.VPrintf(3, ">Enter: ValidateAttribute(%s)", path.UI())
	defer log.VPrintf(3, "<Exit: ValidateAttribute")

	log.VPrintf(3, " val: %v", ToJSON(val))
	log.VPrintf(3, " attr: %v", ToJSON(attr))

	if attr.Type == ANY {
		// All good - let it thru
		return nil
	} else if IsScalar(attr.Type) {
		return e.ValidateScalar(val, attr, path)
	} else if attr.Type == MAP {
		return e.ValidateMap(val, attr.Item, path)
	} else if attr.Type == ARRAY {
		return e.ValidateArray(val, attr.Item, path)
	} else if attr.Type == OBJECT {
		/*
			attrs := e.GetBaseAttributes()
			if useNew {
				attrs.AddIfValuesAttributes(e.NewObject)
			} else {
				attrs.AddIfValuesAttributes(e.Object)
			}
		*/

		return e.ValidateObject(val, attr.Attributes, path)
	}

	ShowStack()
	panic(fmt.Sprintf("Unknown type(%s): %s", path.UI(), attr.Type))
}

func (e *Entity) ValidateMap(val any, item *Item, path *PropPath) error {
	log.VPrintf(3, ">Enter: ValidateMap(%s)", path.UI())
	defer log.VPrintf(3, "<Exit: ValidateMap")

	log.VPrintf(3, " item: %v", ToJSON(item))
	log.VPrintf(3, " val: %v", ToJSON(val))

	if IsNil(val) {
		return nil
	}

	valValue := reflect.ValueOf(val)
	if valValue.Kind() != reflect.Map {
		return fmt.Errorf("Attribute %q must be a map", path.UI())
	}

	// All values in the map must be of the same type
	attr := &Attribute{
		Type:       item.Type,
		Item:       item.Item,
		Attributes: item.Attributes,
	}

	for _, k := range valValue.MapKeys() {
		keyName := k.Interface().(string)
		v := valValue.MapIndex(k).Interface()
		if IsNil(v) {
			continue
		}
		if err := e.ValidateAttribute(v, attr, path.P(keyName)); err != nil {
			return err
		}
	}

	return nil
}

func (e *Entity) ValidateArray(val any, item *Item, path *PropPath) error {
	log.VPrintf(3, ">Enter: ValidateArray(%s)", path.UI())
	defer log.VPrintf(3, "<Exit: ValidateArray")

	log.VPrintf(3, "item: %s", ToJSON(item))
	log.VPrintf(3, "val: %s", ToJSON(val))

	if IsNil(val) {
		return nil
	}

	valValue := reflect.ValueOf(val)
	if valValue.Kind() != reflect.Slice {
		return fmt.Errorf("Attribute %q must be an array", path.UI())
	}

	// All values in the array must be of the same type
	attr := &Attribute{
		Type:       item.Type,
		Item:       item.Item,
		Attributes: item.Attributes,
	}

	for i := 0; i < valValue.Len(); i++ {
		v := valValue.Index(i).Interface()
		if err := e.ValidateAttribute(v, attr, path.I(i)); err != nil {
			return err
		}
	}

	return nil
}

func (e *Entity) ValidateScalar(val any, attr *Attribute, path *PropPath) error {
	log.VPrintf(3, ">Enter: ValidateScalar(%s:%s)", path.UI(), ToJSON(val))
	defer log.VPrintf(3, "<Exit: ValidateScalar")

	valKind := reflect.ValueOf(val).Kind()

	switch attr.Type {
	case BOOLEAN:
		if valKind != reflect.Bool {
			return fmt.Errorf("Attribute %q must be a boolean", path.UI())
		}
	case DECIMAL:
		if valKind != reflect.Int && valKind != reflect.Float64 {
			return fmt.Errorf("Attribute %q must be a decimal", path.UI())
		}
	case INTEGER:
		if valKind == reflect.Float64 {
			f := val.(float64)
			if f != float64(int(f)) {
				return fmt.Errorf("Attribute %q must be an integer", path.UI())
			}
		} else if valKind != reflect.Int {
			return fmt.Errorf("Attribute %q must be an integer", path.UI())
		}
	case UINTEGER:
		i := 0
		if valKind == reflect.Float64 {
			f := val.(float64)
			i = int(f)
			if f != float64(i) {
				return fmt.Errorf("Attribute %q must be a uinteger", path.UI())
			}
		} else if valKind != reflect.Int {
			return fmt.Errorf("Attribute %q must be a uinteger", path.UI())
		} else {
			i = val.(int)
			if valKind != reflect.Int {
				return fmt.Errorf("Attribute %q must be a uinteger", path.UI())
			}
		}
		if i < 0 {
			return fmt.Errorf("Attribute %q must be a uinteger", path.UI())
		}
	case STRING:
		if valKind != reflect.String {
			return fmt.Errorf("Attribute %q must be a string", path.UI())
		}
	case URI:
		if valKind != reflect.String {
			return fmt.Errorf("Attribute %q must be a uri", path.UI())
		}
	case URI_REFERENCE:
		if valKind != reflect.String {
			return fmt.Errorf("Attribute %q must be a uri-reference", path.UI())
		}
	case URI_TEMPLATE:
		if valKind != reflect.String {
			return fmt.Errorf("Attribute %q must be a uri-template", path.UI())
		}
	case URL:
		if valKind != reflect.String {
			return fmt.Errorf("Attribute %q must be a url", path.UI())
		}
	case TIMESTAMP:
		if valKind != reflect.String {
			return fmt.Errorf("Attribute %q must be a timestamp", path.UI())
		}
		str := val.(string)
		_, err := time.Parse(time.RFC3339, str)
		if err != nil {
			return fmt.Errorf("Attribute %q is a malformed timestamp",
				path.UI())
		}
	}

	// don't "return nil" above, we may need to check enum values
	if len(attr.Enum) > 0 && (attr.Strict == nil || *(attr.Strict)) {
		foundOne := false
		valStr := fmt.Sprintf("%v", val)
		for _, enumVal := range attr.Enum {
			enumValStr := fmt.Sprintf("%v", enumVal)
			if enumValStr == valStr {
				foundOne = true
				break
			}
		}
		if !foundOne {
			valids := ""
			for i, v := range attr.Enum {
				if i > 0 {
					valids += ", "
				}
				valids += fmt.Sprintf("%v", v)
			}
			return fmt.Errorf("Attribute %q(%v) must be one of the enum "+
				"values: %s", path.UI(), val, valids)
		}
	}

	return nil
}

func PrepUpdateEntity(e *Entity, isNew bool) error {
	attrs := e.GetAttributes(true)

	for key, _ := range attrs {
		attr := attrs[key]
		if attr.updateFn != nil {
			if err := attr.updateFn(e, isNew); err != nil {
				return err
			}
		}
	}

	return nil
}
