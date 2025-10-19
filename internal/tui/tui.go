package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"kilo/internal/ai"
	"kilo/internal/logo"
	"kilo/internal/tools"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	width    int
	height   int
	client   *ai.Client
	executor *tools.Executor
	input    textarea.Model
	viewport viewport.Model
	messages []ai.Message
	ready    bool
	thinking bool
}

type responseMsg struct {
	content  string
	messages []ai.Message // Include updated messages
	err      error
}

type toolExecutedMsg struct {
	messages []ai.Message
}
type toolResponseMsg struct {
	result string
	err    error
}

func New() model {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")

	// Create textarea for input
	ta := textarea.New()
	ta.Placeholder = "Ask me anything..."
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 2000
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false

	// Create viewport for chat history
	vp := viewport.New(80, 20)

	return model{
		client:   ai.NewClient(apiKey),
		executor: tools.New(),
		input:    ta,
		viewport: vp,
		messages: []ai.Message{},
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.input, tiCmd = m.input.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = msg.Height - 10
			m.input.SetWidth(msg.Width - 4)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = msg.Height - 10
			m.input.SetWidth(msg.Width - 4)
		}

		m.viewport.SetContent(m.renderMessages())
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			if m.thinking {
				return m, nil
			}

			userInput := strings.TrimSpace(m.input.Value())
			if userInput == "" {
				return m, nil
			}

			// Add user message
			m.messages = append(m.messages, ai.Message{
				Role:    "user",
				Content: userInput,
			})

			// Clear input
			m.input.Reset()
			m.thinking = true

			// Update viewport
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()

			// Send message to Claude
			return m, m.sendMessage()
		}

	case responseMsg:
		m.thinking = false
		if msg.err != nil {
			m.messages = append(m.messages, ai.Message{
				Role:    "assistant",
				Content: fmt.Sprintf("Error: %v", msg.err),
			})
		} else {
			if len(msg.messages) > 0 {
				m.messages = msg.messages
			}
			// Add Claude's final response
			m.messages = append(m.messages, ai.Message{
				Role:    "assistant",
				Content: msg.content,
			})
		}
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil

	case toolResponseMsg:
		m.thinking = false
		if msg.err != nil {
			m.messages = append(m.messages, ai.Message{
				Role:    "assistant",
				Content: fmt.Sprintf("Tool Error: %v", msg.err),
			})
		} else {
			m.messages = append(m.messages, ai.Message{
				Role:    "assistant",
				Content: msg.result,
			})
		}
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil
	case toolExecutedMsg:
		m.messages = msg.messages
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, m.sendFinalMessage()

	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m model) sendMessage() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		tools := m.executor.GetAvailableTools()

		response, err := m.client.SendMessageWithTools(ctx, m.messages, tools)
		if err != nil {
			return responseMsg{err: err}
		}

		if len(response.ToolCalls) == 0 {
			return responseMsg{
				content:  response.Content,
				messages: m.messages,
			}
		}

		// Add assistant message with tool call to history
		for _, toolCall := range response.ToolCalls {
			m.messages = append(m.messages, ai.Message{
				Role:          "assistant",
				ToolCallID:    toolCall.ID,
				ToolCallName:  toolCall.Name,
				ToolCallInput: toolCall.Input,
			})

			// Execute the tool
			result, err := m.executor.Execute(ctx, toolCall)
			if err != nil {
				result = fmt.Sprintf("Error: %v", err)
			}

			// Truncate very long results (keep first 5000 chars for Claude)
			if len(result) > 5000 {
				result = result[:5000] + "\n... [output truncated, too long]"
			}

			// Add tool result to history (truncated version for API)
			m.messages = append(m.messages, ai.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: toolCall.ID,
			})
		}

		return toolExecutedMsg{
			messages: m.messages,
		}

	}
}

