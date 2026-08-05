package mcp

import "testing"

func TestRequirementStatusValidation(t *testing.T) {
	for _, status := range []string{"draft", "submitted", "planning", "in_progress", "in_review", "accepted", "rejected", "cancelled"} {
		if err := validateRequirementStatus(status); err != nil {
			t.Fatalf("valid status %q rejected: %v", status, err)
		}
	}
	if err := validateRequirementStatus("bogus"); err == nil {
		t.Fatal("invalid status accepted")
	}
}

func TestRequirementUpdatePresence(t *testing.T) {
	status := "in_progress"
	empty := []string{}
	if !requirementUpdateRequested(&status, nil) {
		t.Fatal("status presence not detected")
	}
	if !requirementUpdateRequested(nil, &empty) {
		t.Fatal("explicit empty task list should request link replacement")
	}
	if requirementUpdateRequested(nil, nil) {
		t.Fatal("empty update should not be accepted")
	}
}
