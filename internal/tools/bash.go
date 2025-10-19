package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"kilo/internal/ai"
)

// BashTool returns the bash tool definition
func BashTool() ai.Tool {
	return ai.Tool{
		Name:        "bash",
		Description: "Execute a bash command and return the output. Use this to run shell commands, check system information, or interact with the filesystem. For commands like 'top', use 'top -l 1' to get a single snapshot instead of continuous output.",
		Parameters: map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The bash command to execute (e.g., 'ls -la', 'date', 'pwd', 'top -l 1'). Use flags to limit output for commands that run continuously.",
			},
		},
		Required: []string{"command"},
	}
}

// ExecuteBash executes a bash command
func ExecuteBash(ctx context.Context, input string) (string, error) {
	var params struct {
		Command string `json:"command"`
	}

	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", params.Command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command failed: %w\nOutput: %s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}