func (m model) sendFinalMessage() tea.Cmd {
	return func() tea.Msg {
		// Add 60 second timeout
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		tools := m.executor.GetAvailableTools()
		maxIterations := 5 // Prevent infinite loops

		// Loop until Claude responds with text (not more tool calls)
		for iteration := 0; iteration < maxIterations; iteration++ {
			// fmt.Fprintf(os.Stderr, "\n[DEBUG] Iteration %d: Sending %d messages to Claude\n", iteration+1, len(m.messages))

			response, err := m.client.SendMessageWithTools(ctx, m.messages, tools)
			if err != nil {
				// fmt.Fprintf(os.Stderr, "[DEBUG] API Error: %v\n", err)
				return responseMsg{err: fmt.Errorf("final response error: %w", err), messages: m.messages}
			}

			// fmt.Fprintf(os.Stderr, "[DEBUG] Response: Content=%d chars, ToolCalls=%d\n",
			// 	len(response.Content), len(response.ToolCalls))

			// If Claude wants to use more tools, execute them and loop
			if len(response.ToolCalls) > 0 {
				// fmt.Fprintf(os.Stderr, "[DEBUG] Claude wants to use %d more tool(s)\n", len(response.ToolCalls))

				for _, toolCall := range response.ToolCalls {
					// fmt.Fprintf(os.Stderr, "  - Executing: %s\n", toolCall.Name)

					// Add tool call to history
					m.messages = append(m.messages, ai.Message{
						Role:          "assistant",
						ToolCallID:    toolCall.ID,
						ToolCallName:  toolCall.Name,
						ToolCallInput: toolCall.Input,
					})

					// Execute the tool
					result, err := m.executor.Execute(ctx, toolCall)
					if err != nil {
						result = fmt.Sprintf("Error: %v", err)
					}

					// Truncate very long results
					if len(result) > 5000 {
						result = result[:5000] + "\n... [output truncated, too long]"
					}

					// Add tool result to history
					m.messages = append(m.messages, ai.Message{
						Role:       "tool",
						Content:    result,
						ToolCallID: toolCall.ID,
					})
				}

				// Continue loop to send tool results back to Claude
				continue
			}

			// Claude responded with text - we're done!
			if response.Content != "" {
				// fmt.Fprintf(os.Stderr, "[DEBUG] Claude provided final response (%d chars)\n", len(response.Content))
				return responseMsg{
					content:  response.Content,
					messages: m.messages,
				}
			}

			// Empty response with no tool calls - something's wrong
			return responseMsg{err: fmt.Errorf("empty response from Claude (no error, just empty content)"), messages: m.messages}
		}

		// Hit max iterations
		return responseMsg{err: fmt.Errorf("reached maximum iterations (%d) - Claude kept calling tools", maxIterations), messages: m.messages}
	}
}

func (m model) renderMessages() string {
	if len(m.messages) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true).
			Render("Start chatting with Claude...")
	}

	var output strings.Builder

	userStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FFFF")).
		Bold(true)

	assistantStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF10F0")).
		Bold(true)

	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA"))

	for _, msg := range m.messages {

		switch msg.Role {
		case "user":
			output.WriteString(userStyle.Render("You: "))
			output.WriteString(contentStyle.Render(msg.Content))
			output.WriteString("\n\n")
		case "assistant":
			// Only render if there's actual content (skip tool call messages)
			if msg.Content != "" {
				output.WriteString(assistantStyle.Render("Kilo: "))
				output.WriteString(contentStyle.Render(msg.Content))
				output.WriteString("\n\n")
			}
		case "tool":
			toolStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#666666")).
				Italic(true)
			output.WriteString(toolStyle.Render(fmt.Sprintf("Tool output:\n%s",
				msg.Content)))
			output.WriteString("\n\n")
		}
	}

	if m.thinking {
		output.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#B026FF")).
			Italic(true).
			Render("Kilo is thinking..."))
	}

	return output.String()
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if !m.ready {
		return "Initializing..."
	}

	hotPink := lipgloss.Color("#FF10F0")
	cyan := lipgloss.Color("#00FFFF")
	purple := lipgloss.Color("#B026FF")

	// Header with logo
	headerStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Width(m.width).
		Align(lipgloss.Center)

	logoView := logo.RenderWithTagline("AI Support Agent")
	header := headerStyle.Render(logoView)

	// Chat viewport
	viewportStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(purple).
		Padding(1, 2).
		Width(m.width - 2).
		Height(m.height - 12)

	chatView := viewportStyle.Render(m.viewport.View())

	// Input area
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(hotPink).
		Padding(0, 1).
		Width(m.width - 2)

	inputView := inputStyle.Render(m.input.View())

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Italic(true).
		Padding(0, 2)

	help := helpStyle.Render("Enter: send message | Esc/Ctrl+C: quit")

	// Status bar
	statusStyle := lipgloss.NewStyle().
		Foreground(cyan).
		Bold(true).
		Padding(0, 2)

	status := statusStyle.Render(fmt.Sprintf("Messages: %d", len(m.messages)))

	// Combine everything
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		chatView,
		inputView,
		help,
		status,
	)
}

func Run() error {
	p := tea.NewProgram(
		New(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running program: %w", err)
	}

	return nil
}
