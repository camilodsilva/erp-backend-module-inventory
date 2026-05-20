package dto

import "github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/domain/access"

type AccessResponse struct {
	Module              string   `json:"module"`
	Enabled             bool     `json:"enabled"`
	CanRead             bool     `json:"can_read"`
	CanWrite            bool     `json:"can_write"`
	Ready               bool     `json:"ready"`
	PendingRequirements []string `json:"pending_requirements"`
}

func NewAccessResponse(status access.AccessStatus) AccessResponse {
	return AccessResponse{
		Module:              status.Module,
		Enabled:             status.Enabled,
		CanRead:             status.CanRead,
		CanWrite:            status.CanWrite,
		Ready:               status.Ready,
		PendingRequirements: status.PendingRequirements,
	}
}
