package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestResourceContents(t *testing.T) {
	reg := NewRegistry("TestResourceContents")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, true)
	rm.AddAttr("str1", registry.STRING)
	rm.AddAttr("str2", registry.STRING)
	rm.AddAttr("int1", registry.INTEGER)
	rm.AddAttr("int2", registry.INTEGER)
	rm.AddAttr("int3", registry.INTEGER)
	rm.AddAttr("bool1", registry.BOOLEAN)
	rm.AddAttr("bool2", registry.BOOLEAN)
	rm.AddAttr("dec1", registry.DECIMAL)
	rm.AddAttr("dec2", registry.DECIMAL)
	rm.AddAttr("dec3", registry.DECIMAL)

	d1, err := reg.AddGroup("dirs", "d1")
	xNoErr(t, err)
	f1, err := d1.AddResource("files", "f1", "v1")
	xNoErr(t, err)

	f1.SetSave("name", "file1")
	f1.SetSave("labels.str1", "foo")
	f1.SetSave("labels.str2", "")
	// f1.SetSave("labels.int", 6)
	// f1.SetSave("labels.bool", true)
	// f1.SetSave("labels.decimal", 123.456)
	f1.SetSave("str1", "foo")
	f1.SetSave("str2", "")
	f1.SetSave("int1", 6)
	f1.SetSave("int2", -5)
	f1.SetSave("int3", 0)
	f1.SetSave("bool1", true)
	f1.SetSave("bool2", false)
	f1.SetSave("dec1", 123.456)
	f1.SetSave("dec2", -456.876)
	f1.SetSave("dec3", 0.0)

	f1.SetSave("#resource", "Hello there")

	f1.Refresh()
	xCheckEqual(t, "", NotNilString(f1.Get("#resource")), "Hello there")

	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1",
		Body:    "Hello there",
		Headers: nil,
	})

	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1/versions/v1",
		Body:    "Hello there",
		Headers: nil,
	})

	v2, _ := f1.AddVersion("v2")
	v2.SetSave("#resource", "This is version 2")

	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1",
		Body:    "This is version 2",
		Headers: nil,
	})

	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1/versions/v2",
		Body:    "This is version 2",
		Headers: nil,
	})

	v3, _ := f1.AddVersion("v3")
	v3.SetSave("#resourceProxyURL", "http://example.com")

	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1",
		Body:    "*Example Domain", // contains
		Headers: []string{"Content-Type: text/html"},
	})

	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1/versions/v3",
		Body:    "*Example Domain", // contains
		Headers: []string{"Content-Type: text/html"},
	})

	v4, _ := f1.AddVersion("v4")
	v4.SetSave("#resourceURL", "http://example.com")

	CompareContentMeta(t, reg, &Test{
		Code: 303,
		URL:  "dirs/d1/files/f1",
		Body: "",
		Headers: []string{
			"xRegistry-fileurl: http://example.com",
			"Location: http://example.com",
		},
	})

	CompareContentMeta(t, reg, &Test{
		Code: 303,
		URL:  "dirs/d1/files/f1/versions/v4",
		Body: "",
		Headers: []string{
			"xRegistry-fileurl: http://example.com",
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

func CompareContentMeta(t *testing.T, reg *registry.Registry, test *Test) {
	t.Helper()
	xNoErr(t, reg.Commit())

	u := test.URL

	t.Logf("Testing: URL: %s", test.URL)
	metaResp, err := http.Get("http://localhost:8181/" + u + "?meta")
	xNoErr(t, err)
	if metaResp == nil {
		t.Fatalf("metaResp is nil")
	}
	metaBody, err := io.ReadAll(metaResp.Body)
	xNoErr(t, err)
	if metaResp.StatusCode/100 != 2 {
		t.Fatalf("Bad response: %s\n%s", metaResp.Status, metaBody)
	}
	metaProps := map[string]any{}
	err = json.Unmarshal(metaBody, &metaProps)
	if err != nil {
		t.Fatalf("Err: %s\n Body:\n%s", err, string(metaBody))
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}

	res, err := client.Get("http://localhost:8181/" + u)
	xNoErr(t, err)

	resBody, err := io.ReadAll(res.Body)
	xNoErr(t, err)

	xCheck(t, res.StatusCode == test.Code,
		fmt.Sprintf("\nTest: %s\nBad http code: %d should be %d\n%s", u,
			res.StatusCode, test.Code, string(resBody)))

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
			t.Fatalf("Unexpected body for %q\nGot:\n%s\nExpected:\n%s",
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
			if name == "self" || name == "defaultversionurl" {
				xCheckEqual(t, "", value[0]+"?meta", str)
				break
			}

			xCheckEqual(t, "", value[0], str)
			break
		}
		if !foundIt {
			t.Fatalf("Missing %q in ?meta version(%s)", name, u)
		}
	}

	var metaLabels map[string]interface{}
	if tmp := metaProps["labels"]; tmp != nil {
		metaLabels = tmp.(map[string]interface{})
	}

	for k, v := range headerLabels {
		metaVal, ok := metaLabels[k]
		if !ok {
			t.Fatalf("metaLabel %v is missing: %s", k, u)
			continue
		}

		metaStr := fmt.Sprintf("%v", metaVal)
		if v != metaStr {
			t.Fatalf("metaLabel %v value mismatch(%q vs %q): %s",
				k, v, metaStr, u)
		}
		delete(metaLabels, k)
	}
	if len(metaLabels) != 0 {
		t.Fatalf("Extra metaLabels: %v for url: %q", metaLabels, u)
	}

	for propName, _ := range metaProps {
		if propName == "labels" {
			continue
		}
		t.Fatalf("Extra prop %q in ?meta, not in header: %s", propName, u)
	}
}
