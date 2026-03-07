package memory

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

type GraphExtractOptions struct {
	Namespace string
	SessionID string
	SourceID  string
	TraceID   string
	Timestamp time.Time
}

type graphRulePattern struct {
	predicate   string
	confidence  float64
	regex       *regexp.Regexp
	subjectIdx  int
	objectIdx   int
	subjectType string
	objectType  string
	reverse     bool
}

var graphFactRulePatterns = []graphRulePattern{
	{
		predicate:   GraphPredicateOwnerOf,
		confidence:  0.92,
		regex:       regexp.MustCompile(`^([\p{Han}A-Za-z0-9_.\-/]{1,40})\s*(?:负责|跟进|lead[s]?|owns?)\s*([\p{Han}A-Za-z0-9_.\-/]{1,60})$`),
		subjectIdx:  1,
		objectIdx:   2,
		subjectType: GraphNodeTypePerson,
		objectType:  GraphNodeTypeProject,
	},
	{
		predicate:   GraphPredicateOwnerOf,
		confidence:  0.92,
		regex:       regexp.MustCompile(`^([\p{Han}A-Za-z0-9_.\-/]{1,60})\s*的负责人(?:是|为)?\s*([\p{Han}A-Za-z0-9_.\-/]{1,40})$`),
		subjectIdx:  2,
		objectIdx:   1,
		subjectType: GraphNodeTypePerson,
		objectType:  GraphNodeTypeProject,
	},
	{
		predicate:   GraphPredicateWorksOn,
		confidence:  0.84,
		regex:       regexp.MustCompile(`(?i)^([\p{Han}A-Za-z0-9_.\-/]{1,40})\s+(?:works on|working on)\s+([\p{Han}A-Za-z0-9_.\-/]{1,60})$`),
		subjectIdx:  1,
		objectIdx:   2,
		subjectType: GraphNodeTypePerson,
		objectType:  GraphNodeTypeProject,
	},
	{
		predicate:   GraphPredicateMemberOf,
		confidence:  0.82,
		regex:       regexp.MustCompile(`(?i)^([\p{Han}A-Za-z0-9_.\-/]{1,40})\s+(?:member of|belongs to)\s+([\p{Han}A-Za-z0-9_.\-/]{1,60})$`),
		subjectIdx:  1,
		objectIdx:   2,
		subjectType: GraphNodeTypePerson,
		objectType:  GraphNodeTypeOrg,
	},
	{
		predicate:   GraphPredicateUses,
		confidence:  0.8,
		regex:       regexp.MustCompile(`(?i)^([\p{Han}A-Za-z0-9_.\-/]{1,40})\s+(?:uses?|using)\s+([\p{Han}A-Za-z0-9_.\-/]{1,60})$`),
		subjectIdx:  1,
		objectIdx:   2,
		subjectType: GraphNodeTypeTopic,
		objectType:  GraphNodeTypeTech,
	},
	{
		predicate:   GraphPredicateUses,
		confidence:  0.8,
		regex:       regexp.MustCompile(`^([\p{Han}A-Za-z0-9_.\-/]{1,40})\s*(?:使用|用)\s*([\p{Han}A-Za-z0-9_.\-/]{1,60})$`),
		subjectIdx:  1,
		objectIdx:   2,
		subjectType: GraphNodeTypeTopic,
		objectType:  GraphNodeTypeTech,
	},
	{
		predicate:   GraphPredicateDependsOn,
		confidence:  0.81,
		regex:       regexp.MustCompile(`(?i)^([\p{Han}A-Za-z0-9_.\-/]{1,40})\s+(?:depends on|relies on)\s+([\p{Han}A-Za-z0-9_.\-/]{1,60})$`),
		subjectIdx:  1,
		objectIdx:   2,
		subjectType: GraphNodeTypeProject,
		objectType:  GraphNodeTypeTopic,
	},
	{
		predicate:   GraphPredicateDependsOn,
		confidence:  0.81,
		regex:       regexp.MustCompile(`^([\p{Han}A-Za-z0-9_.\-/]{1,40})\s*依赖\s*([\p{Han}A-Za-z0-9_.\-/]{1,60})$`),
		subjectIdx:  1,
		objectIdx:   2,
		subjectType: GraphNodeTypeProject,
		objectType:  GraphNodeTypeTopic,
	},
	{
		predicate:   GraphPredicateBlockedBy,
		confidence:  0.8,
		regex:       regexp.MustCompile(`(?i)^([\p{Han}A-Za-z0-9_.\-/]{1,40})\s+(?:blocked by)\s+([\p{Han}A-Za-z0-9_.\-/]{1,60})$`),
		subjectIdx:  1,
		objectIdx:   2,
		subjectType: GraphNodeTypeProject,
		objectType:  GraphNodeTypeTopic,
	},
	{
		predicate:   GraphPredicateBlockedBy,
		confidence:  0.8,
		regex:       regexp.MustCompile(`^([\p{Han}A-Za-z0-9_.\-/]{1,40})\s*被\s*([\p{Han}A-Za-z0-9_.\-/]{1,60})\s*(?:阻塞|卡住)$`),
		subjectIdx:  1,
		objectIdx:   2,
		subjectType: GraphNodeTypeProject,
		objectType:  GraphNodeTypeTopic,
	},
}

