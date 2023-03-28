package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"
)

type GroupModel struct {
	Singular string `json:"singular,omitempty"`
	Plural   string `json:"plural,omitempty"`
	Schema   string `json:"schema,omitempty"`

	Resources map[string]*ResourceModel // Plural
}

type ResourceModel struct {
	Singular string `json:"singular,omitempty"`
	Plural   string `json:"plural,omitempty"`
	Versions int    `json:"versions,omitempty"`
}

type Model struct {
	Groups map[string]*GroupModel // Plural
}

type Registry struct {
	BaseURL string
	Model   *Model `json:"-"`

	ID          string
	Name        string
	Description string
	SpecVersion string
	Tags        map[string]string
	Docs        string

	GroupCollections map[string]*GroupCollection // groupType
}

func (reg *Registry) FindGroupModel(gTypePlural string) *GroupModel {
	for _, gModel := range reg.Model.Groups {
		if strings.EqualFold(gModel.Plural, gTypePlural) {
			return gModel
		}
	}
	return nil
}

func (reg *Registry) FindOrAddGroupCollection(gType string) *GroupCollection {
	gc, _ := reg.GroupCollections[gType]
	if gc == nil {
		gm := reg.Model.Groups[gType]
		if gm == nil {
			panic(fmt.Sprintf("Can't find GroupModel %q", gType))
		}

		gc = &GroupCollection{
			Registry:   reg,
			GroupModel: gm,
			Groups:     map[string]*Group{},
		}
		if reg.GroupCollections == nil {
			reg.GroupCollections = map[string]*GroupCollection{}
		}
		reg.GroupCollections[gType] = gc
	}
	return gc
}

func (reg *Registry) AddGroup(gt string, id string) *Group {
	gc := reg.FindOrAddGroupCollection(gt)
	return gc.NewGroup(id)
}

func (reg *Registry) FindOrAddGroup(gt string, id string) *Group {
	gc := reg.FindOrAddGroupCollection(gt)
	g := gc.Groups[id]
	if g != nil {
		return g
	}
	return gc.NewGroup(id)
}

type GroupCollection struct {
	Registry   *Registry
	GroupModel *GroupModel
	Groups     map[string]*Group // id->*Group
}

func (gc *GroupCollection) NewGroup(id string) *Group {
	group := &Group{
		GroupCollection:     gc,
		ID:                  id,
		Name:                id,
		Epoch:               0,
		ResourceCollections: map[string]*ResourceCollection{}, // id
	}
	gc.Groups[id] = group
	return group
}

func (gc *GroupCollection) FindByID(id string) {
}

func (gc *GroupCollection) ToObject(ctx *Context) *Object {
	obj := NewObject()
	if gc == nil {
		return obj
	}
	for _, key := range SortedKeys(gc.Groups) {
		group := gc.Groups[key]
		ctx.DataPush(group.ID)
		obj.AddProperty(group.ID, gc.Groups[key].ToObject(ctx))
		ctx.DataPop()
	}

	return obj
}

func (gc *GroupCollection) ToJSON(ctx *Context) {
	ctx.Print("{\n")
	ctx.Indent()

	for gCount, key := range SortedKeys(gc.Groups) {
		group := gc.Groups[key]
		if gCount > 0 {
			ctx.Print(",\n")
		}
		ctx.DataPush(group.ID)
		ctx.Printf("\t\"%s\": ", group.ID)
		group.ToJSON(ctx)
		ctx.DataPop()
	}

	ctx.Print("\n")
	ctx.Outdent()
	ctx.Print("\t}")
}

func (gc *GroupCollection) FindGroup(gID string) *Group {
	for _, group := range gc.Groups {
		if strings.EqualFold(group.ID, gID) {
			return group
		}
	}
	return nil
}

type Group struct {
	GroupCollection *GroupCollection

	ID                  string
	Name                string
	Epoch               int
	ResourceCollections map[string]*ResourceCollection
}

