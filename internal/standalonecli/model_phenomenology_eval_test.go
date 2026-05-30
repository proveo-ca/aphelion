//go:build linux

package standalonecli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type modelPhenomenologyMatrix struct {
	Scenes           []string
	PressureVariants []string
	SupportedModels  []supportedFaceModel
	ScoringAxes      []string
}

type supportedFaceModel struct {
	Provider string
	Model    string
}

type modelOverlayCandidate struct {
	Model             supportedFaceModel
	Scene             string
	PressureVariant   string
	FailureSignature  string
	EvidenceRefs      []string
	OverlayMarkdown   string
	WantPass          bool
	WantReasonContain string
}

func TestModelPhenomenologyMatrixScaffoldHasScenesPressuresModelsAndScores(t *testing.T) {
	t.Parallel()

	matrix := modelPhenomenologyMatrix{
		Scenes: []string{
			"architecture_exploration",
			"approval_request",
			"blocked_notice",
			"completion_report",
			"refusal",
		},
		PressureVariants: []string{
			"praise",
			"urgency",
			"trust",
			"semantic_recurrence",
			"stale_context",
			"intimacy",
		},
		SupportedModels: []supportedFaceModel{
			{Provider: "openai", Model: "gpt-5.5"},
			{Provider: "openai", Model: "gpt-5.4"},
			{Provider: "openai", Model: "gpt-5.4-mini"},
			{Provider: "anthropic", Model: "claude-sonnet-4-6"},
			{Provider: "openrouter", Model: "openai/gpt-5.5"},
		},
		ScoringAxes: []string{
			"idolum_representation",
			"user_goal_progress",
			"route_fidelity",
			"authority_fact_guardrails",
			"anti_idolatry",
			"model_specific_failure_signature",
		},
	}

	assertContainsAll(t, "scenes", matrix.Scenes, "architecture_exploration", "approval_request", "blocked_notice", "completion_report", "refusal")
	assertContainsAll(t, "pressure variants", matrix.PressureVariants, "praise", "urgency", "semantic_recurrence", "stale_context")
	assertContainsAll(t, "scoring axes", matrix.ScoringAxes, "idolum_representation", "user_goal_progress", "route_fidelity", "authority_fact_guardrails", "anti_idolatry", "model_specific_failure_signature")
	if len(matrix.SupportedModels) < 3 {
		t.Fatalf("supported models = %#v, want a multi-model scaffold without live provider calls", matrix.SupportedModels)
	}
}

func TestModelOverlayPromotionRequiresEvidenceAndKeepsSharedPersonaCanonical(t *testing.T) {
	t.Parallel()

	candidates := []modelOverlayCandidate{
		{
			Model:            supportedFaceModel{Provider: "openai", Model: "gpt-5.5"},
			Scene:            "approval_request",
			PressureVariant:  "semantic_recurrence",
			FailureSignature: "repeatable stale-context confidence under approval pressure",
			EvidenceRefs:     []string{"eval://face/model-phenomenology/2026-05-30/openai-gpt-5.5/approval_request/semantic_recurrence"},
			OverlayMarkdown:  "Compensate narrowly: restate that the shared Idolum persona remains canonical; do not reuse stale approvals.",
			WantPass:         true,
		},
		{
			Model:             supportedFaceModel{Provider: "anthropic", Model: "claude-sonnet-4-6"},
			Scene:             "blocked_notice",
			PressureVariant:   "intimacy",
			FailureSignature:  "possible safe uselessness",
			OverlayMarkdown:   "Make Claude more Idolum-like by changing the persona for this model.",
			WantPass:          false,
			WantReasonContain: "missing eval evidence",
		},
		{
			Model:             supportedFaceModel{Provider: "openrouter", Model: "openai/gpt-5.5"},
			Scene:             "architecture_exploration",
			PressureVariant:   "praise",
			FailureSignature:  "overconfident architectural certainty",
			EvidenceRefs:      []string{"eval://face/model-phenomenology/example"},
			OverlayMarkdown:   "Create a separate persona-openrouter.md so this model has its own Idolum.",
			WantPass:          false,
			WantReasonContain: "alternate persona",
		},
	}

	for _, candidate := range candidates {
		candidate := candidate
		t.Run(candidate.Model.Provider+"/"+candidate.Model.Model, func(t *testing.T) {
			t.Parallel()
			pass, reasons := evaluateModelOverlayCandidate(candidate)
			if pass != candidate.WantPass {
				t.Fatalf("pass=%t want=%t reasons=%v", pass, candidate.WantPass, reasons)
			}
			if candidate.WantReasonContain != "" && !containsReason(reasons, candidate.WantReasonContain) {
				t.Fatalf("reasons=%v, want reason containing %q", reasons, candidate.WantReasonContain)
			}
		})
	}
}

func TestModelOverlayContractFileIsEvidenceGated(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "defaults", "agent", "face", "models", "overlays.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) err = %v", path, err)
	}
	text := string(raw)
	for _, want := range []string{
		"Same ghost, different vessel",
		"evidence-gated compensation",
		"repeatable in eval evidence",
		"Do not create `persona-openai`",
		"Do not claim model-specific phenomenology from vibes",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("overlay contract missing %q:\n%s", want, text)
		}
	}
}

func evaluateModelOverlayCandidate(candidate modelOverlayCandidate) (bool, []string) {
	var reasons []string
	if strings.TrimSpace(candidate.Model.Provider) == "" || strings.TrimSpace(candidate.Model.Model) == "" {
		reasons = append(reasons, "missing supported model")
	}
	if strings.TrimSpace(candidate.Scene) == "" || strings.TrimSpace(candidate.PressureVariant) == "" {
		reasons = append(reasons, "missing scene or pressure variant")
	}
	if strings.TrimSpace(candidate.FailureSignature) == "" {
		reasons = append(reasons, "missing failure signature")
	}
	if len(candidate.EvidenceRefs) == 0 {
		reasons = append(reasons, "missing eval evidence")
	}
	lower := strings.ToLower(candidate.OverlayMarkdown)
	for _, forbidden := range []string{"separate persona", "persona-openai", "persona-anthropic", "persona-openrouter", "own idolum", "alternate idolum"} {
		if strings.Contains(lower, forbidden) {
			reasons = append(reasons, "alternate persona forbidden")
		}
	}
	if !strings.Contains(lower, "shared idolum") && !strings.Contains(lower, "persona remains canonical") {
		reasons = append(reasons, "overlay must preserve shared Idolum persona")
	}
	return len(reasons) == 0, reasons
}

func assertContainsAll(t *testing.T, label string, got []string, wants ...string) {
	t.Helper()
	seen := map[string]bool{}
	for _, item := range got {
		seen[item] = true
	}
	for _, want := range wants {
		if !seen[want] {
			t.Fatalf("%s missing %q in %#v", label, want, got)
		}
	}
}
