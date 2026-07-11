package group

type Plan struct {
	Nodes        map[string]Node
	EntryGroupID string
	ControlNext  map[string]string
	GroupMembers map[string][]string
}

type PlanBuilder struct{}

func (PlanBuilder) Build(definition *Definition) (Plan, error) {
	graph, err := buildGraphData(definition)
	if err != nil {
		return Plan{}, err
	}
	if err := validateControlDegrees(graph); err != nil {
		return Plan{}, err
	}
	if err := validateConnectivity(graph); err != nil {
		return Plan{}, err
	}
	if err := validateMembers(graph); err != nil {
		return Plan{}, err
	}
	return graph.plan(), nil
}
