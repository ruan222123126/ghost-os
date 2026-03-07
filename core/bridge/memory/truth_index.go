package memory

import (
	"sort"
	"strings"
)

type truthIndex struct {
	objectsByID            map[string]MemoryObject
	claimsByID             map[string]MemoryClaim
	claimsByObjectID       map[string][]MemoryClaim
	claimsByIntentKey      map[string][]MemoryClaim
	claimsByEntityID       map[string][]MemoryClaim
	claimsByAnchorKey      map[string][]MemoryClaim
	claimsByConstraintType map[string][]MemoryClaim
	claimsByRiskType       map[string][]MemoryClaim
	objectsBySourceRef     map[string][]MemoryObject
	objectsByType          map[string][]MemoryObject
}

func buildTruthIndex(objects map[string]MemoryObject, claims map[string]MemoryClaim) truthIndex {
	idx := truthIndex{
		objectsByID:            make(map[string]MemoryObject, len(objects)),
		claimsByID:             make(map[string]MemoryClaim, len(claims)),
		claimsByObjectID:       make(map[string][]MemoryClaim),
		claimsByIntentKey:      make(map[string][]MemoryClaim),
		claimsByEntityID:       make(map[string][]MemoryClaim),
		claimsByAnchorKey:      make(map[string][]MemoryClaim),
		claimsByConstraintType: make(map[string][]MemoryClaim),
		claimsByRiskType:       make(map[string][]MemoryClaim),
		objectsBySourceRef:     make(map[string][]MemoryObject),
		objectsByType:          make(map[string][]MemoryObject),
	}
	for _, object := range objects {
		normalized := normalizeMemoryObject(object)
		if normalized.ObjectID == "" {
			continue
		}
		idx.objectsByID[normalized.ObjectID] = normalized
		appendTruthObjectBucket(idx.objectsByType, truthIndexKey(normalized.ObjectType), normalized)
		for _, ref := range truthObjectSourceRefs(normalized) {
			appendTruthObjectBucket(idx.objectsBySourceRef, truthSourceRefFingerprint(ref), normalized)
		}
	}
	if len(claims) == 0 {
		claims = truthClaimsFromObjects(objects)
	}
	for _, claim := range claims {
		normalized := normalizeMemoryClaim(claim, claim.ObjectID)
		if normalized.ClaimID == "" || normalized.ObjectID == "" {
			continue
		}
		idx.claimsByID[normalized.ClaimID] = normalized
		idx.claimsByObjectID[normalized.ObjectID] = append(idx.claimsByObjectID[normalized.ObjectID], normalized)
		appendTruthClaimBucket(idx.claimsByIntentKey, normalized.IntentKey, normalized)
		appendTruthClaimBucket(idx.claimsByEntityID, normalized.EntityID, normalized)
		appendTruthClaimBucket(idx.claimsByAnchorKey, normalized.AnchorKey, normalized)
		appendTruthClaimBucket(idx.claimsByConstraintType, normalized.ConstraintType, normalized)
		appendTruthClaimBucket(idx.claimsByRiskType, normalized.RiskType, normalized)
	}
	truthSortIndex(idx.claimsByObjectID)
	truthSortIndex(idx.claimsByIntentKey)
	truthSortIndex(idx.claimsByEntityID)
	truthSortIndex(idx.claimsByAnchorKey)
	truthSortIndex(idx.claimsByConstraintType)
	truthSortIndex(idx.claimsByRiskType)
	truthSortObjectIndex(idx.objectsBySourceRef)
	truthSortObjectIndex(idx.objectsByType)
	return idx
}

func appendTruthClaimBucket(index map[string][]MemoryClaim, key string, claim MemoryClaim) {
	normalizedKey := truthIndexKey(key)
	if normalizedKey == "" {
		return
	}
	index[normalizedKey] = append(index[normalizedKey], claim)
}

func appendTruthObjectBucket(index map[string][]MemoryObject, key string, object MemoryObject) {
	normalizedKey := truthIndexKey(key)
	if normalizedKey == "" {
		return
	}
	index[normalizedKey] = append(index[normalizedKey], object)
}

func truthSortIndex(index map[string][]MemoryClaim) {
	for key := range index {
		claims := index[key]
		sort.SliceStable(claims, func(i, j int) bool {
			if claims[i].ObjectID != claims[j].ObjectID {
				return claims[i].ObjectID < claims[j].ObjectID
			}
			return claims[i].ClaimID < claims[j].ClaimID
		})
		index[key] = claims
	}
}

func truthSortObjectIndex(index map[string][]MemoryObject) {
	for key := range index {
		objects := index[key]
		sort.SliceStable(objects, func(i, j int) bool {
			return objects[i].ObjectID < objects[j].ObjectID
		})
		index[key] = objects
	}
}

func truthIndexKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func truthObjectSourceRefs(object MemoryObject) []SourceRef {
	refs := append([]SourceRef(nil), object.SourceRefs...)
	for _, claim := range object.Claims {
		refs = append(refs, claim.SourceRefs...)
	}
	for _, evidence := range object.RawEvidence {
		refs = append(refs, evidence.SourceRefs...)
	}
	for _, ref := range object.EmbeddingRefs {
		refs = append(refs, ref.SourceRefs...)
	}
	return normalizeSourceRefs(refs)
}

func truthClaimsFromObjects(objects map[string]MemoryObject) map[string]MemoryClaim {
	if len(objects) == 0 {
		return nil
	}
	out := make(map[string]MemoryClaim)
	for _, object := range objects {
		for _, claim := range object.Claims {
			normalized := normalizeMemoryClaim(claim, object.ObjectID)
			if normalized.ClaimID == "" {
				continue
			}
			out[normalized.ClaimID] = normalized
		}
	}
	return out
}
