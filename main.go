package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	log "github.com/duglin/dlog"
	"github.com/duglin/xreg-github/registry"
)

func init() {
	log.SetVerbose(2)
}

var Port = 8080
var DBName = "registry"
var Verbose = 2

func main() {
	var err error

	doDelete := flag.Bool("delete", false, "Delete DB an exit")
	doRecreate := flag.Bool("recreate", false, "Recreate DB, then run")
	flag.IntVar(&Verbose, "v", 2, "Verbose level")
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
	reg, err := registry.FindRegistry("SampleRegistry")
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

	registry.DefaultReg = reg
	registry.NewServer(Port).Serve()
}
