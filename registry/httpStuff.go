package registry

import (
	"bytes"
	// "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	// "os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/duglin/dlog"
)

type Server struct {
	Port       int
	HTTPServer *http.Server
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

	tx, err := NewTx()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error talking to DB, try again later\n"))
		return
	}

	defer func() {
		// As of now we should never have more than one active Tx during
		// testing
		if TESTING {
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

	if r.URL.Query().Has("ui") { // Wrap in html page
		info.HTTPWriter = NewPageWriter(info)
	}

	if r.URL.Query().Has("html") || r.URL.Query().Has("noprops") { //HTMLify it
		info.HTTPWriter = NewBufferedWriter(info)
	}

	if info.ResourceModel != nil && info.ResourceModel.GetHasDocument() == false &&
		info.ShowStructure {
		info.StatusCode = http.StatusBadRequest
		err = fmt.Errorf("Specifying \"$structure\" for a Resource that has " +
			"the model \"hasdocument\" value set to \"false\" is invalid")
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
		case "PATCH":
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
	list += fmt.Sprintf("<li><a href='/?ui'>Default</a></li>\n")
	for _, name := range GetRegistryNames() {
		list += fmt.Sprintf("  <li><a href='/reg-%s?ui'>%s</a></li>\n",
			name, name)
	}

	filters := ""
	prefix := MustPropPathFromPath(pw.Info.Abstract).UI()
	if prefix != "" {
		prefix += string(UX_IN)
	}
	for _, arrayF := range pw.Info.Filters {
		if filters != "" {
			filters += "\n"
		}
		subF := ""
		for _, FE := range arrayF {
			if subF != "" {
				subF += ","
			}
			next := MustPropPathFromDB(FE.Path).UI()
			next, _ = strings.CutPrefix(next, prefix)
			subF += next
			if FE.HasEqual {
				subF += "=" + FE.Value
			}
		}
		filters += subF
	}

	inlineOptions := []string{}
	if len(pw.Info.Parts) == 0 {
		inlineOptions = GetRegistryModelInlines(pw.Info.Registry.Model)
	} else if len(pw.Info.Parts) <= 2 {
		inlineOptions = GetGroupModelInlines(pw.Info.GroupModel)
	} else if len(pw.Info.Parts) <= 4 {
		inlineOptions = GetResourceModelInlines(pw.Info.ResourceModel)
	} else {
		inlineOptions = GetVersionModelInlines(pw.Info.ResourceModel)
	}

	checked := ""
	inlines := ""
	inlineCount := 0

	// ----

	if len(pw.Info.Parts) == 0 {
		if pw.Info.IsInlineSet(NewPPP("capabilities").DB()) {
			checked = " checked"
		}
		inlines += fmt.Sprintf(`
    <div class=inlines>
      <input id=inline%d type='checkbox' value='capabilities'`+
			checked+`/>capabilities
    </div>`, inlineCount)
		inlineCount++
		checked = ""
	}

	// ----

	if len(pw.Info.Parts) == 0 {
		if pw.Info.IsInlineSet(NewPPP("model").DB()) {
			checked = " checked"
		}
		inlines += fmt.Sprintf(`
    <div class=inlines>
      <input id=inline%d type='checkbox' value='model'`+checked+`/>model
    </div>`, inlineCount)
		inlineCount++
		checked = ""
	}

	// ----

	if pw.Info.IsInlineSet(NewPPP("*").DB()) {
		checked = " checked"
	}
	except := ""
	if len(pw.Info.Parts) == 0 {
		except = " but model,capabilities"
	}
	inlines += fmt.Sprintf(`
    <div class=inlines>
      <input id=inline%d type='checkbox' value='*'`+checked+`/>* (all%s)
    </div>`, inlineCount, except)
	inlineCount++
	checked = ""

	// ----

	pp, _ := PropPathFromPath(pw.Info.Abstract)
	for _, inline := range inlineOptions {
		checked = ""
		pInline := MustPropPathFromUI(inline)
		fullInline := pp.Append(pInline).DB()
		if pw.Info.IsInlineSet(fullInline) {
			checked = " checked"
		}
		inlines += fmt.Sprintf(`
    <div class=inlines>
      <input id=inline%d type='checkbox' value='%s'%s/>%s
    </div>`, inlineCount, inline, checked, inline)
		inlineCount++
	}

	tmp := pw.Info.BaseURL
	urlPath := fmt.Sprintf(`<a href="%s?ui">%s</a>`, tmp, tmp)
	for _, p := range pw.Info.Parts {
		tmp += "/" + p
		urlPath += fmt.Sprintf(`/<a href="%s?ui">%s</a>`, tmp, p)
	}

	structureswitch := ""
	structuretext := ""
	structureButton := ""
	if pw.Info.ShowStructure {
		structureswitch = "true"
		structuretext = "Show document"
		urlPath += fmt.Sprintf(`$<a href="%s$structure?ui">structure</a>`, tmp)
	} else {
		structureswitch = "false"
		structuretext = "Show structure"
	}
	if pw.Info.ResourceUID != "" && pw.Info.What == "Entity" {
		structureButton = fmt.Sprintf(`
    <div><button id=structure onclick='structureswitch=!structureswitch ; apply()'>%s</button></div>
`, structuretext)
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
    overflow-y: auto ;
    min-width: fit-content ;
  }
  #right {
    display: flex ;
    flex-direction: column ;
    flex-wrap: nowrap ;
    justify-content: flex-start ;
    width: 100%% ;
    overflow: auto ;
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
    // margin-left: 5px ;
  }
  #buttonList {
    display: flex ;
    flex-direction: column ;
    align-items: start ;
  }
  #buttonBar {
    background-color: lightsteelblue;
    display: flex ;
    flex-direction: column ;
    align-items: start ;
    padding: 2px ;
  }
  #structure {
    display: inline ;
    margin-bottom: 10px ;
  }
  #commit {
    font-size: 12px ;
    font-family: courier ;
    position: fixed ;
    bottom: 0 ;
	z-index: 0 ;
  }
  textarea {
    margin-bottom: 10px ;
    min-width: 100%% ;
  }
  #filters {
    display: block ;
    min-height: 8em ;
    font-size: 12px ;
    font-family: courier ;
    width: 100%%
  }
  .inlines {
    font-size: 13px ;
    font-family: courier ;
  }
  #urlPath {
    background-color: lightgray ;
    padding: 3px ;
    font-size: 16px ;
    font-family: courier ;
    border-bottom: 4px solid lightsteelblue ;
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
    list-style-type: circle ;
  }