func (g *Group) ToObject(ctx *Context) *Object {
	obj := NewObject()
	if g == nil {
		return obj
	}
	obj.AddProperty("id", g.ID)
	obj.AddProperty("name", g.Name)
	obj.AddProperty("epoch", g.Epoch)
	obj.AddProperty("self", ctx.DataURL())

	for i, key := range SortedKeys(g.GroupCollection.GroupModel.Resources) {
		rType := g.GroupCollection.GroupModel.Resources[key]
		rColl := g.ResourceCollections[rType.Plural]

		obj.AddProperty(rColl.ResourceModel.Plural+"URL",
			URLBuild(ctx.DataURL(), rColl.ResourceModel.Plural))

		ctx.ModelPush(rColl.ResourceModel.Plural)
		ctx.DataPush(rColl.ResourceModel.Plural)
		resObj := rColl.ToObject(ctx)
		ctx.DataPop()
		ctx.ModelPop()

		obj.AddProperty(rColl.ResourceModel.Plural+"Count", resObj.Len())

		if ctx.ShouldInline(rColl.ResourceModel.Plural) {
			obj.AddProperty(rColl.ResourceModel.Plural, resObj)
			if i+1 != len(g.GroupCollection.GroupModel.Resources) {
				obj.AddProperty("", "")
			}
		}
	}

	return obj
}

func (g *Group) ToJSON(ctx *Context) {
	ctx.Print("{\n")
	ctx.Indent()

	ctx.Printf("\t\"id\": \"%s\",\n", g.ID)
	ctx.Printf("\t\"name\": \"%s\",\n", g.Name)
	ctx.Printf("\t\"epoch\": %d,\n", g.Epoch)
	ctx.Printf("\t\"self\": \"%s\",\n", ctx.DataURL())

	for rCount, key := range SortedKeys(g.GroupCollection.GroupModel.Resources) {
		rType := g.GroupCollection.GroupModel.Resources[key]
		rColl := g.ResourceCollections[rType.Plural]

		if rCount > 0 {
			ctx.Print(",\n")
		}
		ctx.Printf("\t\"%sURL\": \"%s\",\n",
			rColl.ResourceModel.Plural,
			URLBuild(ctx.DataURL(), rColl.ResourceModel.Plural))
		ctx.Printf("\t\"%sCount\": %d",
			rColl.ResourceModel.Plural,
			len(rColl.Resources))

		if ctx.ShouldInline(rColl.ResourceModel.Plural) && len(rColl.Resources) > 0 {
			ctx.ModelPush(rColl.ResourceModel.Plural)
			ctx.DataPush(rColl.ResourceModel.Plural)
			ctx.Print(",\n")
			ctx.Printf("\t\"%s\": ", rColl.ResourceModel.Plural)
			rColl.ToJSON(ctx)
			ctx.DataPop()
			ctx.ModelPop()
		}
	}

	ctx.Print("\n")
	ctx.Outdent()
	ctx.Print("\t}")
}

type ResourceCollection struct {
	Group         *Group
	ResourceModel *ResourceModel
	Resources     map[string]*Resource // id
}

func (rc *ResourceCollection) ToObject(ctx *Context) *Object {
	obj := NewObject()
	if rc == nil {
		return obj
	}

	for _, key := range SortedKeys(rc.Resources) {
		resource := rc.Resources[key]
		ctx.DataPush(resource.ID)
		obj.AddProperty(resource.ID, resource.ToObject(ctx))
		ctx.DataPop()
	}

	return obj
}

func (rc *ResourceCollection) ToJSON(ctx *Context) {
	ctx.Print("{\n")
	ctx.Indent()

	for rCount, key := range SortedKeys(rc.Resources) {
		resource := rc.Resources[key]
		if rCount > 0 {
			ctx.Print(",\n")
		}
		ctx.DataPush(resource.ID)
		ctx.Printf("\t\"%s\": ", resource.ID)
		resource.ToJSON(ctx)
		ctx.DataPop()
	}

	ctx.Print("\n")
	ctx.Outdent()
	ctx.Print("\t}")
}

func (g *Group) FindOrAddResourceCollection(rType string) *ResourceCollection {
	rc, _ := g.ResourceCollections[rType]
	if rc == nil {
		rm := g.GroupCollection.GroupModel.Resources[rType]
		if rm == nil {
			panic(fmt.Sprintf("Can't find ResourceModel %q", rType))
		}

		rc = &ResourceCollection{
			Group:         g,
			ResourceModel: rm,
			Resources:     map[string]*Resource{},
		}
		g.ResourceCollections[rType] = rc
	}
	return rc
}

func (g *Group) FindOrAddResource(rType string, id string) *Resource {
	rc := g.FindOrAddResourceCollection(rType)
	res := rc.Resources[id]
	if res != nil {
		return res
	}
	return rc.NewResource(id)
}

