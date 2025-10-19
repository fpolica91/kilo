package tools

import (
	"context"

	"kilo/internal/ai"
)

// Executor wraps the tool executor with all registered tools
type Executor struct {
	executor *ai.ToolExecutor
}

// New creates a new tool executor with all built-in tools registered
func New() *Executor {
	executor := ai.NewToolExecutor()

	// Register all tools
	executor.RegisterTool("bash", ExecuteBash)
	executor.RegisterTool("nvidia_smi", ExecuteNvidiaSmi)

	return &Executor{executor: executor}
}

// Execute runs a tool
func (e *Executor) Execute(ctx context.Context, toolCall ai.ToolCall) (string, error) {
	return e.executor.Execute(ctx, toolCall)
}

// GetAvailableTools returns all available tools for Claude
func (e *Executor) GetAvailableTools() []ai.Tool {
	return []ai.Tool{
		BashTool(),
		NvidiaSmiTool(),
	}
}
