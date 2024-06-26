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

var doDelete *bool
var doRecreate *bool
var doVerify *bool
var firstTimeDB = true

func InitDB() {
	if firstTimeDB {
		if *doDelete || *doRecreate {
			err := registry.DeleteDB(DBName)
			if err != nil {
				log.Printf("Error deleting DB %q: %s", DBName, err)
				return
			}
			if *doDelete {
				// We're just deleting the DB so exit the program
				os.Exit(0)
			}
		}
		firstTimeDB = false
	}

	if !registry.DBExists(DBName) {
		registry.CreateDB(DBName)
	}

	err := registry.OpenDB(DBName)
	if err != nil {
		log.VPrintf(1, "Can't connect to db: %s", err)
		return
	}

	reg, err := registry.FindRegistry(nil, "SampleRegistry")
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		return
	}

	if reg == nil {
		reg = LoadDirsSample(reg)
		LoadEndpointsSample(nil)
		LoadMessagesSample(nil)
		LoadSchemasSample(nil)
		LoadAPIGuru(nil, "APIs-guru", "openapi-directory")
		LoadDocStore(nil)
		if os.Getenv("XR_LOAD_LARGE") != "" {
			go LoadLargeSample(nil)
		}
	}

	if reg == nil {
		fmt.Fprintf(os.Stderr, "No registry loaded\n")
		os.Exit(1)
	}

	if *doVerify {
		log.VPrintf(1, "Done verifying, exiting")
		os.Exit(0)
	}

	registry.DefaultRegDbSID = reg.DbSID
}

func main() {
	if tmp := os.Getenv("VERBOSE"); tmp != "" {
		if tmpInt, err := strconv.Atoi(tmp); err == nil {
			Verbose = tmpInt
		}
	}

	doDelete = flag.Bool("delete", false, "Delete DB and exit")
	doRecreate = flag.Bool("recreate", false, "Recreate DB, then run")
	doVerify = flag.Bool("verify", false, "Exit after loading - for testing")
	flag.IntVar(&Verbose, "v", Verbose, "Verbose level")
	flag.Parse()

	log.SetVerbose(Verbose)

	if tmp := os.Getenv("PORT"); tmp != "" {
		tmpInt, _ := strconv.Atoi(tmp)
		if tmpInt != 0 {
			Port = tmpInt
		}
	}

	registry.DB_Name = DBName
	// registry.DB_InitFunc = InitDB
	InitDB()

	registry.NewServer(Port).Serve()
}
