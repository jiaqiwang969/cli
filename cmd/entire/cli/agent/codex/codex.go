// Package codex implements the Agent interface for OpenAI Codex CLI.
package codex

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/entireio/cli/cmd/entire/cli/agent"
	"github.com/entireio/cli/cmd/entire/cli/sessionid"
)

const (
	defaultCodexConfigRelativePath = ".codex/config.toml"
)

//nolint:gochecknoinits // Agent self-registration is the intended pattern
func init() {
	agent.Register(agent.AgentNameCodex, NewCodexAgent)
}

// CodexAgent implements the Agent interface for Codex CLI.
//
//nolint:revive // CodexAgent is clearer than Agent in this context
type CodexAgent struct{}

func NewCodexAgent() agent.Agent {
	return &CodexAgent{}
}

// Name returns the agent registry key.
func (c *CodexAgent) Name() agent.AgentName {
	return agent.AgentNameCodex
}

// Type returns the agent type identifier.
func (c *CodexAgent) Type() agent.AgentType {
	return agent.AgentTypeCodex
}

// Description returns a human-readable description.
func (c *CodexAgent) Description() string {
	return "Codex CLI - OpenAI's coding assistant"
}

// DetectPresence checks if Codex CLI is configured on this machine.
func (c *CodexAgent) DetectPresence() (bool, error) {
	configPath := codexConfigPath()
	if _, err := os.Stat(configPath); err == nil {
		return true, nil
	}
	if _, err := os.Stat(".codex"); err == nil {
		return true, nil
	}
	return false, nil
}

// GetHookConfigPath returns the path to Codex notify hook config.
func (c *CodexAgent) GetHookConfigPath() string {
	return codexConfigPath()
}

// SupportsHooks returns true as Codex supports a notify hook command.
func (c *CodexAgent) SupportsHooks() bool {
	return true
}

// ParseHookInput parses Codex notify payload from argv or stdin.
func (c *CodexAgent) ParseHookInput(hookType agent.HookType, reader io.Reader) (*agent.HookInput, error) {
	data, err := readCodexHookPayload(reader)
	if err != nil {
		return nil, err
	}

	var envelope notifyEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse codex notify payload: %w", err)
	}

	input := &agent.HookInput{
		HookType:  hookType,
		Timestamp: time.Now(),
		RawData:   make(map[string]interface{}),
	}
	input.RawData["payload_type"] = envelope.Type

	switch envelope.Type {
	case PayloadTypeAgentTurnComplete:
		var payload agentTurnCompletePayload
		if err := json.Unmarshal(data, &payload); err != nil {
			return nil, fmt.Errorf("failed to parse codex agent payload: %w", err)
		}

		input.SessionID = payload.ThreadID
		if len(payload.InputMessages) > 0 {
			input.UserPrompt = payload.InputMessages[len(payload.InputMessages)-1]
		}

		input.RawData["thread_id"] = payload.ThreadID
		input.RawData["turn_id"] = payload.TurnID
		input.RawData["cwd"] = payload.Cwd
		input.RawData["input_messages"] = payload.InputMessages
		if payload.LastAssistantMessage != nil {
			input.RawData["last_assistant_message"] = *payload.LastAssistantMessage
		}
		input.RawData["provider_name"] = payload.ProviderName
		input.RawData["model_slug"] = payload.ModelSlug

	case PayloadTypeMcpToolCallComplete:
		var payload mcpToolCallCompletePayload
		if err := json.Unmarshal(data, &payload); err != nil {
			return nil, fmt.Errorf("failed to parse codex mcp payload: %w", err)
		}

		input.SessionID = payload.ThreadID
		input.ToolName = payload.ToolName
		input.ToolUseID = payload.CallID

		input.RawData["thread_id"] = payload.ThreadID
		input.RawData["turn_id"] = payload.TurnID
		input.RawData["call_id"] = payload.CallID
		input.RawData["server"] = payload.Server
		input.RawData["tool_name"] = payload.ToolName
		input.RawData["duration_ms"] = payload.DurationMS
		input.RawData["status"] = payload.Status
		if payload.ErrorMessage != nil {
			input.RawData["error_message"] = *payload.ErrorMessage
		}
		input.RawData["provider_name"] = payload.ProviderName
		input.RawData["model_slug"] = payload.ModelSlug
		if payload.AgentName != nil {
			input.RawData["agent_name"] = *payload.AgentName
		}

	default:
		return nil, fmt.Errorf("unsupported codex payload type: %q", envelope.Type)
	}

	return input, nil
}

