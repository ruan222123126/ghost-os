package session

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"

	"ghost-os/bridge/llm"
)

type messageRecord struct {
	Index      int
	TokenCount int
	Message    llm.Message
}

type messagePageRange struct {
	limit           int
	effectiveBefore int
	start           int
	before          *int
}

func loadHotWindowMessagesTx(tx *sql.Tx, sessionID string) ([]llm.Message, int, int, error) {
	rows, err := tx.Query(`
SELECT idx, token_count, message_json
FROM session_messages
WHERE session_id = ?
ORDER BY idx DESC
LIMIT ?`, sessionID, hotWindowMaxMessages)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("load hot window: %w", err)
	}
	defer rows.Close()

	descending := make([]messageRecord, 0, hotWindowMaxMessages)
	totalTokens := 0
	for rows.Next() {
		record, err := scanMessageRecord(rows)
		if err != nil {
			return nil, 0, 0, err
		}
		if totalTokens+record.TokenCount > hotWindowMaxTokens && len(descending) > 0 {
			break
		}
		descending = append(descending, record)
		totalTokens += record.TokenCount
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, fmt.Errorf("load hot window: %w", err)
	}
	if len(descending) == 0 {
		return nil, 0, 0, nil
	}

	ascending := reverseMessageRecords(descending)
	if shouldExtendHotWindowToUserBoundary(ascending) {
		prefix, err := loadHotWindowPrefixToUserBoundaryTx(tx, sessionID, ascending[0].Index)
		if err != nil {
			return nil, 0, 0, err
		}
		if len(prefix) > 0 {
			ascending = append(prefix, ascending...)
			totalTokens += sumMessageRecordTokens(prefix)
		}
	}

	messages := make([]llm.Message, 0, len(ascending))
	windowStart := ascending[0].Index
	for _, record := range ascending {
		messages = append(messages, record.Message)
	}
	return messages, windowStart, totalTokens, nil
}

func loadMessagePageTx(tx *sql.Tx, sessionID string, messageCount int, params PageParams) (MessagePage, error) {
	queryRange := resolveMessagePageRange(messageCount, params)
	if queryRange.effectiveBefore <= queryRange.start {
		return queryRange.emptyPage(), nil
	}

	messages, err := loadIndexedMessagesTx(tx, sessionID, queryRange)
	if err != nil {
		return MessagePage{}, err
	}
	return buildLoadedMessagePage(messages, queryRange), nil
}

func resolveMessagePageRange(messageCount int, params PageParams) messagePageRange {
	queryRange := messagePageRange{
		limit:  params.Limit,
		before: cloneIntPointer(params.Before),
	}
	if queryRange.limit <= 0 {
		queryRange.limit = DefaultDetailPageLimit
	}
	queryRange.effectiveBefore = messageCount
	if params.Before != nil {
		queryRange.effectiveBefore = minPageInt(maxInt(*params.Before, 0), messageCount)
	}
	queryRange.start = maxInt(queryRange.effectiveBefore-queryRange.limit, 0)
	return queryRange
}

func (r messagePageRange) emptyPage() MessagePage {
	return MessagePage{
		Limit:         r.limit,
		Before:        r.before,
		HasMoreBefore: r.start > 0,
	}
}

