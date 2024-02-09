package registry

import (
	"testing"
)

func TestModelVerifySimple(t *testing.T) {
	type Test struct {
		name  string
		model Model
		err   string
	}

	tests := []Test{
		{"empty model", Model{}, ""},
		{"empty model - 2", Model{
			Schemas:    []string{},
			Attributes: map[string]*Attribute{},
			Groups:     map[string]*GroupModel{},
		}, ""},

		{"empty schemas", Model{
			Schemas: []string{},
		}, ""},
		{"json schema", Model{
			Schemas: []string{"jsonSchema"},
		}, ""},
		{"mulitple schemas", Model{
			Schemas: []string{"jsonSchema", "jsonSchema/v1"},
		}, ""},
		{"schema + empty reg attrs", Model{
			Schemas:    []string{"xxx"},
			Attributes: Attributes{},
		}, ""},

		{"reg 1 attr - full", Model{
			Attributes: Attributes{
				"myint": &Attribute{
					Name:           "myint",
					Type:           "integer",
					Description:    "cool int",
					Enum:           []any{},
					Strict:         true,
					ReadOnly:       true,
					ClientRequired: true,
					ServerRequired: true,

					IfValue: map[string]*IfValue{},
				},
			},
		}, ""},
	}

	for _, test := range tests {
		err := test.model.Verify()
		if test.err == "" && err != nil {
			t.Fatalf("ModelVerify: %s - should have worked, got: %s",
				test.name, err)
		}
		if test.err != "" && err == nil {
			t.Fatalf("ModelVerify: %s - should have failed with: %s",
				test.name, test.err)
		}
		if err != nil && test.err != err.Error() {
			t.Fatalf("ModifyVerify: %s - expected %q got %q", test.name,
				test.err, err.Error())
		}
	}
}

