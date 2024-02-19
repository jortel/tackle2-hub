package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	crd "github.com/konveyor/tackle2-hub/k8s/api/tackle/v1alpha1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

// Routes
const (
	ProvidersRoot = "/providers"
	ProviderRoot  = ProvidersRoot + "/:" + Name
)

// ProviderHandler handles provider routes.
type ProviderHandler struct {
	BaseHandler
}

// AddRoutes adds routes.
func (h ProviderHandler) AddRoutes(e *gin.Engine) {
	routeGroup := e.Group("/")
	routeGroup.Use(Required("providers"))
	routeGroup.GET(ProvidersRoot, h.List)
	routeGroup.GET(ProvidersRoot+"/", h.List)
	routeGroup.GET(ProviderRoot, h.Get)
}

// Get godoc
// @summary Get an provider by name.
// @description Get an provider by name.
// @tags providers
// @produce json
// @success 200 {object} api.Provider
// @router /providers/{name} [get]
// @param name path string true "Provider name"
func (h ProviderHandler) Get(ctx *gin.Context) {
	name := ctx.Param(Name)
	provider := &crd.Provider{}
	err := h.Client(ctx).Get(
		context.TODO(),
		k8s.ObjectKey{
			Namespace: Settings.Hub.Namespace,
			Name:      name,
		},
		provider)
	if err != nil {
		if errors.IsNotFound(err) {
			h.Status(ctx, http.StatusNotFound)
			return
		} else {
			_ = ctx.Error(err)
			return
		}
	}
	r := Provider{}
	r.With(provider)

	h.Respond(ctx, http.StatusOK, r)
}

// List godoc
// @summary List all providers.
// @description List all providers.
// @tags providers
// @produce json
// @success 200 {object} []api.Provider
// @router /providers [get]
func (h ProviderHandler) List(ctx *gin.Context) {
	list := &crd.ProviderList{}
	err := h.Client(ctx).List(
		context.TODO(),
		list,
		&k8s.ListOptions{
			Namespace: Settings.Namespace,
		})
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	content := []Provider{}
	for _, m := range list.Items {
		provider := Provider{}
		provider.With(&m)
		content = append(content, provider)
	}

	h.Respond(ctx, http.StatusOK, content)
}

// Provider REST resource.
type Provider struct {
	Name      string         `json:"name"`
	Container core.Container `json:"container"`
}

// With model.
func (r *Provider) With(m *crd.Provider) {
	r.Name = m.Name
	r.Container = m.Spec.Container
}
