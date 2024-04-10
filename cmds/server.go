package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	log "github.com/duglin/dlog"
	"github.com/duglin/xreg-github/registry"
)

var Port = 8080
var DBName = "registry"
var Verbose = 2

func main() {
	if tmp := os.Getenv("VERBOSE"); tmp != "" {
		if tmpInt, err := strconv.Atoi(tmp); err == nil {
			Verbose = tmpInt
		}
	}

	doDelete := flag.Bool("delete", false, "Delete DB and exit")
	doRecreate := flag.Bool("recreate", false, "Recreate DB, then run")
	doVerify := flag.Bool("verify", false, "Exit after loading - for testing")
	flag.IntVar(&Verbose, "v", Verbose, "Verbose level")
	flag.Parse()

	log.SetVerbose(Verbose)

	if *doDelete || *doRecreate {
		err := registry.DeleteDB(DBName)
		if err != nil {
			panic(err)
		}
		if *doDelete {
			os.Exit(0)
		}
	}

	if !registry.DBExists(DBName) {
		registry.CreateDB(DBName)
	}

	registry.OpenDB(DBName)

	// testing
	if 0 == 1 {
		reg, err := registry.NewRegistry(nil, "test")
		ErrFatalf(err)
		gm, err := reg.Model.AddGroupModel("dirs", "dir")
		ErrFatalf(err)
		_, err = gm.AddResourceModel("files", "file", 2, true, true, true)
		ErrFatalf(err)

		g, err := reg.AddGroup("dirs", "dir1")
		r, err := g.AddResource("files", "f1", "v1")
		v1, err := r.FindVersion("v1")
		r.AddVersion("v2")
		ErrFatalf(v1.SetSave("name", "myname"))
		ErrFatalf(reg.Commit())
		os.Exit(0)
	}

	// e-testing

	reg, err := registry.FindRegistry(nil, "SampleRegistry")
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	if reg == nil {
		reg = LoadDirsSample(reg)
		LoadEndpointsSample(nil)
		LoadMessagesSample(nil)
		LoadSchemasSample(nil)
		LoadAPIGuru(nil, "APIs-guru", "openapi-directory")
	}

	if reg == nil {
		fmt.Fprintf(os.Stderr, "No registry loaded\n")
		os.Exit(1)
	}

	if tmp := os.Getenv("PORT"); tmp != "" {
		tmpInt, _ := strconv.Atoi(tmp)
		if tmpInt != 0 {
			Port = tmpInt
		}
	}

	if *doVerify {
		os.Exit(0)
	}

	registry.DefaultRegDbSID = reg.DbSID
	registry.NewServer(Port).Serve()
}
