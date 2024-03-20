package registry

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	// "net/url"
	"io"
	// "maps"
	"net/http"
	"os"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
	"github.com/google/uuid"
)

func NewUUID() string {
	return uuid.NewString()[:8]
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func PanicIf(b bool, msg string, args ...any) {
	if b {
		Panicf(msg, args...)
	}
}
func Panicf(msg string, args ...any) {
	panic(fmt.Sprintf(msg, args...))
}

func init() {
	if !IsNil(nil) {
		panic("help me1")
	}
	if !IsNil(any(nil)) {
		panic("help me2")
	}
	if !IsNil((*any)(nil)) {
		panic("help me3")
	}
}

func IsNil(a any) bool {
	val := reflect.ValueOf(a)
	if !val.IsValid() {
		return true
	}
	switch val.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map,
		reflect.Func, reflect.Interface:

		return val.IsNil()
	}
	return false
}

func NotNilString(val *any) string {
	if val == nil || *val == nil {
		return ""
	}

	if reflect.ValueOf(*val).Kind() == reflect.String {
		return (*val).(string)
	}

	if reflect.ValueOf(*val).Kind() != reflect.Slice {
		panic("Not a slice")
	}

	b := (*val).([]byte)
	return string(b)
}

func NotNilIntDef(val *any, def int) int {
	if val == nil || *val == nil {
		return def
	}

	var b int

	if reflect.ValueOf(*val).Kind() == reflect.Int64 {
		tmp, _ := (*val).(int64)
		b = int(tmp)
	} else {
		b, _ = (*val).(int)
	}

	return b
}

func NotNilInt(val *any) int {
	return NotNilIntDef(val, 0)
}

func PtrIntDef(val *any, def int) *int {
	result := NotNilIntDef(val, def)
	return &result
}

func NotNilBoolDef(val *any, def bool) bool {
	if val == nil || *val == nil {
		return def
	}

	return ((*val).(int64)) == 1
}

func PtrBool(b bool) *bool {
	return &b
}

func PtrBoolDef(val *any, def bool) *bool {
	result := NotNilBoolDef(val, def)
	return &result
}

func JSONEscape(obj interface{}) string {
	buf, _ := json.Marshal(obj)
	return string(buf[1 : len(buf)-1])
}

func ToJSON(obj interface{}) string {
	buf, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		log.Fatalf("Error Marshaling: %s", err)
	}
	return string(buf)
}

func ToJSONOneLine(obj interface{}) string {
	buf, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		log.Fatalf("Error Marshaling: %s", err)
	}

	re := regexp.MustCompile(`[\s\r\n]*`)
	buf = re.ReplaceAll(buf, []byte(""))

	return string(buf)
}

func Keys(m interface{}) []string {
	mk := reflect.ValueOf(m).MapKeys()

	keys := make([]string, 0, len(mk))
	for _, k := range mk {
		keys = append(keys, k.String())
	}
	return keys
}

func SortedKeys(m interface{}) []string {
	mk := reflect.ValueOf(m).MapKeys()

	keys := make([]string, 0, len(mk))
	for _, k := range mk {
		keys = append(keys, k.String())
	}
	sort.Strings(keys)
	return keys
}

func GetStack() []string {
	stack := []string{}

	for i := 1; i < 20; i++ {
		pc, file, line, _ := runtime.Caller(i)
		if line == 0 {
			break
		}
		stack = append(stack,
			fmt.Sprintf("%s %s:%d",
				path.Base(runtime.FuncForPC(pc).Name()), path.Base(file), line))
		if strings.Contains(file, "main") || strings.Contains(file, "testing") {
			break
		}
	}
	return stack
}

func ShowStack() {
	log.VPrintf(0, "-----")
	for i := 1; i < 20; i++ {
		pc, file, line, _ := runtime.Caller(i)
		if line == 0 {
			break
		}
		log.VPrintf(0, "Caller: %s:%d",
			path.Base(runtime.FuncForPC(pc).Name()), line)
		if strings.Contains(file, "main") || strings.Contains(file, "testing") {
			break
		}
	}
}

func OneLine(buf []byte) []byte {
	buf = RemoveProps(buf)

	re := regexp.MustCompile(`[\r\n]*`)
	buf = re.ReplaceAll(buf, []byte(""))
	re = regexp.MustCompile(`([^a-zA-Z])\s+([^a-zA-Z])`)
	buf = re.ReplaceAll(buf, []byte(`$1$2`))
	re = regexp.MustCompile(`([^a-zA-Z])\s+([^a-zA-Z])`)
	buf = re.ReplaceAll(buf, []byte(`$1$2`))

	return buf
}

