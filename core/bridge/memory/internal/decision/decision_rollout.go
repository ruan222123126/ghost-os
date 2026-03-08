package decision

import (
	"crypto/sha1"
	"encoding/binary"
	"strings"
)

func (d *DecisionService) recipeReuseEnabledForQuery() bool {
	return d != nil && d.Enabled() && d.recipeEnabled && d.recipeReuseEnabled
}

func (d *DecisionService) recipeSelectionThreshold() float64 {
	if d == nil {
		return 0
	}
	return clamp01(d.recipeMinSelectionConfidence)
}

func (d *DecisionService) recipeSuccessThreshold() float64 {
	if d == nil {
		return 0
	}
	return clamp01(d.recipeMinSuccessRate)
}

func (d *DecisionService) recipeRolloutDecision(sessionID string, queryText string, selectionScore float64) (bool, bool) {
	if d == nil || !d.recipeDefaultEnabled || clamp01(selectionScore) < d.recipeSelectionThreshold() {
		return false, false
	}
	percent := d.recipeDefaultGrayPercent
	if percent <= 0 {
		return false, false
	}
	if percent >= 100 {
		return true, true
	}
	grayHit := stableRecipeGrayBucket(sessionID, queryText) < percent
	return grayHit, grayHit
}

func stableRecipeGrayBucket(sessionID string, queryText string) int {
	key := strings.TrimSpace(sessionID) + "|" + normalizeRecallText(queryText)
	if strings.TrimSpace(key) == "|" {
		key = "recipe-gray-default"
	}
	sum := sha1.Sum([]byte(key))
	bucket := binary.BigEndian.Uint32(sum[:4]) % 100
	return int(bucket)
}
