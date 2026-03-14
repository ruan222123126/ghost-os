package tools

import (
	"math"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

func filterOCRItems(items []screenOCRItem, query string, matchMode string, maxDistance int) []screenOCRItem {
	queryNormalized := normalizeForMatch(query, matchMode)
	filtered := make([]screenOCRItem, 0, len(items))
	for _, item := range items {
		if !matchesOCRItem(item.Text, query, queryNormalized, matchMode, maxDistance) {
			continue
		}
		filtered = append(filtered, item)
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].BBox.Y == filtered[j].BBox.Y {
			return filtered[i].BBox.X < filtered[j].BBox.X
		}
		return filtered[i].BBox.Y < filtered[j].BBox.Y
	})
	return filtered
}

func matchesOCRItem(text string, query string, queryNormalized string, matchMode string, maxDistance int) bool {
	switch matchMode {
	case "exact":
		return strings.TrimSpace(text) == query
	case "contains":
		return strings.Contains(text, query)
	case "case_insensitive":
		return strings.EqualFold(strings.TrimSpace(text), query)
	case "normalized":
		return normalizeForMatch(text, matchMode) == queryNormalized
	case "fuzzy":
		itemNormalized := normalizeForMatch(text, matchMode)
		if itemNormalized == "" || queryNormalized == "" {
			return false
		}
		return levenshteinDistance(itemNormalized, queryNormalized) <= maxDistance
	default:
		return strings.Contains(text, query)
	}
}

func isMatchModeSupported(matchMode string) bool {
	switch matchMode {
	case "exact", "contains", "case_insensitive", "normalized", "fuzzy":
		return true
	default:
		return false
	}
}

func defaultFuzzyDistance(query string) int {
	length := utf8.RuneCountInString(strings.TrimSpace(query))
	switch {
	case length <= 4:
		return 1
	case length <= 8:
		return 2
	default:
		return 3
	}
}

func normalizeForMatch(text string, matchMode string) string {
	switch matchMode {
	case "case_insensitive":
		return strings.ToLower(strings.TrimSpace(text))
	case "normalized", "fuzzy":
		return normalizeText(text)
	default:
		return strings.TrimSpace(text)
	}
}

func normalizeText(text string) string {
	var builder strings.Builder
	builder.Grow(len(text))
	for _, r := range text {
		switch {
		case r == 0x3000:
			r = ' '
		case r >= 0xFF01 && r <= 0xFF5E:
			r -= 0xFEE0
		}
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			continue
		}
		builder.WriteRune(unicode.ToLower(r))
	}
	return builder.String()
}

func levenshteinDistance(a string, b string) int {
	ra := []rune(a)
	rb := []rune(b)
	if len(ra) == 0 {
		return len(rb)
	}
	if len(rb) == 0 {
		return len(ra)
	}

	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := 0; j <= len(rb); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			curr[j] = minDistance(
				prev[j]+1,
				curr[j-1]+1,
				prev[j-1]+substitutionCost(ra[i-1], rb[j-1]),
			)
		}
		prev, curr = curr, prev
	}
	return prev[len(rb)]
}

func substitutionCost(left rune, right rune) int {
	if left == right {
		return 0
	}
	return 1
}

func minDistance(values ...int) int {
	best := math.MaxInt
	for _, value := range values {
		if value < best {
			best = value
		}
	}
	return best
}

func sortOCRItemsByPoint(items []screenOCRItem, point screenPoint) {
	sort.SliceStable(items, func(i, j int) bool {
		left := distanceSquared(items[i].Center, point)
		right := distanceSquared(items[j].Center, point)
		if left == right {
			if items[i].BBox.Y == items[j].BBox.Y {
				return items[i].BBox.X < items[j].BBox.X
			}
			return items[i].BBox.Y < items[j].BBox.Y
		}
		return left < right
	})
}

func distanceSquared(a screenPoint, b screenPoint) float64 {
	dx := float64(a.X - b.X)
	dy := float64(a.Y - b.Y)
	return math.Pow(dx, 2) + math.Pow(dy, 2)
}
