package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	// log "github.com/duglin/dlog"
	"github.com/duglin/xreg-github/registry"
)

func Fatal(str string, args ...any) {
	str = strings.TrimSpace(str) + "\n"
	fmt.Fprintf(os.Stderr, str, args...)
	os.Exit(1)
}

func Verify(fileName string, buf []byte) {
	re := regexp.MustCompile(`(?m:([^#]*)#[^"]*$)`)
	buf = re.ReplaceAll(buf, []byte("${1}"))

	model := &registry.Model{}
	if err := json.Unmarshal(buf, model); err != nil {
		Fatal("Error parsing %q: %s", fileName, err)
	}

	err := model.Verify()
	if err != nil {
		Fatal("%s", err)
	}
}

func main() {
	if len(os.Args) == 1 {
		buf, err := io.ReadAll(os.Stdin)
		if err != nil {
			Fatal("Error reading from stdin: %s", err)
		}
		Verify("<stdin>", buf)
	}

	for _, fileName := range os.Args[1:] {
		buf, err := os.ReadFile(fileName)
		if err != nil {
			Fatal("Error reading file %q: %s", fileName, err)
		}
		Verify(fileName, buf)
	}
}
