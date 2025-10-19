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

func NvidiaSmiTool() ai.Tool {

	return ai.Tool{
		Name:        "nvidia_smi",
		Description: "Execute the nvidia-smi command on shell and return the output.",
		Parameters: map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The nvidia-smi command to execute (e.g., 'nvidia-smi', 'nvidia-smi -q', 'nvidia-smi -l 1'). Use flags to limit output for commands that run continuously.",
			},
		},
		Required: []string{"command"},
	}
}

func ExecuteNvidiaSmi(ctx context.Context, input string) (string, error) {
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
