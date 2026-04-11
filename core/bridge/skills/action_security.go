package skills

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
)

func normalizeAbsolutePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", ErrSkillPathForbidden
	}
	absolute, err := filepath.Abs(trimmed)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absolute), nil
}

func isValidRelativeSkillPath(path string) bool {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || filepath.IsAbs(trimmed) {
		return false
	}
	cleaned := filepath.Clean(trimmed)
	if cleaned == "." || cleaned != trimmed {
		return false
	}
	return !strings.HasPrefix(cleaned, "..")
}

// EncodeSkillID encodes a decoded skill ID into a managed skill ID string.
func EncodeSkillID(decoded decodedSkillID) string {
	raw := decoded.Source + SkillIDSeparator + decoded.RelativePath
	return SkillIDPrefix + base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeSkillID decodes a managed skill ID string into its components.
func DecodeSkillID(raw string) (decodedSkillID, error) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, SkillIDPrefix) {
		return decodedSkillID{}, ErrInvalidSkillID
	}
	encoded := strings.TrimPrefix(trimmed, SkillIDPrefix)
	decodedBytes, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return decodedSkillID{}, ErrInvalidSkillID
	}
	parts := strings.SplitN(string(decodedBytes), SkillIDSeparator, 2)
	if len(parts) != 2 {
		return decodedSkillID{}, ErrInvalidSkillID
	}
	source := strings.TrimSpace(parts[0])
	relative := strings.TrimSpace(parts[1])
	if source != skillSourceRepo && source != skillSourceUser {
		return decodedSkillID{}, ErrInvalidSkillID
	}
	if !isValidRelativeSkillPath(relative) {
		return decodedSkillID{}, ErrSkillPathForbidden
	}
	return decodedSkillID{Source: source, RelativePath: relative}, nil
}

func findManagedSkill(items []managedSkill, decoded decodedSkillID, rawID string) (managedSkill, error) {
	for _, item := range items {
		if item.ID != rawID {
			continue
		}
		if item.Source != decoded.Source || item.RelativePath != decoded.RelativePath {
			return managedSkill{}, ErrInvalidSkillID
		}
		return item, nil
	}
	return managedSkill{}, ErrSkillNotFound
}

func removeManagedSkill(item managedSkill) error {
	if err := validateManagedSkill(item); err != nil {
		return err
	}
	if _, err := os.Stat(item.Path); err != nil {
		return err
	}
	return os.RemoveAll(item.Path)
}

func validateManagedSkill(item managedSkill) error {
	root, err := normalizeAbsolutePath(item.Root)
	if err != nil {
		return ErrSkillPathForbidden
	}
	path, err := normalizeAbsolutePath(item.Path)
	if err != nil {
		return ErrSkillPathForbidden
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || !isValidRelativeSkillPath(relative) {
		return ErrSkillPathForbidden
	}
	return nil
}