func RemoveProps(buf []byte) []byte {
	re := regexp.MustCompile(`\n[^{}]*\n`)
	buf = re.ReplaceAll(buf, []byte("\n"))

	// re = regexp.MustCompile(`\s"labels": {\s*},*`)
	// buf = re.ReplaceAll(buf, []byte(""))

	re = regexp.MustCompile(`\n *\n`)
	buf = re.ReplaceAll(buf, []byte("\n"))

	re = regexp.MustCompile(`\n *}\n`)
	buf = re.ReplaceAll(buf, []byte("}\n"))

	re = regexp.MustCompile(`}[\s,]+}`)
	buf = re.ReplaceAll(buf, []byte("}}"))
	buf = re.ReplaceAll(buf, []byte("}}"))

	return buf
}

func HTMLify(r *http.Request, buf []byte) []byte {
	str := fmt.Sprintf(`"(https?://[^"\n]*?)"`)
	re := regexp.MustCompile(str)
	repl := fmt.Sprintf(`"<a href="$1?%s">$1?%s</a>"`,
		r.URL.RawQuery, r.URL.RawQuery)

	return re.ReplaceAll(buf, []byte(repl))
}

func RegHTMLify(r *http.Request, buf []byte) []byte {
	str := fmt.Sprintf(`"(https?://[^?"\n]*)(\??)([^"\n]*)"`)
	re := regexp.MustCompile(str)
	repl := fmt.Sprintf(`"<a href='$1?reg&$3'>$1$2$3</a>"`)

	return re.ReplaceAll(buf, []byte(repl))
}

func AnyToUInt(val any) (int, error) {
	var err error

	kind := reflect.ValueOf(val).Kind()
	resInt := 0
	if kind == reflect.Float64 { // JSON ints show up as floats
		resInt = int(val.(float64))
		if float64(resInt) != val.(float64) {
			err = fmt.Errorf("must be a uinteger")
		}
	} else if kind != reflect.Int {
		err = fmt.Errorf("must be a uinteger")
	} else {
		resInt = val.(int)
	}

	if err == nil && resInt < 0 {
		err = fmt.Errorf("must be a uinteger")
	}

	return resInt, err
}

func LineNum(buf []byte, pos int) int {
	return bytes.Count(buf[:pos], []byte("\n")) + 1
}

func Unmarshal(buf []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(buf))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		msg := err.Error()

		if jerr, ok := err.(*json.UnmarshalTypeError); ok {
			msg = fmt.Sprintf("Can't parse %q as a(n) %q at line %d",
				jerr.Value, jerr.Type.String(),
				LineNum(buf, int(jerr.Offset)))
		} else if jerr, ok := err.(*json.SyntaxError); ok {
			msg = fmt.Sprintf("Syntax error at line %d: %s",
				LineNum(buf, int(jerr.Offset)), msg)
		}
		msg, _ = strings.CutPrefix(msg, "json: ")
		return errors.New(msg)
	}
	return nil
}

// var re = regexp.MustCompile(`(?m:([^#]*)#[^"]*$)`)
var removeCommentsRE = regexp.MustCompile(`(gm:^(([^"#]|"[^"]*")*)#.*$)`)

func RemoveComments(buf []byte) []byte {
	return removeCommentsRE.ReplaceAll(buf, []byte("${1}"))
}

type ImportArgs struct {
	// Cache path/name of "" means stdin
	Cache      map[string]map[string]any // Path#.. -> json
	History    []string                  // Just names, no frag, [0]=latest
	LocalFiles bool                      // ok to access local FS files?
}

func ProcessImports(file string, buf []byte, localFiles bool) ([]byte, error) {
	data := map[string]any{}

	buf = RemoveComments(buf)

	if err := Unmarshal(buf, &data); err != nil {
		return nil, fmt.Errorf("Error parsing JSON: %s", err)
	}

	importArgs := ImportArgs{
		Cache: map[string]map[string]any{
			file: data,
		},
		History:    []string{file}, // stack of base names
		LocalFiles: localFiles,
	}

	if err := ImportTraverse(importArgs, data); err != nil {
		return nil, err
	}

	// Convert back to byte
	buf, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Error generating JSON: %s", err)
	}

	return buf, nil
}

