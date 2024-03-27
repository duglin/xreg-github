package registry

import (
	"bytes"
	// "encoding/base64"
	// "encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
)

type Server struct {
	Port       int
	HTTPServer *http.Server
}

var DefaultRegDbSID string

func GetDefaultReg(tx *Tx) *Registry {
	if tx == nil {
		tx = NewTx()
	}

	reg, err := FindRegistryBySID(tx, DefaultRegDbSID)
	Must(err)

	if reg != nil {
		tx.Registry = reg
	}

	return reg
}

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
	var info *RequestInfo
	var err error

	// Don't bother with a Tx for this test flow
	if strings.HasPrefix(r.URL.Path, "/EMPTY") {
		tmp := fmt.Sprintf("hello%s", r.URL.Path[6:])
		w.Write([]byte(tmp))
		return
	}

	tx := NewTx()

	defer func() {
		// As of now we should never have more than one active Tx during
		// testing
		if os.Getenv("TESTING") != "" {
			l := len(TXs)
			if (tx.tx == nil && l > 0) || (tx.tx != nil && l > 1) {
				log.Printf(">End of HTTP Request")
				DumpTXs()

				// Info can be nil in the /EMPTY cases
				log.Printf("Info: %s", ToJSON(info))
				log.Printf("<Exit http req")

				panic("nested Txs")
			}
		}

		// Explicit Commit() is required, else we'll always rollback
		tx.Rollback()
	}()

	if DefaultRegDbSID == "" {
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

	info, err = ParseRequest(tx, w, r)

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

	if info.ResourceModel != nil && info.ResourceModel.HasDocument == false &&
		info.ShowMeta {
		info.StatusCode = http.StatusBadRequest
		err = fmt.Errorf("Specifying \"?meta\" for a Resource that has the " +
			"model \"hasdocument\" value set to \"false\" is invalid")
	}

	if err == nil {
		// These should only return an error if they didn't already
		// send a response back to the client.
		switch strings.ToUpper(r.Method) {
		case "GET":
			err = HTTPGet(info)
		case "PUT":
			err = HTTPPutPost(info)
		case "POST":
			err = HTTPPutPost(info)
		case "DELETE":
			err = HTTPDelete(info)
		default:
			info.StatusCode = http.StatusMethodNotAllowed
			err = fmt.Errorf("HTTP method %q not supported", r.Method)
		}
	}

	Must(tx.Conditional(err))

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

	format := info.OriginalRequest.URL.Query().Get("schema")
	if format == "" {
		format = "xRegistry-json"
	}

	model := info.Registry.Model
	if model == nil {
		model = &Model{}
	}

	ms := GetModelSerializer(format)
	if ms == nil {
		return fmt.Errorf("Unsupported schema format: %s", format)
	}
	buf, err := ms(model, format)
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

	results, err := Query(info.tx, query, args...)
	defer results.Close()

	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return err
	}

	entity, err := readNextEntity(info.tx, results)
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
			v, err := readNextEntity(info.tx, results)
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
		if attr.internals.httpHeader != "" {
			headerName = attr.internals.httpHeader
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

	metaInBody := (info.ResourceModel == nil) ||
		(info.ResourceModel.HasDocument == false || info.ShowMeta)

	if info.What == "Entity" && info.ResourceUID != "" && !metaInBody { // !info.ShowMeta {
		return HTTPGETContent(info)
	}

	query, args, err := GenerateQuery(info)
	results, err := Query(info.tx, query, args...)
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
		if jw.hasData {
			// Add a tailing \n if there's any data, else skip it
			jw.Print("\n")
		}
	} else {
		info.StatusCode = http.StatusInternalServerError
	}

	return err
}

var attrHeaders = map[string]*Attribute{}

func init() {
	// Load-up the attributes that have custom http header names
	for _, attr := range OrderedSpecProps {
		if attr.internals.httpHeader != "" {
			attrHeaders[strings.ToLower(attr.internals.httpHeader)] = attr
		}
	}
}

