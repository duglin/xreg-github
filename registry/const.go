package registry

import "os"

var TESTING = (os.Getenv("TESTING") != "")

const SPECVERSION = "0.5"
const XREGSCHEMA = "xRegistry-json"

// Model attribute default values
const STRICT = true
const MAXVERSIONS = 0
const SETVERSIONID = true
const SETSTICKYDEFAULT = true
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
