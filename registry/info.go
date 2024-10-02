package registry

import (
	"fmt"
	"net/http"
	"strings"

	log "github.com/duglin/dlog"
)

type RequestInfo struct {
	tx               *Tx
	Registry         *Registry
	BaseURL          string
	OriginalPath     string
	OriginalRequest  *http.Request       `json:"-"`
	OriginalResponse http.ResponseWriter `json:"-"`
	Parts            []string
	Root             string
	Abstract         string
	GroupType        string
	GroupUID         string
	GroupModel       *GroupModel
	ResourceType     string
	ResourceUID      string
	ResourceModel    *ResourceModel
	VersionUID       string
	What             string // Registry, Coll, Entity
	HasNested        bool
	Inlines          []string        // TODO store a PropPaths instead
	Filters          [][]*FilterExpr // [OR][AND] filter=e,e(and) &(or) filter=e
	ShowModel        bool
	ShowMeta         bool //	was $meta present

	StatusCode int
	SentStatus bool
	HTTPWriter HTTPWriter `json:"-"`

	extras map[string]any
}

func (info *RequestInfo) AddInline(path string) error {
	// use "*" to inline all
	// path = strings.TrimLeft(path, "/.") // To be nice

	pp, err := PropPathFromUI(path)
	if err != nil {
		return err
	}

	for _, group := range info.Registry.Model.Groups {
		if pp.Equals(NewPPP(group.Plural)) {
			info.Inlines = append(info.Inlines, NewPPP(path).DB())
			return nil
		}
		for _, res := range group.Resources {
			if pp.Equals(NewPPP(group.Plural).P(res.Plural)) ||
				pp.Equals(NewPPP(group.Plural).P(res.Plural).P(res.Singular)) ||
				pp.Equals(NewPPP(group.Plural).P(res.Plural).P("versions")) ||
				pp.Equals(NewPPP(group.Plural).P(res.Plural).P("versions").P(res.Singular)) {

				info.Inlines = append(info.Inlines, pp.DB())
				return nil
			}
		}
	}

	// Convert back to UI version for the error message
	path = pp.UI()

	// Remove Abstract value just to print a nicer error message
	if info.Abstract != "" && strings.HasPrefix(path, info.Abstract) {
		path = path[len(info.Abstract)+1:]
	}

	return fmt.Errorf("Invalid 'inline' value: %s", path)
}

func (info *RequestInfo) IsInlineSet(entityPath string) bool {
	if entityPath == "" {
		entityPath = "*"
	}
	for _, path := range info.Inlines {
		if path == entityPath {
			return true
		}
	}
	return false
}

func (info *RequestInfo) ShouldInline(entityPath string) bool {
	ePP, _ := PropPathFromDB(entityPath) // entity-PP
	for _, path := range info.Inlines {
		iPP, _ := PropPathFromDB(path) // Inline-PP
		if iPP.Top() == "*" || ePP.Equals(iPP) || iPP.HasPrefix(ePP) {
			log.VPrintf(4, "Inline match: %q in %q", entityPath, path)
			return true
		}
	}
	return false
}

func (ri *RequestInfo) Write(b []byte) (int, error) {
	return ri.HTTPWriter.Write(b)
}

func (ri *RequestInfo) AddHeader(name, value string) {
	ri.HTTPWriter.AddHeader(name, value)
}

type FilterExpr struct {
	Path     string // endpoints.id  TODO store a PropPath?
	Value    string // myEndpoint
	HasEqual bool
}

