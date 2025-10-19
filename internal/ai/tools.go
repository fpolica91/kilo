package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type ToolExecutor struct {
	tools map[string]ToolHandler
}

type ToolHandler func(ctx context.Context, input string) (string, error)

func NewToolExecutor() *ToolExecutor {
	executor := &ToolExecutor{
		tools: make(map[string]ToolHandler),
	}

	// Register built-in tools
	executor.RegisterTool("bash", executeBash)
	executor.RegisterTool("get_time", executeGetTime)

	return executor
}

func (te *ToolExecutor) RegisterTool(name string, handler ToolHandler) {
	te.tools[name] = handler
}

func (te *ToolExecutor) Execute(ctx context.Context, toolCall ToolCall) (string, error) {
	handler, exists := te.tools[toolCall.Name]
	if !exists {
		return "", fmt.Errorf("tool not found: %s", toolCall.Name)
	}

	return handler(ctx, toolCall.Input)
}

func (te *ToolExecutor) GetAvailableTools() []Tool {
	return []Tool{
		{
			Name:        "bash",
			Description: "Execute a bash command and return the output",
			Parameters: map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "The bash command to execute",
				},
			},
			Required: []string{"command"},
		},
		{
			Name:        "get_time",
			Description: "Get the current date and time",
			Parameters:  map[string]any{},
			Required:    []string{},
		},
	}
}

func executeBash(ctx context.Context, input string) (string, error) {
	var params struct {
		Command string `json:"command"`
	}

	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	cmd := exec.CommandContext(ctx, "bash", "-c", params.Command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command failed: %w\nOutput: %s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

// executeGetTime returns the current time
func executeGetTime(ctx context.Context, input string) (string, error) {
	cmd := exec.CommandContext(ctx, "date")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get time: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
