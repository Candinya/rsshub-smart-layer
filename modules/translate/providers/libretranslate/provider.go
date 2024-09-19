package libretranslate

import (
	"github.com/candinya/rsshub-smart-layer/modules/translate"
	"go.uber.org/zap"
)

var _ translate.Provider = (*lt)(nil)

type lt struct {
	l *zap.Logger

	url string
	key *string
}

type ltCfg struct {
	API struct {
		URL string  `yaml:"url"`
		Key *string `yaml:"key,omitempty"`
	} `yaml:"api"`
}
