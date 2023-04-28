package api

import (
	"github.com/konveyor/tackle2-hub/model"
)

//
// Routes
const (
	JobFunctionsRoot = "/jobfunctions"
	JobFunctionRoot  = JobFunctionsRoot + "/:" + ID
)

//
// JobFunction REST resource.
type JobFunction struct {
	Resource
	Name         string `json:"name" binding:"required"`
	Stakeholders []Ref  `json:"stakeholders"`
}

//
// With updates the resource with the model.
func (r *JobFunction) With(m *model.JobFunction) {
	r.Resource.With(&m.Model)
	r.Name = m.Name
	for _, s := range m.Stakeholders {
		ref := Ref{}
		ref.With(s.ID, s.Name)
		r.Stakeholders = append(r.Stakeholders, ref)
	}
}

//
// Model builds a model.
func (r *JobFunction) Model() (m *model.JobFunction) {
	m = &model.JobFunction{
		Name: r.Name,
	}
	m.ID = r.ID

	return
}