func ParseRequest(tx *Tx, w http.ResponseWriter, r *http.Request) (*RequestInfo, error) {
	path := strings.Trim(r.URL.Path, " /")
	info := &RequestInfo{
		tx: tx,

		OriginalPath:     path,
		OriginalRequest:  r,
		OriginalResponse: w,
		Registry:         GetDefaultReg(tx),
		BaseURL:          "http://" + r.Host,
		ShowModel:        r.URL.Query().Has("model"),

		extras: map[string]any{},
	}

	tx.IgnoreEpoch = r.URL.Query().Has("noepoch")
	tx.IgnoreDefaultVersionSticky = r.URL.Query().Has("nodefaultversionsticky")
	tx.IgnoreDefaultVersionID = r.URL.Query().Has("nodefaultversionid")

	if info.Registry != nil && tx.Registry == nil {
		tx.Registry = info.Registry
	}

	info.HTTPWriter = DefaultHTTPWriter(info)

	defer func() { log.VPrintf(3, "Info:\n%s\n", ToJSON(info)) }()

	if tmp := r.Header.Get("xRegistry~User"); tmp != "" {
		tx.User = tmp
	}

	err := info.ParseRequestURL()
	if err != nil {
		if info.StatusCode == 0 {
			info.StatusCode = http.StatusBadRequest
		}
		return info, err
	}

	info.HasNested = r.URL.Query().Has("nested")

	if r.URL.Query().Has("inline") {
		// Only pick up inlining values if we're doing a GET, not write ops
		if strings.EqualFold(r.Method, "GET") {
			stopInline := false
			for _, value := range r.URL.Query()["inline"] {
				for _, p := range strings.Split(value, ",") {
					if p == "" || p == "*" {
						info.Inlines = []string{"*"}
						stopInline = true
						break
					} else {
						// if we're not at the root then we need to twiddle
						// the inline path to add the HTTP Path as a prefix
						if info.Abstract != "" {
							// want: p = info.Abstract + "." + p  in UI format
							absPP, err := PropPathFromPath(info.Abstract)
							if err != nil {
								info.StatusCode = http.StatusBadRequest
								return info, err
							}
							pPP, err := PropPathFromUI(p)
							if err != nil {
								info.StatusCode = http.StatusBadRequest
								return info, err
							}
							p = absPP.Append(pPP).UI()
						}
						if err := info.AddInline(p); err != nil {
							info.StatusCode = http.StatusBadRequest
							return info, err
						}
					}
				}
				if stopInline {
					break
				}
			}
		}
	}

	err = info.ParseFilters()
	if err != nil {
		info.StatusCode = http.StatusBadRequest
	}

	return info, err
}

func (info *RequestInfo) ParseFilters() error {
	for _, filterQ := range info.OriginalRequest.URL.Query()["filter"] {
		// ?filter=path.to.attribute[=value],* & filter=...

		filterQ = strings.TrimSpace(filterQ)
		exprs := strings.Split(filterQ, ",")
		AndFilters := ([]*FilterExpr)(nil)
		for _, expr := range exprs {
			expr = strings.TrimSpace(expr)
			if expr == "" {
				continue
			}
			path, value, found := strings.Cut(expr, "=")
			pp, err := PropPathFromUI(path)
			if err != nil {
				return err
			}
			path = pp.DB()

			/*
				if info.What != "Coll" && strings.Index(path, "/") < 0 {
					info.StatusCode = http.StatusBadRequest
					return fmt.Errorf("A filter with just an attribute name (%s) "+
						"isn't allowed in this context", path)
				}
			*/

			if info.Abstract != "" {
				// Want: path = abs + "," + path in DB format
				absPP, _ := PropPathFromPath(info.Abstract)
				absPP = absPP.Append(pp)
				path = absPP.DB()
			}

			exact := false
			if found {
				if exact = strings.HasPrefix(value, "="); exact {
					value = value[1:]
				}
			}

			filter := &FilterExpr{
				Path:     path,
				Value:    value,
				HasEqual: found,
			}

			if AndFilters == nil {
				AndFilters = []*FilterExpr{}
			}
			AndFilters = append(AndFilters, filter)
		}

		if AndFilters != nil {
			if info.Filters == nil {
				info.Filters = [][]*FilterExpr{}
			}
			info.Filters = append(info.Filters, AndFilters)
		}
	}
	return nil
}