func loadIndexedMessagesTx(tx *sql.Tx, sessionID string, queryRange messagePageRange) ([]IndexedMessage, error) {
	rows, err := tx.Query(`
SELECT idx, token_count, message_json
FROM session_messages
WHERE session_id = ? AND idx >= ? AND idx < ?
ORDER BY idx ASC`, sessionID, queryRange.start, queryRange.effectiveBefore)
	if err != nil {
		return nil, fmt.Errorf("load session page: %w", err)
	}
	defer rows.Close()

	messages := make([]IndexedMessage, 0, queryRange.effectiveBefore-queryRange.start)
	for rows.Next() {
		record, err := scanMessageRecord(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, IndexedMessage{
			Index:   record.Index,
			Message: record.Message,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load session page: %w", err)
	}
	return messages, nil
}

func buildLoadedMessagePage(messages []IndexedMessage, queryRange messagePageRange) MessagePage {
	page := MessagePage{
		Messages:      messages,
		Limit:         queryRange.limit,
		Before:        queryRange.before,
		HasMoreBefore: queryRange.start > 0,
	}
	if len(messages) == 0 {
		return page
	}

	startIndex := messages[0].Index
	endIndex := messages[len(messages)-1].Index
	page.StartIndex = &startIndex
	page.EndIndex = &endIndex
	if page.HasMoreBefore {
		nextBefore := startIndex
		page.NextBefore = &nextBefore
	}
	return page
}

func appendedMessages(sess *Session, existing *sessionRecord) ([]llm.Message, int, error) {
	baseCount := 0
	if existing != nil {
		baseCount = existing.MessageCount
		if sess.persistedMessageCount != baseCount {
			return nil, 0, ErrSessionNotAppendOnly
		}
	}
	if sess.MessageCount < baseCount {
		return nil, 0, ErrSessionNotAppendOnly
	}

	appendedCount := sess.MessageCount - baseCount
	if len(sess.persistedMessages) > len(sess.Messages) {
		return nil, 0, ErrSessionNotAppendOnly
	}
	if len(sess.persistedMessages) > 0 {
		currentPrefix := sess.Messages[:len(sess.persistedMessages)]
		if !reflect.DeepEqual(currentPrefix, sess.persistedMessages) {
			return nil, 0, ErrSessionNotAppendOnly
		}
	}
	if len(sess.Messages) != len(sess.persistedMessages)+appendedCount {
		return nil, 0, ErrSessionNotAppendOnly
	}
	if appendedCount == 0 {
		return nil, 0, nil
	}

	appended := llm.CloneMessages(sess.Messages[len(sess.Messages)-appendedCount:])
	return appended, estimateMessagesTokens(appended), nil
}

func estimateMessagesTokens(messages []llm.Message) int {
	total := 0
	for _, message := range messages {
		total += EstimateTokens(message)
	}
	return total
}

func scanMessageRecord(scanner interface{ Scan(...any) error }) (messageRecord, error) {
	var (
		index      int
		tokenCount int
		rawMessage string
	)
	if err := scanner.Scan(&index, &tokenCount, &rawMessage); err != nil {
		return messageRecord{}, fmt.Errorf("scan session message: %w", err)
	}

	var message llm.Message
	if err := json.Unmarshal([]byte(rawMessage), &message); err != nil {
		return messageRecord{}, fmt.Errorf("%w: session message[%d]: %v", ErrSessionCorrupted, index, err)
	}
	return messageRecord{
		Index:      index,
		TokenCount: tokenCount,
		Message:    message,
	}, nil
}

func reverseMessageRecords(descending []messageRecord) []messageRecord {
	ascending := make([]messageRecord, 0, len(descending))
	for i := len(descending) - 1; i >= 0; i-- {
		ascending = append(ascending, descending[i])
	}
	return ascending
}

func shouldExtendHotWindowToUserBoundary(records []messageRecord) bool {
	if len(records) == 0 {
		return false
	}
	role := records[0].Message.Role
	return role != llm.RoleSystem && role != llm.RoleUser
}

func loadHotWindowPrefixToUserBoundaryTx(tx *sql.Tx, sessionID string, beforeIndex int) ([]messageRecord, error) {
	rows, err := tx.Query(`
SELECT idx, token_count, message_json
FROM session_messages
WHERE session_id = ? AND idx < ?
ORDER BY idx DESC`, sessionID, beforeIndex)
	if err != nil {
		return nil, fmt.Errorf("load hot window prefix: %w", err)
	}
	defer rows.Close()

	descending := make([]messageRecord, 0, 16)
	for rows.Next() {
		record, err := scanMessageRecord(rows)
		if err != nil {
			return nil, err
		}
		descending = append(descending, record)
		if record.Message.Role == llm.RoleUser || record.Message.Role == llm.RoleSystem {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load hot window prefix: %w", err)
	}
	return reverseMessageRecords(descending), nil
}

func sumMessageRecordTokens(records []messageRecord) int {
	total := 0
	for _, record := range records {
		total += record.TokenCount
	}
	return total
}

func encodeMessageJSON(message llm.Message) (string, error) {
	encoded, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("encode session message: %w", err)
	}
	return string(encoded), nil
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func minPageInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
