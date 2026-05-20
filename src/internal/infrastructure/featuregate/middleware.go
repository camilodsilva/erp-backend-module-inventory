package featuregate

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	inventoryFeatureSlug                 = "inventory"
	inventoryFeatureDisabledMessage      = "inventory module not enabled for this company"
	inventoryFeatureInternalErrorMessage = "internal server error"
	hasCompanyFeatureEnabledQuery        = `
		select exists (
			select 1
			from public.company_features cf
			join public.features f on f.id = cf.feature_id
			where cf.company_id = $1
			  and f.title = $2
			limit 1
		)
	`
)

type (
	featureGateRepository interface {
		HasFeature(companyID, featureSlug string) (bool, error)
	}

	postgresFeatureGateRepository struct {
		db *sql.DB
	}
)

func newPostgresFeatureGateRepository(db *sql.DB) *postgresFeatureGateRepository {
	return &postgresFeatureGateRepository{db: db}
}

func (r *postgresFeatureGateRepository) HasFeature(companyID, featureSlug string) (bool, error) {
	if r.db == nil {
		return false, errors.New("database connection not configured")
	}

	var enabled bool
	err := r.db.QueryRow(hasCompanyFeatureEnabledQuery, companyID, featureSlug).Scan(&enabled)
	if err != nil {
		return false, err
	}

	return enabled, nil
}

func RequireInventoryFeature(db *sql.DB) gin.HandlerFunc {
	return requireInventoryFeature(newPostgresFeatureGateRepository(db))
}

func requireInventoryFeature(repository featureGateRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		companyID := c.GetString("company_id")
		if companyID == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": inventoryFeatureDisabledMessage})
			return
		}

		enabled, err := repository.HasFeature(companyID, inventoryFeatureSlug)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": inventoryFeatureInternalErrorMessage})
			return
		}
		if !enabled {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": inventoryFeatureDisabledMessage})
			return
		}

		c.Next()
	}
}
