package api

import (
	"github.com/konveyor/tackle2-hub/model"
)

//
// Routes
const (
	BusinessServicesRoot = "/businessservices"
	BusinessServiceRoot  = BusinessServicesRoot + "/:" + ID
)

//
// BusinessService REST resource.
type BusinessService struct {
	Resource
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Stakeholder *Ref   `json:"owner"`
}

//
// With updates the resource with the model.
func (r *BusinessService) With(m *model.BusinessService) {
	r.Resource.With(&m.Model)
	r.Name = m.Name
	r.Description = m.Description
	r.Stakeholder = r.refPtr(m.StakeholderID, m.Stakeholder)
}

//
// Model builds a model.
func (r *BusinessService) Model() (m *model.BusinessService) {
	m = &model.BusinessService{
		Name:        r.Name,
		Description: r.Description,
	}
	m.ID = r.ID
	if r.Stakeholder != nil {
		m.StakeholderID = &r.Stakeholder.ID
	}
	return
}
