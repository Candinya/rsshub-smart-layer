package app

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/feeds"
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

func (a *app) translateItem(item *feeds.Item, targetLang *string, platform string) *feeds.Item {
	// Translate wg
	var translateWg sync.WaitGroup

	// Translate channels
	tTitle := make(chan *string, 1)
	tDescription := make(chan *string, 1)
	tContent := make(chan *string, 1)

	// Title
	if item.Title != "" {
		a.l.Debug("translate item title", zap.String("title", item.Title))
		translateWg.Add(1)
		go func() {
			defer translateWg.Done()
			tTitle <- a.translatePart(item.Title, *targetLang, false, platform, item.Id, "title")
		}()
	}

	// Description
	if item.Description != "" {
		a.l.Debug("translate item description", zap.String("description", item.Description))
		translateWg.Add(1)
		go func() {
			defer translateWg.Done()
			tDescription <- a.translatePart(item.Description, *targetLang, true, platform, item.Id, "description")
		}()
	}

	// Content
	if item.Content != "" {
		a.l.Debug("translate item content", zap.String("content", item.Content))
		translateWg.Add(1)
		go func() {
			defer translateWg.Done()
			tContent <- a.translatePart(item.Content, *targetLang, true, platform, item.Id, "content")
		}()
	}

	// Wait for all finish
	translateWg.Wait()

	// Close channels
	close(tTitle)
	close(tDescription)
	close(tContent)

	// Check channel results
	translatedTitle := <-tTitle
	if translatedTitle != nil {
		a.l.Debug("item title translated", zap.String("title", item.Title), zap.String("translated", *translatedTitle))
		item.Title = *translatedTitle
	}

	translatedDescription := <-tDescription
	if translatedDescription != nil {
		a.l.Debug("item description translated", zap.String("description", item.Description), zap.String("translated", *translatedDescription))
		item.Description = *translatedDescription
	}

	translatedContent := <-tContent
	if translatedContent != nil {
		a.l.Debug("item content translated", zap.String("content", item.Content), zap.String("translated", *translatedContent))
		item.Content = *translatedContent
	}

	return item
}

func (a *app) translatePart(src string, targetLang string, isHTML bool, platform string, id string, part string) *string {
	// Build cache key
	cacheKey := fmt.Sprintf("%s%s:%s:%s:%s:%s", a.cfg.System.Redis.Prefix, "translate", platform, id, part, targetLang)

	// Try to get from redis
	a.l.Debug("try to get cache", zap.String("key", cacheKey))
	cachedResult, err := a.redis.Get(context.Background(), cacheKey).Result()
	if err != nil {
		a.l.Error("failed to check translated result from redis", zap.String("key", cacheKey), zap.Error(err))
	} else if cachedResult != "" {
		// Valid cache result, return
		a.l.Debug("valid translated result found", zap.String("key", cacheKey), zap.String("result", cachedResult))
		return &cachedResult
	}

	// Send to translate provider
	a.l.Debug("try to send with provider")
	translatedPart, err := a.tp.Translate(src, targetLang, isHTML)
	if err != nil {
		a.l.Error("failed to translate", zap.String("part", part), zap.String("id", id), zap.Error(err))
		return nil
	}

	// Save into cache
	a.l.Debug("save translated result into cache", zap.String("part", part), zap.String("id", id), zap.String("source", src), zap.String("translated", *translatedPart))
	a.redis.Set(context.Background(), cacheKey, translatedPart, a.cfg.System.Redis.CacheExpire)

	// Return
	return translatedPart
}

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
