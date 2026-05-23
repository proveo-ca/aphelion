//go:build linux

package core

import (
	"context"
	"errors"
	"net"
	"strings"
)

const (
	ProviderFailureTransportTimeout     = "transport_timeout"
	ProviderFailureTransportDNS         = "transport_dns"
	ProviderFailureTransportInterrupted = "transport_interrupted"
	ProviderFailureProvider5xx          = "provider_5xx"
	ProviderFailureRateLimit            = "rate_limit"
	ProviderFailureContextWindow        = "context_window"
	ProviderFailureRequestBuffer        = "request_buffer"
	ProviderFailureContinuationRejected = "continuation_rejected"
	ProviderFailureContextBudget        = "context_budget_exceeded"
	ProviderFailureAuth                 = "auth"
	ProviderFailureRequest              = "request"
	ProviderFailureCanceled             = "canceled"
	ProviderFailureUnknown              = "provider_failure"
)

type providerFailureStatusCoder interface {
	StatusCode() int
}

type providerFailureCoder interface {
	ProviderFailureCode() string
}

func ProviderFailureKind(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) {
		return ProviderFailureCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ProviderFailureTransportTimeout
	}
	if isProviderNetTimeout(err) {
		return ProviderFailureTransportTimeout
	}
	if code := providerFailureCode(err); code != "" {
		switch code {
		case "rate_limit_exceeded":
			return ProviderFailureRateLimit
		case "server_is_overloaded", "slow_down":
			return ProviderFailureProvider5xx
		case "context_length_exceeded":
			return ProviderFailureContextWindow
		case "invalid_prompt":
			return ProviderFailureRequest
		}
	}
	var sc providerFailureStatusCoder
	if errors.As(err, &sc) {
		switch code := sc.StatusCode(); {
		case code == 429:
			return ProviderFailureRateLimit
		case code == 401 || code == 403:
			return ProviderFailureAuth
		case code >= 500 && code < 600:
			return ProviderFailureProvider5xx
		case code >= 400 && code < 500:
			return ProviderFailureRequest
		}
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case msg == "":
		return ProviderFailureUnknown
	case strings.Contains(msg, "context_budget_exceeded"):
		return ProviderFailureContextBudget
	case containsAny(msg, "request buffer", "response buffer", "buffer limit"):
		return ProviderFailureRequestBuffer
	case containsAny(msg, "context window", "context length", "context_length_exceeded", "maximum context", "too many tokens", "input exceeds", "exceeds the context", "token limit"):
		return ProviderFailureContextWindow
	case containsAny(msg, "stored-response continuation rejected", "incomplete response without stored-response continuation", "incomplete response missing response id", "response remained incomplete"):
		return ProviderFailureContinuationRejected
	case containsAny(msg, "timeout awaiting response headers", "client.timeout exceeded while awaiting headers", "context deadline exceeded", "i/o timeout", "tls handshake timeout"):
		return ProviderFailureTransportTimeout
	case containsAny(msg, "no such host", "lookup ", "dns", "temporary failure in name resolution"):
		return ProviderFailureTransportDNS
	case containsAny(msg, "unexpected eof", "connection reset", "broken pipe", "stream closed", "stream terminated", "incomplete event stream"):
		return ProviderFailureTransportInterrupted
	case containsAny(msg, "rate_limit_exceeded", "rate limit", "rate-limit"):
		return ProviderFailureRateLimit
	case containsAny(msg, "server_is_overloaded", "slow_down", "slow down", "overload", "at capacity") || strings.Contains(msg, " 500") || strings.Contains(msg, " 503"):
		return ProviderFailureProvider5xx
	default:
		return ProviderFailureUnknown
	}
}

func ProviderFailureRetryable(kind string) bool {
	switch strings.TrimSpace(kind) {
	case ProviderFailureTransportTimeout,
		ProviderFailureTransportDNS,
		ProviderFailureTransportInterrupted,
		ProviderFailureProvider5xx,
		ProviderFailureRateLimit:
		return true
	default:
		return false
	}
}

func ProviderFailureFailoverEligible(kind string) bool {
	switch strings.TrimSpace(kind) {
	case ProviderFailureTransportTimeout,
		ProviderFailureTransportDNS,
		ProviderFailureTransportInterrupted,
		ProviderFailureProvider5xx,
		ProviderFailureRateLimit,
		ProviderFailureContextWindow,
		ProviderFailureRequestBuffer,
		ProviderFailureContinuationRejected:
		return true
	default:
		return false
	}
}

func providerFailureCode(err error) string {
	var coded providerFailureCoder
	if errors.As(err, &coded) {
		return strings.ToLower(strings.TrimSpace(coded.ProviderFailureCode()))
	}
	return ""
}

func isProviderNetTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func containsAny(text string, markers ...string) bool {
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
