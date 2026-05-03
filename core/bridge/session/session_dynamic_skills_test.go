package session

import "testing"

func TestDynamicSkillLoadLifecycle(t *testing.T) {
	sess := NewSession("")
	sess.AdvanceToolTurn(3)

	load := sess.EnsureDynamicSkillLoaded("release_flow", "sfind")
	if load.AlreadyLoaded {
		t.Fatal("newly loaded skill should not report already_loaded")
	}
	if got := sess.VisibleDynamicSkillNames(3); len(got) != 1 || got[0] != "release_flow" {
		t.Fatalf("expected loaded skill to be visible now, got %v", got)
	}

	loads := sess.DynamicSkillLoadsSnapshot()
	if len(loads) != 1 || loads[0].SkillName != "release_flow" {
		t.Fatalf("unexpected skill snapshot: %+v", loads)
	}
	if loads[0].RemainingIdleTurns(sess.TurnIndex, 3) != 3 {
		t.Fatalf("unexpected remaining idle turns: %d", loads[0].RemainingIdleTurns(sess.TurnIndex, 3))
	}

	if !sess.NoteDynamicSkillCall("release_flow") {
		t.Fatal("expected NoteDynamicSkillCall to succeed")
	}
	sess.AdvanceToolTurn(3)
	sess.AdvanceToolTurn(3)
	sess.AdvanceToolTurn(3)
	sess.AdvanceToolTurn(3)
	if got := sess.VisibleDynamicSkillNames(3); len(got) != 0 {
		t.Fatalf("expected expired skill to be pruned, got %v", got)
	}
}

func TestUnloadDynamicSkill(t *testing.T) {
	sess := NewSession("")
	sess.AdvanceToolTurn(3)
	sess.EnsureDynamicSkillLoaded("summarize", "sfind")

	if !sess.UnloadDynamicSkill("summarize") {
		t.Fatal("expected unload to succeed")
	}
	if sess.UnloadDynamicSkill("summarize") {
		t.Fatal("expected unloading missing skill to fail")
	}
}
