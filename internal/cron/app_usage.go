package cron

import (
	"context"
	"encoding/json"
	"html"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
)

type AppUsageCrawler struct {
	appQ   *queries.AppQueries
	client *http.Client
	stop   chan struct{}
}

func NewAppUsageCrawler(appQ *queries.AppQueries) *AppUsageCrawler {
	return &AppUsageCrawler{
		appQ:   appQ,
		client: &http.Client{Timeout: 45 * time.Second},
		stop:   make(chan struct{}),
	}
}

func (c *AppUsageCrawler) Start() {
	go c.loop()
	log.Printf("App usage crawler started (daily)")
}

func (c *AppUsageCrawler) Stop() {
	close(c.stop)
}

func (c *AppUsageCrawler) loop() {
	c.run()
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.run()
		case <-c.stop:
			return
		}
	}
}

func (c *AppUsageCrawler) run() {
	if c.appQ == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if err := c.appQ.AggregateOpenPathsDaily(ctx, time.Now().UTC()); err != nil {
		log.Printf("app-usage: openpaths aggregate failed: %v", err)
	}

	apps, err := c.fetchOpenRouterApps(ctx)
	if err != nil {
		log.Printf("app-usage: openrouter crawl failed: %v", err)
		return
	}
	log.Printf("app-usage: crawled %d OpenRouter apps", len(apps))
}

func (c *AppUsageCrawler) fetchOpenRouterApps(ctx context.Context) ([]model.AIApp, error) {
	body, err := c.fetch(ctx, "https://openrouter.ai/apps")
	if err != nil {
		return nil, err
	}
	apps := parseOpenRouterApps(body)
	today := time.Now().UTC()
	for i := range apps {
		appID, err := c.appQ.Upsert(ctx, &apps[i])
		if err != nil {
			log.Printf("app-usage: upsert %s: %v", apps[i].Name, err)
			continue
		}
		if apps[i].ExternalPath != "" {
			if c.fetchOpenRouterAppModels(ctx, appID, apps[i].ExternalPath) {
				continue
			}
		}
		if apps[i].TotalTokens > 0 {
			_ = c.appQ.UpsertDailyUsage(ctx, today, appID, "openrouter", "", "", 0, 0, 0, apps[i].TotalTokens, 0)
		}
	}
	return apps, nil
}

func (c *AppUsageCrawler) fetchOpenRouterAppModels(ctx context.Context, appID, path string) bool {
	body, err := c.fetch(ctx, "https://openrouter.ai"+path)
	if err != nil {
		log.Printf("app-usage: app detail %s: %v", path, err)
		return false
	}
	rows := parseOpenRouterAppModels(body)
	for _, row := range rows {
		_ = c.appQ.UpsertDailyUsage(ctx, row.Day, appID, "openrouter", row.Model, "", 0, 0, 0, row.TotalTokens, 0)
	}
	return len(rows) > 0
}

func (c *AppUsageCrawler) fetch(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "OpenPaths app usage crawler")
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

var appAnchorRe = regexp.MustCompile(`(?s)<a class="[^"]*" href="(/apps/[^"]+)">.*?<img[^>]+alt="Favicon for ([^"]*)"[^>]+src="([^"]*)"[^>]*>.*?<span class="font-medium[^"]*">([^<]+)</span>.*?<p class="truncate">([^<]*)</p>.*?<span class="text-sm font-medium whitespace-nowrap">([^<]+)</span>`)

func parseOpenRouterApps(body string) []model.AIApp {
	matches := appAnchorRe.FindAllStringSubmatch(body, -1)
	seen := map[string]bool{}
	apps := make([]model.AIApp, 0, len(matches))
	for _, m := range matches {
		path := html.UnescapeString(m[1])
		if strings.Contains(path, "/category/") || seen[path] {
			continue
		}
		seen[path] = true
		appURL := strings.TrimSpace(html.UnescapeString(m[2]))
		name := strings.TrimSpace(html.UnescapeString(m[4]))
		apps = append(apps, model.AIApp{
			Slug:         strings.TrimPrefix(path, "/apps/"),
			Name:         name,
			URL:          appURL,
			Description:  strings.TrimSpace(html.UnescapeString(m[5])),
			FaviconURL:   html.UnescapeString(m[3]),
			Source:       "openrouter",
			ExternalPath: path,
			TotalTokens:  parseTokenCount(m[6]),
		})
	}
	return apps
}

type openRouterModelUsage struct {
	Day         time.Time
	Model       string
	TotalTokens int64
}

func parseOpenRouterAppModels(body string) []openRouterModelUsage {
	start := strings.Index(body, `appModelAnalytics`)
	if start < 0 {
		return nil
	}
	s := body[start:]
	lb := strings.Index(s, `[`)
	end := strings.Index(s, `],\"appName`)
	if end < 0 {
		end = strings.Index(s, `],"appName`)
	}
	if lb < 0 || end < 0 || end <= lb {
		return nil
	}
	raw := s[lb : end+1]
	raw = strings.ReplaceAll(raw, `\"`, `"`)
	raw = strings.ReplaceAll(raw, `\\`, `\`)
	var rows []struct {
		Date        string `json:"date"`
		Model       string `json:"model_permaslug"`
		TotalTokens int64  `json:"total_tokens"`
	}
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return nil
	}
	out := make([]openRouterModelUsage, 0, len(rows))
	for _, row := range rows {
		day, err := time.Parse("2006-01-02", row.Date)
		if err != nil || row.Model == "" {
			continue
		}
		out = append(out, openRouterModelUsage{Day: day, Model: row.Model, TotalTokens: row.TotalTokens})
	}
	return out
}

func parseTokenCount(raw string) int64 {
	raw = strings.TrimSpace(strings.TrimSuffix(raw, "tokens"))
	mult := float64(1)
	switch {
	case strings.HasSuffix(raw, "T"):
		mult = 1_000_000_000_000
		raw = strings.TrimSuffix(raw, "T")
	case strings.HasSuffix(raw, "B"):
		mult = 1_000_000_000
		raw = strings.TrimSuffix(raw, "B")
	case strings.HasSuffix(raw, "M"):
		mult = 1_000_000
		raw = strings.TrimSuffix(raw, "M")
	case strings.HasSuffix(raw, "K"):
		mult = 1_000
		raw = strings.TrimSuffix(raw, "K")
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0
	}
	return int64(v * mult)
}
