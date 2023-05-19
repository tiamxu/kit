package jenkins

import (
	"a/kit/cmd/config"

	"github.com/koding/multiconfig"
)

const configPath = "config/config.yaml"

var (
	cfg *config.Config
)

func loadConfig() {
	cfg = new(config.Config)
	multiconfig.MustLoadWithPath(configPath, cfg)
}
