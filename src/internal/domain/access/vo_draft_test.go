package access

import "testing"

func TestAccessDraft_NewDraft_Success(t *testing.T) {
	draft, err := NewDraft("tenant-uuid", true, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if draft.TenantID != "tenant-uuid" {
		t.Errorf("expected tenant-uuid, got %s", draft.TenantID)
	}
}

func TestAccessDraft_NewDraft_EmptyTenantID(t *testing.T) {
	_, err := NewDraft("", true, true)
	if err != ErrAccessTenantRequired {
		t.Errorf("expected ErrAccessTenantRequired, got %v", err)
	}
}

func TestAccessDraft_NewDraft_WhitespaceOnlyTenantID(t *testing.T) {
	_, err := NewDraft("   ", true, true)
	if err != ErrAccessTenantRequired {
		t.Errorf("expected ErrAccessTenantRequired, got %v", err)
	}
}
