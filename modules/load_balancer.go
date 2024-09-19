package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/candinya/rsshub-smart-layer/types"
	"github.com/gorilla/feeds"
	"go.uber.org/zap"
)

type LoadBalancer struct {
	l *zap.Logger

	timeout     time.Duration
	instaceList []string
	platformMap map[string][]int
	fallbacks   []int
}

func NewLoadBalancer(list types.ConfigRSSHubList, timeout time.Duration, l *zap.Logger) (*LoadBalancer, error) {
	instanceList := make([]string, len(list))
	platformMap := make(map[string][]int)

	var fallbacks []int

	for id, instance := range list {
		// Set URL
		instanceList[id] = instance.URL

		// Set platform preferences
		if len(instance.Platforms) > 0 {
			for _, platform := range instance.Platforms {
				// Initialize platform mapping
				if _, ok := platformMap[platform]; !ok {
					platformMap[platform] = []int{}
				}

				// Append current
				platformMap[platform] = append(platformMap[platform], id)
			}
		}

		// Set fallback preferences
		if instance.Fallback {
			fallbacks = append(fallbacks, id)
		}
	}

	// Initialize complete
	return &LoadBalancer{
		l,
		timeout,
		instanceList,
		platformMap,
		fallbacks,
	}, nil
}

func (lb *LoadBalancer) fetchInstance(ctx context.Context, reqUrl string, id int) (*feeds.JSONFeed, error) {
	// Prepare request client
	rc := http.Client{
		Timeout: lb.timeout,
	}

	// Parse request URL
	originUrl, err := url.Parse(reqUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}

	requestUrl, err := url.Parse(lb.instaceList[id] + originUrl.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to concat url: %w", err)
	}

	// Force JSON format to simplify process mechanisms
	query := originUrl.Query()
	query.Set("format", "json")
	requestUrl.RawQuery = query.Encode()

	// Prepare request
	req, err := http.NewRequestWithContext(ctx, "GET", requestUrl.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	lb.l.Debug("do request")
	res, err := rc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}

	defer res.Body.Close() // Ignore errors

	// Check response
	if res.StatusCode != 200 {
		lb.l.Debug("response status not OK")
		return nil, fmt.Errorf("bad status code: %d", res.StatusCode)
	}

	// Parse response into feed
	lb.l.Debug("start parse response to JSON feed")
	var feed feeds.JSONFeed
	err = json.NewDecoder(res.Body).Decode(&feed)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response into JSON: %w", err)
	}

	// Return feed
	lb.l.Debug("JSON feed parsed", zap.Any("feed", feed))
	return &feed, nil
}

func (lb *LoadBalancer) fetchFromGroup(reqUrl string, group []int) (*feeds.JSONFeed, error) {
	if len(group) == 0 {
		lb.l.Debug("empty group")
		return nil, fmt.Errorf("empty group")
	}

	// Get random one
	randInstanceNo := rand.Intn(len(group))
	randInstanceID := group[randInstanceNo]
	lb.l.Debug("get random instance id", zap.Int("randInstanceID", randInstanceID))

	// Prepare context
	ctx := context.Background()

	// Fetch from selected
	lb.l.Debug("fetch from instance", zap.Int("randInstanceID", randInstanceID))
	feed, err := lb.fetchInstance(ctx, reqUrl, randInstanceID)
	if err == nil {
		// Success
		lb.l.Debug("fetch successfully", zap.Any("feed", feed))
		return feed, nil
	}

	// Else: fail to fetch
	lb.l.Warn("failed to get feed from RSSHub instance",
		zap.String("instance", lb.instaceList[randInstanceID]),
		zap.String("url", reqUrl),
		zap.Error(err),
	)

	if len(group) == 1 {
		// No remain possibilities, just return
		lb.l.Debug("no remain member in group")
		return nil, fmt.Errorf("all attempts failed")
	}

	// Try all other instances simultaneously to save time
	lb.l.Debug("try all other instances in group")
	ctxAll, cancelAll := context.WithCancel(context.Background())
	defer cancelAll()
	var feedWg sync.WaitGroup
	feedRes := (*feeds.JSONFeed)(nil)
	var feedLock sync.Mutex

	// Require at least one remain
	feedWg.Add(1)
	for instanceNo, instanceID := range group {
		if instanceNo != randInstanceNo {
			// Copy once to prevent change by loop
			instanceID := instanceID

			// Run by go coroutine
			go func() {
				feed, err := lb.fetchInstance(ctxAll, reqUrl, instanceID)
				if err == nil && feed != nil && feedRes == nil {
					// Sync lock to prevent thread conflict
					feedLock.Lock()
					if feedRes == nil {
						feedRes = feed
						cancelAll() // Cancel all running requests to release resources
						feedWg.Done()
					}
					feedLock.Unlock()
				} // else fail to request, do nothing
			}()
		}
	}

	// Wait for return
	feedWg.Wait()

	// Still no luck :(
	if feedRes == nil {
		lb.l.Debug("all attempts failed")
		return nil, fmt.Errorf("all attempts failed")
	}

	// Gather successfully
	lb.l.Debug("fetch successfully", zap.Any("feed", feedRes))
	return feedRes, nil
}

