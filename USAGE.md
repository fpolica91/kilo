# Kilo - Usage Guide

## How the Agent is Tied to the TUI

### Architecture Overview

```
┌─────────────────────────────────────────────┐
│            Bubble Tea TUI                    │
│  ┌───────────────────────────────────────┐  │
│  │  model struct                         │  │
│  │  - client: *ai.Client                 │  │
│  │  - executor: *ai.ToolExecutor         │  │
│  │  - messages: []ai.Message             │  │
│  │  - input: textarea.Model              │  │
│  │  - viewport: viewport.Model           │  │
│  └───────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────────┐
│         AI Client (internal/ai)              │
│  ┌───────────────────────────────────────┐  │
│  │  SendMessageWithTools()               │  │
│  │  - Converts messages                  │  │
│  │  - Converts tools                     │  │
│  │  - Calls Anthropic SDK                │  │
│  │  - Returns Response + ToolCalls       │  │
│  └───────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────────┐
│      Anthropic SDK                           │
│  github.com/anthropics/anthropic-sdk-go     │
│  - Messages.New()                            │
│  - Handles HTTP requests                     │
│  - Returns anthropic.Message                 │
└─────────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────────┐
│      Tool Executor                           │
│  - Execute(toolCall)                         │
│  - Runs registered handlers                  │
│  - Returns tool results                      │
└─────────────────────────────────────────────┘
```

## Key Components

### 1. TUI Model State

```go
type model struct {
    width    int                  // Terminal width
    height   int                  // Terminal height
    client   *ai.Client          // Anthropic client
    executor *ai.ToolExecutor    // Tool executor
    input    textarea.Model      // User input box
    viewport viewport.Model      // Chat history display
    messages []ai.Message        // Conversation history
    ready    bool                // UI ready flag
    thinking bool                // Waiting for response
}
```

### 2. Message Flow

**User Types Message:**
1. User presses Enter
2. `Update()` handles `tea.KeyEnter`
3. User message added to `m.messages`
4. `m.sendMessage()` returns a `tea.Cmd`
5. `sendMessage()` runs in goroutine (non-blocking)

**Sending to Claude:**
```go
func (m model) sendMessage() tea.Cmd {
    return func() tea.Msg {
        ctx := context.Background()
        tools := m.executor.GetAvailableTools()

        // Send all messages (maintains context)
        response, err := m.client.SendMessageWithTools(ctx, m.messages, tools)

        // Check for tool calls
        if len(response.ToolCalls) > 0 {
            // Execute tools
            for _, toolCall := range response.ToolCalls {
                result, _ := m.executor.Execute(ctx, toolCall)
            }
            return toolResponseMsg{result: results}
        }

        return responseMsg{content: response.Content}
    }
}
```

**Receiving Response:**
1. `sendMessage()` goroutine completes
2. Returns `responseMsg` or `toolResponseMsg`
3. `Update()` handles the message
4. Appends to `m.messages`
5. Re-renders viewport with `m.renderMessages()`
6. `m.viewport.GotoBottom()` scrolls to new message

### 3. Conversation Persistence

**In-Memory Only (Currently):**
- Messages stored in `m.messages` slice
- Lost when app closes
- Simple and lightweight

**To Add Database Persistence (like Crush):**
```go
// 1. Add database
import "github.com/jmoiron/sqlx"

type model struct {
    // ... existing fields
    db *sqlx.DB
    sessionID string
}

// 2. Save messages on create
func (m model) sendMessage() tea.Cmd {
    return func() tea.Msg {
        // ... send to API

        // Save to DB
        m.saveMessage(Message{
            SessionID: m.sessionID,
            Role: "user",
            Content: userInput,
        })

        // ... handle response

        m.saveMessage(Message{
            SessionID: m.sessionID,
            Role: "assistant",
            Content: response.Content,
        })
    }
}

// 3. Load on startup
func New() model {
    db := openDB()
    sessionID := createOrLoadSession()
    messages := loadMessages(sessionID)

    return model{
        db: db,
        sessionID: sessionID,
        messages: messages,
        // ...
    }
}
```

### 4. Tool Calling Integration

**Tool Registration:**
```go
executor := ai.NewToolExecutor()
executor.RegisterTool("bash", executeBash)
executor.RegisterTool("get_time", executeGetTime)
```

**Tool Execution:**
```go
// Claude decides to use tool
response.ToolCalls = [{
    ID: "toolu_123",
    Name: "bash",
    Input: `{"command": "ls -la"}`
}]

// Execute tool
result := executor.Execute(ctx, toolCall)

// Result displayed in chat
```

## Usage

### Run the App

```bash
export ANTHROPIC_API_KEY="your-key"
cd ~/Desktop/kilo
go run main.go
```

### Chat Interface

```
╭─────────────────────────────────────────╮
│              KILO Logo                  │
│        Your AI Support Agent            │
╰─────────────────────────────────────────╯

╭─────────────────────────────────────────╮
│ You: What time is it?                   │
│                                         │
│ Claude: Let me check that for you.     │
│                                         │
│ Executed get_time:                      │
│ Sat Oct 18 14:23:45 PDT 2025           │
╰─────────────────────────────────────────╯

╭─────────────────────────────────────────╮
│ Ask me anything...                      │
╰─────────────────────────────────────────╯

Enter: send | Esc: quit
Messages: 2
```

### Key Bindings

- **Enter**: Send message
- **Esc/Ctrl+C**: Quit
- **Up/Down**: Scroll chat history (when focused on viewport)

## Extending

### Add Custom Tools

```go
// 1. Create handler
func executeMyTool(ctx context.Context, input string) (string, error) {
    var params struct {
        Param1 string `json:"param1"`
    }
    json.Unmarshal([]byte(input), &params)

    // Do something
    return result, nil
}

// 2. Register in tools.go
func NewToolExecutor() *ToolExecutor {
    executor := &ToolExecutor{
        tools: make(map[string]ToolHandler),
    }

    executor.RegisterTool("my_tool", executeMyTool)

    return executor
}

// 3. Add to GetAvailableTools()
func (te *ToolExecutor) GetAvailableTools() []Tool {
    return []Tool{
        {
            Name: "my_tool",
            Description: "Does something cool",
            Parameters: map[string]any{
                "param1": map[string]any{
                    "type": "string",
                    "description": "A parameter",
                },
            },
            Required: []string{"param1"},
        },
    }
}
```

### Add Streaming Responses

Replace `Messages.New()` with `Messages.NewStreaming()`:

```go
func (m model) sendMessage() tea.Cmd {
    return func() tea.Msg {
        stream := m.client.client.Messages.NewStreaming(ctx, params)

        for stream.Next() {
            event := stream.Current()
            // Send partial updates to TUI
            // ...
        }
    }
}
```

## Summary

The agent is tied to the TUI through:

1. **State**: `model` struct holds `client` and `executor`
2. **Messages**: User input → `sendMessage()` → Anthropic → Response
3. **Commands**: Bubble Tea `tea.Cmd` for async operations
4. **Updates**: Response messages trigger re-renders
5. **Tools**: Executor handles tool calls from Claude
6. **Display**: Viewport shows conversation history

It's a clean MVU (Model-View-Update) pattern with Anthropic SDK integration!
