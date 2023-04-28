package api

import (
	"encoding/json"
	"github.com/konveyor/tackle2-hub/model"
	"time"
)

// Params.
const (
	TrackerId = "tracker"
)

// Routes
const (
	TicketsRoot = "/tickets"
	TicketRoot  = "/tickets" + "/:" + ID
)

// Ticket API Resource
type Ticket struct {
	Resource
	Kind        string    `json:"kind" binding:"required"`
	Reference   string    `json:"reference"`
	Link        string    `json:"link"`
	Parent      string    `json:"parent" binding:"required"`
	Error       bool      `json:"error"`
	Message     string    `json:"message"`
	Status      string    `json:"status"`
	LastUpdated time.Time `json:"lastUpdated"`
	Fields      Fields    `json:"fields"`
	Application Ref       `json:"application" binding:"required"`
	Tracker     Ref       `json:"tracker" binding:"required"`
}

// With updates the resource with the model.
func (r *Ticket) With(m *model.Ticket) {
	r.Resource.With(&m.Model)
	r.Kind = m.Kind
	r.Reference = m.Reference
	r.Parent = m.Parent
	r.Link = m.Link
	r.Error = m.Error
	r.Message = m.Message
	r.Status = m.Status
	r.LastUpdated = m.LastUpdated
	r.Application = r.ref(m.ApplicationID, m.Application)
	r.Tracker = r.ref(m.TrackerID, m.Tracker)
	_ = json.Unmarshal(m.Fields, &r.Fields)
}

// Model builds a model.
func (r *Ticket) Model() (m *model.Ticket) {
	m = &model.Ticket{
		Kind:          r.Kind,
		Parent:        r.Parent,
		ApplicationID: r.Application.ID,
		TrackerID:     r.Tracker.ID,
	}
	if r.Fields == nil {
		r.Fields = Fields{}
	}
	m.Fields, _ = json.Marshal(r.Fields)
	m.ID = r.ID

	return
}

type Fields map[string]interface{}
