package registry

import (
	"fmt"
	// log "github.com/duglin/dlog"
	"io"
	"reflect"
)

type JsonWriter struct {
	writer         io.Writer
	indent         string
	plurals        []string   // used coll names in current branch
	remainingColls [][]string // [level][remaining coll names on this level]
	reg            *Registry

	results   [][]*any
	resultPos int
	Obj       *Obj
}

func NewJsonWriter(w io.Writer, reg *Registry, results [][]*any) *JsonWriter {
	return &JsonWriter{
		writer:         w,
		indent:         "",
		plurals:        make([]string, 4),
		remainingColls: make([][]string, 4),
		reg:            reg,
		results:        results,
	}
}

func (jw *JsonWriter) Printf(format string, args ...any) {
	fmt.Fprintf(jw.writer, format, args...)
}

func (jw *JsonWriter) OptPrintf(format string, args ...any) {
	if len(args) == 0 || args[len(args)-1].(string) != "" {
		jw.Printf(format, args...)
	}
}

func (jw *JsonWriter) Indent() {
	jw.indent += "  "
}

func (jw *JsonWriter) Outdent() {
	if l := len(jw.indent); l > 1 {
		jw.indent = jw.indent[:l-2]
	} else {
		panic("Outdent!!!")
	}
}

func (jw *JsonWriter) NextObj() *Obj {
	jw.Obj, jw.resultPos = readObj(jw.results, jw.resultPos)
	return jw.Obj
}

func (jw *JsonWriter) WriteRegistry() error {
	regObj := &Obj{
		Level:  0,
		Plural: "registries",
		ID:     jw.reg.ID,
		Values: map[string]any{
			"id":          jw.reg.ID,
			"name":        jw.reg.Name,
			"description": jw.reg.Description,
			"specVersion": jw.reg.SpecVersion,
			"self":        "...",
		},
	}

	jw.Obj = regObj
	if err := jw.WriteObject(); err != nil {
		return err
	}
	jw.Printf("\n")
	return nil
}

func (jw *JsonWriter) WriteCollectionHeader(extra string) error {
	myPlural := jw.Obj.Plural

	jw.Printf("%s\n%s%q: ", extra, jw.indent, jw.Obj.Plural)
	count, err := jw.WriteCollection()
	if err != nil {
		return err
	}

	jw.Printf(",\n%s\"%sCount\": %d,\n", jw.indent, myPlural, count)
	jw.Printf("%s\"%sUrl\": %q", jw.indent, myPlural, "...")

	return nil
}

func (jw *JsonWriter) WriteCollection() (int, error) {
	jw.Printf("{")
	jw.Indent()

	extra := ""
	myLevel := jw.Obj.Level
	myPlural := jw.Obj.Plural
	count := 0

	for jw.Obj != nil {
		if jw.Obj.Level > myLevel { // Process a child
			jw.NextObj()
			continue
		}

		if jw.Obj.Level < myLevel || jw.Obj.Plural != myPlural {
			// Process a new parent or a new sibling collection
			break
		}

		jw.Printf("%s\n", extra)
		jw.Printf("%s%q: ", jw.indent, jw.Obj.Values["id"])
		if err := jw.WriteObject(); err != nil {
			return count, err
		}

		count++
		extra = ","
	}

	jw.Outdent()
	jw.Printf("\n%s}", jw.indent)

	return count, nil
}

func (jw *JsonWriter) WriteObject() error {
	extra := ""
	myLevel := jw.Obj.Level

	jw.Printf("{")
	jw.Indent()

	keys := []string{"id", "name", "description", "specVersion", "self"}
	for _, key := range SortedKeys(jw.Obj.Values) {
		match := false
		for _, k := range keys {
			if k == key {
				match = true
				break
			}
		}
		if !match {
			keys = append(keys, key)
		}
	}

	for _, key := range keys {
		val, ok := jw.Obj.Values[key]
		if !ok {
			continue
		}

		if reflect.TypeOf(val).Kind() == reflect.String {
			jw.Printf("%s\n%s%q: %q", extra, jw.indent, key, val)
		} else {
			jw.Printf("%s\n%s%q: %v", extra, jw.indent, key, val)
		}
		extra = ","
	}
	if extra != "" {
		extra += "\n"
	}

	jw.LoadCollections(myLevel) // load it up
	jw.NextObj()

	if jw.Obj != nil && jw.Obj.Level > myLevel {
		extra = jw.WritePreCollections(extra, jw.Obj.Plural, myLevel)
		if err := jw.WriteCollectionHeader(extra); err != nil {
			return err
		}
	}
	extra = jw.WritePostCollections(extra, myLevel)

	jw.Outdent()
	jw.Printf("\n%s}", jw.indent)

	return nil
}

func (jw *JsonWriter) LoadCollections(level int) {
	if jw.plurals[level] != "" {
		return // already loaded
	}

	names := []string{}
	if level == 0 {
		names = SortedKeys(jw.reg.Model.Groups)
	} else if level == 1 {
		groupName := jw.plurals[0]
		names = SortedKeys(jw.reg.Model.Groups[groupName].Resources)
	} else if level == 2 {
		names = []string{"versions"}
	} else if level == 3 {
		names = []string{} // no children of versions
	} else {
		panic("Too many levels")
	}
	jw.remainingColls[level] = names
}

func (jw *JsonWriter) WritePreCollections(extra string, plural string, level int) string {
	jw.plurals[level] = plural

	for i, collName := range jw.remainingColls[level] {
		if collName == plural {
			jw.remainingColls[level] = jw.remainingColls[level][i+1:]
			break
		}
		jw.Printf("%s\n%s\"%sCount\": 0,\n", extra, jw.indent, collName)
		jw.Printf("%s\"%sUrl\": \"...\"", jw.indent, collName)
		extra = ","
	}
	return extra
}

func (jw *JsonWriter) WritePostCollections(extra string, level int) string {
	for _, collName := range jw.remainingColls[level] {
		jw.Printf("%s\n%s\"%sCount\": 0,\n", extra, jw.indent, collName)
		jw.Printf("%s\"%sUrl\": \"...\"", jw.indent, collName)
		extra = ","
	}

	jw.remainingColls[level] = nil
	jw.plurals[level] = ""
	return extra
}