func HTTPPutPost(info *RequestInfo) error {
	method := strings.ToUpper(info.OriginalRequest.Method)
	isNew := false

	metaInBody := (info.ResourceModel == nil) ||
		(info.ResourceModel.HasDocument == false || info.ShowMeta)

	log.VPrintf(3, "HTTPPutPost: %s %s", method, info.OriginalPath)

	info.Root = strings.Trim(info.Root, "/")

	// The model has its own special func
	if len(info.Parts) > 0 && info.Parts[0] == "model" {
		return HTTPPUTModel(info)
	}

	// POST /groups/gID/resources/rID?setlatestversiond is special
	if len(info.Parts) == 4 && method == "POST" {
		if _, ok := info.OriginalRequest.URL.Query()["setlatestversionid"]; ok {
			return HTTPSetLatestVersionID(info)
		}
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

	// PUT/POST /GROUPs/gID/RESOURCEs... + ReadOnly Resource
	if info.ResourceModel != nil && info.ResourceModel.ReadOnly &&
		(method == "PUT" || method == "POST") {
		// Note that we only block it for end-user interactions, like via
		// HTTP. If people try to change it via the internal APIs, then
		// we don't stop it yet. Not sure if we should. TODO

		info.StatusCode = http.StatusMethodNotAllowed
		return fmt.Errorf("Write operations to read-only resources are not " +
			"allowed")
	}

	// Ok, now start to del with the incoming request
	/////////////////////////////////////////////////

	// First lets get the Resource's model to get the RESOURCE 'singular'
	resSingular := ""
	if info.ResourceModel != nil {
		resSingular = info.ResourceModel.Singular
	}

	// Get the incoming Object either from the body or from xRegistry headers
	IncomingObj, err := ExtractIncomingObject(info, resSingular)
	if err != nil {
		return err
	}

	// ID should be in the body so grab it for later use
	propsID := ""
	if v, ok := IncomingObj["id"]; ok {
		if reflect.ValueOf(v).Kind() == reflect.String {
			propsID = NotNilString(&v)
		}
	}

	// Walk the PATH and process things
	///////////////////////////////////

	// URL: /
	// ////////////////////////////////////////////////////////////////
	if len(info.Parts) == 0 {
		// MUST be PUT /

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
		// Use the ID from the body if present
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
		if len(info.Parts) > 2 {
			// Incoming object isn't for the Group
			group, err = info.Registry.AddGroup(info.GroupType, groupUID)
		} else {
			group, err = info.Registry.AddGroup(info.GroupType, groupUID,
				IncomingObj)
		}

		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf("Error processing group(%s): %s", groupUID, err)
		}
	}

	if len(info.Parts) < 3 {
		// Either /GROUPs or /GROUPs/gID

		if !isNew {
			// Didn't create a new one, so update existing Group
			group.NewObject = IncomingObj

			if err = group.Entity.ValidateAndSave(isNew); err != nil {
				info.StatusCode = http.StatusBadRequest
				return fmt.Errorf("Error processing group: %s", err)
			}
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

	// URL: /GROUPs/gID/RESOURCEs...
	// ////////////////////////////////////////////////////////////////

	// Some global vars
	resource := (*Resource)(nil)
	version := (*Version)(nil)
	resourceUID := info.ResourceUID
	versionUID := info.VersionUID

	isResource := false // is entity we're pointing to a Resource or not?
	isResourceNew := false

	// Do Resources and Versions at the same time
	// URL: /GROUPs/gID/RESOURCEs
	// URL: /GROUPs/gID/RESOURCEs/rID
	// URL: /GROUPs/gID/RESOURCEs/rID/versions[/vID]
	// ////////////////////////////////////////////////////////////////

	// This assumes that in the end we'll be dealing with a Version.
	// If it's new then IncomingObj will be used during the create(), else
	// IncomingObj will be used for an update in common code after all of the
	// "if" statements

	if len(info.Parts) == 3 {
		// GROUPs/gID/RESOURCEs - must be POST

		// No ID provided so generate one
		if resourceUID = propsID; resourceUID == "" {
			resourceUID = NewUUID()
		}

		isNew = true
		isResourceNew = true
		isResource = true

		delete(IncomingObj, "id") // id is for Res not Version so remove it

		// Create a new Resource and it's first/only/latest Version
		resource, err = group.AddResource(info.ResourceType, resourceUID,
			versionUID, IncomingObj) // vID should be ""
		if err == nil {
			version, err = resource.GetLatest()
		}
	}

	if len(info.Parts) > 3 {
		// GROUPs/gID/RESOURCEs/rID...

		resource, err = group.FindResource(info.ResourceType, resourceUID)
		if err != nil {
			info.StatusCode = http.StatusInternalServerError
			return fmt.Errorf("Error finding resource(%s): %s", resourceUID, err)
		}
		// Must(err)
	}

	if len(info.Parts) == 4 && method == "PUT" {
		// PUT GROUPs/gID/RESOURCEs/rID

		if propsID != "" && propsID != resourceUID {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf("Metadata id(%s) doesn't match ID in "+
				"URL(%s)", propsID, resourceUID)
		}

		if resource != nil {
			version, err = resource.GetLatest()

			if !metaInBody { // !info.ShowMeta {
				// Copy existing props into IncomingObj w/o overwriting
				CopyNewProps(IncomingObj, version.Object)
			}
			isResource = true

			// Fall thru and we'll update the version later on, check err too
		}

		if resource == nil {
			// Create a new Resource and it's first/only/latest Version
			resource, err = group.AddResource(info.ResourceType, resourceUID,
				versionUID, IncomingObj) // vID is ""
			if err == nil {
				version, err = resource.GetLatest()
			}

			isNew = true
			isResourceNew = true
		}

		// Fall thru-we'll update the version with IncomingObj below & check err
	}

	if (len(info.Parts) == 4 && method == "POST") || len(info.Parts) == 5 {
		// POST GROUPs/gID/RESOURCEs/rID, POST GROUPs/gID/RESOURCEs/rID/versions

		if resource == nil {
			// Implicitly create the resource
			versionUID = propsID

			resource, err = group.AddResource(info.ResourceType, resourceUID,
				versionUID, IncomingObj) // no IncomingObj
			isNew = true
			isResourceNew = true
			if err == nil {
				version, err = resource.GetLatest()
			}
		} else {
			isNew = true
			versionUID = propsID
			version, err = resource.AddVersion(versionUID, false, IncomingObj)
		}

		// Fall thru and check err
	}

	if len(info.Parts) == 6 {
		// PUT GROUPs/gID/RESOURCEs/rID/versions/vID

		if resource == nil {
			// Implicitly create the resource
			resource, err = group.AddResource(info.ResourceType, resourceUID,
				versionUID, IncomingObj)
			isNew = true
			isResourceNew = true
		}

		if err == nil {
			version, err = resource.FindVersion(versionUID)
			if err != nil {
				info.StatusCode = http.StatusInternalServerError
				return fmt.Errorf("Error finding version(%s): %s", versionUID, err)
			}
			// Must(err)
		}

		if err == nil {
			if version == nil {
				// We have a Resource, so add a new Version based on IncomingObj
				version, err = resource.AddVersion(versionUID, false,
					IncomingObj)
				isNew = true
			} else {
				if !metaInBody { // !info.ShowMeta {
					CopyNewProps(IncomingObj, version.Object)
				}
			}
		}

		// Fall thru and check err
	}

	if err != nil || resource == nil {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf("Error processing resource(%s): %s", resourceUID, err)
	}

	if err != nil || version == nil {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf("Error processing version(%s): %s", versionUID, err)
	}

	Must(err) // Previous IFs should stop everything

	versionUID = version.UID

	// if the incoming entity is a Resource (not Version) then delete the
	// Versions collection stuff before continuing.
	if isResource {
		delete(IncomingObj, "versions")
		delete(IncomingObj, "versionscount")
		delete(IncomingObj, "versionsurl")
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
			if info.ResourceModel.SetLatest == false {
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

func HTTPSetLatestVersionID(info *RequestInfo) error {
	group, err := info.Registry.FindGroup(info.GroupType, info.GroupUID)
	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return fmt.Errorf("Error finding group(%s): %s", info.GroupUID, err)
	}
	if group == nil {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Group %q not found", info.GroupUID)
	}

	resource, err := group.FindResource(info.ResourceType, info.ResourceUID)
	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return fmt.Errorf("Error finding resource(%s): %s",
			info.ResourceUID, err)
	}
	if resource == nil {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Resource %q not found", info.ResourceUID)
	}

	if info.ResourceModel.SetLatest == false {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf(`Resource %q doesn't allow setting of `+
			`"latestversionid"`, info.ResourceModel.Plural)
	}

	vID := info.OriginalRequest.URL.Query().Get("setlatestversionid")
	if vID == "" {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf(`"setlatestversionid" must not be empty`)
	}

	version, err := resource.FindVersion(vID)
	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return fmt.Errorf("Error finding version(%s): %s", vID, err)
	}
	if version == nil {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Version %q not found", vID)
	}

	err = resource.SetLatest(version)
	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return fmt.Errorf("Error setting latest version: %s", err)
	}

	return HTTPGet(info)
}

func HTTPDelete(info *RequestInfo) error {
	// DELETE /...
	if len(info.Parts) == 0 {
		// DELETE /
		info.StatusCode = http.StatusMethodNotAllowed
		return fmt.Errorf("Can't delete an entire registry")
	}

	var err error
	epochStr := info.OriginalRequest.URL.Query().Get("epoch")
	epochInt := -1
	if epochStr != "" {
		epochInt, err = strconv.Atoi(epochStr)
		if err != nil || epochInt < 0 {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf("Epoch value %q must be an UINTEGER", epochStr)
		}
	}

	// DELETE /GROUPs...
	gm := info.Registry.Model.Groups[info.GroupType]
	if gm == nil {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Group type %q not found", info.GroupType)
	}

	if len(info.Parts) == 1 {
		// DELETE /GROUPs
		return HTTPDeleteGroups(info)
	}

	// DELETE /GROUPs/gID...
	group, err := info.Registry.FindGroup(info.GroupType, info.GroupUID)
	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return fmt.Errorf(`Error finding Group %q: %s`, info.GroupUID, err)
	}
	if group == nil {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf(`Group %q not found`, info.GroupUID)
	}

	if len(info.Parts) == 2 {
		// DELETE /GROUPs/gID
		if epochInt >= 0 {
			if e := group.Get("epoch"); e != epochInt {
				return fmt.Errorf(`Epoch value for %q must be %d`, group.UID, e)
			}
		}
		if err = group.Delete(); err != nil {
			info.StatusCode = http.StatusInternalServerError
			return fmt.Errorf(`Error deleting Group %q: %s`, info.GroupUID, err)
		}

		info.StatusCode = http.StatusNoContent
		return nil
	}

	// DELETE /GROUPs/gID/RESOURCEs...
	rm := gm.Resources[info.ResourceType]
	if rm == nil {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf(`Resource type %q not found`, info.ResourceType)
	}

	if len(info.Parts) == 3 {
		// DELETE /GROUPs/gID/RESOURCEs
		return HTTPDeleteResources(info)
	}

	// DELETE /GROUPs/gID/RESOURCEs/rID...
	resource, err := group.FindResource(info.ResourceType, info.ResourceUID)
	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return fmt.Errorf(`Error finding resource %q`, info.ResourceUID)
	}
	if resource == nil {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf(`Resource %q not found`, info.ResourceUID)
	}

	if len(info.Parts) == 4 {
		// DELETE /GROUPs/gID/RESOURCEs/rID
		if epochInt >= 0 {
			if e := resource.Get("epoch"); e != epochInt {
				return fmt.Errorf(`Epoch value for %q must be %d`,
					resource.UID, e)
			}
		}
		err = resource.Delete()

		if err != nil {
			info.StatusCode = http.StatusInternalServerError
			return fmt.Errorf(`Error deleting Resource %q: %s`,
				info.ResourceUID, err)
		}

		info.StatusCode = http.StatusNoContent
		return nil
	}

	if len(info.Parts) == 5 {
		// DELETE /GROUPs/gID/RESOURCEs/rID/versions
		return HTTPDeleteVersions(info)
	}

	// DELETE /GROUPs/gID/RESOURCEs/rID/versions/vID...
	version, err := resource.FindVersion(info.VersionUID)
	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return fmt.Errorf(`Error finding version %q`, info.VersionUID)
	}
	if version == nil {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf(`Version %q not found`, info.VersionUID)
	}

	if len(info.Parts) == 6 {
		// DELETE /GROUPs/gID/RESOURCEs/rID/versions/vID
		if epochInt >= 0 {
			if e := version.Get("epoch"); e != epochInt {
				return fmt.Errorf(`Epoch value for Version %q must be %d`,
					info.VersionUID, e)
			}
		}
		nextLatest := info.OriginalRequest.URL.Query().Get("setlatestversionid")
		err = version.Delete(nextLatest)

		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return err
		}

		info.StatusCode = http.StatusNoContent
		return nil
	}

	return fmt.Errorf("Bad API: %s", info.BaseURL)
}

