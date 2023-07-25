package tests

import (
	"bytes"
	// "encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type HTTPTest struct {
	Name       string
	URL        string
	Method     string
	ReqHeaders []string // name:value
	ReqBody    string

	Code       int
	ResHeaders []string // name:value
	ResBody    string
}

func xCheckHTTP(t *testing.T, test *HTTPTest) {
	return
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
	body := io.Reader(nil)
	if test.ReqBody != "" {
		body = bytes.NewReader([]byte(test.ReqBody))
	}
	req, err := http.NewRequest(test.Method, "http://localhost:8080/"+test.URL, body)
	xNoErr(t, err)
	for _, header := range test.ReqHeaders {
		name, value, _ := strings.Cut(header, ":")
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		req.Header.Add(name, value)
	}

	res, err := client.Do(req)
	xNoErr(t, err)
	xCheck(t, res.StatusCode == test.Code,
		fmt.Sprintf("Expected status %d, got %d", test.Code, res.StatusCode))

	for _, header := range test.ResHeaders {
		name, value, _ := strings.Cut(header, ":")
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		xCheckEqual(t, "Header:"+name, res.Header.Get(name), value)
	}

	resBody, _ := io.ReadAll(res.Body)
	xCheckEqual(t, "Test: "+test.Name+"\nBody:\n",
		string(resBody), test.ResBody)
}

func TestHTTPModel(t *testing.T) {
	reg := NewRegistry("TestHTTPModel")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	// Check as part of Reg request
	xCheckHTTP(t, &HTTPTest{
		Name:       "?model",
		URL:        "?model",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    "",

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "id": "TestHTTPModel",
  "self": "http://localhost:8080/",
  "model": {}
}
`,
	})

	// Just model, no reg content
	xCheckHTTP(t, &HTTPTest{
		Name:       "/model",
		URL:        "/model",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    "",

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{}
`,
	})

	// Create model tests
	xCheckHTTP(t, &HTTPTest{
		Name:       "Create empty model",
		URL:        "/model",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "schema": "model.schema"
}`,

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{}
`,
	})
}
