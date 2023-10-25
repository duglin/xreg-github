package registry

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

	if r.URL.Query().Has("reg") {
		list := ""
		list += fmt.Sprintf("<li onclick=choose('/')>Default</li>\n")
		for _, name := range GetRegistryNames() {
			list += fmt.Sprintf("<li onclick=choose('/reg-%s')>%s</li>\n",
				name, name)
		}

		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte(`<html>
<style>
  form {
    display: inline ;
  }
  body {
    display: flex ;
    flex-direction: row ;
    flex-wrap: nowrap ;
    justify-content: flex-start ;
    height: 100% ;
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
    width: 100% ;

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
  #output {
    background-color: ghostwhite;
    border: 0px ;
    flex: 1 ;
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
<script>
  function choose(e) {
    var val = "http://ubuntu:8080" + e ;
    document.getElementById('myURL').value = val ;
    go(val);
  }

  function go() {
    var val1 = document.getElementById('myURL').value ;
    // val1 += (val1.includes("?") ? "&":"?") + "html"

    // document.getElementById('output').src = val1 ;
	getPage(val1)
  }

  function go1(url) {
    document.getElementById('myURL').value = url ;
	go(url)
  }

  function getPage(url) {
    var req = new XMLHttpRequest();
    req.open("GET", url, false);
    req.send(null);
    var text = req.responseText ;
    text = "<pre>" + text + "</pre>" ;

	// text.replace('"(https?://ubuntu[^"\n]*?)"','"<a href=\'$1\'>$1</a>"');
	const regex = new RegExp('"(https?://ubuntu[^"\n]*)"',"g");
	text = text.replaceAll(regex,'"<a onclick=\'go1("$1");return false;\' href=\'$1\'>$1</a>"');
    document.getElementById('myOutput').innerHTML = text ;
  }
</script>
<div id=left>
<b>Choose a registry:</b>
<br><br>
` + list + `
</div>
<div id=right>
    <form id=url target=iframe onsubmit="go();return false;">
      <div style="margin:0 5 0 10">URL:</div>
      <input id=myURL type=text>
      <button type=submit> Go! </button>
    </form>
  <!-- <iframe id=output name='iframe'></iframe> -->
  <div id=myOutput>
  </div>
</div>
`))
		return
	}

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

	// If we want to tweak the output we'll need to buffer it
	if r.URL.Query().Has("html") || r.URL.Query().Has("noprops") {
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
			// Only default to BadRequest is not set by someone else
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
	Info    *RequestInfo
	Headers *map[string]string
	Buffer  *bytes.Buffer
}

func NewBufferedWriter(info *RequestInfo) *BufferedWriter {
	return &BufferedWriter{
		Info:    info,
		Headers: &map[string]string{},
		Buffer:  &bytes.Buffer{},
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
		bw.Info.OriginalResponse.Header()[k] = []string{v}
	}

	code := bw.Info.StatusCode
	if code == 0 {
		code = http.StatusOK
	}
	bw.Info.OriginalResponse.WriteHeader(code)

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
		bw.Info.OriginalResponse.Write([]byte("<pre>\n"))
		buf = HTMLify(req, buf)
	}
	bw.Info.OriginalResponse.Write(buf)
}

type DiscardWriter struct{}

func (dw *DiscardWriter) Write(b []byte) (int, error)  { return len(b), nil }
func (dw *DiscardWriter) AddHeader(name, value string) {}
func (dw *DiscardWriter) Done()                        {}

var DefaultDiscardWriter = &DiscardWriter{}

