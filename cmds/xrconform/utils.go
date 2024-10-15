package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
)

func GetStack() []string {
	stack := []string{}

	for i := 1; i < 20; i++ {
		pc, file, line, _ := runtime.Caller(i)
		if line == 0 {
			break
		}
		stack = append(stack,
			fmt.Sprintf("%s - %s:%d",
				path.Base(runtime.FuncForPC(pc).Name()), path.Base(file), line))
		if strings.Contains(file, "main") || strings.Contains(file, "testing") {
			break
		}
	}
	return stack
}

func ShowStack() {
	stack := GetStack()
	fmt.Println("----- Stack")
	for _, line := range stack {
		fmt.Println(line)
	}
}

func ToJSON(obj interface{}) string {
	buf, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("Error Marshaling: %s", err))
	}
	return string(buf)
}

type XRegistry struct {
	// Config values:
	// server.url: VALUE
	// header.NAME: VALUE
	Config map[string]string
}

func (xr *XRegistry) GetServerURL() string {
	return xr.GetConfig("server.url")
}

// File syntax:
// prop: value
// # comment
func (xr *XRegistry) LoadConfigFromFile(filename string) error {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return xr.LoadConfigFromBuffer(string(buf))
}

// Buffer syntax:
// prop: value
// # comment
func (xr *XRegistry) LoadConfigFromBuffer(buffer string) error {
	lines := strings.Split(buffer, "/n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		name, value, _ := strings.Cut(line, ":")
		if name == "" {
			return fmt.Errorf("Error in config data - no name: %q", line)
		}
		xr.SetConfig(name, value)
	}
	return nil
}

func (xr *XRegistry) GetConfig(name string) string {
	if xr.Config == nil {
		return ""
	}
	return xr.Config[name]
}

func (xr *XRegistry) SetConfig(name string, value string) error {
	name = strings.TrimSpace(name)
	value = strings.TrimSpace(value)

	if name == "" {
		return fmt.Errorf("Config name can't be blank")
	}
	if value == "" {
		delete(xr.Config, name)
	} else {
		xr.Config[name] = value
	}
	return nil
}

type HTTPResponse struct {
	Error      error
	StatusCode int
	Headers    http.Header
	Body       []byte
	JSON       map[string]any
}

// HTTPResponse
// golang error if things failed at the tranport level
func (xr *XRegistry) Curl(verb string, path string, body string) *HTTPResponse {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
	bodyReader := io.Reader(nil)
	if body != "" {
		bodyReader = bytes.NewReader([]byte(body))
	}

	req, err := http.NewRequest(verb, xr.GetServerURL()+"/"+path, bodyReader)
	if err != nil {
		return &HTTPResponse{Error: err}
	}

	for key, value := range xr.Config {
		key = strings.TrimSpace(key[7:])
		if !strings.HasSuffix(key, "header.") {
			continue
		}
		key = strings.TrimSpace(key[7:])
		if key == "" {
			continue
		}
		value = strings.TrimSpace(value)
		req.Header.Add(key, value) // ok even if value is ""
	}

	doRes, err := client.Do(req)
	if err != nil || doRes == nil {
		return &HTTPResponse{Error: err}
	}

	res := &HTTPResponse{
		StatusCode: doRes.StatusCode,
		Headers:    doRes.Header.Clone(),
	}
	res.Body, err = io.ReadAll(doRes.Body)
	if err != nil {
		return &HTTPResponse{Error: err}
	}

	if len(res.Body) > 0 {
		// Ignore any error, just assume it's not JSON and keep going
		json.Unmarshal(res.Body, &res.JSON)
	}

	return res
}
