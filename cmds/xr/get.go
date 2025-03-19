package main

import (
	"encoding/json"
	"fmt"
	// "text/tabwriter"

	// log "github.com/duglin/dlog"
	"github.com/xregistry/server/cmds/xr/xrlib"
	// "github.com/xregistry/server/registry"
	"github.com/spf13/cobra"
)

func addGetCmd(parent *cobra.Command) {
	getCmd := &cobra.Command{
		Use:   "get [ XID ]",
		Short: "Retrieve data from the registry",
		Run:   getFunc,
	}
	getCmd.Flags().StringP("output", "o", "table", "Output format(table,json)")

	parent.AddCommand(getCmd)
}

func getFunc(cmd *cobra.Command, args []string) {
	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, err := xrlib.GetRegistry(Server)
	if err != nil {
		Error(err.Error())
	}

	output, _ := cmd.Flags().GetString("output")
	if !xrlib.ArrayContains([]string{"table", "json"}, output) {
		Error("--ouput must be one of 'table', 'json'")
	}

	if len(args) == 0 {
		args = []string{"/"}
	}

	objects := map[string]any{}
	for _, xid := range args {
		suffix := ""
		if len(args) > 1 {
			rm, err := reg.GetResourceModelFromXID(xid)
			if err != nil {
				Error(err.Error())
			}
			if rm != nil && rm.HasDocument != nil && *(rm.HasDocument) == true {
				suffix = "$details"
			}
		}
		body, err := reg.HttpDo("GET", xid+suffix, nil)
		if err != nil {
			if len(args) > 1 {
				Error(xid + ": " + err.Error())
			} else {
				Error(err.Error())
			}
		}
		obj := map[string]any(nil)
		if err = json.Unmarshal(body, &obj); err != nil {
			Error(err.Error())
		}
		objects[xid] = obj
	}

	if output == "json" {
		str, _ := json.MarshalIndent(objects, "", "  ")
		fmt.Printf("%s\n", str)
		return
	}

	// output == "table"
}
