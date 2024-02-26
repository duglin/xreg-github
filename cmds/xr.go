package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	// log "github.com/duglin/dlog"
	"github.com/duglin/xreg-github/registry"
	"github.com/spf13/cobra"
)

var worked = true

func Error(str string, args ...any) {
	str = strings.TrimSpace(str) + "\n"
	fmt.Fprintf(os.Stderr, str, args...)
	worked = false
	os.Exit(1)
}

func VerifyModel(fileName string, buf []byte) {
	var err error

	if len(os.Args) > 2 && fileName != "" {
		fileName += ": "
	} else {
		fileName = ""
	}

	buf, err = registry.ProcessImports(fileName, buf, true)
	if err != nil {
		Error("%s%s", fileName, err)
	}

	model := &registry.Model{}

	if err := registry.Unmarshal(buf, model); err != nil {
		Error("%s%s", fileName, err)
	}

	if err := model.Verify(); err != nil {
		Error("%s%s", fileName, err)
	}
}

func modelVerifyFunc(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		buf, err := io.ReadAll(os.Stdin)
		if err != nil {
			Error("Error reading from stdin: %s", err)
		}
		VerifyModel("", buf)
	}

	for _, fileName := range args {
		buf, err := os.ReadFile(fileName)
		if err != nil {
			Error("Error reading file %q: %s", fileName, err)
		}
		VerifyModel(fileName, buf)
	}
}

func modelNormalizeFunc(cmd *cobra.Command, args []string) {
	var err error
	var buf []byte
	fileName := ""

	if len(args) == 0 {
		buf, err = io.ReadAll(os.Stdin)
		if err != nil {
			Error("Error reading from stdin: %s", err)
		}
	}

	for _, fileName = range args {
		buf, err = os.ReadFile(fileName)
		if err != nil {
			Error("Error reading file %q: %s", fileName, err)
		}
	}

	buf, err = registry.ProcessImports(fileName, buf, true)
	if err != nil {
		Error(err.Error())
	}

	tmp := map[string]any{}
	err = registry.Unmarshal(buf, &tmp)
	if err != nil {
		Error(err.Error())
	}
	fmt.Printf("%s\n", registry.ToJSON(tmp))
}

func main() {
	xrCmd := &cobra.Command{
		Use:   "xr",
		Short: "xRegistry CLI",
	}
	xrCmd.CompletionOptions.HiddenDefaultCmd = true

	modelCmd := &cobra.Command{
		Use:   "model",
		Short: "model commands",
	}
	xrCmd.AddCommand(modelCmd)

	modelNormalizeCmd := &cobra.Command{
		Use:   "normalize [ - | FILE... ]",
		Short: "Parse and resolve imports in an xRegistry model document",
		Run:   modelNormalizeFunc,
	}
	modelCmd.AddCommand(modelNormalizeCmd)

	modelVerifyCmd := &cobra.Command{
		Use:   "verify [ - | FILE... ]",
		Short: "Parse and verify xRegistry model document",
		Run:   modelVerifyFunc,
	}
	modelCmd.AddCommand(modelVerifyCmd)

	if err := xrCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
