package memory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func readLedgerSegment(path string, repairTail bool) ([]LedgerEvent, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read ledger segment: %w", err)
	}
	if len(data) == 0 {
		return nil, false, nil
	}

	lines := bytes.Split(data, []byte{'\n'})
	events := make([]LedgerEvent, 0, len(lines))
	lastGoodEnd := 0
	offset := 0
	for index, line := range lines {
		lineEnd := offset + len(line)
		if index < len(lines)-1 {
			lineEnd++
		}
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			lastGoodEnd = lineEnd
			offset = lineEnd
			continue
		}

		var event LedgerEvent
		if err := json.Unmarshal(trimmed, &event); err != nil {
			if repairTail && ledgerOnlyTailRemains(lines[index+1:]) {
				if writeErr := os.WriteFile(path, data[:lastGoodEnd], 0o600); writeErr != nil {
					return nil, false, fmt.Errorf("repair ledger tail: %w", writeErr)
				}
				return events, true, nil
			}
			return nil, false, fmt.Errorf("decode ledger event: %w", err)
		}
		events = append(events, normalizeLedgerEvent(event))
		lastGoodEnd = lineEnd
		offset = lineEnd
	}
	return events, false, nil
}

func ledgerOnlyTailRemains(lines [][]byte) bool {
	for _, line := range lines {
		if len(bytes.TrimSpace(line)) > 0 {
			return false
		}
	}
	return true
}

func appendLedgerSegment(path string, events []LedgerEvent) error {
	if len(events) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create ledger segment dir: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open ledger segment: %w", err)
	}
	defer file.Close()

	buffer := make([]byte, 0, len(events)*256)
	for _, event := range events {
		data, err := json.Marshal(normalizeLedgerEvent(event))
		if err != nil {
			return fmt.Errorf("marshal ledger event: %w", err)
		}
		buffer = append(buffer, data...)
		buffer = append(buffer, '\n')
	}
	if _, err := file.Write(buffer); err != nil {
		return fmt.Errorf("append ledger segment: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync ledger segment: %w", err)
	}
	return nil
}

func ledgerSegmentOffsetRange(path string) (int64, int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, fmt.Errorf("stat ledger segment: %w", err)
	}
	if info.Size() <= 0 {
		return 0, 0, nil
	}
	return 0, info.Size(), nil
}

func listLedgerMonthDirs(rootDir string) ([]string, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read ledger root: %w", err)
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if _, err := time.Parse("2006-01", name); err != nil {
			continue
		}
		out = append(out, filepath.Join(rootDir, name))
	}
	sort.Strings(out)
	return out, nil
}