func (rc *ResourceCollection) NewResource(id string) *Resource {
	res := &Resource{
		ResourceCollection: rc,
		ID:                 id,
		Latest:             "",
		VersionCollection:  &VersionCollection{},
	}

	rc.Resources[id] = res
	return res
}

type Resource struct {
	ResourceCollection *ResourceCollection

	ID                string
	Latest            string
	VersionCollection *VersionCollection // map[string]*Version // version
}

func (r *Resource) FindVersion(verString string) *Version {
	return r.VersionCollection.Versions[verString]
}

func (r *Resource) FindOrAddVersion(verStr string) *Version {
	ver := r.VersionCollection.Versions[verStr]
	if ver != nil {
		return ver
	}
	ver = &Version{
		Resource: r,
		ID:       verStr,
		Version:  verStr,
		Data:     map[string]interface{}{},
	}

	if r.VersionCollection.Versions == nil {
		r.VersionCollection.Versions = map[string]*Version{}
	}

	r.VersionCollection.Versions[verStr] = ver

	if r.Latest == "" {
		r.Latest = verStr
	}
	return ver
}

func (r *Resource) ToObject(ctx *Context) *Object {
	obj := NewObject()
	if r == nil {
		return obj
	}

	var latest *Version
	if r.Latest != "" {
		latest = r.VersionCollection.Versions[r.Latest]
	}
	if latest == nil {
		panic("Help")
		for _, latest = range r.VersionCollection.Versions {
			break
		}
	}

	obj.AddProperty("id", r.ID)
	latest.AddToJsonInner(ctx, obj)

	myURI := URLBuild(ctx.DataURL())
	mySelf := myURI
	if mySelf[0] != '#' {
		mySelf += "?self"
	}

	contentURI := latest.Data["resourceURI"]
	if contentURI == nil {
		contentURI = myURI
	}

	obj.AddProperty(r.ResourceCollection.ResourceModel.Singular+"URI",
		contentURI)
	obj.AddProperty("self", mySelf)

	ctx.ModelPush("versions")
	ctx.DataPush("versions")
	vers := r.VersionCollection.ToObject(ctx)
	ctx.DataPop()
	ctx.ModelPop()

	if ctx.ShouldInline("versions") && vers.Len() > 0 {
		obj.AddProperty("versions", vers)
	}

	return obj
}

func (r *Resource) ToJSON(ctx *Context) {
	var latest *Version
	if r.Latest != "" {
		latest = r.VersionCollection.Versions[r.Latest]
	}
	if latest == nil {
		panic("Help")
		for _, latest = range r.VersionCollection.Versions {
			break
		}
	}

	ctx.Print("{\n")
	ctx.Indent()

	ctx.Printf("\t\"id\": \"%s\",\n", r.ID)
	latest.ToJSONInner(ctx)

	myURI := URLBuild(ctx.DataURL())
	mySelf := myURI
	if mySelf[0] != '#' {
		mySelf += "?self"
	}

	contentURI := latest.Data["resourceURI"]
	if contentURI == nil {
		contentURI = myURI
	}

	ctx.Printf("\t\"%sURI\": \"%s\",\n",
		r.ResourceCollection.ResourceModel.Singular,
		contentURI)
	ctx.Printf("\t\"self\": \"%s\"", mySelf)

	if ctx.ShouldInline("versions") && len(r.VersionCollection.Versions) > 0 {
		ctx.Print(",\n")
		ctx.ModelPush("versions")
		ctx.DataPush("versions")
		ctx.Print("\t\"versions\": ")
		r.VersionCollection.ToJSON(ctx)
		ctx.DataPop()
		ctx.ModelPop()
	}

	ctx.Print("\n")
	ctx.Outdent()
	ctx.Print("\t}")
}

type VersionCollection struct {
	Resource *Resource
	Versions map[string]*Version // version
}

func (vc *VersionCollection) ToObject(ctx *Context) *Object {
	obj := NewObject()
	if vc == nil {
		return obj
	}

	for _, key := range SortedKeys(vc.Versions) {
		ver := vc.Versions[key]
		ctx.DataPush(ver.ID)
		obj.AddProperty(ver.ID, ver.ToObject(ctx))
		ctx.DataPop()
	}

	return obj
}

