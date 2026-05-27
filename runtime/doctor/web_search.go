//go:build linux

package doctor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/idolum-ai/aphelion/session"
)

func (r *Runtime) writeDoctorWebSearchStatus(b *strings.Builder) {
	if r == nil || r.cfg == nil {
		WriteLine(b, "web_search: unavailable")
		return
	}
	cfg := r.cfg.Tools.WebSearch
	WriteLine(b, "web_search: configured")
	WriteKV(b, "web_search_enabled", strconv.FormatBool(cfg.Enabled))
	WriteKV(b, "web_search_provider_order", strings.Join(cfg.ProviderOrder, ","))
	WriteKV(b, "web_search_openai_hosted", webSearchDoctorEnabled(cfg.OpenAIHosted.Enabled))
	braveStatus := webSearchDoctorEnabled(cfg.Brave.Enabled)
	if cfg.Brave.Enabled {
		switch {
		case strings.TrimSpace(cfg.Brave.APIKeyEnv) != "":
			braveStatus += ":api_key_env"
		case strings.TrimSpace(cfg.Brave.APIKeyFile) != "":
			braveStatus += ":api_key_file"
		default:
			braveStatus += ":missing_credential_reference"
		}
	}
	WriteKV(b, "web_search_brave", braveStatus)
	if r.store == nil {
		WriteKV(b, "web_search_grant", "unknown_no_store")
		return
	}
	grants, err := r.store.CapabilityGrants(200, session.CapabilityGrantStatusActive, session.CapabilityKindTool, "")
	if err != nil {
		WriteKV(b, "web_search_grant_error", err.Error())
		return
	}
	active := 0
	for _, grant := range grants {
		if strings.TrimSpace(grant.TargetResource) == "web_search" {
			active++
		}
	}
	WriteKV(b, "web_search_active_grants", fmt.Sprintf("%d", active))
}

func webSearchDoctorEnabled(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}
