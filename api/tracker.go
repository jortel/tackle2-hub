package api

import (
	"encoding/json"
	"github.com/konveyor/tackle2-hub/model"
	"time"
)

// Params
const (
	Connected = "connected"
)

// Routes
const (
	TrackersRoot = "/trackers"
	TrackerRoot  = "/trackers" + "/:" + ID
)

// Tracker API Resource
type Tracker struct {
	Resource
	Name        string    `json:"name" binding:"required"`
	URL         string    `json:"url" binding:"required"`
	Kind        string    `json:"kind" binding:"required,oneof=jira-cloud jira-server jira-datacenter"`
	Message     string    `json:"message"`
	Connected   bool      `json:"connected"`
	LastUpdated time.Time `json:"lastUpdated"`
	Metadata    Metadata  `json:"metadata"`
	Identity    Ref       `json:"identity" binding:"required"`
	Insecure    bool      `json:"insecure"`
}

// With updates the resource with the model.
func (r *Tracker) With(m *model.Tracker) {
	r.Resource.With(&m.Model)
	r.Name = m.Name
	r.URL = m.URL
	r.Kind = m.Kind
	r.Message = m.Message
	r.Connected = m.Connected
	r.LastUpdated = m.LastUpdated
	r.Insecure = m.Insecure
	r.Identity = r.ref(m.IdentityID, m.Identity)
	_ = json.Unmarshal(m.Metadata, &r.Metadata)
}

// Model builds a model.
func (r *Tracker) Model() (m *model.Tracker) {
	m = &model.Tracker{
		Name:       r.Name,
		URL:        r.URL,
		Kind:       r.Kind,
		Insecure:   r.Insecure,
		IdentityID: r.Identity.ID,
	}

	m.ID = r.ID

	return
}

type Metadata map[string]interface{}
