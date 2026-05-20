package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func CanReadFeature(roles []string, feature string) bool {
	if feature == "" {
		return false
	}

	for _, role := range roles {
		if role == "read" || role == "write" || role == feature+".read" || role == feature+".write" {
			return true
		}
	}

	return false
}

func CanWriteFeature(roles []string, feature string) bool {
	if feature == "" {
		return false
	}

	for _, role := range roles {
		if role == "write" || role == feature+".write" {
			return true
		}
	}

	return false
}

func RequireFeatureRead(feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !CanReadFeature(RolesFromContext(c), feature) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			return
		}

		c.Next()
	}
}

func RequireFeatureWrite(feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !CanWriteFeature(RolesFromContext(c), feature) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			return
		}

		c.Next()
	}
}

func RolesFromContext(c *gin.Context) []string {
	raw, ok := c.Get("roles")
	if !ok {
		return make([]string, 0)
	}

	roles, ok := raw.([]string)
	if !ok {
		return make([]string, 0)
	}

	return roles
}