// GraphExtractor 先做规则抽取，再可选接 LLM 结构化抽取。
type GraphExtractor struct {
	worker        GraphFactExtractor
	useWorker     bool
	minConfidence float64
}

func NewGraphExtractor(summarizer Summarizer, useWorker bool, minConfidence float64) *GraphExtractor {
	var worker GraphFactExtractor
	if extractor, ok := summarizer.(GraphFactExtractor); ok {
		worker = extractor
	}
	return &GraphExtractor{
		worker:        worker,
		useWorker:     useWorker,
		minConfidence: clamp01(maxFloat(minConfidence, defaultGraphMinConfidence)),
	}
}

func (e *GraphExtractor) ExtractFromMessages(messages []llm.Message, opts GraphExtractOptions) []GraphFact {
	facts := e.extractRuleFacts(messages, opts)
	if e.useWorker && e.worker != nil {
		if workerFacts, err := e.worker.ExtractGraphFacts(messages); err == nil {
			facts = append(facts, e.normalizeWorkerFacts(workerFacts, opts)...)
		}
	}
	return dedupeGraphFacts(facts, e.minConfidence)
}

func (e *GraphExtractor) ExtractFromEntries(entries []MemoryEntry, opts GraphExtractOptions) []GraphFact {
	if len(entries) == 0 {
		return nil
	}
	messages := evolutionMessages(entries)
	if opts.Timestamp.IsZero() {
		opts.Timestamp = latestEntryTimestamp(entries)
	}
	return e.ExtractFromMessages(messages, opts)
}

func (e *GraphExtractor) ExtractFromMarkdownNode(node MarkdownNode, namespace string) []GraphFact {
	opts := GraphExtractOptions{
		Namespace: namespace,
		SessionID: strings.TrimSpace(node.SessionID),
		SourceID:  "markdown:" + strings.TrimSpace(node.ID),
		Timestamp: markdownNodeTimestamp(node),
	}
	messages := make([]llm.Message, 0, 2)
	if summary := strings.TrimSpace(node.Summary); summary != "" {
		messages = append(messages, llm.Message{Role: llm.RoleAssistant, Text: summary})
	}
	if content := strings.TrimSpace(node.Content); content != "" {
		messages = append(messages, llm.Message{Role: llm.RoleAssistant, Text: content})
	}
	facts := e.ExtractFromMessages(messages, opts)
	for _, anchor := range node.Anchors {
		fact, ok := graphFactFromAnchor(anchor, opts)
		if !ok {
			continue
		}
		facts = append(facts, fact)
	}
	return dedupeGraphFacts(facts, e.minConfidence)
}

