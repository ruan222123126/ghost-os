package graph

const (
	GraphNodeTypePerson     = "person"
	GraphNodeTypeProject    = "project"
	GraphNodeTypeOrg        = "org"
	GraphNodeTypeRepo       = "repo"
	GraphNodeTypeTopic      = "topic"
	GraphNodeTypeTech       = "tech"
	GraphNodeTypePreference = "preference"

	GraphPredicateOwnerOf   = "owner_of"
	GraphPredicateMemberOf  = "member_of"
	GraphPredicateUses      = "uses"
	GraphPredicatePrefers   = "prefers"
	GraphPredicateAvoids    = "avoids"
	GraphPredicateDependsOn = "depends_on"
	GraphPredicateRelatedTo = "related_to"
	GraphPredicateBlockedBy = "blocked_by"
	GraphPredicateWorksOn   = "works_on"

	GraphStatusActive     = "active"
	GraphStatusConflicted = "conflicted"
	GraphStatusSuperseded = "superseded"

	defaultGraphMaxHops       = 1
	defaultGraphMaxHits       = 6
	defaultGraphMinConfidence = 0.72
	defaultGraphNamespace     = "default"
	defaultGraphEvidenceLimit = 6
	defaultGraphAliasPathName = "aliases.json"
	defaultGraphNodesPathName = "nodes.json"
	defaultGraphEdgesPathName = "edges.json"
)

var allowedGraphPredicates = map[string]struct{}{
	GraphPredicateOwnerOf:   {},
	GraphPredicateMemberOf:  {},
	GraphPredicateUses:      {},
	GraphPredicatePrefers:   {},
	GraphPredicateAvoids:    {},
	GraphPredicateDependsOn: {},
	GraphPredicateRelatedTo: {},
	GraphPredicateBlockedBy: {},
	GraphPredicateWorksOn:   {},
}

var singleValueGraphPredicates = map[string]struct{}{
	GraphPredicateOwnerOf: {},
}
