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

func (g *Group) FindResource(rType string, id string) (*Resource, error) {
	log.VPrintf(3, ">Enter: FindResource(%s,%s)", rType, id)
	defer log.VPrintf(3, "<Exit: FindResource")

	ent, err := RawEntityFromPath(g.tx, g.Registry.DbSID,
		g.Plural+"/"+g.UID+"/"+rType+"/"+id)
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
	return g.AddResourceWithObject(rType, id, vID, nil)
}

func (g *Group) AddResourceWithObject(rType string, id string, vID string, obj Object) (*Resource, error) {
	log.VPrintf(3, ">Enter: AddResourceWithObject(%s,%s)", rType, id)
	defer log.VPrintf(3, "<Exit: AddResourceWithObject")

	rModel := g.Registry.Model.Groups[g.Plural].Resources[rType]
	if rModel == nil {
		return nil, fmt.Errorf("Unknown Resource type (%s) for Group %q",
			rType, g.Plural)
	}

	r, err := g.FindResource(rType, id)
	if err != nil {
		return nil, fmt.Errorf("Error checking for Resource(%s) %q: %s",
			rType, id, err)
	}
	if r != nil {
		return nil, fmt.Errorf("Resource %q of type %q already exists",
			id, rType)
	}

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
		return nil, err
	}

	err = r.JustSet("id", r.UID)
	if err != nil {
		return nil, err
	}

	// TODO See if this should be JustSet and then call Save() after AddVer?
	err = r.SetSave("#nextversionid", 1)
	if err != nil {
		return nil, err
	}

	_, err = r.AddVersionWithObject(vID, obj)
	if err != nil {
		return nil, err
	}

	log.VPrintf(3, "Created new one - dbSID: %s", r.DbSID)
	return r, err
}

func (g *Group) UpsertResource(rType string, id string, vID string) (*Resource, bool, error) {
	return g.UpsertResourceWithObject(rType, id, vID, nil)
}

func (g *Group) UpsertResourceWithObject(rType string, id string, vID string, obj Object) (*Resource, bool, error) {
	log.VPrintf(3, ">Enter: UpsertResourceWithObject(%s,%s)", rType, id)
	defer log.VPrintf(3, "<Exit: UpsertResourceWithObject")

	rModel := g.Registry.Model.Groups[g.Plural].Resources[rType]
	if rModel == nil {
		return nil, false, fmt.Errorf("Unknown Resource type (%s) for Group %q",
			rType, g.Plural)
	}

	r, err := g.FindResource(rType, id)
	if err != nil {
		return nil, false, fmt.Errorf("Error checking for Resource(%s) %q: %s",
			rType, id, err)
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
			v.NewObject = obj
			v.NewObject["id"] = v.UID // ID is Resource's switch to Version's

			err = v.ValidateAndSave()
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
