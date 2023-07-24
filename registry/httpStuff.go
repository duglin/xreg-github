package registry

import (
	"bytes"
	"encoding/json"
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
		err = HTTPGet(out, info)
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

func HTTPGETModel(w io.Writer, info *RequestInfo) error {
	if len(info.Parts) > 1 {
		info.ErrCode = http.StatusNotFound
		return fmt.Errorf("404: Not found\n")
	}

	model := info.Registry.Model
	if model == nil {
		model = &Model{}
	}

	buf, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		info.ErrCode = http.StatusInternalServerError
		return fmt.Errorf("500: " + err.Error())
	}

	fmt.Fprintf(w, "%s\n", string(buf))
	return nil
}

func HTTPGETContent(w io.Writer, info *RequestInfo) error {
	query := `
SELECT
  RegSID,Level,Plural,eSID,UID,PropName,PropValue,PropType,Path,Abstract
FROM FullTree WHERE RegSID=? AND `
	args := []any{info.Registry.DbSID}

	path := strings.Join(info.Parts, "/")

	if info.VersionUID == "" {
		query += `(Path=? OR Path LIKE ?)`
		args = append(args, path, path+"/%")
	} else {
		query += `Path=?`
		args = append(args, path)
	}
	query += " ORDER BY Path"

	log.VPrintf(3, "Query:\n%s", SubQuery(query, args))

	results, err := Query(query, args...)
	defer results.Close()

	if err != nil {
		info.ErrCode = http.StatusInternalServerError
		return fmt.Errorf("500: " + err.Error())
	}

	entity := readNextEntity(results)
	if entity == nil {
		info.ErrCode = http.StatusNotFound
		return fmt.Errorf("404: Not found\n")
	}

	var version *Entity
	versionsCount := 0
	if info.VersionUID == "" {
		// We're on a Resource, so go find the right Version
		vID := entity.Get("latestId").(string)
		for {
			v := readNextEntity(results)
			if v == nil && version == nil {
				info.ErrCode = http.StatusInternalServerError
				return fmt.Errorf("500: Can't find version: %s", vID)
			}
			if v == nil {
				break
			}
			versionsCount++
			if v.UID == vID {
				version = v
			}
		}
	} else {
		version = entity
	}

	log.VPrintf(3, "Entity: %#v", entity)
	log.VPrintf(3, "Version: %#v", version)

	headerIt := func(e *Entity, info *RequestInfo, key string, val any) error {
		str := ""
		if key == "tags" {
			buf, _ := json.Marshal(val)
			str = string(buf)
		} else {
			str = fmt.Sprintf("%v", val)
		}
		info.OriginalResponse.Header()["xRegistry-"+key] = []string{str}
		return nil
	}

	err = entity.SerializeProps(info, headerIt)
	if err != nil {
		panic(err)
	}

	if info.VersionUID == "" {
		info.OriginalResponse.Header()["xRegistry-versionsCount"] =
			[]string{fmt.Sprintf("%d", versionsCount)}
		info.OriginalResponse.Header()["xRegistry-versionsUrl"] =
			[]string{info.BaseURL + "/" + entity.Path + "/versions"}
	}

	url := ""
	if val := entity.Get("#resourceURL"); val != nil {
		url = val.(string)
	}
	if url != "" {
		info.OriginalResponse.Header().Add("Location", url)
		info.OriginalResponse.WriteHeader(http.StatusSeeOther)
		/*
			http.Redirect(info.OriginalResponse, info.OriginalRequest, url,
				http.StatusSeeOther)
		*/
		return nil
	}

	if val := entity.Get("#resourceProxyURL"); val != nil {
		url = val.(string)
	}
	if url != "" {
		// Just act as a proxy and copy the remote resource as our response
		resp, err := http.Get(url)
		if err != nil {
			info.ErrCode = http.StatusInternalServerError
			return fmt.Errorf("500: " + err.Error())
		}
		if resp.StatusCode/100 != 2 {
			info.ErrCode = resp.StatusCode
			return fmt.Errorf("%s: Remote error", resp.Status)
		}

		// Copy all HTTP headers
		for header, value := range resp.Header {
			info.OriginalResponse.Header()[header] = value
		}

		// Now copy the body
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			info.ErrCode = http.StatusInternalServerError
			return fmt.Errorf("500: " + err.Error())
		}
		return nil
	}

	buf := version.Get("#resource")
	if buf == nil {
		// No data so just return
		info.OriginalResponse.WriteHeader(200) // http.StatusNoContent)
		return nil
	}
	info.OriginalResponse.Write(buf.([]byte))

	return nil
}

func HTTPGet(w io.Writer, info *RequestInfo) error {
	info.Root = strings.Trim(info.Root, "/")

	if len(info.Parts) > 0 && info.Parts[0] == "model" {
		return HTTPGETModel(w, info)
	}

	if info.What == "Entity" && info.ResourceUID != "" && !info.ShowMeta {
		return HTTPGETContent(w, info)
	}

	query, args, err := GenerateQuery(info)

	results, err := Query(query, args...)
	defer results.Close()

	if err != nil {
		info.ErrCode = http.StatusInternalServerError
		return fmt.Errorf("500: " + err.Error())
	}

	jw := NewJsonWriter(w, info, results)
	jw.NextEntity()

	if info.What == "Coll" {
		_, err = jw.WriteCollection()
	} else {
		if jw.Entity == nil {
			info.ErrCode = http.StatusNotFound
			return fmt.Errorf("404: Not found\n")
		}
		err = jw.WriteEntity()
	}

	if err == nil {
		jw.Print("\n")
	} else {
		info.ErrCode = http.StatusInternalServerError
		err = fmt.Errorf("500: " + err.Error())
	}

	return err
}
