package registry

import (
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
	Level    int // 0=registry, 1=group, 2=resource, 3=version
	Path     string
	Abstract string
	EpochSet bool `json:"-"` // Has epoch been updated this transaction?
}

type EntitySetter interface {
	Get(name string) any
	SetCommit(name string, val any) error // Should never be used
	SetSave(name string, val any) error
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

func (e *Entity) GetAsString(path string) string {
	val := e.Get(path)
	str, _ := val.(string)
	// PanicIf(!ok, fmt.Sprintf("Val: %v  T: %T", val, val))
	return str
}

func (e *Entity) GetAsInt(path string) int {
	val := e.Get(path)
	i, ok := val.(int)
	PanicIf(!ok, fmt.Sprintf("Val: %v  T: %T", val, val))
	return i
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
			`, e.DbSID, e.DbSID, NewPPP("isdefault").DB())
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

	// See if we have an updated value in NewObject, if not grab from Object
	var val any
	if e.NewObject != nil {
		var ok bool
		val, ok, _ = ObjectGetProp(e.NewObject, pp)
		if !ok {
			// TODO: DUG - we should not need this
			// val, _, _ = ObjectGetProp(e.Object, pp)
		}
	} else {
		val, _, _ = ObjectGetProp(e.Object, pp)
	}

	// We used to just grab from Object, not NewObject
	/*
		// An error from ObjectGetProp is ignored because if they tried to
		// go into something incorrect/bad we should just return 'nil'.
		// This may not be the best choice in the long-run - which in case we
		// should return the 'error'
		val, _ , _ := ObjectGetProp(e.Object, pp)
	*/
	log.VPrintf(4, "%s(%s).Get(%s) -> %v", e.Plural, e.UID, name, val)
	return val
}

// Value, Found, Error
func ObjectGetProp(obj any, pp *PropPath) (any, bool, error) {
	return NestedGetProp(obj, pp, NewPP())
}

// Value, Found, Error
func NestedGetProp(obj any, pp *PropPath, prev *PropPath) (any, bool, error) {
	if log.GetVerbose() > 2 {
		log.VPrintf(3, "ObjectGetProp: %q\nobj:\n%s", pp.UI(), ToJSON(obj))
	}
	if pp == nil || pp.Len() == 0 {
		return obj, true, nil
	}
	if IsNil(obj) {
		return nil, false,
			fmt.Errorf("Can't traverse into nothing: %s", prev.UI())
	}

	objValue := reflect.ValueOf(obj)
	part := pp.Parts[0]
	if index := part.Index; index >= 0 {
		// Is an array
		if objValue.Kind() != reflect.Slice {
			return nil, false,
				fmt.Errorf("Can't index into non-array: %s", prev.UI())
		}
		if index < 0 || index >= objValue.Len() {
			return nil, false,
				fmt.Errorf("Array reference %q out of bounds: "+
					"(max:%d-1)", prev.Append(pp.First()).UI(), objValue.Len())
		}
		objValue = objValue.Index(index)
		if objValue.IsValid() {
			obj = objValue.Interface()
		} else {
			panic("help") // Should never get here
			obj = nil
		}
		return NestedGetProp(obj, pp.Next(), prev.Append(pp.First()))
	}

	// Is map/object
	if objValue.Kind() != reflect.Map {
		return nil, false, fmt.Errorf("Can't reference a non-map/object: %s",
			prev.UI())
	}
	if objValue.Type().Key().Kind() != reflect.String {
		return nil, false, fmt.Errorf("Key of %q must be a string, not %s",
			prev.UI(), objValue.Type().Key().Kind())
	}

	objValue = objValue.MapIndex(reflect.ValueOf(pp.Top()))
	if objValue.IsValid() {
		obj = objValue.Interface()
	} else {
		if pp.Next().Len() == 0 {
			return nil, false, nil
		}
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

func RawEntitiesFromQuery(tx *Tx, regID string, query string, args ...any) ([]*Entity, error) {
	log.VPrintf(3, ">Enter: RawEntititiesFromQuery(%s)", query)
	defer log.VPrintf(3, "<Exit: RawEntitiesFromQuery")

	// RegSID,Level,Plural,eSID,UID,PropName,PropValue,PropType,Path,Abstract
	//   0     1      2     3    4     5         6         7     8      9

	if query != "" {
		query = "AND (" + query + ") "
	}
	args = append(append([]any{}, regID), args...)
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
        WHERE e.RegSID=? `+query+` ORDER BY Path`, args...)
	defer results.Close()

	if err != nil {
		return nil, err
	}

	entities := []*Entity{}
	for {
		e, err := readNextEntity(tx, results)
		if err != nil {
			return nil, err
		}
		if e == nil {
			break
		}
		entities = append(entities, e)
	}

	return entities, nil
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
// Should never be used because the act of committing should be done
// by the caller once all of the changes are done. This is a holdover from
// before we had transaction support - once we're sure, delete it
func (e *Entity) SetCommit(path string, val any) error {
	log.VPrintf(3, ">Enter: SetCommit(%s=%v)", path, val)
	defer log.VPrintf(3, "<Exit Set")

	err := e.SetSave(path, val)
	Must(e.tx.Conditional(err))

	return err
}

// Set, Validate and Save to DB but not Commit
func (e *Entity) SetSave(path string, val any) error {
	log.VPrintf(3, ">Enter: SetSave(%s=%v)", path, val)
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
	log.VPrintf(3, ">Enter: JustSet(%s.%s=%v)", e.UID, pp.UI(), val)
	defer log.VPrintf(3, "<Exit: JustSet")

	// Assume no other edits are pending
	// e.Refresh() // trying not to have this here

	if e.NewObject == nil {
		// If we don't have a NewObject yet then this is our first update
		// so clone the current values before adding the new prop/val
		if e.Object == nil {
			e.NewObject = map[string]any{}
		} else {
			e.NewObject = maps.Clone(e.Object)
		}
	}

	// Cheat a little just to make caller's life easier by converting
	// empty structs and maps need to be of the type we like (meaning 'any's)
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
		e.EpochSet = true
	}

	if log.GetVerbose() > 2 {
		log.VPrintf(3, "Abstract/ID: %s/%s", e.Abstract, e.UID)
		log.VPrintf(3, "e.Object:\n%s", ToJSON(e.Object))
		log.VPrintf(3, "e.NewObject:\n%s", ToJSON(e.NewObject))
	}

	return ObjectSetProp(e.NewObject, pp, val)
}

