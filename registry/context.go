package registry

import (
	"fmt"
	"log"
	"reflect"
	"strings"
)

type Context struct {
	Flags     *RegistryFlags
	BaseURL   string
	DataPath  string
	ModelPath string

	currentIndent string
	indent        string

	buffer     strings.Builder
	Filters    []*Filter
	MatchStack []int
}

func (c *Context) Printf(format string, args ...interface{}) {
	c.Print(fmt.Sprintf(format, args...))
}

func (c *Context) Print(str string) {
	if str[0] == '\t' {
		c.buffer.WriteString(c.currentIndent)
		str = str[1:]
	}
	c.buffer.WriteString(str)
}

func (c *Context) Result() string {
	return c.buffer.String()
}

func (c *Context) Spaces() string {
	return c.currentIndent
}

func (c *Context) Indent() string {
	c.currentIndent += c.indent
	return c.currentIndent
}

func (c *Context) Outdent() string {
	c.currentIndent = c.currentIndent[:len(c.currentIndent)-len(c.indent)]
	return c.currentIndent
}

func (c *Context) Sprintf(str string, args ...interface{}) string {
	return fmt.Sprintf(c.Spaces()+str, args...)
}

func (c *Context) BaseURLPush(word string) string {
	c.BaseURL += "/" + word
	return c.BaseURL
}

func (c *Context) DataURL() string {
	if c.Flags.AsDoc {
		return "#" + "/" + c.DataPath
	}
	return c.BaseURL + "/" + strings.ToLower(c.DataPath)
}

func (c *Context) DocifyURL(daURL string) string {
	if c.Flags.AsDoc && strings.HasPrefix(daURL, c.BaseURL) {
		return "#" + daURL[len(c.BaseURL):]
	}
	return daURL
}

func (c *Context) DataPush(word string) string {
	if c.DataPath != "" {
		c.DataPath += "/"
	}
	c.DataPath += word
	return c.DataPath
}

func (c *Context) DataPop() string {
	if c.DataPath == "" {
		panic("Popping empty DataPath")
	}
	if i := strings.LastIndex(c.DataPath, "/"); i >= 0 {
		c.DataPath = c.DataPath[:i]
	} else {
		c.DataPath = ""
	}
	return c.DataPath
}

func (c *Context) ModelPush(word string) string {
	if c.ModelPath != "" {
		c.ModelPath += "."
	}
	c.ModelPath += word
	return c.ModelPath
}

func (c *Context) ModelPop() string {
	if c.ModelPath == "" {
		panic("Popping empty ModelPath")
	}
	if i := strings.LastIndex(c.ModelPath, "."); i >= 0 {
		c.ModelPath = c.ModelPath[:i]
	} else {
		c.ModelPath = ""
	}
	return c.ModelPath
}

func (c *Context) ShouldInline(section string) bool {
	sectionPath := c.ModelPath
	if sectionPath != "" {
		sectionPath += "."
	}
	sectionPath += section

	if c.Flags.InlineAll {
		return true
	}
	for _, path := range c.Flags.InlinePaths {
		if path == sectionPath {
			return true
		}
		if strings.HasPrefix(path, sectionPath) {
			return true
		}
		if path[0] == '.' && strings.HasSuffix(sectionPath, path) {
			return true
		}
	}
	return false
}

func (c *Context) MatchPush(match int) int {
	c.MatchStack = append(c.MatchStack, match)
	return match
}

func (c *Context) MatchPop() int {
	size := len(c.MatchStack)
	match := c.MatchStack[size-1]
	c.MatchStack = c.MatchStack[:size-1]
	return match
}

func (c *Context) MatchLast() int {
	if len(c.MatchStack) == 0 {
		return 0
	}
	return c.MatchStack[len(c.MatchStack)-1]
}

type Filter struct {
	// 0->id
	// 1->groupType.id
	// 2->groupType.resType.id
	// 3->groupType.resType.version.id
	Depth []string
	Field string
	Value string
}

func ParseFilterExpr(str string) *Filter {
	// filter=[GROUP[.RESOURCE].]FIELD=VALUE
	attribute, value, _ := strings.Cut(str, "=")
	parts := strings.SplitN(attribute, ".", 4)

	return &Filter{
		Depth: parts[:len(parts)-1],
		Field: parts[len(parts)-1],
		Value: value,
	}
}

func ParseFilterExprs(exprs []string) []*Filter {
	if len(exprs) == 0 {
		return nil
	}
	res := []*Filter{}
	for _, expr := range exprs {
		res = append(res, ParseFilterExpr(expr))
	}
	return res
}

// TypeName->weirdCaseFieldName->realCaseFieldName
var FieldMapping = map[string]map[string]string{}

func FieldByName(v reflect.Value, field string) (reflect.Value, bool) {
	vType := v.Type()
	typeMap := FieldMapping[vType.Name()]
	if typeMap == nil {
		typeMap = map[string]string{}
		FieldMapping[vType.Name()] = typeMap
	}

	realName := typeMap[field]
	if realName == "" {
		num := vType.NumField()
		for j := 0; j < num; j++ {
			vField := vType.Field(j)
			if strings.EqualFold(field, vField.Name) {
				realName = vField.Name
				break
			}
		}
		if realName == "" {
			// log.Printf("Can't find %q in resource %q", field, vType.Name())
			return reflect.Zero(reflect.TypeOf(0)), false
		}
		typeMap[field] = realName
	}

	return v.FieldByName(realName), true
}

func (ctx *Context) MatchFilters(group, resource, version string, obj interface{}) int {
	// rc: -1 exclude, +1 include, 0 no-comment
	if len(ctx.Filters) == 0 {
		return 1
	}

	v := reflect.ValueOf(obj).Elem()

	log.Printf("Filters: %#v\n", ctx.Filters)

	gotOne := 0
	for _, f := range ctx.Filters {
		id := v.FieldByName("ID").Interface()
		log.Printf("Checking %s.%s.%s.%v: f.%s\n", group, resource, version, id, f.Field)

		l := len(f.Depth)
		log.Printf("depth: %d  %#v", l, f.Depth)

		if (l == 0) ||
			(l == 1 && group == f.Depth[0] && resource == "" && version == "") ||
			(l == 2 && group == f.Depth[0] && resource == f.Depth[1] && version == "") ||
			(l == 3 && group == f.Depth[0] && resource == f.Depth[1] && version == f.Depth[2]) {

			log.Printf("Type: %v\n", v.Type())

			objField, present := FieldByName(v, f.Field)
			if !present {
				log.Printf("Can't find %q in %q", f.Field, v.Type().Name())
				if l == 0 {
					continue
				}
				return -1
			}
			prop := objField.Interface()
			if prop == nil {
				// Not there, so exclude this obj
				return -1
			}
			str := fmt.Sprintf("%v", prop)
			log.Printf("Diffing %q & %q", str, f.Value)
			if str != f.Value {
				// No match, so exclude this obj
				return -1
			}

			gotOne |= 1
		}
	}

	return gotOne
}
