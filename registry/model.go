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
	ID       string    `json:"-"`
	Registry *Registry `json:"-"`

	Plural   string `json:"plural,omitempty"`
	Singular string `json:"singular,omitempty"`
	Schema   string `json:"schema,omitempty"`

	Resources map[string]*ResourceModel `json:"resources,omitempty"` // Plural
}

type ResourceModel struct {
	ID         string      `json:"-"`
	GroupModel *GroupModel `json:"-"`

	Plural    string `json:"plural,omitempty"`
	Singular  string `json:"singular,omitempty"`
	Versions  int    `json:"versions,omitempty"`
	VersionId bool   `json:"versionId"`
	Latest    bool   `json:"latest"`
}

type Model struct {
	Registry *Registry              `json:"-"`
	Groups   map[string]*GroupModel `json:"groups,omitempty"` // Plural
}

func (g *GroupModel) AddResourceModel(plural string, singular string, versions int, verId bool, latest bool) (*ResourceModel, error) {
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
			Versions,
			VersionId,
			Latest)
		VALUES(?,?,?,?,?,?,?,?,?) `,
		mID, g.Registry.DbID, g.ID, plural, singular, nil, versions,
		verId, latest)
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
		VersionId:  verId,
		Latest:     latest,
	}

	g.Resources[plural] = r

	return r, nil
}
