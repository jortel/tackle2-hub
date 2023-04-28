package api

import (
	crd "github.com/konveyor/tackle2-hub/k8s/api/tackle/v1alpha1"
)

//
// Addon REST resource.
type Addon struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

//
// With model.
func (r *Addon) With(m *crd.Addon) {
	r.Name = m.Name
	r.Image = m.Spec.Image
}
