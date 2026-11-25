package assistant

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Backend is a chat-only assistant: it answers questions and keeps the
// conversation, but can never run tools, read files or change the machine.
type Backend interface {
	// Ask sends one turn; text is streamed to onDelta as it arrives.
	Ask(ctx context.Context, prompt string, onDelta func(string)) (string, error)
	// Name is what the panel calls it: "Claude", "OpenCode (ollama/qwen3)".
	Name() string
}

// Providers lists the valid Config.Provider values ("" = auto).
var Providers = []string{"claude", "opencode", "openai"}

// Config selects and configures a backend (the `assistant:` config block).
type Config struct {
	// Provider: claude (Claude Code CLI), opencode (OpenCode CLI, any of its
	// providers), openai (any OpenAI-compatible HTTP API: Ollama, LM Studio,
	// vLLM, OpenRouter, OpenAI…). Empty: claude if installed, else opencode.
	Provider string `yaml:"provider,omitempty"`
	// Model: claude → --model (e.g. claude-sonnet-5-5); opencode →
	// provider/model (e.g. ollama/qwen3:14b); openai → the API's model id.
	Model string `yaml:"model,omitempty"`
	// Command overrides the CLI path (claude or opencode).
	Command string `yaml:"command,omitempty"`
	// BaseURL is the openai endpoint, e.g. http://localhost:11434/v1 (Ollama).
	BaseURL string `yaml:"base_url,omitempty"`
	// APIKeyEnv names the environment variable holding the openai API key
	// (keys never go in config files). Empty: no Authorization header.
	APIKeyEnv string `yaml:"api_key_env,omitempty"`
}

// New builds the configured backend. system is the system prompt; dir is a
// private working directory (never a project or engagement directory, since
// some CLIs read instructions from it).
func New(c Config, system, dir string) (Backend, error) {
	p := strings.ToLower(strings.TrimSpace(c.Provider))
	if p == "" {
		switch {
		case c.Command != "" || Find() != "":
			p = "claude"
		case findOpenCode() != "":
			p = "opencode"
		default:
			return nil, fmt.Errorf("no assistant found: install Claude Code or OpenCode, or set assistant.provider: openai with base_url")
		}
	}
	switch p {
	case "claude":
		bin := c.Command
		if bin == "" {
			bin = Find()
		}
		if bin == "" {
			return nil, fmt.Errorf("Claude Code CLI (claude) not found on PATH")
		}
		return &Assistant{Bin: bin, Dir: dir, Model: c.Model, System: system}, nil
	case "opencode":
		bin := c.Command
		if bin == "" {
			bin = findOpenCode()
		}
		if bin == "" {
			return nil, fmt.Errorf("OpenCode CLI (opencode) not found on PATH")
		}
		return &OpenCode{Bin: bin, Dir: dir, Model: c.Model, System: system}, nil
	case "openai":
		if c.BaseURL == "" || c.Model == "" {
			return nil, fmt.Errorf("assistant.provider openai needs base_url and model")
		}
		key := ""
		if c.APIKeyEnv != "" {
			if key = os.Getenv(c.APIKeyEnv); key == "" {
				return nil, fmt.Errorf("%s is not set (assistant.api_key_env)", c.APIKeyEnv)
			}
		}
		return &OpenAI{BaseURL: c.BaseURL, Model: c.Model, APIKey: key, System: system}, nil
	}
	return nil, fmt.Errorf("assistant.provider must be one of %v, not %q", Providers, c.Provider)
}

func findOpenCode() string {
	p, err := exec.LookPath("opencode")
	if err != nil {
		return ""
	}
	return p
}

// Name implements Backend for the Claude Code CLI.
func (a *Assistant) Name() string {
	if a.Model != "" {
		return "Claude (" + a.Model + ")"
	}
	return "Claude"
}