</style>
<div id=left>
  <b>Registries:</b>
  <br>
  `+list+`
  <hr style="width:100%% ; margin-top:15px ; margin-bottom:15px">
  <div id=buttonList>
    <b>Filters:</b>
    <textarea id=filters>`+filters+`</textarea>
    <b>Inlines:</b>`+inlines+`
    <hr style="width:100%% ; margin-top:15px ; margin-bottom:15px">
    <div style="display:ruby">
      <button onclick="apply()">Apply</button>`+
		structureButton+`
    </div>
  </div>
  <div style="height:12px"></div> <!-- buffer for "Commmit:" line -->
  <div id=commit><a target=_blank
        href="http://github.com/duglin/xreg-github/tree/`+
		GitCommit+`">Commit: `+fmt.Sprintf("%.12s", GitCommit)+`</a></div>
</div>

<script>

var structureswitch = `+structureswitch+`;

function apply() {
  var loc = "`+pw.Info.BaseURL+`/`+strings.Join(pw.Info.Parts, "/")+`"

  if (structureswitch) loc += "$structure"
  loc += "?ui"

  var filters = document.getElementById("filters").value
  var lines = filters.split("\n")
  for (var i = 0 ; i < lines.length ; i++ ) {
    if (lines[i] != "") {
      loc += "&filter=" + lines[i]
    }
  }

  for (var i = 0 ; ; i++ ) {
    var box = document.getElementById("inline"+i)
    if (box == null) { break }
    if (box.checked) {
      loc += "&inline=" + box.value
    }
  }

  window.location = loc
}
</script>

<div id=right>
    <!--
    <form id=url onsubmit="go();return false;">
      <div style="margin:0 5 0 10">URL:</div>
      <input id=myURL type=text>
      <button type=submit> Go! </button>
    </form>
    -->
  <div id=urlPath>
  <b>Path:</b> `+urlPath+`
  </div>
  <div id=myOutput>
    <pre>%s</pre>
  </div>
  <!-- <div id=buttonBar>%s</div> -->
</div>
`, RegHTMLify(pw.Info.OriginalRequest, buf))))

	pw.OldWriter.Done()
}

func GetRegistryModelInlines(m *Model) []string {
	res := []string{}

	for _, gm := range m.Groups {
		res = append(res, gm.Plural)
		for _, inline := range GetGroupModelInlines(gm) {
			res = append(res, gm.Plural+"."+inline)
		}
	}

	sort.Strings(res)

	return res
}

func GetGroupModelInlines(gm *GroupModel) []string {
	res := []string{}

	for _, rm := range gm.Resources {
		res = append(res, rm.Plural)
		for _, inline := range GetResourceModelInlines(rm) {
			res = append(res, rm.Plural+"."+inline)
		}
	}
	return res
}

func GetResourceModelInlines(rm *ResourceModel) []string {
	res := []string{}

	if rm.GetHasDocument() {
		res = append(res, rm.Singular)
	}

	res = append(res, "meta")

	res = append(res, "versions")
	for _, inline := range GetVersionModelInlines(rm) {
		res = append(res, "versions."+inline)
	}

	return res
}

func GetVersionModelInlines(rm *ResourceModel) []string {
	return []string{rm.Singular}
}

func HTTPGETCapabilities(info *RequestInfo) error {
	if len(info.Parts) > 1 {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Not found")
	}

	cap := info.Registry.Capabilities
	capStr := info.Registry.GetAsString("#capabilities")
	if capStr != "" {
		var err error
		cap, err = ParseCapabilitiesJSON([]byte(capStr))
		Must(err)
	}

	buf, err := json.MarshalIndent(cap, "", "  ")
	if err != nil {
		return err
	}

	info.AddHeader("Content-Type", "application/json")
	info.Write(buf)
	info.Write([]byte("\n"))
	return nil
}

