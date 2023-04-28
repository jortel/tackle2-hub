package api

import (
	"encoding/json"
	"github.com/konveyor/tackle2-hub/model"
)

//
// Routes
const (
	ProxiesRoot = "/proxies"
	ProxyRoot   = ProxiesRoot + "/:" + ID
)

const (
	Kind = "kind"
)

//
// Proxy REST resource.
type Proxy struct {
	Resource
	Enabled  bool     `json:"enabled"`
	Kind     string   `json:"kind" binding:"oneof=http https"`
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Excluded []string `json:"excluded"`
	Identity *Ref     `json:"identity"`
}

//
// With updates the resource with the model.
func (r *Proxy) With(m *model.Proxy) {
	r.Resource.With(&m.Model)
	r.Enabled = m.Enabled
	r.Kind = m.Kind
	r.Host = m.Host
	r.Port = m.Port
	r.Identity = r.refPtr(m.IdentityID, m.Identity)
	_ = json.Unmarshal(m.Excluded, &r.Excluded)
	if r.Excluded == nil {
		r.Excluded = []string{}
	}
}

//
// Model builds a model.
func (r *Proxy) Model() (m *model.Proxy) {
	m = &model.Proxy{
		Enabled: r.Enabled,
		Kind:    r.Kind,
		Host:    r.Host,
		Port:    r.Port,
	}
	m.ID = r.ID
	m.IdentityID = r.idPtr(r.Identity)
	if r.Excluded != nil {
		m.Excluded, _ = json.Marshal(r.Excluded)
	}

	return
}
