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
