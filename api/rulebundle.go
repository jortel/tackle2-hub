package api

import (
	"encoding/json"
	"github.com/konveyor/tackle2-hub/model"
)

//
// Routes
const (
	RuleBundlesRoot = "/rulebundles"
	RuleBundleRoot  = RuleBundlesRoot + "/:" + ID
)

//
// RuleBundle REST resource.
type RuleBundle struct {
	Resource
	Kind        string      `json:"kind,omitempty"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Image       Ref         `json:"image"`
	RuleSets    []RuleSet   `json:"rulesets"`
	Custom      bool        `json:"custom,omitempty"`
	Repository  *Repository `json:"repository,omitempty"`
	Identity    *Ref        `json:"identity,omitempty"`
}

//
// With updates the resource with the model.
func (r *RuleBundle) With(m *model.RuleBundle) {
	r.Resource.With(&m.Model)
	r.Kind = m.Kind
	r.Name = m.Name
	r.Description = m.Description
	r.Custom = m.Custom
	r.Identity = r.refPtr(m.IdentityID, m.Identity)
	imgRef := Ref{ID: m.ImageID}
	if m.Image != nil {
		imgRef.Name = m.Image.Name
	}
	r.Image = imgRef
	_ = json.Unmarshal(m.Repository, &r.Repository)
	r.RuleSets = []RuleSet{}
	for i := range m.RuleSets {
		rule := RuleSet{}
		rule.With(&m.RuleSets[i])
		r.RuleSets = append(
			r.RuleSets,
			rule)
	}
}

//
// Model builds a model.
func (r *RuleBundle) Model() (m *model.RuleBundle) {
	m = &model.RuleBundle{
		Kind:        r.Kind,
		Name:        r.Name,
		Description: r.Description,
		Custom:      r.Custom,
	}
	m.ID = r.ID
	m.ImageID = r.Image.ID
	m.IdentityID = r.idPtr(r.Identity)
	m.RuleSets = []model.RuleSet{}
	for _, rule := range r.RuleSets {
		m.RuleSets = append(m.RuleSets, *rule.Model())
	}
	if r.Repository != nil {
		m.Repository, _ = json.Marshal(r.Repository)
	}
	return
}

//
// HasRuleSet - determine if the ruleset is referenced.
func (r *RuleBundle) HasRuleSet(id uint) (b bool) {
	for _, ruleset := range r.RuleSets {
		if id == ruleset.ID {
			b = true
			break
		}
	}
	return
}

//
// RuleSet - REST Resource.
type RuleSet struct {
	Resource
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	Metadata    interface{} `json:"metadata,omitempty"`
	File        *Ref        `json:"file,omitempty"`
}

//
// With updates the resource with the model.
func (r *RuleSet) With(m *model.RuleSet) {
	r.Resource.With(&m.Model)
	r.Name = m.Name
	_ = json.Unmarshal(m.Metadata, &r.Metadata)
	r.File = r.refPtr(m.FileID, m.File)
}

//
// Model builds a model.
func (r *RuleSet) Model() (m *model.RuleSet) {
	m = &model.RuleSet{}
	m.ID = r.ID
	m.Name = r.Name
	if r.Metadata != nil {
		m.Metadata, _ = json.Marshal(r.Metadata)
	}
	m.FileID = r.idPtr(r.File)
	return
}
