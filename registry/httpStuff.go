package registry

import (
	"bytes"
	// "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
)

type Server struct {
	Port       int
	HTTPServer *http.Server
}

var DefaultReg *Registry

func NewServer(port int) *Server {
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
	if strings.HasPrefix(r.URL.Path, "/EMPTY") {
		tmp := fmt.Sprintf("hello%s", r.URL.Path[6:])
		w.Write([]byte(tmp))
		return
	}

	if DefaultReg == nil {
		panic("No registry specified")
	}

	saveVerbose := log.GetVerbose()
	if tmp := r.URL.Query().Get("verbose"); tmp != "" {
		if v, err := strconv.Atoi(tmp); err == nil {
			log.SetVerbose(v)
		}
		defer log.SetVerbose(saveVerbose)
	}

	log.VPrintf(2, "%s %s", r.Method, r.URL)

	info, err := ParseRequest(w, r)
	if err != nil {
		w.WriteHeader(info.StatusCode)
		w.Write([]byte(fmt.Sprintf("%s\n", err.Error())))
		return
	}

	defer func() {
		// If we haven't written anything, this will force the HTTP status code
		// to be written and not default to 200
		info.HTTPWriter.Done()
	}()

	if r.URL.Query().Has("reg") { // Wrap in html page
		info.HTTPWriter = NewPageWriter(info)
	}

	if r.URL.Query().Has("html") || r.URL.Query().Has("noprops") { //HTMLify it
		info.HTTPWriter = NewBufferedWriter(info)
	}

	// These should only return an error if they didn't already
	// send a response back to the client.
	switch strings.ToUpper(r.Method) {
	case "GET":
		err = HTTPGet(info)
	case "PUT":
		err = HTTPPutPost(info)
	case "POST":
		err = HTTPPutPost(info)
	default:
		info.StatusCode = http.StatusMethodNotAllowed
		err = fmt.Errorf("HTTP method %q not supported", r.Method)
	}

	if err != nil {
		if info.StatusCode == 0 {
			// Only default to BadRequest if not set by someone else
			info.StatusCode = http.StatusBadRequest
		}
		info.Write([]byte(err.Error() + "\n"))
	}

}

type HTTPWriter interface {
	Write([]byte) (int, error)
	AddHeader(string, string)
	Done()
}

var _ HTTPWriter = &DefaultWriter{}
var _ HTTPWriter = &BufferedWriter{}
var _ HTTPWriter = &DiscardWriter{}
var _ HTTPWriter = &PageWriter{}

func DefaultHTTPWriter(info *RequestInfo) HTTPWriter {
	return &DefaultWriter{
		Info: info,
	}
}

type DefaultWriter struct {
	Info *RequestInfo
}

func (dw *DefaultWriter) Write(b []byte) (int, error) {
	if !dw.Info.SentStatus {
		dw.Info.SentStatus = true
		if dw.Info.StatusCode == 0 {
			dw.Info.StatusCode = http.StatusOK
		}
		dw.Info.OriginalResponse.WriteHeader(dw.Info.StatusCode)
	}
	return dw.Info.OriginalResponse.Write(b)
}

func (dw *DefaultWriter) AddHeader(name, value string) {
	dw.Info.OriginalResponse.Header()[name] = []string{value}
}

func (dw *DefaultWriter) Done() {
	dw.Write(nil)
}

type BufferedWriter struct {
	Info      *RequestInfo
	OldWriter HTTPWriter
	Headers   *map[string]string
	Buffer    *bytes.Buffer
}

func NewBufferedWriter(info *RequestInfo) *BufferedWriter {
	return &BufferedWriter{
		Info:      info,
		OldWriter: info.HTTPWriter,
		Headers:   &map[string]string{},
		Buffer:    &bytes.Buffer{},
	}
}

func (bw *BufferedWriter) Write(b []byte) (int, error) {
	return bw.Buffer.Write(b)
}

func (bw *BufferedWriter) AddHeader(name, value string) {
	(*bw.Headers)[name] = value
}

