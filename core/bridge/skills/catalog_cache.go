package skills

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type discoveryCacheEntry struct {
	signature string
	result    DiscoveryResult
}

var discoveryCache sync.Map

func loadDiscoveryCacheEntry(
	key string,
	signature string,
) (DiscoveryResult, bool) {
	cached, ok := discoveryCache.Load(key)
	if !ok {
		return DiscoveryResult{}, false
	}
	entry, ok := cached.(discoveryCacheEntry)
	if !ok || entry.signature != signature {
		return DiscoveryResult{}, false
	}
	return cloneDiscoveryResult(entry.result), true
}

func storeDiscoveryCacheEntry(
	key string,
	signature string,
	result DiscoveryResult,
) {
	discoveryCache.Store(key, discoveryCacheEntry{
		signature: signature,
		result:    cloneDiscoveryResult(result),
	})
}

func discoverySignature(roots []string) (string, bool) {
	normalized := normalizeRoots(roots)
	if len(normalized) == 0 {
		return "", true
	}

	parts := make([]string, 0, len(normalized))
	for _, root := range normalized {
		rootParts, ok := discoverySignaturePartsForRoot(root)
		if !ok {
			return "", false
		}
		parts = append(parts, rootParts...)
	}
	sort.Strings(parts)
	return strings.Join(parts, "\n"), true
}

func discoverySignaturePartsForRoot(root string) ([]string, bool) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{root + "|missing"}, true
		}
		return nil, false
	}
	if !info.IsDir() {
		return []string{
			root + "|file|" + fileInfoSignature(info),
		}, true
	}

	parts := make([]string, 0, 8)
	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, ok := discoverySignatureRelativePath(root, path, entry)
		if !ok {
			return nil
		}
		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}
		parts = append(parts, root+"|"+relPath+"|"+fileInfoSignature(fileInfo))
		return nil
	})
	if walkErr != nil {
		return nil, false
	}
	return parts, true
}

func discoverySignatureRelativePath(
	root string,
	path string,
	entry fs.DirEntry,
) (string, bool) {
	if entry.IsDir() {
		return "", false
	}
	relPath, err := filepath.Rel(root, path)
	if err != nil {
		return "", false
	}
	normalized := filepath.ToSlash(relPath)
	switch {
	case entry.Name() == skillFileName:
		return normalized, true
	case normalized == openAIMetadataPath:
		return normalized, true
	default:
		return "", false
	}
}

func fileInfoSignature(info fs.FileInfo) string {
	return strings.Join([]string{
		strconvFormatInt(info.Size()),
		strconvFormatInt(info.ModTime().UnixNano()),
	}, "|")
}

func strconvFormatInt(value int64) string {
	return strconv.FormatInt(value, 10)
}
