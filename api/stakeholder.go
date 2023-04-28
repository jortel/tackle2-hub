package api

import (
	"github.com/konveyor/tackle2-hub/model"
)

//
// Routes
const (
	StakeholdersRoot = "/stakeholders"
	StakeholderRoot  = StakeholdersRoot + "/:" + ID
)

//
// Stakeholder REST resource.
type Stakeholder struct {
	Resource
	Name             string `json:"name" binding:"required"`
	Email            string `json:"email" binding:"required"`
	Groups           []Ref  `json:"stakeholderGroups"`
	BusinessServices []Ref  `json:"businessServices"`
	JobFunction      *Ref   `json:"jobFunction"`
	Owns             []Ref  `json:"owns"`
	Contributes      []Ref  `json:"contributes"`
	MigrationWaves   []Ref  `json:"migrationWaves"`
}

//
// With updates the resource with the model.
func (r *Stakeholder) With(m *model.Stakeholder) {
	r.Resource.With(&m.Model)
	r.Name = m.Name
	r.Email = m.Email
	r.JobFunction = r.refPtr(m.JobFunctionID, m.JobFunction)
	r.Groups = []Ref{}
	for _, g := range m.Groups {
		ref := Ref{}
		ref.With(g.ID, g.Name)
		r.Groups = append(r.Groups, ref)
	}
	r.BusinessServices = []Ref{}
	for _, b := range m.BusinessServices {
		ref := Ref{}
		ref.With(b.ID, b.Name)
		r.BusinessServices = append(r.BusinessServices, ref)
	}
	r.Owns = []Ref{}
	for _, o := range m.Owns {
		ref := Ref{}
		ref.With(o.ID, o.Name)
		r.Owns = append(r.Owns, ref)
	}
	r.Contributes = []Ref{}
	for _, c := range m.Contributes {
		ref := Ref{}
		ref.With(c.ID, c.Name)
		r.Contributes = append(r.Contributes, ref)
	}
	r.MigrationWaves = []Ref{}
	for _, w := range m.MigrationWaves {
		ref := Ref{}
		ref.With(w.ID, w.Name)
		r.MigrationWaves = append(r.MigrationWaves, ref)
	}
}

//
// Model builds a model.
func (r *Stakeholder) Model() (m *model.Stakeholder) {
	m = &model.Stakeholder{
		Name:  r.Name,
		Email: r.Email,
	}
	m.ID = r.ID
	if r.JobFunction != nil {
		m.JobFunctionID = &r.JobFunction.ID
	}
	for _, g := range r.Groups {
		m.Groups = append(
			m.Groups,
			model.StakeholderGroup{
				Model: model.Model{ID: g.ID},
			})
	}
	for _, b := range r.BusinessServices {
		m.BusinessServices = append(
			m.BusinessServices, model.BusinessService{
				Model: model.Model{ID: b.ID},
			})
	}
	for _, o := range r.Owns {
		m.Owns = append(
			m.Owns,
			model.Application{
				Model: model.Model{ID: o.ID},
			})
	}
	for _, c := range r.Contributes {
		m.Contributes = append(
			m.Contributes,
			model.Application{
				Model: model.Model{ID: c.ID},
			})
	}
	for _, w := range r.MigrationWaves {
		m.MigrationWaves = append(
			m.MigrationWaves,
			model.MigrationWave{
				Model: model.Model{ID: w.ID},
			})
	}
	return
}
