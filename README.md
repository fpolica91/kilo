# Kilo - AI Support Agent

A simple, elegant terminal application with Anthropic Claude integration.

## Features

- 80s Miami Vice inspired UI with neon colors
- Basic Anthropic Claude API integration
- Tool calling support (bash commands, time, etc.)
- Built with Bubble Tea TUI framework

## Setup

1. Set your Anthropic API key:
```bash
export ANTHROPIC_API_KEY="your-api-key-here"
```

2. Run the application:
```bash
go run main.go
```

## Anthropic Client Usage

### Simple Message Example

```go
package main

import (
    "context"
    "fmt"
    "kilo/internal/ai"
    "os"
)

func main() {
    client := ai.NewClient(os.Getenv("ANTHROPIC_API_KEY"))
    ctx := context.Background()

    messages := []ai.Message{
        {Role: "user", Content: "Hello, Claude!"},
    }

    response, err := client.SendMessage(ctx, messages)
    if err != nil {
        panic(err)
    }

    fmt.Println(response)
}
```

### With Tool Calling

```go
package main

import (
    "context"
    "fmt"
    "kilo/internal/ai"
    "os"
)

func main() {
    client := ai.NewClient(os.Getenv("ANTHROPIC_API_KEY"))
    executor := ai.NewToolExecutor()
    ctx := context.Background()

    messages := []ai.Message{
        {Role: "user", Content: "What time is it?"},
    }

    // Send message with available tools
    response, err := client.SendMessageWithTools(ctx, messages, executor.GetAvailableTools())
    if err != nil {
        panic(err)
    }

    // Execute any tool calls
    if len(response.ToolCalls) > 0 {
        for _, toolCall := range response.ToolCalls {
            result, err := executor.Execute(ctx, toolCall)
            if err != nil {
                fmt.Printf("Tool error: %v\n", err)
                continue
            }
            fmt.Printf("Tool result: %s\n", result)
        }
    } else {
        fmt.Println(response.Content)
    }
}
```

## Built-in Tools

- **bash**: Execute bash commands
  - Parameter: `command` (string)

- **get_time**: Get current date and time
  - No parameters

## Project Structure

```
kilo/
├── main.go              # Entry point
├── internal/
│   ├── ai/
│   │   ├── client.go    # Anthropic client wrapper
│   │   ├── tools.go     # Tool executor and definitions
│   │   └── example.go   # Usage examples
│   ├── logo/
│   │   └── logo.go      # 80s style logo rendering
│   └── tui/
│       └── tui.go       # Terminal UI
└── go.mod
```

## How It Works

### 1. Tool Definition

Tools are defined with a name, description, and parameters:

```go
Tool{
    Name:        "bash",
    Description: "Execute a bash command",
    Parameters: map[string]any{
        "command": map[string]any{
            "type":        "string",
            "description": "The bash command to execute",
        },
    },
    Required: []string{"command"},
}
```

### 2. Sending to Anthropic

Tools are converted to Anthropic's format and sent with the message:

```go
anthropicTools := make([]anthropic.ToolUnionParam, len(tools))
for i, tool := range tools {
    toolParam := anthropic.ToolParam{
        Name:        tool.Name,
        Description: anthropic.String(tool.Description),
        InputSchema: anthropic.ToolInputSchemaParam{
            Properties: tool.Parameters,
            Required:   tool.Required,
        },
    }
    anthropicTools[i] = anthropic.ToolUnionParam{OfTool: &toolParam}
}
```

### 3. Handling Response

Claude's response may include tool calls:

```go
for _, block := range response.Content {
    switch b := block.AsAny().(type) {
    case anthropic.TextBlock:
        content += b.Text
    case anthropic.ToolUseBlock:
        toolCalls = append(toolCalls, ToolCall{
            ID:    b.ID,
            Name:  b.Name,
            Input: string(b.Input),
        })
    }
}
```

### 4. Executing Tools

Tool calls are executed by registered handlers:

```go
func executeBash(ctx context.Context, input string) (string, error) {
    var params struct {
        Command string `json:"command"`
    }
    json.Unmarshal([]byte(input), &params)

    cmd := exec.CommandContext(ctx, "bash", "-c", params.Command)
    return cmd.CombinedOutput()
}
```

## License

MIT
# kilo
# kilo
# kilo