func (lb *LoadBalancer) Fetch(reqUrl string, platform string) (*feeds.Feed, error) {
	lb.l.Debug("start fetch", zap.String("url", reqUrl))

	if group, found := lb.platformMap[platform]; found {
		// Find instance with platform match, try them before using fallback group
		lb.l.Debug("find instance with platform match", zap.Any("group", group))
		feed, err := lb.fetchFromGroup(reqUrl, group)
		if err != nil {
			lb.l.Warn("failed to get feed from preferred instance list, try with fallback")
		} else if feed != nil {
			// Successfully get feed
			lb.l.Debug("successfully fetched feed from preferred instance", zap.Any("feed", feed))
			return lb.convertJSON2Feed(feed), nil
		}
	}

	// Try with fallback instances
	lb.l.Debug("try to get from fallback instances", zap.Any("group", lb.fallbacks))
	feed, err := lb.fetchFromGroup(reqUrl, lb.fallbacks)
	if err != nil {
		return nil, fmt.Errorf("failed to request feed: %w", err)
	}

	// Success
	lb.l.Debug("successfully fetched feed from fallbacks", zap.Any("feed", feed))
	return lb.convertJSON2Feed(feed), nil
}

// convertJSON2Feed: Convert feeds.JSONFeed to feeds.Feed
// basically the reverse of feeds.JSON.JSONFeed()
func (lb *LoadBalancer) convertJSON2Feed(jsonFeed *feeds.JSONFeed) *feeds.Feed {
	// Check if is empty feed
	if jsonFeed == nil {
		return nil
	}

	// Set basic
	feed := &feeds.Feed{
		Title:       jsonFeed.Title,
		Description: jsonFeed.Description,
	}

	// Set link
	if jsonFeed.HomePageUrl != "" {
		feed.Link = &feeds.Link{
			Href: jsonFeed.HomePageUrl,
		}
	}

	// Set author
	if len(jsonFeed.Authors) > 0 {
		feed.Author = &feeds.Author{
			Name: jsonFeed.Authors[0].Name,
		}
	} else if jsonFeed.Author != nil {
		feed.Author = &feeds.Author{
			Name: jsonFeed.Author.Name,
		}
	}

	// Set items
	for _, jsonItem := range jsonFeed.Items {
		feed.Items = append(feed.Items, lb.convertJSON2Item(jsonItem))
	}

	return feed
}

func (lb *LoadBalancer) convertJSON2Item(jsonItem *feeds.JSONItem) *feeds.Item {
	item := &feeds.Item{
		Id:          jsonItem.Id,
		Title:       jsonItem.Title,
		Description: jsonItem.Summary,
		Content:     jsonItem.ContentHTML,
	}

	if jsonItem.Url != "" {
		item.Link = &feeds.Link{
			Href: jsonItem.Url,
		}
	}
	if jsonItem.ExternalUrl != "" {
		item.Source = &feeds.Link{
			Href: jsonItem.ExternalUrl,
		}
	}
	if len(jsonItem.Authors) > 0 {
		item.Author = &feeds.Author{
			Name: jsonItem.Authors[0].Name,
		}
	} else if jsonItem.Author != nil {
		item.Author = &feeds.Author{
			Name: jsonItem.Author.Name,
		}
	}
	if jsonItem.PublishedDate != nil && !jsonItem.PublishedDate.IsZero() {
		item.Created = *jsonItem.PublishedDate
	}
	if jsonItem.ModifiedDate != nil && !jsonItem.ModifiedDate.IsZero() {
		item.Updated = *jsonItem.ModifiedDate
	}

	if jsonItem.Image != "" {
		item.Enclosure = &feeds.Enclosure{
			Url: jsonItem.Image,
		}
	}

	return item
}
