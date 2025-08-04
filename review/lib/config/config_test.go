package config

import (
	"reflect"
	"testing"

	review "github.com/jtamagnan/git-utils/review/lib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestCommaString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single item",
			input:    "bug",
			expected: []string{"bug"},
		},
		{
			name:     "multiple items",
			input:    "bug,feature,enhancement",
			expected: []string{"bug", "feature", "enhancement"},
		},
		{
			name:     "items with spaces",
			input:    "bug, feature , enhancement",
			expected: []string{"bug", "feature", "enhancement"},
		},
		{
			name:     "empty items filtered",
			input:    "bug,,feature,",
			expected: []string{"bug", "feature"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := ParseCommaString(tt.input)
			result := cs.ToStringSlice()

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseCommaString(%q) = %v, want %v", tt.input, result, tt.expected)
			}

			// Test round-trip: string -> CommaString -> string
			if tt.input != "" {
				roundTrip := cs.String()
				expectedRoundTrip := joinNonEmpty(tt.expected, ",")
				if roundTrip != expectedRoundTrip {
					t.Errorf("Round-trip failed: %q -> %q, expected %q", tt.input, roundTrip, expectedRoundTrip)
				}
			}
		})
	}
}

func TestCommaStringAppend(t *testing.T) {
	cs := CommaString{"bug"}
	cs.Append("feature", "enhancement")

	expected := []string{"bug", "feature", "enhancement"}
	result := cs.ToStringSlice()

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Append failed: got %v, want %v", result, expected)
	}

	// Test duplicate prevention
	cs.Append("bug", "docs")
	expected = []string{"bug", "feature", "enhancement", "docs"}
	result = cs.ToStringSlice()

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Append with duplicates failed: got %v, want %v", result, expected)
	}
}

func TestCommaStringContains(t *testing.T) {
	cs := CommaString{"bug", "feature", "enhancement"}

	if !cs.Contains("bug") {
		t.Error("Contains should return true for 'bug'")
	}

	if cs.Contains("docs") {
		t.Error("Contains should return false for 'docs'")
	}
}

