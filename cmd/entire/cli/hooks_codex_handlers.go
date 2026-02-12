// hooks_codex_handlers.go contains Codex CLI specific hook handler implementations.
package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/entireio/cli/cmd/entire/cli/agent"
	"github.com/entireio/cli/cmd/entire/cli/agent/codex"
	"github.com/entireio/cli/cmd/entire/cli/logging"
	"github.com/entireio/cli/cmd/entire/cli/paths"
	"github.com/entireio/cli/cmd/entire/cli/strategy"
)

// handleCodexNotify handles both Codex notify payload types:
// - agent-turn-complete
// - mcp-tool-call-complete
func handleCodexNotify() error {
	ag, err := GetCurrentHookAgent()
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	input, err := ag.ParseHookInput(agent.HookStop, os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to parse hook input: %w", err)
	}

	payloadType, _ := input.RawData["payload_type"].(string)
	switch payloadType {
	case codex.PayloadTypeAgentTurnComplete:
		return handleCodexAfterAgent(ag, input)
	case codex.PayloadTypeMcpToolCallComplete:
		return handleCodexAfterMcpToolCall(ag, input)
	default:
		return fmt.Errorf("unsupported codex payload type: %q", payloadType)
	}
}

func handleCodexAfterAgent(ag agent.Agent, input *agent.HookInput) error {
	logCtx := logging.WithAgent(logging.WithComponent(context.Background(), "hooks"), ag.Name())
	logging.Info(logCtx, "codex-after-agent",
		slog.String("hook", codex.HookNameNotify),
		slog.String("hook_type", "agent"),
		slog.String("model_session_id", input.SessionID),
		slog.String("provider_name", rawString(input.RawData, "provider_name")),
		slog.String("model_slug", rawString(input.RawData, "model_slug")),
	)

	if input.SessionID == "" {
		return fmt.Errorf("missing thread-id in codex payload")
	}

	// Bail out quickly in empty repositories (same behavior as other handlers).
	if repo, err := strategy.OpenRepository(); err == nil && strategy.IsEmptyRepository(repo) {
		fmt.Fprintln(os.Stderr, "Entire: skipping checkpoint. Will activate after first commit.")
		return NewSilentError(strategy.ErrEmptyRepository)
	}

	prompts := rawStringSlice(input.RawData, "input_messages")
	lastPrompt := ""
	if len(prompts) > 0 {
		lastPrompt = prompts[len(prompts)-1]
	}

	commitMessage := "Codex session updates"
	if lastPrompt != "" {
		commitMessage = generateCommitMessage(lastPrompt)
	}

	if err := saveCodexCheckpoint(input.SessionID, prompts, rawString(input.RawData, "last_assistant_message"), commitMessage, ag.Type()); err != nil {
		return err
	}

	// Best-effort phase transition for strategies that track turn boundaries.
	transitionSessionTurnEnd(input.SessionID)
	return nil
}

func handleCodexAfterMcpToolCall(ag agent.Agent, input *agent.HookInput) error {
	sessionID := input.SessionID
	status := rawString(input.RawData, "status")
	agentName := rawString(input.RawData, "agent_name")
	server := rawString(input.RawData, "server")
	toolName := rawString(input.RawData, "tool_name")

	logCtx := logging.WithAgent(logging.WithComponent(context.Background(), "hooks"), ag.Name())
	logging.Debug(logCtx, "codex-after-mcp-tool-call",
		slog.String("hook", codex.HookNameNotify),
		slog.String("hook_type", "tool"),
		slog.String("model_session_id", sessionID),
		slog.String("agent_name", agentName),
		slog.String("server", server),
		slog.String("tool_name", toolName),
		slog.String("status", status),
	)

	// Only checkpoint successful/handled subagent calls.
	if sessionID == "" || agentName == "" || (status != "ok" && status != "tool-error") {
		return nil
	}

	summary := "MCP tool call completed."
	if errMsg := rawString(input.RawData, "error_message"); errMsg != "" {
		summary = "MCP tool call completed with message: " + errMsg
	}

	prompt := fmt.Sprintf("Codex MCP %s: %s/%s (%s)", agentName, server, toolName, status)
	commitMessage := generateCommitMessage(prompt)

	return saveCodexCheckpoint(sessionID, []string{prompt}, summary, commitMessage, ag.Type())
}

