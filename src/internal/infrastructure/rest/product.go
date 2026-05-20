package rest

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/domain/product"
	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/dto"
	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/postgres"
	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/shared"
	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/tenant"
	"github.com/gin-gonic/gin"
)

type productHttpHandler struct {
	createUseCase   *product.CreateUseCase
	findByIDUseCase *product.FindByIDUseCase
	findAllUseCase  *product.FindAllUseCase
	updateUseCase   *product.UpdateUseCase
	deleteUseCase   *product.DeleteUseCase
}

func newProductHttpHandler(db *sql.DB) *productHttpHandler {
	repository := postgres.NewProductPostgresRepository(db)
	factory := product.NewProductFactory(&shared.UUIDGenerator{})

	return &productHttpHandler{
		createUseCase:   product.NewCreateUseCase(repository, factory),
		findByIDUseCase: product.NewFindByIDUseCase(repository),
		findAllUseCase:  product.NewFindAllUseCase(repository),
		updateUseCase:   product.NewUpdateUseCase(repository),
		deleteUseCase:   product.NewDeleteUseCase(repository),
	}
}

func (h *productHttpHandler) HandleCreate(c *gin.Context) {
	var req dto.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		buildResponseError(c, http.StatusBadRequest, errors.New("invalid request body"))
		return
	}

	draft, err := req.ToDraft()
	if err != nil {
		buildResponseError(c, http.StatusBadRequest, err)
		return
	}

	p, err := h.createUseCase.Execute(tenant.GetTenantID(c), actorIDFromContext(c), draft)
	if err != nil {
		handleProductError(c, err)
		return
	}

	buildResponseSuccess(c, http.StatusCreated, dto.NewProductResponse(p))
}

func (h *productHttpHandler) HandleList(c *gin.Context) {
	page := parseStringToInt(c.Query("page"), 1)
	size := parseStringToInt(c.Query("size"), 10)

	result, err := h.findAllUseCase.Execute(tenant.GetTenantID(c), page, size)
	if err != nil {
		handleProductError(c, err)
		return
	}

	buildResponseSuccess(c, http.StatusOK, dto.NewProductPaginated(result))
}

func (h *productHttpHandler) HandleFindByID(c *gin.Context) {
	id := c.Param("id")

	p, err := h.findByIDUseCase.Execute(tenant.GetTenantID(c), id)
	if err != nil {
		handleProductError(c, err)
		return
	}

	buildResponseSuccess(c, http.StatusOK, dto.NewProductResponse(p))
}

func (h *productHttpHandler) HandleUpdate(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		buildResponseError(c, http.StatusBadRequest, errors.New("invalid request body"))
		return
	}

	draft, err := req.ToDraft()
	if err != nil {
		buildResponseError(c, http.StatusBadRequest, err)
		return
	}

	p, err := h.updateUseCase.Execute(tenant.GetTenantID(c), id, actorIDFromContext(c), draft)
	if err != nil {
		handleProductError(c, err)
		return
	}

	buildResponseSuccess(c, http.StatusOK, dto.NewProductResponse(p))
}

func (h *productHttpHandler) HandleDelete(c *gin.Context) {
	id := c.Param("id")

	if err := h.deleteUseCase.Execute(tenant.GetTenantID(c), id, actorIDFromContext(c)); err != nil {
		handleProductError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func handleProductError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, product.ErrProductNotFound):
		buildResponseError(c, http.StatusNotFound, err)
	case errors.Is(err, product.ErrProductAlreadyExists):
		buildResponseError(c, http.StatusConflict, err)
	default:
		log.Printf("product handler error: %v", err)
		buildResponseError(c, http.StatusInternalServerError, errors.New("internal server error"))
	}
}
