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
	err := s.HTTPServer.ListenAndServe()
	if err != http.ErrServerClosed {
		log.Printf("Serve: %s", err)
	}
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

		defer func() {
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
		}()
	}

	// These should only return an error if they didn't already
	// send a response back to the client.
	switch strings.ToUpper(r.Method) {
	case "GET":
		err = HTTPGet(out, info)
	case "PUT":
		err = HTTPPut(out, info)
	default:
		info.ErrCode = http.StatusMethodNotAllowed
		err = fmt.Errorf("HTTP method %q not supported", r.Method)
	}

	if err != nil {
		str := ""
		if info.ErrCode != 0 {
			w.WriteHeader(info.ErrCode)
			str = fmt.Sprintf("%d: ", info.ErrCode)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
		str += err.Error()
		if str[len(str)-1] != '\n' {
			str += "\n"
		}
		w.Write([]byte(str))
	}
}

func HTTPGETModel(w io.Writer, info *RequestInfo) error {
	if len(info.Parts) > 1 {
		info.ErrCode = http.StatusNotFound
		return fmt.Errorf("Not found")
	}

	model := info.Registry.Model
	if model == nil {
		model = &Model{}
	}

	httpModel := ModelToHTTPModel(model)

	buf, err := json.MarshalIndent(httpModel, "", "  ")
	if err != nil {
		info.ErrCode = http.StatusInternalServerError
		return err
	}

	info.OriginalResponse.Header().Add("Content-Type", "application/json")
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
		return err
	}

	entity := readNextEntity(results)
	if entity == nil {
		info.ErrCode = http.StatusNotFound
		return fmt.Errorf("Not found")
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
				return fmt.Errorf("Can't find version: %s", vID)
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
		if key == "labels" {
			header := info.OriginalResponse.Header()
			for name, value := range val.(map[string]string) {
				header["xRegistry-labels-"+name] = []string{value}
			}
			return nil
		}
		str = fmt.Sprintf("%v", val)

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
			return err
		}
		if resp.StatusCode/100 != 2 {
			info.ErrCode = resp.StatusCode
			return fmt.Errorf("Remote error")
		}

		// Copy all HTTP headers
		for header, value := range resp.Header {
			info.OriginalResponse.Header()[header] = value
		}

		// Now copy the body
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			info.ErrCode = http.StatusInternalServerError
			return err
		}
		return nil
	}

	buf := version.Get("#resource")
	if buf == nil {
		// No data so just return
		info.OriginalResponse.WriteHeader(200) // http.StatusNoContent)
		return nil
	}
	w.Write(buf.([]byte))

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
		return err
	}

	jw := NewJsonWriter(w, info, results)
	jw.NextEntity()

	if info.What != "Coll" {
		// Collections will need to print the {}, so don't error for them
		if jw.Entity == nil {
			info.ErrCode = http.StatusNotFound
			return fmt.Errorf("Not found")
		}
	}

	info.OriginalResponse.Header().Add("Content-Type", "application/json")
	if info.What == "Coll" {
		_, err = jw.WriteCollection()
	} else {
		err = jw.WriteEntity()
	}

	if err == nil {
		jw.Print("\n")
	} else {
		info.ErrCode = http.StatusInternalServerError
	}

	return err
}

func HTTPPut(w io.Writer, info *RequestInfo) error {
	info.Root = strings.Trim(info.Root, "/")

	if len(info.Parts) > 0 && info.Parts[0] == "model" {
		return HTTPPUTModel(w, info)
	}

	// reg := info.Registry

	props, err := LoadHTTPProps(info)
	if err != nil {
		return err
	}

	log.Printf("Props:\n%#v\n", props)

	/*
		if len(info.Parts) == 0 {
			return HTTPPutRegistry(w, info)
		}
	*/

	return nil
}

type HTTPResourceModel struct {
	Plural    string `json:"plural,omitempty"`
	Singular  string `json:"singular,omitempty"`
	Versions  int    `json:"versions"`
	VersionId bool   `json:"versionId"`
	Latest    bool   `json:"latest"`
}

type HTTPGroupModel struct {
	Plural   string `json:"plural,omitempty"`
	Singular string `json:"singular,omitempty"`
	Schema   string `json:"schema,omitempty"`

	Resources []HTTPResourceModel `json:"resources,omitempty"`
}

type HTTPModel struct {
	Schema string           `json:"schema,omitempty"`
	Groups []HTTPGroupModel `json:"groups,omitempty"`
}

func (httpModel *HTTPModel) ToModel() *Model {
	model := &Model{
		Schema: httpModel.Schema,
	}

	for _, g := range httpModel.Groups {
		if model.Groups == nil {
			model.Groups = map[string]*GroupModel{}
		}
		newG := &GroupModel{
			Plural:   g.Plural,
			Singular: g.Singular,
			Schema:   g.Schema,
		}
		model.Groups[newG.Plural] = newG

		for _, r := range g.Resources {
			if newG.Resources == nil {
				newG.Resources = map[string]*ResourceModel{}
			}
			newR := &ResourceModel{
				Plural:    r.Plural,
				Singular:  r.Singular,
				Versions:  r.Versions,
				VersionId: r.VersionId,
				Latest:    r.Latest,
			}
			newG.Resources[newR.Plural] = newR
		}
	}

	return model
}

func ModelToHTTPModel(m *Model) *HTTPModel {
	httpModel := &HTTPModel{
		Schema: m.Schema,
	}

	// To ensure consistent - especially when diffing the output
	for _, groupKey := range SortedKeys(m.Groups) {
		group := m.Groups[groupKey]
		newG := HTTPGroupModel{
			Plural:   group.Plural,
			Singular: group.Singular,
			Schema:   group.Schema,
		}

		for _, resKey := range SortedKeys(group.Resources) {
			resource := group.Resources[resKey]
			newR := HTTPResourceModel{
				Plural:    resource.Plural,
				Singular:  resource.Singular,
				Versions:  resource.Versions,
				VersionId: resource.VersionId,
				Latest:    resource.Latest,
			}
			newG.Resources = append(newG.Resources, newR)
		}

		httpModel.Groups = append(httpModel.Groups, newG)
	}

	return httpModel
}

