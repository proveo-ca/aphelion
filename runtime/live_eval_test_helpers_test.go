//go:build linux

package runtime

import (
	"encoding/json"
	"os"

	"github.com/idolum-ai/aphelion/config"
)

func loadMissionAskLiveEvalConfig() (*config.Config, string, error) {
	path, err := config.ResolveConfigPath(os.Getenv("APHELION_CONFIG"))
	if err != nil {
		return nil, "", err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, path, err
	}
	return cfg, path, nil
}

func strconvQuoteForEval(text string) string {
	raw, _ := json.Marshal(text)
	return string(raw)
}
