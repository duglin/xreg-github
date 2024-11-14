package registry

import "os"

var TESTING = (os.Getenv("TESTING") != "")
var GitCommit = "<n/a>"

const SPECVERSION = "0.5"
const XREGSCHEMA = "xRegistry-json"

// Model attribute default values
const STRICT = true
const MAXVERSIONS = 0
const SETVERSIONID = true
const SETDEFAULTSTICKY = true
const HASDOCUMENT = true
const READONLY = false

// Attribute types
const ANY = "any"
const ARRAY = "array"
const BOOLEAN = "boolean"
const DECIMAL = "decimal"
const INTEGER = "integer"
const MAP = "map"
const OBJECT = "object"
const STRING = "string"
const TIMESTAMP = "timestamp"
const UINTEGER = "uinteger"
const URI = "uri"
const URI_REFERENCE = "urireference"
const URI_TEMPLATE = "uritemplate"
const URL = "url"

const IN_CHAR = '.'
const IN_STR = string(IN_CHAR)

// Entity "add" options
type AddType int

const (
	ADD_ADD AddType = iota + 1
	ADD_UPDATE
	ADD_UPSERT
	ADD_PATCH // includes UPSERT
)

const (
	// NEVER ADD TO THE MIDDLE - ALWAYS TO THE END
	ENTITY_REGISTRY = iota
	ENTITY_GROUP
	ENTITY_RESOURCE
	ENTITY_META
	ENTITY_VERSION
	ENTITY_MODEL
)
