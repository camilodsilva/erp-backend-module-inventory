package access

import "testing"

func TestCheckAccessUseCase_Execute_FullAccess(t *testing.T) {
	draft, _ := NewDraft("tenant-uuid", true, true)
	uc := NewCheckUseCase()
	status, err := uc.Execute(draft)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !status.CanRead || !status.CanWrite {
		t.Errorf("expected full access, got can_read=%v can_write=%v", status.CanRead, status.CanWrite)
	}
	if !status.Ready {
		t.Errorf("expected ready=true")
	}
	if len(status.PendingRequirements) != 0 {
		t.Errorf("expected empty pending_requirements")
	}
}

func TestCheckAccessUseCase_Execute_ReadOnly(t *testing.T) {
	draft, _ := NewDraft("tenant-uuid", true, false)
	uc := NewCheckUseCase()
	status, err := uc.Execute(draft)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !status.CanRead || status.CanWrite {
		t.Errorf("expected read-only, got can_read=%v can_write=%v", status.CanRead, status.CanWrite)
	}
}

func TestCheckAccessUseCase_Execute_NoAccess(t *testing.T) {
	draft, _ := NewDraft("tenant-uuid", false, false)
	uc := NewCheckUseCase()
	status, err := uc.Execute(draft)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if status.CanRead || status.CanWrite {
		t.Errorf("expected no access, got can_read=%v can_write=%v", status.CanRead, status.CanWrite)
	}
}
