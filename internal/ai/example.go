package ai

import (
	"context"
	"fmt"
	"os"
)

// Example demonstrates how to use the Anthropic client
func Example() {
	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: ANTHROPIC_API_KEY environment variable not set")
		return
	}

	// Create client
	client := NewClient(apiKey)
	ctx := context.Background()

	// Example 1: Simple message without tools
	fmt.Println("=== Example 1: Simple Message ===")
	messages := []Message{
		{Role: "user", Content: "What is the capital of France?"},
	}

	response, err := client.SendMessage(ctx, messages)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Response: %s\n\n", response)

	// Example 2: Message with tools
	fmt.Println("=== Example 2: Message with Tools ===")
	executor := NewToolExecutor()
	tools := executor.GetAvailableTools()

	messagesWithTools := []Message{
		{Role: "user", Content: "What time is it right now?"},
	}

	responseWithTools, err := client.SendMessageWithTools(ctx, messagesWithTools, tools)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Check if Claude wants to use tools
	if len(responseWithTools.ToolCalls) > 0 {
		fmt.Println("Claude wants to use tools:")
		for _, toolCall := range responseWithTools.ToolCalls {
			fmt.Printf("  Tool: %s (ID: %s)\n", toolCall.Name, toolCall.ID)
			fmt.Printf("  Input: %s\n", toolCall.Input)

			// Execute the tool
			result, err := executor.Execute(ctx, toolCall)
			if err != nil {
				fmt.Printf("  Error executing tool: %v\n", err)
				continue
			}
			fmt.Printf("  Result: %s\n", result)
		}
	} else {
		fmt.Printf("Response: %s\n", responseWithTools.Content)
	}

	// Example 3: Bash command execution
	fmt.Println("\n=== Example 3: Bash Command ===")
	bashMessages := []Message{
		{Role: "user", Content: "List the files in the current directory using ls -la"},
	}

	bashResponse, err := client.SendMessageWithTools(ctx, bashMessages, tools)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if len(bashResponse.ToolCalls) > 0 {
		for _, toolCall := range bashResponse.ToolCalls {
			fmt.Printf("Executing: %s\n", toolCall.Name)
			result, err := executor.Execute(ctx, toolCall)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			fmt.Printf("Output:\n%s\n", result)
		}
	}
}
