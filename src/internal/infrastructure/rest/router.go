package rest

import (
	"database/sql"
	"net/http"

	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/auth"
	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/featuregate"
	"github.com/gin-gonic/gin"
)

type router struct {
	Server *gin.Engine
}

func NewRouter(db *sql.DB, jwtSecret string) *router {
	r := gin.Default()

	r.GET("/api/inventories/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "module": "inventory"})
	})

	inventory := r.Group("/api/inventories")
	inventory.Use(auth.RequireCollaboratorAuth(jwtSecret))
	inventory.Use(featuregate.RequireInventoryFeature(db))
	inventory.Use(auth.RequireFeatureRead("inventory"))
	{
		accessHandler := newAccessHttpHandler()
		inventory.GET("/access", accessHandler.HandleCheck)

		productHandler := newProductHttpHandler(db)
		inventory.GET("/products", productHandler.HandleList)
		inventory.GET("/products/:id", productHandler.HandleFindByID)
		inventory.POST("/products", auth.RequireFeatureWrite("inventory"), productHandler.HandleCreate)
		inventory.PUT("/products/:id", auth.RequireFeatureWrite("inventory"), productHandler.HandleUpdate)
		inventory.DELETE("/products/:id", auth.RequireFeatureWrite("inventory"), productHandler.HandleDelete)
	}

	return &router{Server: r}
}

func buildResponseError(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{"message": err.Error()})
}

func buildResponseSuccess(c *gin.Context, status int, content any) {
	c.JSON(status, content)
}

func parseStringToInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return defaultVal
		}
		v = v*10 + int(ch-'0')
	}
	return v
}

func actorIDFromContext(c *gin.Context) string {
	return c.GetString("actor_id")
}

