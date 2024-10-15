package main

import (
	// "fmt"
	"os"
)

func TestPass1(td *TD) {
}

func TestPass2(td *TD) {
	td.Pass()
}

func TestPass3(td *TD) {
	td.Pass("pass2 test")
}

func TestFail1(td *TD) {
	td.Fail()
}

func TestFail2(td *TD) {
	td.Fail("fail3 test")
}

func TestXRegistry(td *TD) {
	td.Log("log line 2")
	td.Pass("hello")
	td.Warn("hello1")
	td.Log("log line 3")
}

func TestServer(td *TD) {
	// xr := (*XRegistry) td.Props["xr"]
	td.Log("log line 1")
	td.Warn("warn 1")
	td.Fail("fail 1")
	td.Pass("pass 1")
}

func main() {
	td := NewTD(os.Args[0])
	td.Props["xr"] = &XRegistry{}

	td.Run(TestPass1)
	td.Run(TestPass2)
	td.Run(TestPass3)
	td.Run(TestFail1)
	td.Run(TestFail2)

	td.Run(TestXRegistry)
	td.Run(TestServer)
	td.Run(TestRegistry0)

	td.Run(TestAll)
	// td.Run(TestAll)
	// td.Dump("")
	// fmt.Printf("TD: %s\n", ToJSON(td))
	td.Print(os.Stdout, "", false)
	os.Exit(td.ExitCode())
}
