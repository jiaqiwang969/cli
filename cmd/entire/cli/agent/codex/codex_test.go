package codex

import (
	"os"
	"strings"
	"testing"

	"github.com/entireio/cli/cmd/entire/cli/agent"
)

func TestCodexAgentMetadata(t *testing.T) {
	t.Parallel()

	ag := &CodexAgent{}
	if ag.Name() != agent.AgentNameCodex {
		t.Fatalf("Name() = %q, want %q", ag.Name(), agent.AgentNameCodex)
	}
	if ag.Type() != agent.AgentTypeCodex {
		t.Fatalf("Type() = %q, want %q", ag.Type(), agent.AgentTypeCodex)
	}
	if !ag.SupportsHooks() {
		t.Fatal("SupportsHooks() = false, want true")
	}
	if got := ag.FormatResumeCommand("thread-1"); got != "codex resume thread-1" {
		t.Fatalf("FormatResumeCommand() = %q, want %q", got, "codex resume thread-1")
	}
}

func TestParseHookInput_AgentTurnCompleteFromArgv(t *testing.T) {
	oldArgs := make([]string, len(os.Args))
	copy(oldArgs, os.Args)
	os.Args = []string{
		"entire",
		"hooks",
		"codex",
		"notify",
		`{"type":"agent-turn-complete","thread-id":"thread-1","turn-id":"turn-1","cwd":"/repo","input-messages":["do A","do B"],"last-assistant-message":"done","provider-name":"Gemini","model-slug":"gemini-3-flash-preview"}`,
	}
	t.Cleanup(func() { os.Args = oldArgs })

	ag := &CodexAgent{}
	input, err := ag.ParseHookInput(agent.HookStop, strings.NewReader(""))
	if err != nil {
		t.Fatalf("ParseHookInput() error = %v", err)
	}

	if input.SessionID != "thread-1" {
		t.Fatalf("SessionID = %q, want %q", input.SessionID, "thread-1")
	}
	if input.UserPrompt != "do B" {
		t.Fatalf("UserPrompt = %q, want %q", input.UserPrompt, "do B")
	}
	if input.RawData["payload_type"] != PayloadTypeAgentTurnComplete {
		t.Fatalf("payload_type = %v, want %q", input.RawData["payload_type"], PayloadTypeAgentTurnComplete)
	}
}

func TestParseHookInput_McpToolCallFromArgv(t *testing.T) {
	oldArgs := make([]string, len(os.Args))
	copy(oldArgs, os.Args)
	os.Args = []string{
		"entire",
		"hooks",
		"codex",
		"notify",
		`{"type":"mcp-tool-call-complete","thread-id":"thread-1","turn-id":"turn-1","call-id":"call-1","server":"claude-code","tool-name":"claude_code","duration-ms":25,"status":"ok","provider-name":"OpenAI","model-slug":"gpt-5","agent-name":"claude-code"}`,
	}
	t.Cleanup(func() { os.Args = oldArgs })

	ag := &CodexAgent{}
	input, err := ag.ParseHookInput(agent.HookPostToolUse, strings.NewReader(""))
	if err != nil {
		t.Fatalf("ParseHookInput() error = %v", err)
	}

	if input.SessionID != "thread-1" {
		t.Fatalf("SessionID = %q, want %q", input.SessionID, "thread-1")
	}
	if input.ToolName != "claude_code" {
		t.Fatalf("ToolName = %q, want %q", input.ToolName, "claude_code")
	}
	if input.ToolUseID != "call-1" {
		t.Fatalf("ToolUseID = %q, want %q", input.ToolUseID, "call-1")
	}
	if input.RawData["payload_type"] != PayloadTypeMcpToolCallComplete {
		t.Fatalf("payload_type = %v, want %q", input.RawData["payload_type"], PayloadTypeMcpToolCallComplete)
	}
}

func TestParseHookInput_MissingPayload(t *testing.T) {
	oldArgs := make([]string, len(os.Args))
	copy(oldArgs, os.Args)
	os.Args = []string{"entire", "hooks", "codex", "notify"}
	t.Cleanup(func() { os.Args = oldArgs })

	ag := &CodexAgent{}
	if _, err := ag.ParseHookInput(agent.HookStop, strings.NewReader("")); err == nil {
		t.Fatal("ParseHookInput() expected error, got nil")
	}
}
