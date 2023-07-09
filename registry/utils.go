package registry

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
	"github.com/google/uuid"
)

func NewUUID() string {
	return uuid.NewString()[:8]
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

	b := (*val).([]byte)
	return string(b)
}

func NotNilInt(val *any) int {
	if val == nil || *val == nil {
		return 0
	}

	b := (*val).(int64)
	return int(b)
}

func NotNilBool(val *any) bool {
	if val == nil || *val == nil {
		return false
	}

	return ((*val).(int64)) == 1
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

func URLBuild(base string, paths ...string) string {
	isFrag := strings.Index(base, "#") >= 0
	url := base
	url = strings.TrimRight(url, "/")

	for _, path := range paths {
		if isFrag {
			url += "/" + path
		} else {
			url += "/" + strings.ToLower(path)
		}
	}
	return url
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

func SetField(res interface{}, name string, value *string, propType string) {
	k, _ := strconv.Atoi(propType)
	v := fmt.Sprintf("%v", value)
	if value != nil {
		v = fmt.Sprintf("%v", *value)
	}

	log.VPrintf(3, ">Enter: SetField(%T, %s/%q,%s)", res, name, v,
		reflect.Kind(k).String())
	defer log.VPrintf(3, "<")

	var err error
	val := reflect.ValueOf(res).Elem()

	field := val.FieldByName(name)
	// Use "Extensions" for invalid fields (meaning not defined in the resource)
	if !field.IsValid() {
		field := reflect.ValueOf(res).Elem().FieldByName("Extensions")
		if !field.IsValid() {
			log.VPrintf(2, "Can't Set unknown field(%T/%s)", res, name)
		} else {
			if field.IsNil() {
				field.Set(reflect.ValueOf(map[string]any{}))
			}
			newValue := reflect.Value{}

			if reflect.Kind(k) == reflect.Int {
				tmpInt, _ := strconv.Atoi(*value)
				newValue = reflect.ValueOf(tmpInt)
			} else {
				newValue = reflect.ValueOf(*value)
			}
			field.SetMapIndex(reflect.ValueOf(name), newValue)
		}
	} else {
		if field.Type().Kind() == reflect.String {
			tmpVal := ""
			if value != nil {
				tmpVal = *value
			}

			field.SetString(tmpVal)
			log.VPrintf(4, "set %q to %q", name, reflect.ValueOf(value).Elem())
		} else if field.Type().Kind() == reflect.Int {
			tmpInt := 0
			if value != nil {
				tmpInt, err = strconv.Atoi(*value)
				if err != nil {
					log.Printf("Error converting %q int int: %s", *value, err)
				}
			}
			field.SetInt(int64(tmpInt)) // reflect.ValueOf(value).Elem())
			log.VPrintf(4, "set %q to %d", name, tmpInt)
		}
	}
}

type JSONData struct {
	Prefix   string
	Indent   string
	Registry *Registry
}
