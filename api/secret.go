package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/konveyor/tackle2-hub/model"
)

// Routes
const (
	SecretsRoot = "/secrets"
	SecretRoot  = "/secrets" + "/:" + ID
)

// SecretHandler handles secret routes.
type SecretHandler struct {
	BaseHandler
}

// AddRoutes adds routes.
func (h SecretHandler) AddRoutes(e *gin.Engine) {
	routeGroup := e.Group("/")
	routeGroup.Use(Required("secrets"))
	routeGroup.GET(SecretsRoot, h.List)
	routeGroup.GET(SecretsRoot+"/", h.List)
	routeGroup.POST(SecretsRoot, h.Create)
	routeGroup.GET(SecretRoot, h.Get)
	routeGroup.DELETE(SecretRoot, h.Delete)
}

// Get godoc
// @summary Get a secret by ID.
// @description Get a secret by ID.
// @tags secrets
// @produce json
// @success 200 {object} api.Secret
// @router /secrets/{id} [get]
// @param id path int true "Secret ID"
func (h SecretHandler) Get(ctx *gin.Context) {
	id := h.pk(ctx)
	m := &model.Secret{}
	db := h.DB(ctx)
	err := db.First(m, id).Error
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	resource := Secret{}
	resource.With(m)
	h.Respond(ctx, http.StatusOK, resource)
}

// List godoc
// @summary List all secrets.
// @description List all secrets.
// @tags secrets
// @produce json
// @success 200 {object} []api.Secret
// @router /secrets [get]
func (h SecretHandler) List(ctx *gin.Context) {
	var list []model.Secret
	db := h.DB(ctx)
	err := db.Find(&list).Error
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	resources := []Secret{}
	for i := range list {
		r := Secret{}
		r.With(&list[i])
		resources = append(resources, r)
	}

	h.Respond(ctx, http.StatusOK, resources)
}

// Create godoc
// @summary Create a secret.
// @description Create a secret.
// @tags secrets
// @accept json
// @produce json
// @success 201 {object} api.Secret
// @router /secrets [post]
// @param secret body api.Secret true "Secret data"
func (h SecretHandler) Create(ctx *gin.Context) {
	r := &Secret{}
	err := h.Bind(ctx, r)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	m := r.Model()
	m.CreateUser = h.BaseHandler.CurrentUser(ctx)
	err = h.DB(ctx).Create(m).Error
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	r.With(m)

	h.Respond(ctx, http.StatusCreated, r)
}

// Delete godoc
// @summary Delete a secret.
// @description Delete a secret.
// @tags secrets
// @success 204
// @router /secrets/{id} [delete]
// @param id path int true "Secret id"
func (h SecretHandler) Delete(ctx *gin.Context) {
	id := h.pk(ctx)
	m := &model.Secret{}
	err := h.DB(ctx).First(m, id).Error
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	err = h.DB(ctx).Delete(m).Error
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	h.Status(ctx, http.StatusNoContent)
}

// Secret API Resource
type Secret struct {
	Resource `yaml:",inline"`
	Content  Map
}

// With updates the resource with the model.
func (r *Secret) With(m *model.Secret) {
	r.Resource.With(&m.Model)
	r.Content = m.Content
}

// Model builds a model.
func (r *Secret) Model() (m *model.Secret) {
	m = &model.Secret{Content: m.Content}
	return
}
