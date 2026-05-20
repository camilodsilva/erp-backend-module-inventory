package access

import "errors"

var ErrAccessTenantRequired = errors.New("tenant id is required")

const ModuleInventory = "inventory"

type (
	AccessStatus struct {
		Module              string
		Enabled             bool
		CanRead             bool
		CanWrite            bool
		Ready               bool
		PendingRequirements []string
	}
)

func NewAccessStatus(canRead, canWrite bool) AccessStatus {
	return AccessStatus{
		Module:              ModuleInventory,
		Enabled:             true,
		CanRead:             canRead,
		CanWrite:            canWrite,
		Ready:               true,
		PendingRequirements: []string{},
	}
}
