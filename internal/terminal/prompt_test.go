package terminal

import (
	"strings"
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
)

func TestConfirm_YesInput(t *testing.T) {
	input := strings.NewReader("y\n")
	output := &strings.Builder{}

	cmd := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}}
	check := &security.CheckResult{
		Allowed:      true,
		RequiresAuth: true,
		Warning:      "Dangerous command",
		Reason:       "In dangerous list",
	}

	result, err := ConfirmWithIO(cmd, check, input, output)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result {
		t.Error("Expected confirmation to succeed")
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "此操作需要您的授权") {
		t.Error("Expected authorization prompt in output")
	}
}

func TestConfirm_SkipInput(t *testing.T) {
	input := strings.NewReader("s\n")
	output := &strings.Builder{}

	cmd := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}}
	check := &security.CheckResult{
		Allowed:      true,
		RequiresAuth: true,
		Warning:      "Dangerous command",
	}

	result, err := ConfirmWithIO(cmd, check, input, output)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result {
		t.Error("Expected confirmation to be skipped")
	}
}

func TestConfirm_QuitInput(t *testing.T) {
	input := strings.NewReader("q\n")
	output := &strings.Builder{}

	cmd := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}}
	check := &security.CheckResult{Allowed: true, RequiresAuth: true}

	_, err := ConfirmWithIO(cmd, check, input, output)
	if err != ErrQuitAll {
		t.Errorf("Expected ErrQuitAll, got %v", err)
	}
}

func TestConfirm_InvalidThenValid(t *testing.T) {
	input := strings.NewReader("invalid\ny\n")
	output := &strings.Builder{}

	cmd := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}}
	check := &security.CheckResult{Allowed: true, RequiresAuth: true}

	result, err := ConfirmWithIO(cmd, check, input, output)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result {
		t.Error("Expected confirmation to succeed after invalid input")
	}
}

func TestConfirm_CaseInsensitive(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"Y\n", true},
		{"y\n", true},
		{"S\n", false},
		{"s\n", false},
		{"Q\n", false},
		{"q\n", false},
	}

	for _, tt := range tests {
		input := strings.NewReader(tt.input)
		output := &strings.Builder{}

		cmd := ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}}
		check := &security.CheckResult{Allowed: true, RequiresAuth: true}

		result, err := ConfirmWithIO(cmd, check, input, output)
		if tt.input == "Q\n" || tt.input == "q\n" {
			if err != ErrQuitAll {
				t.Errorf("Input %s: expected ErrQuitAll", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("Input %s: expected no error, got %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("Input %s: expected %v, got %v", tt.input, tt.expected, result)
			}
		}
	}
}
