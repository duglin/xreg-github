package registry

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	// log "github.com/duglin/dlog"
)

const UX_IN = '.'

// If DB_IN changes then DefaultProps in init.sql needs to change too
const DB_IN = ','
const DB_INDEX = '#'

type PropPath struct {
	Parts []PropPart
}

func NewPPP(prop string) *PropPath {
	return NewPP().P(prop)
}

func NewPP() *PropPath {
	return &PropPath{}
}

func (pp *PropPath) Len() int {
	if pp == nil {
		return 0
	}
	return len(pp.Parts)
}

func (pp *PropPath) Top() string {
	if len(pp.Parts) == 0 {
		return ""
	}
	return pp.Parts[0].Text
}

func (pp *PropPath) IsIndexed() int {
	if pp.Len() > 1 && pp.Parts[1].Index >= 0 {
		return pp.Parts[1].Index
	}
	return -1
}

func (pp *PropPath) First() *PropPath {
	if pp.Len() == 0 {
		return nil
	}
	return &PropPath{
		Parts: pp.Parts[:1],
	}
}

func (pp *PropPath) Next() *PropPath {
	if pp.Len() == 1 {
		return nil
	}
	return &PropPath{
		Parts: pp.Parts[1:],
	}
}

func PropPathFromPath(str string) (*PropPath, error) {
	str = strings.Trim(str, "/")
	parts := strings.Split(str, "/")
	res := &PropPath{}
	for _, p := range parts {
		res.Parts = append(res.Parts, PropPart{
			Text:  p,
			Index: -1,
		})
	}
	return res, nil
}

func (pp *PropPath) Path() string {
	if pp == nil {
		return ""
	}
	res := strings.Builder{}
	for _, part := range pp.Parts {
		if res.Len() > 0 {
			res.WriteRune('/')
		}
		res.WriteString(part.Text)
	}
	return res.String()
}

func (pp *PropPath) DB() string {
	if pp == nil {
		return ""
	}
	res := strings.Builder{}
	for _, part := range pp.Parts {
		if res.Len() != 0 {
			// res.WriteRune(DB_IN)
		}
		if part.Index >= 0 {
			res.WriteRune(DB_INDEX)
		}
		res.WriteString(part.Text)
		res.WriteRune(DB_IN)
	}
	return res.String()
}

func (pp *PropPath) Abstract() string {
	if pp == nil {
		return ""
	}
	res := strings.Builder{}
	for _, part := range pp.Parts {
		if part.Index >= 0 {
			res.WriteRune(DB_INDEX)
		} else if res.Len() != 0 {
			res.WriteRune(DB_IN)
		}
		res.WriteString(part.Text)
	}
	return res.String()
}

func MustPropPathFromDB(str string) *PropPath {
	pp, err := PropPathFromDB(str)
	PanicIf(err != nil, "Bad pp: %s", str)
	return pp
}

func PropPathFromDB(str string) (*PropPath, error) {
	res := &PropPath{}

	if len(str) == 0 || str[0] == '.' || str[0] == '#' {
		str = strings.TrimRight(str, string(DB_IN))
		res.Parts = append(res.Parts, PropPart{
			Text:  str,
			Index: -1,
		})
	} else {
		// Assume what's in the DB is correct, so no error checking
		parts := strings.Split(str, string(DB_IN))
		PanicIf(len(parts) == 1 && parts[0] == "", "Empty str")
		for _, p := range parts {
			if p == "" {
				continue // should only happen on trailing DB_IN
			}
			index := -1
			if p[0] == DB_INDEX {
				p = p[1:]
				var err error
				index, err = strconv.Atoi(p)
				PanicIf(err != nil, "%q isnt an int: %s", p, err)
			}
			res.Parts = append(res.Parts, PropPart{
				Text:  p,
				Index: index,
			})
		}
	}

	return res, nil
}

var stateTable = [][]string{
	// TODO: switch to a-z instead of 0-9 for state char if we need more than 10
	// nextState + ACTIONS    nextState of '/' means stop
	// a-z   0-9    -      _      .       [      ]     '     \0    else
	{"1  ", "/U ", "/U ", "2BI", "/U ", "9I ", "/U ", "/U", "/U", "/U"}, // 0-nothing
	{"2BI", "2BI", "/U ", "2BI", "/U ", "/U ", "/U ", "/U", "/U", "/U"}, // 1-strtAttr
	{"2BI", "2BI", "2BI", "2BI", "1IS", "3IS", "/U ", "/U", "/S", "/U"}, // 2-in attr
	{"/P ", "4BI", "/U ", "/U ", "/U ", "/U ", "/U ", "6I", "/U", "/U"}, // 3-start [
	{"/P ", "4BI", "/U ", "/U ", "/U ", "/U ", "5IN", "/U", "/U", "/U"}, // 4-in [
	{"/U ", "/U ", "/U ", "/U ", "1IA", "3I ", "/U ", "/U", "/ ", "/U"}, // 5-post ]
	{"7BI", "7BI", "/U ", "/U ", "/U ", "/U ", "/U ", "/U", "/U", "/U"}, // 6-start ['
	{"7BI", "7BI", "7BI", "7BI", "7BI", "/U ", "/U ", "8I", "/U", "/U"}, // 7-in ['
	{"/U ", "/U ", "/U ", "/U ", "/U ", "/U ", "5IS", "8I", "/U", "/U"}, // 8-in ['..'
	{"/Q ", "/U ", "/U ", "/U ", "/U ", "/U ", "/U ", "6I", "/U", "/U"}, // 9-str [
}

