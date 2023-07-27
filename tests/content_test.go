package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestResourceContents(t *testing.T) {
	reg := NewRegistry("TestResourceContents")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir", "")
	gm.AddResourceModel("files", "file", 0, true, true)

	d1, _ := reg.AddGroup("dirs", "d1")
	f1, _ := d1.AddResource("files", "f1", "v1")

	f1.Set("name", "file1")
	f1.Set("labels.str1", "foo")
	f1.Set("labels.str2", "")
	f1.Set("labels.int", 6)
	f1.Set("labels.bool", true)
	f1.Set("labels.float", 123.456)
	f1.Set("str1", "foo")
	f1.Set("str2", "")
	f1.Set("int1", 6)
	f1.Set("int2", -5)
	f1.Set("int3", 0)
	f1.Set("bool1", true)
	f1.Set("bool2", false)
	f1.Set("float1", 123.456)
	f1.Set("float2", -456.876)
	f1.Set("float3", 0.0)

	f1.Set("#resource", "Hello there")

	f1.Refresh()
	xCheckEqual(t, "", NotNilString(f1.Get("#resource")), "Hello there")

	CompareContentMeta(t, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1",
		Body:    "Hello there",
		Headers: nil,
	})

	CompareContentMeta(t, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1/versions/v1",
		Body:    "Hello there",
		Headers: nil,
	})

	v2, _ := f1.AddVersion("v2")
	v2.Set("#resource", "This is version 2")

	CompareContentMeta(t, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1",
		Body:    "This is version 2",
		Headers: nil,
	})

	CompareContentMeta(t, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1/versions/v2",
		Body:    "This is version 2",
		Headers: nil,
	})

	v3, _ := f1.AddVersion("v3")
	v3.Set("#resourceProxyURL", "http://example.com")

	CompareContentMeta(t, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1",
		Body:    "*Example Domain", // contains
		Headers: []string{"Content-Type: text/html"},
	})

	CompareContentMeta(t, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1/versions/v3",
		Body:    "*Example Domain", // contains
		Headers: []string{"Content-Type: text/html"},
	})

	v4, _ := f1.AddVersion("v4")
	v4.Set("#resourceURL", "http://example.com")

	CompareContentMeta(t, &Test{
		Code: 303,
		URL:  "dirs/d1/files/f1",
		Body: "",
		Headers: []string{
			"Location: http://example.com",
		},
	})

	CompareContentMeta(t, &Test{
		Code: 303,
		URL:  "dirs/d1/files/f1/versions/v4",
		Body: "",
		Headers: []string{
			"Location: http://example.com",
		},
	})

}

type Test struct {
	Code    int
	URL     string
	Body    string
	Headers []string
}

func CompareContentMeta(t *testing.T, test *Test) {
	t.Logf("\n*** TESTING: URL: %s", test.URL)
	u := test.URL

	metaResp, err := http.Get("http://localhost:8181/" + u + "?meta")
	xNoErr(t, err)
	if metaResp == nil {
		return
	}
	metaBody, err := io.ReadAll(metaResp.Body)
	xNoErr(t, err)
	metaProps := map[string]any{}
	// t.Logf("MetaBody: %s", string(metaBody))
	err = json.Unmarshal(metaBody, &metaProps)
	xNoErr(t, err)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}

	res, err := client.Get("http://localhost:8181/" + u)
	xNoErr(t, err)
	xCheck(t, res.StatusCode == test.Code,
		fmt.Sprintf("\nTest: %s\nBad http code: %d", u, res.StatusCode))
	resBody, err := io.ReadAll(res.Body)
	xNoErr(t, err)

	// Make sure any headers have the expected text someplace
	for _, header := range test.Headers {
		name, value, _ := strings.Cut(header, ":")
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		h := res.Header.Get(name)
		xCheck(t, strings.Contains(h, value),
			fmt.Sprintf("Test %s\nHeader %q(%s) missing %q",
				u, name, h, value))
	}

	if test.Body == "" || test.Body[0] != '*' {
		xCheckEqual(t, "", string(resBody), test.Body)
	} else {
		if !strings.Contains(string(resBody), test.Body[1:]) {
			t.Errorf("Unexpected body for %q\nGot:\n%s\nExpected:\n%s",
				u, string(resBody), test.Body[1:])
		}
	}

	headerLabels := map[string]string{}

	for name, value := range res.Header {
		name = strings.ToLower(name)
		// t.Logf("Header: %s", name)
		if !strings.HasPrefix(name, "xregistry-") {
			continue
		}
		name = name[len("xregistry-"):]

		if strings.HasPrefix(name, "labels-") {
			headerLabels[name[7:]] = value[0]
			continue
		}

		foundIt := false
		for propName, propValue := range metaProps {
			if strings.ToLower(propName) != name {
				continue
			}

			delete(metaProps, propName)
			foundIt = true

			// TODO will need to sort the maps before diff'ing
			str := ""
			str = fmt.Sprintf("%v", propValue)
			// t.Logf("Checking %q: %q vs %q", name, str, value[0])
			xCheckEqual(t, "", value[0], str)
			break
		}
		if !foundIt {
			t.Errorf("Missing %q in ?meta version(%s)", name, u)
		}
	}

	var metaLabels map[string]interface{}
	if tmp := metaProps["labels"]; tmp != nil {
		metaLabels = tmp.(map[string]interface{})
	}

	for k, v := range headerLabels {
		metaVal, ok := metaLabels[k]
		if !ok {
			t.Errorf("metaLabel %v is missing: %s", k, u)
			continue
		}

		metaStr := fmt.Sprintf("%v", metaVal)
		if v != metaStr {
			t.Errorf("metaLabel %v value mismatch(%q vs %q): %s",
				k, v, metaStr, u)
		}
		delete(metaLabels, k)
	}
	if len(metaLabels) != 0 {
		t.Errorf("Extra metaLabels: %v for url: %q", metaLabels, u)
	}

	for propName, _ := range metaProps {
		if propName == "labels" {
			continue
		}
		t.Errorf("Extra prop %q in ?meata, not in header: %s", propName, u)
	}
}