func (vc *VersionCollection) ToJSON(ctx *Context) {
	ctx.Print("{\n")
	ctx.Indent()

	for vCount, key := range SortedKeys(vc.Versions) {
		ver := vc.Versions[key]
		if vCount > 0 {
			ctx.Print(",\n")
		}
		ctx.DataPush(ver.ID)
		ctx.Printf("\t\"%s\": ", ver.ID)
		ver.ToJSON(ctx)
		ctx.DataPop()
	}

	ctx.Print("\n")
	ctx.Outdent()
	ctx.Print("\t}")
}

type Version struct {
	Resource *Resource

	ID      string
	Name    string
	Type    string
	Version string
	Epoch   int

	Data map[string]interface{}
}

func (v *Version) ToObject(ctx *Context) *Object {
	obj := NewObject()
	if v == nil {
		return obj
	}

	obj.AddProperty("id", v.ID)
	v.AddToJsonInner(ctx, obj)

	myURI := ctx.DataURL()
	mySelf := myURI
	if mySelf[0] != '#' {
		mySelf += "?self"
	}

	contentURI := v.Data["resourceURI"]
	if contentURI == nil {
		contentURI = myURI
	}

	obj.AddProperty(v.Resource.ResourceCollection.ResourceModel.Singular+"URI",
		contentURI)
	obj.AddProperty("self", mySelf)

	return obj
}

func (v *Version) ToJSON(ctx *Context) {
	ctx.Print("{\n")
	ctx.Indent()

	ctx.Printf("\t\"id\": \"%s\",\n", v.ID)
	v.ToJSONInner(ctx)

	myURI := ctx.DataURL()
	mySelf := myURI
	if mySelf[0] != '#' {
		mySelf += "?self"
	}

	contentURI := v.Data["resourceURI"]
	if contentURI == nil {
		contentURI = myURI
	}

	ctx.Printf("\t\"%sURI\": \"%s\",\n",
		v.Resource.ResourceCollection.ResourceModel.Singular,
		contentURI)
	ctx.Printf("\t\"self\": \"%s\"\n", mySelf)

	ctx.Outdent()
	ctx.Print("\t}")
}

func (v *Version) AddToJsonInner(ctx *Context, obj *Object) {
	obj.AddProperty("name", v.Name)
	obj.AddProperty("type", v.Type)
	obj.AddProperty("version", v.Version)
	obj.AddProperty("epoch", v.Epoch)
}

func (v *Version) ToJSONInner(ctx *Context) {
	ctx.Printf("\t\"name\": \"%s\",\n", v.Name)
	ctx.Printf("\t\"type\": \"%s\",\n", v.Type)
	ctx.Printf("\t\"version\": \"%s\",\n", v.Version)
	ctx.Printf("\t\"epoch\": %d,\n", v.Epoch)
}

func JSONEscape(obj interface{}) string {
	buf, _ := json.Marshal(obj)
	return string(buf[1 : len(buf)-1])
}

func URLBuild(base string, paths ...string) string {
	isFrag := strings.Index(base, "#") >= 0
	url := base
	url = strings.TrimRight(url, "/")

	for _, path := range paths {
		if isFrag {
			url += "/" + path
		} else {
			url += "/" + strings.ToLower(path)
		}
	}
	return url
}

func SortedKeys(m interface{}) []string {
	mk := reflect.ValueOf(m).MapKeys()

	keys := make([]string, 0, len(mk))
	for _, k := range mk {
		keys = append(keys, k.String())
	}
	sort.Strings(keys)
	return keys
}

func (reg *Registry) ToObject(ctx *Context) *Object {
	obj := NewObject()
	if reg == nil {
		return obj
	}

	obj.AddProperty("id", reg.ID)
	obj.AddProperty("name", reg.Name)
	obj.AddProperty("description", reg.Description)
	obj.AddProperty("specVersion", reg.SpecVersion)
	obj.AddProperty("self", ctx.DataURL())

	tags := NewObject()
	for key, value := range reg.Tags {
		tags.AddProperty(key, value)
	}
	if len(tags.Children) != 0 {
		obj.AddProperty("tags", tags)
	}

	obj.AddProperty("docs", reg.Docs)
	obj.AddProperty("", "")

	if ctx.ShouldInline("model") {
		ctx.ModelPush("model")
		obj.AddProperty("model", reg.Model.ToObject(ctx))
		ctx.ModelPop()
		obj.AddProperty("", "")
	}

	for i, key := range SortedKeys(reg.Model.Groups) {
		gType := reg.Model.Groups[key]
		gCollection := reg.GroupCollections[gType.Plural]

		obj.AddProperty(gType.Plural+"URL",
			URLBuild(ctx.DataURL(), gType.Plural))

		ctx.DataPush(gType.Plural)
		ctx.ModelPush(gType.Plural)
		groupObj := NewObject()
		if gCollection != nil {
			groupObj = gCollection.ToObject(ctx)
		}
		ctx.ModelPop()
		ctx.DataPop()

		obj.AddProperty(gType.Plural+"Count", groupObj.Len())
		if ctx.ShouldInline(gType.Plural) {
			obj.AddProperty(gType.Plural, groupObj)
		}
		if i+1 != len(reg.Model.Groups) {
			obj.AddProperty("", "")
		}
	}

	return obj
}