func (e *Entity) ValidateAndSave() error {
	log.VPrintf(3, ">Enter: ValidateAndSave %s/%s", e.Abstract, e.UID)
	defer log.VPrintf(3, "<Exit: ValidateAndSave")

	// Make sure we have a tx since Validate assumes it
	e.tx.NewTx()

	// If nothing changed, just exit
	// if e.NewObject == nil {
	// return nil
	// }

	if log.GetVerbose() > 2 {
		log.VPrintf(3, "Validating %s/%s e.Object:\n%s\n\ne.NewObject:\n%s",
			e.Abstract, e.UID, ToJSON(e.Object), ToJSON(e.NewObject))
	}

	if err := e.Validate(); err != nil {
		return err
	}

	if err := PrepUpdateEntity(e); err != nil {
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
		if log.GetVerbose() > 2 {
			log.VPrintf(3, "SetPP exit: e.Object:\n%s", ToJSON(e.Object))
		}
	}()

	if err := e.JustSet(pp, val); err != nil {
		return err
	}

	err := e.ValidateAndSave()
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
	if sp, ok := SpecProps[pp.Top()]; ok && sp.internals.dontStore {
		return nil
	}

	// Must be a private/temporary prop used internally - don't save it
	if strings.HasPrefix(pp.Top(), "#-") {
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
		ReadOnly:       true,
		Immutable:      true,
		ServerRequired: true,

		internals: AttrInternals{
			levels:    "0",
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
	},
	{
		Name:           "id",
		Type:           STRING,
		Immutable:      true,
		ServerRequired: true,

		internals: AttrInternals{
			levels:    "",
			dontStore: false,
			getFn:     nil,
			checkFn: func(e *Entity) error {
				oldID := any(e.UID)
				newID := any(e.NewObject["id"])

				if IsNil(newID) {
					return nil // Not trying to be updated, so skip it
				}

				if newID == "" {
					return fmt.Errorf("ID can't be an empty string")
				}

				if oldID != "" && newID != oldID {
					return fmt.Errorf("Can't change the ID of an "+
						"entity(%s->%s)", oldID, newID)
				}
				return nil
			},
			updateFn: func(e *Entity) error {
				// Make sure the ID is always set
				e.NewObject["id"] = e.UID
				return nil
			},
		},
	},
	{
		Name: "name",
		Type: STRING,

		internals: AttrInternals{
			levels:    "",
			dontStore: false,
			getFn:     nil,
			checkFn:   nil,
			updateFn:  nil,
		},
	},
	{
		Name:           "epoch",
		Type:           UINTEGER,
		ServerRequired: true,

		internals: AttrInternals{
			levels:    "013",
			dontStore: false,
			getFn:     nil,
			checkFn: func(e *Entity) error {
				// If we explicitly setEpoch via internal API then don't check
				if e.EpochSet {
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
					return fmt.Errorf("Attribute %q(%d) doesn't match "+
						"existing value (%d)", "epoch", newEpoch, oldEpoch)
				}
				return nil
			},
			updateFn: func(e *Entity) error {
				// If we already set Epoch in this Tx, just exit
				if e.EpochSet {
					return nil
				}

				// This assumes that ALL entities must have an Epoch property
				// that we wnt to set. At one point this wasn't true for
				// Resources but hopefully that's no loner true

				oldEpoch := e.Object["epoch"]
				epoch := NotNilInt(&oldEpoch)

				e.NewObject["epoch"] = epoch + 1
				e.EpochSet = true
				return nil
			},
		},
	},
	{
		Name:           "self",
		Type:           URL,
		ReadOnly:       true,
		ServerRequired: true,

		internals: AttrInternals{
			levels:    "",
			dontStore: false,
			getFn: func(e *Entity, info *RequestInfo) any {
				base := ""
				if info != nil {
					base = info.BaseURL
				}
				if e.Level > 1 {
					meta := info != nil && (info.ShowMeta || info.ResourceUID == "")
					_, rm := e.GetModels()
					if rm.GetHasDocument() == false {
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
	},
	{
		Name:     "isdefault",
		Type:     BOOLEAN,
		ReadOnly: true,

		internals: AttrInternals{
			levels:    "3",
			dontStore: true,
			getFn:     nil,
			checkFn:   nil,
			updateFn: func(e *Entity) error {
				// TODO if set, set defaultversionid in the resource to this
				// guy's UID

				return nil
			},
		},
	},
	{
		Name:     "stickydefaultversion",
		Type:     BOOLEAN,
		ReadOnly: true,

		internals: AttrInternals{
			levels:    "2",
			dontStore: false,
			getFn:     nil,
			checkFn:   nil,
			updateFn:  nil,
		},
	},
	{
		Name:     "defaultversionid",
		Type:     STRING,
		ReadOnly: true,
		// ServerRequired: true,

		internals: AttrInternals{
			levels:    "2",
			dontStore: false,
			getFn:     nil,
			checkFn:   nil,
			updateFn:  nil,
		},
	},
	{
		Name:     "defaultversionurl",
		Type:     URL,
		ReadOnly: true,
		// ServerRequired: true,

		internals: AttrInternals{
			levels:    "2",
			dontStore: false,
			getFn: func(e *Entity, info *RequestInfo) any {
				val := e.Object["defaultversionid"]
				if IsNil(val) {
					return nil
				}
				base := ""
				if info != nil {
					base = info.BaseURL
				}

				tmp := base + "/" + e.Path + "/versions/" + val.(string)

				meta := info != nil && (info.ShowMeta || info.ResourceUID == "")
				_, rm := e.GetModels()
				if rm.GetHasDocument() == false {
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
	},
	{
		Name: "description",
		Type: STRING,

		internals: AttrInternals{
			levels:    "",
			dontStore: false,
			getFn:     nil,
			checkFn:   nil,
			updateFn:  nil,
		},
	},
	{
		Name: "documentation",
		Type: URL,

		internals: AttrInternals{
			levels:    "",
			dontStore: false,
			getFn:     nil,
			checkFn:   nil,
			updateFn:  nil,
		},
	},
	{
		Name: "labels",
		Type: MAP,
		Item: &Item{
			Type: STRING,
		},

		internals: AttrInternals{
			levels:    "",
			dontStore: false,
			getFn:     nil,
			checkFn:   nil,
			updateFn:  nil,
		},
	},
	{
		Name: "origin",
		Type: URI,

		internals: AttrInternals{
			levels:    "123",
			dontStore: false,
			getFn:     nil,
			checkFn:   nil,
			updateFn:  nil,
		},
	},
	{
		Name: "createdat",
		Type: TIMESTAMP,

		internals: AttrInternals{
			levels:    "013",
			dontStore: false,
			getFn:     nil,
			checkFn:   nil,
			updateFn: func(e *Entity) error {
				ca, ok := e.NewObject["createdat"]
				// If not there use the existing value, if present
				if !ok {
					ca = e.Object["createdat"]
					e.NewObject["createdat"] = ca
				}
				// Still no value, so use "now"
				if IsNil(ca) {
					e.NewObject["createdat"] = e.tx.CreateTime
				}
				return nil
			},
		},
	},
	{
		Name: "modifiedat",
		Type: TIMESTAMP,

		internals: AttrInternals{
			levels:    "013",
			dontStore: false,
			getFn:     nil,
			checkFn:   nil,
			updateFn: func(e *Entity) error {
				ma := e.NewObject["modifiedat"]
				// If there's no value, or it's the same as the existing
				// value, set to "now"
				if IsNil(ma) || (ma == e.Object["modifiedat"]) {
					e.NewObject["modifiedat"] = e.tx.CreateTime
				}
				return nil
			},
		},
	},
	{
		Name: "contenttype",
		Type: STRING,

		internals: AttrInternals{
			levels:     "23",
			dontStore:  false,
			httpHeader: "Content-Type",
			getFn:      nil,
			checkFn:    nil,
			updateFn:   nil,
		},
	},
	{
		Name:     "model",
		Type:     OBJECT,
		ReadOnly: true,
		Attributes: Attributes{
			"*": &Attribute{
				Name: "*",
				Type: ANY,
			},
		},

		internals: AttrInternals{
			levels:    "0",
			dontStore: false,
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
	},
}

var SpecProps = map[string]*Attribute{}

func init() {
	// Load map via lower-case version of prop name
	for _, sp := range OrderedSpecProps {
		SpecProps[sp.Name] = sp
	}
}

// This is used to serialize an Entity regardless of the format.
// This will:
//   - Use AddCalcProps() to fill in any missing props (eg Entity's getFn())
//   - Call that passed-in 'fn' to serialize each prop but in the right order
//     as defined by OrderedSpecProps
func (e *Entity) SerializeProps(info *RequestInfo,
	fn func(*Entity, *RequestInfo, string, any, *Attribute) error) error {

	daObj := e.AddCalcProps(info)
	attrs := e.GetAttributes(e.Object)

	if log.GetVerbose() > 3 {
		log.VPrintf(4, "SerProps.Obj: %s", ToJSON(e.Object))
		log.VPrintf(4, "SerProps daObj: %s", ToJSON(daObj))
	}

	// Do spec defined props first, in order
	for _, prop := range OrderedSpecProps {
		attr, ok := attrs[prop.Name]
		if !ok {
			delete(daObj, prop.Name)
			continue // not allowed at this level so skip it
		}

		if val, ok := daObj[prop.Name]; ok {
			if !IsNil(val) {
				if err := fn(e, info, prop.Name, val, attr); err != nil {
					return err
				}
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
	log.VPrintf(3, ">Enter: Save(%s/%s)", e.Abstract, e.UID)
	defer log.VPrintf(3, "<Exit: Save")

	// TODO remove at some point when we're sure it's safe
	if SpecProps["epoch"].InLevel(e.Level) && IsNil(e.NewObject["epoch"]) {
		log.Printf("Save.NewObject:\n%s", ToJSON(e.NewObject))
		panic("Epoch is nil")
	}

	if log.GetVerbose() > 2 {
		log.VPrintf(3, "Saving - %s (id:%s):\n%s\n", e.Abstract, e.UID,
			ToJSON(e.NewObject))
	}

	// make a dup so we can delete some attributes
	newObj := maps.Clone(e.NewObject)

	// TODO calculate which to delete based on attr properties
	delete(newObj, "self")

	e.RemoveCollections(newObj)

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

// This will add in the calculated properties into the entity. This will
// normally be called after a query using FullTree view and before we serialize
// the entity we need to add the non-DB-stored properties (meaning, the
// calculated ones.
// Note that we make a copy and don't touch the entity itself. Serializing
// an entity shouldn't have side-effects.
func (e *Entity) AddCalcProps(info *RequestInfo) map[string]any {
	mat := maps.Clone(e.Object)

	// Regardless of the type of entity, set the generated properties
	for _, prop := range OrderedSpecProps {
		// Only generate props that are for this level, and have a Fn
		if prop.internals.getFn == nil || !prop.InLevel(e.Level) {
			continue
		}

		// Only generate/set the value if it's not already set
		if _, ok := mat[prop.Name]; !ok {
			if val := prop.internals.getFn(e, info); !IsNil(val) {
				// Only write it if we have a value
				mat[prop.Name] = val
			}
		}
	}

	return mat
}

// This will remove all Collection related attributes from the entity.
// While this is an Entity.Func, we allow callers to pass in the Object
// data to use instead of the e.Object/NewObject so that we'll use this
// Entity's Level (which tells us which collections it has), on the 'obj'.
// This is handy for cases where we need to remove the Resource's collections
// from a Version's Object - like on  a PUT to /GROUPs/gID/RESOURECEs/rID
// where we're passing in what looks like a Resource entity, but we're
// really using it to create a Version
func (e *Entity) RemoveCollections(obj Object) {
	if obj == nil {
		obj = e.NewObject
	}

	for _, coll := range e.GetCollections() {
		delete(obj, coll)
		delete(obj, coll+"count")
		delete(obj, coll+"url")
	}
}

func (e *Entity) GetCollections() []string {
	switch e.Level {
	case 0:
		return SortedKeys(e.Registry.Model.Groups)
	case 1:
		gm, _ := e.GetModels()
		return SortedKeys(gm.Resources)
	case 2:
		return []string{"versions"}
	case 3:
		return nil
	}
	panic(fmt.Sprintf("bad level: %d", e.Level))
	return nil
}

func (e *Entity) GetAttributes(obj Object) Attributes {
	attrs := e.GetBaseAttributes()
	if obj == nil {
		obj = e.NewObject
	}

	attrs.AddIfValuesAttributes(obj)

	return attrs
}

// Returns the initial set of attributes defined for the entity.
func (e *Entity) GetBaseAttributes() Attributes {
	// Add attributes from the model (core and user-defined)
	gm, rm := e.GetModels()

	if gm == nil {
		return e.Registry.Model.GetBaseAttributes()
	}
	if rm == nil {
		return gm.GetBaseAttributes()
	}
	return rm.GetBaseAttributes()
}

// Given a PropPath and a value this will add the necessary golang data
// structures to 'obj' to materialize PropPath and set the appropriate
// fields to 'val'
func ObjectSetProp(obj map[string]any, pp *PropPath, val any) error {
	log.VPrintf(4, "ObjectSetProp(%s=%v)", pp.UI(), val)
	if pp.Len() == 0 && IsNil(val) {
		// A bit of a special case, not 100% sure if this is ok.
		// Treat nil val as a request to delete all properties.
		// e.g. obj={}
		for k, _ := range obj {
			delete(obj, k)
		}
		return nil
	}
	PanicIf(pp.Len() == 0, "Can't be zero w/non-nil val")

	_, err := MaterializeProp(obj, pp, val, NewPP())
	return err
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
	attrs := e.GetAttributes(e.NewObject)

	if log.GetVerbose() > 2 {
		log.VPrintf(3, "========")
		log.VPrintf(3, "Validating:\n%s", ToJSON(e.NewObject))
	}
	return e.ValidateObject(e.NewObject, attrs, NewPP())
}

// This should be called after all level-specific calculated properties have
// been removed - such as collections
func (e *Entity) ValidateObject(val any, origAttrs Attributes, path *PropPath) error {

	log.VPrintf(3, ">Enter: ValidateObject(path: %s)", path.UI())
	defer log.VPrintf(3, "<Exit: ValidateObject")

	if log.GetVerbose() > 2 {
		log.VPrintf(3, "Check Obj:\n%s", ToJSON(val))
		log.VPrintf(3, "OrigAttrs:\n%s", ToJSON(SortedKeys(origAttrs)))
	}

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

	// Don't touch what was passed in by just saving the keys and then
	// removing the ones we don't want from it (ie. the collections ones)
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
						// add new attr to the list so we can check its ifValues
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
				if attr.internals.checkFn != nil {
					if err := attr.internals.checkFn(e); err != nil {
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
			if attr.internals.checkFn != nil {
				if err := attr.internals.checkFn(e); err != nil {
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

	if log.GetVerbose() > 2 {
		log.VPrintf(3, " val: %v", ToJSON(val))
		log.VPrintf(3, " attr: %v", ToJSON(attr))
	}

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

	if log.GetVerbose() > 2 {
		log.VPrintf(3, " item: %v", ToJSON(item))
		log.VPrintf(3, " val: %v", ToJSON(val))
	}

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

	if log.GetVerbose() > 2 {
		log.VPrintf(3, "item: %s", ToJSON(item))
		log.VPrintf(3, "val: %s", ToJSON(val))
	}

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
	if log.GetVerbose() > 2 {
		log.VPrintf(3, ">Enter: ValidateScalar(%s:%s)", path.UI(), ToJSON(val))
		defer log.VPrintf(3, "<Exit: ValidateScalar")
	}

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
	if len(attr.Enum) > 0 && attr.GetStrict() {
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

func (e *Entity) GetModels() (*GroupModel, *ResourceModel) {
	return AbstractToModels(e.Registry, e.Abstract)
}

func PrepUpdateEntity(e *Entity) error {
	attrs := e.GetAttributes(e.NewObject)

	for key, _ := range attrs {
		attr := attrs[key]

		// Any ReadOnly attribute in Object, but not in NewObject, must
		// be one that we want to keep around. Note that a 'nil' in NewObject
		// will not grab the one in Object - assumes we want to erase the val
		/*
			if attr.ReadOnly {
				oldVal, ok1 := e.Object[attr.Name]
				_, ok2 := e.NewObject[attr.Name]
				if ok1 && !ok2 {
					e.NewObject[attr.Name] = oldVal
				}
			}
		*/

		if attr.InLevel(e.Level) && attr.internals.updateFn != nil {
			if err := attr.internals.updateFn(e); err != nil {
				return err
			}
		}
	}

	return nil
}