func (bw *BufferedWriter) Done() {
	req := bw.Info.OriginalRequest
	if req.URL.Query().Has("html") {
		// Override content-type
		bw.AddHeader("Content-Type", "text/html")
	}

	for k, v := range *bw.Headers {
		bw.OldWriter.AddHeader(k, v)
	}

	buf := bw.Buffer.Bytes()
	/*
		if req.URL.Query().Has("noprops") {
			buf = RemoveProps(buf)
		}
		if req.URL.Query().Has("oneline") {
			buf = OneLine(buf)
		}
	*/
	if req.URL.Query().Has("html") {
		bw.OldWriter.Write([]byte("<pre>\n"))
		buf = HTMLify(req, buf)
	}
	bw.OldWriter.Write(buf)
}

type DiscardWriter struct{}

func (dw *DiscardWriter) Write(b []byte) (int, error)  { return len(b), nil }
func (dw *DiscardWriter) AddHeader(name, value string) {}
func (dw *DiscardWriter) Done()                        {}

var DefaultDiscardWriter = &DiscardWriter{}

type PageWriter struct {
	Info      *RequestInfo
	OldWriter HTTPWriter
	Headers   *map[string]string
	Buffer    *bytes.Buffer
}

func NewPageWriter(info *RequestInfo) *PageWriter {
	return &PageWriter{
		Info:      info,
		OldWriter: info.HTTPWriter,
		Headers:   &map[string]string{},
		Buffer:    &bytes.Buffer{},
	}
}

func (pw *PageWriter) Write(b []byte) (int, error) {
	return pw.Buffer.Write(b)
}

func (pw *PageWriter) AddHeader(name, value string) {
	(*pw.Headers)[name] = value
}

func (pw *PageWriter) Done() {
	pw.AddHeader("Content-Type", "text/html")

	if !pw.Info.SentStatus {
		pw.Info.SentStatus = true
		if pw.Info.StatusCode == 0 {
			pw.Info.StatusCode = http.StatusOK
		}
		pw.Info.OriginalResponse.WriteHeader(pw.Info.StatusCode)
	}

	for k, v := range *pw.Headers {
		pw.OldWriter.AddHeader(k, v)
	}

	buf := pw.Buffer.Bytes()

	list := ""
	list += fmt.Sprintf("<li><a href='/?reg'>Default</a></li>\n")
	for _, name := range GetRegistryNames() {
		list += fmt.Sprintf("<li><a href='/reg-%s?reg'>%s</a></li>\n",
			name, name)
	}

	pw.OldWriter.Write([]byte(fmt.Sprintf(`<html>
<style>
  form {
    display: inline ;
  }
  body {
    display: flex ;
    flex-direction: row ;
    flex-wrap: nowrap ;
    justify-content: flex-start ;
    height: 100%% ;
    margin: 0 ;
  }
  #left {
    padding: 8 20 8 8 ;
    background-color: lightsteelblue;
    white-space: nowrap ;
  }
  #right {
    display: flex ;
    flex-direction: column ;
    flex-wrap: nowrap ;
    justify-content: flex-start ;
    width: 100%% ;

  }
  #url {
    background-color: lightgray;
    border: 0px ;
    display: flex ;
    flex-direction: row ;
    align-items: center ;
    padding: 5px ;
    margin: 0px ;
  }
  #myURL {
    width: 40em ;
  }
  button {
    margin-left: 5px ;
  }
  #myOutput {
    background-color: ghostwhite;
    border: 0px ;
	padding: 5px ;
    flex: 1 ;
	overflow: auto ;
  }
  pre {
    margin: 0px ;
  }
  li {
    white-space: nowrap ;
    cursor: pointer ;
  }
</style>
<div id=left>
  <b>Choose a registry:</b>
  <br><br>
  `+list+`
</div>

<div id=right>
	<!--
    <form id=url onsubmit="go();return false;">
      <div style="margin:0 5 0 10">URL:</div>
      <input id=myURL type=text>
      <button type=submit> Go! </button>
    </form>
	-->
  <div id=myOutput>
    <pre>%s</pre>
  </div>
</div>
`, RegHTMLify(pw.Info.OriginalRequest, buf))))

	pw.OldWriter.Done()
}