func (reg *Registry) ToJSON(ctx *Context) {
	ctx.Print("{\n")

	ctx.Indent()

	ctx.Printf("\t\"id\": \"%s\",\n", reg.ID)
	if reg.Name != "" {
		ctx.Printf("\t\"name\": \"%s\",\n", reg.Name)
	}
	if reg.Description != "" {
		ctx.Printf("\t\"description\": \"%s\",\n", reg.Description)
	}
	ctx.Printf("\t\"specVersion\": \"%s\",\n", reg.SpecVersion)
	ctx.Printf("\t\"self\": \"%s\",\n", ctx.DataURL())
	if len(reg.Tags) > 0 {
		ctx.Print("\t\"tags\": {\n")
		ctx.Indent()
		count := 0
		for key, value := range reg.Tags {
			count++
			ctx.Printf("\t\"%s\": \"%s\"", key, value)
			if count != len(reg.Tags) {
				ctx.Print(",")
			}
			ctx.Print("\n")
		}
		ctx.Outdent()
		ctx.Print("\t},\n")
	}
	if reg.Docs != "" {
		ctx.Printf("\t\"docs\": \"%s\",\n", reg.Docs)
	}

	// Add the Registry model
	if ctx.ShouldInline("model") {
		ctx.ModelPush("model")
		ctx.Print("\n")
		ctx.Print("\t\"model\": ")
		reg.Model.ToJSON(ctx)
		ctx.Print(",\n")
		ctx.ModelPop()
	}

	for gCount, key := range SortedKeys(reg.Model.Groups) {
		gType := reg.Model.Groups[key]
		gCollection := reg.GroupCollections[gType.Plural]

		if gCount > 0 {
			ctx.Print(",\n")
		}
		ctx.Print("\n")
		ctx.Printf("\t\"%sURL\": \"%s\",\n",
			gType.Plural,
			URLBuild(ctx.DataURL(), gType.Plural))

		l := 0
		if gCollection != nil {
			l = len(gCollection.Groups)
		}
		ctx.Printf("\t\"%sCount\": %d", gType.Plural, l)

		if ctx.ShouldInline(gType.Plural) && l > 0 {
			ctx.DataPush(gType.Plural)
			ctx.ModelPush(gType.Plural)
			ctx.Print(",\n")
			ctx.Printf("\t\"%s\": ", gType.Plural)
			gCollection.ToJSON(ctx)
			ctx.ModelPop()
			ctx.DataPop()
		}
	}
	ctx.Print("\n}")
}

func (m *Model) ToObject(ctx *Context) *Object {
	obj := NewObject()
	if m == nil {
		return obj
	}

	groups := NewObject()
	for _, key := range SortedKeys(m.Groups) {
		group := m.Groups[key]
		groupObj := NewObject()
		groupObj.AddProperty("singular", group.Singular)
		groupObj.AddProperty("plural", group.Plural)
		groupObj.AddProperty("schema", group.Schema)

		resObjs := NewObject()
		for _, key := range SortedKeys(group.Resources) {
			res := group.Resources[key]
			resObj := NewObject()
			resObj.AddProperty("singular", res.Singular)
			resObj.AddProperty("plural", res.Plural)
			resObj.AddProperty("versions", res.Versions)
			resObjs.AddProperty(key, resObj)
		}

		groupObj.AddProperty("resources", resObjs)
		groups.AddProperty(key, groupObj)
	}
	obj.AddProperty("groups", groups)
	return obj
}

