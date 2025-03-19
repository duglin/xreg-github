package xrlib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/xregistry/server/registry"
)

// var VerboseFlag = EnvBool("XR_VERBOSE", false)
var DebugFlag = EnvBool("XR_DEBUG", false)
var Server = EnvString("XR_SERVER", "")

func Debug(args ...any) {
	if !DebugFlag || len(args) == 0 || registry.IsNil(args[0]) {
		return
	}
	// VerboseFlag = true
	// Verbose(args)
	fmtStr := ""
	ok := false

	if fmtStr, ok = args[0].(string); ok {
		// fmtStr already set
	} else {
		fmtStr = fmt.Sprintf("%v", args[0])
	}

	fmt.Fprintf(os.Stderr, fmtStr+"\n", args[1:]...)
}

/*
func Verbose(args ...any) {
	if !VerboseFlag || len(args) == 0 || registry.IsNil(args[0]) {
		return
	}

	fmtStr := ""
	ok := false

	if fmtStr, ok = args[0].(string); ok {
		// fmtStr already set
	} else {
		fmtStr = fmt.Sprintf("%v", args[0])
	}

	fmt.Fprintf(os.Stderr, fmtStr+"\n", args[1:]...)
}
*/

func EnvBool(name string, def bool) bool {
	val := os.Getenv(name)
	if val != "" {
		def = strings.EqualFold(val, "true")
	}
	return def
}

func EnvString(name string, def string) string {
	val := os.Getenv(name)
	if val != "" {
		def = val
	}
	return def
}

// statusCode, body
// Add headers (in and out) later
func HttpDo(verb string, url string, body []byte) ([]byte, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}

	bodyReader := bytes.NewReader(body)

	req, err := http.NewRequest(verb, url, bodyReader)
	if err != nil {
		return nil, err
	}

	Debug("Request: %s %s", verb, url)
	if len(body) != 0 {
		Debug("Body:\n%s", string(body))
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err = io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode/100 != 2 {
		tmp := res.Status
		if len(body) != 0 {
			tmp = string(body)
		}
		err = fmt.Errorf(tmp)
	}

	Debug("Response: %s", res.Status)
	if len(body) != 0 {
		Debug("Body:\n%s", string(body))
	}

	return body, err
}

// Support "http" and "-" (stdin)
func ReadFile(fileName string) ([]byte, error) {
	buf := []byte(nil)
	var err error

	if fileName == "" || fileName == "-" {
		buf, err = io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("Error reading from stdin: %s", err)
		}
	} else if strings.HasPrefix(fileName, "http") {
		res, err := http.Get(fileName)
		if err != nil {
			return nil, err
		}

		buf, err = io.ReadAll(res.Body)
		res.Body.Close()

		if err != nil {
			return nil, err
		}

		if res.StatusCode/100 != 2 {
			return nil, fmt.Errorf("Error downloading %q: %s\n%s",
				fileName, res.Status, string(buf))
		}
	} else {
		buf, err = os.ReadFile(fileName)
		if err != nil {
			return nil, fmt.Errorf("Error reading file %q: %s", fileName, err)
		}
	}

	return buf, nil
}

func IsValidJSON(buf []byte) error {
	tmp := map[string]any{}
	if err := registry.Unmarshal(buf, &tmp); err != nil {
		return err
	}
	return nil
}

func AnyToString(val any) (string, error) {
	valStr, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("%q isn't a string value", val)
	}
	return valStr, nil
}

func ToJSON(val any) string {
	buf, _ := json.MarshalIndent(val, "", "  ")
	return string(buf)
}

func ArrayContains(strs []string, needle string) bool {
	for _, s := range strs {
		if needle == s {
			return true
		}
	}
	return false
}

type XID struct {
	Group      string
	GroupID    string
	Resource   string
	ResourceID string
	Version    string // non-"" if xid included .../versions
	VersionID  string
}

func ParseXID(xidStr string) *XID {
	xidStr = strings.TrimLeft(xidStr, "/")
	parts := strings.SplitN(xidStr, "/", 6)

	xid := &XID{}
	xid.Group = parts[0]
	if len(parts) > 1 {
		xid.GroupID = parts[1]
		if len(parts) > 2 {
			xid.Resource = parts[2]
			if len(parts) > 3 {
				xid.ResourceID = parts[3]
				if len(parts) > 4 {
					xid.Version = parts[4]
					if len(parts) > 5 {
						xid.VersionID = parts[5]
					}
				}
			}
		}
	}
	return xid
}
