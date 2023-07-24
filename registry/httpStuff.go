package registry

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
)

type Server struct {
	Port       int
	HTTPServer *http.Server
}

var Reg *Registry

func NewServer(reg *Registry, port int) *Server {
	Reg = reg
	server := &Server{
		Port: port,
		HTTPServer: &http.Server{
			Addr: fmt.Sprintf(":%d", port),
		},
	}
	server.HTTPServer.Handler = server
	return server
}

func (s *Server) Close() {
	s.HTTPServer.Close()
}

func (s *Server) Start() *Server {
	go s.Serve()
	/*
		for {
			_, err := http.Get(fmt.Sprintf("http://localhost:%d", s.Port))
			if err == nil || !strings.Contains(err.Error(), "refused") {
				break
			}
		}
	*/
	return s
}

func (s *Server) Serve() {
	log.VPrintf(1, "Listening on %d", s.Port)
	s.HTTPServer.ListenAndServe()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if Reg == nil {
		panic("No registry specified")
	}

	saveVerbose := log.GetVerbose()
	if tmp := r.URL.Query().Get("verbose"); tmp != "" {
		if v, err := strconv.Atoi(tmp); err == nil {
			log.SetVerbose(v)
		}
		defer log.SetVerbose(saveVerbose)
	}

	log.VPrintf(2, "%s %s", r.Method, r.URL.Path)

	info, err := Reg.ParseRequest(w, r)
	if err != nil {
		w.WriteHeader(info.ErrCode)
		w.Write([]byte(err.Error()))
		return
	}

	var out = io.Writer(w)
	buf := (*bytes.Buffer)(nil)

	// If we want to tweak the output we'll need to buffer it
	if r.URL.Query().Has("html") || r.URL.Query().Has("noprops") {
		buf = &bytes.Buffer{}
		out = io.Writer(buf)
	}

	switch strings.ToUpper(r.Method) {
	case "GET":
		err = Reg.HTTPGet(out, info)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(fmt.Sprintf("HTTP method %q not supported", r.Method)))
		return
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
		buf = bytes.NewBuffer(RemoveProps(buf.Bytes()))
	}
	if r.URL.Query().Has("oneline") {
		buf = bytes.NewBuffer(OneLine(buf.Bytes()))
	}

	if r.URL.Query().Has("html") {
		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte("<pre>\n"))
		buf = bytes.NewBuffer(HTMLify(r, buf.Bytes()))
	}

	w.Write(buf.Bytes())

	if err != nil {
		w.Write([]byte(err.Error()))
	}
}