func HTTPGETModel(info *RequestInfo) error {
	if len(info.Parts) > 1 {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Not found")
	}

	format := info.GetFlag("schema")
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
  RegSID,Type,Plural,Singular,eSID,UID,PropName,PropValue,PropType,Path,Abstract
FROM FullTree WHERE RegSID=? AND `
	args := []any{info.Registry.DbSID}

	path := strings.Join(info.Parts, "/")

	// TODO consider excluding the META object from the query instead of
	// dropping it via the if-statement below in the versioncount logic
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
		// We're on a Resource, so go find the default Version and count
		// how many versions there are for the VersionsCount attribute
		group, err := info.Registry.FindGroup(info.GroupType, info.GroupUID, false)
		if err != nil {
			info.StatusCode = http.StatusInternalServerError
			return fmt.Errorf("Error finding group(%s): %s", info.GroupUID, err)
		}
		if group == nil {
			info.StatusCode = http.StatusNotFound
			return fmt.Errorf("Group %q not found", info.GroupUID)
		}

		resource, err := group.FindResource(info.ResourceType, info.ResourceUID, false)
		if err != nil {
			info.StatusCode = http.StatusInternalServerError
			return fmt.Errorf("Error finding resource(%s): %s",
				info.ResourceUID, err)
		}
		if resource == nil {
			info.StatusCode = http.StatusNotFound
			return fmt.Errorf("Resource %q not found", info.ResourceUID)
		}
		meta, err := resource.FindMeta(false)
		PanicIf(err != nil, "%s", err)

		vID := meta.GetAsString("defaultversionid")
		for {
			v, err := readNextEntity(info.tx, results)

			if v != nil && v.Type == ENTITY_META {
				// Skip the "meta" subobject
				continue
			}

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
	if url = entity.GetAsString("#resourceURL"); url != "" {
		gModel := info.Registry.Model.Groups[info.GroupType]
		rModel := gModel.Resources[info.ResourceType]
		singular := rModel.Singular

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

	url = entity.GetAsString("#resourceProxyURL")

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

	if len(info.Parts) > 0 && info.Parts[0] == "capabilities" {
		return HTTPGETCapabilities(info)
	}

	// 'metaInBody' tells us whether xReg metadata should be in the http
	// response body or not (meaning, the hasDoc doc)
	metaInBody := (info.ResourceModel == nil) ||
		(info.ResourceModel.GetHasDocument() == false || info.ShowStructure ||
			(len(info.Parts) == 5 && info.Parts[4] == "meta"))

	// Return the Resource's document
	if info.What == "Entity" && info.ResourceUID != "" && !metaInBody {
		return HTTPGETContent(info)
	}

	// Serialize the xReg metadata
	return SerializeQuery(info, []string{strings.Join(info.Parts, "/")},
		info.What, info.Filters)
}

func SerializeQuery(info *RequestInfo, paths []string, what string,
	filters [][]*FilterExpr) error {
	start := time.Now()

	defer func() {
		if log.GetVerbose() > 3 {
			diff := time.Now().Sub(start).Truncate(time.Millisecond)
			log.Printf("  Total Time: %s", diff)
		}
	}()

	query, args, err := GenerateQuery(info.Registry, what, paths, filters)
	results, err := Query(info.tx, query, args...)
	defer results.Close()

	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return err
	}

	if log.GetVerbose() > 3 {
		log.Printf("SerializeQuery: %s", SubQuery(query, args))
		diff := time.Now().Sub(start).Truncate(time.Millisecond)
		log.Printf("  Query: # results: %d (time: %s)",
			len(results.AllRows), diff)
	}

	jw := NewJsonWriter(info, results)
	jw.NextEntity()

	// Collections will need to print the {}, so don't error for them
	if what != "Coll" {
		if jw.Entity == nil {
			info.StatusCode = http.StatusNotFound
			return fmt.Errorf("Not found")
		}
	}

	// Special case, if we're doing a collection, let's make sure we didn't
	// get an empty result due to it's parent not even existing - for example
	// the user used the wrong case (or even name) in the parent's Path
	if jw.Entity == nil && len(info.Parts) > 1 {
		path := strings.Join(info.Parts[:len(info.Parts)-1], "/")
		entity, err := RawEntityFromPath(info.tx, info.Registry.DbSID, path,
			false)
		if err != nil {
			info.StatusCode = http.StatusInternalServerError
			return fmt.Errorf("Error finding parent(%s): %s", path, err)
		}
		if IsNil(entity) {
			info.StatusCode = http.StatusNotFound
			return fmt.Errorf("Not found")
		}
	}

	info.AddHeader("Content-Type", "application/json")
	if what == "Coll" {
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
	paths := ([]string)(nil)
	what := "Entity"

	metaInBody := (info.ResourceModel == nil) ||
		(info.ResourceModel.GetHasDocument() == false || info.ShowStructure ||
			(len(info.Parts) == 5 && info.Parts[4] == "meta"))

	log.VPrintf(3, "HTTPPutPost: %s %s", method, info.OriginalPath)

	info.Root = strings.Trim(info.Root, "/")

	// Capabilities has its own special func
	if len(info.Parts) > 0 && info.Parts[0] == "capabilities" {
		return HTTPPUTCapabilities(info)
	}

	// The model has its own special func
	if len(info.Parts) > 0 && info.Parts[0] == "model" {
		return HTTPPUTModel(info)
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

	// POST /groups/gID/resources/rID?setdefaultversiond is special in that
	// it only moves the "default" pointer, nothing else is meant to be done
	if metaInBody && len(info.Parts) == 4 && method == "POST" && body == nil {
		if info.HasFlag("setdefaultversionid") {
			return HTTPSetDefaultVersionID(info)
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

	if info.What == "Coll" && method == "PATCH" {
		info.StatusCode = http.StatusMethodNotAllowed
		return fmt.Errorf("PATCH not allowed on collections")
	}

	if len(info.Parts) == 2 && method == "POST" {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf("POST not allowed on a group")
	}

	if len(info.Parts) >= 5 && info.Parts[4] == "meta" && method == "POST" {
		info.StatusCode = http.StatusMethodNotAllowed
		return fmt.Errorf("POST not allowed on a 'meta'")
	}

	if len(info.Parts) == 6 && method == "POST" {
		info.StatusCode = http.StatusMethodNotAllowed
		return fmt.Errorf("POST not allowed on a version")
	}

	if !metaInBody && method == "PATCH" {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf("PATCH is not allowed on Resource documents")
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

	// Ok, now start to deal with the incoming request
	//////////////////////////////////////////////////

	// Get the incoming Object either from the body or from xRegistry headers
	IncomingObj, err := ExtractIncomingObject(info, body)
	if err != nil {
		return err
	}

	// Walk the PATH and process things
	///////////////////////////////////

	// URL: /
	// ////////////////////////////////////////////////////////////////
	if len(info.Parts) == 0 {
		// PUT /

		err = ConvertRegistryContents(IncomingObj, info.Registry.Model)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return err
		}

		addType := ADD_UPDATE
		if method == "PATCH" {
			addType = ADD_PATCH
		}
		err = info.Registry.Update(IncomingObj, addType, info.HasNested)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return err
		}

		// Return HTTP GET of Registry root
		return SerializeQuery(info, []string{""}, "Registry", info.Filters)
	}

	// URL: /GROUPs[/gID]...
	// ////////////////////////////////////////////////////////////////
	group := (*Group)(nil)
	groupUID := info.GroupUID

	if len(info.Parts) == 1 {
		// POST /GROUPs + body:map[id]Group

		objMap, err := IncomingObj2Map(IncomingObj)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return err
		}

		addType := ADD_UPSERT
		if method == "PATCH" {
			addType = ADD_PATCH
		}

		for id, obj := range objMap {
			err = ConvertGroupContents(obj, info.GroupModel)
			if err != nil {
				info.StatusCode = http.StatusBadRequest
				return err
			}

			g, _, err := info.Registry.UpsertGroupWithObject(info.GroupType,
				id, obj, addType, info.HasNested)
			if err != nil {
				info.StatusCode = http.StatusBadRequest
				return err
			}
			paths = append(paths, g.Path)
		}

		if len(paths) == 0 {
			paths = []string{"!"} // Force an empty collection to be returned
		}

		// Return HTTP GET of Groups created or updated
		return SerializeQuery(info, paths, "Coll", info.Filters)
	}

	if len(info.Parts) == 2 {
		// PUT /GROUPs/gID
		addType := ADD_UPSERT
		if method == "PATCH" {
			addType = ADD_PATCH
		}

		err = ConvertGroupContents(IncomingObj, info.GroupModel)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return err
		}

		group, isNew, err := info.Registry.UpsertGroupWithObject(info.GroupType,
			info.GroupUID, IncomingObj, addType, info.HasNested)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return err
		}

		if isNew { // 201, else let it default to 200
			info.AddHeader("Location", info.BaseURL+"/"+group.Path)
			info.StatusCode = http.StatusCreated
		}

		// Return HTTP GET of Group
		return SerializeQuery(info, []string{group.Path}, "Entity",
			info.Filters)
	}

	// Must be PUT/POST /GROUPs/gID/...

	// This will either find or create an empty Group as needed
	group, _, err = info.Registry.UpsertGroup(info.GroupType, groupUID)
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return err
	}

	// URL: /GROUPs/gID/RESOURCEs...
	// ////////////////////////////////////////////////////////////////

	// Some global vars
	resource := (*Resource)(nil)
	version := (*Version)(nil)
	resourceUID := info.ResourceUID
	versionUID := info.VersionUID

	// Do Resources and Versions at the same time
	// URL: /GROUPs/gID/RESOURCEs
	// URL: /GROUPs/gID/RESOURCEs/rID
	// URL: /GROUPs/gID/RESOURCEs/rID/versions[/vID]
	// ////////////////////////////////////////////////////////////////

	// If there isn't an explicit "return" then this assumes we're left with
	// a version and will return that back to the client

	if len(info.Parts) == 3 {
		// POST GROUPs/gID/RESOURCEs + body:map[id]Resource

		objMap, err := IncomingObj2Map(IncomingObj)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return err
		}

		// For each Resource in the map, upsert it and add it's path to result
		addType := ADD_UPSERT
		if method == "PATCH" {
			addType = ADD_PATCH
		}

		for id, obj := range objMap {
			err = ConvertResourceContents(obj, info.ResourceModel)
			if err != nil {
				info.StatusCode = http.StatusBadRequest
				return err
			}

			r, _, err := group.UpsertResourceWithObject(info.ResourceType,
				id, "", obj, addType, info.HasNested, false)
			if err != nil {
				info.StatusCode = http.StatusBadRequest
				return err
			}
			paths = append(paths, r.Path)
		}

		if len(paths) == 0 {
			paths = []string{"!"} // Force an empty collection to be returned
		}

		// Return HTTP GET of Resources created or modified
		return SerializeQuery(info, paths, "Coll", info.Filters)
	}

	if len(info.Parts) > 3 {
		// GROUPs/gID/RESOURCEs/rID...

		resource, err = group.FindResource(info.ResourceType, resourceUID,
			false)
		if err != nil {
			info.StatusCode = http.StatusInternalServerError
			return fmt.Errorf("Error finding resource(%s): %s", resourceUID,
				err)
		}
	}

	if len(info.Parts) == 4 && (method == "PUT" || method == "PATCH") {
		// PUT GROUPs/gID/RESOURCEs/rID [$structure]

		propsID := "" // RESOURCEid
		if v, ok := IncomingObj[info.ResourceModel.Singular+"id"]; ok {
			if reflect.ValueOf(v).Kind() == reflect.String {
				propsID = NotNilString(&v)
			}
		}

		if propsID != "" && propsID != resourceUID {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf("The \"%sid\" attribute must be set to %q, "+
				"not %q", info.ResourceModel.Singular, resourceUID, propsID)
		}

		err = ConvertResourceContents(IncomingObj, info.ResourceModel)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return err
		}

		if resource != nil {
			// version, err = resource.GetDefault()

			// ID needs to be the version's ID, not the Resources
			// IncomingObj["id"] = version.UID

			// Create a new Resource and it's first/only/default Version
			addType := ADD_UPSERT
			if method == "PATCH" || !metaInBody {
				addType = ADD_PATCH
			}
			resource, _, err = group.UpsertResourceWithObject(
				info.ResourceType, resourceUID, "" /*versionUID*/, IncomingObj,
				addType, info.HasNested, false)
			if err != nil {
				info.StatusCode = http.StatusBadRequest
				return err
			}

			version, err = resource.GetDefault()
		} else {
			// Upsert resource's default version
			delete(IncomingObj, info.ResourceModel.Singular+"id") // ID is the Resource's delete it
			addType := ADD_UPSERT
			if method == "PATCH" {
				addType = ADD_PATCH
			}
			resource, isNew, err = group.UpsertResourceWithObject(
				info.ResourceType, resourceUID, "" /*versionUID*/, IncomingObj,
				addType, info.HasNested, false)
			if err != nil {
				info.StatusCode = http.StatusBadRequest
				return err
			}

			version, err = resource.GetDefault()
		}
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return err
		}
	}

	if method == "POST" && len(info.Parts) == 4 {
		// POST GROUPs/gID/RESOURCEs/rID[$structure], body=obj or doc
		propsID := "" // versionid
		if v, ok := IncomingObj["versionid"]; ok {
			if reflect.ValueOf(v).Kind() == reflect.String {
				propsID = NotNilString(&v)
			}
		}

		err = ConvertVersionContents(IncomingObj, info.ResourceModel)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return err
		}

		if resource == nil {
			// Implicitly create the resource
			resource, isNew, err = group.UpsertResourceWithObject(
				info.ResourceType, resourceUID, propsID, IncomingObj,
				ADD_ADD, info.HasNested, true)
			if err != nil {
				info.StatusCode = http.StatusBadRequest
				return err
			}
			version, err = resource.GetDefault()
		} else {
			version, isNew, err = resource.UpsertVersionWithObject(propsID,
				IncomingObj, ADD_UPSERT)
		}
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return err
		}
		// Default to just returning the version
	}

	// GROUPs/gID/RESOURCEs/rID/meta
	if len(info.Parts) > 4 && info.Parts[4] == "meta" {
		// PUT /GROUPs/gID/RESOURCEs/rID/meta
		addType := ADD_UPSERT
		if method == "PATCH" {
			addType = ADD_PATCH
		}

		if resource == nil {
			isNew = true
			propsID := "" // RESOURCEid
			if v, ok := IncomingObj[info.ResourceModel.Singular+"id"]; ok {
				if reflect.ValueOf(v).Kind() == reflect.String {
					propsID = NotNilString(&v)
				}
			}

			if propsID != "" && propsID != resourceUID {
				info.StatusCode = http.StatusBadRequest
				return fmt.Errorf("The \"%sid\" attribute must be set to %q, "+
					"not %q", info.ResourceModel.Singular, resourceUID, propsID)
			}

			// Implicitly create the resource
			resource, _, err = group.UpsertResourceWithObject(
				// TODO check to see if "" should be propsID
				info.ResourceType, resourceUID, "", map[string]any{},
				ADD_ADD, false, false)
			if err != nil {
				info.StatusCode = http.StatusBadRequest
				return err
			}
		}

		// Technically, this will always "update" not "insert"
		meta, _, err := resource.UpsertMetaWithObject(IncomingObj, addType)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return err
		}

		// Return HTTP GET of 'meta'
		if isNew { // 201, else let it default to 200
			info.AddHeader("Location", info.BaseURL+"/"+meta.Path)
			info.StatusCode = http.StatusCreated
		}

		return SerializeQuery(info, []string{meta.Path}, "Entity", info.Filters)
	}

	// Just double-check
	if len(info.Parts) > 4 {
		PanicIf(info.Parts[4] != "versions", "Not 'versions': %s"+info.Parts[4])
	}

	// GROUPs/gID/RESOURCEs/rID/versions...

	if info.ShowStructure && method == "POST" && len(info.Parts) == 5 {
		// POST GROUPs/gID/RESOURCEs/rID/versions$structure - error
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf("Use of \"$structure\" on the \"versions\" " +
			" collection is not allowed")
	}

	if method == "POST" && len(info.Parts) == 5 {
		// POST GROUPs/gID/RESOURCEs/rID/versions, body=map[id]->Version

		// Convert IncomingObj to a map of Objects
		objMap, err := IncomingObj2Map(IncomingObj)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return err
		}

		for _, obj := range objMap {
			err = ConvertVersionContents(obj, info.ResourceModel)
			if err != nil {
				info.StatusCode = http.StatusBadRequest
				return err
			}
		}

		thisVersion := (*Version)(nil)

		if resource == nil {
			// Implicitly create the resource
			if len(objMap) == 0 {
				info.StatusCode = http.StatusBadRequest
				return fmt.Errorf("Set of Versions to add can't be empty")
			}

			vID := info.GetFlag("setdefaultversionid")
			if vID == "" || vID == "request" {
				if len(objMap) > 1 {
					info.StatusCode = http.StatusBadRequest
					if vID == "" {
						return fmt.Errorf("?setdefaultversionid is required")
					}
					return fmt.Errorf("?setdefaultversionid can not be " +
						"'request'")
				}
				// Only one Version so use its ID as the default version
				for k, _ := range objMap {
					vID = k
					break
				}
			}

			if vID == "null" {
				info.StatusCode = http.StatusBadRequest
				return fmt.Errorf("?setdefaultversionid can not be 'null'")
			}

			if IncomingObj, _ = objMap[vID]; IncomingObj == nil {
				info.StatusCode = http.StatusBadRequest
				return fmt.Errorf("Version %q not found", vID)
			}

			resource, err = group.AddResourceWithObject(info.ResourceType,
				resourceUID, vID, IncomingObj, info.HasNested, true)

			if err != nil {
				info.StatusCode = http.StatusBadRequest
				return err
			}

			v, err := resource.GetDefault()
			Must(err)
			thisVersion = v

			// Remove the newly created default version from objMap so we
			// won't process it again, but add it to the reuslts collection
			paths = append(paths, v.Path)
			delete(objMap, vID)
		}

		// Process the remaining versions
		for id, obj := range objMap {
			addType := ADD_UPSERT
			if method == "PATCH" {
				addType = ADD_PATCH
			}
			v, _, err := resource.UpsertVersionWithObject(id, obj, addType)
			if err != nil {
				info.StatusCode = http.StatusBadRequest
				return err
			}
			paths = append(paths, v.Path)
		}

		err = ProcessSetDefaultVersionIDFlag(info, resource, thisVersion)
		if err != nil {
			return err
		}

		if len(paths) == 0 {
			paths = []string{"!"} // Force an empty collection to be returned
		}
		return SerializeQuery(info, paths, "Coll", info.Filters)
	}

	if len(info.Parts) == 6 {
		// PUT GROUPs/gID/RESOURCEs/rID/versions/vID [$structure]
		propsID := "" //versionid
		if v, ok := IncomingObj["versionid"]; ok {
			if reflect.ValueOf(v).Kind() == reflect.String {
				propsID = NotNilString(&v)
			}
		}

		err = ConvertVersionContents(IncomingObj, info.ResourceModel)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return err
		}

		if resource == nil {
			// Implicitly create the resource
			resource, err = group.AddResourceWithObject(info.ResourceType,
				resourceUID, versionUID, IncomingObj, info.HasNested, true)
			if err != nil {
				info.StatusCode = http.StatusBadRequest
				return err
			}

			isNew = true
		}

		version, err = resource.FindVersion(versionUID, false)
		if err != nil {
			info.StatusCode = http.StatusInternalServerError
			return fmt.Errorf("Error finding version(%s): %s", versionUID,
				err)
		}

		if version == nil {
			// We have a Resource, so add a new Version based on IncomingObj
			version, isNew, err = resource.UpsertVersionWithObject(versionUID,
				IncomingObj, ADD_UPSERT)
		} else if !isNew {
			if propsID != "" && propsID != version.UID {
				info.StatusCode = http.StatusBadRequest
				return fmt.Errorf("The \"versionid\" attribute must be set "+
					"to %q, not %q", version.UID, propsID)
			}

			IncomingObj["versionid"] = version.UID
			addType := ADD_UPSERT
			if method == "PATCH" || !metaInBody {
				addType = ADD_PATCH
			}
			version, _, err = resource.UpsertVersionWithObject(version.UID,
				IncomingObj, addType)
		}
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return err
		}
	}

	PanicIf(err != nil, "err should be nil")

	// Process any ?setdefaultversionid query parameter there might be
	err = ProcessSetDefaultVersionIDFlag(info, resource, version)
	if err != nil {
		return err
	}

	originalLen := len(info.Parts)

	// Need to setup info stuff in case we call HTTPGetContent
	info.Parts = []string{info.Parts[0], groupUID,
		info.Parts[2], resourceUID}
	info.What = "Entity"
	info.GroupUID = groupUID
	info.ResourceUID = resourceUID // needed for $structure in URLs

	location := info.BaseURL + "/" + resource.Path
	if originalLen > 4 || (originalLen == 4 && method == "POST") {
		info.Parts = append(info.Parts, "versions", version.UID)
		info.VersionUID = version.UID
		location += "/versions/" + version.UID
	}

	if info.ShowStructure { // not 100% sure this the right way/spot
		location += "$structure"
	}

	if isNew { // 201, else let it default to 200
		info.AddHeader("Location", location)
		info.StatusCode = http.StatusCreated
	}

	// Return the contents of the entity instead of the xReg metadata
	if !metaInBody {
		return HTTPGETContent(info)
	}

	// Return the xReg metadata of the entity processed
	if paths == nil {
		paths = []string{strings.Join(info.Parts, "/")}
	}

	return SerializeQuery(info, paths, what, info.Filters)
}

func HTTPPUTCapabilities(info *RequestInfo) error {
	if len(info.Parts) > 1 {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Not found")
	}

	reqBody, err := io.ReadAll(info.OriginalRequest.Body)
	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return err
	}

	cap, err := ParseCapabilitiesJSON(reqBody)
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return err
	}

	err = cap.Validate()
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return err
	}

	info.Registry.SetSave("#capabilities", ToJSON(cap))

	return HTTPGETCapabilities(info)
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

// Process the ?setdefaultversionid query parameter
// "resource" is the resource we're processing
// "version" is the version that was processed
func ProcessSetDefaultVersionIDFlag(info *RequestInfo, resource *Resource, version *Version) error {
	vIDs := info.GetFlagValues("setdefaultversionid")
	if len(vIDs) == 0 {
		return nil
	}

	if info.ResourceModel.GetSetDefaultSticky() == false {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf(`Resource %q doesn't allow setting of `+
			`"defaultversionid"`, info.ResourceModel.Plural)
	}

	vID := vIDs[0]

	if vID == "" {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf(`"setdefaultversionid" must not be empty`)
	}

	// "null" and "request" have special meaning
	if vID == "null" {
		// Unstick the default version and go back to newest=default
		return resource.SetDefault(nil)
	}

	if vID == "request" {
		if version == nil {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf("Can't use 'request' if a version wasn't processed")
		}
		// stick default version to current one we just processed
		return resource.SetDefault(version)
	}

	version, err := resource.FindVersion(vID, false)
	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return fmt.Errorf("Error finding version(%s): %s", vID, err)
	}
	if version == nil {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf("Version %q not found", vID)
	}

	err = resource.SetDefault(version)
	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return fmt.Errorf("Error setting default version: %s", err)
	}

	return nil
}

