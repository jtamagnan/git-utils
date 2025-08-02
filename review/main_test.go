package main

import (
	"testing"

	review "github.com/jtamagnan/git-utils/review/lib"
)

func TestLabelsParsing(t *testing.T) {
	tests := []struct {
		name           string
		labelsFlag     string
		expectedLabels []string
		description    string
	}{
		{
			name:           "NoLabels",
			labelsFlag:     "",
			expectedLabels: nil,
			description:    "Empty labels flag should result in nil slice",
		},
		{
			name:           "SingleLabel",
			labelsFlag:     "bug",
			expectedLabels: []string{"bug"},
			description:    "Single label should be parsed correctly",
		},
		{
			name:           "MultipleLabels",
			labelsFlag:     "bug,enhancement,high-priority",
			expectedLabels: []string{"bug", "enhancement", "high-priority"},
			description:    "Multiple labels should be split by comma",
		},
		{
			name:           "LabelsWithSpaces",
			labelsFlag:     "bug, enhancement , high-priority ",
			expectedLabels: []string{"bug", "enhancement", "high-priority"},
			description:    "Labels with spaces should be trimmed",
		},
		{
			name:           "EmptyLabelsInList",
			labelsFlag:     "bug,,enhancement",
			expectedLabels: []string{"bug", "", "enhancement"},
			description:    "Empty labels should be preserved (GitHub will ignore them)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test command
			cmd := generateCommand()

			// Set the labels flag
			var args []string
			if tt.labelsFlag != "" {
				args = []string{"--labels", tt.labelsFlag}
			} else {
				args = []string{}
			}

			// Set the arguments and parse flags
			cmd.SetArgs(args)
			err := cmd.ParseFlags(args)
			if err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			// Call parseArgs to test our parsing logic
			parsedArgs, err := parseArgs(cmd, []string{})
			if err != nil {
				t.Fatalf("%s: parseArgs failed: %v", tt.description, err)
			}

			// Check the parsed labels
			if len(parsedArgs.Labels) != len(tt.expectedLabels) {
				t.Errorf("%s: expected %d labels, got %d", tt.description, len(tt.expectedLabels), len(parsedArgs.Labels))
				t.Errorf("Expected: %v", tt.expectedLabels)
				t.Errorf("Got: %v", parsedArgs.Labels)
				return
			}

			for i, expected := range tt.expectedLabels {
				if i < len(parsedArgs.Labels) && parsedArgs.Labels[i] != expected {
					t.Errorf("%s: expected label %d to be %q, got %q", tt.description, i, expected, parsedArgs.Labels[i])
				}
			}
		})
	}
}

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

func TestCommandFlags(t *testing.T) {
	cmd := generateCommand()

	// Test that all expected flags are present
	expectedFlags := []string{"no-verify", "open-browser", "draft", "labels"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag %q to be present", flagName)
		}
	}

	// Test the labels flag specifically
	labelsFlag := cmd.Flags().Lookup("labels")
	if labelsFlag == nil {
		t.Fatal("labels flag not found")
	}

	if labelsFlag.Shorthand != "l" {
		t.Errorf("Expected labels flag shorthand to be 'l', got %q", labelsFlag.Shorthand)
	}

	if labelsFlag.Usage == "" {
		t.Error("Expected labels flag to have usage text")
	}
}
