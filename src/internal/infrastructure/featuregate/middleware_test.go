package featuregate

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type mockFeatureGateRepository struct {
	hasFeature bool
	err        error
}

func (m *mockFeatureGateRepository) HasFeature(companyID, featureSlug string) (bool, error) {
	return m.hasFeature, m.err
}

func setup(companyID string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	if companyID != "" {
		c.Set("company_id", companyID)
	}
	return c, w
}

func TestRequireInventoryFeature_EmptyCompanyID_Returns403(t *testing.T) {
	c, w := setup("")

	handler := requireInventoryFeature(&mockFeatureGateRepository{hasFeature: true})
	handler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
	if !c.IsAborted() {
		t.Error("expected context to be aborted")
	}
}

func TestRequireInventoryFeature_HasFeatureFalse_Returns403(t *testing.T) {
	c, w := setup("company-uuid")

	handler := requireInventoryFeature(&mockFeatureGateRepository{hasFeature: false})
	handler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
	if !c.IsAborted() {
		t.Error("expected context to be aborted")
	}
}

func TestRequireInventoryFeature_RepositoryError_Returns500(t *testing.T) {
	c, w := setup("company-uuid")

	handler := requireInventoryFeature(&mockFeatureGateRepository{err: errors.New("db error")})
	handler(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
	if !c.IsAborted() {
		t.Error("expected context to be aborted")
	}
}

func TestRequireInventoryFeature_HasFeatureTrue_CallsNext(t *testing.T) {
	c, w := setup("company-uuid")

	nextCalled := false
	c.Set("_test_next", &nextCalled)

	handler := requireInventoryFeature(&mockFeatureGateRepository{hasFeature: true})
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (no abort), got %d", w.Code)
	}
	if c.IsAborted() {
		t.Error("expected context NOT to be aborted")
	}
}
