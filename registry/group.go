package registry

import (
	"fmt"
	"strings"

	log "github.com/duglin/dlog"
)

type Group struct {
	Entity
	Registry *Registry
}

func (g *Group) Get(name string) any {
	return g.Entity.Get(name)
}

func (g *Group) SetCommit(name string, val any) error {
	return g.Entity.SetCommit(name, val)
}

func (g *Group) JustSet(name string, val any) error {
	return g.Entity.JustSet(NewPPP(name), val)
}

func (g *Group) SetSave(name string, val any) error {
	return g.Entity.SetSave(name, val)
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

	if r != nil && r.UID != id {
		return nil, false, fmt.Errorf("Attempting to create a Resource with "+
			"an \"id\" of %q, when one already exists as %q", id, r.UID)
	}

	if obj != nil && !IsNil(obj["id"]) && !objIsVer {
		if id != obj["id"] {
			return nil, false,
				fmt.Errorf(`The "id" attribute must be set `+
					`to %q, not %q`, id, obj["id"])
		}
	}

	if addType == ADD_ADD && r != nil {
		return nil, false, fmt.Errorf("Resource %q of type %q already exists",
			id, rType)
	}

	xref := ""
	var xrefAny any
	hasXref := false

	if xrefAny, hasXref = obj["xref"]; hasXref {
		if IsNil(xrefAny) {
			// Do nothing
			// delete(obj, "xref")
		} else {
			xref, _ = xrefAny.(string)

			if objIsVer {
				return nil, false, fmt.Errorf("Can't create a Version with " +
					"the 'xref' attribute")
			}

			xref = strings.TrimSpace(xref)
			parts := strings.Split(xref, "/")
			if len(parts) != 4 {
				return nil, false, fmt.Errorf("'xref' must be of the form: " +
					"GROUPs/gID/RESOURCEs/rID")
			}
			/*
				group, err := g.Registry.FindGroup(parts[0], parts[1], false)
				if err != nil {
					return nil, false, err
				}
				if IsNil(group) {
					return nil, false, fmt.Errorf("Can't find group '%s/%s'",
						parts[0], parts[1])
				}
				res, err := group.FindResource(parts[2], parts[3], false)
				if err != nil {
					return nil, false, err
				}
				if IsNil(res) {
					return nil, false, fmt.Errorf("Can't find resource '%s/%s'",
						parts[2], parts[3])
				}
			*/

			// Erase all attributes except id and xref
			obj = map[string]any{
				"id":   id,
				"xref": xref,
			}
		}
	}

	// List of versions in the incoming request
	versions := map[string]any(nil)

	if !objIsVer {
		// If obj is for the resource then save and delete the versions
		// collection (and it's attributes) so we don't try to save them
		// as extensions on the Version
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
				UID:      id,

				Level:    2,
				Path:     g.Plural + "/" + g.UID + "/" + rType + "/" + id,
				Abstract: g.Plural + string(DB_IN) + rType,
			},
			Group: g,
		}

		err = DoOne(r.tx, `
        INSERT INTO Resources(SID, UID, GroupSID, ModelSID, Path, Abstract)
        SELECT ?,?,?,SID,?,?
        FROM ModelEntities
        WHERE RegistrySID=?
          AND ParentSID IN (
            SELECT SID FROM ModelEntities
            WHERE RegistrySID=?
            AND ParentSID IS NULL
            AND Plural=?)
            AND Plural=?`,
			r.DbSID, r.UID, g.DbSID,
			g.Plural+"/"+g.UID+"/"+rType+"/"+r.UID, g.Plural+string(DB_IN)+rType,
			g.Registry.DbSID,
			g.Registry.DbSID, g.Plural,
			rType)
		if err != nil {
			err = fmt.Errorf("Error adding Resource: %s", err)
			log.Print(err)
			return nil, false, err
		}

		// Use the ID passed as an arg, not from the metadata, as the true
		// ID. If the one in the metadata differs we'll flag it down below
		err = r.JustSet("id", r.UID)
		if err != nil {
			return nil, false, err
		}

		err = r.SetSave("#nextversionid", 1)
		if err != nil {
			return nil, false, err
		}
	}

	if hasXref {
		if IsNil(xref) {
			delete(obj, "xref")
			err = r.SetSave("xref", nil)
			if err != nil {
				return nil, false, err
			}
			hasXref = false
		} else {
			err = r.SetSave("xref", xref)
			if err != nil {
				return nil, false, err
			}
			return r, isNew, nil
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

	// Now process the defaultversionsticky and defaultversionid attributes.
	// Start with current sticky value
	sticky := (r.Get("defaultversionsticky") == true)

	// If there's an incoming obj and it changes 'sticky' then use it
	if !r.tx.IgnoreDefaultVersionSticky && !IsNil(obj) {
		stickyAny, ok := obj["defaultversionsticky"]
		if ok && (stickyAny != true && stickyAny != false && !IsNil(stickyAny)) {
			return nil, false, fmt.Errorf("'defaultversionsticky' must be " +
				"a boolean or null")
		}

		if addType == ADD_PATCH {
			if ok {
				sticky = (stickyAny == true)
			}
		} else {
			sticky = (stickyAny == true)
		}
	}

	// DefaultVersionID will only apply if sticky is set, otherwise
	// we'll just use the latest Version
	defVerID := r.GetAsString("defaultversionid")
	if sticky {
		if !r.tx.IgnoreDefaultVersionID && !IsNil(obj) {
			// TODO - should this take into account "patch" like above?
			defAny, ok := obj["defaultversionid"]
			if ok {
				if IsNil(defAny) {
					defVerID = "" // Use latest
				} else {
					v, err := r.FindVersion(defAny.(string), false)
					if err != nil {
						return nil, false, err
					}
					if IsNil(v) {
						return nil, false,
							fmt.Errorf("Can't find version %q", defAny)
					}
					defVerID = defAny.(string)
				}
			}
		}
	}

	if defVerID == "" {
		// Use latest
		vIDs, err := r.GetVersionIDs()
		Must(err)

		if len(vIDs) > 0 {
			defVerID = vIDs[0]
		}
	}

	// If the passed-in vID is empty then use the defaultVersion
	if vID == "" {
		vID = defVerID
	}

	// Update the appropriate Version (vID), but only if the versionID
	// doesn't match a Version ID from the "versions" collection (if there).
	// If both Resource attrs and Version attrs are present, use the Version's
	if vID != "" {
		if _, ok := versions[defVerID]; !ok {
			RemoveResourceAttributes(obj)
			_, _, err := r.UpsertVersionWithObject(vID, obj, addType)
			if err != nil {
				return nil, false, err
			}
		}
	} else {
		RemoveResourceAttributes(obj)
		_, _, err := r.UpsertVersionWithObject(vID, obj, addType)
		if err != nil {
			return nil, false, err
		}
	}

	// Make sure defaultversionid is set appropriately
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

	return r, isNew, err
}

// Return: *Resource, isNew, error
func (g *Group) oldUpsertResourceWithObject(rType string, id string, vID string, obj Object, addType AddType, doChildren bool, objIsVer bool) (*Resource, bool, error) {
	log.VPrintf(3, ">Enter: UpsertResourceWithObject(%s,%s)", rType, id)
	defer log.VPrintf(3, "<Exit: UpsertResourceWithObject")

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

	if r != nil && r.UID != id {
		return nil, false, fmt.Errorf("Attempting to create a Resource with "+
			"an \"id\" of %q, when one already exists as %q", id, r.UID)
	}

	if obj != nil && !IsNil(obj["id"]) && !objIsVer {
		if id != obj["id"] {
			return nil, false,
				fmt.Errorf(`The "id" attribute must be set `+
					`to %q, not %q`, id, obj["id"])
		}
	}

	if addType == ADD_ADD && r != nil {
		return nil, false, fmt.Errorf("Resource %q of type %q already exists",
			id, rType)
	}

	versions := map[string]any(nil)
	if !objIsVer {
		// If obj is for the resource then save and delete the versions
		// collection (and it's attributes) so we don't try to save them
		// as extensions on the Version
		var ok bool
		val, _ := obj["versions"]
		if !IsNil(val) {
			versions, ok = val.(map[string]any)
			if !ok {
				return nil, false,
					fmt.Errorf("Attribute %q doesn't appear to be of a "+
						"map of %q", "versions", "versions")
			}
			delete(obj, "versions")
			delete(obj, "versionscount")
			delete(obj, "versionsurl")
		}
	}

	isNew := (r == nil)
	if r == nil {
		r = &Resource{
			Entity: Entity{
				tx: g.tx,

				Registry: g.Registry,
				DbSID:    NewUUID(),
				Plural:   rType,
				UID:      id,

				Level:    2,
				Path:     g.Plural + "/" + g.UID + "/" + rType + "/" + id,
				Abstract: g.Plural + string(DB_IN) + rType,
			},
			Group: g,
		}

		err = DoOne(r.tx, `
        INSERT INTO Resources(SID, UID, GroupSID, ModelSID, Path, Abstract)
        SELECT ?,?,?,SID,?,?
        FROM ModelEntities
        WHERE RegistrySID=?
          AND ParentSID IN (
            SELECT SID FROM ModelEntities
            WHERE RegistrySID=?
            AND ParentSID IS NULL
            AND Plural=?)
            AND Plural=?`,
			r.DbSID, r.UID, g.DbSID,
			g.Plural+"/"+g.UID+"/"+rType+"/"+r.UID, g.Plural+string(DB_IN)+rType,
			g.Registry.DbSID,
			g.Registry.DbSID, g.Plural,
			rType)
		if err != nil {
			err = fmt.Errorf("Error adding Resource: %s", err)
			log.Print(err)
			return nil, false, err
		}

		// Use the ID passed as an arg, not from the metadata, as the true
		// ID. If the one in the metadata differs we'll flag it down below
		err = r.JustSet("id", r.UID)
		if err != nil {
			return nil, false, err
		}

		err = r.SetSave("#nextversionid", 1)
		if err != nil {
			return nil, false, err
		}

		if !objIsVer {
			delete(obj, "id") // Clear any ID there since it's the Resource's
		}

		// We only process the Resource level attributes if we're not
		// processing children, or there is no versions(children), or
		// the default version isn't part of the versions collections
		def := ""
		defAny := obj["defaultversionid"]
		if !IsNil(defAny) {
			def = defAny.(string)
		}
		if !doChildren || IsNil(versions) || (def != "" && versions[def] == nil) {
			// _, err = r.AddVersionWithObject(vID, obj)
			_, _, err = r.UpsertVersionWithObject(vID, obj, ADD_UPSERT)

			if err != nil {
				return nil, false, err
			}
		}
	} else {
		v, err := r.GetDefault()
		if err != nil {
			return nil, false, err
		}

		// We only process the Resource level attributes if we're not
		// processing children, or there is no versions(children), or
		// the default version isn't part of the versions collections
		if !doChildren || IsNil(versions) || versions[v.UID] == nil {
			if obj != nil {
				// If there's a doc but no "contenttype" value then:
				// - if existing entity doesn't have one, set it
				// - if existing entity does have one then only override it
				//   if we're not doing PATCH (PUT/POST are compelte overrides)
				if eval, ok := obj["#-contenttype"]; ok && !IsNil(eval) {
					if _, ok = obj["contenttype"]; !ok {
						val := v.Get("contenttype")
						if IsNil(val) || addType != ADD_PATCH {
							obj["contenttype"] = eval
						}
					}
				}

				v.NewObject = obj
				// if !objIsVer {
				v.NewObject["id"] = v.UID // ID is Resource's, switch to Version's
				// }

				if addType == ADD_PATCH {
					// Copy existing props if the incoming obj doesn't set them
					for k, val := range v.Object {
						if _, ok := v.NewObject[k]; !ok {
							v.NewObject[k] = val
						}
					}
				}

				err = v.ValidateAndSave()
				if err != nil {
					return nil, false, err
				}
			}
		}
	}

	if !objIsVer && doChildren && !IsNil(versions) {
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

	return r, isNew, err
}

func (g *Group) Delete() error {
	log.VPrintf(3, ">Enter: Group.Delete(%s)", g.UID)
	defer log.VPrintf(3, "<Exit: Group.Delete")

	return DoOne(g.tx, `DELETE FROM "Groups" WHERE SID=?`, g.DbSID)
}
