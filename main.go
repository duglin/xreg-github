package main

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strconv"

	log "github.com/duglin/dlog"
	"github.com/duglin/xreg-github/registry"
	"github.com/duglin/xreg-github/tests"
)

func init() {
	log.SetVerbose(2)
}

var Port = "8080"
var Reg = (*registry.Registry)(nil)

func handler(w http.ResponseWriter, r *http.Request) {
	saveVerbose := log.GetVerbose()
	if tmp := r.URL.Query().Get("verbose"); tmp != "" {
		if v, err := strconv.Atoi(tmp); err == nil {
			log.SetVerbose(v)
		}
	}
	defer log.SetVerbose(saveVerbose)

	log.VPrintf(2, "%s %s", r.Method, r.URL.Path)

	info, err := Reg.ParseRequest(r)
	if err != nil {
		w.WriteHeader(info.ErrCode)
		w.Write([]byte(err.Error()))
		return
	}

	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var out = io.Writer(w)
	buf := (*bytes.Buffer)(nil)

	// If we want to tweak the output we'll need to buffer it
	if r.URL.Query().Has("html") || r.URL.Query().Has("noprops") {
		buf = &bytes.Buffer{}
		out = io.Writer(buf)
	}

	err = Reg.NewGet(out, info)

	if tmp := r.URL.Query().Get("verbose"); tmp != "" {
		log.SetVerbose(saveVerbose)
	}

	if buf == nil {
		// Not buffering so just return
		if err != nil {
			if info.ErrCode != 0 {
				w.WriteHeader(info.ErrCode)
			} else {
				w.WriteHeader(http.StatusBadRequest)
			}
			w.Write([]byte(err.Error()))
		}
		return
	}

	if r.URL.Query().Has("noprops") {
		buf = bytes.NewBuffer(tests.RemoveProps(buf.Bytes()))
	}

	if r.URL.Query().Has("html") {
		w.Header().Add("Content-Type", "text/html")
		buf = bytes.NewBuffer(tests.HTMLify(r, buf.Bytes()))
		w.Write([]byte("<pre>\n"))
	}

	w.Write(buf.Bytes())
}

func main() {
	Reg = tests.TestAll()
	// Reg.Delete()

	// Reg = LoadSample()
	// Reg = LoadGitRepo("APIs-guru", "openapi-directory")

	if tmp := os.Getenv("PORT"); tmp != "" {
		Port = tmp
	}

	http.HandleFunc("/", handler)
	log.VPrintf(1, "Listening on %s", Port)
	http.ListenAndServe(":"+Port, nil)
}
