package registry

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	// dlog "github.com/duglin/dlog"
)

type Context struct {
	Flags   *RegistryFlags
	BaseURL string

	dataStack  []string // URL path entities we walked thru (model+IDs)
	modelStack []string // resource model entities we walked thru

	buffer      strings.Builder
	OrFilters   [][]*Filter // [ORs][ANDs]
	FilterStack []string

	Filters []*Filter // OLD
}

func (c *Context) BaseURLPush(word string) string {
	c.BaseURL += "/" + word
	return c.BaseURL
}

func (c *Context) DataURL() string {
	dPath := strings.Join(c.dataStack, "/")
	if c.Flags.AsDoc {
		return "#" + "/" + dPath
	}
	return c.BaseURL + "/" + strings.ToLower(dPath)
}

func (c *Context) DocifyURL(daURL string) string {
	if c.Flags.AsDoc && strings.HasPrefix(daURL, c.BaseURL) {
		return "#" + daURL[len(c.BaseURL):]
	}
	return daURL
}

func (c *Context) DataPush(word string) string {
	c.dataStack = append(c.dataStack, word)
	return strings.Join(c.dataStack, "/")
}

func (c *Context) DataPop() string {
	l := len(c.dataStack)
	if l == 0 {
		panic("Popping empty dataStack")
	}
	c.dataStack = c.dataStack[:l-1]
	return strings.Join(c.dataStack, "/")
}

func (c *Context) ModelPush(word string) string {
	c.modelStack = append(c.modelStack, word)
	return strings.Join(c.modelStack, ".")
}

func (c *Context) ModelPop() string {
	l := len(c.modelStack)
	if l == 0 {
		panic("Popping empty modelStack")
	}
	save := c.modelStack[l-1]
	c.modelStack = c.modelStack[:l-1]
	return save
}