func readCodexHookPayload(reader io.Reader) ([]byte, error) {
	stdinData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}
	trimmedStdin := bytes.TrimSpace(stdinData)
	if len(trimmedStdin) > 0 {
		return trimmedStdin, nil
	}

	if len(os.Args) > 0 {
		lastArg := bytes.TrimSpace([]byte(os.Args[len(os.Args)-1]))
		if len(lastArg) > 0 && json.Valid(lastArg) {
			return lastArg, nil
		}
	}

	return nil, errors.New("codex notify payload missing (expected JSON argv argument)")
}

// GetSessionID extracts the session ID from hook input.
func (c *CodexAgent) GetSessionID(input *agent.HookInput) string {
	return input.SessionID
}

// TransformSessionID converts a Codex session ID to an Entire session ID.
// This is an identity function.
func (c *CodexAgent) TransformSessionID(agentSessionID string) string {
	return agentSessionID
}

// ExtractAgentSessionID extracts the Codex session ID from an Entire session ID.
func (c *CodexAgent) ExtractAgentSessionID(entireSessionID string) string {
	return sessionid.ModelSessionID(entireSessionID)
}

// ProtectedDirs returns directories that Codex uses for repo-local config.
func (c *CodexAgent) ProtectedDirs() []string {
	return []string{".codex"}
}

// GetSessionDir returns where Codex commonly stores session data.
func (c *CodexAgent) GetSessionDir(_ string) (string, error) {
	if override := os.Getenv("ENTIRE_TEST_CODEX_SESSION_DIR"); override != "" {
		return override, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".codex", "sessions"), nil
}

// ResolveSessionFile returns the path to a Codex session log.
func (c *CodexAgent) ResolveSessionFile(sessionDir, agentSessionID string) string {
	return filepath.Join(sessionDir, agentSessionID+".jsonl")
}

// ReadSession reads a session file if SessionRef is provided.
func (c *CodexAgent) ReadSession(input *agent.HookInput) (*agent.AgentSession, error) {
	if input.SessionRef == "" {
		return nil, errors.New("session reference (transcript path) is required")
	}

	data, err := os.ReadFile(input.SessionRef)
	if err != nil {
		return nil, fmt.Errorf("failed to read transcript: %w", err)
	}

	return &agent.AgentSession{
		SessionID:     input.SessionID,
		AgentName:     c.Name(),
		SessionRef:    input.SessionRef,
		StartTime:     time.Now(),
		NativeData:    data,
		ModifiedFiles: nil,
	}, nil
}

// WriteSession writes a session file if SessionRef and NativeData are provided.
func (c *CodexAgent) WriteSession(session *agent.AgentSession) error {
	if session == nil {
		return errors.New("session is nil")
	}
	if session.SessionRef == "" {
		return errors.New("session reference (transcript path) is required")
	}
	if len(session.NativeData) == 0 {
		return errors.New("session has no native data to write")
	}

	if err := os.MkdirAll(filepath.Dir(session.SessionRef), 0o750); err != nil {
		return fmt.Errorf("failed to create transcript directory: %w", err)
	}
	if err := os.WriteFile(session.SessionRef, session.NativeData, 0o600); err != nil {
		return fmt.Errorf("failed to write transcript: %w", err)
	}
	return nil
}

// FormatResumeCommand returns the command to resume a Codex session.
func (c *CodexAgent) FormatResumeCommand(sessionID string) string {
	return "codex resume " + sessionID
}

func codexConfigPath() string {
	if override := os.Getenv("ENTIRE_TEST_CODEX_CONFIG_PATH"); override != "" {
		return override
	}

	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		return defaultCodexConfigRelativePath
	}
	return filepath.Join(homeDir, defaultCodexConfigRelativePath)
}
