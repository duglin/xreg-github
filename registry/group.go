package registry

import (
	"fmt"

	log "github.com/duglin/dlog"
)

type Group struct {
	Entity
	Registry *Registry
}

var _ EntitySetter = &Group{}

func (g *Group) Get(name string) any {
	return g.Entity.Get(name)
}

func (g *Group) SetCommit(name string, val any) error {
	return g.Entity.eSetCommit(name, val)
}

func (g *Group) JustSet(name string, val any) error {
	return g.Entity.eJustSet(NewPPP(name), val)
}

func (g *Group) SetSave(name string, val any) error {
	return g.Entity.eSetSave(name, val)
}

func (g *Group) Delete() error {
	log.VPrintf(3, ">Enter: Group.Delete(%s)", g.UID)
	defer log.VPrintf(3, "<Exit: Group.Delete")

	if err := g.Registry.Touch(); err != nil {
		return err
	}

	return DoOne(g.tx, `DELETE FROM "Groups" WHERE SID=?`, g.DbSID)
}

func (g *Group) FindResource(rType string, id string, anyCase bool) (*Resource, error) {
	log.VPrintf(3, ">Enter: FindResource(%s,%s,%v)", rType, id, anyCase)
	defer log.VPrintf(3, "<Exit: FindResource")

	ent, err := RawEntityFromPath(g.tx, g.Registry.DbSID,
		g.Plural+"/"+g.UID+"/"+rType+"/"+id, anyCase)
	if err != nil {
		return nil, fmt.Errorf("Error finding Resource %q(%s): %s",
			id, rType, err)
	}
	if ent == nil {
		log.VPrintf(3, "None found")
		return nil, nil
	}

	return &Resource{Entity: *ent, Group: g}, nil
}

func (g *Group) AddResource(rType string, id string, vID string) (*Resource, error) {
	return g.AddResourceWithObject(rType, id, vID, nil, false, false)
}

func (g *Group) AddResourceWithObject(rType string, id string, vID string, obj Object, doChildren bool, objIsVer bool) (*Resource, error) {

	r, _, err := g.UpsertResourceWithObject(rType, id, vID, obj,
		ADD_ADD, doChildren, objIsVer)
	return r, err
}

func (g *Group) UpsertResource(rType string, id string, vID string) (*Resource, bool, error) {
	return g.UpsertResourceWithObject(rType, id, vID, nil, ADD_ADD, false, false)
}

