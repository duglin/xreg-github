package registry

import (
	"encoding/json"
	"fmt"
	// log "github.com/duglin/dlog"
	"io"
	"io/ioutil"
	// "reflect"
	"strings"
)

type JsonWriter struct {
	writer      io.Writer
	info        *RequestInfo
	indent      string
	unusedColls [][]string // [level][remaining coll names on this level]

	results   [][]*any // results of DB query
	resultPos int      // index into results array of current row
	Obj       *Obj     // Current row in the DB results
}

func NewJsonWriter(w io.Writer, info *RequestInfo, results [][]*any) *JsonWriter {
	return &JsonWriter{
		writer:      w,
		info:        info,
		indent:      "",
		unusedColls: make([][]string, 4),
		results:     results,
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
		ID:     jw.info.Registry.ID,
		Values: map[string]any{
			"id":          jw.info.Registry.ID,
			"name":        jw.info.Registry.Name,
			"description": jw.info.Registry.Description,
			"specVersion": jw.info.Registry.SpecVersion,
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

func (jw *JsonWriter) WriteCollectionHeader(extra string) (string, error) {
	myPlural := jw.Obj.Plural

	saveWriter := jw.writer
	saveExtra := extra

	if !jw.info.ShouldInline(jw.Obj.Abstract) {
		jw.writer = ioutil.Discard
	}

	jw.Printf("%s\n%s%q: ", extra, jw.indent, jw.Obj.Plural)
	extra = ","
	count, err := jw.WriteCollection()
	if err != nil {
		return "", err
	}

	if jw.writer == ioutil.Discard {
		jw.writer = saveWriter
		extra = saveExtra
	}

	jw.Printf("%s\n%s\"%sCount\": %d,\n", extra, jw.indent, myPlural, count)
	jw.Printf("%s\"%sUrl\": %q", jw.indent, myPlural, "...")

	return ",", nil
}

func (jw *JsonWriter) WriteCollection() (int, error) {
	jw.Printf("{")
	jw.Indent()

	extra := ""
	myLevel := 0
	myPlural := ""
	count := 0

	for jw.Obj != nil {
		if myLevel == 0 {
			myLevel = jw.Obj.Level
			myPlural = jw.Obj.Plural
		}
		if jw.Obj.Level > myLevel { // Process a child
			jw.NextObj()
			continue
		}

		if jw.Obj.Level < myLevel || jw.Obj.Plural != myPlural {
			// Stop on a new parent or a new sibling collection
			break
		}

		jw.Printf("%s\n%s%q: ", extra, jw.indent, jw.Obj.Values["id"])
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
	if jw.Obj == nil {
		jw.Printf("{}")
		return nil
	}

	var err error
	extra := ""
	myLevel := jw.Obj.Level

	jw.Printf("{")
	jw.Indent()

	keys := []string{"specVersion", "id", "name", "epoch", "self", "latestId",
		"latestUrl", "description", "docs", "format", // not "tags"
		"createdBy", "createdOn", "modifiedBy", "modifiedOn"}

	// Write the well-known attributes first, in order
	for _, key := range keys {
		val, ok := jw.Obj.Values[key]
		if ok {
			buf, _ := json.Marshal(val)
			jw.Printf("%s\n%s%q: %s", extra, jw.indent, key, string(buf))
			delete(jw.Obj.Values, key)
			extra = ","
		}
	}

	// Now write all extensions - buffering "tags"
	tags := map[string]any{}
	for _, key := range SortedKeys(jw.Obj.Values) {
		val, _ := jw.Obj.Values[key]
		if strings.HasPrefix(key, "tags.") {
			tags[key[5:]] = val
			continue
		}
		buf, _ := json.Marshal(val)
		jw.Printf("%s\n%s%q: %s", extra, jw.indent, key, string(buf))
		extra = ","
	}

	// And finally any tags
	if len(tags) > 0 {
		buf, _ := json.MarshalIndent(tags, jw.indent, "  ")
		jw.Printf("%s\n%s\"tags\": %s", extra, jw.indent, string(buf))
		extra = ","
	}

	if extra != "" {
		extra += "\n" // just because it looks nicer with a blank line
	}

	jw.LoadCollections(myLevel) // load the list of current collections
	jw.NextObj()

	if jw.Obj != nil && jw.Obj.Level > myLevel {
		extra = jw.WritePreCollections(extra, jw.Obj.Plural, myLevel)
		if extra, err = jw.WriteCollectionHeader(extra); err != nil {
			return err
		}
	}
	extra = jw.WritePostCollections(extra, myLevel)

	jw.Outdent()
	jw.Printf("\n%s}", jw.indent)

	return nil
}

func (jw *JsonWriter) LoadCollections(level int) {
	names := []string{}
	if level == 0 {
		names = SortedKeys(jw.info.Registry.Model.Groups)
	} else if level == 1 {
		gName, _ := strings.CutSuffix(jw.Obj.Abstract, "/")
		names = SortedKeys(jw.info.Registry.Model.Groups[gName].Resources)
	} else if level == 2 {
		names = []string{"versions"}
	} else if level == 3 {
		names = []string{} // no children of versions
	} else {
		panic("Too many levels")
	}
	jw.unusedColls[level] = names
}

func (jw *JsonWriter) WritePreCollections(extra string, plural string, level int) string {
	for i, collName := range jw.unusedColls[level] {
		if collName == plural {
			jw.unusedColls[level] = jw.unusedColls[level][i+1:]
			break
		}
		jw.Printf("%s\n%s\"%sCount\": 0,\n", extra, jw.indent, collName)
		jw.Printf("%s\"%sUrl\": \"...\"", jw.indent, collName)
		extra = ","
	}
	return extra
}

func (jw *JsonWriter) WritePostCollections(extra string, level int) string {
	for _, collName := range jw.unusedColls[level] {
		jw.Printf("%s\n%s\"%sCount\": 0,\n", extra, jw.indent, collName)
		jw.Printf("%s\"%sUrl\": \"...\"", jw.indent, collName)
		extra = ","
	}

	jw.unusedColls[level] = nil
	return extra
}
