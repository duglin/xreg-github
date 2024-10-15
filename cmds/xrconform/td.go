package main

import (
	"fmt"
	"io"
	"reflect"
	"runtime"
	"strings"
	"time"
)

var PASS = 1
var FAIL = 2
var WARN = 3
var SKIP = 4
var LOG = 5 // Only shown on failure or when they ask to see all logs
var MSG = 6 // Like LOG but will always be printed

var StatusText = []string{"", "PASS", "FAIL", "WARN", "SKIP", "LOG", "MSG"}

var FailFast = true
var IgnoreWarn = true
var TestsRun = map[string]*TD{}

type TestFn func(td *TD)

func (fn TestFn) Name() string {
	name := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	before, name, _ := strings.Cut(name, ".")
	if name == "" {
		name = before
	}
	return name
}

type LogEntry struct {
	Date    time.Time
	Type    int // pass, fail, warning, skip, else log or TD
	Text    string
	Subtest *TD
}

// TestData
type TD struct {
	TestName string
	Parent   *TD `json:"-"`
	Logs     []*LogEntry

	Status int // PASS, FAIL, ...
	Props  map[string]any

	NumPass int // These will include the status of _this_ TD and its children
	NumFail int
	NumWarn int
	NumSkip int
}

func NewTD(name string, parent ...*TD) *TD {
	p := (*TD)(nil)
	if len(parent) > 0 {
		if len(parent) > 1 {
			panic("too many parents")
		}
		p = parent[0]
	}

	newTD := &TD{
		TestName: name,
		Parent:   p,
		Logs:     []*LogEntry{},

		Status:  PASS,
		Props:   map[string]any{},
		NumPass: 1,
	}

	if p != nil {
		newLE := &LogEntry{
			Date:    time.Now(),
			Type:    0,
			Text:    "",
			Subtest: newTD,
		}
		p.Logs = append(p.Logs, newLE)
		p.AddStatus(PASS)
	}

	return newTD
}

func (td *TD) ExitCode() int {
	if td.Status == PASS || td.Status == SKIP ||
		(IgnoreWarn && td.Status == WARN) {
		return 0
	}
	if td.Status == 0 {
		return 255
	}
	return td.Status
}

func (td *TD) Dump(indent string) {
	fmt.Printf("%sTestName: %s\n", indent, td.TestName)
	if len(td.Logs) > 0 {
		fmt.Printf("%s  Logs:\n", indent)
		for _, le := range td.Logs {
			if le.Subtest != nil {
				fmt.Printf("%s    %s", indent, le.Date.Format(time.RFC3339))
				le.Subtest.Dump(indent + "    ")
			} else {
				fmt.Printf("%s    %s (%s) %q\n", indent,
					le.Date.Format(time.RFC3339), StatusText[le.Type],
					le.Text)
			}
		}
	}
}

func (td *TD) Print(out io.Writer, indent string, showLogs bool) {
	td.write(out, indent, showLogs)
	fmt.Printf("\n"+indent+"Pass: %d   Fail: %d   Warn: %d   Skip: %d\n",
		td.NumPass, td.NumFail, td.NumWarn, td.NumSkip)
}

func (td *TD) write(out io.Writer, indent string, showLogs bool) {
	td.writeHeader(out, indent, showLogs)
	td.writeBody(out, indent, showLogs)
}

func (td *TD) writeHeader(out io.Writer, indent string, showLogs bool) {
	str := indent + StatusText[td.Status] + ": "

	out.Write([]byte(str + td.TestName + "\n"))
	// out.Write([]byte(fmt.Sprintf("%s%s %d/%d/%d/%d\n",
	// str, td.TestName, td.NumPass, td.NumFail, td.NumWarn, td.NumSkip)))
}

func (td *TD) writeBody(out io.Writer, indent string, showLogs bool) {
	saveIndent := indent
	endSaveIndent := strings.ReplaceAll(saveIndent, "├─", "│ ")

	// Calc the last logEntry - removing the LOG messages at the end
	lastLog := len(td.Logs) - 1
	for ; showLogs == false && lastLog > 0; lastLog-- {
		if td.Logs[lastLog].Type != LOG {
			break
		}
	}

	for i := 0; i <= lastLog; i++ {
		le := td.Logs[i]
		str := ""
		// date := le.Date.Format(time.RFC3339)
		if i == lastLog {
			indent = endSaveIndent + "└─ "
		} else {
			indent = endSaveIndent + "├─ "
		}

		if le.Type > 0 && le.Type < LOG {
			str = PrettyPrint(indent, StatusText[le.Type]+": ", le.Text) + "\n"
		} else if le.Subtest == nil { // log or msg
			// Show logs it's a MSG, they asked for all logs, or TD=FAIL
			if le.Type == MSG || showLogs || td.Status == FAIL {
				str = PrettyPrint(indent, "", le.Text) + "\n"
			}
		} else { // subtest
			if i == lastLog {
				le.Subtest.writeHeader(out, endSaveIndent+"└─ ", showLogs)
				le.Subtest.writeBody(out, saveIndent+"   ", showLogs)
			} else {
				le.Subtest.write(out, indent, showLogs)
			}
		}
		out.Write([]byte(str))
	}
}

