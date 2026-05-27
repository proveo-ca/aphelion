//go:build linux

package runtime

import (
	"context"
	"time"

	"github.com/idolum-ai/aphelion/core"
	runtimecodex "github.com/idolum-ai/aphelion/runtime/codex"
)

type codexAppServerRequest = runtimecodex.Request
type codexAppServerResult = runtimecodex.Result
type codexAppServerApprovalDecision = runtimecodex.ApprovalDecision
type codexAppServerApprovalHandler = runtimecodex.ApprovalHandler
type codexAppServerStreamOptions = runtimecodex.StreamOptions
type codexAppServerDoer = runtimecodex.Doer
type realCodexAppServerDoer = runtimecodex.RealAppServerDoer
type codexAppServerArtifactManifest = runtimecodex.ArtifactManifest
type codexAppServerArtifactManifestEntry = runtimecodex.ArtifactManifestEntry

const (
	codexAppServerAdapterName     = runtimecodex.AdapterName
	codexAppServerWakeChannel     = runtimecodex.WakeChannel
	codexAppServerMaxMessageBytes = runtimecodex.MaxMessageBytes
)

type codexAppServerClient struct {
	*runtimecodex.Client
}

func newCodexAppServerClient(address string, handlers ...codexAppServerApprovalHandler) *codexAppServerClient {
	return &codexAppServerClient{Client: runtimecodex.NewClient(address, handlers...)}
}

func (c *codexAppServerClient) readMessage(ctx context.Context) (map[string]any, error) {
	if c == nil || c.Client == nil {
		return nil, context.Canceled
	}
	return c.Client.ReadMessage(ctx)
}

func checkCodexAppServerHTTP(ctx context.Context, address string, path string) error {
	return runtimecodex.CheckHTTP(ctx, address, path)
}

func codexAppServerStatusPrompt(agent core.DurableAgent, now time.Time) string {
	return runtimecodex.StatusPrompt(agent, now)
}

func codexAppServerBoolString(value bool) string { return runtimecodex.BoolString(value) }

func loadCodexAppServerArtifactManifest(artifactRoot string, agentID string) (codexAppServerArtifactManifest, error) {
	return runtimecodex.LoadArtifactManifest(artifactRoot, agentID)
}

func upsertCodexAppServerArtifactManifestEntry(manifest codexAppServerArtifactManifest, entry codexAppServerArtifactManifestEntry, updatedAt time.Time) codexAppServerArtifactManifest {
	return runtimecodex.UpsertArtifactManifestEntry(manifest, entry, updatedAt)
}

func writeCodexAppServerArtifactManifest(artifactRoot string, manifest codexAppServerArtifactManifest) error {
	return runtimecodex.WriteArtifactManifest(artifactRoot, manifest)
}

func codexAppServerWakeSummary(agent core.DurableAgent, result codexAppServerResult, artifactRel string) string {
	return runtimecodex.WakeSummary(agent, result, artifactRel)
}

func summarizeCodexApprovalDecisions(values []codexAppServerApprovalDecision) string {
	return runtimecodex.SummarizeApprovalDecisions(values)
}

func asObject(value any) map[string]any { return runtimecodex.AsObject(value) }

func stringField(obj map[string]any, key string) string { return runtimecodex.StringField(obj, key) }

func nestedString(obj map[string]any, keys ...string) string {
	return runtimecodex.NestedString(obj, keys...)
}

func codexAppServerBaseInstructions(agent core.DurableAgent) string {
	return runtimecodex.BaseInstructions(agent)
}

func codexAppServerDeveloperInstructions(agent core.DurableAgent) string {
	return runtimecodex.DeveloperInstructions(agent)
}

func codexAppServerCommandAllowed(command string) bool {
	return runtimecodex.AppServerCommandAllowed(command)
}
