package providers

import (
	"fmt"

	"github.com/candinya/rsshub-smart-layer/modules/translate"
	"github.com/candinya/rsshub-smart-layer/modules/translate/providers/libretranslate"
	"github.com/candinya/rsshub-smart-layer/types"
	"go.uber.org/zap"
)

func NewTranslator(cfg *types.ConfigTranslate, l *zap.Logger) (translate.Provider, error) {
	switch cfg.Platform {
	case "libretranslate":
		return libretranslate.New(cfg.Settings, l)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", cfg.Platform)
	}
}
