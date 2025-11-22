package cmd

import (
	"testing"
)

func TestSyncCommand(t *testing.T) {
	// This test verifies the sync command can be created and has correct properties
	if syncCmd == nil {
		t.Fatal("syncCmd is nil")
	}

	if syncCmd.Use != "sync" {
		t.Errorf("syncCmd.Use = %q, want %q", syncCmd.Use, "sync")
	}

	if syncCmd.RunE == nil {
		t.Error("syncCmd.RunE is nil, should be set")
	}
}

func TestStatusCommand(t *testing.T) {
	// This test verifies the status command can be created and has correct properties
	if statusCmd == nil {
		t.Fatal("statusCmd is nil")
	}

	if statusCmd.Use != "status" {
		t.Errorf("statusCmd.Use = %q, want %q", statusCmd.Use, "status")
	}

	if statusCmd.RunE == nil {
		t.Error("statusCmd.RunE is nil, should be set")
	}
}
