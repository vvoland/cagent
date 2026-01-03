package hooks

import (
	"github.com/docker/cagent/pkg/config/latest"
)

// FromConfig converts a latest.HooksConfig to a hooks.Config
func FromConfig(cfg *latest.HooksConfig) *Config {
	if cfg == nil {
		return nil
	}

	result := &Config{}

	// Convert PreToolUse
	for _, matcher := range cfg.PreToolUse {
		mc := MatcherConfig{
			Matcher: matcher.Matcher,
			Hooks:   make([]Hook, 0, len(matcher.Hooks)),
		}
		for _, h := range matcher.Hooks {
			mc.Hooks = append(mc.Hooks, Hook{
				Type:    HookType(h.Type),
				Command: h.Command,
				Timeout: h.Timeout,
			})
		}
		result.PreToolUse = append(result.PreToolUse, mc)
	}

	// Convert PostToolUse
	for _, matcher := range cfg.PostToolUse {
		mc := MatcherConfig{
			Matcher: matcher.Matcher,
			Hooks:   make([]Hook, 0, len(matcher.Hooks)),
		}
		for _, h := range matcher.Hooks {
			mc.Hooks = append(mc.Hooks, Hook{
				Type:    HookType(h.Type),
				Command: h.Command,
				Timeout: h.Timeout,
			})
		}
		result.PostToolUse = append(result.PostToolUse, mc)
	}

	// Convert SessionStart
	for _, h := range cfg.SessionStart {
		result.SessionStart = append(result.SessionStart, Hook{
			Type:    HookType(h.Type),
			Command: h.Command,
			Timeout: h.Timeout,
		})
	}

	// Convert SessionEnd
	for _, h := range cfg.SessionEnd {
		result.SessionEnd = append(result.SessionEnd, Hook{
			Type:    HookType(h.Type),
			Command: h.Command,
			Timeout: h.Timeout,
		})
	}

	return result
}
