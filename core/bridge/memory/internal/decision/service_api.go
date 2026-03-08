package decision

import "time"

func (d *DecisionService) CaptureOnTurnEnabled() bool {
	return d != nil && d.captureOnTurn
}

func (d *DecisionService) MaxHits() int {
	if d == nil {
		return 0
	}
	return d.maxHits
}

func (d *DecisionService) DebugEnabled() bool {
	return d != nil && d.debugEnabled
}

func (d *DecisionService) HasDistiller() bool {
	return d != nil && d.distiller != nil
}

func (d *DecisionService) StartDistiller() {
	if d != nil && d.distiller != nil {
		d.distiller.Start()
	}
}

func (d *DecisionService) StopDistiller() {
	if d != nil && d.distiller != nil {
		d.distiller.Stop()
	}
}

func (d *DecisionService) DistillerStopCh() <-chan struct{} {
	if d == nil || d.distiller == nil {
		return nil
	}
	return d.distiller.stop
}

func (d *DecisionService) DistillAll(namespace string) (DecisionDistillStats, error) {
	if d == nil || d.distiller == nil {
		return DecisionDistillStats{Namespace: distillStatsNamespace(namespace)}, nil
	}
	return d.distiller.DistillAll(namespace)
}

func (d *DecisionService) UpsertMemo(memo DecisionMemo) (bool, error) {
	if d == nil || d.store == nil {
		return false, nil
	}
	return d.store.UpsertMemo(memo)
}

func (d *DecisionService) UpsertRecipe(recipe DecisionRecipe) (bool, error) {
	if d == nil || d.store == nil {
		return false, nil
	}
	return d.store.UpsertRecipe(recipe)
}

func (d *DecisionService) UpsertRecipeRun(run RecipeRun) (bool, error) {
	if d == nil || d.store == nil {
		return false, nil
	}
	return d.store.UpsertRecipeRun(run)
}

func (d *DecisionService) ListRecipeRuns(namespace string) []RecipeRun {
	if d == nil || d.store == nil {
		return nil
	}
	return d.store.ListRecipeRuns(namespace)
}

func (d *DecisionService) Recipe(id string) (DecisionRecipe, bool) {
	if d == nil || d.store == nil {
		return DecisionRecipe{}, false
	}
	return d.store.Recipe(id)
}

func (d *DecisionService) ExplainMemoLineage(memoID string) (MemoLineageExplanation, bool) {
	if d == nil || d.store == nil {
		return MemoLineageExplanation{}, false
	}
	return d.store.ExplainMemoLineage(memoID)
}

func (d *DecisionService) ExplainRecipeLineage(recipeID string) (RecipeLineageExplanation, bool) {
	if d == nil || d.store == nil {
		return RecipeLineageExplanation{}, false
	}
	return d.store.ExplainRecipeLineage(recipeID)
}

func (d *DecisionService) ExplainRecipeRunLineage(runID string) (RecipeRunLineageExplanation, bool) {
	if d == nil || d.store == nil {
		return RecipeRunLineageExplanation{}, false
	}
	return d.store.ExplainRecipeRunLineage(runID)
}

func (d *DecisionService) RecordNow() time.Time { return time.Now().UTC() }
