package app

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func (a *app) process(c echo.Context) error {
	// Get raw request
	req := c.Request()

	a.l.Debug("raw request url", zap.Any("request", req))

	// Get platform from path
	platform := c.Param("platform")

	a.l.Debug("platform", zap.String("platform", platform))

	// Get data from load balancer
	feed, err := a.lb.Fetch(req.URL.Path, platform)
	if err != nil {
		a.l.Error("failed to fetch feed", zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	a.l.Debug("raw feed data", zap.Any("feed", feed))

	// Check if translate is enabled
	var targetLang *string = nil
	a.l.Debug("check if translate is enabled", zap.String("host", req.Host))
	if a.cfg.Translate != nil && a.cfg.Translate.HostBase != "" &&
		strings.Contains(req.Host, a.cfg.Translate.HostBase) {
		// Found
		prefix := strings.SplitN(req.Host, a.cfg.Translate.HostBase, 2)
		targetLang = &prefix[0]

		a.l.Info("translate is enabled", zap.String("target", *targetLang), zap.String("host", req.Host))
	}

	a.l.Debug("start translate & image proxy")

	// Translate & Image proxy
	if a.tp != nil && targetLang != nil || a.ip != nil {
		for i, item := range feed.Items {
			processedItem := item

			// Translate
			if a.tp != nil && targetLang != nil {
				processedItem = a.translateItem(processedItem, targetLang, platform)
			}

			// Image proxy
			if a.ip != nil {
				processedItem = a.imageProxyItem(processedItem, req.Host, platform)
			}

			feed.Items[i] = processedItem
		}
	}

	// Re-construct to target format
	format := c.QueryParam("format")

	a.l.Debug("start re-construct format", zap.String("format", format))

	var (
		result      string
		contentType string
	)
	switch format {
	case "rss":
		result, err = feed.ToRss()
		contentType = "application/rss+xml"
	case "atom":
		result, err = feed.ToAtom()
		contentType = "application/atom+xml"
	case "json":
		result, err = feed.ToJSON()
		contentType = "application/json"
	default:
		// RSS 2.0
		result, err = feed.ToRss()
		contentType = "application/rss+xml"
	}

	if err != nil {
		a.l.Error("failed to format feed", zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.Blob(http.StatusOK, contentType, []byte(result))
}
