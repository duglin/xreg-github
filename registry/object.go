package registry

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

type Jsoner interface {
	ToJson(bld *strings.Builder, prefix string, indent string)
}

var JsonerType = reflect.TypeOf((*Jsoner)(nil)).Elem()

// Object

type Object struct {
	Children []*Child
}

func (o *Object) Len() int {
	return len(o.Children)
}

func NewObject() *Object {
	return &Object{Children: []*Child{}}
}

type Child struct {
	Name  string
	Value interface{}
}

func (o *Object) AddProperty(name string, val interface{}) {
	if name != "" && reflect.TypeOf(val).Kind() == reflect.String {
		s, ok := val.(string)
		if ok && s == "" {
			return
		}
	}

	child := &Child{
		Name:  name,
		Value: val,
	}
	o.Children = append(o.Children, child)
}

func (o *Object) GetProperty(name string) interface{} {
	for _, c := range o.Children {
		if c.Name == name {
			return c.Value
		}
	}
	return nil
}

func (o *Object) ToJson(bld *strings.Builder, prefix, indent string) {
	if o == nil || len(o.Children) == 0 {
		bld.WriteString("{}")
		return
	}
	bld.WriteString("{\n")

	for i, c := range o.Children {
		if c.Name == "" {
			bld.WriteString("\n")
			continue
		}

		val := c.Value

		if reflect.TypeOf(val).String() != CollectionType {
			bld.WriteString(fmt.Sprintf(prefix+indent+"\"%s\": ", c.Name))
		}
		if reflect.TypeOf(val).Implements(JsonerType) {
			(val.(Jsoner)).ToJson(bld, prefix+indent, indent)
		} else {
			buf, _ := json.MarshalIndent(val, prefix+indent, indent)
			bld.WriteString(strings.TrimSpace(string(buf)))
		}
		if i+1 != len(o.Children) {
			bld.WriteString(",")
		}
		bld.WriteString("\n")
	}
	bld.WriteString(prefix + "}")
}

// Collection
type Collection struct {
	Name   string
	URL    string
	Inline bool
	Object *Object
}

var CollectionType = reflect.TypeOf((*Collection)(nil)).String()

func (c *Collection) ToJson(bld *strings.Builder, prefix, indent string) {
	bld.WriteString("\n")
	bld.WriteString(fmt.Sprintf(prefix+"\"%sUrl\": \"%s\",\n", c.Name, c.URL))
	bld.WriteString(fmt.Sprintf(prefix+"\"%sCount\": %d", c.Name, len(c.Object.Children)))

	if c.Inline {
		bld.WriteString(",\n")
		bld.WriteString(fmt.Sprintf(prefix+"\"%s\": ", c.Name))
		c.Object.ToJson(bld, prefix, indent)
	}
}

// Array

type Array struct {
	Values []interface{}
}

func NewArray() *Array {
	return &Array{Values: []interface{}{}}
}

func (a *Array) AddItem(val interface{}) {
	a.Values = append(a.Values, val)
}

func (a *Array) Len() int {
	return len(a.Values)
}

func (a *Array) ToJson(bld *strings.Builder, prefix, indent string) {
	if len(a.Values) == 0 {
		bld.WriteString("[]")
		return
	}
	bld.WriteString("[\n")
	for i, item := range a.Values {
		if reflect.TypeOf(item).Implements(JsonerType) {
			// bld.WriteString(prefix + indent)
			(item.(Jsoner)).ToJson(bld, prefix+indent, indent)
		} else {
			buf, _ := json.MarshalIndent(item, prefix+indent, indent)
			bld.WriteString(prefix + indent)
			// bld.WriteString(strings.TrimSpace(string(buf)))
			bld.WriteString(string(buf))
		}
		if i+1 != len(a.Values) {
			bld.WriteString(",")
		}
		bld.WriteString("\n")
	}
	bld.WriteString(prefix + "]")
}

// Stuff

func test() {
	obj := NewObject()
	bld := &strings.Builder{}

	obj.AddProperty("item1", 5)
	obj.AddProperty("item2", 55)

	array1 := NewArray()
	array1.AddItem("hello")

	obj2 := NewObject()
	obj2.AddProperty("foo", "bar")
	array1.AddItem(obj2)
	obj.AddProperty("arr1", array1)
	obj2.AddProperty("foo2", "bar")

	obj.ToJson(bld, "", "  ")
	fmt.Printf("%s\n", bld.String())
}