func saveCodexCheckpoint(sessionID string, prompts []string, summary string, commitMessage string, agentType agent.AgentType) error {
	repoRoot, err := paths.RepoRoot()
	if err != nil {
		return fmt.Errorf("failed to get repo root: %w", err)
	}

	changes, err := DetectFileChanges(nil)
	if err != nil {
		return fmt.Errorf("failed to detect file changes: %w", err)
	}

	relModifiedFiles := FilterAndNormalizePaths(changes.Modified, repoRoot)
	relNewFiles := FilterAndNormalizePaths(changes.New, repoRoot)
	relDeletedFiles := FilterAndNormalizePaths(changes.Deleted, repoRoot)

	totalChanges := len(relModifiedFiles) + len(relNewFiles) + len(relDeletedFiles)
	if totalChanges == 0 {
		fmt.Fprintln(os.Stderr, "No files were modified during this event")
		fmt.Fprintln(os.Stderr, "Skipping commit")
		return nil
	}
	logFileChanges(relModifiedFiles, relNewFiles, relDeletedFiles)

	sessionDir := paths.SessionMetadataDirFromSessionID(sessionID)
	sessionDirAbs, err := paths.AbsPath(sessionDir)
	if err != nil {
		sessionDirAbs = sessionDir
	}
	if err := os.MkdirAll(sessionDirAbs, 0o750); err != nil {
		return fmt.Errorf("failed to create session metadata directory: %w", err)
	}

	promptFile := filepath.Join(sessionDirAbs, paths.PromptFileName)
	if err := os.WriteFile(promptFile, []byte(strings.Join(prompts, "\n\n---\n\n")), 0o600); err != nil {
		return fmt.Errorf("failed to write prompt file: %w", err)
	}

	summaryFile := filepath.Join(sessionDirAbs, paths.SummaryFileName)
	if err := os.WriteFile(summaryFile, []byte(summary), 0o600); err != nil {
		return fmt.Errorf("failed to write summary file: %w", err)
	}

	contextFile := filepath.Join(sessionDirAbs, paths.ContextFileName)
	if err := createContextFileForGemini(contextFile, commitMessage, sessionID, prompts, summary); err != nil {
		return fmt.Errorf("failed to create context file: %w", err)
	}

	author, err := GetGitAuthor()
	if err != nil {
		return fmt.Errorf("failed to get git author: %w", err)
	}

	strat := GetStrategy()
	if err := strat.EnsureSetup(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to ensure strategy setup: %v\n", err)
	}

	saveCtx := strategy.SaveContext{
		SessionID:                sessionID,
		ModifiedFiles:            relModifiedFiles,
		NewFiles:                 relNewFiles,
		DeletedFiles:             relDeletedFiles,
		MetadataDir:              sessionDir,
		MetadataDirAbs:           sessionDirAbs,
		CommitMessage:            commitMessage,
		TranscriptPath:           "",
		AuthorName:               author.Name,
		AuthorEmail:              author.Email,
		AgentType:                agentType,
		StepTranscriptStart:      0,
		StepTranscriptIdentifier: "",
		TokenUsage:               nil,
	}

	if err := strat.SaveChanges(saveCtx); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Session saved successfully")
	return nil
}

func rawString(raw map[string]interface{}, key string) string {
	value, ok := raw[key]
	if !ok {
		return ""
	}
	str, _ := value.(string)
	return str
}

func rawStringSlice(raw map[string]interface{}, key string) []string {
	value, ok := raw[key]
	if !ok {
		return nil
	}
	items, ok := value.([]string)
	if ok {
		return items
	}

	values, ok := value.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, item := range values {
		str, ok := item.(string)
		if ok {
			result = append(result, str)
		}
	}
	return result
}
