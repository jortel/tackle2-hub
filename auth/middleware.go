package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/konveyor/tackle2-hub/settings"
	"net/http"
	"strings"
)

const (
	Header = "Authorization"
)

//
// AuthorizationRequired enforces that the user (identified by a token) has
// been granted the necessary scope to access a resource.
func AuthorizationRequired(p Provider, requiredScope string) func(*gin.Context) {
	return func(c *gin.Context) {
		token := c.GetHeader(Header)
		addonToken := settings.Settings.Auth.AddonToken
		if addonToken != "" && token == addonToken {
			c.Next()
			return
		}

		scopes, err := p.Scopes(token)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		for _, s := range scopes {
			if s.Allow(requiredScope, strings.ToLower(c.Request.Method)) {
				c.Next()
				return
			}
		}
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
}

//
// CurrentUser returns the current login user.
func CurrentUser(p Provider, token string) (user string, err error) {
	addonToken := settings.Settings.Auth.AddonToken
	if token == addonToken {
		return
	}
	user, err = p.User(token)
	return
}