func HTTPGETModel(info *RequestInfo) error {
	if len(info.Parts) > 1 {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Not found")
	}

	model := info.Registry.Model
	if model == nil {
		model = &Model{}
	}

	httpModel := ModelToHTTPModel(model)

	buf, err := json.MarshalIndent(httpModel, "", "  ")
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

	entity := readNextEntity(results)
	if entity == nil {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Not found")
	}

	var version *Entity
	versionsCount := 0
	if info.VersionUID == "" {
		// We're on a Resource, so go find the right Version
		vID := entity.Get("latestVersionId").(string)
		for {
			v := readNextEntity(results)
			if v == nil && version == nil {
				info.StatusCode = http.StatusInternalServerError
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
			for name, value := range val.(map[string]string) {
				info.AddHeader("xRegistry-labels-"+name, value)
			}
			return nil
		}

		str = fmt.Sprintf("%v", val)

		info.AddHeader("xRegistry-"+key, str)
		return nil
	}

	err = entity.SerializeProps(info, headerIt)
	if err != nil {
		panic(err)
	}

	if info.VersionUID == "" {
		info.AddHeader("xRegistry-versionsCount",
			fmt.Sprintf("%d", versionsCount))
		info.AddHeader("xRegistry-versionsUrl",
			info.BaseURL+"/"+entity.Path+"/versions")

		info.AddHeader("Content-Location", info.BaseURL+"/"+version.Path)
	}

	url := ""
	if val := entity.Get("#resourceURL"); val != nil {
		gModel := info.Registry.Model.Groups[info.GroupType]
		rModel := gModel.Resources[info.ResourceType]
		singular := rModel.Singular

		url = val.(string)
		info.AddHeader("xRegistry-"+singular+"Url", url)

		if info.StatusCode == 0 {
			// If we set it during a PUT/POST, don't override the 201
			info.StatusCode = http.StatusSeeOther
		}
		info.AddHeader("Location", url)
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

func HTTPPutPost(info *RequestInfo) error {
	isNew := false
	method := strings.ToUpper(info.OriginalRequest.Method)
	entityData := EntityData{
		Props: map[string]any{},
	}

	info.Root = strings.Trim(info.Root, "/")

	// The model has its own special func
	if len(info.Parts) > 0 && info.Parts[0] == "model" {
		return HTTPPUTModel(info)
	}

	// Load-up the body
	body, err := io.ReadAll(info.OriginalRequest.Body)
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf("Error reading body: %s", err)
	}
	if len(body) == 0 {
		body = nil
	}

	var groupModel *GroupModel
	var resourceModel *ResourceModel

	if info.GroupType != "" {
		groupModel = info.Registry.Model.Groups[info.GroupType]

		if info.ResourceType != "" {
			resourceModel = groupModel.Resources[info.ResourceType]
		}
	}

	// Load the xReg properties - either from headers or body

	// We have /GROUPs/gID/RESOURCEs but not ?meta so grab headers
	if len(info.Parts) >= 3 && !info.ShowMeta {
		entityData.Patch = true
		entityData.Props["#resource"] = body

		for key, value := range info.OriginalRequest.Header {
			lowerKey := strings.ToLower(key)
			if !strings.HasPrefix(lowerKey, "xregistry-") {
				continue
			}
			lowerKey = strings.TrimSpace(lowerKey[10:]) // remove xRegistry-
			key = strings.TrimSpace(key[10:])           // remove xRegistry-
			if key == "" {
				continue
			}

			lowerSingular := strings.ToLower(resourceModel.Singular)
			if lowerKey == lowerSingular {
				return fmt.Errorf("'xRegistry-%s' isn't allowed as an HTTP "+
					"header", resourceModel.Singular)
			}
			if lowerKey == lowerSingular+"base64" {
				return fmt.Errorf("'xRegistry-%qBase64' isn't allowed as an "+
					" HTTP header", resourceModel.Singular)
			}

			// If it's a specProp then set Key to it's real case
			specProp, isSpec := SpecProps[lowerKey]
			if isSpec {
				key = specProp.name
			}

			val := any(value[0])
			if val == "null" {
				val = nil
			}

			if !strings.HasPrefix(lowerKey, "labels-") {
				// Not a label
				if lowerKey == lowerSingular+"url" {
					// is #resourceURL
					if len(body) != 0 {
						return fmt.Errorf("'xRegistry-%sUrl' isn't allowed "+
							"if there's a body", resourceModel.Singular)
					}

					entityData.Props["#resource"] = nil
					key = "#resourceURL"
				}
			} else {
				key = "labels/" + key[7:]
			}

			entityData.Props[key] = val
		}
	} else {
		// Assume body is xReg metadata so parse it into entityData.Props
		if strings.TrimSpace(string(body)) == "" {
			body = []byte("{}") // Be forgiving
		}

		err = json.Unmarshal(body, &entityData.RawProps)
		err = json.Unmarshal(body, &entityData.Props)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf("Error parsing body: %s", err)
		}
		entityData.UpdateCase = true

		// Fix labels and check for RESOURCE properties
		for k, v := range entityData.Props {
			lowerK := strings.ToLower(k)
			if lowerK == "labels" && k != "labels" {
				return fmt.Errorf("Property name %q is invalid, one in "+
					"a different case already exists", k)
			}

			// If we're on a resource or version, check for its content
			if resourceModel != nil {
				singular := resourceModel.Singular

				if k == singular {
					delete(entityData.Props, k)
					delete(entityData.RawProps, k)

					entityData.Props["#resourceURL"] = nil
					entityData.Props["#resource"] = entityData.RawProps[k]
				}

				if k == singular+"Base64" {
					if body != nil {
						return fmt.Errorf("Only one of '%s', '%sUrl' and "+
							" '%sBase64' can be present at a time",
							singular, singular, singular)
					}
					data := entityData.RawProps[k]
					content, err := base64.StdEncoding.DecodeString(string(data))
					if err != nil {
						return fmt.Errorf("Error decoding base64 data(%s): "+
							"%s", k, err)
					}

					delete(entityData.Props, k)
					delete(entityData.RawProps, k)

					entityData.Props["#resource"] = content
					entityData.Props["#resourceURL"] = nil
				}

				if k == resourceModel.Singular+"Url" {
					if body != nil {
						return fmt.Errorf("'%sUrl' and an HTTP body can not "+
							"both be present", resourceModel.Singular)
					}

					delete(entityData.Props, k)
					delete(entityData.RawProps, k)

					entityData.Props["#resourceURL"] = v
					entityData.Props["#resource"] = nil
				}
			}

			if k != "labels" {
				continue
			}

			labelMap, ok := v.(map[string]any)
			if !ok {
				return fmt.Errorf("Property 'labels' must be a "+
					"map(string,any), not %T", v)
			}

			for k, v := range labelMap {
				if v == nil {
					continue
				}
				// Be nice and convert any non-string into a string
				entityData.Props["labels/"+k] = fmt.Sprintf("%v", v)
			}

			delete(entityData.Props, "labels")
		}
	}

	log.VPrintf(3, "entityData.Props:\n%s", ToJSON(entityData.Props))
	log.VPrintf(3, "Body: %d bytes", len(body))

	// Check for some obvious high-level bad states up-front
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

	tmp := entityData.Props["id"]
	propsID := NotNilString(&tmp)

	// All ready to go, let's walk the path

	// URL: /
	// ////////////////////////////////////////////////////////////////
	if len(info.Parts) == 0 {
		// MUST be PUT /
		entityData.IsNew = isNew
		entityData.Level = 0
		err = UserUpdateEntity(&info.Registry.Entity, &entityData)

		if err != nil {
			info.StatusCode = http.StatusInternalServerError
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
	}
	if err == nil && group == nil {
		// Group doesn't exist so create it
		isNew = true
		group, err = info.Registry.AddGroup(info.GroupType, groupUID)
	}

	if err == nil && len(info.Parts) < 3 {
		// Either /GROUPs or /GROUPs/gID
		entityData.IsNew = isNew
		entityData.Level = 1
		entityData.Plural = groupModel.Plural
		entityData.Singular = groupModel.Singular
		err = UserUpdateEntity(&group.Entity, &entityData)
	}

	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf("Error processing group(%s): %s", groupUID, err)
	}

	if len(info.Parts) < 3 {
		// Either /GROUPs or /GROUPs/gID - so all done
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

		if err == nil && resource == nil {
			if versionUID == "" &&
				((len(info.Parts) == 4 && method == "POST") ||
					(len(info.Parts) == 5)) {

				// No vID in URL, grab from props. If missing, auto-generate
				versionUID = propsID
			}

			isNew = true
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
		if versionUID != "" {
			version, err = resource.FindVersion(versionUID)
		}
		if err == nil && version == nil {
			if versionUID == "" {
				versionUID = propsID
			}
			if versionUID == "" {
				version, err = resource.GetLatest()
			} else {
				isNew = true
				version, err = resource.AddVersion(versionUID)
			}
		}
	}

	if err != nil || version == nil {
		info.StatusCode = http.StatusInternalServerError
		return fmt.Errorf("Error processing version(%s): %s", versionUID, err)
	}

	// Update Resource or Version based on the URL of the request
	entityData.IsNew = isNew
	entityData.Plural = resourceModel.Plural
	entityData.Singular = resourceModel.Singular

	if len(info.Parts) < 5 {
		entityData.Props["id"] = version.UID
	}
	/*
		if len(info.Parts) < 5 {
			// Either /GROUPs/gID/RESOURCEs or /GROUPs/gID/RESOURCEs/rID
			entityData.Level = 2
			err = UserUpdateEntity(&resource.Entity, &entityData)
		} else {
	*/
	// Either ..RESOURCEs/rID/versions or ..RESOURCEs/rID/versions/vID
	entityData.Level = 3
	err = UserUpdateEntity(&version.Entity, &entityData)
	// }

	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return fmt.Errorf("Error processing request: %s", err)
	}

	originalLen := len(info.Parts)

	info.Parts = []string{info.Parts[0], groupUID,
		info.Parts[2], resourceUID}
	info.What = "Entity"
	info.GroupUID = groupUID
	info.ResourceUID = resourceUID

	location := info.BaseURL + "/" + resource.Path
	// location := resource.Path
	if originalLen > 4 {
		info.Parts = append(info.Parts, "versions", versionUID)
		info.VersionUID = versionUID
		location += "/version" + info.VersionUID
		// location = version.Path
	}

	if isNew { // 201, else let it default to 200
		info.AddHeader("Location", location)
		info.StatusCode = http.StatusCreated
	}

	return HTTPGet(info)
}

type EntityData struct {
	Level      int
	Plural     string
	Singular   string
	Props      map[string]any
	RawProps   map[string]json.RawMessage
	UpdateCase bool
	Patch      bool
	IsNew      bool
}

// check for props to be removed - old props
// check for casing against list of existing props
func UserUpdateEntity(entity *Entity, ed *EntityData) error {
	var err error
	hadLabel := false

	log.VPrintf(3, "Updating:\n%s\n", ToJSON(ed))

	tmp := entity.Props["epoch"]
	epoch := NotNilInt(&tmp)
	if epoch <= 0 {
		epoch = 0
	}

	if tmp := ed.Props["id"]; tmp != nil {
		if tmp != entity.Get("id") {
			return fmt.Errorf("Metadata id(%s) doesn't match ID in "+
				"URL(%s)", tmp, entity.Get("id"))
		}
	}

	if incomingEpoch, ok := ed.Props["epoch"]; ok && !ed.IsNew {
		kind := reflect.ValueOf(incomingEpoch).Kind()
		incoming := 0
		if kind == reflect.String {
			tmpStr := incomingEpoch.(string)
			incoming, err = strconv.Atoi(tmpStr)
			if err != nil {
				return fmt.Errorf("Error parsing 'epoch'(%s): %s",
					incomingEpoch, err)
			}
		} else if kind == reflect.Float64 { // JSON ints show up as floats
			incoming = int(incomingEpoch.(float64))
		} else if kind != reflect.Int {
			return fmt.Errorf("Epoch must be an int, not %s", kind.String())
		} else {
			incoming = incomingEpoch.(int)
		}

		if incoming != epoch {
			return fmt.Errorf("Incoming epoch(%d) doesn't match existing "+
				"epoch(%d)", incoming, epoch)
		}
	}

	oldLabels := map[string]bool{}

	// Find all mutable spec-defined props or extensions
	// and save them in a map (key=lower-name) for easy reference
	// and these are the ones we'll want to delete when done
	tmpLowerEntityProps := map[string]string{} // key == lower name
	for k, _ := range entity.Props {
		lowerK := strings.ToLower(k)
		specProp, isSpec := SpecProps[lowerK]

		// Only save it if it's an extension or if the spec prop is mutable
		if !isSpec || specProp.mutable == true {
			tmpLowerEntityProps[lowerK] = k
		}

		// Save existing label keys too
		if strings.HasPrefix(k, "labels/") {
			oldLabels[k] = true
		}
	}

	for k, v := range ed.Props {
		lowerK := strings.ToLower(k)

		specProp := SpecProps[lowerK]
		if specProp != nil {
			// It's a spec defined property name
			if ed.UpdateCase && k != specProp.name {
				return fmt.Errorf("Property name %q is invalid, one in "+
					"a different case already exists", k)
			}
			if specProp.mutable == false {
				log.VPrintf(4, "Skipping immutable prop %q", k)
				continue
			}

			// Remove from delete list
			delete(tmpLowerEntityProps, lowerK)

			// OK, let it thru so we can set it
		} else {
			// It's a user-defined property name - aka an extension(or label)

			// See if it exists in the existing entity's Props
			if caseProp, ok := tmpLowerEntityProps[lowerK]; ok {
				// Found one!

				// Case doesn't match and we're not supposed to update it
				// so just use the existing case instead
				if caseProp != k && !ed.UpdateCase {
					k = caseProp
				}

				// Remove from delete list
				delete(tmpLowerEntityProps, lowerK)

				// OK, let it thru so we can set it
			} else {
				// Not an existing prop so just let it thru so we can set it
			}

			if strings.HasPrefix(lowerK, "labels/") {
				hadLabel = true
				delete(oldLabels, k)
			}
		}

		err := SetProp(entity, k, v)
		if err != nil {
			return err
		}
	}

	if hadLabel {
		for k, _ := range oldLabels {
			err := SetProp(entity, k, nil)
			if err != nil {
				return err
			}
		}
	}

	// Delete any remaining properties from the Entity, if not patching
	if !ed.Patch {
		for _, v := range tmpLowerEntityProps {
			err := SetProp(entity, v, nil)
			if err != nil {
				return err
			}
		}
	}

	// Only update the epoch if the entity isn't new
	if !ed.IsNew {
		epoch++
	}

	return SetProp(entity, "epoch", epoch)
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

	tmpModel := HTTPModel{}
	err = json.Unmarshal(reqBody, &tmpModel)
	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return err
	}

	model := tmpModel.ToModel()
	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return err
	}

	err = info.Registry.Model.ApplyNewModel(model)
	if err != nil {
		info.StatusCode = http.StatusBadRequest
		return err
	}

	return HTTPGETModel(info)
}
