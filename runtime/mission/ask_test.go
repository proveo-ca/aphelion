//go:build linux

package mission

import (
	"strings"
	"testing"

	"github.com/idolum-ai/aphelion/session"
)

func TestMissionAskClassifierMessagesUseProductionContract(t *testing.T) {
	t.Parallel()

	observation := missionAskObservation{
		Query:       "Please review the README docs again.",
		MissionID:   "mission-readme",
		MissionName: "README cleanup",
		Question:    "Should this stay with README cleanup?",
		Confidence:  session.MissionAskConfidenceLow,
	}
	messages := missionAskClassifierMessages(observation)
	if len(messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(messages))
	}
	if messages[0].Role != "system" || messages[1].Role != "user" {
		t.Fatalf("message roles = %q/%q, want system/user", messages[0].Role, messages[1].Role)
	}
	system := messages[0].Content
	for _, want := range []string{
		"## Role",
		"## Goal",
		"## Success Criteria",
		"## Output",
		"## Confidence",
		"## Stop Rules",
		"Return JSON only",
		"same_objective|new_objective|ignore|unclear",
		"mission_id may be copied from the supplied candidate only",
		"include a non-empty compact question",
		"When a question would feel like pestering, choose ignore.",
	} {
		if !strings.Contains(system, want) {
			t.Fatalf("classifier system prompt missing %q: %q", want, system)
		}
	}
	user := messages[1].Content
	for _, want := range []string{
		"candidate_mission_id=mission-readme",
		"candidate_name=README cleanup",
		"local_confidence=low",
		"query=Please review the README docs again.",
		"question=Should this stay with README cleanup?",
	} {
		if !strings.Contains(user, want) {
			t.Fatalf("classifier user prompt missing %q: %q", want, user)
		}
	}
}

func TestExtractMissionAskJSONTrimsWrappedObject(t *testing.T) {
	t.Parallel()

	got := extractMissionAskJSON("```json\n{\"action\":\"ignore\",\"confidence\":\"low\"}\n```")
	if got != "{\"action\":\"ignore\",\"confidence\":\"low\"}" {
		t.Fatalf("extractMissionAskJSON() = %q", got)
	}
}
