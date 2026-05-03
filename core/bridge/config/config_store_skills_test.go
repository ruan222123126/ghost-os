package config

import (
	"errors"
	"testing"
)

func TestStoreSetSkillEnabledPersistsBlocklist(t *testing.T) {
	_, store := newToolStoreForTest(t)

	if err := store.SetSkillEnabled("skill_repo_release", false); err != nil {
		t.Fatalf("SetSkillEnabled disable: %v", err)
	}
	assertSkillBlocked(t, "skill_repo_release", true)

	if err := store.SetSkillEnabled("skill_repo_release", true); err != nil {
		t.Fatalf("SetSkillEnabled enable: %v", err)
	}
	assertSkillBlocked(t, "skill_repo_release", false)
}

func TestStoreSetSkillEnabledRejectsEmptyID(t *testing.T) {
	_, store := newToolStoreForTest(t)
	if err := store.SetSkillEnabled(" ", false); !errors.Is(err, errSkillIDRequired) {
		t.Fatalf("expected errSkillIDRequired, got %v", err)
	}
}

func assertSkillBlocked(t *testing.T, skillID string, blocked bool) {
	t.Helper()
	fileCfg := mustLoadToolFileConfig(t)
	if hasToolName(fileCfg.SkillBlocklist, skillID) != blocked {
		t.Fatalf("unexpected skill blocklist state for %s", skillID)
	}
}
