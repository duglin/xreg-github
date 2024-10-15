package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

func addRegistryCmd(parent *cobra.Command) {
	registryCmd := &cobra.Command{
		Use:   "registry",
		Short: "registry commands",
	}

	registryGetCmd := &cobra.Command{
		Use:   "get [ [PATH][?QUERY] ]",
		Short: "Retrieve the Registry",
		Run:   registryGetFunc,
	}
	registryCmd.AddCommand(registryGetCmd)
	registryGetCmd.Flags().BoolP("model", "m", false, "Show model")
	registryGetCmd.Flags().StringArrayP("inline", "i", nil, "Inline value")
	registryGetCmd.Flags().StringArrayP("filter", "f", nil, "Filter value")

	registrySetCmd := &cobra.Command{
		Use:   "set attributePath[=value | -]",
		Short: "Modify an attribute on the Registry entity",
		Run:   registrySetFunc,
	}
	registryCmd.AddCommand(registrySetCmd)

	parent.AddCommand(registryCmd)
}

func registryGetFunc(cmd *cobra.Command, args []string) {
	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	url := Server
	if len(args) == 1 {
		url += "/" + args[0]
	} else if len(args) > 1 {
		Error("Too many arguments - just PATH[?QUERY] allowed")
	}

	next := "?"
	if strings.Contains(url, "?") {
		next = "&"
	}

	model, _ := cmd.Flags().GetBool("model")
	if model {
		url += next + "model"
		next = "&"
	}

	inlines, _ := cmd.Flags().GetStringArray("inline")
	for _, inline := range inlines {
		url += next + "inline=" + inline
		next = "&"
	}

	filters, _ := cmd.Flags().GetStringArray("filter")
	for _, filter := range filters {
		url += next + "filter=" + filter
		next = "&"
	}

	res, err := http.Get(url)
	ErrStop(err, "Error talking to server (%s): %s", Server, err)
	if err != nil {
		Error(err.Error())
	}

	body, err := io.ReadAll(res.Body)
	ErrStop(err, "Error reading server response: %s", err)
	fmt.Printf("%s", string(body))
}

func registrySetFunc(cmd *cobra.Command, args []string) {
	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	if len(args) == 0 {
		Error("Need at least one name=value pair")
	}

	values := map[string]*string{}

	for _, arg := range args {
		// Note: foo= and foo are equivalent
		// Note: foo- means delete it
		path, value, found := strings.Cut(arg, "=")
		if len(path) == 0 {
			Error("Missing an attribute path on %q", arg)
		}
		valPtr := &value

		del := false
		if path, del = strings.CutSuffix(path, "-"); del {
			if found {
				Error("Using both \"-\" and \"=\" on %q isn't allowed", arg)
			}
			valPtr = nil
		}

		values[path] = valPtr
	}

	fmt.Printf("Values:\n%v\n", values)
}
