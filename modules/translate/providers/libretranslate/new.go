package libretranslate

import (
	"fmt"

	"github.com/candinya/rsshub-smart-layer/modules/translate"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func New(settings string, l *zap.Logger) (translate.Provider, error) {
	var cfg ltCfg
	err := yaml.Unmarshal([]byte(settings), &cfg)
	if err != nil {
		return nil, fmt.Errorf("libretranslate config parse err: %v", err)
	}

	return &lt{
		l:   l,
		url: cfg.API.URL,
		key: cfg.API.Key,
	}, nil
}