// Return: *Resource, isNew, error
func (g *Group) UpsertResourceWithObject(rType string, id string, vID string, obj Object, addType AddType, doChildren bool, objIsVer bool) (*Resource, bool, error) {
	log.VPrintf(3, ">Enter: UpsertResourceWithObject(%s,%s)", rType, id)
	defer log.VPrintf(3, "<Exit: UpsertResourceWithObject")

	// vID is the version ID we want to use for the update/create.
	// A value of "" means just use the default Version

	rModel := g.Registry.Model.Groups[g.Plural].Resources[rType]
	if rModel == nil {
		return nil, false, fmt.Errorf("Unknown Resource type (%s) for Group %q",
			rType, g.Plural)
	}

	r, err := g.FindResource(rType, id, true)
	if err != nil {
		return nil, false, fmt.Errorf("Error checking for Resource(%s) %q: %s",
			rType, id, err)
	}

	// Can this ever happen??
	if r != nil && r.UID != id {
		return nil, false, fmt.Errorf("Attempting to create a Resource with "+
			"a \"%sid\" of %q, when one already exists as %q",
			rModel.Singular, id, r.UID)
	}

	if obj != nil && !IsNil(obj[rModel.Singular+"id"]) && !objIsVer {
		if id != obj[rModel.Singular+"id"] {
			return nil, false,
				fmt.Errorf(`The "%sid" attribute must be set to %q, not %q`,
					rModel.Singular, id, obj[rModel.Singular+"id"])
		}
	}

	if addType == ADD_ADD && r != nil {
		return nil, false, fmt.Errorf("Resource %q of type %q already exists",
			id, rType)
	}

	metaObj := (map[string]any)(nil)
	metaObjAny, hasMeta := obj["meta"]

	if hasMeta && !objIsVer {
		delete(obj, "meta")
	}

	if hasMeta {
		if objIsVer {
			return nil, false, fmt.Errorf("Can't include a Version with a " +
				"\"meta\" attribute")
		}

		if IsNil(metaObjAny) {
			// Convert "null" to empty {}
			metaObjAny = map[string]any{}
		}

		metaObj = metaObjAny.(map[string]any)
	}

	// List of versions in the incoming request
	versions := map[string]any(nil)

	if !objIsVer {
		// If obj is for the resource then save and delete the versions
		// collection (and it's attributes) so we don't try to save them
		// as extensions on the Resource
		var ok bool
		val, _ := obj["versions"]
		if !IsNil(val) {
			versions, ok = val.(map[string]any)
			if !ok {
				return nil, false,
					fmt.Errorf("Attribute %q doesn't appear to be of a "+
						"map of %q", "versions", "versions")
			}
		}

		// Remove the "versions" collection attributes
		delete(obj, "versions")
		delete(obj, "versionscount")
		delete(obj, "versionsurl")
	} else {
		if _, ok := obj["versions"]; ok {
			return nil, false, fmt.Errorf("Can't create a Version with a " +
				"\"versions\" attribute")
		}
		if _, ok := obj["versionscount"]; ok {
			return nil, false, fmt.Errorf("Can't create a Version with a " +
				"\"versionscount\" attribute")
		}
		if _, ok := obj["versionsurl"]; ok {
			return nil, false, fmt.Errorf("Can't create a Version with a " +
				"\"versionsurl\" attribute")
		}
	}

	isNew := (r == nil)
	if r == nil {
		// If Resource doesn't exist, go ahead and create it.
		// This will not create any Versions yet, just the Resource
		r = &Resource{
			Entity: Entity{
				tx: g.tx,

				Registry: g.Registry,
				DbSID:    NewUUID(),
				Plural:   rType,
				Singular: rModel.Singular,
				UID:      id,

				Type:     ENTITY_RESOURCE,
				Path:     g.Plural + "/" + g.UID + "/" + rType + "/" + id,
				Abstract: g.Plural + string(DB_IN) + rType,
			},
			Group: g,
		}

		if err = g.Touch(); err != nil {
			return nil, false, err
		}

		m := &Meta{
			Entity: Entity{
				tx: g.tx,

				Registry: g.Registry,
				DbSID:    NewUUID(),
				Plural:   "metas",
				Singular: "meta",
				UID:      r.UID,

				Type:     ENTITY_META,
				Path:     r.Path + "/meta",
				Abstract: r.Abstract + string(DB_IN) + "meta",
			},
			Resource: r,
		}

		err = DoOne(r.tx, `
        INSERT INTO Resources(SID, UID, RegistrySID, GroupSID, ModelSID, Path, Abstract)
        SELECT ?,?,?,?,SID,?,?
        FROM ModelEntities
        WHERE RegistrySID=?
          AND ParentSID IN (
            SELECT SID FROM ModelEntities
            WHERE RegistrySID=?
            AND ParentSID IS NULL
            AND Plural=?)
            AND Plural=?`,
			r.DbSID, r.UID, g.Registry.DbSID, g.DbSID,
			g.Plural+"/"+g.UID+"/"+rType+"/"+r.UID, g.Plural+string(DB_IN)+rType,
			g.Registry.DbSID,
			g.Registry.DbSID, g.Plural,
			rType)
		if err != nil {
			return nil, false, fmt.Errorf("Error adding Resource: %s", err)
		}

		err = DoOne(r.tx, `
        INSERT INTO Metas(SID, RegistrySID, ResourceSID, Path, Abstract)
        SELECT ?,?,?,?,?`,
			m.DbSID, g.Registry.DbSID, r.DbSID,
			m.Path, m.Abstract)
		if err != nil {
			return nil, false, fmt.Errorf("Error adding Meta: %s", err)
		}

		err = m.JustSet(r.Singular+"id", r.UID)
		if err != nil {
			return nil, false, err
		}

		// Use the ID passed as an arg, not from the metadata, as the true
		// ID. If the one in the metadata differs we'll flag it down below
		err = r.SetSaveResource(r.Singular+"id", r.UID)
		if err != nil {
			return nil, false, err
		}

		r.tx.AddMeta(m)

		err = m.JustSet("#nextversionid", 1)
		if err != nil {
			return nil, false, err
		}
	}

	// Now we have a Resource.
	// Order of processing:
	// - "versions" collection if there
	// - "defaultversionsticky" flag if there
	// - "defaultversionid" flag if sticky is set
	// - Resource level properties applied to default version IFF default
	//   version wasn't already uploaded as part of the "versions" collection

	// If we're processing children, and have a versions collection, process it
	if doChildren && len(versions) > 0 {
		plural := "versions"
		singular := "version"

		for verID, val := range versions {
			verObj, ok := val.(map[string]any)
			if !ok {
				return nil, false,
					fmt.Errorf("Key %q in attribute %q doesn't "+
						"appear to be of type %q", verID, plural, singular)
			}

			_, _, err := r.UpsertVersionWithObject(verID, verObj, addType)
			if err != nil {
				return nil, false, err
			}
		}
	}

	// Process the "meta" sub-object if there
	if !IsNil(metaObj) {
		_, _, err := r.UpsertMetaWithObject(metaObj, addType)
		if err != nil {
			if isNew {
				// Needed if doing local func calls to create the Resource
				// and we don't commit/rollback the tx upon failure
				r.Delete()
			}
			return nil, false, err
		}
	}

	meta, err := r.FindMeta(false)
	PanicIf(err != nil, "No meta %q: %s", r.UID, err)

	if !IsNil(meta.Get("xref")) {
		// All versions should have been deleted alread so just return
		return r, isNew, nil
	}

	defVerID := meta.GetAsString("defaultversionid")

	if !objIsVer {
		// Clear any ID there since it's the Resource's
		delete(obj, r.Singular+"id")
	}

	attrVersionID := ""
	if val, ok := obj["versionid"]; ok {
		attrVersionID = NotNilString(&val)
	}

	// If both vID and attrVersionID are set, they MUST match if obj is
	// the Resource, not a new Version.
	// Not sure this can ever happen, but just in case...
	if !objIsVer && vID != "" && attrVersionID != "" {
		return nil, false, fmt.Errorf("The desired \"versionid\"(%s) must "+
			"match the \"versionid\" attribute(%s)", vID, attrVersionID)
	}

	// If the passed-in vID is empty, and we're new, look for "versionid"
	if vID == "" && isNew && attrVersionID != "" {
		vID = attrVersionID
	}

	// if vID is still empty, then use the defaultversionid
	if vID == "" {
		vID = defVerID
	}

	if defVerID != "" && attrVersionID != "" && attrVersionID != defVerID {
		return nil, false, fmt.Errorf("When \"versionid\"(%s) is "+
			"present it must match the \"defaultversionid\"(%s)",
			attrVersionID, defVerID)
	}

	// Update the appropriate Version (vID), but only if the versionID
	// doesn't match a Version ID from the "versions" collection (if there).
	// If both Resource attrs and Version attrs are present, use the Version's
	if vID != "" {
		if _, ok := versions[defVerID]; !ok {
			RemoveResourceAttributes(rModel.Singular, obj)
			_, _, err := r.UpsertVersionWithObject(vID, obj, addType)
			if err != nil {
				return nil, false, err
			}
		}
	} else {
		RemoveResourceAttributes(rModel.Singular, obj)
		_, _, err := r.UpsertVersionWithObject(vID, obj, addType)
		if err != nil {
			return nil, false, err
		}
	}

	// Make sure defaultversionid is set appropriately
	// TODO I don't think this is right, so comment out for now
	/*
		if !sticky {
			// Not sticky == always use latest
			if err := r.SetDefault(nil); err != nil {
				return nil, false, err
			}
		} else {
			// Note that vID can be "", in which case use latest
			if err := r.SetDefaultID(vID); err != nil {
				return nil, false, err
			}
		}
	*/

	return r, isNew, err
}
