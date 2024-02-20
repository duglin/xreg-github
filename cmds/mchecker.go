package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/duglin/xreg-github/registry"
)

func Fatal(str string, args ...any) {
	str = strings.TrimSpace(str) + "\n"
	fmt.Fprintf(os.Stderr, str, args...)
	os.Exit(1)
}

func Verify(prefix string, buf []byte) {
	re := regexp.MustCompile(`(?m:([^#]*)#[^"]*$)`)
	buf = re.ReplaceAll(buf, []byte("${1}"))

	model := &registry.Model{}

	if err := registry.Unmarshal(buf, model); err != nil {
		Fatal("%sParsing error: %s", prefix, err)
	}

	if err := model.Verify(); err != nil {
		Fatal("%s%s", prefix, err)
	}
}

func main() {
	if len(os.Args) == 1 {
		buf, err := io.ReadAll(os.Stdin)
		if err != nil {
			Fatal("Error reading from stdin: %s", err)
		}
		Verify("", buf)
	}

	numFiles := len(os.Args) - 1
	for _, fileName := range os.Args[1:] {
		buf, err := os.ReadFile(fileName)
		if err != nil {
			Fatal("Error reading file %q: %s", fileName, err)
		}
		prefix := ""
		if numFiles > 1 {
			prefix = fileName + ": "
		}
		Verify(prefix, buf)
	}
}