func (e *GraphExtractor) extractRuleFacts(messages []llm.Message, opts GraphExtractOptions) []GraphFact {
	if len(messages) == 0 {
		return nil
	}
	out := make([]GraphFact, 0, len(messages)*2)
	for _, msg := range messages {
		text := strings.TrimSpace(messageToContent(msg))
		if text == "" {
			continue
		}
		for _, segment := range splitGraphSegments(text) {
			out = append(out, extractFirstPersonGraphFacts(segment, opts)...)
			for _, pattern := range graphFactRulePatterns {
				matches := pattern.regex.FindStringSubmatch(segment)
				if len(matches) == 0 {
					continue
				}
				subject := normalizeGraphEntityValue(matches[pattern.subjectIdx])
				object := normalizeGraphEntityValue(matches[pattern.objectIdx])
				if subject == "" || object == "" {
					continue
				}
				out = append(out, GraphFact{
					Namespace:   opts.Namespace,
					Subject:     subject,
					SubjectType: pattern.subjectType,
					Predicate:   pattern.predicate,
					Object:      object,
					ObjectType:  pattern.objectType,
					Confidence:  pattern.confidence,
					Snippet:     summarizeLine(segment, 220),
					Timestamp:   effectiveGraphTimestamp(opts.Timestamp),
					SessionID:   strings.TrimSpace(opts.SessionID),
					SourceID:    strings.TrimSpace(opts.SourceID),
					TraceID:     strings.TrimSpace(opts.TraceID),
				})
			}
		}
	}
	return out
}

func (e *GraphExtractor) normalizeWorkerFacts(workerFacts []GraphFact, opts GraphExtractOptions) []GraphFact {
	out := make([]GraphFact, 0, len(workerFacts))
	for _, rawFact := range workerFacts {
		fact := normalizeGraphFact(rawFact)
		fact.Namespace = firstNonEmpty(fact.Namespace, opts.Namespace)
		fact.SessionID = firstNonEmpty(fact.SessionID, opts.SessionID)
		fact.SourceID = firstNonEmpty(fact.SourceID, opts.SourceID)
		fact.TraceID = firstNonEmpty(fact.TraceID, opts.TraceID)
		if fact.Timestamp.IsZero() {
			fact.Timestamp = effectiveGraphTimestamp(opts.Timestamp)
		}
		if fact.Predicate == "" || fact.Subject == "" || fact.Object == "" || strings.TrimSpace(fact.Snippet) == "" {
			continue
		}
		out = append(out, fact)
	}
	return out
}

func graphFactFromAnchor(anchor MemoryAnchor, opts GraphExtractOptions) (GraphFact, bool) {
	normalized := normalizeAnchor(anchor)
	if normalized.Value == "" {
		return GraphFact{}, false
	}
	subject := "self"
	subjectType := GraphNodeTypePerson
	predicate := ""
	objectType := GraphNodeTypePreference
	switch normalized.Type {
	case MemoryAnchorPreference:
		predicate = GraphPredicatePrefers
		objectType = GraphNodeTypeTech
	case MemoryAnchorAvoidance:
		predicate = GraphPredicateAvoids
	case MemoryAnchorIdentity:
		predicate = GraphPredicateRelatedTo
		objectType = GraphNodeTypeTopic
	default:
		return GraphFact{}, false
	}
	return GraphFact{
		Namespace:   opts.Namespace,
		Subject:     subject,
		SubjectType: subjectType,
		Predicate:   predicate,
		Object:      normalizeGraphEntityValue(normalized.Value),
		ObjectType:  objectType,
		Confidence:  clamp01(maxFloat(normalized.Weight, 0.78)),
		Snippet:     summarizeLine(firstNonEmpty(normalized.Reason, normalized.Value), 220),
		Timestamp:   effectiveGraphTimestamp(opts.Timestamp),
		AnchorKeys:  normalizeGraphAliases([]string{normalized.Key}),
		SessionID:   strings.TrimSpace(opts.SessionID),
		SourceID:    strings.TrimSpace(opts.SourceID),
		TraceID:     strings.TrimSpace(opts.TraceID),
	}, true
}

func splitGraphSegments(text string) []string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}
	splitter := regexp.MustCompile(`[\n。！？!?;；]+`)
	parts := splitter.Split(trimmed, -1)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		segment := strings.TrimSpace(part)
		if segment == "" {
			continue
		}
		out = append(out, segment)
	}
	return out
}

