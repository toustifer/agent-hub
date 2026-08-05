package handler

import "testing"

func TestRequirementTransitionMetadata(t *testing.T) {
	tests := []struct {
		status    string
		submitted bool
		accepted  bool
		decision  string
	}{
		{status: "submitted", submitted: true},
		{status: "accepted", accepted: true, decision: "accept"},
		{status: "rejected", decision: "reject"},
		{status: "cancelled"},
	}
	for _, tt := range tests {
		got := requirementTransitionMetadata(tt.status)
		if got.submitted != tt.submitted || got.accepted != tt.accepted || got.decision != tt.decision {
			t.Fatalf("status %q metadata = %+v, want submitted=%v accepted=%v decision=%q", tt.status, got, tt.submitted, tt.accepted, tt.decision)
		}
	}
}

func TestRequirementRolePermissions(t *testing.T) {
	for _, role := range []string{"admin", "owner"} {
		if !canManageRequirement(role) || !canCancelRequirement(role) {
			t.Fatalf("role %q should manage and cancel requirements", role)
		}
	}
	for _, role := range []string{"member", "reviewer", "api_key", ""} {
		if canCancelRequirement(role) {
			t.Fatalf("role %q should not cancel requirements", role)
		}
	}
}