// data is the current map to check for $import statements
func ImportTraverse(importArgs ImportArgs, data map[string]any) error {
	var err error
	currFile, _ := SplitFragement(importArgs.History[0]) // Grab just base name

	// log.Printf("ImportTraverse:")
	// log.Printf("  Cache: %v", SortedKeys(importArgs.Cache))
	// log.Printf("  History: %v", importArgs.History)
	// log.Printf("  Recurse:")
	// log.Printf("    Data keys: %v", SortedKeys(data))

	_, ok1 := data["$import"]
	_, ok2 := data["$imports"]
	if ok1 && ok2 {
		return fmt.Errorf("In %q, both $import and $imports is not allowed",
			currFile)
	}

	dataKeys := Keys(data) // so we can add/delete keys
	for _, key := range dataKeys {
		val := data[key]
		if key == "$import" || key == "$imports" {
			delete(data, key)
			list := []string{}

			valValue := reflect.ValueOf(val)
			if key == "$import" {
				if valValue.Kind() != reflect.String {
					return fmt.Errorf("In %q, $import isn't a string", currFile)
				}
				list = []string{val.(string)}
			} else {
				if valValue.Kind() != reflect.Slice {
					return fmt.Errorf("In %q, $imports isn't an array",
						currFile)
				}

				for i := 0; i < valValue.Len(); i++ {
					impInt := valValue.Index(i).Interface()
					imp, ok := impInt.(string)
					if !ok {
						return fmt.Errorf("In %q, $imports contains a "+
							"non-string value (%v)", currFile, impInt)
					}
					list = append(list, imp)
				}
			}

			for _, impStr := range list {
				for _, name := range importArgs.History {
					if name == impStr {
						return fmt.Errorf("Recursive on %q", name)
					}
				}

				if len(impStr) == 0 {
					return fmt.Errorf("In %q, $import can't be an empty string",
						currFile)
				}

				// log.Printf("CurrFile: %s\nImpStr: %s", currFile, impStr)
				nextFile := ResolvePath(currFile, impStr)
				// log.Printf("NextFile: %s", nextFile)
				importData := importArgs.Cache[nextFile]
				base, fragment := SplitFragement(nextFile)

				if importData == nil {
					importData = importArgs.Cache[base]
					if importData == nil {
						data := []byte(nil)
						if strings.HasPrefix(base, "http") {
							res, err := http.Get(base)
							if err != nil {
								return err
							}
							if res.StatusCode != 200 {
								return fmt.Errorf("Error getting %q: %s",
									base, res.Status)
							}
							data, err = io.ReadAll(res.Body)
							res.Body.Close()
							if err != nil {
								return err
							}
						} else {
							if importArgs.LocalFiles {
								if data, err = os.ReadFile(base); err != nil {
									return fmt.Errorf("Error reading file %q: %s",
										base, err)
								}
							} else {
								return fmt.Errorf("Not allowed to access file: %s",
									base)
							}
						}
						data = RemoveComments(data)

						if err := Unmarshal(data, &importData); err != nil {
							return err
						}
						importArgs.Cache[base] = importData
					}

					// Now, traverse down to the specific field - if needed
					if fragment != "" {
						nextTop := importArgs.Cache[base]
						impData, err := GetJSONPointer(nextTop, fragment)
						if err != nil {
							return err
						}

						if reflect.ValueOf(impData).Kind() != reflect.Map {
							return fmt.Errorf("In %q, $import(%s) is not a map: %s",
								currFile, impStr, reflect.ValueOf(importData).Kind())
						}

						importData = impData.(map[string]any)
						importArgs.Cache[nextFile] = importData
					}
				}

				// Go deep! (recurse) before we add it to current map
				importArgs.History = append([]string{nextFile},
					importArgs.History...)
				if err = ImportTraverse(importArgs, importData); err != nil {
					return err
				}
				importArgs.History = importArgs.History[1:]

				// Only copy if we don't already have one by this name
				for k, v := range importData {
					if _, ok := data[k]; !ok {
						data[k] = v
					}
				}
			}
		} else {
			if reflect.ValueOf(val).Kind() == reflect.Map {
				nextLevel := val.(map[string]any)
				if err = ImportTraverse(importArgs, nextLevel); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func SplitFragement(str string) (string, string) {
	parts := strings.SplitN(str, "#", 2)

	if len(parts) != 2 {
		return parts[0], ""
	} else {
		return parts[0], parts[1]
	}
}

var dotdotRE = regexp.MustCompile(`(^|/)[^/]*/\.\.(/|$)`) // removes /../
var slashesRE = regexp.MustCompile(`([^:])//+`)           // : is for URL's ://
var urlPrefixRE = regexp.MustCompile(`^https?://`)
var justHostRE = regexp.MustCompile(`^https?://[^/]*$`) // no path?
var extractHostRE = regexp.MustCompile(`^(https?://[^/]*/).*`)
var endingDots = regexp.MustCompile(`(/\.\.?)$`) // ends with . or ..

func ResolvePath(baseFile string, next string) string {
	baseFile, _ = SplitFragement(baseFile)
	baseFile = endingDots.ReplaceAllString(baseFile, "$1/")

	if next == "" {
		return baseFile
	}
	if next[0] == '#' {
		return baseFile + next
	}

	// Abs URLs
	if strings.HasPrefix(next, "http:") || strings.HasPrefix(next, "https:") {
		return next
	}

	// baseFile is a URL
	if urlPrefixRE.MatchString(baseFile) {
		if justHostRE.MatchString(baseFile) {
			baseFile += "/"
		}

		if next != "" && next[0] == '/' {
			baseFile = extractHostRE.ReplaceAllString(baseFile, "$1")
		}

		if baseFile[len(baseFile)-1] == '/' { // ends with /
			next = baseFile + next
		} else {
			i := strings.LastIndex(baseFile, "/") // remove last word
			if i >= 0 {
				baseFile = baseFile[:i+1] // keep last /
			}
			next = baseFile + next
		}
	} else {
		// Look for abs path for files
		if len(next) > 0 && next[0] == '/' {
			return next
		}

		if len(next) > 2 && next[1] == ':' {
			// Windows abs path ?
			ch := next[0]
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
				return next
			}
		}

		baseFile = path.Dir(baseFile) // remove file name
		next = path.Join(baseFile, next)
	}

	// log.Printf("Before clean: %q", next)
	next, _ = strings.CutPrefix(next, "./")        // remove leading ./
	next = slashesRE.ReplaceAllString(next, "$1/") // squash //'s
	next = strings.ReplaceAll(next, "/./", "/")    // remove pointless /./
	next = dotdotRE.ReplaceAllString(next, "/")    // remove ../'s
	return next
}

var JPtrEsc0 = regexp.MustCompile(`~0`)
var JPtrEsc1 = regexp.MustCompile(`~1`)

func GetJSONPointer(data any, path string) (any, error) {
	// log.Printf("GPtr: path: %q\nData: %s", path, ToJSON(data))
	path = strings.TrimSpace(path)
	if path == "" {
		return data, nil
	}

	if IsNil(data) {
		return nil, nil
	}

	path, _ = strings.CutPrefix(path, "/")
	parts := strings.Split(path, "/")
	// log.Printf("Parts: %q", strings.Join(parts, "|"))

	for i, part := range parts {
		part = JPtrEsc1.ReplaceAllString(part, `/`)
		part = JPtrEsc0.ReplaceAllString(part, `~`)
		// log.Printf("  Part: %s", part)

		dataVal := reflect.ValueOf(data)
		kind := dataVal.Kind()
		if kind == reflect.Map {
			dataVal = dataVal.MapIndex(reflect.ValueOf(part))
			// log.Printf("dataVal: %#v", dataVal)
			if !dataVal.IsValid() {
				return nil, fmt.Errorf("Attribute %q not found",
					strings.Join(parts[:i+1], "/"))
			}
			data = dataVal.Interface()
			continue
		} else if kind == reflect.Slice {
			j, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("Index %q must be an integer",
					"/"+strings.Join(parts[:i+1], "/"))
			}
			if j < 0 || j >= dataVal.Len() { // len(daSlice) {
				return nil, fmt.Errorf("Index %q is out of bounds(0-%d)",
					"/"+strings.Join(parts[:i+1], "/"), dataVal.Len()-1)
			}
			data = dataVal.Index(j).Interface()
			continue
		} else {
			return nil, fmt.Errorf("Can't step into a type of %q, at: %s",
				kind, "/"+strings.Join(parts[:i+1], "/"))
		}
	}

	return data, nil
}
