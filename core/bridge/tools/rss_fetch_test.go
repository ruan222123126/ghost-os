package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	internalrss "ghost-os/bridge/tools/internal/rss"
)

func TestRSSFetchToolExecuteParsesRSSAndDedupes(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Ghost Feed</title>
    <link>https://example.com</link>
    <description>Latest updates</description>
    <item>
      <guid>dup-guid</guid>
      <title>Older Post</title>
      <link>https://example.com/older</link>
      <description><![CDATA[<p>Older <b>summary</b></p>]]></description>
      <pubDate>Sat, 07 Mar 2026 08:00:00 GMT</pubDate>
    </item>
    <item>
      <guid>post-2</guid>
      <title>Newer Post</title>
      <link>https://example.com/newer</link>
      <description>Newest summary</description>
      <pubDate>Sun, 08 Mar 2026 08:00:00 GMT</pubDate>
    </item>
    <item>
      <guid>dup-guid</guid>
      <title>Duplicate Post</title>
      <link>https://example.com/duplicate</link>
      <description>Should be skipped</description>
      <pubDate>Sun, 08 Mar 2026 09:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`))
	}))
	defer server.Close()

	fixedNow := time.Date(2026, 3, 8, 10, 30, 0, 0, time.UTC)
	tool := &RSSFetchTool{
		httpClient: server.Client(),
		validateURL: func(context.Context, *url.URL) error {
			return nil
		},
		now:       func() time.Time { return fixedNow },
		bodyLimit: defaultRSSBodyLimitBytes,
	}

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"`+server.URL+`/feed.xml","max_items":5}`), "trace-rss-1")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	var result internalrss.Result
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Feed.Title != "Ghost Feed" {
		t.Fatalf("unexpected feed title: got %q want %q", result.Feed.Title, "Ghost Feed")
	}
	if result.FetchedAt != fixedNow.Format(time.RFC3339) {
		t.Fatalf("unexpected fetched_at: got %q want %q", result.FetchedAt, fixedNow.Format(time.RFC3339))
	}
	if len(result.Items) != 2 {
		t.Fatalf("unexpected item count: got %d want %d", len(result.Items), 2)
	}
	if result.Items[0].Title != "Newer Post" {
		t.Fatalf("expected newest item first, got %+v", result.Items[0])
	}
	if result.Items[1].Summary != "Older summary" {
		t.Fatalf("expected cleaned summary, got %q", result.Items[1].Summary)
	}
	if result.Items[0].SourceTitle != "Ghost Feed" || result.Items[0].SourceLink != "https://example.com" {
		t.Fatalf("unexpected source fields: %+v", result.Items[0])
	}
}

func TestRSSFetchToolExecuteParsesAtomAndRespectsSummaryToggle(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Ghost Atom</title>
  <subtitle>Atom updates</subtitle>
  <link href="https://example.com/atom" rel="alternate" />
  <entry>
    <id>tag:example.com,2026:newest</id>
    <title>Atom New</title>
    <link href="https://example.com/atom/new" />
    <summary>&lt;p&gt;Atom summary&lt;/p&gt;</summary>
    <updated>2026-03-08T11:00:00Z</updated>
  </entry>
</feed>`))
	}))
	defer server.Close()

	tool := &RSSFetchTool{
		httpClient: server.Client(),
		validateURL: func(context.Context, *url.URL) error {
			return nil
		},
		now:       func() time.Time { return time.Date(2026, 3, 8, 11, 30, 0, 0, time.UTC) },
		bodyLimit: defaultRSSBodyLimitBytes,
	}

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"`+server.URL+`/atom.xml","include_summary":false}`), "trace-rss-2")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	var result internalrss.Result
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Feed.Title != "Ghost Atom" {
		t.Fatalf("unexpected feed title: got %q want %q", result.Feed.Title, "Ghost Atom")
	}
	if len(result.Items) != 1 {
		t.Fatalf("unexpected item count: got %d want %d", len(result.Items), 1)
	}
	if result.Items[0].Summary != "" {
		t.Fatalf("expected summary to be omitted, got %q", result.Items[0].Summary)
	}
	if result.Items[0].PublishedAt != "2026-03-08T11:00:00Z" {
		t.Fatalf("unexpected published time: got %q", result.Items[0].PublishedAt)
	}
}

func TestRSSFetchToolRequiresHTTPSURL(t *testing.T) {
	tool := NewRSSFetchTool()
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"http://example.com/feed.xml"}`), "trace-rss-3")
	if err == nil {
		t.Fatal("expected error for non-https url")
	}
	if !strings.Contains(err.Error(), "https") {
		t.Fatalf("expected https validation error, got %v", err)
	}
}

func TestValidateRSSURLRejectsLocalTargets(t *testing.T) {
	for _, raw := range []string{
		"https://127.0.0.1/feed.xml",
		"https://localhost/feed.xml",
	} {
		parsed, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("parse url %q: %v", raw, err)
		}
		if err := validateRSSURL(context.Background(), parsed); err == nil {
			t.Fatalf("expected validation error for %q", raw)
		}
	}
}

func TestRSSFetchToolRejectsInvalidXML(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<rss><channel><title>broken</title>`))
	}))
	defer server.Close()

	tool := &RSSFetchTool{
		httpClient: server.Client(),
		validateURL: func(context.Context, *url.URL) error {
			return nil
		},
		now:       time.Now,
		bodyLimit: defaultRSSBodyLimitBytes,
	}

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"`+server.URL+`/broken.xml"}`), "trace-rss-4")
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "parse rss feed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRSSFetchToolCapsMaxItems(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>Ghost Feed</title>
  <item><guid>a</guid><title>A</title><pubDate>Sun, 08 Mar 2026 10:00:00 GMT</pubDate></item>
  <item><guid>b</guid><title>B</title><pubDate>Sun, 08 Mar 2026 09:00:00 GMT</pubDate></item>
</channel></rss>`))
	}))
	defer server.Close()

	tool := &RSSFetchTool{
		httpClient: server.Client(),
		validateURL: func(context.Context, *url.URL) error {
			return nil
		},
		now:       time.Now,
		bodyLimit: defaultRSSBodyLimitBytes,
	}

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"`+server.URL+`/feed.xml","max_items":1}`), "trace-rss-5")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	var result internalrss.Result
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("unexpected item count: got %d want %d", len(result.Items), 1)
	}
}