type IDEntry struct {
	ID    string
	Epoch *int
}

type IDArray []IDEntry

func LoadIDList(info *RequestInfo) (IDArray, error) {
	list := IDArray{}

	body, err := io.ReadAll(info.OriginalRequest.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading body: %s", err)
	}

	bodyStr := strings.TrimSpace(string(body))

	if len(bodyStr) > 0 {
		err = Unmarshal([]byte(bodyStr), &list)
		if err != nil {
			return nil, err
		}
	} else {
		// IDArray == nil mean no list at all, not same as empty list
		return nil, nil
	}

	return list, nil
}

func HTTPDeleteGroups(info *RequestInfo) error {
	list, err := LoadIDList(info)
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return err
	}

	// No list provided so get list of Groups so we can delete them all
	if list == nil {
		results, err := Query(info.tx, `
			SELECT UID
			FROM Entities
			WHERE RegSID=? AND Abstract=?`,
			info.Registry.DbSID, info.GroupType)
		if err != nil {
			info.StatusCode = http.StatusInternalServerError
			return fmt.Errorf("Error getting the list: %s", err)
		}
		for row := results.NextRow(); row != nil; row = results.NextRow() {
			list = append(list, IDEntry{NotNilString(row[0]), nil})
		}
		defer results.Close()
	}

	// Delete each Group, checking epoch first if provided
	for _, entry := range list {
		group, err := info.Registry.FindGroup(info.GroupType, entry.ID)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf(`Error getting Group /%q`, entry.ID)
		}
		if group == nil {
			// Silently ignore the 404
			continue
			/*
				info.StatusCode = http.StatusNotFound
				return fmt.Errorf(`Group %q not found`, entry.ID)
			*/
		}

		if entry.Epoch != nil {
			if group.Get("epoch") != *(entry.Epoch) {
				info.StatusCode = http.StatusBadRequest
				return fmt.Errorf(`Epoch value for %q must be %d`,
					entry.ID, group.Get("epoch"))
			}
		}

		err = group.Delete()
		if err != nil {
			info.StatusCode = http.StatusInternalServerError
			return fmt.Errorf(`Error deleting %q: %s`, entry.ID, err)
		}
	}

	info.StatusCode = http.StatusNoContent
	return nil
}