func (m *Model) ToJSON(ctx *Context) {
	ctx.Print("{\n")
	ctx.Indent()

	ctx.Print("\t\"groups\": {\n")
	ctx.Indent()
	for gCount, key := range SortedKeys(m.Groups) {
		group := m.Groups[key]
		if gCount > 0 {
			ctx.Print(",\n")
		}

		ctx.Printf("\t\"%s\": {\n", key)
		ctx.Indent()
		ctx.Printf("\t\"singular\": \"%s\",\n", group.Singular)
		ctx.Printf("\t\"plural\": \"%s\",\n", group.Plural)
		ctx.Printf("\t\"schema\": \"%s\",\n", group.Schema)
		ctx.Print("\t\"resources\": {\n")
		ctx.Indent()

		for rCount, key := range SortedKeys(group.Resources) {
			res := group.Resources[key]
			if rCount > 0 {
				ctx.Print(",")
			}

			ctx.Printf("\t\"%s\": {\n", key)
			ctx.Indent()
			ctx.Printf("\t\"singular\": \"%s\",\n", res.Singular)
			ctx.Printf("\t\"plural\": \"%s\",\n", res.Plural)
			ctx.Printf("\t\"versions\": %d\n", res.Versions)
			ctx.Outdent()
			ctx.Print("\t}")
		}
		ctx.Print("\n")

		ctx.Outdent()
		ctx.Print("\t}\n")
		ctx.Outdent()
		ctx.Print("\t}")
	}
	ctx.Print("\n")
	ctx.Outdent()
	ctx.Print("\t}\n")
	ctx.Outdent()
	ctx.Print("\t}")
}

type RegistryFlags struct {
	BaseURL     string
	Indent      string
	InlineAll   bool
	InlinePaths []string
	Self        bool
	AsDoc       bool
}

type Context struct {
	Flags     *RegistryFlags
	BaseURL   string
	DataPath  string
	ModelPath string

	currentIndent string
	indent        string

	buffer strings.Builder
}

func (c *Context) Printf(format string, args ...interface{}) {
	c.Print(fmt.Sprintf(format, args...))
}

func (c *Context) Print(str string) {
	if str[0] == '\t' {
		c.buffer.WriteString(c.currentIndent)
		str = str[1:]
	}
	c.buffer.WriteString(str)
}

func (c *Context) Result() string {
	return c.buffer.String()
}

func (c *Context) Spaces() string {
	return c.currentIndent
}

func (c *Context) Indent() string {
	c.currentIndent += c.indent
	return c.currentIndent
}

func (c *Context) Outdent() string {
	c.currentIndent = c.currentIndent[:len(c.currentIndent)-len(c.indent)]
	return c.currentIndent
}

func (c *Context) Sprintf(str string, args ...interface{}) string {
	return fmt.Sprintf(c.Spaces()+str, args...)
}

func (c *Context) BaseURLPush(word string) string {
	c.BaseURL += "/" + word
	return c.BaseURL
}

func (c *Context) DataURL() string {
	if c.Flags.AsDoc {
		return "#" + "/" + c.DataPath
	}
	return c.BaseURL + "/" + strings.ToLower(c.DataPath)
}

func (c *Context) DocifyURL(daURL string) string {
	if c.Flags.AsDoc && strings.HasPrefix(daURL, c.BaseURL) {
		return "#" + daURL[len(c.BaseURL):]
	}
	return daURL
}

func (c *Context) DataPush(word string) string {
	if c.DataPath != "" {
		c.DataPath += "/"
	}
	c.DataPath += word
	return c.DataPath
}

func (c *Context) DataPop() string {
	if c.DataPath == "" {
		panic("Popping empty DataPath")
	}
	if i := strings.LastIndex(c.DataPath, "/"); i >= 0 {
		c.DataPath = c.DataPath[:i]
	} else {
		c.DataPath = ""
	}
	return c.DataPath
}

func (c *Context) ModelPush(word string) string {
	if c.ModelPath != "" {
		c.ModelPath += "."
	}
	c.ModelPath += word
	return c.ModelPath
}

func (c *Context) ModelPop() string {
	if c.ModelPath == "" {
		panic("Popping empty ModelPath")
	}
	if i := strings.LastIndex(c.ModelPath, "."); i >= 0 {
		c.ModelPath = c.ModelPath[:i]
	} else {
		c.ModelPath = ""
	}
	return c.ModelPath
}

