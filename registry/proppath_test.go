package registry

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseUI(t *testing.T) {
	type Test struct {
		In  string
		Exp string
	}

	tests := []Test{
		{"", `{null}`},
		{"prop", `{[{"prop",-1}]}`},
		{"#prop", `{[{"#prop",-1}]}`},
		{"_", `{[{"_",-1}]}`},
		{"_1", `{[{"_1",-1}]}`},

		{"['prop']", `{[{"prop",-1}]}`},
		{"['prop1.prop2']", `{[{"prop1.prop2",-1}]}`},
		{"['prop1-prop2']", `{[{"prop1-prop2",-1}]}`},
		{"['1']", `{[{"1",-1}]}`},

		{"a1.a2", `{[{"a1",-1},{"a2",-1}]}`},
		{"a1.a2.a3", `{[{"a1",-1},{"a2",-1},{"a3",-1}]}`},
		{"a1.2", `{[{"a1",-1},{"2",-1}]}`},

		{"a1['a2']", `{[{"a1",-1},{"a2",-1}]}`},
		{"a1['2']", `{[{"a1",-1},{"2",-1}]}`},

		{"a1[2]", `{[{"a1",-1},{"2",2}]}`},
		{"a1['a2'].a3", `{[{"a1",-1},{"a2",-1},{"a3",-1}]}`},
		{"a1['a2'].3", `{[{"a1",-1},{"a2",-1},{"3",-1}]}`},
		{"a1['a2']['a3']", `{[{"a1",-1},{"a2",-1},{"a3",-1}]}`},
		{"a1[2].a3", `{[{"a1",-1},{"2",2},{"a3",-1}]}`},
		{"a1[2]['a3']", `{[{"a1",-1},{"2",2},{"a3",-1}]}`},
		{"a1[2]['a3']", `{[{"a1",-1},{"2",2},{"a3",-1}]}`},
		{"a1[2][3]", `{[{"a1",-1},{"2",2},{"3",3}]}`},

		// Errors
		{".prop", `Unexpected . in ".prop" at pos 1`},
		{"1", `Unexpected 1 in "1" at pos 1`},
		{"_#1", `Unexpected # in "_#1" at pos 2`},
		{"a1..a2", `Unexpected . in "a1..a2" at pos 4`},
		{"a1.[a]", `Unexpected [ in "a1.[a]" at pos 4`},
		{"[2]", `Unexpected 2 in "[2]" at pos 2`},
		{"[prop1.prop2]", `Expecting a ' at pos 2 in "[prop1.prop2]"`},

		{"a1.", `Unexpected end of property in "a1."`},
		{"a1['a", `Unexpected end of property in "a1['a"`},
		{"a1['a'", `Unexpected end of property in "a1['a'"`},
		{"a1[1", `Unexpected end of property in "a1[1"`},

		{"a1[]", `Unexpected ] in "a1[]" at pos 4`},
		{"a1['']", `Unexpected ' in "a1['']" at pos 5`},
		{"a1[']", `Unexpected ] in "a1[']" at pos 5`},
	}

	for _, test := range tests {
		pp, err := PropPathFromUI(test.In)
		res := ""
		if pp != nil {
			tmp, _ := json.Marshal(pp)
			res = string(tmp)
			res = strings.ReplaceAll(res, `"Parts":`, "")
			res = strings.ReplaceAll(res, `"Text":`, "")
			res = strings.ReplaceAll(res, `"Index":`, "")
		} else {
			res = err.Error()
		}
		if res != test.Exp {
			t.Logf("Test: %q:\nExp: %s\nGot: %s\n", test.In, test.Exp, res)
			t.Fail()
		}
	}
}

func TestAbstract(t *testing.T) {
	type Test struct {
		P   *PropPath
		Exp string
	}

	tests := []Test{
		{NewPPP("one"), "one"},
		{NewPPP("one").P("two"), "one,two"},
		{NewPPP("one").I(2), "one#2"},
		{NewPPP("one").I(2).P("two"), "one#2,two"},
	}

	for _, test := range tests {
		res := test.P.Abstract()
		if res != test.Exp {
			t.Fatalf("Test: %s\nExp: %s\nGot: %s", test.P.UI(), test.Exp, res)
		}
	}
}
