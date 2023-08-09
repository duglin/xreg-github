package registry

import (
	"bytes"
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
	dw.Info.OriginalResponse.Header().Add(name, value)
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
	if req.URL.Query().Has("noprops") {
		buf = RemoveProps(buf)
	}
	if req.URL.Query().Has("oneline") {
		buf = OneLine(buf)
	}
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
		info.StatusCode = http.StatusSeeOther
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
			info.OriginalResponse.Header()[header] = value
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
		// info.OriginalResponse.WriteHeader(200) // http.StatusNoContent)
		return nil
	}
	info.Write(buf.([]byte))

	return nil
}

func HTTPGet(info *RequestInfo) error {
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
	bodyIsContent := false
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

	// Load the xReg properties - either from headers or body

	// We have /GROUPs/gID/RESOURCEs but not ?meta so grab headers
	if len(info.Parts) >= 3 && !info.ShowMeta {
		bodyIsContent = true
		entityData.Content = body
		entityData.Patch = true
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

			if !strings.HasPrefix(lowerKey, "labels-") {
				entityData.Props[key] = value[0] // only grab first one
			} else {
				entityData.Props["labels/"+key[7:]] = value[0]
			}
		}
	} else {
		// Assume body is xReg metadata so parse it into entityData.Props
		if strings.TrimSpace(string(body)) == "" {
			body = []byte("{}") // Be forgiving
		}
		err = json.Unmarshal(body, &entityData.Props)
		if err != nil {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf("Error parsing body: %s", err)
		}
		entityData.UpdateCase = true

		// Fix labels
		for k, v := range entityData.Props {
			if strings.ToLower(k) == "labels" && k != "labels" {
				return fmt.Errorf("Property name %q is invalid, one in "+
					"a different case already exists", k)
			}
			if k != "labels" {
				continue
			}

			labelMap, ok := v.(map[string]string)
			if !ok {
				return fmt.Errorf("Property 'labels' must be a "+
					" map(string,string), not %T", v)
			}
			entityData.Props["label"] = labelMap
		}
	}

	log.VPrintf(3, "entityData.Props:\n%s", ToJSON(entityData.Props))
	log.VPrintf(3, "Body: %d bytes / IsContent: %v", len(body), bodyIsContent)

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
		err = UserUpdateEntity(&group.Entity, &entityData)
	}

	if err != nil {
		info.StatusCode = http.StatusInternalServerError
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
		resource, err = group.FindResource(info.ResourceType, resourceUID)
		if err == nil && resource == nil {
			if versionUID == "" {
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

	if err == nil {
		// Update Resource or Version based on the URL of the request
		entityData.IsNew = isNew
		if len(info.Parts) < 5 {
			// Either /GROUPs/gID/RESOURCEs or /GROUPs/gID/RESOURCEs/rID
			entityData.Level = 2
			err = UserUpdateEntity(&resource.Entity, &entityData)
		} else {
			// Either ..RESOURCEs/rID/versions or ..RESOURCEs/rID/versions/vID
			entityData.Level = 3
			err = UserUpdateEntity(&version.Entity, &entityData)
		}

	}

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

	// location := info.BaseURL + "/" + resourcePath
	location := resource.Path
	if originalLen > 4 {
		info.Parts = append(info.Parts, "versions", versionUID)
		info.VersionUID = versionUID
		// location += "/version" + info.VersionUID
		location = version.Path
	}

	if isNew { // 201, else let it default to 200
		info.AddHeader("Location", location)
		info.StatusCode = http.StatusCreated
	}

	return HTTPGet(info)
}

type EntityData struct {
	Level      int
	Props      map[string]any
	Content    []byte
	UpdateCase bool
	Patch      bool
	IsNew      bool
}

// check for props to be removed - old props
// check for casing against list of existing props
func UserUpdateEntity(entity *Entity, ed *EntityData) error {
	var err error

	tmp := entity.Props["epoch"]
	epoch := NotNilInt(&tmp)
	if epoch <= 0 {
		epoch = 0
	}

	if incomingEpoch, ok := ed.Props["epoch"]; ok {
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
			// It's a user-defined property name - aka an extension

			// See if it exsits in the existing entity's Props
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
		}

		log.VPrintf(1, "Setting %q->%q", k, v)
		err := SetProp(entity, k, v)
		if err != nil {
			return err
		}
	}

	// Delete any remaining properties from the Entity, if not patching
	if !ed.Patch {
		for _, v := range tmpLowerEntityProps {
			log.VPrintf(1, "Deleting %q", v)
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
