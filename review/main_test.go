package main

import (
	"testing"

	review "github.com/jtamagnan/git-utils/review/lib"
)

func TestParsedArgsStructure(t *testing.T) {
	// Test that ParsedArgs has all expected fields
	args := review.ParsedArgs{
		NoVerify:    true,
		OpenBrowser: false,
		Draft:       true,
		Labels:      []string{"test", "label"},
	}

	if !args.NoVerify {
		t.Error("Expected NoVerify to be true")
	}
	if args.OpenBrowser {
		t.Error("Expected OpenBrowser to be false")
	}
	if !args.Draft {
		t.Error("Expected Draft to be true")
	}
	if len(args.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(args.Labels))
	}
	if args.Labels[0] != "test" || args.Labels[1] != "label" {
		t.Errorf("Expected ['test', 'label'], got %v", args.Labels)
	}
}