func TestViperConfigIntegration(t *testing.T) {
	// Reset viper state for testing
	viper.Reset()

	// Test that InitConfig sets expected defaults
	InitConfig()

	if !viper.GetBool("open-browser") {
		t.Error("Expected open-browser default to be true")
	}

	if viper.GetBool("draft") {
		t.Error("Expected draft default to be false")
	}

	if viper.GetBool("no-verify") {
		t.Error("Expected no-verify default to be false")
	}

	labels := viper.GetStringSlice("labels")
	if len(labels) != 0 {
		t.Errorf("Expected empty labels default, got %v", labels)
	}
}

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
			expectedLabels: []string{"bug", "enhancement"},
			description:    "Empty labels should be filtered out (they serve no purpose)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper for each test
			viper.Reset()
			InitConfig()

			// Create a test command
			cmd := &cobra.Command{Use: "test"}
			SetupFlags(cmd)

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

			// Call ParseArgs to test our parsing logic
			var parsedArgs review.ParsedArgs
			parsedArgs, err = ParseArgs(cmd, []string{})
			if err != nil {
				t.Fatalf("%s: ParseArgs failed: %v", tt.description, err)
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

func TestCommandFlags(t *testing.T) {
	viper.Reset()
	InitConfig()

	cmd := &cobra.Command{Use: "test"}
	SetupFlags(cmd)

	// Test that all expected flags are present
	expectedFlags := []string{"no-verify", "open-browser", "draft", "labels", "reviewers"}

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

	// Test the open-browser flag
	openBrowserFlag := cmd.Flags().Lookup("open-browser")
	if openBrowserFlag == nil {
		t.Fatal("open-browser flag not found")
	}

	if openBrowserFlag.Shorthand != "b" {
		t.Errorf("Expected open-browser flag shorthand to be 'b', got %q", openBrowserFlag.Shorthand)
	}

	if openBrowserFlag.Usage == "" {
		t.Error("Expected open-browser flag to have usage text")
	}
}

func TestEnvironmentVariableSupport(t *testing.T) {
	// Test that environment variables work with REVIEW_ prefix
	viper.Reset()

	// Set test environment variables
	t.Setenv("REVIEW_OPEN_BROWSER", "false")
	t.Setenv("REVIEW_DRAFT", "true")
	t.Setenv("REVIEW_NO_VERIFY", "true")
	t.Setenv("REVIEW_LABELS", "env-test,automated")
	t.Setenv("REVIEW_REVIEWERS", "alice,bob")

	// Initialize config which should pick up env vars
	InitConfig()

	// Test that environment variables are respected
	if viper.GetBool("open-browser") {
		t.Error("Expected open-browser to be false from env var")
	}

	if !viper.GetBool("draft") {
		t.Error("Expected draft to be true from env var")
	}

	if !viper.GetBool("no-verify") {
		t.Error("Expected no-verify to be true from env var")
	}

	if viper.GetString("labels") != "env-test,automated" {
		t.Errorf("Expected labels to be 'env-test,automated' from env var, got: %s", viper.GetString("labels"))
	}

	if viper.GetString("reviewers") != "alice,bob" {
		t.Errorf("Expected reviewers to be 'alice,bob' from env var, got: %s", viper.GetString("reviewers"))
	}
}

func TestReviewersParsing(t *testing.T) {
	tests := []struct {
		name              string
		reviewersFlag     string
		expectedReviewers []string
		description       string
	}{
		{
			name:              "NoReviewers",
			reviewersFlag:     "",
			expectedReviewers: nil,
			description:       "Empty reviewers flag should result in nil slice",
		},
		{
			name:              "SingleReviewer",
			reviewersFlag:     "alice",
			expectedReviewers: []string{"alice"},
			description:       "Single reviewer should be parsed correctly",
		},
		{
			name:              "MultipleReviewers",
			reviewersFlag:     "alice,bob,charlie",
			expectedReviewers: []string{"alice", "bob", "charlie"},
			description:       "Multiple reviewers should be split by comma",
		},
		{
			name:              "ReviewersWithSpaces",
			reviewersFlag:     "alice, bob , charlie ",
			expectedReviewers: []string{"alice", "bob", "charlie"},
			description:       "Reviewers with spaces should be trimmed",
		},
		{
			name:              "EmptyReviewersInList",
			reviewersFlag:     "alice,,bob",
			expectedReviewers: []string{"alice", "bob"},
			description:       "Empty reviewers should be filtered out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper for each test
			viper.Reset()
			InitConfig()

			// Create a test command
			cmd := &cobra.Command{Use: "test"}
			SetupFlags(cmd)

			// Set the reviewers flag
			var args []string
			if tt.reviewersFlag != "" {
				args = []string{"--reviewers", tt.reviewersFlag}
			} else {
				args = []string{}
			}

			// Set the arguments and parse flags
			cmd.SetArgs(args)
			err := cmd.ParseFlags(args)
			if err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			// Call ParseArgs to test our parsing logic
			var parsedArgs review.ParsedArgs
			parsedArgs, err = ParseArgs(cmd, []string{})
			if err != nil {
				t.Fatalf("%s: ParseArgs failed: %v", tt.description, err)
			}

			// Check the parsed reviewers
			if len(parsedArgs.Reviewers) != len(tt.expectedReviewers) {
				t.Errorf("%s: expected %d reviewers, got %d", tt.description, len(tt.expectedReviewers), len(parsedArgs.Reviewers))
				t.Errorf("Expected: %v", tt.expectedReviewers)
				t.Errorf("Got: %v", parsedArgs.Reviewers)
				return
			}

			for i, expected := range tt.expectedReviewers {
				if i < len(parsedArgs.Reviewers) && parsedArgs.Reviewers[i] != expected {
					t.Errorf("%s: expected reviewer %d to be %q, got %q", tt.description, i, expected, parsedArgs.Reviewers[i])
				}
			}
		})
	}
}

// joinNonEmpty joins non-empty strings with separator
func joinNonEmpty(items []string, sep string) string {
	var nonEmpty []string
	for _, item := range items {
		if item != "" {
			nonEmpty = append(nonEmpty, item)
		}
	}
	if len(nonEmpty) == 0 {
		return ""
	}
	result := nonEmpty[0]
	for _, item := range nonEmpty[1:] {
		result += sep + item
	}
	return result
}
