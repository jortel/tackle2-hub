package api

import (
	"github.com/gin-gonic/gin"
	"github.com/konveyor/tackle2-hub/model"
	"net/http"
)

//
// Kind
const (
	TagKind = "tag"
)

//
// Routes
const (
	TagsRoot = ControlsRoot + "/tag"
	TagRoot  = TagsRoot + "/:" + ID
)

//
// TagHandler handles tag routes.
type TagHandler struct {
	BaseHandler
}

//
// AddRoutes adds routes.
func (h TagHandler) AddRoutes(e *gin.Engine) {
	e.GET(TagsRoot, h.List)
	e.GET(TagsRoot+"/", h.List)
	e.POST(TagsRoot, h.Create)
	e.GET(TagRoot, h.Get)
	e.PUT(TagRoot, h.Update)
	e.DELETE(TagRoot, h.Delete)
}

// Get godoc
// @summary Get a tag by ID.
// @description Get a tag by ID.
// @tags get
// @produce json
// @success 200 {object} api.Tag
// @router /controls/tag/{id} [get]
// @param id path string true "Tag ID"
func (h TagHandler) Get(ctx *gin.Context) {
	m := &model.Tag{}
	id := ctx.Param(ID)
	db := h.preLoad(h.DB, "TagType")
	result := db.First(m, id)
	if result.Error != nil {
		h.getFailed(ctx, result.Error)
		return
	}

	resource := Tag{}
	resource.With(m)
	ctx.JSON(http.StatusOK, resource)
}

// List godoc
// @summary List all tags.
// @description List all tags.
// @tags get
// @produce json
// @success 200 {object} []api.Tag
// @router /controls/tag [get]
func (h TagHandler) List(ctx *gin.Context) {
	var count int64
	var list []model.Tag
	h.DB.Model(model.Tag{}).Count(&count)
	pagination := NewPagination(ctx)
	db := pagination.apply(h.DB)
	db = h.preLoad(db, "TagType")
	result := db.Find(&list)
	if result.Error != nil {
		h.listFailed(ctx, result.Error)
		return
	}
	resources := []Tag{}
	for i := range list {
		r := Tag{}
		r.With(&list[i])
		resources = append(resources, r)
	}

	h.listResponse(ctx, TagKind, resources, int(count))
}

// Create godoc
// @summary Create a tag.
// @description Create a tag.
// @tags create
// @accept json
// @produce json
// @success 201 {object} api.Tag
// @router /controls/tag [post]
// @param tag body Tag true "Tag data"
func (h TagHandler) Create(ctx *gin.Context) {
	r := &Tag{}
	err := ctx.BindJSON(r)
	if err != nil {
		h.bindFailed(ctx, err)
		return
	}
	m := r.Model()
	result := h.DB.Create(m)
	if result.Error != nil {
		h.createFailed(ctx, result.Error)
		return
	}
	r.With(m)

	ctx.JSON(http.StatusCreated, r)
}

// Delete godoc
// @summary Delete a tag.
// @description Delete a tag.
// @tags delete
// @success 204
// @router /controls/tag/{id} [delete]
// @param id path string true "Tag ID"
func (h TagHandler) Delete(ctx *gin.Context) {
	id := ctx.Param(ID)
	result := h.DB.Delete(&model.Tag{}, id)
	if result.Error != nil {
		h.deleteFailed(ctx, result.Error)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// Update godoc
// @summary Update a tag.
// @description Update a tag.
// @tags update
// @accept json
// @success 204
// @router /controls/tag/{id} [put]
// @param id path string true "Tag ID"
// @param tag body api.Tag true "Tag data"
func (h TagHandler) Update(ctx *gin.Context) {
	id := ctx.Param(ID)
	r := &Tag{}
	err := ctx.BindJSON(r)
	if err != nil {
		h.bindFailed(ctx, err)
		return
	}
	m := r.Model()
	result := h.DB.Model(&model.Tag{}).Where("id = ?", id).Omit("id").Updates(m)
	if result.Error != nil {
		h.updateFailed(ctx, result.Error)
		return
	}

	ctx.Status(http.StatusNoContent)
}

//
// Tag REST resource.
type Tag struct {
	Resource
	Name    string `json:"name" binding:"required"`
	TagType struct {
		ID    uint   `json:"id" binding:"required"`
		Name  string `json:"name"`
		Color string `json:"colour"`
	} `json:"tagType" binding:"required"`
}

//
// With updates the resource with the model.
func (r *Tag) With(m *model.Tag) {
	r.Resource.With(&m.Model)
	r.Name = m.Name
	r.TagType.ID = m.TagTypeID
	r.TagType.Name = m.TagType.Name
	r.TagType.Color = m.TagType.Color
}

//
// Model builds a model.
func (r *Tag) Model() (m *model.Tag) {
	m = &model.Tag{
		Name:      r.Name,
		TagTypeID: r.TagType.ID,
	}
	m.ID = r.ID
	return
}
