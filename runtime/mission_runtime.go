//go:build linux

package runtime

import (
	"context"

	"github.com/idolum-ai/aphelion/core"
	"github.com/idolum-ai/aphelion/runtime/mission"
	"github.com/idolum-ai/aphelion/session"
	"github.com/idolum-ai/aphelion/turn"
)

const hiddenInputMissionAsk = mission.HiddenInputMissionAsk

func FormatMissionControlProposalMessage(proposal core.MissionControlProposal) string {
	return mission.FormatMissionControlProposalMessage(proposal)
}

func MissionControlProposalMetadataJSON(proposal core.MissionControlProposal) (string, error) {
	return core.MissionControlProposalMetadataJSON(proposal)
}

func (r *Runtime) missionRuntime() *mission.Runtime {
	deps := mission.Dependencies{
		Store:                            r.store,
		Resolver:                         r.resolver,
		Outbound:                         r.outbound,
		ModelSlotProvider:                r.modelSlotProvider,
		RecordExecutionEvent:             r.recordExecutionEvent,
		OperationArtifactRequestUserText: operationArtifactRequestUserText,
	}
	deps.PrefixTelegramText = func(msg core.InboundMessage, text string) string {
		return r.prefixTelegramPresentedText(r.telegramPresentationForMessage(msg), text)
	}
	return mission.NewRuntime(deps)
}

func (r *Runtime) recordWorkingObjectiveForInbound(key session.SessionKey, msg core.InboundMessage) {
	r.missionRuntime().RecordWorkingObjectiveForInbound(key, msg)
}

func (r *Runtime) maybeOfferMissionAsk(ctx context.Context, key session.SessionKey, msg core.InboundMessage, ledgerText string, result *turn.Result) error {
	return r.missionRuntime().MaybeOfferMissionAsk(ctx, key, msg, ledgerText, result)
}

func (r *Runtime) MissionCommand(ctx context.Context, chatID int64, senderID int64, args string) (string, error) {
	return r.missionRuntime().MissionCommand(ctx, chatID, senderID, args)
}

func (r *Runtime) MissionHome(ctx context.Context, chatID int64, senderID int64) ([]session.MissionState, session.WorkingObjective, bool, error) {
	return r.missionRuntime().MissionHome(ctx, chatID, senderID)
}

func (r *Runtime) MissionDetails(ctx context.Context, chatID int64, senderID int64, missionID string) (session.MissionState, []session.MissionEvent, error) {
	return r.missionRuntime().MissionDetails(ctx, chatID, senderID, missionID)
}

func (r *Runtime) SetMissionPinned(ctx context.Context, chatID int64, senderID int64, missionID string, pinned bool) (session.MissionState, error) {
	return r.missionRuntime().SetMissionPinned(ctx, chatID, senderID, missionID, pinned)
}

func (r *Runtime) UpdateMissionStatus(ctx context.Context, chatID int64, senderID int64, missionID string, status session.MissionStatus) (session.MissionState, error) {
	return r.missionRuntime().UpdateMissionStatus(ctx, chatID, senderID, missionID, status)
}

func (r *Runtime) MissionLedgerHealth(ctx context.Context, senderID int64) (session.MissionLedgerHealth, error) {
	return r.missionRuntime().MissionLedgerHealth(ctx, senderID)
}

func (r *Runtime) MissionAskPrompt(ctx context.Context, senderID int64, promptID string) (session.MissionAskPrompt, bool, error) {
	return r.missionRuntime().MissionAskPrompt(ctx, senderID, promptID)
}

func (r *Runtime) ResolveMissionAskPrompt(ctx context.Context, senderID int64, promptID string, status session.MissionAskStatus, summary string) (session.MissionAskPrompt, error) {
	return r.missionRuntime().ResolveMissionAskPrompt(ctx, senderID, promptID, status, summary)
}
