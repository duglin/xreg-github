package registry

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

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

func ToGoType(s string) reflect.Type {
	switch s {
	case ANY:
		return reflect.TypeOf(any(true))
	case BOOLEAN:
		return reflect.TypeOf(true)
	case DECIMAL:
		return reflect.TypeOf(float64(1.1))
	case INTEGER:
		return reflect.TypeOf(int(1))
	case STRING, TIME, URI, URI_REFERENCE, URI_TEMPLATE, URL:
		return reflect.TypeOf("")
	case UINTEGER:
		return reflect.TypeOf(uint(0))
	}
	panic("ToGoType - not supported: " + s)
}

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

	// Erase all old props first
	e.Props = map[string]any{}

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		name := NotNilString(row[0])
		val := NotNilString(row[1])
		propType := NotNilString(row[2])

		e.SetPropFromString(name, &val, propType)
	}
	return nil
}

// Maybe replace error with a panic?
func (e *Entity) SetFromDB(name string, val any) error {
	pp, err := PropPathFromDB(name)
	if err != nil {
		return err
	}
	return e.SetPP(pp, val)
}

func (e *Entity) SetFromUI(name string, val any) error {
	log.VPrintf(3, ">Enter: SetFromUI(%s=%v)", name, val)
	defer log.VPrintf(3, "<Exit SetFromUI")
	pp, err := PropPathFromUI(name)
	if err != nil {
		return err
	}
	return e.SetPP(pp, val)
}

