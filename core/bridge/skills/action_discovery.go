package skills

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (h *ActionHandler) discoverManagedSkills() ([]managedSkill, skillRoots, error) {
	roots, err := h.resolveSkillRoots()
	if err != nil {
		return nil, skillRoots{}, err
	}
	items, err := discoverManagedSkillsFromRoots(roots)
	if err != nil {
		return nil, skillRoots{}, err
	}
	return items, roots, nil
}

func (h *ActionHandler) resolveSkillRoots() (skillRoots, error) {
	if h == nil || h.store == nil {
		return skillRoots{}, errors.New("config store is not configured")
	}
	cfg, err := h.store.Config()
	if err != nil {
		return skillRoots{}, fmt.Errorf("load config: %w", err)
	}
	repoRoot, err := resolveManagedRepoRoot(cfg.ProjectRoot)
	if err != nil {
		return skillRoots{}, err
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return skillRoots{}, fmt.Errorf("resolve user home: %w", err)
	}
	return skillRoots{
		Repo: filepath.Join(repoRoot, ".agents", "skills"),
		User: filepath.Join(strings.TrimSpace(homeDir), ".ghost-os", "skills"),
	}, nil
}

func resolveManagedRepoRoot(projectRoot string) (string, error) {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve current directory: %w", err)
		}
		root = cwd
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}
	return filepath.Clean(absolute), nil
}

func discoverManagedSkillsFromRoots(roots skillRoots) ([]managedSkill, error) {
	repoSkills, err := discoverManagedSkillsBySource(skillSourceRepo, roots.Repo)
	if err != nil {
		return nil, err
	}
	userSkills, err := discoverManagedSkillsBySource(skillSourceUser, roots.User)
	if err != nil {
		return nil, err
	}
	items := append(repoSkills, userSkills...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].Source == items[j].Source {
			return items[i].Name < items[j].Name
		}
		return items[i].Source < items[j].Source
	})
	return items, nil
}

func discoverManagedSkillsBySource(source string, root string) ([]managedSkill, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("%w: %s", ErrSkillSourceNotFound, source)
	}
	result := NewCatalogWithRoots([]string{root}).ForceReload()
	if err := firstSkillDiscoveryError(source, result.Errors); err != nil {
		return nil, err
	}
	items := make([]managedSkill, 0, len(result.Skills))
	for _, item := range result.Skills {
		managed, err := managedSkillFromDiscovery(source, root, item)
		if err != nil {
			return nil, err
		}
		items = append(items, managed)
	}
	return items, nil
}

func firstSkillDiscoveryError(source string, errorsList []DiscoveryError) error {
	if len(errorsList) == 0 {
		return nil
	}
	first := errorsList[0]
	return fmt.Errorf("discover skills (%s) failed at %s: %s", source, first.Path, first.Error)
}

func managedSkillFromDiscovery(source string, root string, item Skill) (managedSkill, error) {
	rootPath, err := normalizeAbsolutePath(root)
	if err != nil {
		return managedSkill{}, err
	}
	skillPath, err := normalizeAbsolutePath(filepath.Dir(item.Path))
	if err != nil {
		return managedSkill{}, err
	}
	relative, err := filepath.Rel(rootPath, skillPath)
	if err != nil {
		return managedSkill{}, err
	}
	if !isValidRelativeSkillPath(relative) {
		return managedSkill{}, ErrSkillPathForbidden
	}
	decoded := decodedSkillID{Source: source, RelativePath: relative}
	return managedSkill{
		ID:           EncodeSkillID(decoded),
		Name:         strings.TrimSpace(item.Name),
		Description:  strings.TrimSpace(item.Description),
		Path:         skillPath,
		Source:       source,
		Root:         rootPath,
		RelativePath: relative,
	}, nil
}

func refreshManagedSkillCatalog(roots skillRoots) error {
	rootList := managedSkillRootsList(roots)
	for _, root := range rootList {
		result := NewCatalogWithRoots([]string{root}).ForceReload()
		if err := firstSkillDiscoveryError(root, result.Errors); err != nil {
			return err
		}
	}
	if len(rootList) == 0 {
		return nil
	}
	result := NewCatalogWithRoots(rootList).ForceReload()
	return firstSkillDiscoveryError("combined", result.Errors)
}

func managedSkillRootsList(roots skillRoots) []string {
	items := make([]string, 0, 2)
	if root := strings.TrimSpace(roots.Repo); root != "" {
		items = append(items, root)
	}
	if root := strings.TrimSpace(roots.User); root != "" {
		items = append(items, root)
	}
	return items
}
