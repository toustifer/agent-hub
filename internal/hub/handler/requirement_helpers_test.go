package handler

import (
	"strings"
	"testing"
)

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

func TestBuildUpdateRequirementSets(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	id := "req-123"
	var bizID int64 = 456

	t.Run("only title", func(t *testing.T) {
		req := updateRequirementReq{
			Title: strPtr("New Title"),
		}
		sets, args := buildUpdateRequirementSets(id, bizID, req)
		if len(args) != 3 {
			t.Fatalf("expected args len 3, got %d: %+v", len(args), args)
		}
		if len(sets) != 2 || sets[0] != "title = $3" || sets[1] != "updated_at = now()" {
			t.Fatalf("unexpected sets: %+v", sets)
		}
		if args[0] != id || args[1] != bizID || args[2] != "New Title" {
			t.Fatalf("unexpected args: %+v", args)
		}
	})

	t.Run("only description", func(t *testing.T) {
		req := updateRequirementReq{
			Description: strPtr("New Description"),
		}
		sets, args := buildUpdateRequirementSets(id, bizID, req)
		if len(args) != 3 {
			t.Fatalf("expected args len 3, got %d: %+v", len(args), args)
		}
		if len(sets) != 2 || sets[0] != "description = $3" || sets[1] != "updated_at = now()" {
			t.Fatalf("unexpected sets: %+v", sets)
		}
		if args[0] != id || args[1] != bizID || args[2] != "New Description" {
			t.Fatalf("unexpected args: %+v", args)
		}
	})

	t.Run("both title and description", func(t *testing.T) {
		req := updateRequirementReq{
			Title:       strPtr("New Title"),
			Description: strPtr("New Description"),
		}
		sets, args := buildUpdateRequirementSets(id, bizID, req)
		if len(args) != 4 {
			t.Fatalf("expected args len 4, got %d: %+v", len(args), args)
		}
		if len(sets) != 3 || sets[0] != "title = $3" || sets[1] != "description = $4" || sets[2] != "updated_at = now()" {
			t.Fatalf("unexpected sets: %+v", sets)
		}
		if args[0] != id || args[1] != bizID || args[2] != "New Title" || args[3] != "New Description" {
			t.Fatalf("unexpected args: %+v", args)
		}
	})
}

func TestRequirementTasksDoneQueryStatusCompatibility(t *testing.T) {
	queries := map[string]string{
		"ListRequirements": listRequirementsBaseQuery,
		"GetRequirement":   getRequirementQuery,
	}

	expectedStatuses := []string{"'completed'", "'done'", "'passed'"}

	for name, q := range queries {
		t.Run(name, func(t *testing.T) {
			if strings.Contains(q, "d.status = 'completed'") {
				t.Errorf("%s query still contains legacy hardcoded status = 'completed'", name)
			}
			if !strings.Contains(q, "d.status IN ('completed', 'done', 'passed')") {
				t.Errorf("%s query does not match expected d.status IN clause", name)
			}
			for _, status := range expectedStatuses {
				if !strings.Contains(q, status) {
					t.Errorf("%s query missing status %s in tasks_done condition", name, status)
				}
			}
		})
	}
}

func TestIsTaskStatusDone(t *testing.T) {
	doneStatuses := []string{"completed", "done", "passed"}
	for _, st := range doneStatuses {
		if !isTaskStatusDone(st) {
			t.Errorf("expected isTaskStatusDone(%q) = true, got false", st)
		}
	}

	notDoneStatuses := []string{"pending", "in_progress", "executing", "failed", "cancelled", "", "unknown"}
	for _, st := range notDoneStatuses {
		if isTaskStatusDone(st) {
			t.Errorf("expected isTaskStatusDone(%q) = false, got true", st)
		}
	}
}