func HTTPSetDefaultVersionID(info *RequestInfo) error {
	group, err := info.Registry.FindGroup(info.GroupType, info.GroupUID, false)
	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return fmt.Errorf("Error finding group(%s): %s", info.GroupUID, err)
	}
	if group == nil {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Group %q not found", info.GroupUID)
	}

	resource, err := group.FindResource(info.ResourceType, info.ResourceUID, false)
	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return fmt.Errorf("Error finding resource(%s): %s",
			info.ResourceUID, err)
	}
	if resource == nil {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Resource %q not found", info.ResourceUID)
	}

	err = ProcessSetDefaultVersionIDFlag(info, resource, nil)
	if err != nil {
		return err
	}

	return SerializeQuery(info, []string{resource.Path}, "Entity", info.Filters)
}

func HTTPDelete(info *RequestInfo) error {
	// DELETE /...
	if len(info.Parts) == 0 {
		// DELETE /
		info.StatusCode = http.StatusMethodNotAllowed
		return fmt.Errorf("Can't delete an entire registry")
	}

	var err error
	epochStr := info.GetFlag("epoch")
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
	group, err := info.Registry.FindGroup(info.GroupType, info.GroupUID, false)
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
				return fmt.Errorf(`Epoch value for %q must be %d not %d`,
					group.UID, e, epochInt)
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
	resource, err := group.FindResource(info.ResourceType, info.ResourceUID, false)
	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return fmt.Errorf(`Error finding resource %q`, info.ResourceUID)
	}
	if resource == nil {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf(`Resource %q not found`, info.ResourceUID)
	}

	meta, err := resource.FindMeta(false)
	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return fmt.Errorf(`Error finding resource %q: %s`,
			info.ResourceUID, err)
	}

	if len(info.Parts) == 4 {
		// DELETE /GROUPs/gID/RESOURCEs/rID
		if epochInt >= 0 {
			if e := meta.Get("epoch"); e != epochInt {
				return fmt.Errorf(`Epoch value for %q must be %d not %d`,
					resource.UID, e, epochInt)
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

	if len(info.Parts) == 5 && info.Parts[4] == "meta" {
		// DELETE /GROUPs/gID/RESOURCEs/rID/meta
		info.StatusCode = http.StatusMethodNotAllowed
		return fmt.Errorf(`DELETE is not allowed on a "meta"`)
	}

	if len(info.Parts) == 5 {
		// DELETE /GROUPs/gID/RESOURCEs/rID/versions
		return HTTPDeleteVersions(info)
	}

	// DELETE /GROUPs/gID/RESOURCEs/rID/versions/vID...
	version, err := resource.FindVersion(info.VersionUID, false)
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
				return fmt.Errorf(`Epoch value for %q must be %d not %d`,
					info.VersionUID, e, epochInt)
			}
		}
		nextDefault := info.GetFlag("setdefaultversionid")
		err = version.DeleteSetNextVersion(nextDefault)

		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return err
		}

		info.StatusCode = http.StatusNoContent
		return nil
	}

	return fmt.Errorf("Bad API: %s", info.BaseURL)
}

