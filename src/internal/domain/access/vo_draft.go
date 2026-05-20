package access

import "strings"

type Draft struct {
	TenantID string
	CanRead  bool
	CanWrite bool
}

func NewDraft(tenantID string, canRead, canWrite bool) (Draft, error) {
	draft := Draft{
		TenantID: strings.TrimSpace(tenantID),
		CanRead:  canRead,
		CanWrite: canWrite,
	}

	if draft.TenantID == "" {
		return Draft{}, ErrAccessTenantRequired
	}

	return draft, nil
}
