package tenant

import "github.com/gin-gonic/gin"

const tenantIDKey = "tenant_id"

func SetTenantID(c *gin.Context, tenantID string) {
	c.Set(tenantIDKey, tenantID)
}

func GetTenantID(c *gin.Context) string {
	v, _ := c.Get(tenantIDKey)
	id, _ := v.(string)
	return id
}