type EpochEntry map[string]any
type EpochEntryMap map[string]EpochEntry

func LoadEpochMap(info *RequestInfo) (EpochEntryMap, error) {
	res := EpochEntryMap{}

	body, err := io.ReadAll(info.OriginalRequest.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading body: %s", err)
	}

	bodyStr := strings.TrimSpace(string(body))

	if len(bodyStr) > 0 {
		err = Unmarshal([]byte(bodyStr), &res)
		if err != nil {
			return nil, err
		}
	} else {
		// EpochEntryMap == nil mean no list at all, not same as empty list
		return nil, nil
	}

	return res, nil
}

func HTTPDeleteGroups(info *RequestInfo) error {
	list, err := LoadEpochMap(info)
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return err
	}

	// No list provided so get list of Groups so we can delete them all
	// TODO: optimize this to just delete it all in one shot
	if list == nil {
		list = EpochEntryMap{}
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
			list[NotNilString(row[0])] = EpochEntry{}
		}
		defer results.Close()
	}

	// Delete each Group, checking epoch first if provided
	for id, entry := range list {
		group, err := info.Registry.FindGroup(info.GroupType, id, false)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf(`Error getting Group %q`, id)
		}
		if group == nil {
			// Silently ignore the 404
			continue
		}

		if tmp, ok := entry["epoch"]; ok {
			tmpInt, err := AnyToUInt(tmp)
			if err != nil {
				return fmt.Errorf(`Epoch value for %q must be a uinteger`,
					id)
			}
			if tmpInt != group.Get("epoch") {
				return fmt.Errorf(`Epoch value for %q must be %d not %d`,
					id, group.Get("epoch"), tmpInt)
			}
		}

		singular := group.Singular + "id"
		if tmp, ok := entry[singular]; ok && tmp != id {
			return fmt.Errorf(`%q value for %q must be %q not "%v"`,
				singular, id, id, tmp)
		}

		err = group.Delete()
		if err != nil {
			info.StatusCode = http.StatusInternalServerError
			return fmt.Errorf(`Error deleting %q: %s`, id, err)
		}
	}

	info.StatusCode = http.StatusNoContent
	return nil
}

