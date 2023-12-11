package registry

import (
	"encoding/json"
	"fmt"
	log "github.com/duglin/dlog"
	"path"
	"strings"
)

type JsonWriter struct {
	info        *RequestInfo
	indent      string
	collPaths   []string   // [level] URL path to the root of Colls
	unusedColls [][]string // [level][remaining coll names on this level]

	results *Result // results of DB query
	Entity  *Entity // Current row in the DB results
}

func NewJsonWriter(info *RequestInfo, results *Result) *JsonWriter {
	return &JsonWriter{
		info:        info,
		indent:      "",
		collPaths:   make([]string, 4),
		unusedColls: make([][]string, 4),
		results:     results,
	}
}

func (jw *JsonWriter) Print(str string) {
	fmt.Fprint(jw.info, str)
}

func (jw *JsonWriter) Printf(format string, args ...any) {
	fmt.Fprintf(jw.info, format, args...)
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

func (jw *JsonWriter) NextEntity() *Entity {
	jw.Entity = readNextEntity(jw.results)
	/*
		pc, _, line, _ := runtime.Caller(1)
		log.VPrintf(4, "Caller: %s:%d", path.Base(runtime.FuncForPC(pc).Name()), line)
		log.VPrintf(4, "  > Next: %v", jw.Entity)
	*/
	return jw.Entity
}

func (jw *JsonWriter) WriteCollectionHeader(extra string) (string, error) {
	myPlural := jw.Entity.Plural
	myURL := fmt.Sprintf("%s/%s", jw.info.BaseURL, path.Dir(jw.Entity.Path))

	saveWriter := jw.info.HTTPWriter
	saveExtra := extra

	// TODO optimize this to avoid the ioutil.Discard and just count the
	// children from the result set instead
	if !jw.info.ShouldInline(jw.Entity.Abstract) {
		jw.info.HTTPWriter = DefaultDiscardWriter
	}

	jw.Printf("%s\n%s%q: ", extra, jw.indent, jw.Entity.Plural)
	extra = ","
	count, err := jw.WriteCollection()
	if err != nil {
		return "", err
	}

	if jw.info.HTTPWriter == DefaultDiscardWriter {
		jw.info.HTTPWriter = saveWriter
		extra = saveExtra
	}

	jw.Printf("%s\n%s\"%scount\": %d,\n", extra, jw.indent, myPlural, count)
	jw.Printf("%s\"%surl\": %q", jw.indent, myPlural, myURL)

	return ",", nil
}

func (jw *JsonWriter) WriteCollection() (int, error) {
	jw.Printf("{")
	jw.Indent()

	extra := ""
	myLevel := 0
	myPlural := ""
	count := 0

	for jw.Entity != nil {
		if myLevel == 0 {
			myLevel = jw.Entity.Level
			myPlural = jw.Entity.Plural
		}

		if jw.Entity.Level > myLevel { // Process a child
			jw.NextEntity()
			continue
		}

		if jw.Entity.Level < myLevel || jw.Entity.Plural != myPlural {
			// Stop on a new parent or a new sibling collection
			break
		}

		jw.Printf("%s\n%s%q: ", extra, jw.indent,
			jw.Entity.Props[NewPPP("id").DB()])
		if err := jw.WriteEntity(); err != nil {
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

func (jw *JsonWriter) WriteEntity() error {
	log.VPrintf(3, ">Enter: WriteEntity (%v)", jw.Entity)
	defer log.VPrintf(3, "<Exit: WriteEntity")

	if jw.Entity == nil {
		jw.Printf("{}")
		return nil
	}

	extra := ""
	myLevel := jw.Entity.Level
	log.VPrintf(4, "Level: %d", myLevel)

	jw.Printf("{")
	jw.Indent()

	jsonIt := func(e *Entity, info *RequestInfo, key string, val any) error {
		buf, _ := json.MarshalIndent(val, jw.indent, "  ")
		jw.Printf("%s\n%s%q: %s", extra, jw.indent, key, string(buf))
		extra = ","
		return nil
	}

	err := jw.Entity.SerializeProps(jw.info, jsonIt)
	if err != nil {
		panic(err)
	}

	// Add resource content released properties
	if myLevel >= 2 {
		if val := jw.Entity.GetPropFromUI("#resourceURL"); val != nil {
			gModel := jw.info.Registry.Model.Groups[jw.info.GroupType]
			rModel := gModel.Resources[jw.info.ResourceType]
			singular := rModel.Singular

			url := val.(string)
			jw.Printf("%s\n%s%q: %q", extra, jw.indent, singular+"Url", url)
			extra = ","
		}
	}

	// Now show all of the nested collections
	if extra != "" {
		extra += "\n" // just because it looks nicer with a blank line
	}

	jw.LoadCollections(myLevel) // load the list of current collections
	jw.NextEntity()

	for jw.Entity != nil && jw.Entity.Level > myLevel {
		extra = jw.WritePreCollections(extra, jw.Entity.Plural, myLevel)

		if extra, err = jw.WriteCollectionHeader(extra); err != nil {
			return err
		}
	}
	extra = jw.WritePostCollections(extra, myLevel)

	// And finally done with this Entity
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
		gName, _ := strings.CutSuffix(jw.Entity.Abstract, IN_STR)
		names = SortedKeys(jw.info.Registry.Model.Groups[gName].Resources)
	} else if level == 2 {
		names = []string{"versions"}
	} else if level == 3 {
		names = []string{} // no children of versions
	} else {
		panic("Too many levels")
	}
	jw.unusedColls[level] = names

	p := jw.Entity.Path + "/"
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

		jw.Printf("%s\n%s\"%scount\": 0,\n", extra, jw.indent, collName)
		jw.Printf("%s\"%surl\": \"%s/%s%s\"", jw.indent, collName,
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

		jw.Printf("%s\n%s\"%scount\": 0,\n", extra, jw.indent, collName)
		jw.Printf("%s\"%surl\": \"%s/%s%s\"", jw.indent, collName,
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
				res += IN_STR
			}
			res += part
		}
	}
	if addSlash {
		res += IN_STR
	}
	return res
}
