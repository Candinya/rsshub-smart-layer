package modules

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"

	"github.com/candinya/rsshub-smart-layer/types"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type ImageProxy struct {
	l *zap.Logger

	path  string
	rules map[string]types.ConfigImageProxyRule
}

func NewImageProxy(cfg *types.ConfigImageProxy, l *zap.Logger) *ImageProxy {
	return &ImageProxy{
		l:     l,
		path:  cfg.Path,
		rules: cfg.Rules,
	}
}

func (p *ImageProxy) ProcessLink(src string, host string, platform string) string {
	// Prepare query
	query := url.Values{}
	query.Set("s", src)
	query.Set("p", platform)

	return (&url.URL{
		Host:     host,
		Path:     p.path,
		RawQuery: query.Encode(),
	}).String()
}

func (p *ImageProxy) ProcessHTML(contentWithoutProxy string, host string, platform string) string {
	// Parse HTML
	p.l.Debug("start process html")
	parsed, err := html.ParseFragment(strings.NewReader(contentWithoutProxy), &html.Node{
		Type:     html.ElementNode,
		Data:     "body",
		DataAtom: atom.Body,
	})
	if err != nil {
		p.l.Error("failed to parse html", zap.Error(err))
		return contentWithoutProxy // Keep untouched
	}

	// Replace all src from img tags
	var b bytes.Buffer
	for _, node := range parsed {
		p.l.Debug("proxy all img tags", zap.Any("node", node))
		p.traverseHTMLTree(node, host, platform)

		p.l.Debug("render back to html", zap.Any("node", node))
		err = html.Render(&b, node)
		if err != nil {
			p.l.Error("failed to render html", zap.Error(err))
			return contentWithoutProxy
		}
	}

	// Return
	return b.String()
}

func (p *ImageProxy) traverseHTMLTree(n *html.Node, host string, platform string) {
	// Find all img tags
	if n.Type == html.ElementNode && n.Data == "img" {
		p.l.Debug("found an img tag", zap.Any("node", n))
		for i, a := range n.Attr {
			if a.Key == "src" {
				proxied := p.ProcessLink(a.Val, host, platform)
				p.l.Debug("replace src", zap.String("old", a.Val), zap.String("new", proxied))
				n.Attr[i].Val = proxied
			}
		}
	}

	// Traverse children
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		p.traverseHTMLTree(c, host, platform)
	}
}

func (p *ImageProxy) Proxy(c echo.Context) error {
	imageSrc := c.QueryParam("s")
	platform := c.QueryParam("p")

	imageRequest, err := http.NewRequest("GET", imageSrc, nil)
	if err != nil {
		p.l.Error("image proxy new request", zap.String("src", imageSrc), zap.String("platform", platform), zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	// Apply platform specific rules
	if _, ok := p.rules[platform]; ok {
		if p.rules[platform].Origin != nil {
			imageRequest.Header.Add("Origin", *p.rules[platform].Origin)
		}
		if p.rules[platform].Referer != nil {
			imageRequest.Header.Add("Referer", *p.rules[platform].Referer)
		}
	}

	// Execute request
	res, err := http.DefaultClient.Do(imageRequest)
	if err != nil {
		p.l.Error("image proxy execute request", zap.String("src", imageSrc), zap.String("platform", platform), zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	defer res.Body.Close()

	return c.Stream(res.StatusCode, res.Header.Get("Content-Type"), res.Body)
}

func (p *ImageProxy) Path() string {
	return p.path
}
