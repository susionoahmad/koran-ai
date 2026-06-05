package rss

import (
	"context"
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"
)

// FeedItem represents a single parsed article entry from an RSS/Atom feed.
type FeedItem struct {
	Title       string
	URL         string
	Author      string
	PublishedAt *time.Time
	Content     string
	ImageURL    string
}

// Parser defines the interface for fetching and parsing RSS/Atom feeds.
type Parser interface {
	Parse(ctx context.Context, rssURL string) ([]FeedItem, error)
}

type gofeedParser struct {
	fp *gofeed.Parser
}

// NewParser creates a new gofeed-based RSS/Atom parser.
func NewParser() Parser {
	return &gofeedParser{
		fp: gofeed.NewParser(),
	}
}

// Parse fetches the RSS/Atom feed at rssURL and returns a slice of FeedItems.
// Supports RSS 2.0 and Atom feeds via gofeed's auto-detection.
func (p *gofeedParser) Parse(ctx context.Context, rssURL string) ([]FeedItem, error) {
	feed, err := p.fp.ParseURLWithContext(rssURL, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed %q: %w", rssURL, err)
	}

	items := make([]FeedItem, 0, len(feed.Items))
	for _, item := range feed.Items {
		fi := FeedItem{
			Title: item.Title,
			URL:   item.Link,
		}

		// Author: try item-level first, fallback to feed-level
		if item.Author != nil && item.Author.Name != "" {
			fi.Author = item.Author.Name
		} else if feed.Author != nil {
			fi.Author = feed.Author.Name
		}

		// PublishedAt: prefer UpdatedParsed, then PublishedParsed
		if item.UpdatedParsed != nil {
			fi.PublishedAt = item.UpdatedParsed
		} else if item.PublishedParsed != nil {
			fi.PublishedAt = item.PublishedParsed
		}

		// Content: prefer full content extension, fallback to description
		if item.Content != "" {
			fi.Content = item.Content
		} else {
			fi.Content = item.Description
		}

		// Image extraction with priority:
		// 1. media:content
		// 2. media:thumbnail
		// 3. enclosure
		// 4. item.Image fallback
		var imageURL string

		// Check media:content and media:thumbnail
		if ext, ok := item.Extensions["media"]; ok {
			if contentList, ok := ext["content"]; ok && len(contentList) > 0 {
				if u, ok := contentList[0].Attrs["url"]; ok && u != "" {
					imageURL = u
				}
			}
			if imageURL == "" {
				if thumbList, ok := ext["thumbnail"]; ok && len(thumbList) > 0 {
					if u, ok := thumbList[0].Attrs["url"]; ok && u != "" {
						imageURL = u
					}
				}
			}
		}

		// Check enclosure image
		if imageURL == "" && len(item.Enclosures) > 0 {
			for _, enc := range item.Enclosures {
				if enc.Type != "" && len(enc.Type) >= 5 && enc.Type[:5] == "image" {
					imageURL = enc.URL
					break
				}
			}
		}

		// Check item.Image
		if imageURL == "" && item.Image != nil {
			imageURL = item.Image.URL
		}

		fi.ImageURL = imageURL

		// Note: Quality filter check (like title/content empty check) will be handled in service layer.
		// We still parse URL and Title to make sure we don't have completely corrupt entries.
		if fi.URL != "" && fi.Title != "" {
			items = append(items, fi)
		}
	}

	return items, nil
}
