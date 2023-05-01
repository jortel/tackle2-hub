package api

import (
	"github.com/gin-gonic/gin"
	"github.com/konveyor/tackle2-hub/api"
	"github.com/konveyor/tackle2-hub/auth"
	"net/http"
)

//
// AuthHandler handles auth routes.
type AuthHandler struct {
	BaseHandler
}

//
// AddRoutes adds routes.
func (h AuthHandler) AddRoutes(e *gin.Engine) {
	e.POST(api.AuthLoginRoot, h.Login)
}

// Login godoc
// @summary Login and obtain a bearer token.
// @description Login and obtain a bearer token.
// @tags auth
// @produce json
// @success 201 {object} api.Login
// @router /auth/login [post]
func (h AuthHandler) Login(ctx *gin.Context) {
	r := &Login{}
	err := h.Bind(ctx, r)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	r.Token, err = auth.Remote.Login(r.User, r.Password)
	if err != nil {
		h.Render(ctx,
			http.StatusUnauthorized,
			gin.H{
				"error": err.Error(),
			})
		return
	}
	r.Password = "" // Clear out password from response
	h.Render(ctx, http.StatusCreated, r)
}

//
// Required enforces that the user (identified by a token) has
// been granted the necessary scope to access a resource.
func Required(scope string) func(*gin.Context) {
	return func(ctx *gin.Context) {
		rtx := WithContext(ctx)
		token := ctx.GetHeader(Authorization)
		request := &auth.Request{
			Token:  token,
			Scope:  scope,
			Method: ctx.Request.Method,
			DB:     rtx.DB,
		}
		result, err := request.Permit()
		if err != nil {
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if !result.Authenticated {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if !result.Authorized {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		rtx.User = result.User
		rtx.Scopes = result.Scopes
	}
}

//
// REST Resources
type Login = api.Login