var ch2Col = map[byte]int{}

func init() {
	for ch := 'a'; ch <= 'z'; ch++ {
		ch2Col[byte(ch)] = 0
		ch2Col[byte('A'+(ch-'a'))] = 0
	}
	for ch := '0'; ch <= '9'; ch++ {
		ch2Col[byte(ch)] = 1
	}
	ch2Col['-'] = 2
	ch2Col['_'] = 3
	ch2Col['.'] = 4
	ch2Col['['] = 5
	ch2Col[']'] = 6
	ch2Col['\''] = 7
	ch2Col[0] = 8
}

func PropPathFromUI(str string) (*PropPath, error) {
	res := &PropPath{}

	if len(str) == 0 {
		return res, nil
	}

	if str[0] == '#' {
		res.Parts = append(res.Parts, PropPart{
			Text:  str,
			Index: -1,
		})
	} else {
		chIndex := 0
		ch := str[chIndex]
		buf := strings.Builder{}
		for state := 0; state != 255; { // '/' (exit) in stateTable
			col, ok := ch2Col[ch]
			if !ok {
				col = 9
			}

			actions := stateTable[state][col]
			PanicIf(actions[0] < '/' || actions[0] > '9',
				"Bad state: %xx%x", state, col)
			/*
				if str == "a1." {
					fmt.Printf("S: %d B:%q c:%c ACT:%q\n",
						state, buf.String(), ch, actions)
				}
			*/
			state = int(actions[0] - '0')
			for i := 1; i < len(actions); i++ {
				switch actions[i] {
				case ' ': // ignore
				case 'B': // buffer it
					buf.WriteRune(rune(ch))
				case 'I': // increment ch
					chIndex++
					if chIndex < len(str) {
						ch = str[chIndex]
					} else {
						ch = 0
					}
				case 'S': // end of string part
					res.Parts = append(res.Parts, PropPart{
						Text:  buf.String(),
						Index: -1,
					})
					buf.Reset()
				case 'N': // end of index(numeric) part
					tmp, err := strconv.Atoi(buf.String())
					if err != nil {
						return nil, fmt.Errorf("%q should be an integer",
							buf.String())
					}
					res.Parts = append(res.Parts, PropPart{
						Text:  buf.String(),
						Index: tmp,
					})
					buf.Reset()
				case 'P': // error case
					return nil,
						fmt.Errorf("Expecting an integer at pos %d in %q",
							chIndex+1, str)
				case 'Q': // error case
					return nil, fmt.Errorf("Expecting a ' at pos %d in %q",
						chIndex+1, str)
				case 'U': // error case
					if ch == 0 {
						return nil,
							fmt.Errorf("Unexpected end of property in %q", str)
					} else {
						return nil, fmt.Errorf("Unexpected %c in %q at pos %d",
							ch, str, chIndex+1)
					}
				}
			}
		}
	}

	return res, nil
}

func (pp *PropPath) UI() string {
	if pp == nil {
		return ""
	}
	res := strings.Builder{}
	for _, part := range pp.Parts {
		if part.Index >= 0 {
			res.WriteString(fmt.Sprintf("[%d]", part.Index))
		} else {
			if res.Len() > 0 {
				if strings.Contains(part.Text, string(UX_IN)) {
					res.WriteString("['" + part.Text + "']")
				} else {
					res.WriteString(string(UX_IN) + part.Text)
				}
			} else {
				res.WriteString(part.Text)
			}
		}
	}
	return res.String()
}

func (pp *PropPath) I(i int) *PropPath {
	return pp.Index(i)
}

func (pp *PropPath) Index(i int) *PropPath {
	newPP := NewPP()
	newPP.Parts = append(pp.Parts, PropPart{
		Text:  fmt.Sprintf("%d", i),
		Index: i,
	})
	return newPP
}

func (pp *PropPath) P(prop string) *PropPath {
	return pp.Prop(prop)
}

func (pp *PropPath) Prop(prop string) *PropPath {
	newPP := NewPP()
	newPP.Parts = append(pp.Parts, PropPart{
		Text:  prop,
		Index: -1,
	})
	return newPP
}

func (pp *PropPath) Clone() *PropPath {
	newPP := NewPP()
	newPP.Parts = append([]PropPart{}, pp.Parts...)
	return newPP
}

func (pp *PropPath) Append(addPP *PropPath) *PropPath {
	newPP := NewPP()
	newPP.Parts = append(pp.Parts, addPP.Parts...)
	return newPP
}

func (pp *PropPath) Equals(other *PropPath) bool {
	return reflect.DeepEqual(pp, other)
}

func (pp *PropPath) HasPrefix(other *PropPath) bool {
	for i, p := range other.Parts {
		if i >= pp.Len() {
			return false
		}
		if !reflect.DeepEqual(pp.Parts[i], p) {
			return false
		}
	}
	return true
}

type PropPart struct {
	Text  string
	Index int
}

func (pp *PropPart) ToInt() int {
	val, err := strconv.Atoi(pp.Text)
	PanicIf(err != nil, "Error parsing int %q: %s", pp.Text, err)
	return val
}