func HTTPGETModel(info *RequestInfo) error {
	if len(info.Parts) > 1 {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Not found")
	}

	model := info.Registry.Model
	if model == nil {
		model = &Model{}
	}

	buf, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return err
	}

	info.AddHeader("Content-Type", "application/json")
	info.Write(buf)
	info.Write([]byte("\n"))
	return nil
}

func HTTPGETContent(info *RequestInfo) error {
	log.VPrintf(3, ">Enter: HTTPGetContent")
	defer log.VPrintf(3, "<Exit: HTTPGetContent")
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
		info.StatusCode = http.StatusInternalServerError
		return err
	}

	entity, err := readNextEntity(results)
	log.VPrintf(3, "Entity: %#v", entity)
	if entity == nil {
		info.StatusCode = http.StatusNotFound
		if err != nil {
			log.Printf("Error loading entity: %s", err)
			return fmt.Errorf("Error finding entity: %s", err)
		} else {
			return fmt.Errorf("Not found")
		}
	}

	var version *Entity
	versionsCount := 0
	if info.VersionUID == "" {
		// We're on a Resource, so go find the latest Version and count
		// how many versions there are for the VersionsCount attribute
		vID := entity.Get("latestversionid").(string)
		for {
			v, err := readNextEntity(results)
			if v == nil && version == nil {
				info.StatusCode = http.StatusInternalServerError
				return fmt.Errorf("Can't find version: %s : %s", vID, err)
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

	log.VPrintf(3, "Version: %#v", version)

	headerIt := func(e *Entity, info *RequestInfo, key string, val any, attr *Attribute) error {
		if key[0] == '#' {
			return nil
		}

		if attr.Type == MAP && IsScalar(attr.Item.Type) {
			for name, value := range val.(map[string]any) {
				info.AddHeader("xRegistry-"+key+"-"+name,
					fmt.Sprintf("%v", value))
			}
			return nil
		}

		if !IsScalar(attr.Type) {
			return nil
		}

		var headerName string
		if attr.httpHeader != "" {
			headerName = attr.httpHeader
		} else {
			headerName = "xRegistry-" + key
		}

		str := fmt.Sprintf("%v", val)
		info.AddHeader(headerName, str)

		return nil
	}

	err = entity.SerializeProps(info, headerIt)
	if err != nil {
		panic(err)
	}

	if info.VersionUID == "" {
		info.AddHeader("xRegistry-versionscount",
			fmt.Sprintf("%d", versionsCount))
		info.AddHeader("xRegistry-versionsurl",
			info.BaseURL+"/"+entity.Path+"/versions")
	}
	info.AddHeader("Content-Location", info.BaseURL+"/"+version.Path)

	url := ""
	if val := entity.Get("#resourceURL"); val != nil {
		gModel := info.Registry.Model.Groups[info.GroupType]
		rModel := gModel.Resources[info.ResourceType]
		singular := rModel.Singular

		url = val.(string)
		info.AddHeader("xRegistry-"+singular+"url", url)

		if info.StatusCode == 0 {
			// If we set it during a PUT/POST, don't override the 201
			info.StatusCode = http.StatusSeeOther
			info.AddHeader("Location", url)
		}
		/*
			http.Redirect(info.OriginalResponse, info.OriginalRequest, url,
				http.StatusSeeOther)
		*/
		return nil
	}

	if val := entity.Get("#resourceProxyURL"); val != nil {
		url = val.(string)
	}

	log.VPrintf(3, "#resourceProxyURL: %s", url)
	if url != "" {
		// Just act as a proxy and copy the remote resource as our response
		resp, err := http.Get(url)
		if err != nil {
			info.StatusCode = http.StatusInternalServerError
			return err
		}
		if resp.StatusCode/100 != 2 {
			info.StatusCode = resp.StatusCode
			return fmt.Errorf("Remote error")
		}

		// Copy all HTTP headers
		for header, value := range resp.Header {
			info.AddHeader(header, strings.Join(value, ","))
		}

		// Now copy the body
		_, err = io.Copy(info, resp.Body)
		if err != nil {
			info.StatusCode = http.StatusInternalServerError
			return err
		}
		return nil
	}

	buf := version.Get("#resource")
	if buf == nil {
		// No data so just return
		/*
			if info.StatusCode == 0 {
				info.StatusCode = http.StatusNoContent
			}
		*/
		return nil
	}
	info.Write(buf.([]byte))

	return nil
}

func HTTPGet(info *RequestInfo) error {
	log.VPrintf(3, ">Enter: HTTPGet(%s)", info.What)
	defer log.VPrintf(3, "<Exit: HTTPGet(%s)", info.What)

	info.Root = strings.Trim(info.Root, "/")

	if len(info.Parts) > 0 && info.Parts[0] == "model" {
		return HTTPGETModel(info)
	}

	if info.What == "Entity" && info.ResourceUID != "" && !info.ShowMeta {
		return HTTPGETContent(info)
	}

	query, args, err := GenerateQuery(info)
	results, err := Query(query, args...)
	defer results.Close()

	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return err
	}

	jw := NewJsonWriter(info, results)
	jw.NextEntity()

	if info.What != "Coll" {
		// Collections will need to print the {}, so don't error for them
		if jw.Entity == nil {
			info.StatusCode = http.StatusNotFound
			return fmt.Errorf("Not found")
		}
	}

	info.AddHeader("Content-Type", "application/json")
	if info.What == "Coll" {
		_, err = jw.WriteCollection()
	} else {
		err = jw.WriteEntity()
	}

	if err == nil {
		jw.Print("\n")
	} else {
		info.StatusCode = http.StatusInternalServerError
	}

	return err
}

var attrHeaders = map[string]*Attribute{}

func init() {
	// Load-up the attributes that have custom http header names
	for _, attr := range SpecProps {
		if attr.httpHeader != "" {
			attrHeaders[strings.ToLower(attr.httpHeader)] = attr
		}
	}
}

func HTTPPutPost(info *RequestInfo) error {
	method := strings.ToUpper(info.OriginalRequest.Method)
	IncomingObj := map[string]any{}
	propsID := "" // ID from body or http header
	isNew := false
	isResourceNew := false

	log.VPrintf(3, "HTTPPutPost: %s %s", method, info.OriginalPath)

	info.Root = strings.Trim(info.Root, "/")

	// The model has its own special func
	if len(info.Parts) > 0 && info.Parts[0] == "model" {
		return HTTPPUTModel(info)
	}

	// Check for some obvious high-level bad states up-front
	// //////////////////////////////////////////////////////
	if len(info.Parts) == 0 && method == "POST" {
		info.StatusCode = http.StatusMethodNotAllowed
		return fmt.Errorf("POST not allowed on the root of the registry")
	}

	if info.What == "Coll" && method == "PUT" {
		info.StatusCode = http.StatusMethodNotAllowed
		return fmt.Errorf("PUT not allowed on collections")
	}

	if len(info.Parts) == 2 && method == "POST" {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf("POST not allowed on a group")
	}

	if len(info.Parts) == 6 && method == "POST" {
		info.StatusCode = http.StatusMethodNotAllowed
		return fmt.Errorf("POST not allowed on a version")
	}

	// Load-up the body
	// //////////////////////////////////////////////////////
	body, err := io.ReadAll(info.OriginalRequest.Body)
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf("Error reading body: %s", err)
	}
	if len(body) == 0 {
		body = nil
	}

	// Reg or Group or res/ver+?meta, so parse body as Object
	// //////////////////////////////////////////////////////
	if len(info.Parts) < 3 || info.ShowMeta {
		if strings.TrimSpace(string(body)) == "" {
			body = []byte("{}") // Be forgiving
		}

		// err = json.Unmarshal(body, &IncomingObj)
		err = Unmarshal(body, &IncomingObj)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf("Error parsing body: %s", err)
		}

		// ID should be in the body so grab it for later use
		tmp := IncomingObj["id"]
		if reflect.ValueOf(tmp).Kind() == reflect.String {
			propsID = NotNilString(&tmp)
		}
	} else {
		propsID = info.OriginalRequest.Header.Get("xRegistry-ID")
	}

	// URL: /
	// ////////////////////////////////////////////////////////////////
	if len(info.Parts) == 0 {
		// MUST be PUT / - do PUT

		info.Registry.Entity.NewObject = IncomingObj

		if err = info.Registry.Entity.ValidateAndSave(false); err != nil {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf("Error processing registry: %s", err)
		}

		info.Parts = []string{}
		info.What = "Registry"
		return HTTPGet(info)
	}

	// URL: /GROUPs[/gID]...
	// ////////////////////////////////////////////////////////////////
	group := (*Group)(nil)
	groupUID := info.GroupUID
	if len(info.Parts) == 1 {
		// must be POST /GROUPs
		if groupUID = propsID; groupUID == "" {
			groupUID = NewUUID()
		}
	} else {
		// must be PUT/POST /GROUPs/gID...
		group, err = info.Registry.FindGroup(info.GroupType, groupUID)

		if err != nil {
			info.StatusCode = http.StatusInternalServerError
			return fmt.Errorf("Error processing group(%s): %s", groupUID, err)
		}
	}

	if group == nil {
		// Group doesn't exist so create it
		isNew = true
		group, err = info.Registry.AddGroup(info.GroupType, groupUID)

		if err != nil {
			info.StatusCode = http.StatusInternalServerError
			return fmt.Errorf("Error processing group(%s): %s", groupUID, err)
		}
	}

	if len(info.Parts) < 3 {
		// Either /GROUPs or /GROUPs/gID - do PUT
		group.NewObject = IncomingObj

		if err = group.Entity.ValidateAndSave(isNew); err != nil {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf("Error processing group: %s", err)
		}

		info.Parts = []string{info.Parts[0], groupUID}
		info.What = "Entity"
		info.GroupUID = groupUID

		if isNew { // 201, else let it default to 200
			info.AddHeader("Location", info.BaseURL+"/"+group.Path)
			info.StatusCode = http.StatusCreated
		}

		return HTTPGet(info)
	}

	// Do Resources and Versions at the same time
	// URL: /GROUPs/gID/RESOURCEs
	// URL: /GROUPs/gID/RESOURCEs/rID
	// URL: /GROUPs/gID/RESOURCEs/rID/versions[/vID]
	// ////////////////////////////////////////////////////////////////

	var resourceModel *ResourceModel
	resSingular := ""

	if info.GroupType != "" {
		groupModel := info.Registry.Model.Groups[info.GroupType]

		if info.ResourceType != "" {
			resourceModel = groupModel.Resources[info.ResourceType]
			resSingular = resourceModel.Singular
		}
	}

	resource := (*Resource)(nil)
	version := (*Version)(nil)
	resourceUID := info.ResourceUID
	versionUID := info.VersionUID

	if len(info.Parts) == 3 {
		// must be: POST /GROUPs/gID/RESOURCEs
		if resourceUID = propsID; resourceUID == "" {
			resourceUID = NewUUID()
		}
		isNew = true
		isResourceNew = true
		// Create a new Resource and it's first/only/latest Version
		resource, err = group.AddResource(info.ResourceType, resourceUID,
			versionUID) // vID should be ""
		if err == nil {
			version, err = resource.GetLatest()
		}
	} else {
		// must be PUT/POST /GROUPs/gID/RESOURCEs/rID...

		// Check metadata ID == ID in URL, only if doing a resource+PUT.
		// Check here because later on we'll replace id with the version's
		// ID and won't be able to check it in updateentity
		if len(info.Parts) == 4 && method == "PUT" &&
			propsID != "" && propsID != resourceUID {

			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf("Metadata id(%s) doesn't match ID in "+
				"URL(%s)", propsID, resourceUID)
		}

		resource, err = group.FindResource(info.ResourceType, resourceUID)
		if err != nil {
			info.StatusCode = http.StatusInternalServerError
			return fmt.Errorf("Error processing resource(%s): %s", resourceUID, err)
		}

		if resource == nil {
			if versionUID == "" &&
				((len(info.Parts) == 4 && method == "POST") ||
					(len(info.Parts) == 5)) {

				// No vID in URL, grab from props. If missing, auto-generate
				versionUID = propsID
			}

			isNew = true
			isResourceNew = true
			// Create a new Resource and it's first/only/latest Version
			resource, err = group.AddResource(info.ResourceType, resourceUID,
				versionUID)
			if err == nil {
				version, err = resource.GetLatest()
			}
		}
	}

	if err != nil || resource == nil {
		info.StatusCode = http.StatusInternalServerError
		return fmt.Errorf("Error processing resource(%s): %s", resourceUID, err)
	}

	// No version means the resource already existed, find/create version
	if version == nil {
		if versionUID == "" && len(info.Parts) >= 4 && method == "POST" {
			// must be: POST /groups/gID/resources/rID
			//      or: POST /groups/gID/resources/rID/versions
			// so any ID provided is for the Version.
			// The len part of the "if" is probably not necessary
			versionUID = propsID
		}

		if versionUID != "" {
			// must be: XXX /groups/gID/resources/rID/versions/vID
			version, err = resource.FindVersion(versionUID)
		}

		if err == nil && version == nil {
			/* Commented out and made the "== 4" into a ">= 4" above
			// Use the ID from the entity only if the URL pointed to a
			// version and not the resource
			if versionUID == "" && len(info.Parts) == 5 {
				versionUID = propsID
			}
			*/
			if len(info.Parts) == 4 && versionUID == "" && method == "PUT" {
				// must be: PUT /groups/gID/resources/rID
				// which means we're updating the latest Version, so get it.
				// the versionUID == "" is probably not necessary
				version, err = resource.GetLatest()
			} else {
				// else, we were asked to create a new Version.
				// Don't set "latest" on the resource yet, we'll do that later
				isNew = true
				version, err = resource.AddVersion(versionUID, false)
			}
		}
	}

	if err != nil || version == nil {
		info.StatusCode = http.StatusInternalServerError
		return fmt.Errorf("Error processing version(%s): %s", versionUID, err)
	}

	isResource := false
	versionUID = version.UID
	currObj := (map[string]any)(nil)

	if len(info.Parts) >= 5 || (len(info.Parts) == 4 && method == "POST") {
		// entity is a version not a resource
		currObj = version.Object // Materialize(info)
	} else {
		// entity is a resource not a version
		currObj = resource.Materialize(info)
		isResource = true
	}

	if !info.ShowMeta {
		// xReg metadata are in headers, so apply them as a patch over currObj
		maps.Copy(IncomingObj, currObj)

		// IncomingObj["#resource"] = body // save new body
		IncomingObj[resSingular] = body // save new body

		seenMaps := map[string]bool{}

		for name, attr := range attrHeaders {
			// TODO we may need some kind of "delete if missing" flag on
			// each httpHeader attribute since some may want to have an
			// explicit 'null' to be erased instead of just missing (eg patch)
			if val := info.OriginalRequest.Header.Get(name); val != "" {
				IncomingObj[attr.Name] = val
			} else {
				IncomingObj[attr.Name] = nil
			}
		}

		for key, value := range info.OriginalRequest.Header {
			key := strings.ToLower(key)

			if !strings.HasPrefix(key, "xregistry-") {
				continue
			}

			key = strings.TrimSpace(key[10:]) // remove xRegistry-
			if key == "" {
				continue
			}

			if key == resSingular || key == resSingular+"base64" {
				return fmt.Errorf("'xRegistry-%s' isn't allowed as an HTTP "+
					"header", key)
			}

			if key == resSingular+"url" || key == resSingular+"proxyurl" {
				if len(body) != 0 {
					return fmt.Errorf("'xRegistry-%s' isn't allowed "+
						"if there's a body", key)
				}

				delete(IncomingObj, "#resourceProxyURL")
				delete(IncomingObj, "#resourceURL")
				delete(IncomingObj, "#resource")
			}

			val := any(value[0])
			if val == "null" {
				val = nil
			}

			// If there are -'s then it's a non-scalar, convert it.
			// Note that any "-" after the 1st is part of the key name
			// labels-keyName && labels-"key-name"
			parts := strings.SplitN(key, "-", 2)
			if len(parts) > 1 {
				obj := IncomingObj

				if _, ok := seenMaps[parts[0]]; !ok {
					// First time we've seen this map, delete old stuff
					delete(IncomingObj, parts[0])
					seenMaps[parts[0]] = true
				}

				for i, part := range parts {
					if i+1 == len(parts) {
						obj[part] = val
						continue
					}

					prop, ok := obj[part]
					if !ok {
						if val == nil {
							break
						}
						tmpO := map[string]any{}
						obj[part] = tmpO
						obj = map[string]any(tmpO)
					} else {
						obj, ok = prop.(map[string]any)
						if !ok {
							return fmt.Errorf("HTTP header %q should "+
								"reference a map", key)
						}
					}
				}
			} else {
				if IsNil(val) {
					if _, ok := seenMaps[key]; ok {
						// Do nothing if we've seen keys for this map already.
						// We don't want to erase any keys we just added.
						// This is an edge/error? case where someone included
						// xReg-label:null AND xreg-label-foo:foo - keep "foo"
					} else {
						delete(IncomingObj, key)
					}
				} else {
					IncomingObj[key] = val
				}
			}
		}
	} else {
		// if the incoming entity is a Resource (not Version) then delete the
		// Versions collection stuff before continuing.
		if isResource {
			delete(IncomingObj, "versions")
			delete(IncomingObj, "versionscount")
			delete(IncomingObj, "versionsurl")
		}
	}

	IncomingObj["id"] = version.UID
	delete(IncomingObj, "latestversionid")
	delete(IncomingObj, "latestversionurl")

	version.NewObject = IncomingObj
	version.ConvertStrings()

	// If "latest" is in the incoming msg then see if we can do what they ask.
	// Else set current version to the latest if it's new
	setLatest := isNew
	if latestVal, ok := IncomingObj["latest"]; ok {
		latest, ok := latestVal.(bool)
		if !ok {
			return fmt.Errorf(`"latest" must be a boolean`)
		}
		if isResourceNew {
			if latest == false {
				return fmt.Errorf(`"latest" can not be "false" since ` +
					`there is only one version, so it must be the latest`)
			}
		} else {
			latestVID := resource.Get("latestversionid").(string)
			if err != nil {
				return err
			}
			if latest == false {
				if version.UID == latestVID {
					return fmt.Errorf(`"latest" can not be "false" since ` +
						`doing so would result in no latest version`)
				}
			}

			// If the user can't control "latest", but they passed in
			// what we were going to do anyway, let it pass
			if resourceModel.Latest == false {
				if latest != (latestVID == version.UID) {
					return fmt.Errorf(`"latest" can not be "%v", it is `+
						`controlled by the server`, latest)
				}
			} else {
				// Ok, let them control it
				setLatest = latest
			}
		}
	}
	if setLatest {
		resource.SetLatest(version)
	}

	if err = version.ValidateAndSave(isNew); err != nil {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf("Error processing resource: %s", err)
	}

	originalLen := len(info.Parts)

	info.Parts = []string{info.Parts[0], groupUID,
		info.Parts[2], resourceUID}
	info.What = "Entity"
	info.GroupUID = groupUID
	info.ResourceUID = resourceUID

	location := info.BaseURL + "/" + resource.Path
	// location := resource.Path
	if originalLen > 4 || (originalLen == 4 && method == "POST") {
		info.Parts = append(info.Parts, "versions", versionUID)
		info.VersionUID = versionUID
		location += "/versions/" + info.VersionUID
		// location = version.Path
	}

	if info.ShowMeta { // not 100% sure this the right way/spot
		location += "?meta"
	}

	if isNew { // 201, else let it default to 200
		info.AddHeader("Location", location)
		info.StatusCode = http.StatusCreated
	}

	return HTTPGet(info)
}

func HTTPPUTModel(info *RequestInfo) error {
	if len(info.Parts) > 1 {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Not found")
	}

	reqBody, err := io.ReadAll(info.OriginalRequest.Body)
	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return err
	}

	model := Model{}
	// err = json.Unmarshal(reqBody, &model)
	err = Unmarshal(reqBody, &model)
	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return err
	}

	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return err
	}

	err = info.Registry.Model.ApplyNewModel(&model)
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return err
	}

	return HTTPGETModel(info)
}
