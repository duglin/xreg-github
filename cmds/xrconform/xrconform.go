package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var Model = JSON{}

func TestAll(td *TD) {
	td.DependsOn(TestSniffTest)
	td.DependsOn(TestLoadModel)
	td.Run(TestRoot)
}

var ConfigFile = EnvString("XRC_CONFIG", "xrconform.config")
var Verbose = EnvBool("XRC_VERBOSE", false)
var ShowLogs = EnvBool("XRC_SHOWLOGS", false)

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

func RunConformanceTests(cmd *cobra.Command, args []string) {
	var err error

	td := NewTD(os.Args[0])
	td.Props["xr"], err = NewXRegistryWithConfigPath(ConfigFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	FailFast = false
	td.Run(TestAll)

	// td.Dump("")
	td.Print(os.Stdout, "", ShowLogs)
	if td.ExitCode() != 0 {
		os.Exit(td.ExitCode())
	}
}

func main() {
	cmd := &cobra.Command{
		Use:   "xrconform",
		Short: "xRegistry Conformance Tester",
		Run:   RunConformanceTests,
	}
	cmd.CompletionOptions.HiddenDefaultCmd = true
	cmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false,
		"Chatty?")
	cmd.PersistentFlags().StringVarP(&ConfigFile, "config", "c", ConfigFile,
		"Location of config file")
	cmd.PersistentFlags().BoolVarP(&ShowLogs, "logs", "l", ShowLogs,
		"Show logs on success")

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
