package rest

import (
	"errors"
	"log"
	"net/http"

	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/domain/access"
	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/auth"
	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/dto"
	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/tenant"
	"github.com/gin-gonic/gin"
)

type accessHttpHandler struct {
	checkUseCase *access.CheckUseCase
}

func newAccessHttpHandler() *accessHttpHandler {
	return &accessHttpHandler{
		checkUseCase: access.NewCheckUseCase(),
	}
}

func (h *accessHttpHandler) HandleCheck(c *gin.Context) {
	roles := auth.RolesFromContext(c)
	draft, err := access.NewDraft(
		tenant.GetTenantID(c),
		auth.CanReadFeature(roles, access.ModuleInventory),
		auth.CanWriteFeature(roles, access.ModuleInventory),
	)
	if err != nil {
		handleAccessError(c, err)
		return
	}

	status, err := h.checkUseCase.Execute(draft)
	if err != nil {
		handleAccessError(c, err)
		return
	}

	buildResponseSuccess(c, http.StatusOK, dto.NewAccessResponse(status))
}

func handleAccessError(c *gin.Context, err error) {
	if errors.Is(err, access.ErrAccessTenantRequired) {
		buildResponseError(c, http.StatusForbidden, access.ErrAccessTenantRequired)
		return
	}

	log.Printf("access handler error: %v", err)
	buildResponseError(c, http.StatusInternalServerError, errors.New("internal server error"))
}