func HTTPDeleteResources(info *RequestInfo) error {
	list, err := LoadIDList(info)
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return err
	}

	// No list provided so get list of Resources so we can delete them all
	if list == nil {
		results, err := Query(info.tx, `
			SELECT UID
			FROM Entities
			WHERE RegSID=? AND Abstract=?`,
			info.Registry.DbSID,
			NewPPP(info.GroupType).P(info.ResourceType).Abstract())
		if err != nil {
			info.StatusCode = http.StatusInternalServerError
			return fmt.Errorf("Error getting the list: %s", err)
		}
		for row := results.NextRow(); row != nil; row = results.NextRow() {
			list = append(list, IDEntry{NotNilString(row[0]), nil})
		}
		defer results.Close()
	}

	group, err := info.Registry.FindGroup(info.GroupType, info.GroupUID)
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf(`Error getting Group %q`, info.GroupUID)
	}

	// Delete each Resource, checking epoch first if provided
	for _, entry := range list {
		resource, err := group.FindResource(info.ResourceType, entry.ID)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf(`Error getting Resource %q`, entry.ID)
		}
		if resource == nil {
			// Silently ignore the 404
			continue
			/*
				info.StatusCode = http.StatusNotFound
				return fmt.Errorf(`Resource %q not found`, entry.ID)
			*/
		}

		if entry.Epoch != nil {
			if resource.Get("epoch") != *(entry.Epoch) {
				info.StatusCode = http.StatusBadRequest
				return fmt.Errorf(`Epoch value for %q must be %d`,
					entry.ID, resource.Get("epoch"))
			}
		}

		err = resource.Delete()
		if err != nil {
			info.StatusCode = http.StatusInternalServerError
			return fmt.Errorf(`Error deleting %q: %s`, entry.ID, err)
		}
	}

	info.StatusCode = http.StatusNoContent
	return nil
}

