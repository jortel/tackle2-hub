package api

import (
	"encoding/json"
	"github.com/konveyor/tackle2-hub/model"
)

//
// Params
const (
	Source = "source"
)

//
// Routes
const (
	ApplicationsRoot     = "/applications"
	ApplicationRoot      = ApplicationsRoot + "/:" + ID
	ApplicationTagsRoot  = ApplicationRoot + "/tags"
	ApplicationTagRoot   = ApplicationTagsRoot + "/:" + ID2
	ApplicationFactsRoot = ApplicationRoot + "/facts"
	ApplicationFactRoot  = ApplicationFactsRoot + "/:" + Key + "/*" + Source
	AppBucketRoot        = ApplicationRoot + "/bucket"
	AppBucketContentRoot = AppBucketRoot + "/*" + Wildcard
	AppStakeholdersRoot  = ApplicationRoot + "/stakeholders"
)

//
// Application REST resource.
type Application struct {
	Resource
	Name            string      `json:"name" binding:"required"`
	Description     string      `json:"description"`
	Bucket          *Ref        `json:"bucket"`
	Repository      *Repository `json:"repository"`
	Binary          string      `json:"binary"`
	Review          *Ref        `json:"review"`
	Comments        string      `json:"comments"`
	Identities      []Ref       `json:"identities"`
	Tags            []TagRef    `json:"tags"`
	BusinessService *Ref        `json:"businessService"`
	Owner           *Ref        `json:"owner"`
	Contributors    []Ref       `json:"contributors"`
	MigrationWave   *Ref        `json:"migrationWave"`
}

//
// With updates the resource using the model.
func (r *Application) With(m *model.Application, tags []model.ApplicationTag) {
	r.Resource.With(&m.Model)
	r.Name = m.Name
	r.Description = m.Description
	r.Bucket = r.refPtr(m.BucketID, m.Bucket)
	r.Comments = m.Comments
	r.Binary = m.Binary
	_ = json.Unmarshal(m.Repository, &r.Repository)
	if m.Review != nil {
		ref := &Ref{}
		ref.With(m.Review.ID, "")
		r.Review = ref
	}
	r.BusinessService = r.refPtr(m.BusinessServiceID, m.BusinessService)
	r.Identities = []Ref{}
	for _, id := range m.Identities {
		ref := Ref{}
		ref.With(id.ID, id.Name)
		r.Identities = append(
			r.Identities,
			ref)
	}
	for i := range tags {
		ref := TagRef{}
		ref.With(tags[i].TagID, tags[i].Tag.Name, tags[i].Source)
		r.Tags = append(r.Tags, ref)
	}
	r.Owner = r.refPtr(m.OwnerID, m.Owner)
	r.Contributors = []Ref{}
	for _, c := range m.Contributors {
		ref := Ref{}
		ref.With(c.ID, c.Name)
		r.Contributors = append(
			r.Contributors,
			ref)
	}
	r.MigrationWave = r.refPtr(m.MigrationWaveID, m.MigrationWave)
}

//
// Model builds a model.
func (r *Application) Model() (m *model.Application) {
	m = &model.Application{
		Name:        r.Name,
		Description: r.Description,
		Comments:    r.Comments,
		Binary:      r.Binary,
	}
	m.ID = r.ID
	if r.Repository != nil {
		m.Repository, _ = json.Marshal(r.Repository)
	}
	if r.BusinessService != nil {
		m.BusinessServiceID = &r.BusinessService.ID
	}
	for _, ref := range r.Identities {
		m.Identities = append(
			m.Identities,
			model.Identity{
				Model: model.Model{
					ID: ref.ID,
				},
			})
	}
	for _, ref := range r.Tags {
		m.Tags = append(
			m.Tags,
			model.Tag{
				Model: model.Model{
					ID: ref.ID,
				},
			})
	}
	if r.Owner != nil {
		m.OwnerID = &r.Owner.ID
	}
	for _, ref := range r.Contributors {
		m.Contributors = append(
			m.Contributors,
			model.Stakeholder{
				Model: model.Model{
					ID: ref.ID,
				},
			})
	}
	if r.MigrationWave != nil {
		m.MigrationWaveID = &r.MigrationWave.ID
	}

	return
}

//
// Repository REST nested resource.
type Repository struct {
	Kind   string `json:"kind"`
	URL    string `json:"url"`
	Branch string `json:"branch"`
	Tag    string `json:"tag"`
	Path   string `json:"path"`
}

//
// Fact REST nested resource.
type Fact struct {
	Key    string      `json:"key"`
	Value  interface{} `json:"value"`
	Source string      `json:"source"`
}

func (r *Fact) With(m *model.Fact) {
	r.Key = m.Key
	r.Source = m.Source
	_ = json.Unmarshal(m.Value, &r.Value)
}

func (r *Fact) Model() (m *model.Fact) {
	m = &model.Fact{}
	m.Key = r.Key
	m.Source = r.Source
	m.Value, _ = json.Marshal(r.Value)
	return
}

//
// Stakeholders REST subresource.
type Stakeholders struct {
	Owner        *Ref  `json:"owner"`
	Contributors []Ref `json:"contributors"`
}

func (r *Stakeholders) OwnerID() (ownerID *uint) {
	if r.Owner != nil {
		ownerID = &r.Owner.ID
	}
	return
}

func (r *Stakeholders) GetContributors() (contributors []model.Stakeholder) {
	for _, ref := range r.Contributors {
		contributors = append(
			contributors,
			model.Stakeholder{
				Model: model.Model{
					ID: ref.ID,
				},
			})
	}
	return
}
