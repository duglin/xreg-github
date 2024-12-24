package registry

import (
	"reflect"
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
			Attributes: map[string]*Attribute{},
			Groups:     map[string]*GroupModel{},
		}, ""},

		{"reg 1 attr - full", Model{
			Attributes: Attributes{
				"myint": &Attribute{
					Name:           "myint",
					Type:           "integer",
					Description:    "cool int",
					Enum:           []any{1},
					Strict:         PtrBool(true),
					ReadOnly:       true,
					ClientRequired: true,
					ServerRequired: true,

					IfValues: IfValues{},
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
		}, `"model.myint" must have a "name" set to "myint"`},
		{
			"err - wrong name", Model{
				Attributes: Attributes{"myint": {Name: "bad"}},
			}, `"model.myint" must have a "name" set to "myint"`},
		{"err - missing type", Model{
			Attributes: Attributes{"myint": {Name: "myint"}},
		}, `"model.myint" is missing a "type"`},
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
				Item: &Item{Type: INTEGER}}}}, ``},
		{"type - map", Model{
			Attributes: Attributes{"x": {Name: "x", Type: MAP,
				Item: &Item{Type: STRING}}}}, ``},
		{"type - object - 1", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT}}}, ``},
		{"type - object - 2", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT,
				Attributes: Attributes{}}}}, ``},

		{"type - any", Model{
			Attributes: Attributes{".foo": {Name: ".foo", Type: ANY}}},
			`"model" has an invalid attribute key ".foo" - must match "^[a-z_][a-z0-9_./]{0,62}$"`},
		{"type - array - missing item", Model{
			Attributes: Attributes{"x": {Name: "x", Type: ARRAY}}},
			`"model.x" must have an "item" section`},
		{"type - map - missing item", Model{
			Attributes: Attributes{"x": {Name: "x", Type: MAP}}},
			`"model.x" must have an "item" section`},
		{"type - object - missing item", Model{ // odd but allowable
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT}}}, ""},

		{"type - bad urlx", Model{
			Attributes: Attributes{"x": {Name: "x", Type: "urlx"}}},
			`"model.x" has an invalid type: urlx`},

		{"type - bad required", Model{
			Attributes: Attributes{"x": {Name: "x", Type: "url",
				ClientRequired: true}}},
			`"model.x" must have "serverrequired" since "clientrequired" is "true"`},

		{"type - missing server required", Model{
			Attributes: Attributes{"x": {Name: "x", Type: "url",
				Default: "xxx"}}},
			`"model.x" must have "serverrequired" since a "default" value is provided`},

		// Now some Item stuff
		{"Item - missing", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT}}}, ""},
		{"Item - empty - ", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT,
				Item: &Item{}}}},
			`"model.x" must not have an "item" section`},

		// Nested stuff
		{"Nested - map - obj", Model{
			Attributes: Attributes{"m": {Name: "m", Type: MAP,
				Item: &Item{Type: OBJECT}}}},
			``},
		{"Nested - map - obj - missing item - valid", Model{
			Attributes: Attributes{"m": {Name: "m", Type: MAP,
				Item: &Item{Type: OBJECT, Attributes: Attributes{}}}}},
			``},
		{"Nested - map - map - misplaced attrs", Model{
			Attributes: Attributes{"m": {Name: "m", Type: MAP,
				Item: &Item{Type: MAP, Attributes: Attributes{}}}}},
			`"model.m.item" must not have "attributes"`},
		{"Nested - map - array - misplaced attrs", Model{
			Attributes: Attributes{"m": {Name: "m", Type: MAP,
				Item: &Item{Type: ARRAY, Attributes: Attributes{}}}}},
			`"model.m.item" must not have "attributes"`},

		{"Nested - map - obj + attr", Model{
			Attributes: Attributes{"m": {Name: "m", Type: MAP,
				Item: &Item{Type: OBJECT, Attributes: Attributes{
					"i": {Name: "i", Type: INTEGER}}}}}},
			``},
		{"Nested - map - obj + obj + attr", Model{
			Attributes: Attributes{"m": {Name: "m", Type: MAP,
				Item: &Item{Type: OBJECT, Attributes: Attributes{
					"i": {Name: "i", Type: OBJECT,
						Attributes: Attributes{"s": {Name: "s",
							Type: STRING}}}}}}}},
			``},
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
			t.Fatalf("ModifyVerify: %s:\nExp: %s\nGot: %s", test.name,
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
			"x": {Name: "x", Type: INTEGER, Enum: []any{1}}}}, ""},
		{"empty enum - obj", Model{Attributes: Attributes{
			"x": {Name: "x", Type: OBJECT, Enum: []any{1}}}},
			`"model.x" is not a scalar, so "enum" is not allowed`},
		{"empty enum - array", Model{Attributes: Attributes{
			"x": {Name: "x", Type: ARRAY, Enum: []any{1}}}},
			`"model.x" is not a scalar, so "enum" is not allowed`},
		{"empty enum - map", Model{Attributes: Attributes{
			"x": {Name: "x", Type: MAP, Enum: []any{1}}}},
			`"model.x" is not a scalar, so "enum" is not allowed`},
		{"empty enum - any", Model{Attributes: Attributes{
			"x": {Name: "x", Type: ANY, Enum: []any{}}}},
			`"model.x" specifies an "enum" but it is empty`},

		{"enum - bool - true ", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{true}}}}, ""},
		{"enum - bool 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{true, false}}}}, ""},
		{"enum - bool string", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{true, ""}}}},
			`"model.x" enum value "" must be of type "boolean"`},
		{"enum - bool float", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{5.5}}}},
			`"model.x" enum value "5.5" must be of type "boolean"`},
		{"enum - bool map", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{map[string]string{}}}}},
			`"model.x" enum value "map[]" must be of type "boolean"`},

		{"enum - decimal 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: DECIMAL, Enum: []any{5.5}}}}, ""},
		{"enum - decimal 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: DECIMAL, Enum: []any{5.5, 2}}}}, ""},
		{"enum - decimal bool", Model{Attributes: Attributes{
			"x": {Name: "x", Type: DECIMAL, Enum: []any{true, 5}}}},
			`"model.x" enum value "true" must be of type "decimal"`},

		{"enum - integer 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{1}}}}, ""},
		{"enum - integer 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{-1, 1}}}}, ""},
		{"enum - integer float", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{-1, 1, 3.1}}}},
			`"model.x" enum value "3.1" must be of type "integer"`},
		{"enum - integer float", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{[]int{}}}}},
			`"model.x" enum value "[]" must be of type "integer"`},

		{"enum - string 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: STRING, Enum: []any{"a"}}}}, ""},
		{"enum - string 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: STRING, Enum: []any{"a", ""}}}}, ""},
		{"enum - string int", Model{Attributes: Attributes{
			"x": {Name: "x", Type: STRING, Enum: []any{"a", 0}}}},
			`"model.x" enum value "0" must be of type "string"`},
		{"enum - string struct", Model{Attributes: Attributes{
			"x": {Name: "x", Type: STRING, Enum: []any{"a", struct{}{}}}}},
			`"model.x" enum value "{}" must be of type "string"`},

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
			`"model.x" enum value "bad" must be of type "timestamp"`},
		{"enum - timestamp type", Model{Attributes: Attributes{
			"x": {Name: "x", Type: TIMESTAMP,
				Enum: []any{"2024-01-02T12:01:02Z", 5.5}}}},
			`"model.x" enum value "5.5" must be of type "timestamp"`},

		{"enum - uint 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: UINTEGER, Enum: []any{1}}}}, ""},
		{"enum - uint 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: UINTEGER, Enum: []any{1, 2}}}},
			""},
		{"enum - uint bad", Model{Attributes: Attributes{
			"x": {Name: "x", Type: UINTEGER, Enum: []any{2, -1}}}},
			`"model.x" enum value "-1" must be of type "uinteger"`},
		{"enum - uint type", Model{Attributes: Attributes{
			"x": {Name: "x", Type: UINTEGER,
				Enum: []any{5.5}}}},
			`"model.x" enum value "5.5" must be of type "uinteger"`},

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
			t.Fatalf("ModifyVerify: %s\nExp: %s\nGot: %s", test.name,
				test.err, err.Error())
		}
	}
}

