package rss

import "time"

type Result struct {
	Feed      FeedInfo `json:"feed"`
	Items     []Item   `json:"items"`
	FetchedAt string   `json:"fetched_at"`
}

type FeedInfo struct {
	Title       string `json:"title,omitempty"`
	Link        string `json:"link,omitempty"`
	Description string `json:"description,omitempty"`
	FeedURL     string `json:"feed_url,omitempty"`
}

type Item struct {
	ID          string `json:"id,omitempty"`
	Title       string `json:"title,omitempty"`
	Link        string `json:"link,omitempty"`
	Summary     string `json:"summary,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
	SourceTitle string `json:"source_title,omitempty"`
	SourceLink  string `json:"source_link,omitempty"`
}

type rssDocument struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	GUID        string `xml:"guid"`
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Summary     string `xml:"summary"`
	Content     string `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
	PubDate     string `xml:"pubDate"`
	Published   string `xml:"published"`
	Updated     string `xml:"updated"`
}

type atomDocument struct {
	Title    string      `xml:"title"`
	Subtitle string      `xml:"subtitle"`
	Links    []atomLink  `xml:"link"`
	Entries  []atomEntry `xml:"entry"`
}

type atomEntry struct {
	ID        string     `xml:"id"`
	Title     string     `xml:"title"`
	Summary   string     `xml:"summary"`
	Content   string     `xml:"content"`
	Published string     `xml:"published"`
	Updated   string     `xml:"updated"`
	Links     []atomLink `xml:"link"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type sortableItem struct {
	item          Item
	publishedTime time.Time
	hasPublished  bool
	index         int
}
