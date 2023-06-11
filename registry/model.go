package registry

import ()

type ModelElement struct {
	Singular string
	Plural   string
	Children map[string]*ModelElement
}

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

func (m *Model) ToObject(ctx *Context) (*Object, error) {
	obj := NewObject()
	if m == nil {
		return obj, nil
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
	return obj, nil
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

func CreateGenericModel(model *Model) *ModelElement {
	newModel := &ModelElement{}

	for gKey, gModel := range model.Groups {
		newGroup := &ModelElement{
			Singular: gModel.Singular,
			Plural:   gModel.Plural,
		}

		for rKey, rModel := range gModel.Resources {
			newResource := &ModelElement{
				Singular: rModel.Singular,
				Plural:   rModel.Plural,
				Children: map[string]*ModelElement{
					"versions": &ModelElement{
						Singular: "version",
						Plural:   "versions",
					},
				},
			}

			if len(newGroup.Children) == 0 {
				newGroup.Children = map[string]*ModelElement{}
			}
			newGroup.Children[rKey] = newResource
		}

		if len(newModel.Children) == 0 {
			newModel.Children = map[string]*ModelElement{}
		}
		newModel.Children[gKey] = newGroup
	}

	return newModel
}
