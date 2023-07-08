package registry

import (
// "fmt"

// log "github.com/duglin/dlog"
)

type Version struct {
	Entity
	Resource *Resource
}

func (v *Version) Set(name string, val any) error {
	return SetProp(v, name, val)
}
