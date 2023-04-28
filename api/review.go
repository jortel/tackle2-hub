package api

import (
	"github.com/konveyor/tackle2-hub/model"
)

//
// Routes
const (
	ReviewsRoot = "/reviews"
	ReviewRoot  = ReviewsRoot + "/:" + ID
	CopyRoot    = ReviewsRoot + "/copy"
)

//
// Review REST resource.
type Review struct {
	Resource
	BusinessCriticality uint   `json:"businessCriticality"`
	EffortEstimate      string `json:"effortEstimate"`
	ProposedAction      string `json:"proposedAction"`
	WorkPriority        uint   `json:"workPriority"`
	Comments            string `json:"comments"`
	Application         Ref    `json:"application" binding:"required"`
}

// With updates the resource with the model.
func (r *Review) With(m *model.Review) {
	r.Resource.With(&m.Model)
	r.BusinessCriticality = m.BusinessCriticality
	r.EffortEstimate = m.EffortEstimate
	r.ProposedAction = m.ProposedAction
	r.WorkPriority = m.WorkPriority
	r.Comments = m.Comments
	r.Application = r.ref(m.ApplicationID, m.Application)
}

//
// Model builds a model.
func (r *Review) Model() (m *model.Review) {
	m = &model.Review{
		BusinessCriticality: r.BusinessCriticality,
		EffortEstimate:      r.EffortEstimate,
		ProposedAction:      r.ProposedAction,
		WorkPriority:        r.WorkPriority,
		Comments:            r.Comments,
		ApplicationID:       r.Application.ID,
	}
	m.ID = r.ID
	return
}

//
// CopyRequest REST resource.
type CopyRequest struct {
	SourceReview       uint   `json:"sourceReview" binding:"required"`
	TargetApplications []uint `json:"targetApplications" binding:"required"`
}
