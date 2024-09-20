package app

import (
	"github.com/gorilla/feeds"
	"go.uber.org/zap"
)

func (a *app) imageProxyItem(item *feeds.Item, host string, platform string) *feeds.Item {
	// Proxy content
	if item.Content != "" {
		a.l.Debug("image proxy item content", zap.String("content", item.Content))
		item.Content = a.ip.ProcessHTML(item.Content, host, platform)
	}

	// Proxy description
	if item.Description != "" {
		a.l.Debug("image proxy item description", zap.String("description", item.Description))
		item.Description = a.ip.ProcessHTML(item.Description, host, platform)
	}

	// Proxy enclosure
	if item.Enclosure != nil {
		a.l.Debug("image proxy image enclosure", zap.String("enclosure", item.Enclosure.Url))
		item.Enclosure.Url = a.ip.ProcessLink(item.Enclosure.Url, host, platform)
	}

	return item
}