func (c *Context) ShouldInline(section string) bool {
	sectionPath := strings.Join(c.modelStack, ".")
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

func (c *Context) FilterPush(word string) string {
	c.FilterStack = append(c.FilterStack, word)
	return word
}

func (c *Context) FilterPop() string {
	size := len(c.FilterStack)
	word := c.FilterStack[size-1]
	c.FilterStack = c.FilterStack[:size-1]
	return word
}

type Filter struct {
	// 0->id
	// 1->groupType.id
	// 2->groupType.resType.id
	// 3->groupType.resType.version.id
	Depth []string
	Field string
	Value string

	Path      string // everything before the "=". Value is stuff after the =
	HasEquals bool

	ModelPath  string
	ModelStack []string
}

func ParseFilterExpr(reg *Registry, paths []string, str string) (*Filter, error) {
	// filter=[GROUP[.RESOURCE].]FIELD=VALUE
	attribute, value, hasEquals := strings.Cut(str, "=")
	parts := strings.SplitN(attribute, ".", 4)

	modelElement := reg.GenericModel

	for i, p := range paths {
		if i%2 == 0 {
			if modelElement.Children == nil {
				return nil, fmt.Errorf("Unexpected %q in path", p)
			}
			for _, c := range modelElement.Children {
				// if strings.EqualFold(p, c.Plural) {
				if p == c.Plural {
					modelElement = c
					break
				}
			}
			// modelElement = modelElement.Children[p]
			if modelElement == nil {
				return nil, fmt.Errorf("Unexpected %q in path", p)
			}
			continue
		}
	}

	fieldStart := 0
	if modelElement != nil {
		for _, p := range parts {
			if modelElement = modelElement.Children[p]; modelElement == nil {
				break
			}
			fieldStart++
		}
	}
	modelPath := strings.Join(parts[:fieldStart], ".")

	val := fmt.Sprintf("=%q", value)
	if !hasEquals {
		val = "no '='"
	}

	// dlog.SetVerbose(1)
	log.Printf("Filter> Pth:%s  Fld:%s %s", modelPath, parts[fieldStart:], val)

	return &Filter{
		Depth:     parts[:len(parts)-1],
		Field:     strings.Join(parts[fieldStart:], "."),
		Value:     value,
		Path:      attribute,
		HasEquals: hasEquals,

		ModelPath:  modelPath,
		ModelStack: parts[:fieldStart],
	}, nil
}

func ParseFilterExprs(reg *Registry, paths []string, exprs []string) ([]*Filter, error) {
	if len(exprs) == 0 {
		return nil, nil
	}
	res := []*Filter{}
	for _, expr := range exprs {
		filter, err := ParseFilterExpr(reg, paths, expr)
		if err != nil {
			return nil, err
		}
		res = append(res, filter)
	}
	return res, nil
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

func (ctx *Context) Filter(obj interface{}) (gotOne int, err error) {
	if len(ctx.Filters) == 0 {
		return
	}

	// rc: -1 exclude, +1 include, 0 no-comment
	log.Printf("Filter check(len:%d) - obj: %T", len(ctx.Filters), obj)
	log.Printf("Stack Prefix: %q", ctx.FilterStack)

	defer func() {
		log.Printf("   --> %d, err: %s", gotOne, err)
	}()

	if obj == nil || reflect.ValueOf(obj).IsNil() {
		// Odd case but if we pass-in nil and there are filters then
		// just say it doesn't match and stop. Avoids having to add a ton
		// of logic before each call to this func all over the place
		// log.Printf("--> -1, nil")
		// return -1, nil
		gotOne = -1
		return
	}

	// objValue := reflect.ValueOf(obj)
	// objElem := reflect.ValueOf(obj).Elem()
	objType := reflect.TypeOf(obj).Elem().Name()
	// log.Printf("   ObjValue: %s", objValue.String())
	// log.Printf("   Type: %s", reflect.TypeOf(obj).Elem().Name())
	// log.Printf("   Stack Prefix: %q", strings.Join(ctx.FilterStack, "."))

	for i, filter := range ctx.Filters {
		log.Printf("%d) %s=%s", i, filter.Path, filter.Value)
		if !HasPrefix(filter.Path, ctx.FilterStack) {
			log.Printf("   Skipping - missing prefix")
			continue
		}
		prefix := strings.Join(ctx.FilterStack, ".")
		length := len(prefix)
		if len(filter.Path) == length {
			// log.Printf("--> 0, Missing prop")
			// return 0, fmt.Errorf("Missing property for %q in filter", prefix)
			err = fmt.Errorf("Missing property for %q in filter", prefix)
			return
		}
		if length != 0 {
			length++ // add/skip trailing "."
		}
		remainder := filter.Path[length:]
		words := strings.Split(remainder, ".")
		log.Printf("   Remainder words: %v", words)

		if len(words) == 0 {
			// log.Printf("   --> 0, Bad filter expr: %s", filter.Path)
			// return 0, fmt.Errorf("Bad filter expression: %s", filter.Path)
			err = fmt.Errorf("Bad filter expression: %s", filter.Path)
			return
		}

		/*
			if objType == "GroupCollection" {
				gColl := obj.(*GroupCollection)

				// Do some error checking - make sure the word is even valid
				model := gColl.Registry.Model.Groups[words[0]]
				if model == nil {
					// log.Printf("   --> -1, Uknown group %q", words[0])
					// return -1, fmt.Errorf("Unknown Group: %s", words[0])
					err =fmt.Errorf("Unknown Group: %s", words[0])
					return
				}

				if words[0] != gColl.GroupModel.Plural {
					// log.Printf("   --> -1, nil")
					// return -1, nil
					gotOne = -1
					return
				}
			}
		*/

		if objType == "Group" {
			// Search for RESOURCE names
			group := obj.(*Group)
			if group.GroupCollection.GroupModel.Resources[words[0]] != nil {
				log.Printf("   Found %q, which is ok", words[0])
				continue
			}
			// log.Printf("   Check %q against %q", remainder, filter.Value)
			match := CheckFieldValue(obj, remainder, filter.Value)
			if match {
				gotOne = 1
				continue
			}
			// log.Printf("   --> -1, nil")
			// return -1, nil
			gotOne = -1
			return
		}

		/*
			if objType == "ResourceCollection" {
				rColl := obj.(*ResourceCollection)
				// Do some error checking - make sure the word is even valid
				model := rColl.Group.GroupCollection.GroupModel.Resources[words[0]]
				if model == nil {
					// log.Printf("   --> -1, Unknown resource: %s", words[0])
					// return -1, fmt.Errorf("Unknown Resource: %s", words[0])
					err =  fmt.Errorf("Unknown Resource: %s", words[0])
					return
				}

				if words[0] != rColl.ResourceModel.Plural {
					// log.Printf("   --> -1, nil")
					// return -1, nil
					gotOne = -1
					return
				}
			}
		*/

		if objType == "Version" {
			// Search for RESOURCE names
			if words[0] == "versions" {
				log.Printf("   Found %q, which is ok", words[0])
				continue
			}
			// log.Printf("   Check %q against %q", remainder, filter.Value)
			match := CheckFieldValue(obj, remainder, filter.Value)
			if match {
				gotOne = 1
				continue
			}
			// log.Printf("   --> -1, nil")
			// return -1, nil
			gotOne = -1
			return
		}
	}

	// log.Printf("   --> %d, nil", gotOne)
	// return gotOne, nil
	return
}

func HasPrefix(fPath string, fStack []string) bool {
	if len(fStack) == 0 {
		return true
	}
	stack := strings.Join(fStack, ".")
	if !strings.HasPrefix(fPath, stack) {
		return false
	}

	return len(fPath) == len(stack) || fPath[len(stack)] == '.'
}

func CheckFieldValue(obj interface{}, field string, value interface{}) bool {
	v := reflect.ValueOf(obj).Elem()
	objField, present := FieldByName(v, field)
	if !present {
		log.Printf("   Can't find %q in %q", field, v.Type().Name())
		/*
			if l == 0 {
				continue
			}
			return -1
		*/
		return false
	}
	prop := objField.Interface()
	if prop == nil {
		// Not there, so exclude this obj
		// return -1
		return false
	}
	str := strings.ToUpper(fmt.Sprintf("%v", prop))
	valueStr := strings.ToUpper(fmt.Sprintf("%v", value))
	match := strings.Contains(str, valueStr)
	log.Printf("   Diff (%q)%q=%q -> %v", field, str, valueStr, match)
	if !match {
		// if str != value {
		// No match, so exclude this obj
		return false
	}
	return true
}

func (ctx *Context) HasChildrenFilters() bool {
	l := len(ctx.FilterStack)
	for _, filter := range ctx.Filters {
		if len(filter.ModelStack) > l && reflect.DeepEqual(ctx.FilterStack, filter.ModelStack[:l]) {
			return true
		}
	}
	return false
}