func HTTPDeleteResources(info *RequestInfo) error {
	list, err := LoadEpochMap(info)
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return err
	}

	// No list provided so get list of Resources so we can delete them all
	// TODO: optimize this to just delete it all in one shot
	if list == nil {
		list = EpochEntryMap{}
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
			list[NotNilString(row[0])] = EpochEntry{}
		}
		defer results.Close()
	}

	group, err := info.Registry.FindGroup(info.GroupType, info.GroupUID, false)
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf(`Error getting Group %q`, info.GroupUID)
	}

	// Delete each Resource, checking epoch first if provided
	for id, entry := range list {
		resource, err := group.FindResource(info.ResourceType, id, false)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf(`Error getting Resource %q`, id)
		}
		if resource == nil {
			// Silently ignore the 404
			continue
		}

		meta, err := resource.FindMeta(false)
		if err != nil {
			info.StatusCode = http.StatusInternalServerError
			return fmt.Errorf(`Error finding resource %q: %s`, id, err)
		}

		singular := resource.Singular + "id"

		if metaJSON, ok := entry["meta"]; ok {
			metaMap, ok := metaJSON.(map[string]any)
			if !ok {
				if err != nil {
					info.StatusCode = http.StatusBadRequest
					return fmt.Errorf(`"meta" isn't a map`)
				}
			}

			if tmp, ok := metaMap[singular]; ok && tmp != id {
				return fmt.Errorf(`%q value for %q must be %q not "%v"`,
					singular, id, id, tmp)
			}

			if tmp, ok := metaMap["epoch"]; ok {
				tmpInt, err := AnyToUInt(tmp)
				if err != nil {
					return fmt.Errorf(`Epoch value for %q must be a uinteger`,
						id)
				}
				if tmpInt != meta.Get("epoch") {
					return fmt.Errorf(`Epoch value for %q must be %d not %d`,
						id, meta.Get("epoch"), tmpInt)
				}
			}
		} else {
			if _, ok := entry["epoch"]; ok {
				return fmt.Errorf(`"epoch" should be under a "meta" map`)
			}
		}

		if tmp, ok := entry[singular]; ok && tmp != id {
			return fmt.Errorf(`%q value for %q must be %q not "%v"`,
				singular, id, id, tmp)
		}

		err = resource.Delete()
		if err != nil {
			info.StatusCode = http.StatusInternalServerError
			return fmt.Errorf(`Error deleting %q: %s`, id, err)
		}
	}

	info.StatusCode = http.StatusNoContent
	return nil
}