func HTTPDeleteVersions(info *RequestInfo) error {
	nextLatest := info.OriginalRequest.URL.Query().Get("setlatestversionid")

	list, err := LoadIDList(info)
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return err
	}

	// No list provided so get list of Versions so we can delete them all
	if list == nil {
		results, err := Query(info.tx, `
			SELECT UID
			FROM Entities
			WHERE RegSID=? AND Abstract=?`,
			info.Registry.DbSID,
			NewPPP(info.GroupType).P(info.ResourceType).P("versions").Abstract())
		if err != nil {
			info.StatusCode = http.StatusInternalServerError
			return fmt.Errorf("Error getting the list: %s", err)
		}
		for row := results.NextRow(); row != nil; row = results.NextRow() {
			list = append(list, IDEntry{NotNilString(row[0]), nil})
		}
		defer results.Close()
	}

	group, err := info.Registry.FindGroup(info.GroupType, info.GroupUID)
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf(`Error getting Group %q: %s`, info.GroupUID, err)
	}

	resource, err := group.FindResource(info.ResourceType, info.ResourceUID)
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf(`Error getting Resource %q: %s`,
			info.ResourceUID, err)
	}

	// Delete each Version, checking epoch first if provided
	for _, entry := range list {
		version, err := resource.FindVersion(entry.ID)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf(`Error getting Version %q: %s`, entry.ID, err)
		}
		if version == nil {
			// Silently ignore the 404
			continue
			/*
				info.StatusCode = http.StatusNotFound
				return fmt.Errorf(`Version %q not found`, entry.ID)
			*/
		}

		if entry.Epoch != nil {
			if version.Get("epoch") != *(entry.Epoch) {
				info.StatusCode = http.StatusBadRequest
				return fmt.Errorf(`Epoch value for %q must be %d`,
					entry.ID, version.Get("epoch"))
			}
		}

		err = version.Delete(nextLatest)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return err
		}
	}

	if nextLatest != "" {
		version, err := resource.FindVersion(nextLatest)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf(`Error getting Version %q: %s`, nextLatest, err)
		}
		err = resource.SetLatest(version)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return err
		}
	}

	info.StatusCode = http.StatusNoContent
	return nil
}

