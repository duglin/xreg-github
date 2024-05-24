package registry

import (
	"fmt"

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
		_, err = r.AddVersionWithObject(vID, obj)
		if err != nil {
			return nil, false, err
		}
	} else {
		v, err := r.GetDefault()
		if err != nil {
			return nil, false, err
		}

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
