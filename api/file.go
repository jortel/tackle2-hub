package api

import (
	"github.com/konveyor/tackle2-hub/model"
	"time"
)

//
// Routes
const (
	FilesRoot = "/files"
	FileRoot  = FilesRoot + "/:" + ID
)

//
// File REST resource.
type File struct {
	Resource
	Name       string     `json:"name"`
	Path       string     `json:"path"`
	Expiration *time.Time `json:"expiration,omitempty"`
}

//
// With updates the resource with the model.
func (r *File) With(m *model.File) {
	r.Resource.With(&m.Model)
	r.Name = m.Name
	r.Path = m.Path
	r.Expiration = m.Expiration
}