func (info *RequestInfo) ParseRequestURL() error {
	path := strings.Trim(info.OriginalPath, " /")
	info.Parts = strings.Split(path, "/")
	if len(info.Parts) == 1 && info.Parts[0] == "" {
		info.Parts = []string{}
	}

	if len(info.Parts) > 0 && strings.HasPrefix(info.Parts[0], "reg-") {
		info.BaseURL += "/" + info.Parts[0]
		name := info.Parts[0][4:]
		info.Parts = info.Parts[1:] // shift

		reg, err := FindRegistry(info.tx, name)
		if reg == nil {
			extra := ""
			if err != nil {
				extra = ": " + err.Error()
			}
			return fmt.Errorf("Can't find registry %q%s", name, extra)
		}
		info.tx.Rollback() // Not sure why
		info.tx.Registry = reg
		info.Registry = reg
	}

	if len(info.Parts) == 0 {
		info.Parts = nil
		info.What = "Registry"
		return nil
	}

	// /???
	if len(info.Parts) > 0 && info.Parts[0] == "model" {
		return nil
	}

	// /GROUPs
	if strings.HasSuffix(info.Parts[0], "$meta") {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf("$meta isn't allowed on %q", "/"+info.Parts[0])
	}

	gModel := (*GroupModel)(nil)
	if info.Registry.Model != nil && info.Registry.Model.Groups != nil {
		gModel = info.Registry.Model.Groups[info.Parts[0]]
	}
	if gModel == nil && (info.Parts[0] != "model" || len(info.Parts) > 1) {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Unknown Group type: %s", info.Parts[0])
	}
	info.GroupModel = gModel
	info.GroupType = info.Parts[0]
	info.Root += info.Parts[0]
	info.Abstract += info.Parts[0]

	if len(info.Parts) == 1 {
		info.What = "Coll"
		return nil
	}

	// /GROUPs/gID
	if strings.HasSuffix(info.Parts[1], "$meta") {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf("$meta isn't allowed on %q",
			"/"+strings.Join(info.Parts[:2], "/"))
	}

	info.GroupUID = info.Parts[1]
	info.Root += "/" + info.Parts[1]

	if len(info.Parts) == 2 {
		info.What = "Entity"
		return nil
	}

	// /GROUPs/gID/RESOURCEs
	if strings.HasSuffix(info.Parts[2], "$meta") {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf("$meta isn't allowed on %q",
			"/"+strings.Join(info.Parts[:3], "/"))
	}

	rModel := (*ResourceModel)(nil)
	if gModel.Resources != nil {
		rModel = gModel.Resources[info.Parts[2]]
	}
	if rModel == nil {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Unknown Resource type: %s", info.Parts[2])
	}
	info.ResourceModel = rModel
	info.ResourceType = info.Parts[2]
	info.Root += "/" + info.Parts[2]
	info.Abstract += "/" + info.Parts[2]

	if len(info.Parts) == 3 {
		info.What = "Coll"
		return nil
	}

	// /GROUPs/gID/RESOURCEs/rID
	info.ResourceUID = info.Parts[3]
	info.Root += "/" + info.Parts[3]

	// GROUPs/gID/RESOURCEs/rID
	if len(info.Parts) == 4 {
		info.ResourceUID, info.ShowMeta =
			strings.CutSuffix(info.ResourceUID, "$meta")

		if info.ResourceUID == "" {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf("Resource id in URL can't be blank")
		}

		info.Parts[3] = info.ResourceUID
		info.What = "Entity"
		return nil
	}

	// GROUPs/gID/RESOURCEs/rID/???
	if strings.HasSuffix(info.ResourceUID, "$meta") {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf("$meta isn't allowed on %q",
			"/"+strings.Join(info.Parts[:4], "/"))
	}

	if strings.HasSuffix(info.Parts[4], "$meta") {
		info.StatusCode = http.StatusBadRequest
		return fmt.Errorf("$meta isn't allowed on %q",
			"/"+strings.Join(info.Parts[:5], "/"))
	}

	if info.Parts[4] != "versions" {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Expected \"versions\", got: %s", info.Parts[4])
	}

	// GROUPs/gID/RESOURCEs/rID/versions
	info.Root += "/versions"
	info.Abstract += "/versions"
	if len(info.Parts) == 5 {
		info.What = "Coll"
		return nil
	}

	// GROUPs/gID/RESOURCEs/rID/versions/vID
	info.VersionUID = info.Parts[5]
	info.Root += "/" + info.Parts[5]

	if len(info.Parts) == 6 {
		info.VersionUID, info.ShowMeta =
			strings.CutSuffix(info.VersionUID, "$meta")

		if info.VersionUID == "" {
			info.StatusCode = http.StatusBadRequest
			return fmt.Errorf("Version id in URL can't be blank")
		}

		info.Parts[5] = info.VersionUID
		info.What = "Entity"
		return nil
	}

	info.StatusCode = http.StatusBadRequest
	return fmt.Errorf("URL is too long")
}