func TestModelVerifyRegAttr(t *testing.T) {
	type Test struct {
		name  string
		model Model
		err   string
	}

	tests := []Test{
		{"empty attrs", Model{Attributes: Attributes{}}, ""},
		{"err - missing name", Model{
			Attributes: Attributes{"myint": {}},
		}, `Attribute 'myint's 'name' property must be 'myint' not ''`},
		{
			"err - wrong name", Model{
				Attributes: Attributes{"myint": {Name: "bad"}},
			}, `Attribute 'myint's 'name' property must be 'myint' not 'bad'`},
		{"err - missing type", Model{
			Attributes: Attributes{"myint": {Name: "myint"}},
		}, `Attribute 'myint' has an invalid type: ''`},
		// Test all valid types
		{"type - boolean", Model{
			Attributes: Attributes{"x": {Name: "x", Type: BOOLEAN}}}, ``},
		{"type - decimal", Model{
			Attributes: Attributes{"x": {Name: "x", Type: DECIMAL}}}, ``},
		{"type - integer", Model{
			Attributes: Attributes{"x": {Name: "x", Type: INTEGER}}}, ``},
		{"type - string", Model{
			Attributes: Attributes{"x": {Name: "x", Type: STRING}}}, ``},
		{"type - timestamp", Model{
			Attributes: Attributes{"x": {Name: "x", Type: TIMESTAMP}}}, ``},
		{"type - uinteger", Model{
			Attributes: Attributes{"x": {Name: "x", Type: UINTEGER}}}, ``},
		{"type - uri", Model{
			Attributes: Attributes{"x": {Name: "x", Type: URI}}}, ``},
		{"type - urireference", Model{
			Attributes: Attributes{"x": {Name: "x", Type: URI_REFERENCE}}}, ``},
		{"type - uritempalte", Model{
			Attributes: Attributes{"x": {Name: "x", Type: URI_TEMPLATE}}}, ``},
		{"type - url", Model{
			Attributes: Attributes{"x": {Name: "x", Type: URL}}}, ``},
		{"type - any", Model{
			Attributes: Attributes{"x": {Name: "x", Type: ANY}}}, ``},
		{"type - any", Model{
			Attributes: Attributes{"*": {Name: "*", Type: ANY}}}, ``},

		{"type - array", Model{
			Attributes: Attributes{"x": {Name: "x", Type: ARRAY,
				Item: &Item{}}}}, ``},
		{"type - map", Model{
			Attributes: Attributes{"x": {Name: "x", Type: MAP,
				Item: &Item{}}}}, ``},
		{"type - object", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT,
				Item: &Item{}}}}, ``},

		{"type - any", Model{
			Attributes: Attributes{".foo": {Name: ".foo", Type: ANY}}},
			`Invalid Attribute name/key '.foo' - must match '^[a-z_][a-z0-9_./]{0,62}$'`},
		{"type - array - missing item", Model{
			Attributes: Attributes{"x": {Name: "x", Type: ARRAY}}}, ""},
		{"type - map - missing item", Model{
			Attributes: Attributes{"x": {Name: "x", Type: MAP}}}, ""},
		{"type - object - missing item", Model{ // odd but allowable
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT}}}, ""},

		{"type - bad urlx", Model{
			Attributes: Attributes{"x": {Name: "x", Type: "urlx"}}},
			`Attribute 'x' has an invalid type: 'urlx'`},

		{"type - bad required", Model{
			Attributes: Attributes{"x": {Name: "x", Type: "url",
				ClientRequired: true}}},
			`Attribute 'x' is 'clientrequired' so 'serverrequired' must be 'true' as well`},

		// Now some Item stuff
		{"Item - missing", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT}}}, ""},
		{"Item - empty - ", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT,
				Item: &Item{}}}}, ""},
		{"Item - Type any - ", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT,
				Item: &Item{Type: ANY}}}},
			`Attribute 'x' must not have an item.type of: object`},
		{"Item - Type scalar + item  - ", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT,
				Item: &Item{Type: BOOLEAN, Item: &Item{}}}}},
			`Attribute 'x' must not have an item.type of: object`},
		{"Item - Type any + item  - ", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT,
				Item: &Item{Type: ANY, Item: &Item{}}}}},
			`Attribute 'x' must not have an item.type of: object`},
	}

	for _, test := range tests {
		err := test.model.Verify()
		if test.err == "" && err != nil {
			t.Fatalf("ModelVerify: %s - should have worked, got: %s",
				test.name, err)
		}
		if test.err != "" && err == nil {
			t.Fatalf("ModelVerify: %s - should have failed with: %s",
				test.name, test.err)
		}
		if err != nil && test.err != err.Error() {
			t.Fatalf("ModifyVerify: %s - expected %q got %q", test.name,
				test.err, err.Error())
		}
	}
}

