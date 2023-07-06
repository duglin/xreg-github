package registry

import (
	"fmt"

	log "github.com/duglin/dlog"
)

type ModelElement struct {
	Singular string
	Plural   string
	Children map[string]*ModelElement
}

type GroupModel struct {
	ID       string
	Registry *Registry

	Plural   string `json:"plural,omitempty"`
	Singular string `json:"singular,omitempty"`
	Schema   string `json:"schema,omitempty"`
	Versions int    `json:"versions"`

	Resources map[string]*ResourceModel // Plural
}

type ResourceModel struct {
	ID         string
	GroupModel *GroupModel

	Plural   string `json:"plural,omitempty"`
	Singular string `json:"singular,omitempty"`
	Versions int    `json:"versions,omitempty"`
}

type Model struct {
	Registry *Registry
	Groups   map[string]*GroupModel // Plural
}

func (g *GroupModel) AddResourceModel(plural string, singular string, versions int) (*ResourceModel, error) {
	if plural == "" {
		return nil, fmt.Errorf("Can't add a group with an empty plural name")
	}
	if singular == "" {
		return nil, fmt.Errorf("Can't add a group with an empty sigular name")
	}
	if versions < 0 {
		return nil, fmt.Errorf("''versions'(%d) must be >= 0", versions)
	}

	mID := NewUUID()

	err := Do(`
		INSERT INTO ModelEntities(
			ID,
			RegistryID,
			ParentID,
			Plural,
			Singular,
			SchemaURL,
			Versions)
		VALUES(?,?,?,?,?,?,?) `,
		mID, g.Registry.ID, g.ID, plural, singular, nil, versions)
	if err != nil {
		log.Printf("Error inserting resourceModel(%s): %s", plural, err)
		return nil, err
	}
	r := &ResourceModel{
		ID:         mID,
		GroupModel: g,
		Singular:   singular,
		Plural:     plural,
		Versions:   versions,
	}

	g.Resources[plural] = r

	return r, nil
}