func (td *TD) AddStatus(status int) {
	if status == 0 || status >= LOG {
		return
	}
	/*
		fmt.Printf("  %q before status: %s", td.TestName, StatusText[td.Status])
		fmt.Printf(" ( %d / %d / %d / %d\n", td.NumPass, td.NumFail,
		td.NumWarn, td.NumSkip)
	*/
	switch td.Status {
	case 0, PASS:
		td.Status = status
	case WARN:
		switch status {
		case FAIL:
			td.Status = status
		}
	case SKIP:
		switch status {
		case FAIL:
			td.Status = status
		case WARN:
			td.Status = status
		}
	}

	// Recalc totals, up the chain
	for p := td; p != nil; p = p.Parent {
		// fmt.Printf("    Recalc'ing: %s\n", p.TestName)
		sums := [4]int{0, 0, 0, 0}
		sums[p.Status-1] = 1
		for _, le := range p.Logs {
			if le.Subtest != nil {
				sums[0] += le.Subtest.NumPass
				sums[1] += le.Subtest.NumFail
				sums[2] += le.Subtest.NumWarn
				sums[3] += le.Subtest.NumSkip
			} else if le.Type > 0 && le.Type < LOG {
				sums[le.Type-1]++
			}
		}
		p.NumPass = sums[0]
		p.NumFail = sums[1]
		p.NumWarn = sums[2]
		p.NumSkip = sums[3]
		// fmt.Printf("    After Recalc: %s - %v\n", p.TestName, sums)
	}

	// ShowStack()
	/*
		fmt.Printf("  %q after status: %s", td.TestName, StatusText[td.Status])
		fmt.Printf(" ( %d / %d / %d / %d\n", td.NumPass, td.NumFail,
		td.NumWarn, td.NumSkip)
	*/

	if td.Parent != nil {
		td.Parent.AddStatus(td.Status)
	}
}

// PASS|FAIL|WARN|SKIP, testNameText, substitute args for testName
func (td *TD) Report(status int, args ...any) {
	// fmt.Printf("Report: %q %s : %v\n", td.TestName, StatusText[status], args)

	if len(args) > 0 {
		td.Logs = append(td.Logs, &LogEntry{
			Date:    time.Now(),
			Type:    status,
			Text:    fmt.Sprintf(args[0].(string), args[1:]...),
			Subtest: nil,
		})
	}
	td.AddStatus(status)
}

func (td *TD) Pass(args ...any)    { td.Report(PASS, args...) }
func (td *TD) Fail(args ...any)    { td.Report(FAIL, args...) }
func (td *TD) FailNow(args ...any) { td.Report(FAIL, args...); td.Stop() }
func (td *TD) Warn(args ...any)    { td.Report(WARN, args...) }
func (td *TD) Skip(args ...any)    { td.Report(SKIP, args...) }
func (td *TD) Log(args ...any)     { td.Report(LOG, args...) }
func (td *TD) Msg(args ...any)     { td.Report(MSG, args...) }
func (td *TD) Stop()               { panic("stop") }

func (td *TD) DependsOn(fn TestFn) {
	if prevTD, ok := TestsRun[fn.Name()]; ok {
		if prevTD.Status == FAIL {
			td.FailNow("Dependency %q (cached), exiting", fn.Name())
		} else {
			td.Pass("Dependency %q (cached)", fn.Name())
		}
	} else {
		newTD := td.Run(fn)

		if newTD.Status == FAIL {
			td.Msg("Dependency %q failed, exiting", fn.Name())
			td.Stop()
			return
		} else {
			// td.Pass("Dependency %q", fn.Name())
		}
	}
}

func (td *TD) Run(fn TestFn) *TD {
	before, name, _ := strings.Cut(fn.Name(), ".")
	if name == "" {
		name = before
	}
	newTD := NewTD(name, td)

	// Save in the cache
	TestsRun[fn.Name()] = newTD

	// Run it and catch any panic()
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Do nothing
				// Just allow the panic() caller to exit immediately
				if r != "stop" {
					panic(r)
				}
			}
		}()
		fn(newTD)
	}()

	return newTD
}

func PrettyPrint(indent string, prefix string, text string) string {
	width := 79
	line := strings.TrimRight(indent+prefix+text, " ")

	indent = indent + strings.Repeat(" ", len(prefix))

	str := ""

	for len(line) > width {
		left := ""
		chopAt := width // do not use width-1
		for i := chopAt; i+1 > len(indent); i-- {
			runeIt := ([]rune)(line)
			if runeIt[i] == ' ' {
				left = string(runeIt[:i])
				line = strings.TrimLeft(string(runeIt[i+1:]), " ")
				break
			}
		}

		if strings.Contains(indent, "└") {
			indent = strings.ReplaceAll(indent, "└─", "  ")
		} else {
			indent = strings.ReplaceAll(indent, "├─", "│ ")
		}

		line = indent + line
		str += left + "\n"
	}
	str += line
	return str
}
