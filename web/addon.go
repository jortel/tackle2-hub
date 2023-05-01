package api

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/konveyor/tackle2-hub/api"
	crd "github.com/konveyor/tackle2-hub/k8s/api/tackle/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"net/http"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

//
// AddonHandler handles addon routes.
type AddonHandler struct {
	BaseHandler
}

//
// AddRoutes adds routes.
func (h AddonHandler) AddRoutes(e *gin.Engine) {
	routeGroup := e.Group("/")
	routeGroup.Use(Required("addons"))
	routeGroup.GET(api.AddonsRoot, h.List)
	routeGroup.GET(api.AddonsRoot+"/", h.List)
	routeGroup.GET(api.AddonRoot, h.Get)
}

// Get godoc
// @summary Get an addon by name.
// @description Get an addon by name.
// @tags addons
// @produce json
// @success 200 {object} api.Addon
// @router /addons/{name} [get]
// @param name path string true "Addon name"
func (h AddonHandler) Get(ctx *gin.Context) {
	name := ctx.Param(Name)
	addon := &crd.Addon{}
	err := h.Client.Get(
		context.TODO(),
		k8s.ObjectKey{
			Namespace: Settings.Hub.Namespace,
			Name:      name,
		},
		addon)
	if err != nil {
		if errors.IsNotFound(err) {
			ctx.Status(http.StatusNotFound)
			return
		} else {
			_ = ctx.Error(err)
			return
		}
	}
	r := Addon{}
	r.With(addon)

	h.Render(ctx, http.StatusOK, r)
}

// List godoc
// @summary List all addons.
// @description List all addons.
// @tags addons
// @produce json
// @success 200 {object} []api.Addon
// @router /addons [get]
func (h AddonHandler) List(ctx *gin.Context) {
	list := &crd.AddonList{}
	err := h.Client.List(
		context.TODO(),
		list,
		&k8s.ListOptions{
			Namespace: Settings.Namespace,
		})
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	content := []api.Addon{}
	for _, m := range list.Items {
		addon := api.Addon{}
		addon.With(&m)
		content = append(content, addon)
	}

	h.Render(ctx, http.StatusOK, content)
}

//
// REST Resources
type Addon = api.Addon