func ExtractIncomingObject(info *RequestInfo, resSingular string) (Object, error) {
	IncomingObj := map[string]any{}

	// Load-up the body
	// //////////////////////////////////////////////////////
	body, err := io.ReadAll(info.OriginalRequest.Body)
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return nil, fmt.Errorf("Error reading body: %s", err)
	}
	if len(body) == 0 {
		body = nil
	}

	metaInBody := (info.ShowMeta ||
		(info.ResourceModel != nil && info.ResourceModel.HasDocument == false))

	if len(info.Parts) < 3 || metaInBody { // info.ShowMeta { // body != nil {
		for k, _ := range info.OriginalRequest.Header {
			k := strings.ToLower(k)
			if strings.HasPrefix(k, "xregistry-") {
				info.StatusCode = http.StatusBadRequest
				if info.ResourceModel.HasDocument == false {
					return nil, fmt.Errorf("Including \"xRegistry\" headers " +
						"for a Resource that has the model \"hasdocument\" " +
						"value of \"false\" is invalid")
				}
				return nil, fmt.Errorf("Including \"xRegistry\" headers " +
					"when \"?meta\" is used is invalid")
			}
		}

		if strings.TrimSpace(string(body)) == "" {
			body = []byte("{}") // Be forgiving
		}

		// err = json.Unmarshal(body, &IncomingObj)
		err = Unmarshal(body, &IncomingObj)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return nil, err
		}
	}

	// xReg metadata are in headers, so move them into IncomingObj. We'll
	// copy over the existing properties latest once we knwo what entity
	// we're dealing with
	if len(info.Parts) > 2 && !metaInBody { // !info.ShowMeta {
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
				return nil, fmt.Errorf("'xRegistry-%s' isn't allowed as "+
					"an HTTP header", key)
			}

			if key == resSingular+"url" || key == resSingular+"proxyurl" {
				if len(body) != 0 {
					return nil, fmt.Errorf("'xRegistry-%s' isn't allowed "+
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
							return nil, fmt.Errorf("HTTP header %q should "+
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
						// delete(IncomingObj, key)
						IncomingObj[key] = nil
					}
				} else {
					IncomingObj[key] = val
				}
			}
		}
	}

	return IncomingObj, nil
}

func CopyNewProps(tgt Object, from Object) {
	// Copy all keys from "from" if there isn't that key in "tgt" already
	for k, v := range from {
		if _, ok := tgt[k]; !ok {
			tgt[k] = v
		}
	}

	/*
		// for each key in tgt that has a value of "nil" delete it
		nilKeys := []string{}

		for k, v := range tgt {
			if IsNil(v) {
				nilKeys = append(nilKeys, k)
			}
		}

		for _, k := range nilKeys {
			delete(tgt, k)
		}
	*/
}