func TestModelVerifyEnum(t *testing.T) {
	type Test struct {
		name  string
		model Model
		err   string
	}

	tests := []Test{
		{"empty enum - int", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{}}}}, ""},
		{"empty enum - obj", Model{Attributes: Attributes{
			"x": {Name: "x", Type: OBJECT, Enum: []any{}}}},
			"Attribute 'x' must be a scalar, not: object"},
		{"empty enum - array", Model{Attributes: Attributes{
			"x": {Name: "x", Type: ARRAY, Enum: []any{}}}},
			"Attribute 'x' must be a scalar, not: array"},
		{"empty enum - map", Model{Attributes: Attributes{
			"x": {Name: "x", Type: MAP, Enum: []any{}}}},
			"Attribute 'x' must be a scalar, not: map"},
		{"empty enum - any", Model{Attributes: Attributes{
			"x": {Name: "x", Type: ANY, Enum: []any{}}}},
			"Attribute 'x' must be a scalar, not: any"},

		{"enum - bool - true ", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{true}}}}, ""},
		{"enum - bool 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{true, false}}}}, ""},
		{"enum - bool string", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{true, ""}}}},
			"Attribute 'x' enum value of '' must be of type: boolean"},
		{"enum - bool float", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{5.5}}}},
			"Attribute 'x' enum value of '5.5' must be of type: boolean"},
		{"enum - bool map", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{map[string]string{}}}}},
			"Attribute 'x' enum value of 'map[]' must be of type: boolean"},

		{"enum - decimal 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: DECIMAL, Enum: []any{5.5}}}}, ""},
		{"enum - decimal 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: DECIMAL, Enum: []any{5.5, 2}}}}, ""},
		{"enum - decimal bool", Model{Attributes: Attributes{
			"x": {Name: "x", Type: DECIMAL, Enum: []any{true, 5}}}},
			"Attribute 'x' enum value of 'true' must be of type: decimal"},

		{"enum - integer 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{1}}}}, ""},
		{"enum - integer 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{-1, 1}}}}, ""},
		{"enum - integer float", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{-1, 1, 3.1}}}},
			"Attribute 'x' enum value of '3.1' must be of type: integer"},
		{"enum - integer float", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{[]int{}}}}},
			"Attribute 'x' enum value of '[]' must be of type: integer"},

		{"enum - string 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: STRING, Enum: []any{"a"}}}}, ""},
		{"enum - string 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: STRING, Enum: []any{"a", ""}}}}, ""},
		{"enum - string int", Model{Attributes: Attributes{
			"x": {Name: "x", Type: STRING, Enum: []any{"a", 0}}}},
			"Attribute 'x' enum value of '0' must be of type: string"},
		{"enum - string struct", Model{Attributes: Attributes{
			"x": {Name: "x", Type: STRING, Enum: []any{"a", struct{}{}}}}},
			"Attribute 'x' enum value of '{}' must be of type: string"},

		{"enum - timestamp 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: TIMESTAMP,
				Enum: []any{"2024-01-02T12:01:02Z"}}}}, ""},
		{"enum - timestamp 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: TIMESTAMP,
				Enum: []any{"2024-01-02T12:01:02Z", "2000-12-31T01:02:03Z"}}}},
			""},
		{"enum - timestamp bad", Model{Attributes: Attributes{
			"x": {Name: "x", Type: TIMESTAMP,
				Enum: []any{"2024-01-02T12:01:02Z", "bad"}}}},
			"Attribute 'x' enum value of 'bad' must be of type: timestamp"},
		{"enum - timestamp type", Model{Attributes: Attributes{
			"x": {Name: "x", Type: TIMESTAMP,
				Enum: []any{"2024-01-02T12:01:02Z", 5.5}}}},
			"Attribute 'x' enum value of '5.5' must be of type: timestamp"},

		{"enum - uint 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: UINTEGER, Enum: []any{1}}}}, ""},
		{"enum - uint 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: UINTEGER, Enum: []any{1, 2}}}},
			""},
		{"enum - uint bad", Model{Attributes: Attributes{
			"x": {Name: "x", Type: UINTEGER, Enum: []any{2, -1}}}},
			"Attribute 'x' enum value of '-1' must be of type: uinteger"},
		{"enum - uint type", Model{Attributes: Attributes{
			"x": {Name: "x", Type: UINTEGER,
				Enum: []any{5.5}}}},
			"Attribute 'x' enum value of '5.5' must be of type: uinteger"},

		{"empty enum - int", Model{Attributes: Attributes{
			"x": {Name: "x", Type: URI, Enum: []any{"..."}}}}, ""},
		{"empty enum - int", Model{Attributes: Attributes{
			"x": {Name: "x", Type: URI_REFERENCE, Enum: []any{"..."}}}}, ""},
		{"empty enum - int", Model{Attributes: Attributes{
			"x": {Name: "x", Type: URI_TEMPLATE, Enum: []any{"..."}}}}, ""},
		{"empty enum - int", Model{Attributes: Attributes{
			"x": {Name: "x", Type: URL, Enum: []any{"..."}}}}, ""},
	}

	for _, test := range tests {
		err := test.model.Verify()
		if test.err == "" && err != nil {
			t.Fatalf("ModelVerify: %s - should have worked, got: %s",
				test.name, err)
		}
		if test.err != "" && err == nil {
			t.Fatalf("ModelVerify: %s - should have failed with: %s",
				test.name, test.err)
		}
		if err != nil && test.err != err.Error() {
			t.Fatalf("ModifyVerify: %s - expected %q got %q", test.name,
				test.err, err.Error())
		}
	}
}
