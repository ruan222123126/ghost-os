package rss

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

func Parse(raw []byte, feedURL string, fetchedAt time.Time) (Result, error) {
	var root struct {
		XMLName xml.Name
	}
	if err := decodeXML(raw, &root); err != nil {
		return Result{}, fmt.Errorf("decode feed root: %w", err)
	}

	var result Result
	switch strings.ToLower(strings.TrimSpace(root.XMLName.Local)) {
	case "rss":
		parsed, err := parseRSS(raw, feedURL)
		if err != nil {
			return Result{}, err
		}
		result = parsed
	case "feed":
		parsed, err := parseAtom(raw, feedURL)
		if err != nil {
			return Result{}, err
		}
		result = parsed
	default:
		return Result{}, fmt.Errorf("unsupported feed root %q", root.XMLName.Local)
	}

	result.FetchedAt = fetchedAt.UTC().Format(time.RFC3339)
	return result, nil
}

func decodeXML(raw []byte, target any) error {
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	decoder.Strict = false
	return decoder.Decode(target)
}