func TestGetModelSerializer(t *testing.T) {
	type Match struct {
		format string
		result ModelSerializer
	}
	type Format struct {
		format string
		fn     ModelSerializer
	}

	fn1 := func(m *Model, format string) ([]byte, error) { return nil, nil }
	fn2 := func(m *Model, format string) ([]byte, error) { return nil, nil }
	fn3 := func(m *Model, format string) ([]byte, error) { return nil, nil }
	fn4 := func(m *Model, format string) ([]byte, error) { return nil, nil }

	type Test struct {
		formats []Format
		matches []Match
	}

	tests := []Test{
		{
			formats: []Format{},
			matches: []Match{
				{format: "foo", result: nil},
			},
		},
		{
			formats: []Format{
				{format: "f1", fn: fn1},
			},
			matches: []Match{
				{format: "f1", result: fn1},
				{format: "f1/v1", result: nil},
				{format: "f2", result: nil},
				{format: "f2/v1", result: nil},
			},
		},
		{
			formats: []Format{
				{format: "f1", fn: fn1},
				{format: "f1/v1", fn: fn2},
				{format: "f1/v2", fn: fn3},
			},
			matches: []Match{
				{format: "f1", result: fn1},
				{format: "f1/v1", result: fn2},
				{format: "f1/v2", result: fn3},
				{format: "f1/vx", result: nil},
			},
		},
		{
			formats: []Format{
				{format: "f1/v1", fn: fn2},
				{format: "f1/v2", fn: fn3},
			},
			matches: []Match{
				{format: "f1", result: fn3},
				{format: "f1/v1", result: fn2},
				{format: "f1/v2", result: fn3},
				{format: "f1/vx", result: nil},
			},
		},
		{
			formats: []Format{
				{format: "f1/v1", fn: fn1},
				{format: "f1/v2", fn: fn2},
				{format: "f2/va", fn: fn3},
				{format: "f2/vb", fn: fn4},
			},
			matches: []Match{
				{format: "f1", result: fn2},
				{format: "f1/v1", result: fn1},
				{format: "f1/v2", result: fn2},
				{format: "f1/vx", result: nil},
				{format: "f1vx", result: nil},
				{format: "f2", result: fn3},
				{format: "f2/va", result: fn3},
				{format: "f2/vb", result: fn4},
				{format: "f2/vx", result: nil},
				{format: "f2vx", result: nil},
			},
		},
	}

	for _, test := range tests {
		// Start out clean
		ModelSerializers = map[string]ModelSerializer{}
		for _, f := range test.formats {
			RegisterModelSerializer(f.format, f.fn)
		}
		for _, m := range test.matches {
			fn := GetModelSerializer(m.format)
			if reflect.ValueOf(fn).Pointer() == reflect.ValueOf(m.result).Pointer() {
				continue
			}
			if fn == nil {
				t.Fatalf("Fn is nil for match: %q, Test:\n%s",
					m.format, ToJSON(test.formats))
			}
			if fn != nil && m.result == nil {
				t.Fatalf("Fn should have been nil for match: %q, Test:\n%s",
					m.format, ToJSON(test.formats))
			}
		}
	}
}
