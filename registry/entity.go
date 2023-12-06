package registry

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
	_ "github.com/go-sql-driver/mysql"
)

type EType interface {
	String() string
	Type() string
}

type Entity struct {
	RegistrySID string
	DbSID       string // Entity's SID
	Plural      string
	UID         string  // Entity's UID
	Props       EObject // map[string]EType // any

	// These were added just for convinience and so we can use the same
	// struct for traversing the SQL results
	Level    int
	Path     string
	Abstract string
}

type EntitySetter interface {
	Get(name string) any
	Set(name string, val any) error
}

var _ EType = EArray(nil)
var _ EType = EBoolean(true)
var _ EType = EDecimal(1.0)
var _ EType = EInt(1)
var _ EType = EMap(nil)
var _ EType = EObject(nil)
var _ EType = EString("hi")
var _ EType = ETime("")
var _ EType = EUInt(0)
var _ EType = EURI("")
var _ EType = EURIReference("")
var _ EType = EURITemplate("")
var _ EType = EURL("")

func ToGoType(s string) reflect.Type {
	switch s {
	case BOOLEAN:
		return reflect.TypeOf(true)
	case DECIMAL:
		return reflect.TypeOf(float64(1.1))
	case INT:
		return reflect.TypeOf(int(1))
	case TIME, URI, URI_REFERENCE, URI_TEMPLATE, URL:
		return reflect.TypeOf("")
	case UINT:
		return reflect.TypeOf(uint(0))
	}
	panic("ToGoType - not supported: " + s)
}

type EArray []any

func (ea EArray) String() string     { return "ARRAY" }
func (ea EArray) Type() string       { return ARRAY }
func StringToEArray(s string) EArray { return EArray{} }

type EBoolean bool

func (eb EBoolean) String() string       { return fmt.Sprintf("%v", bool(eb)) }
func (eb EBoolean) Type() string         { return BOOLEAN }
func StringToEBoolean(s string) EBoolean { return s == "true" }

type EDecimal float64

func (ed EDecimal) String() string { return fmt.Sprintf("%v", float64(ed)) }
func (ed EDecimal) Type() string   { return DECIMAL }
func StringToEDecimal(s string) EDecimal {
	ed, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(fmt.Sprintf("Bad format for decimal %q: %s", s, err))
	}
	return EDecimal(ed)
}

type EInt int

func (ei EInt) String() string { return fmt.Sprintf("%v", int(ei)) }
func (ei EInt) Type() string   { return INT }
func StringToEInt(s string) EInt {
	ei, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("Bad format for int %q: %s", s, err))
	}
	return EInt(ei)
}

type EMap map[any]any

func (ei EMap) String() string   { return "MAP" }
func (ei EMap) Type() string     { return MAP }
func StringToEMap(s string) EMap { return EMap{} }

type EObject map[string]any

func (ei EObject) String() string      { return "OBJECT" }
func (ei EObject) Type() string        { return OBJECT }
func StringToEObject(s string) EObject { return EObject{} }

type EString string

func (es EString) String() string      { return string(es) }
func (ei EString) Type() string        { return STRING }
func StringToEString(s string) EString { return EString(s) }

type ETime string

func (et ETime) String() string    { return string(et) }
func (ei ETime) Type() string      { return TIME }
func StringToETime(s string) ETime { return ETime(s) }

type EUInt uint

func (eui EUInt) String() string { return fmt.Sprintf("%v", uint(eui)) }
func (ei EUInt) Type() string    { return UINT }
func StringToEUInt(s string) EUInt {
	eui, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("Bad format for int %q: %s", s, err))
	}
	return EUInt(eui)
}

type EURI string

func (eu EURI) String() string   { return string(eu) }
func (ei EURI) Type() string     { return URI }
func StringToEURI(s string) EURI { return EURI(s) }

type EURIReference string

func (eur EURIReference) String() string           { return string(eur) }
func (ei EURIReference) Type() string              { return URI_REFERENCE }
func StringToEURIReference(s string) EURIReference { return EURIReference(s) }

type EURITemplate string

func (eut EURITemplate) String() string          { return string(eut) }
func (ei EURITemplate) Type() string             { return URI_TEMPLATE }
func StringToEURITemplate(s string) EURITemplate { return EURITemplate(s) }

