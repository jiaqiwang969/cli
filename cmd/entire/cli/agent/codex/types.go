package codex

// Codex notify payload types.
const (
	PayloadTypeAgentTurnComplete   = "agent-turn-complete"
	PayloadTypeMcpToolCallComplete = "mcp-tool-call-complete"
)

// Codex hook command names used under `entire hooks codex`.
const (
	HookNameNotify = "notify"
)

type notifyEnvelope struct {
	Type string `json:"type"`
}

type agentTurnCompletePayload struct {
	Type                 string   `json:"type"`
	ThreadID             string   `json:"thread-id"`
	TurnID               string   `json:"turn-id"`
	Cwd                  string   `json:"cwd"`
	InputMessages        []string `json:"input-messages"`
	LastAssistantMessage *string  `json:"last-assistant-message"`
	ProviderName         string   `json:"provider-name"`
	ModelSlug            string   `json:"model-slug"`
}

type mcpToolCallCompletePayload struct {
	Type         string  `json:"type"`
	ThreadID     string  `json:"thread-id"`
	TurnID       string  `json:"turn-id"`
	CallID       string  `json:"call-id"`
	Server       string  `json:"server"`
	ToolName     string  `json:"tool-name"`
	DurationMS   uint64  `json:"duration-ms"`
	Status       string  `json:"status"`
	ErrorMessage *string `json:"error-message"`
	ProviderName string  `json:"provider-name"`
	ModelSlug    string  `json:"model-slug"`
	AgentName    *string `json:"agent-name"`
}
