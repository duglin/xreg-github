package registry

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	log "github.com/duglin/dlog"
	"io"
	"net/http"
	"path"
	"strings"
)

type JsonWriter struct {
	info        *RequestInfo
	indent      string
	collPaths   map[int]string   // [eType] URL path to the root of Colls
	unusedColls map[int][]string // [eType][remaining coll names on this eType]

	results *Result // results of DB query
	Entity  *Entity // Current row in the DB results
	hasData bool
}

func NewJsonWriter(info *RequestInfo, results *Result) *JsonWriter {
	return &JsonWriter{
		info:        info,
		indent:      "",
		collPaths:   map[int]string{},
		unusedColls: map[int][]string{},
		results:     results,
		hasData:     false,
	}
}

func (jw *JsonWriter) Print(str string) {
	fmt.Fprint(jw.info, str)
	jw.hasData = true
}

func (jw *JsonWriter) Printf(format string, args ...any) {
	fmt.Fprintf(jw.info, format, args...)
	jw.hasData = true
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

func (jw *JsonWriter) NextEntity() (*Entity, error) {
	var err error
	jw.Entity, err = readNextEntity(jw.info.tx, jw.results)
	/*
		pc, _, line, _ := runtime.Caller(1)
		log.VPrintf(4, "Caller: %s:%d", path.Base(runtime.FuncForPC(pc).Name()), line)
		log.VPrintf(4, "  > Next: %v", jw.Entity)
	*/
	return jw.Entity, err
}

func (jw *JsonWriter) WriteCollectionHeader(extra string) (string, error) {
	myPlural := jw.Entity.Plural
	myURL := fmt.Sprintf("%s/%s", jw.info.BaseURL, path.Dir(jw.Entity.Path))

	jw.Printf("%s\n%s\"%surl\": %q,\n", extra, jw.indent, myPlural, myURL)
	extra = ""

	saveWriter := jw.info.HTTPWriter
	// saveExtra := extra

	// TODO optimize this to avoid the ioutil.Discard and just count the
	// children from the result set instead
	if !jw.info.ShouldInline(jw.Entity.Abstract) {
		jw.info.HTTPWriter = DefaultDiscardWriter
	}

	jw.Printf("%s%q: ", jw.indent, jw.Entity.Plural)
	count, err := jw.WriteCollection()
	if err != nil {
		return "", err
	}

	if jw.info.HTTPWriter == DefaultDiscardWriter {
		jw.info.HTTPWriter = saveWriter
		// extra = saveExtra
	} else {
		extra = ",\n"
	}

	jw.Printf("%s%s\"%scount\": %d", extra, jw.indent, myPlural, count)

	return ",", nil
}

func (jw *JsonWriter) WriteCollection() (int, error) {
	jw.Printf("{")
	jw.Indent()

	extra := ""
	myAbstract := "-"
	myPlural := ""
	count := 0

	for jw.Entity != nil {
		if myAbstract == "-" {
			myAbstract = jw.Entity.Abstract
			myPlural = jw.Entity.Plural
		}

		if strings.HasPrefix(jw.Entity.Abstract, myAbstract+string(DB_IN)) {
			// Process a child
			if _, err := jw.NextEntity(); err != nil {
				return count, err
			}
			continue
		}

		if strings.HasPrefix(myAbstract, jw.Entity.Abstract+string(DB_IN)) ||
			jw.Entity.Plural != myPlural {
			// Stop on a new parent or a new sibling collection
			break
		}

		jw.Printf("%s\n%s%q: ", extra, jw.indent, jw.Entity.UID)
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

	// Is this entity a Resource and does it have a meta.xref value?
	hasXref := false

	extra := "" // stuff to go at end of line during next print - eg: ,
	myType := jw.Entity.Type
	myAbstract := jw.Entity.Abstract
	addSpace := false // Add space before next attribute?

	if log.GetVerbose() > 3 {
		log.VPrintf(0, "eType: %d", myType)
		log.VPrintf(0, "JW:\n%s\n", ToJSON(jw))
		log.VPrintf(0, "JW.Obj:\n%s\n", ToJSON(jw.Entity.Object))
		log.VPrintf(0, "JW.NObj:\n%s\n", ToJSON(jw.Entity.NewObject))
	}

	jw.Printf("{")
	jw.Indent()

	jsonIt := func(e *Entity, info *RequestInfo, key string, val any, attr *Attribute) error {
		log.VPrintf(4, "jsonIt: %q", key)
		if key == "$space" {
			addSpace = true
			return nil
		}

		if key[0] == '#' {
			// Skip all other internal attributes
			return nil
		}

		if key == "$resource" {
			return SerializeResourceContents(jw, jw.Entity, jw.info, &extra)
		}

		if addSpace {
			jw.Printf("%s\n", extra)
			extra = ""
			addSpace = false
		}

		buf, _ := json.MarshalIndent(val, jw.indent, "  ")
		jw.Printf("%s\n%s%q: %s", extra, jw.indent, key, string(buf))
		extra = ","
		return nil
	}

	err := jw.Entity.SerializeProps(jw.info, jsonIt)
	if err != nil {
		panic(err)
	}

	// Now show all of the nested collections
	if extra != "" && myType != ENTITY_RESOURCE {
		// Resources already added the \n before "metaurl"
		extra += "\n" // just because it looks nicer with a blank line
	}

	jw.LoadCollections(myType) // load the list of current collections
	if _, err := jw.NextEntity(); err != nil {
		return err
	}

	// If next entity is 'meta' then skip it.
	// Note, we're getting lucky that "meta" comes before "versions".
	// We really should fix this.
	if jw.Entity != nil && jw.Entity.Type == ENTITY_META {
		hasXref = !IsNil(jw.Entity.Get("xref"))

		p, _ := PropPathFromPath(jw.Entity.Abstract)
		if jw.info.ShouldInline(p.DB()) {
			jw.Printf("%s\n%s%q: ", extra, jw.indent, "meta")
			if err := jw.WriteEntity(); err != nil {
				return err
			}
			extra = ","
			// We don't need to call "jw.NextEntity()" because the WriteEntity()
			// call above would have already done it for us.
		} else {
			// Skip "meta" entity
			if _, err = jw.NextEntity(); err != nil {
				return err
			}
		}
	}

	// Loop thru all of this entity's children
	for jw.Entity != nil &&
		(myAbstract == "" ||
			strings.HasPrefix(jw.Entity.Abstract, myAbstract+string(DB_IN))) {

		extra = jw.WritePreCollections(hasXref, extra, jw.Entity.Plural, myType)

		if extra, err = jw.WriteCollectionHeader(extra); err != nil {
			return err
		}
	}
	extra = jw.WritePostCollections(hasXref, extra, myType)

	// And finally done with this Entity
	jw.Outdent()
	jw.Printf("\n%s}", jw.indent)

	return nil
}

func SerializeResourceContents(jw *JsonWriter, e *Entity, info *RequestInfo, extra *string) error {
	PanicIf(e.Type != ENTITY_RESOURCE && e.Type != ENTITY_VERSION, "Bad eType: %d", e.Type)
	// Add the "resource*" props
	_, rm := jw.Entity.GetModels()
	singular := rm.Singular

	// If the #resource* props aren't there then just exit.
	// This will happen when "export/compact" is enabled because the
	// props won't show up in the Resorce but will on the default version
	// TODO really should do this check in entity.SerializeProps
	if IsNil(jw.Entity.Object["#resourceURL"]) &&
		IsNil(jw.Entity.Object["#resource"]) &&
		IsNil(jw.Entity.Object["#resourceProxyURL"]) {
		return nil
	}

	if url := jw.Entity.GetAsString("#resourceURL"); url != "" {
		jw.Printf("%s\n%s%q: %q", *extra, jw.indent, singular+"url", url)
		*extra = ","
	} else {
		p2, _ := PropPathFromDB(jw.Entity.Abstract)
		p := p2.P(singular).DB()
		if jw.info.ShouldInline(p) {
			data := []byte{}
			if val := jw.Entity.Get("#resource"); val != nil {
				var ok bool
				data, ok = val.([]byte)
				PanicIf(!ok, "Can't convert to []byte: %s", val)
			}

			if url := jw.Entity.GetAsString("#resourceProxyURL"); url != "" {
				resp, err := http.Get(url)
				if err != nil {
					data = []byte("GET error:" + err.Error())
				} else if resp.StatusCode/100 != 2 {
					data = []byte("GET error:" + resp.Status)
				} else {
					data, err = io.ReadAll(resp.Body)
					if err != nil {
						data = []byte("GET error:" + err.Error())
					}
				}
			}

			if len(data) > 0 {
				ct := jw.Entity.GetAsString("contenttype")
				ct = rm.MapContentType(ct)

				// Try to write the body in either JSON (the current
				// raw bytes stored in the DB), or if not valid JSON then
				// base64 encode it.
				if ct == "json" {
					if json.Valid(data) {
						// Only write the data as raw JSON (with indents)
						// if it doesn't start with quotes. For that case
						// since we need to escape the quotes we're going to
						// need to escape things, and in those cases
						// we just base64 encode it (the 'else' clause)
						pretty := bytes.Buffer{}
						err := json.Indent(&pretty, data, jw.indent, "  ")
						PanicIf(err != nil, "Bad JSON: %s", string(data))
						jw.Printf("%s\n%s%q: %s", *extra, jw.indent,
							singular, pretty.String())
					} else {
						// Write as escaped string
						ct = "string"
					}
				}

				if ct == "string" {
					// Write as escaped string
					buf, err := json.Marshal(string(data))
					PanicIf(err != nil, "Can't serialize: %s", string(data))
					jw.Printf("%s\n%s%q: %s", *extra, jw.indent,
						singular, string(buf))
				} else if ct == "binary" {
					str := base64.StdEncoding.EncodeToString(data)
					jw.Printf("%s\n%s\"%sbase64\": %q",
						*extra, jw.indent, singular, str)
				}
				*extra = ","
			}
		}
	}
	return nil
}

func (jw *JsonWriter) LoadCollections(eType int) {
	names := []string{}
	if eType == ENTITY_REGISTRY {
		if jw.info.Registry.Model != nil && jw.info.Registry.Model.Groups != nil {

			names = SortedKeys(jw.info.Registry.Model.Groups)
		}
	} else if eType == ENTITY_GROUP {
		gName, _ := strings.CutSuffix(jw.Entity.Abstract, IN_STR)
		names = SortedKeys(jw.info.Registry.Model.Groups[gName].Resources)
	} else if eType == ENTITY_RESOURCE {
		names = []string{"versions"}
	} else if eType == ENTITY_META {
		names = []string{}
	} else if eType == ENTITY_VERSION {
		names = []string{} // no children of versions
	} else {
		panic(fmt.Sprintf("Unknown eType: %d", eType))
	}
	jw.unusedColls[eType] = names

	p := jw.Entity.Path + "/"
	if p == "/" {
		p = ""
	}
	jw.collPaths[eType] = p
}

func (jw *JsonWriter) WritePreCollections(hasXref bool, extra string, plural string, eType int) string {
	for i, collName := range jw.unusedColls[eType] {
		if collName == plural {
			jw.unusedColls[eType] = jw.unusedColls[eType][i+1:]
			break
		}
		extra = jw.WriteEmptyCollection(hasXref, extra, eType, collName)
	}
	return extra
}

func (jw *JsonWriter) WritePostCollections(hasXref bool, extra string, eType int) string {
	for _, collName := range jw.unusedColls[eType] {
		extra = jw.WriteEmptyCollection(hasXref, extra, eType, collName)
	}

	delete(jw.collPaths, eType)
	delete(jw.unusedColls, eType)
	return extra
}

func (jw *JsonWriter) WriteEmptyCollection(hasXref bool, extra string, eType int, collName string) string {
	// If we're doing a Resource that has a meta.xref, skip "versions"
	if hasXref && collName == "versions" {
		return extra
	}

	p := Path2Abstract(jw.collPaths[eType] + collName)

	jw.Printf("%s\n%s\"%surl\": \"%s/%s%s\",\n", extra, jw.indent,
		collName, jw.info.BaseURL, jw.collPaths[eType], collName)

	if jw.info.ShouldInline(p) {
		jw.Printf("%s\"%s\": {},\n", jw.indent, collName)
	}

	jw.Printf("%s\"%scount\": 0", jw.indent, collName)
	extra = ","

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