func HTTPDeleteVersions(info *RequestInfo) error {
	nextDefault := info.GetFlag("setdefaultversionid")

	list, err := LoadEpochMap(info)
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return err
	}

	// No list provided so get list of Versions so we can delete them all
	// TODO: optimize this to just delete it all in one shot
	if list == nil {
		list = EpochEntryMap{}
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
			list[NotNilString(row[0])] = EpochEntry{}
		}
		defer results.Close()
	}

	group, err := info.Registry.FindGroup(info.GroupType, info.GroupUID, false)
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf(`Error getting Group %q: %s`, info.GroupUID, err)
	}

	resource, err := group.FindResource(info.ResourceType, info.ResourceUID, false)
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf(`Error getting Resource %q: %s`,
			info.ResourceUID, err)
	}

	// Delete each Version, checking epoch first if provided
	for id, entry := range list {
		version, err := resource.FindVersion(id, false)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf(`Error getting Version %q: %s`, id, err)
		}
		if version == nil {
			// Silently ignore the 404
			continue
		}

		if tmp, ok := entry["epoch"]; ok {
			tmpInt, err := AnyToUInt(tmp)
			if err != nil {
				return fmt.Errorf(`Epoch value for %q must be a uinteger`,
					id)
			}
			if tmpInt != version.Get("epoch") {
				return fmt.Errorf(`Epoch value for %q must be %d not %d`,
					id, version.Get("epoch"), tmpInt)
			}
		}

		singular := version.Singular + "id"
		if tmp, ok := entry[singular]; ok && tmp != version.Get(singular) {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf(`%q value for %q must be %q not "%v"`,
				singular, id, version.Get(singular), tmp)
		}

		err = version.DeleteSetNextVersion(nextDefault)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return err
		}
	}

	if nextDefault != "" {
		version, err := resource.FindVersion(nextDefault, false)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf(`Error getting Version %q: %s`, nextDefault, err)
		}
		err = resource.SetDefault(version)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return err
		}
	}

	info.StatusCode = http.StatusNoContent
	return nil
}

