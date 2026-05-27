//go:build linux

package doctor

import (
	"context"
	"strings"
	"time"

	"github.com/idolum-ai/aphelion/pipeline"
	"github.com/idolum-ai/aphelion/session"
	"github.com/idolum-ai/aphelion/tool/sandbox"
)

func (r *Runtime) WriteRuntimeConfig(b *strings.Builder, exec pipeline.TurnExecutionContract, scope sandbox.Scope) {
	r.writeDoctorRuntimeConfig(b, exec, scope)
}

func (r *Runtime) WriteAutonomyStatus(b *strings.Builder, key session.SessionKey, senderID int64, now time.Time) {
	r.writeDoctorAutonomyStatus(b, key, senderID, now)
}

func (r *Runtime) WriteSandboxReadiness(b *strings.Builder, now time.Time) {
	r.writeDoctorSandboxReadiness(b, now)
}

func (r *Runtime) WriteIssueStatusChecks(b *strings.Builder, input DiagnosticInput) {
	r.writeDoctorIssueStatusChecks(b, input)
}

func (r *Runtime) WriteDesignPrincipleHealth(b *strings.Builder, input DiagnosticInput) {
	r.writeDoctorDesignPrincipleHealth(b, input)
}

func (r *Runtime) WriteExternalToolInvocationReadiness(b *strings.Builder, input DiagnosticInput) {
	r.writeDoctorExternalToolInvocationReadiness(b, input)
}

func (r *Runtime) WriteRuntimeAdjudications(ctx context.Context, b *strings.Builder, key session.SessionKey, now time.Time) {
	r.writeDoctorRuntimeAdjudications(ctx, b, key, now)
}

func (r *Runtime) WriteMissionLedger(b *strings.Builder, key session.SessionKey, now time.Time) {
	r.writeDoctorMissionLedger(b, key, now)
}

func (r *Runtime) WriteTelegramThreads(b *strings.Builder, key session.SessionKey) {
	r.writeDoctorTelegramThreads(b, key)
}