type EURL string

func (eu EURL) String() string   { return string(eu) }
func (ei EURL) Type() string     { return URL }
func StringToEURL(s string) EURL { return EURL(s) }

func (e *Entity) GetPropFromUI(name string) any {
	pp, err := PropPathFromUI(name)
	PanicIf(err != nil, fmt.Sprintf("%s", err))
	return e.GetPropPP(pp)
}

func (e *Entity) GetPropFromDB(name string) any {
	pp, err := PropPathFromDB(name)
	PanicIf(err != nil, fmt.Sprintf("%s", err))
	return e.GetPropPP(pp)
}

func (e *Entity) GetPropPP(pp *PropPath) any {
	name := pp.DB()
	if pp.Len() == 1 && pp.Top() == "#resource" {
		// if name == "#resource" {
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

func (e *Entity) Set(name string, val any) error {
	fmt.Printf("E: %#v\n", e)
	panic("NO!")
}

func RawEntityFromPath(regID string, path string) (*Entity, error) {
	log.VPrintf(3, ">Enter: RawEntityFromPath(%s)", path)
	defer log.VPrintf(3, "<Exit: RawEntityFromPath")

	// RegSID,Level,Plural,eSID,UID,PropName,PropValue,PropType,Path,Abstract
	//   0     1      2     3    4     5         6         7     8      9

	results, err := Query(`
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

	entity := readNextEntity(results)
	return entity, nil
}

func (e *Entity) Find() (bool, error) {
	log.VPrintf(3, ">Enter: Find(%s)", e.UID)
	defer log.VPrintf(3, "<Exit: Find")

	// TODO NEED REGID

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

		switch propType {
		case ARRAY:
			Panicf("Not supported: %s", propType)
		case BOOLEAN:
			e.Props[name] = (val == "true")
		case DECIMAL:
			tmpFloat, err := strconv.ParseFloat(val, 64)
			if err != nil {
				panic(fmt.Sprintf("error parsing float: %s", val))
			}
			e.Props[name] = tmpFloat
		case INT:
			tmpInt, err := strconv.Atoi(val)
			if err != nil {
				panic(fmt.Sprintf("error parsing int: %s", val))
			}
			e.Props[name] = tmpInt
		case MAP:
			Panicf("Not supported: %s", propType)
		case OBJECT:
			Panicf("Not supported: %s", propType)
		case STRING:
			e.Props[name] = val
		case TIME:
			e.Props[name] = val
		case UINT:
			tmpInt, err := strconv.Atoi(val)
			if err != nil {
				panic(fmt.Sprintf("error parsing int: %s", val))
			}
			e.Props[name] = tmpInt
		case URI:
			e.Props[name] = val
		case URI_REFERENCE:
			e.Props[name] = val
		case URI_TEMPLATE:
			e.Props[name] = val
		case URL:
			e.Props[name] = val
		default:
			panic(fmt.Sprintf("Unknown type: %s", propType))
		}
	}
	return nil
}

var RegexpPropName = regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_./]*$")
var RegexpMapKey = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9_.\\-]*$")

// Maybe replace error with a panic?
func SetPropFromDB(entity any, name string, val any) error {
	pp, err := PropPathFromDB(name)
	if err != nil {
		return err
	}
	return SetPropPP(entity, pp, val)
}

func SetPropFromUI(entity any, name string, val any) error {
	pp, err := PropPathFromUI(name)
	if err != nil {
		return err
	}
	return SetPropPP(entity, pp, val)
}

func SetPropPP(entity any, pp *PropPath, val any) error {
	name := pp.DB()
	log.VPrintf(3, ">Enter: SetPropPP(%s=%v)", pp, val)
	defer log.VPrintf(3, "<Exit SetPropPP")

	if pp.Top() == "labels" {
		if pp.Len() == 1 {
			return fmt.Errorf("Invalid property name: %s", pp.Top())
		}
		mapName := pp.Top()
		key := pp.Next().Top()
		if len(key) == 0 {
			return fmt.Errorf("Map %q key is empty", mapName)
		}
		if !RegexpMapKey.MatchString(key) {
			return fmt.Errorf("Invalid label key: %s", key)
		}
	} else if pp.Top()[0] != '#' && !RegexpPropName.MatchString(pp.Top()) {
		return fmt.Errorf("Invalid property name: %s", pp.Top())
	}

	e := (*Entity)(nil)
	if reflect.TypeOf(entity) == reflect.TypeOf((*Entity)(nil)) {
		e = entity.(*Entity)
	} else {
		eField := reflect.ValueOf(entity).Elem().FieldByName("Entity")
		if !eField.IsValid() {
			panic(fmt.Sprintf("Passing a non-entity to SetProp: %#v", entity))
			// e = entity.(*Entity)
		} else {
			e = eField.Addr().Interface().(*Entity)
		}
	}

	if e.DbSID == "" {
		log.Fatalf("DbSID should not be empty")
	}
	if e.RegistrySID == "" {
		log.Fatalf("RegistrySID should not be empty")
	}

	// Check to see if attribute is defined in the model
	attrType, err := GetAttributeType(e.RegistrySID, e.Abstract, pp)
	if err != nil {
		// log.Printf("Error on getAttr(%s): %s", pp.UI(), err)
		return err
	}
	if attrType == "" && name[0] != '#' {
		return fmt.Errorf("Can't find attribute %q", pp.UI())
	}

	// #resource is special and is saved in it's own table
	if pp.Len() == 1 && pp.Top() == "#resource" {
		// if name == "#resource" {
		if IsNil(val) {
			err = Do(`DELETE FROM ResourceContents WHERE VersionSID=?`, e.DbSID)
		} else {
			// The actual contents
			err = DoOneTwo(`
                REPLACE INTO ResourceContents(VersionSID, Content)
            	VALUES(?,?)`, e.DbSID, val)
		}
		return err
	}

	if IsNil(val) {
		err = Do(`DELETE FROM Props WHERE EntitySID=? and PropName=?`,
			e.DbSID, name)
	} else {
		dbVal := val
		propType := ""
		k := reflect.ValueOf(val).Type().Kind()
		if k == reflect.Bool {
			propType = BOOLEAN
			if val.(bool) {
				dbVal = "true"
			} else {
				dbVal = "false"
			}
		} else if k == reflect.String {
			propType = STRING
		} else if k == reflect.Int {
			propType = INT
		} else if k == reflect.Float64 {
			propType = DECIMAL
		} else {
			panic(fmt.Sprintf("Bad property kind: %s", k.String()))
		}
		if attrType != "" {
			propType = attrType
		}
		err = DoOneTwo(`
            REPLACE INTO Props(
              RegistrySID, EntitySID, PropName, PropValue, PropType)
            VALUES( ?,?,?,?,? )`,
			e.RegistrySID, e.DbSID, name, dbVal, propType)
	}

	if err != nil {
		log.Printf("Error updating prop(%s/%v): %s", pp.UI(), val, err)
		return fmt.Errorf("Error updating prop(%s/%v): %s", pp.UI(), val, err)
	}

	field := reflect.ValueOf(e).Elem().FieldByName("Props")
	if !field.IsValid() {
		panic(fmt.Sprintf("Can't find Props(%s)", e.DbSID))
	}

	if val == nil {
		if field.IsNil() {
			return nil // already gone
		}
		// Delete map key
		field.SetMapIndex(reflect.ValueOf(name), reflect.Value{})
	} else {
		if field.IsNil() {
			// Props is nil so create an empty map
			field.Set(reflect.ValueOf(map[string]any{}))
		}
		// Add key/value to the Props map
		field.SetMapIndex(reflect.ValueOf(name), reflect.ValueOf(val))
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
				Props:       EObject{}, // map[string]any{},

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

		if propType == STRING {
			entity.Props[propName] = propVal
		} else if propType == BOOLEAN {
			entity.Props[propName] = (propVal == "true")
		} else if propType == INT {
			tmpInt, err := strconv.Atoi(propVal)
			if err != nil {
				panic(fmt.Sprintf("error parsing int: %s", propVal))
			}
			entity.Props[propName] = tmpInt
		} else if propType == DECIMAL {
			tmpFloat, err := strconv.ParseFloat(propVal, 64)
			if err != nil {
				panic(fmt.Sprintf("error parsing float: %s", propVal))
			}
			entity.Props[propName] = tmpFloat
		} else {
			panic(fmt.Sprintf("bad type(%s): %v", propType, propType))
		}
	}

	return entity
}

type SpecProp struct {
	name           string // prop name
	daType         string
	levels         string                          // only show for these levels
	mutable        bool                            // user editable
	fn             func(*Entity, *RequestInfo) any // caller will Marshal the 'any'
	modelAttribute *Attribute
}

// This allows for us to choose the order and define custom logic per prop
var OrderedSpecProps = []*SpecProp{
	{"specVersion", STRING, "0", false, nil, &Attribute{
		Name:     "specVersion",
		Type:     STRING,
		Required: true,
	}},
	{"id", STRING, "", false, nil, &Attribute{
		Name:     "id",
		Type:     STRING,
		Required: true,
	}},
	{"name", STRING, "", true, nil, &Attribute{
		Name:     "name",
		Type:     STRING,
		Required: true,
	}},
	{"epoch", INT, "", false, nil, &Attribute{
		Name:     "epoch",
		Type:     INT,
		Required: true,
	}},
	{"self", STRING, "", false, func(e *Entity, info *RequestInfo) any {
		return info.BaseURL + "/" + e.Path
	}, &Attribute{
		Name:     "self",
		Type:     STRING,
		Required: true,
	}},
	{"latest", BOOLEAN, "3", false, nil, &Attribute{
		Name:     "latest",
		Type:     BOOLEAN,
		Required: true,
	}},
	{"latestVersionId", STRING, "2", false, nil, &Attribute{
		Name:     "latestVersionId",
		Type:     STRING,
		Required: true,
	}},
	{"latestVersionUrl", URL, "2", false, func(e *Entity, info *RequestInfo) any {
		val := e.Props[NewPPP("latestVersionId").DB()]
		if IsNil(val) {
			return nil
		}
		return info.BaseURL + "/" + e.Path + "/versions/" + val.(string)
	}, &Attribute{
		Name:     "latestVersionUrl",
		Type:     URL,
		Required: true,
	}},
	{"description", STRING, "", true, nil, &Attribute{
		Name: "description",
		Type: STRING,
	}},
	{"documentation", STRING, "", true, nil, &Attribute{
		Name: "description",
		Type: STRING,
	}},
	{"labels", MAP, "", true, func(e *Entity, info *RequestInfo) any {
		var res map[string]string

		for _, key := range SortedKeys(e.Props) {
			if key[0] > 't' { // Why t and not l ? can't remember. typo?
				break
			}

			pp, _ := PropPathFromDB(key)
			if pp.Len() == 2 && pp.Top() == "labels" {
				val, _ := e.Props[key]
				if res == nil {
					res = map[string]string{}
				}
				// Convert it to a string per the spec
				res[pp.Next().Top()] = fmt.Sprintf("%v", val)
			}
		}
		return res
	}, &Attribute{
		Name:     "labels",
		Type:     MAP,
		KeyType:  STRING,
		ItemType: STRING,
	}},
	{"format", STRING, "23", true, nil, &Attribute{
		Name: "format",
		Type: STRING,
	}},
	{"createdBy", STRING, "", false, nil, &Attribute{
		Name: "createdBy",
		Type: STRING,
	}},
	{"createdOn", TIME, "", false, nil, &Attribute{
		Name: "createdOn",
		Type: TIME,
	}},
	{"modifiedBy", STRING, "", false, nil, &Attribute{
		Name: "modifiedBy",
		Type: STRING,
	}},
	{"modifiedOn", TIME, "", false, nil, &Attribute{
		Name: "modifiedOn",
		Type: TIME,
	}},
	{"model", OBJECT, "0", false, func(e *Entity, info *RequestInfo) any {
		if info.ShowModel {
			model := info.Registry.Model
			if model == nil {
				model = &Model{}
			}
			httpModel := model // ModelToHTTPModel(model)
			return httpModel
		}
		return nil
	}, nil},
}

var SpecProps = map[string]*SpecProp{}

func init() {
	// Load map via lower-case version of prop name
	for _, sp := range OrderedSpecProps {
		SpecProps[strings.ToLower(sp.name)] = sp
	}
}

// This is used to serialize Prop regardless of the format.
func (e *Entity) SerializeProps(info *RequestInfo,
	fn func(*Entity, *RequestInfo, string, any) error) error {

	daObj := e.Materialize(info)

	// fmt.Printf("----\n")
	// Do spec defined props first, in order
	for _, prop := range OrderedSpecProps {
		// fmt.Printf("Trying to serialize: %s\n", prop.name)
		if val, ok := daObj[prop.name]; ok {
			// fmt.Printf("  found it\n")
			if err := fn(e, info, prop.name, val); err != nil {
				log.Printf("Error serializing %q(%v): %s", prop.name, val, err)
				return err
			}
			delete(daObj, prop.name)
		}
	}

	// Now do all other props (extensions) alphabetically
	for _, key := range SortedKeys(daObj) {
		val, _ := daObj[key]

		if err := fn(e, info, key, val); err != nil {
			log.Printf("Error serializing %q(%v): %s", key, val, err)
			return err
		}
	}

	return nil
}

func (e *Entity) Materialize(info *RequestInfo) map[string]any {
	result := map[string]any{}
	usedProps := map[string]bool{}

	for _, prop := range OrderedSpecProps {
		pp := NewPPP(prop.name)
		propName := pp.DB()
		usedProps[propName] = true

		// Only show props that are for this level
		ch := rune('0' + byte(e.Level))
		if prop.levels != "" && !strings.ContainsRune(prop.levels, ch) {
			continue
		}

		// Even if it has a func, if there's a val in Values let it override
		val, ok := e.Props[propName]
		if !ok && prop.fn != nil {
			val = prop.fn(e, info)
		}

		// Only write it if we have a value
		if !IsNil(val) {
			// result[pp.UI()] = val
			result[pp.Top()] = val
		}
	}

	for key, val := range e.Props {
		pp, _ := PropPathFromDB(key)
		if pp.Top()[0] == '#' { // Internal use only, skip
			continue
		}

		// skip processed ones, "labels" is special & we know we did it above
		if usedProps[key] || pp.Top() == "labels" {
			continue
		}
		/*
			k, _, _ := strings.Cut(key, ".")
			if usedProps[k] {
				continue
			}
			usedProps[k] = true
		*/

		processProp(result, key, val)
		// result[k] = MaterializeProperty(k)
	}

	return result
}

/*
func (e *Entity) MaterializeProperty(name string) (any, error) {
	keys := map[string]bool{}

	for key, _ := range e.Props {
		if !strings.HasPrefix(key, name+string(DB_IN)) && key != name {
			continue
		}
	}

	pp, err := PropPathFromDB(name)
}
*/

func processProp(daMap map[string]any, key string, val any) {
	pp, err := PropPathFromDB(key)
	PanicIf(err != nil, fmt.Sprint(err))

	name := pp.Top()
	index := pp.IsIndexed()

	if index < 0 { // Not indexed
		daMap[name] = processPropValue(daMap[name], pp.Next(), val)
		return
	}

	currentVal := daMap[name]
	if currentVal == nil {
		currentVal = []any{}
	}

	daArray := currentVal.([]any)
	if diff := (1 + index - len(daArray)); diff > 0 { // Resize if needed
		daArray = append(daArray, make([]any, diff)...)
	}
	pp = pp.Next().Next() // Skip current and index
	daArray[index] = processPropValue(daArray[index], pp, val)
	daMap[name] = currentVal
}

func processPropValue(currentVal any, pp *PropPath, val any) any {
	if pp.Len() == 0 {
		return val
	}

	name := pp.Top()
	index := pp.IsIndexed()

	if index >= 0 {
		var daArray []any
		if currentVal == nil {
			daArray = make([]any, index+1)
		} else {
			daArray = currentVal.([]any)
		}
		pp = pp.Next().Next()
		daArray[index] = processPropValue(daArray[index], pp, val)
		return daArray
	}

	// Is scalar
	var daMap map[string]any
	if currentVal == nil {
		daMap = map[string]any{}
	} else {
		daMap = currentVal.(map[string]any)
	}
	pp = pp.Next()
	daMap[name] = processPropValue(daMap[name], pp, val)
	return daMap
}
