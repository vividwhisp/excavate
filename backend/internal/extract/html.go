package extract

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// HTTP fetches pages over the network and strips HTML to plain text for
// grounding the model. It needs no API key.
type HTTP struct {
	client *http.Client
}

func NewHTTP() *HTTP {
	return &HTTP{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

var (
	reScript = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle  = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reNoscript = regexp.MustCompile(`(?is)<noscript[^>]*>.*?</noscript>`)
	reTag    = regexp.MustCompile(`(?i)<[^>]+>`)
	reTitle  = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	reSpace  = regexp.MustCompile(`[ \t\r\f\v]+`)
	reNew    = regexp.MustCompile(`\n{3,}`)
)

func (h *HTTP) Extract(ctx context.Context, url string) (Page, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Page{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) ExcavateResearch/0.1")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := h.client.Do(req)
	if err != nil {
		return Page{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Page{}, fmt.Errorf("fetch %s: status %d", url, resp.StatusCode)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return Page{}, err
	}

	title := strings.TrimSpace(htmlText([]byte(matchGroup(reTitle, raw))))
	content := htmlText(raw)
	if title == "" {
		title = url
	}
	return Page{URL: url, Title: title, Content: content}, nil
}

// htmlText converts raw HTML into collapsed plain text.
func htmlText(raw []byte) string {
	s := string(raw)
	s = reScript.ReplaceAllString(s, " ")
	s = reStyle.ReplaceAllString(s, " ")
	s = reNoscript.ReplaceAllString(s, " ")
	s = reTag.ReplaceAllString(s, "\n")
	s = html.UnescapeString(s)
	s = reSpace.ReplaceAllString(s, " ")
	s = reNew.ReplaceAllString(s, "\n")
	return strings.TrimSpace(s)
}

func matchGroup(re *regexp.Regexp, raw []byte) string {
	m := re.FindSubmatch(raw)
	if len(m) < 2 {
		return ""
	}
	return string(m[1])
}

var _ Extractor = (*HTTP)(nil)