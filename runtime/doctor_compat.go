//go:build linux

package runtime

import (
	"context"
	"strings"
	"time"

	"github.com/idolum-ai/aphelion/pipeline"
	"github.com/idolum-ai/aphelion/runtime/doctor"
	"github.com/idolum-ai/aphelion/session"
	"github.com/idolum-ai/aphelion/tool/sandbox"
)

type DiagnosticInput = doctor.DiagnosticInput

const RequestMarker = doctor.RequestMarker
const doctorSummaryMarker = doctor.SummaryMarker

func (r *Runtime) writeDoctorCodexWorkEvidenceReview(ctx context.Context, b *strings.Builder, input DiagnosticInput) {
	r.doctorRuntime().WriteCodexWorkEvidenceReview(ctx, b, input)
}

func (r *Runtime) writeDoctorRuntimeConfig(b *strings.Builder, exec pipeline.TurnExecutionContract, scope sandbox.Scope) {
	r.doctorRuntime().WriteRuntimeConfig(b, exec, scope)
}

func (r *Runtime) writeDoctorAutonomyStatus(b *strings.Builder, key session.SessionKey, senderID int64, now time.Time) {
	r.doctorRuntime().WriteAutonomyStatus(b, key, senderID, now)
}

func (r *Runtime) writeDoctorSandboxReadiness(b *strings.Builder, now time.Time) {
	r.doctorRuntime().WriteSandboxReadiness(b, now)
}

func (r *Runtime) writeDoctorIssueStatusChecks(b *strings.Builder, input DiagnosticInput) {
	r.doctorRuntime().WriteIssueStatusChecks(b, input)
}

func (r *Runtime) writeDoctorDesignPrincipleHealth(b *strings.Builder, input DiagnosticInput) {
	r.doctorRuntime().WriteDesignPrincipleHealth(b, input)
}

func (r *Runtime) writeDoctorExternalToolInvocationReadiness(b *strings.Builder, input DiagnosticInput) {
	r.doctorRuntime().WriteExternalToolInvocationReadiness(b, input)
}

func (r *Runtime) writeDoctorRuntimeAdjudications(ctx context.Context, b *strings.Builder, key session.SessionKey, now time.Time) {
	r.doctorRuntime().WriteRuntimeAdjudications(ctx, b, key, now)
}

func (r *Runtime) writeDoctorMissionLedger(b *strings.Builder, key session.SessionKey, now time.Time) {
	r.doctorRuntime().WriteMissionLedger(b, key, now)
}

func (r *Runtime) writeDoctorTelegramThreads(b *strings.Builder, key session.SessionKey) {
	r.doctorRuntime().WriteTelegramThreads(b, key)
}