func ExtractIncomingObject(info *RequestInfo, body []byte) (Object, error) {
	IncomingObj := map[string]any{}

	if len(body) == 0 {
		body = nil
	}

	resSingular := ""
	if info.ResourceModel != nil {
		resSingular = info.ResourceModel.Singular
	}

	// len=5 is a special case where we know .../versions always has the
	// metadata in the body so $structure isn't needed, and in fact an error

	// GROUPS/gID/RESOURCES/rID/meta|versions/vID
	metaInBody := (info.ShowStructure ||
		len(info.Parts) == 3 ||
		len(info.Parts) == 5 ||
		(info.ResourceModel != nil && info.ResourceModel.GetHasDocument() == false))

	if len(info.Parts) < 3 || metaInBody {
		for k, _ := range info.OriginalRequest.Header {
			k := strings.ToLower(k)
			if strings.HasPrefix(k, "xregistry-") {
				info.StatusCode = http.StatusBadRequest
				if info.ResourceModel.GetHasDocument() == false {
					return nil, fmt.Errorf("Including \"xRegistry\" headers " +
						"for a Resource that has the model \"hasdocument\" " +
						"value of \"false\" is invalid")
				}
				return nil, fmt.Errorf("Including \"xRegistry\" headers " +
					"when \"$structure\" is used is invalid")
			}
		}

		if strings.TrimSpace(string(body)) == "" {
			body = []byte("{}") // Be forgiving
		}

		err := Unmarshal(body, &IncomingObj)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return nil, err
		}
	}

	// xReg metadata are in headers, so move them into IncomingObj. We'll
	// copy over the existing properties later once we know what entity
	// we're dealing with
	if len(info.Parts) > 2 && !metaInBody {
		IncomingObj[resSingular] = body // save new body

		seenMaps := map[string]bool{}

		for name, attr := range attrHeaders {
			// TODO we may need some kind of "delete if missing" flag on
			// each HttpHeader attribute since some may want to have an
			// explicit 'null' to be erased instead of just missing (eg patch)
			vals, ok := info.OriginalRequest.Header[http.CanonicalHeaderKey(name)]
			if ok {
				val := vals[0]
				if val == "null" {
					IncomingObj[attr.Name] = nil
				} else {
					IncomingObj[attr.Name] = val
				}
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
						// Should we just skip all of this logic if nil?
						// If we try, watch for the case where someone
						// has just xReg-label-foo:null, it should probably
						// create the empty map anyway. And watch for the
						// case mentioned below
						if val != nil {
							obj[part] = val
						}
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

	// Convert all HTTP header values into their proper data types since
	// as of now they're all just strings
	if !metaInBody && info.ResourceModel != nil {
		attrs := info.ResourceModel.GetBaseAttributes()
		attrs.AddIfValuesAttributes(IncomingObj)
		attrs.ConvertStrings(IncomingObj)
	}

	return IncomingObj, nil
}

// Find all Groups/Resources/Versions in 'obj' and make sure their
// RESOURCE attribute is converted into a byte array
func ConvertRegistryContents(obj map[string]any, m *Model) error {
	for _, gm := range m.Groups {
		val, ok := obj[gm.Plural]
		if ok && !IsNil(val) {
			gColl, ok := val.(map[string]any)
			if !ok {
				return fmt.Errorf("%q should be a map", gm.Plural)
			}
			for _, val := range gColl {
				g, ok := val.(map[string]any)
				if !ok {
					return fmt.Errorf("%q should be a map of %q, not %T",
						gm.Plural, gm.Singular, val)
				}
				// Traverse into each Group
				if err := ConvertGroupContents(g, gm); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Find all Resources/Versions in 'obj' and make sure their
// RESOURCE attribute is converted into a byte array
func ConvertGroupContents(obj map[string]any, gm *GroupModel) error {
	for _, rm := range gm.Resources {
		val, ok := obj[rm.Plural]
		if ok && !IsNil(val) {
			rColl, ok := val.(map[string]any)
			if !ok {
				return fmt.Errorf("%q should be a map", rm.Plural)
			}
			for _, val := range rColl {
				r, ok := val.(map[string]any)
				if !ok {
					return fmt.Errorf("%s should be a map of %q, not %T",
						rm.Plural, rm.Singular, val)
				}
				// Traverse into each Resource
				if err := ConvertResourceContents(r, rm); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Convert the RESOURCE attribute from JSON to a byte array, if present.
// Then do any Versions in the "versions" collection, if there.
func ConvertResourceContents(obj map[string]any, rm *ResourceModel) error {
	if rm == nil || rm.GetHasDocument() == false {
		return nil
	}

	// Process the local RESOURCE attribute before we look for "versions"
	if err := ConvertVersionContents(obj, rm); err != nil {
		return err
	}

	val, ok := obj["versions"]
	if ok && !IsNil(val) {
		vColl, ok := val.(map[string]any)
		if !ok {
			return fmt.Errorf("%q should be a map", "versions")
		}
		for _, val := range vColl {
			v, ok := val.(map[string]any)
			if !ok {
				return fmt.Errorf("%q should be a map of %q, not %T",
					"versions", "version", val)
			}
			// Traverse into each Version
			if err := ConvertVersionContents(v, rm); err != nil {
				return err
			}
		}
	}

	return nil
}

// Convert the RESOURCE attribute from JSON to a byte array, if present.
func ConvertVersionContents(obj map[string]any, rm *ResourceModel) error {
	if rm == nil || rm.GetHasDocument() == false {
		return nil
	}

	var err error
	data, ok := obj[rm.Singular]

	if ok {

		// Special case of: RESOURCE:null
		if IsNil(data) {
			obj[rm.Singular] = nil
			obj["#-contenttype"] = nil
			return nil
		}

		// Keep bytes as bytes - possibly already converted (eg was in body)
		if reflect.ValueOf(data).Type().String() == "[]uint8" {
			return nil
		}

		// Get the raw bytes of the "rm.Singular" json attribute
		buf := []byte(nil)
		switch reflect.ValueOf(data).Kind() {
		case reflect.Float64, reflect.Map, reflect.Slice, reflect.Bool:
			buf, err = json.Marshal(data)
			if err != nil {
				return err
			}
		case reflect.Invalid:
			// I think this only happens when it's "null".
			// just let 'buf' stay as nil
		default:
			str := fmt.Sprintf("%s", data)
			buf = []byte(str)
		}
		obj[rm.Singular] = buf
		obj["#-contenttype"] = "application/json"
	}
	return nil
}