func (e *Entity) SetPP(pp *PropPath, val any) error {
	log.VPrintf(3, ">Enter: SetPP(%s=%v)", pp.UI(), val)
	defer log.VPrintf(3, "<Exit SetPP")

	name := pp.DB()

	if pp.Top() == "labels" {
		if pp.Len() == 1 {
			return fmt.Errorf("Invalid property name: %s", pp.Top())
		}
		mapName := pp.Top()
		key := pp.Next().Top()
		if len(key) == 0 {
			return fmt.Errorf("Map %q key is empty", mapName)
		}
	}

	if e.DbSID == "" {
		log.Fatalf("DbSID should not be empty")
	}
	if e.RegistrySID == "" {
		log.Fatalf("RegistrySID should not be empty")
	}

	// Make sure the attribute is defined in the model and has valid chars
	attrType, err := GetAttributeType(e.RegistrySID, e.Abstract, pp)
	if err != nil {
		// log.Printf("Error on getAttr(%s): %s", pp.UI(), err)
		return err
	}
	if attrType == "" {
		return fmt.Errorf("Can't find attribute %q", pp.UI())
	}

	if !IsNil(val) {
		if err = ValidatePropValue(val, attrType); err != nil {
			return err
		}
	}

	// #resource is special and is saved in it's own table
	if pp.Len() == 1 && pp.Top() == "#resource" {
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
		propType := attrType
		if attrType == ANY {
			propType = GoToOurType(val)
		}

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

	if val == nil {
		delete(e.Props, name)
	} else {
		if e.Props == nil {
			e.Props = map[string]any{}
		}
		e.Props[name] = val
	}

	return nil
}

func (e *Entity) SetPropFromString(name string, val *string, propType string) {
	if val == nil {
		delete(e.Props, name)
	}
	if e.Props == nil {
		e.Props = map[string]any{}
	}

	if propType == STRING || propType == URI || propType == URI_REFERENCE ||
		propType == URI_TEMPLATE || propType == URL || propType == TIME {
		e.Props[name] = *val
	} else if propType == BOOLEAN {
		// Technically "1" check shouldn't be needed, but just in case
		e.Props[name] = (*val == "1") || (*val == "true")
	} else if propType == INTEGER || propType == UINTEGER {
		tmpInt, err := strconv.Atoi(*val)
		if err != nil {
			panic(fmt.Sprintf("error parsing int: %s", *val))
		}
		e.Props[name] = tmpInt
	} else if propType == DECIMAL {
		tmpFloat, err := strconv.ParseFloat(*val, 64)
		if err != nil {
			panic(fmt.Sprintf("error parsing float: %s", *val))
		}
		e.Props[name] = tmpFloat
	} else if propType == MAP {
		if *val != "" {
			panic(fmt.Sprintf("MAP value should be empty string"))
		}
		e.Props[name] = map[string]any{}
	} else if propType == ARRAY {
		if *val != "" {
			panic(fmt.Sprintf("MAP value should be empty string"))
		}
		e.Props[name] = []any{}
	} else if propType == OBJECT {
		if *val != "" {
			panic(fmt.Sprintf("MAP value should be empty string"))
		}
		e.Props[name] = map[string]any{}
	} else {
		panic(fmt.Sprintf("bad type(%s): %v", propType, name))
	}
}

func ValidatePropValue(val any, attrType string) error {
	vKind := reflect.ValueOf(val).Kind()

	switch attrType {
	case ANY:
		return nil
	case BOOLEAN:
		if vKind != reflect.Bool {
			return fmt.Errorf(`"%v" should be a boolean`, val)
		}
	case DECIMAL:
		if vKind != reflect.Int && vKind != reflect.Float64 {
			return fmt.Errorf(`"%v" should be a decimal`, val)
		}
	case INTEGER:
		if vKind != reflect.Int {
			return fmt.Errorf(`"%v" should be an integer`, val)
		}
	case UINTEGER:
		if vKind != reflect.Int {
			return fmt.Errorf(`"%v" should be an integer`, val)
		}
		i := val.(int)
		if i < 0 {
			return fmt.Errorf(`"%v" should be an uinteger`, val)
		}
	case STRING, URI, URI_REFERENCE, URI_TEMPLATE, URL: // cheat
		if vKind != reflect.String {
			return fmt.Errorf(`"%v" should be a string`, val)
		}
	case TIME:
		if vKind != reflect.String {
			return fmt.Errorf(`"%v" should be a timestamp`, val)
		}
		str := val.(string)
		_, err := time.Parse(time.RFC3339, str)
		if err != nil {
			return fmt.Errorf("Malformed timestamp %q: %s", str, err)
		}

	// For the non-scalar types, these should only be used when someone
	// passing in something like:
	//    "foo": {}
	// and we need to save an empty (non-scalar) value. Hence the "if" below.
	case MAP:
		// anything but an empty map means we did something wrong before this
		v := reflect.ValueOf(val)
		if v.Kind() != reflect.Map || v.Len() > 0 {
			return fmt.Errorf("Value must be an empty map")
		}
		val = ""

	case ARRAY:
		// anything but an empty map means we did something wrong before this
		v := reflect.ValueOf(val)
		if v.Kind() != reflect.Slice || v.Len() > 0 {
			return fmt.Errorf("Value must be an empty slice")
		}
		val = ""

	case OBJECT:
		// anything but an empty map means we did something wrong before this
		v := reflect.ValueOf(val)
		if v.Kind() != reflect.Struct || v.NumField() > 0 {
			return fmt.Errorf("Value must be an empty struct")
		}
		val = ""

	default:
		ShowStack()
		log.Printf("AttrType: %q  Val: %#q", attrType, val)
		return fmt.Errorf("Unsupported type: %s", attrType)
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

		entity.SetPropFromString(propName, &propVal, propType)
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
	{"specversion", STRING, "0", false, nil, &Attribute{
		Name:     "specversion",
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
	{"epoch", UINTEGER, "", false, nil, &Attribute{
		Name:     "epoch",
		Type:     UINTEGER,
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
	{"latestversionid", STRING, "2", false, nil, &Attribute{
		Name:     "latestversionid",
		Type:     STRING,
		Required: true,
	}},
	{"latestversionurl", URL, "2", false, func(e *Entity, info *RequestInfo) any {
		val := e.Props[NewPPP("latestversionid").DB()]
		if IsNil(val) {
			return nil
		}
		return info.BaseURL + "/" + e.Path + "/versions/" + val.(string)
	}, &Attribute{
		Name:     "latestversionurl",
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
		Name: "labels",
		Type: MAP,
		Item: &Item{
			Type: STRING,
		},
	}},
	{"format", STRING, "123", true, nil, &Attribute{
		Name: "format",
		Type: STRING,
	}},
	{"createdby", STRING, "", false, nil, &Attribute{
		Name: "createdby",
		Type: STRING,
	}},
	{"createdon", TIME, "", false, nil, &Attribute{
		Name: "createdon",
		Type: TIME,
	}},
	{"modifiedby", STRING, "", false, nil, &Attribute{
		Name: "modifiedby",
		Type: STRING,
	}},
	{"modifiedon", TIME, "", false, nil, &Attribute{
		Name: "modifiedon",
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

	// Do spec defined props first, in order
	for _, prop := range OrderedSpecProps {
		if val, ok := daObj[prop.name]; ok {
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
		if key[0] == '#' || usedProps[key] { // Skip internal and "done" ones
			continue
		}

		pp, err := PropPathFromDB(key)
		PanicIf(err != nil, "Error DBparsing %q: %s", key, err)

		propName := pp.Top()

		// "labels" is special & we know we did it above
		if propName == "labels" {
			continue
		}
		// usedProps[k] = true

		current := result[propName] // needed for non-scalars
		result[propName], err = MaterializeProp(current, pp.Next(), val)
		PanicIf(err != nil, "MaterializeProp: %s", err)
	}

	return result
}

func MaterializeProp(current any, pp *PropPath, val any) (any, error) {
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
				return nil, fmt.Errorf("Current isn't an array: %T", current)
			}
		}

		// Resize if needed
		if diff := (1 + index - len(daArray)); diff > 0 {
			daArray = append(daArray, make([]any, diff)...)
		}

		daArray[index], err = MaterializeProp(daArray[index], pp.Next(), val)
		return daArray, err
	}

	// Is a map/object
	// TODO look for cases where Kind(val) == obj/map too - maybe?
	daMap := map[string]any{}

	if current != nil {
		daMap, ok = current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("Current isn't a map: %T", current)
		}
	}
	daMap[pp.Top()], err = MaterializeProp(daMap[pp.Top()], pp.Next(), val)
	return daMap, err
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