func HTTPPUTModel(w io.Writer, info *RequestInfo) error {
	if len(info.Parts) > 1 {
		info.ErrCode = http.StatusNotFound
		return fmt.Errorf("Not found")
	}

	reqBody, err := io.ReadAll(info.OriginalRequest.Body)
	if err != nil {
		info.ErrCode = http.StatusInternalServerError
		return err
	}

	tmpModel := HTTPModel{}
	err = json.Unmarshal(reqBody, &tmpModel)
	if err != nil {
		info.ErrCode = http.StatusInternalServerError
		return err
	}

	model := tmpModel.ToModel()
	if err != nil {
		info.ErrCode = http.StatusInternalServerError
		return err
	}

	err = info.Registry.Model.ApplyNewModel(model)
	if err != nil {
		info.ErrCode = http.StatusBadRequest
		return err
	}

	return HTTPGETModel(w, info)
}

func LoadHTTPProps(info *RequestInfo) (map[string]any, error) {
	var registry = info.Registry
	var entity *Entity
	var group *Group
	var resource *Resource
	var version *Version
	var realID string
	var banned = []string{}
	var err error

	if info.What == "Coll" {
		info.ErrCode = http.StatusBadRequest
		return nil, fmt.Errorf("Can't update a collection(%s) directly",
			info.Abstract)
	}

	if info.What == "Registry" {
		entity = &registry.Entity
		entity.Refresh()

		for _, gModel := range registry.Model.Groups {
			banned = append(banned, gModel.Plural)
			banned = append(banned, gModel.Singular)
		}
	} else {
		// GROUP
		group, err = registry.FindGroup(info.GroupType, info.GroupUID)
		if err == nil && group == nil {
			group, err = registry.AddGroup(info.GroupType, NewUUID())
		}
		if err != nil {
			info.ErrCode = http.StatusBadRequest
			return nil, fmt.Errorf("Error processing group: %s", err.Error())
		}
		entity = &group.Entity

		for _, rModel := range registry.Model.Groups[info.GroupType].Resources {
			banned = append(banned, rModel.Plural)
			banned = append(banned, rModel.Singular)
		}

		// RESOURCE
		if info.ResourceUID != "" {
			resource, err = group.FindResource(info.ResourceType, info.ResourceUID)
			if err != nil {
				info.ErrCode = http.StatusBadRequest
				return nil, fmt.Errorf("Error processing resource: %s", err.Error())
			}
			verUID := info.VersionUID
			if resource != nil {
				if verUID != "" {
					version, err = resource.FindVersion(verUID)
					if err != nil {
						info.ErrCode = http.StatusBadRequest
						return nil, fmt.Errorf("Error processing version: %s", err.Error())
					}
					if version == nil {
						version, err = resource.AddVersion(verUID)
					}
					if err != nil {
						info.ErrCode = http.StatusBadRequest
						return nil, fmt.Errorf("Error processing version: %s", err.Error())
					}
					entity = &version.Entity
					realID = version.UID
				} else {
					version, err = resource.GetLatest()
					if err != nil {
						info.ErrCode = http.StatusBadRequest
						return nil, fmt.Errorf("Error processing version: %s", err.Error())
					}
				}
			} else {
				if verUID == "" {
					verUID = NewUUID()
				}

				resource, err = group.AddResource(info.ResourceType, info.ResourceUID, verUID)
				if err != nil {
					info.ErrCode = http.StatusBadRequest
					return nil, fmt.Errorf("Error processing resource: %s", err.Error())
				}
				version, err = resource.GetLatest()
				if err != nil {
					info.ErrCode = http.StatusBadRequest
					return nil, fmt.Errorf("Error processing version: %s", err.Error())
				}
				if info.VersionUID == "" {
					entity = &resource.Entity
					realID = resource.UID
				} else {
					entity = &version.Entity
					realID = version.UID
				}
			}

			if err != nil {
				info.ErrCode = http.StatusBadRequest
				return nil, fmt.Errorf("Error processing resource: %s", err.Error())
			}
			if resource == nil {
				info.ErrCode = http.StatusNotFound
				return nil, fmt.Errorf("Not found")
			}
			entity = &resource.Entity
			realID = resource.UID

			if info.VersionUID != "" {
				version, err = resource.FindVersion(info.VersionUID)
				if err != nil {
					info.ErrCode = http.StatusBadRequest
					return nil, fmt.Errorf("Error processing version: %s", err.Error())
				}
				if version == nil {
					info.ErrCode = http.StatusNotFound
					return nil, fmt.Errorf("Not found")
				}
				entity = &version.Entity
				realID = version.UID

			}
		} else {
			// rUID := NewUUID()
			// vUID := NewUUID()
		}

	}

	log.Printf("Entity:\n%s\n", ToJSON(entity))

	body, err := io.ReadAll(info.OriginalRequest.Body)
	if err != nil {
		return nil, err
	}

	props := map[string]any{}

	if err = json.Unmarshal(body, &props); err != nil {
		return nil, err
	}

	for k, v := range props {
		log.Printf("%s => %v", k, v)
		if k == "id" {
			if v != realID {
				return nil,
					fmt.Errorf("`id` in Body (%s )doesn't match actual ID(%s)",
						v, realID)
			}
			continue
		}
		SetProp(entity, k, v)
	}

	return props, nil
}
