package api

import (
	"context"
	"net/http"
	"net/http/httputil"
	"strconv"

	"github.com/gin-gonic/gin"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

// Routes
const (
	ServiceRoot = "/service/:name/*" + Wildcard
)

// ServiceHandler handles service routes.
type ServiceHandler struct {
	BaseHandler
}

// AddRoutes adds routes.
func (h ServiceHandler) AddRoutes(e *gin.Engine) {
	e.Any(ServiceRoot, h.Forward)
}

// Forward provides RBAC and forwards request to the service.
func (h ServiceHandler) Forward(ctx *gin.Context) {
	name := ctx.Param(Name)
	path := ctx.Param(Wildcard)
	Required("service." + name)(ctx)
	if len(ctx.Errors) > 0 {
		return
	}
	service := &core.Service{}
	err := h.Client(ctx).Get(
		context.TODO(),
		k8s.ObjectKey{
			Namespace: Settings.Hub.Namespace,
			Name:      name,
		},
		service)
	if err != nil {
		if errors.IsNotFound(err) {
			h.Status(ctx, http.StatusNotFound)
			return
		} else {
			_ = ctx.Error(err)
			return
		}
	}
	host := service.Spec.ClusterIP
	port := int(service.Spec.Ports[0].Port)
	host = host + ":" + strconv.Itoa(port)
	proxy := httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = h.scheme(service)
			req.URL.Host = h.host(service)
			req.URL.Path = path
		},
	}

	proxy.ServeHTTP(ctx.Writer, ctx.Request)
}

func (h *ServiceHandler) scheme(service *core.Service) (scheme string) {
	scheme = "http"
	s, found := service.Annotations["konveyor.io/scheme"]
	if found {
		scheme = s
	}
	return
}

func (h *ServiceHandler) host(service *core.Service) (host string) {
	host = service.Spec.ClusterIP
	for _, p := range service.Spec.Ports {
		port := int(p.Port)
		host += ":"
		host += strconv.Itoa(port)
		break
	}
	return
}