func extractFirstPersonGraphFacts(segment string, opts GraphExtractOptions) []GraphFact {
	out := make([]GraphFact, 0, 2)
	patterns := []struct {
		predicate  string
		confidence float64
		regex      *regexp.Regexp
		objectType string
	}{
		{GraphPredicatePrefers, 0.84, regexp.MustCompile(`(?i)(?:^|\b)(?:i|we)\s+(?:prefer|like to use|mostly use|lean toward)\s+([A-Za-z0-9_.\-/]+)`), GraphNodeTypeTech},
		{GraphPredicatePrefers, 0.84, regexp.MustCompile(`我(?:更)?偏好\s*([^\s，。；;]+)`), GraphNodeTypeTech},
		{GraphPredicatePrefers, 0.8, regexp.MustCompile(`(?:以后都用|以后用|都用)\s*([^\s，。；;]+)`), GraphNodeTypeTech},
		{GraphPredicateAvoids, 0.9, regexp.MustCompile(`(?i)(?:^|\b)(?:i|we)\s+(?:avoid|don't use|do not use|won't use)\s+([A-Za-z0-9_.\-/]+)`), GraphNodeTypeTech},
		{GraphPredicateAvoids, 0.9, regexp.MustCompile(`(?:不要(?:再)?用|别用)\s*([^\s，。；;]+)`), GraphNodeTypeTech},
	}
	for _, pattern := range patterns {
		matches := pattern.regex.FindStringSubmatch(strings.TrimSpace(segment))
		if len(matches) < 2 {
			continue
		}
		object := normalizeGraphEntityValue(matches[1])
		if object == "" {
			continue
		}
		out = append(out, GraphFact{
			Namespace:   opts.Namespace,
			Subject:     "self",
			SubjectType: GraphNodeTypePerson,
			Predicate:   pattern.predicate,
			Object:      object,
			ObjectType:  pattern.objectType,
			Confidence:  pattern.confidence,
			Snippet:     summarizeLine(segment, 220),
			Timestamp:   effectiveGraphTimestamp(opts.Timestamp),
			SessionID:   strings.TrimSpace(opts.SessionID),
			SourceID:    strings.TrimSpace(opts.SourceID),
			TraceID:     strings.TrimSpace(opts.TraceID),
		})
	}
	return out
}

func normalizeGraphEntityValue(raw string) string {
	trimmed := cleanGraphDisplayName(raw)
	if trimmed == "" {
		return ""
	}
	switch strings.ToLower(trimmed) {
	case "i", "we", "me", "user", "myself", "我", "我们":
		return "self"
	default:
		return trimmed
	}
}

func effectiveGraphTimestamp(ts time.Time) time.Time {
	if ts.IsZero() {
		return time.Now().UTC()
	}
	return ts.UTC()
}

func dedupeGraphFacts(facts []GraphFact, minConfidence float64) []GraphFact {
	if len(facts) == 0 {
		return nil
	}
	threshold := clamp01(minConfidence)
	if threshold <= 0 {
		threshold = defaultGraphMinConfidence
	}
	seen := make(map[string]GraphFact, len(facts))
	for _, rawFact := range facts {
		fact := normalizeGraphFact(rawFact)
		if fact.Predicate == "" || fact.Subject == "" || fact.Object == "" {
			continue
		}
		if fact.Confidence > 0 && fact.Confidence < threshold {
			continue
		}
		key := strings.Join([]string{
			normalizeGraphNamespace(fact.Namespace),
			normalizeGraphLookup(fact.Subject),
			fact.Predicate,
			normalizeGraphLookup(fact.Object),
			strings.TrimSpace(fact.SourceID),
		}, "|")
		if existing, ok := seen[key]; ok && existing.Confidence >= fact.Confidence {
			continue
		}
		seen[key] = fact
	}
	out := make([]GraphFact, 0, len(seen))
	for _, fact := range seen {
		out = append(out, fact)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Timestamp.Equal(out[j].Timestamp) {
			return out[i].Predicate < out[j].Predicate
		}
		return out[i].Timestamp.Before(out[j].Timestamp)
	})
	return out
}
