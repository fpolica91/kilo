# Tools Directory

This directory contains all tool definitions and executors for the Kilo AI agent.

## Structure

```
tools/
├── tools.go      # Main executor and registration
├── bash.go       # Bash command execution
├── time.go       # Get current time
└── README.md     # This file
```

## Adding a New Tool

### 1. Create a new file (e.g., `mytool.go`)

```go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"kilo/internal/ai"
)

// MyToolTool returns the tool definition
func MyToolTool() ai.Tool {
	return ai.Tool{
		Name:        "my_tool",
		Description: "Does something cool when the user asks for X",
		Parameters: map[string]any{
			"param1": map[string]any{
				"type":        "string",
				"description": "What this parameter does",
			},
		},
		Required: []string{"param1"},
	}
}

// ExecuteMyTool executes the tool
func ExecuteMyTool(ctx context.Context, input string) (string, error) {
	var params struct {
		Param1 string `json:"param1"`
	}

	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	// Do your thing
	result := fmt.Sprintf("Did something with: %s", params.Param1)

	return result, nil
}
```

### 2. Register it in `tools.go`

```go
func New() *Executor {
	executor := ai.NewToolExecutor()

	executor.RegisterTool("bash", ExecuteBash)
	executor.RegisterTool("get_time", ExecuteGetTime)
	executor.RegisterTool("my_tool", ExecuteMyTool)  // ← Add this

	return &Executor{executor: executor}
}

func (e *Executor) GetAvailableTools() []ai.Tool {
	return []ai.Tool{
		BashTool(),
		GetTimeTool(),
		MyToolTool(),  // ← Add this
	}
}
```

## How Tool Calling Works

### The Problem (Old Way)
```
User: "What time is it?"
  ↓
Claude: "I need to use get_time tool"
  ↓
Tool executes: "Sat Oct 18 14:23:45 PDT 2025"
  ↓
You display raw output: "Sat Oct 18 14:23:45 PDT 2025"  ← Not conversational!
```

### The Solution (New Way)
```
User: "What time is it?"
  ↓
Claude: "I need to use get_time tool"
  ↓
Tool executes: "Sat Oct 18 14:23:45 PDT 2025"
  ↓
Send tool result BACK to Claude
  ↓
Claude interprets: "The current time is 2:23 PM on Saturday, October 18th, 2025."  ← Conversational!
```

## Message Flow

### Complete Tool Calling Cycle

```go
// 1. User message
messages = [{role: "user", content: "What time is it?"}]

// 2. Send to Claude with tools
response = SendMessageWithTools(messages, tools)

// 3. Claude returns tool call
response.ToolCalls = [{
    ID: "toolu_123",
    Name: "get_time",
    Input: "{}"
}]

// 4. Add tool call to messages
messages.append({
    role: "assistant",
    toolCallID: "toolu_123",
    toolCallName: "get_time",
    toolCallInput: "{}"
})

// 5. Execute tool
result = "Sat Oct 18 14:23:45 PDT 2025"

// 6. Add tool result to messages
messages.append({
    role: "tool",
    content: "Sat Oct 18 14:23:45 PDT 2025",
    toolCallID: "toolu_123"
})

// 7. Send back to Claude with tool result
response = SendMessageWithTools(messages, tools)

// 8. Claude interprets and responds naturally
response.Content = "The current time is 2:23 PM on Saturday, October 18th, 2025."
```

## Key Points

1. **Never display raw tool output** - Always send it back to Claude for interpretation
2. **Tool results must reference the tool call ID** - This links the result to the call
3. **Add both tool call and result to message history** - This maintains context
4. **Claude needs to see the tool result** - Otherwise it can't interpret it

## Debugging

If Claude isn't responding conversationally:

```go
// Check 1: Are you adding tool results to messages?
messages = append(messages, ai.Message{
    Role: "tool",
    Content: result,
    ToolCallID: toolCall.ID,  // ← Must match!
})

// Check 2: Are you sending back to Claude?
response = client.SendMessageWithTools(ctx, messages, tools)  // ← Second call!

// Check 3: Are you displaying Claude's interpretation?
fmt.Println(response.Content)  // ← Not the raw tool output!
```

## Example: Adding a Weather Tool

```go
// weather.go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"kilo/internal/ai"
)

func WeatherTool() ai.Tool {
	return ai.Tool{
		Name:        "get_weather",
		Description: "Get the current weather for a location",
		Parameters: map[string]any{
			"location": map[string]any{
				"type":        "string",
				"description": "City name (e.g., 'San Francisco, CA')",
			},
		},
		Required: []string{"location"},
	}
}

func ExecuteGetWeather(ctx context.Context, input string) (string, error) {
	var params struct {
		Location string `json:"location"`
	}

	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	// In real life, call a weather API
	// For demo, return mock data
	result := fmt.Sprintf("Temperature: 72°F, Conditions: Sunny, Location: %s", params.Location)

	return result, nil
}
```

Then register it in `tools.go` and you're done!

User asks: "What's the weather in SF?"
Claude uses `get_weather` tool with location "San Francisco, CA"
Tool returns: "Temperature: 72°F, Conditions: Sunny, Location: San Francisco, CA"
Claude responds: "It's currently 72 degrees and sunny in San Francisco!"
