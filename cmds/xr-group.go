package main

import (
	// log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
)

func addGroupCmd(parent *cobra.Command) {
	groupsCmd := &cobra.Command{
		Use:   "groups",
		Short: "groups commands",
	}

	groupAddCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new Group",
		Run:   groupAddFunc,
	}
	groupsCmd.AddCommand(groupAddCmd)

	parent.AddCommand(groupsCmd)
}

func groupAddFunc(cmd *cobra.Command, args []string) {
}
