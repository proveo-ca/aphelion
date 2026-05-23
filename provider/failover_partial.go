//go:build linux

package provider

import (
	"errors"
	"strings"

	"github.com/idolum-ai/aphelion/agent"
)

type partialProviderError interface {
	PartialProviderResponse() *agent.Response
	PartialProviderResponseID() string
	PartialProviderReason() string
}

func appendPartialProviderRecoveryMessage(messages []agent.Message, provider string, err error) []agent.Message {
	partial, responseID, reason, ok := partialProviderSnapshot(err)
	if !ok {
		return messages
	}
	note := renderPartialProviderRecoveryNote(provider, responseID, reason, partial)
	if strings.TrimSpace(note) == "" {
		return messages
	}
	out := append([]agent.Message(nil), messages...)
	out = append(out, agent.Message{
		Role:    "user",
		Content: note,
	})
	return out
}

func partialProviderSnapshot(err error) (*agent.Response, string, string, bool) {
	if err == nil {
		return nil, "", "", false
	}
	var partialErr partialProviderError
	if !errors.As(err, &partialErr) {
		return nil, "", "", false
	}
	partial := partialErr.PartialProviderResponse()
	responseID := strings.TrimSpace(partialErr.PartialProviderResponseID())
	reason := strings.TrimSpace(partialErr.PartialProviderReason())
	if partial == nil && responseID == "" && reason == "" {
		return nil, "", "", false
	}
	return partial, responseID, reason, true
}

func renderPartialProviderRecoveryNote(provider string, responseID string, reason string, partial *agent.Response) string {
	var b strings.Builder
	b.WriteString("Provider recovery note: ")
	b.WriteString(firstNonEmpty(strings.TrimSpace(provider), "primary provider"))
	b.WriteString(" produced an incomplete response before failing. Treat this as partial, non-authoritative evidence while completing the user's request.")
	if reason = strings.TrimSpace(reason); reason != "" {
		b.WriteString("\nreason: ")
		b.WriteString(reason)
	}
	if responseID = strings.TrimSpace(responseID); responseID != "" {
		b.WriteString("\nresponse_id: ")
		b.WriteString(responseID)
	}
	if partial == nil {
		return b.String()
	}
	if text := strings.TrimSpace(partial.Content); text != "" {
		b.WriteString("\npartial_text:\n")
		b.WriteString(agent.CompactProviderContextText(text, providerFallbackRecentToolChars))
	}
	if len(partial.ToolCalls) > 0 {
		b.WriteString("\npartial_tool_calls:")
		for _, call := range partial.ToolCalls {
			b.WriteString("\n- ")
			b.WriteString(firstNonEmpty(strings.TrimSpace(call.Name), "tool"))
			if id := strings.TrimSpace(call.ID); id != "" {
				b.WriteString(" id=")
				b.WriteString(id)
			}
			if input := strings.TrimSpace(string(call.Input)); input != "" {
				b.WriteString(" input=")
				b.WriteString(agent.CompactProviderContextText(input, providerFallbackOlderToolChars))
			}
		}
	}
	return b.String()
}

const (
	providerFallbackRecentToolChars = 4000
	providerFallbackOlderToolChars  = 800
)

func compactToolResultMessagesForProviderFallback(messages []agent.Message) []agent.Message {
	return agent.CompactToolResultMessagesForProviderContext(messages)
}
