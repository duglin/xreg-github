package main

import (
	"fmt"
	"os"
	"strings"

	// log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
)

var worked = true
var Verbose = EnvBool("XR_VERBOSE", false)
var Server = EnvString("XR_SERVER", "")

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

func ErrStop(err error, prefix ...any) {
	if err == nil {
		return
	}

	str := err.Error()
	if prefix != nil {
		str = fmt.Sprintf(prefix[0].(string), prefix[1:]...)
	}
	Error(str)
}

func Error(str string, args ...any) {
	str = strings.TrimSpace(str) + "\n"
	fmt.Fprintf(os.Stderr, str, args...)
	worked = false
	os.Exit(1)
}

func main() {
	xrCmd := &cobra.Command{
		Use:   "xr",
		Short: "xRegistry CLI",
	}
	xrCmd.CompletionOptions.HiddenDefaultCmd = true
	xrCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false,
		"Chatty?")
	xrCmd.PersistentFlags().StringVarP(&Server, "server", "s", Server,
		"URL to server")

	addModelCmd(xrCmd)
	addRegistryCmd(xrCmd)
	addGroupCmd(xrCmd)

	if err := xrCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
