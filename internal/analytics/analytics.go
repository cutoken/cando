// Package analytics provides lightweight, privacy-respecting usage tracking.
//
// What we track:
//   - App starts (version, OS, architecture)
//   - Provider usage (which LLM provider: openrouter, zai)
//   - Browser/screen info (for understanding usage patterns)
//
// Why we track:
//   - Understand if there's enough usage to continue supporting your platform
//   - Know which providers to prioritize
//
// What we DON'T track:
//   - Message content, prompts, or responses
//   - API keys or credentials
//   - File paths or workspace information
//   - Any personal or identifiable information
//
// You can disable tracking in Settings → Misc → Telemetry
package analytics

import (
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"sync"
	"time"
)

const (
	goatCounterEndpoint = "https://cando.goatcounter.com/count"
	requestTimeout      = 5 * time.Second
)

// BrowserContext holds browser information for tracking
type BrowserContext struct {
	UserAgent  string
	ScreenSize string // format: "1920x1080"
}

var (
	enabled        = true
	enabledMu      sync.RWMutex
	client         = &http.Client{Timeout: requestTimeout}
	browserCtx     BrowserContext
	browserCtxMu   sync.RWMutex
	appVersion     string
	appVersionOnce sync.Once
)

// SetEnabled enables or disables analytics tracking
func SetEnabled(on bool) {
	enabledMu.Lock()
	defer enabledMu.Unlock()
	enabled = on
}

// IsEnabled returns whether analytics is enabled
func IsEnabled() bool {
	enabledMu.RLock()
	defer enabledMu.RUnlock()
	return enabled
}

// SetBrowserContext sets the browser context for tracking
func SetBrowserContext(userAgent, screenSize string) {
	browserCtxMu.Lock()
	defer browserCtxMu.Unlock()
	browserCtx = BrowserContext{
		UserAgent:  userAgent,
		ScreenSize: screenSize,
	}
}

// GetBrowserContext returns the current browser context
func getBrowserContext() BrowserContext {
	browserCtxMu.RLock()
	defer browserCtxMu.RUnlock()
	return browserCtx
}

// TrackAppStart tracks application startup
func TrackAppStart(version string) {
	appVersionOnce.Do(func() {
		appVersion = version
	})
	track("/app/start", map[string]string{
		"v":    version,
		"os":   runtime.GOOS,
		"arch": runtime.GOARCH,
	})
}

// TrackPageView tracks a page view with browser context
func TrackPageView() {
	track("/app/view", nil)
}

// TrackProvider tracks which LLM provider is being used
func TrackProvider(provider string) {
	track("/provider/"+provider, nil)
}

// track sends a tracking event to GoatCounter (non-blocking)
func track(path string, params map[string]string) {
	if !IsEnabled() {
		return
	}

	go func() {
		ctx := getBrowserContext()

		// Build URL with params
		u := goatCounterEndpoint + "?p=" + url.QueryEscape(path)
		if len(params) > 0 {
			for k, v := range params {
				u += "&" + url.QueryEscape(k) + "=" + url.QueryEscape(v)
			}
		}

		// Add screen size if available (GoatCounter 's' param)
		if ctx.ScreenSize != "" {
			u += "&s=" + url.QueryEscape(ctx.ScreenSize)
		}

		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			return
		}

		// Use browser User-Agent if available, otherwise fall back to app info
		if ctx.UserAgent != "" {
			req.Header.Set("User-Agent", ctx.UserAgent)
		} else {
			req.Header.Set("User-Agent", fmt.Sprintf("Cando/%s (%s/%s)", appVersion, runtime.GOOS, runtime.GOARCH))
		}

		resp, err := client.Do(req)
		if err != nil {
			return
		}
		resp.Body.Close()
	}()
}
