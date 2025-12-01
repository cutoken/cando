package tooling

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"
)

// WebFetchJSONTool fetches a web page and returns a cleaned JSON summary.
type WebFetchJSONTool struct {
	client   *http.Client
	maxBytes int64
}

func NewWebFetchJSONTool(timeout time.Duration) *WebFetchJSONTool {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &WebFetchJSONTool{
		client:   &http.Client{Timeout: timeout},
		maxBytes: 2 << 20, // 2MB
	}
}

func (t *WebFetchJSONTool) Definition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: ToolFunction{
			Name:        "web_fetch_json",
			Description: "Fetch a web page and return cleaned JSON (title, description, headings, paragraphs).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "Absolute URL to fetch (http or https).",
					},
					"max_paragraphs": map[string]any{
						"type":        "integer",
						"description": "Maximum number of paragraph snippets to include (default 5).",
					},
					"include_headings": map[string]any{
						"type":        "boolean",
						"description": "Whether to include h1-h3 headings (default true).",
					},
				},
				"required": []string{"url"},
			},
		},
	}
}

func (t *WebFetchJSONTool) Call(ctx context.Context, args map[string]any) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	rawURL, ok := stringArg(args, "url")
	if !ok || strings.TrimSpace(rawURL) == "" {
		return "", errors.New("url is required")
	}
	maxParagraphs := intArg(args, "max_paragraphs", 5)
	if maxParagraphs <= 0 {
		maxParagraphs = 5
	}
	includeHeadings := boolArg(args, "include_headings", true)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Cando/1.0 (+https://github.com/cutoken/cando)")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	limited := &io.LimitedReader{R: resp.Body, N: t.maxBytes}
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	truncated := limited.N == 0

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("parse html: %w", err)
	}

	title := strings.TrimSpace(doc.Find("title").First().Text())
	desc := strings.TrimSpace(doc.Find(`meta[name="description"]`).AttrOr("content", ""))

	var headings []string
	if includeHeadings {
		doc.Find("h1, h2, h3").EachWithBreak(func(_ int, sel *goquery.Selection) bool {
			text := normalizeWhitespace(sel.Text())
			if text != "" {
				headings = append(headings, text)
			}
			return true
		})
	}

	paragraphs := make([]string, 0, maxParagraphs)
	doc.Find("p").EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		if len(paragraphs) >= maxParagraphs {
			return false
		}
		text := normalizeWhitespace(sel.Text())
		if len(text) < 40 { // skip super short fragments
			return true
		}
		paragraphs = append(paragraphs, text)
		return true
	})

	payload := map[string]any{
		"url":              resp.Request.URL.String(),
		"status":           resp.StatusCode,
		"fetched_at":       time.Now().UTC().Format(time.RFC3339),
		"bytes_downloaded": len(body),
		"truncated":        truncated,
		"title":            title,
		"description":      desc,
		"headings":         headings,
		"paragraphs":       paragraphs,
	}

	data, err := jsonMarshalNoEscape(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func normalizeWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	space := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			space = true
			continue
		}
		if space && b.Len() > 0 {
			b.WriteByte(' ')
		}
		space = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}