func (c *Context) ShouldInline(section string) bool {
	sectionPath := c.ModelPath
	if sectionPath != "" {
		sectionPath += "."
	}
	sectionPath += section

	if c.Flags.InlineAll {
		return true
	}
	for _, path := range c.Flags.InlinePaths {
		// fmt.Printf("%s -> %s\n", sectionPath, path)
		if path == sectionPath {
			return true
		}
		if strings.HasPrefix(path, sectionPath) {
			return true
		}
		if path[0] == '.' && strings.HasSuffix(sectionPath, path) {
			return true
		}
	}
	return false
}

func (r *Registry) Get(path string, rFlags *RegistryFlags) (string, error) {
	paths := strings.Split(strings.Trim(path, "/"), "/")
	for len(paths) > 0 && paths[0] == "" {
		paths = paths[1:]
	}
	fmt.Printf("Paths: %v\n", paths)

	ctx := &Context{
		Flags:         rFlags,
		BaseURL:       r.BaseURL,
		DataPath:      "",
		ModelPath:     "",
		currentIndent: "",
		indent:        rFlags.Indent,
	}

	if rFlags.BaseURL != "" {
		ctx.BaseURL = rFlags.BaseURL
	}
	ctx.BaseURL = strings.TrimRight(ctx.BaseURL, "/")

	if len(paths) == 0 {
		r.ToObject(ctx).ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	if len(paths) == 1 && paths[0] == "model" {
		r.Model.ToObject(ctx).ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	// GROUPs
	var gModel *GroupModel
	if gModel = r.FindGroupModel(paths[0]); gModel == nil {
		return "", fmt.Errorf("Unknown group %q", paths[0])
	}
	groupColl := r.GroupCollections[gModel.Plural]
	ctx.BaseURLPush(paths[0])

	if len(paths) == 1 {
		groupColl.ToObject(ctx).ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	// GROUPs/ID
	group := groupColl.Groups[paths[1]]
	if group == nil {
		return "", fmt.Errorf("Unknown group ID %q", paths[1])
	}
	ctx.BaseURLPush(paths[1])
	if len(paths) == 2 {
		group.ToObject(ctx).ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	// GROUPs/ID/RESOURCEs
	resColl := group.ResourceCollections[paths[2]]
	ctx.BaseURLPush(paths[2])
	if resColl == nil {
		return "", fmt.Errorf("Unknown rescource collection %q", paths[2])
	}
	if len(paths) == 3 {
		resColl.ToObject(ctx).ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	// GROUPs/ID/RESOURCEs/ID
	res := resColl.Resources[paths[3]]
	ctx.BaseURLPush(paths[3])
	if res == nil {
		return "", fmt.Errorf("Unknown resource ID %q", paths[3])
	}

	if len(paths) == 4 {
		if ctx.Flags.Self {
			res.ToObject(ctx).ToJson(&ctx.buffer, "", "  ")
			return ctx.buffer.String(), nil
		}
		latest := res.FindVersion(res.Latest)
		data, ok := latest.Data["resourceContent"]
		if ok {
			str, _ := data.(string)
			return str, nil
		}

		uri, ok := latest.Data["resourceProxyURI"]
		if ok {
			resp, err := http.Get(uri.(string))
			if err != nil {
				return "", err
			}
			if resp.StatusCode/100 != 2 {
				return "", fmt.Errorf("%s ->%d",
					uri, resp.StatusCode)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", err
			}
			return string(body), nil
		}
	}

	// GROUPs/ID/RESOURCEs/ID/versions
	if paths[4] != "versions" {
		return "", fmt.Errorf("Unknown subresource %q", paths[4])
	}
	verColl := res.VersionCollection
	ctx.BaseURLPush(paths[4])
	if len(paths) == 5 {
		verColl.ToObject(ctx).ToJson(&ctx.buffer, "", "  ")
		return ctx.buffer.String(), nil
	}

	// GROUPs/ID/RESOURCEs/ID/versions/ID
	ver := verColl.Versions[paths[5]]
	if ver == nil {
		return "", fmt.Errorf("Unknown version id %q", paths[5])
	}

	ctx.BaseURLPush(paths[5])
	if len(paths) == 6 {
		if ctx.Flags.Self {
			ver.ToObject(ctx).ToJson(&ctx.buffer, "", "  ")
			return ctx.buffer.String(), nil
		}
		data, ok := ver.Data["resourceContent"]
		if ok {
			str, _ := data.(string)
			return str, nil
		}

		uri, ok := ver.Data["resourceProxyURI"]
		if ok {
			resp, err := http.Get(uri.(string))
			if err != nil {
				return "", err
			}
			if resp.StatusCode/100 != 2 {
				return "", fmt.Errorf("%s ->%d",
					uri, resp.StatusCode)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", err
			}
			return string(body), nil
		}
	}

	return "", fmt.Errorf("Can't figure out what to do with %q",
		strings.Join(paths, "/"))

	return ctx.buffer.String(), nil

	/*
	   	for _, groupColl := range r.GroupCollections {
	   		if strings.ToLower(groupColl.GroupModel.Plural) != paths[0] {
	   			continue
	   		}
	   		ctx.BaseURLPush(paths[0])
	   		if len(paths) == 1 {
	   			groupColl.ToJSON(ctx)
	   			return ctx.Result(), nil
	   		}

	   		// GROUPs/ID
	   		for _, group := range groupColl.Groups {
	   			if group.ID != paths[1] {
	   				continue
	   			}
	   			ctx.BaseURLPush(paths[1])
	   			if len(paths) == 2 {
	   				group.ToJSON(ctx)
	   				return ctx.Result(), nil
	   			}

	   			// GROUPs/ID/RESOURCEs
	   			for _, resColl := range group.ResourceCollections {
	   				if strings.ToLower(resColl.ResourceModel.Plural) != paths[2] {
	   					continue
	   				}
	   				ctx.BaseURLPush(paths[2])
	   				if len(paths) == 3 {
	   					resColl.ToJSON(ctx)
	   					return ctx.Result(), nil
	   				}

	   				// GROUPs/ID/RESOURCEs/ID
	   				for _, res := range resColl.Resources {
	   					if res.ID != paths[3] {
	   						continue
	   					}
	   					ctx.BaseURLPush(paths[3])

	   					if len(paths) == 4 {
	   						if ctx.Flags.Self {
	   							res.ToJSON(ctx)
	   							return ctx.Result(), nil
	   						}
	   						latest := res.FindVersion(res.Latest)
	   						data, ok := latest.Data["resourceContent"]
	   						if ok {
	   							str, _ := data.(string)
	   							return str, nil
	   						}

	   						uri, ok := latest.Data["resourceProxyURI"]
	   						if ok {
	   							resp, err := http.Get(uri.(string))
	   							if err != nil {
	   								return "", err
	   							}
	   							if resp.StatusCode/100 != 2 {
	   								return "", fmt.Errorf("%s ->%d",
	   									uri, resp.StatusCode)
	   							}
	   							body, err := io.ReadAll(resp.Body)
	   							if err != nil {
	   								return "", err
	   							}
	   							return string(body), nil
	   						}
	   					}

	   					// GROUPs/ID/RESOURCEs/ID/versions
	   					if paths[4] == "versions" {
	   						ctx.BaseURLPush(paths[4])
	   						if len(paths) == 5 {
	   							res.VersionCollection.ToJSON(ctx)
	   							return ctx.Result(), nil
	   						}

	   						// GROUPs/ID/RESOURCEs/ID/versions/ID
	   						for _, ver := range res.VersionCollection.Versions {
	   							if ver.ID == paths[5] {
	   								ctx.BaseURLPush(paths[5])
	   								if len(paths) == 6 {
	   									if ctx.Flags.Self {
	   										ver.ToJSON(ctx)
	   										return ctx.Result(), nil
	   									}
	   									data, ok := ver.Data["resourceContent"]
	   									if ok {
	   										str, _ := data.(string)
	   										return str, nil
	   									}

	   									uri, ok := ver.Data["resourceProxyURI"]
	   									if ok {
	   										resp, err := http.Get(uri.(string))
	   										if err != nil {
	   											return "", err
	   										}
	   										if resp.StatusCode/100 != 2 {
	   											return "", fmt.Errorf("%s ->%d",
	   												uri, resp.StatusCode)
	   										}
	   										body, err := io.ReadAll(resp.Body)
	   										if err != nil {
	   											return "", err
	   										}
	   										return string(body), nil
	   									}
	   								}
	   							}
	   						}
	   					}
	   				}
	   			}
	   		}
	   	}

	   return "", fmt.Errorf("Can't figure out what to do with %q",

	   	strings.Join(paths, "/"))
	*/
}
