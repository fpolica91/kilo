package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type Client struct {
	client anthropic.Client
	model  string
}

func NewClient(apiKey string) *Client {
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &Client{
		client: client,
		model:  "claude-sonnet-4-20250514",
	}
}

type Message struct {
	Role          string
	Content       string
	ToolCallID    string // For assistant messages with tool calls, and tool result messages
	ToolCallName  string // For assistant messages with tool calls
	ToolCallInput string // For assistant messages with tool calls (JSON string)
}

func (c *Client) SendMessage(ctx context.Context, messages []Message) (string, error) {

	anthropicMessages := make([]anthropic.MessageParam, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(
				anthropic.NewTextBlock(msg.Content),
			))
		case "assistant":
			anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(
				anthropic.NewTextBlock(msg.Content),
			))
		}
		// Skip tool messages in simple send
	}

	response, err := c.client.Messages.New(
		ctx,
		anthropic.MessageNewParams{
			Model:     anthropic.Model(c.model),
			MaxTokens: 1024,
			Messages:  anthropicMessages,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	var content string
	for _, block := range response.Content {
		if text, ok := block.AsAny().(anthropic.TextBlock); ok {
			content += text.Text
		}
	}

	return content, nil
}

// SendMessageWithTools sends a message with tool support
func (c *Client) SendMessageWithTools(ctx context.Context, messages []Message, tools []Tool) (*Response, error) {
	// Convert messages to Anthropic format
	anthropicMessages := make([]anthropic.MessageParam, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(
				anthropic.NewTextBlock(msg.Content),
			))
		case "assistant":
			// For assistant messages, check if there are tool calls
			if msg.ToolCallID != "" && msg.ToolCallName != "" && msg.ToolCallInput != "" {
				// This is an assistant message with a tool call
				inputMap := make(map[string]any)
				json.Unmarshal([]byte(msg.ToolCallInput), &inputMap)
				anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(
					anthropic.NewToolUseBlock(msg.ToolCallID, inputMap, msg.ToolCallName),
				))
			} else if msg.Content != "" {
				anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(
					anthropic.NewTextBlock(msg.Content),
				))
			}
		case "tool":
			// Tool result message
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(
				anthropic.NewToolResultBlock(msg.ToolCallID, msg.Content, false),
			))
		}
	}

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

	response, err := c.client.Messages.New(
		ctx,
		anthropic.MessageNewParams{
			Model:     anthropic.Model(c.model),
			MaxTokens: 1024,
			Messages:  anthropicMessages,
			Tools:     anthropicTools,
			System: []anthropic.TextBlockParam{
				{
					Text: `You are Kilo, a helpful AI support agent. Use the tools available to you to assist the user.

# Tool Usage
- When you need information to answer a question, use tools immediately without announcing your intention
- The user sees the tool output, so you should interpret and explain what the results mean
- Be concise and direct in your responses

# Examples
<example>
user: what time is it?
assistant: [uses get_time tool which returns "Sat Oct 18 14:23:45 PDT 2025"]
The current time is 2:23 PM on Saturday, October 18th, 2025.
</example>

<example>
user: list files in current directory
assistant: [uses bash tool with "ls" which returns file list]
Your directory contains: main.go, README.md, and an internal/ folder.
</example>

IMPORTANT: Keep responses under 4 lines unless the user asks for more detail.`,
				},
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Extract content and tool calls
	var content string
	var toolCalls []ToolCall

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

	return &Response{
		Content:   content,
		ToolCalls: toolCalls,
	}, nil
}

// Response represents an AI response
type Response struct {
	Content   string
	ToolCalls []ToolCall
}

// ToolCall represents a tool call from Claude
type ToolCall struct {
	ID    string
	Name  string
	Input string
}

// Tool represents a tool definition
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any
	Required    []string
}
