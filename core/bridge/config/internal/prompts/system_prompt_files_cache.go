package prompts

import (
	"io/fs"
	"os"
	"strconv"
	"strings"
	"sync"
)

type systemPromptFilesCacheEntry struct {
	signature string
	files     SystemPromptFiles
}

var systemPromptFilesCache sync.Map

func loadCachedSystemPromptFiles(root string) (SystemPromptFiles, bool, error) {
	signature, err := systemPromptFilesSignature(root)
	if err != nil {
		return SystemPromptFiles{}, false, err
	}
	cached, ok := systemPromptFilesCache.Load(strings.TrimSpace(root))
	if !ok {
		return SystemPromptFiles{}, false, nil
	}
	entry, ok := cached.(systemPromptFilesCacheEntry)
	if !ok || entry.signature != signature {
		return SystemPromptFiles{}, false, nil
	}
	return cloneSystemPromptFiles(entry.files), true, nil
}

func storeSystemPromptFilesCache(
	root string,
	files SystemPromptFiles,
) error {
	signature, err := systemPromptFilesSignature(root)
	if err != nil {
		return err
	}
	systemPromptFilesCache.Store(strings.TrimSpace(root), systemPromptFilesCacheEntry{
		signature: signature,
		files:     cloneSystemPromptFiles(files),
	})
	return nil
}

func systemPromptFilesSignature(root string) (string, error) {
	parts := []string{strings.TrimSpace(root)}
	for _, key := range systemPromptFileKeys() {
		path, err := systemPromptFilePath(root, key)
		if err != nil {
			return "", err
		}
		info, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		parts = append(parts, key+"|"+fileInfoSignature(info))
	}
	return strings.Join(parts, "\n"), nil
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

func cloneSystemPromptFiles(files SystemPromptFiles) SystemPromptFiles {
	cloned := SystemPromptFiles{
		CorePrompt: strings.TrimSpace(files.CorePrompt),
	}
	if len(files.PromptLibrary) > 0 {
		cloned.PromptLibrary = append(
			[]SystemPromptLibraryItem(nil),
			files.PromptLibrary...,
		)
	}
	return cloned
}
