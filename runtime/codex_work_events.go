//go:build linux

package runtime

import (
	runtimecodex "github.com/idolum-ai/aphelion/runtime/codex"
	"github.com/idolum-ai/aphelion/session"
)

func codexWorkEventFromNotification(method string, params map[string]any) (session.WorkCodexEvent, bool) {
	return runtimecodex.WorkEventFromNotification(method, params)
}

func codexWorkResultFromAppServer(req WorkRequest, threadID string, turnID string, result codexAppServerResult) WorkResult {
	out := runtimecodex.WorkResultFromAppServer(codexWorkRequest(req), threadID, turnID, result)
	return WorkResult(out)
}

func codexWorkPatchPreviewFromEvents(events []session.WorkCodexEvent) string {
	return runtimecodex.WorkPatchPreviewFromEvents(events)
}

func codexWorkChangedFilesFromEvents(events []session.WorkCodexEvent) []string {
	return runtimecodex.WorkChangedFilesFromEvents(events)
}

func codexWorkCommandsFromEvents(events []session.WorkCodexEvent) []string {
	return runtimecodex.WorkCommandsFromEvents(events)
}

func codexWorkCommitLaneStatus(req WorkRequest, events []session.WorkCodexEvent, approvals []codexAppServerApprovalDecision) string {
	return runtimecodex.WorkCommitLaneStatus(codexWorkRequest(req), events, approvals)
}

func codexWorkRequest(req WorkRequest) runtimecodex.WorkRequest {
	return runtimecodex.WorkRequest{
		OperationID: req.OperationID,
		RepoRoot:    req.RepoRoot,
		Workdir:     req.Workdir,
		Prompt:      req.Prompt,
		Mode:        runtimecodex.WorkMode(req.Mode),
		LeaseID:     req.LeaseID,
		State:       req.State,
	}
}
