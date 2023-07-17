package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	// "runtime"
	// "reflect"
	log "github.com/duglin/dlog"
	"strings"
)

type JsonWriter struct {
	writer      io.Writer
	info        *RequestInfo
	indent      string
	collPaths   []string   // [level] URL path to the root of Colls
	unusedColls [][]string // [level][remaining coll names on this level]

	results *Result // results of DB query
	Obj     *Obj    // Current row in the DB results
}

func NewJsonWriter(w io.Writer, info *RequestInfo, results *Result) *JsonWriter {
	return &JsonWriter{
		writer:      w,
		info:        info,
		indent:      "",
		collPaths:   make([]string, 4),
		unusedColls: make([][]string, 4),
		results:     results,
	}
}

func (jw *JsonWriter) Print(str string) {
	fmt.Fprint(jw.writer, str)
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
	jw.Obj = readObj(jw.results)
	/*
		pc, _, line, _ := runtime.Caller(1)
		log.VPrintf(4, "Caller: %s:%d", path.Base(runtime.FuncForPC(pc).Name()), line)
		log.VPrintf(4, "  > Next: %v", jw.Obj)
	*/
	return jw.Obj
}

func (jw *JsonWriter) WriteCollectionHeader(extra string) (string, error) {
	myPlural := jw.Obj.Plural
	myURL := fmt.Sprintf("%s/%s", jw.info.BaseURL, path.Dir(jw.Obj.Path))

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
	jw.Printf("%s\"%sUrl\": %q", jw.indent, myPlural, myURL)

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
	if extra != "" {
		jw.Printf("\n%s", jw.indent)
	}
	jw.Print("}")

	return count, nil
}

// This allows for us to choose the order and define custom logic per prop
var orderedProps = []struct {
	key    string                // prop name
	levels string                // only show for these levels
	fn     func(*JsonWriter) any // we'll Marshal the 'any'
}{
	{"specVersion", "", nil},
	{"id", "", nil},
	{"name", "", nil},
	{"epoch", "23", nil},
	{"self", "", func(jw *JsonWriter) any {
		return jw.info.BaseURL + "/" + jw.Obj.Path
	}},
	{"latestId", "2", nil},
	{"latestUrl", "2", func(jw *JsonWriter) any {
		val := jw.Obj.Values["latestId"]
		if IsNil(val) {
			return nil
		}
		return jw.info.BaseURL + "/" + jw.Obj.Path + "/versions/" + val.(string)
	}},
	{"description", "", nil},
	{"docs", "", nil},
	{"tags", "", func(jw *JsonWriter) any {
		var res map[string]string

		for _, key := range SortedKeys(jw.Obj.Values) {
			if key[0] > 't' {
				break
			}

			if strings.HasPrefix(key, "tags.") {
				val, _ := jw.Obj.Values[key]
				if res == nil {
					res = map[string]string{}
				}
				// Convert it to a string per the spec
				res[key[5:]] = fmt.Sprintf("%v", val)
				// Technically we shouldn't remove it but for now it's safe
				delete(jw.Obj.Values, key)
			}
		}
		return res
	}},
	{"createdBy", "", nil},
	{"createdOn", "", nil},
	{"modifiedBy", "", nil},
	{"modifiedOn", "", nil},
	{"model", "0", func(jw *JsonWriter) any {
		if jw.info.ShowModel {
			if jw.info.Registry.Model == nil {
				return &Model{}
			}
			return jw.info.Registry.Model
		}
		return nil
	}},
	{"baseURL", "-", nil}, // always skip
}

func (jw *JsonWriter) WriteObject() error {
	log.VPrintf(3, ">Enter: WriteObj (%v)", jw.Obj)
	defer log.VPrintf(3, "<Exit: WriteObj")

	if jw.Obj == nil {
		jw.Printf("{}")
		return nil
	}

	var err error
	extra := ""
	myLevel := jw.Obj.Level
	log.VPrintf(4, "Level: %d", myLevel)

	jw.Printf("{")
	jw.Indent()

	// Write the well-known attributes first, in order
	usedProps := map[string]bool{}
	for _, prop := range orderedProps {
		usedProps[prop.key] = true
		// Only show props that are for this level
		ch := rune('0' + byte(myLevel))
		if prop.levels != "" && !strings.ContainsRune(prop.levels, ch) {
			continue
		}

		// Even if it has a func, if there's a val in Values let it override
		val, ok := jw.Obj.Values[prop.key]
		if !ok && prop.fn != nil {
			val = prop.fn(jw)
			ok = !IsNil(val)
		}

		// Only write it if we have a value
		if ok {
			buf, _ := json.MarshalIndent(val, jw.indent, "  ")
			jw.Printf("%s\n%s%q: %s", extra, jw.indent, prop.key, string(buf))
			extra = ","
		}
	}

	// Now write the remaining properties/extensions (sorted)
	for _, key := range SortedKeys(jw.Obj.Values) {
		if usedProps[key] {
			continue
		}
		val, _ := jw.Obj.Values[key]
		buf, _ := json.Marshal(val)
		jw.Printf("%s\n%s%q: %s", extra, jw.indent, key, string(buf))
		extra = ","
	}

	if extra != "" {
		extra += "\n" // just because it looks nicer with a blank line
	}

	jw.LoadCollections(myLevel) // load the list of current collections
	jw.NextObj()

	for jw.Obj != nil && jw.Obj.Level > myLevel {
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
		if jw.info.Registry.Model != nil && jw.info.Registry.Model.Groups != nil {

			names = SortedKeys(jw.info.Registry.Model.Groups)
		}
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

	p := jw.Obj.Path + "/"
	if p == "/" {
		p = ""
	}
	jw.collPaths[level] = p
}

func (jw *JsonWriter) WritePreCollections(extra string, plural string, level int) string {
	for i, collName := range jw.unusedColls[level] {
		if collName == plural {
			jw.unusedColls[level] = jw.unusedColls[level][i+1:]
			break
		}
		p := Path2Abstract(jw.collPaths[level] + collName)
		if jw.info.ShouldInline(p) {
			jw.Printf("%s\n%s\"%s\": {}", extra, jw.indent, collName)
			extra = ","
		}

		jw.Printf("%s\n%s\"%sCount\": 0,\n", extra, jw.indent, collName)
		jw.Printf("%s\"%sUrl\": \"%s/%s%s\"", jw.indent, collName,
			jw.info.BaseURL, jw.collPaths[level], collName)
		extra = ","
	}
	return extra
}

func (jw *JsonWriter) WritePostCollections(extra string, level int) string {
	for _, collName := range jw.unusedColls[level] {
		p := Path2Abstract(jw.collPaths[level] + collName)
		if jw.info.ShouldInline(p) {
			jw.Printf("%s\n%s\"%s\": {}", extra, jw.indent, collName)
			extra = ","
		}

		jw.Printf("%s\n%s\"%sCount\": 0,\n", extra, jw.indent, collName)
		jw.Printf("%s\"%sUrl\": \"%s/%s%s\"", jw.indent, collName,
			jw.info.BaseURL, jw.collPaths[level], collName)
		extra = ","
	}

	jw.collPaths[level] = ""
	jw.unusedColls[level] = nil
	return extra
}

func Path2Abstract(path string) string {
	parts := strings.Split(path, "/")
	addSlash := strings.HasSuffix(path, "/")
	res := ""
	for i, part := range parts {
		if i%2 == 0 {
			if res != "" {
				res += "/"
			}
			res += part
		}
	}
	if addSlash {
		res += "/"
	}
	return res
}
