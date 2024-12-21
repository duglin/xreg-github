package registry

import (
	"fmt"
	log "github.com/duglin/dlog"
	// "reflect"
	"slices"
	"sort"
	"strings"
)

type Capabilities struct {
	Mutable         []string `json:"mutable"`
	Pagination      bool     `json:"pagination"`
	QueryParameters []string `json:"queryparameters"`
	Schemas         []string `json:"schemas"`
	ShortSelf       bool     `json:"shortself"`
	SpecVersions    []string `json:"specversions"`
}

var AllowableMutable = ArrayToLower([]string{
	"capabilities", "entities", "model"})

var AllowableQueryParameters = ArrayToLower([]string{
	"epoch", "export", "filter", "inline",
	"nested", "nodefaultversionid", "nodefaultversionsticky",
	"noepoch", "noreadonly", "setdefaultversionid"})

var AllowableSchemas = ArrayToLower([]string{"xRegistry-json"})

var AllowableSpecVersions = ArrayToLower([]string{"0.5"})

var DefaultCapabilities = &Capabilities{
	Mutable:         []string{"entities", "model"},
	Pagination:      false,
	QueryParameters: AllowableQueryParameters,
	Schemas:         AllowableSchemas,
	ShortSelf:       false,
	SpecVersions:    AllowableSpecVersions,
}

func init() {
	sort.Strings(AllowableMutable)
	sort.Strings(AllowableQueryParameters)
	sort.Strings(AllowableSchemas)
	sort.Strings(AllowableSpecVersions)

	Must(DefaultCapabilities.Validate())
}

func ArrayToLower(arr []string) []string {
	for i, s := range arr {
		arr[i] = strings.ToLower(s)
	}
	return arr
}

func CleanArray(arr []string, full []string, text string) ([]string, error) {
	// Make a copy so we can tweak it
	arr = slices.Clone(arr)

	// Lowercase evrything and look for "*"
	for i, s := range arr {
		s = strings.ToLower(s)
		arr[i] = s
		if s == "*" {
			if len(arr) != 1 {
				return nil, fmt.Errorf(`"*" must be the only value `+
					`specified for %q`, text)
			}
			return full, nil
		}
	}

	sort.Strings(arr)         // sort 'em
	arr = slices.Compact(arr) // remove dups
	if len(arr) == 0 {
		arr = []string{}
	}

	// Now look for valid values
	ai, fi := len(arr)-1, len(full)-1
	for ai >= 0 && fi >= 0 {
		as, fs := arr[ai], full[fi]
		if as == fs {
			ai--
		} else if as > fs {
			return nil, fmt.Errorf(`Unknown %q value: %q`, text, as)
		}
		fi--
	}
	if ai < 0 {
		return arr, nil
	}
	return nil, fmt.Errorf(`Unknown %q value: %q`, text, arr[ai])
}

func (c *Capabilities) Validate() error {
	var err error

	if c.Schemas == nil {
		c.Schemas = []string{XREGSCHEMA}
	}
	if c.SpecVersions == nil {
		c.SpecVersions = []string{SPECVERSION}
	}

	c.Mutable, err = CleanArray(c.Mutable, AllowableMutable, "mutable")
	if err != nil {
		return err
	}

	c.QueryParameters, err = CleanArray(c.QueryParameters,
		AllowableQueryParameters, "queryparameter")
	if err != nil {
		return err
	}

	c.Schemas, err = CleanArray(c.Schemas, AllowableSchemas, "schemas")
	if err != nil {
		return err
	}

	c.SpecVersions, err = CleanArray(c.SpecVersions, AllowableSpecVersions,
		"specversions")
	if err != nil {
		return err
	}

	if !ArrayContains(c.Schemas, strings.ToLower(XREGSCHEMA)) {
		return fmt.Errorf(`"schemas" must contain %q`, XREGSCHEMA)
	}
	if !ArrayContains(c.SpecVersions, strings.ToLower(SPECVERSION)) {
		return fmt.Errorf(`"specversions" must contain %q`, SPECVERSION)
	}

	return nil
}

func ParseCapabilitiesJSON(buf []byte) (*Capabilities, error) {
	log.VPrintf(4, "Enter: ParseCapabilitiesJSON")
	cap := Capabilities{}

	err := Unmarshal(buf, &cap)
	if err != nil {
		if strings.HasPrefix(err.Error(), "unknown field ") {
			return nil, fmt.Errorf("Unknown capability: %s",
				err.Error()[14:])
		}
		return nil, err
	}
	return &cap, nil

	/*
	   tmpmap := map[string]any{}

	   // check generial json syntax first

	   	if err := Unmarshal(buf, &tmpmap); err != nil {
	   		return nil, err
	   	}

	   // Look for unknown keys
	   capValue := reflect.ValueOf(cap)

	   	for key, _ := range tmpmap {
	   		fieldVal := capValue.FieldByNameFunc(func(name string) bool {
	   			return strings.ToLower(name) == key
	   		})
	   		if !fieldVal.IsValid() {
	   			return nil, fmt.Errorf("Unknown capability: %s", key)
	   		}
	   	}

	   // Now parse into 'cap'

	   	if err := Unmarshal(buf, &cap); err != nil {
	   		return nil, err
	   	}

	   return &cap, nil
	*/
}

// Mutable         []string `json:"mutable,omitempty"`
// Pagination      *bool    `json:"pagination,omitempty"`
// QueryParameters []string `json:"queryparameters,omitempty"`
// Schemas         []string `json:"schemas,omitempty"`
// ShortSelf       *bool    `json:"shortself,omitempty"`
// SpecVersions    []string `json:"specversions,omitempty"`

func (c *Capabilities) MutableEnabled(str string) bool {
	return ArrayContains(c.Mutable, strings.ToLower(str))
}

func (c *Capabilities) PaginationEnabled() bool {
	return c.Pagination
}

func (c *Capabilities) QueryParameterEnabled(str string) bool {
	return ArrayContains(c.QueryParameters, strings.ToLower(str))
}

func (c *Capabilities) SchemaEnabled(str string) bool {
	return ArrayContains(c.Schemas, strings.ToLower(str))
}

func (c *Capabilities) ShortSelfEnabled(str string) bool {
	return c.ShortSelf
}

func (c *Capabilities) SpecVersionEnabled(str string) bool {
	return ArrayContains(c.SpecVersions, strings.ToLower(str))
}
