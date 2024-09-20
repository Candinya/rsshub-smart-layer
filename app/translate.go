package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/gorilla/feeds"
	"go.uber.org/zap"
)

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
