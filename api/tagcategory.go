package api

import (
	"github.com/konveyor/tackle2-hub/model"
)

//
// Routes
const (
	TagCategoriesRoot   = "/tagcategories"
	TagCategoryRoot     = TagCategoriesRoot + "/:" + ID
	TagCategoryTagsRoot = TagCategoryRoot + "/tags"
)

//
// TagCategory REST resource.
type TagCategory struct {
	Resource
	Name     string `json:"name" binding:"required"`
	Username string `json:"username"`
	Rank     uint   `json:"rank"`
	Color    string `json:"colour"`
	Tags     []Ref  `json:"tags"`
}

//
// With updates the resource with the model.
func (r *TagCategory) With(m *model.TagCategory) {
	r.Resource.With(&m.Model)
	r.ID = m.ID
	r.Name = m.Name
	r.Username = m.Username
	r.Rank = m.Rank
	r.Color = m.Color
	for _, tag := range m.Tags {
		ref := Ref{}
		ref.With(tag.ID, tag.Name)
		r.Tags = append(r.Tags, ref)
	}
}

//
// Model builds a model.
func (r *TagCategory) Model() (m *model.TagCategory) {
	m = &model.TagCategory{
		Name:     r.Name,
		Username: r.Username,
		Rank:     r.Rank,
		Color:    r.Color,
	}
	m.ID = r.ID
	return
}
