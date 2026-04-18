package tasks

import "testing"

func TestCloneWorkflowDefinitionClonesStartInputsAndDefaults(t *testing.T) {
	original := &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{
				ID:   "start-node",
				Type: "start",
				Start: &WorkflowStartNode{
					Inputs: []WorkflowInputVariable{
						{
							Name:        "filters",
							Type:        "object",
							Required:    true,
							Default:     []byte(`{"priority":"high"}`),
							Description: "workflow filters",
						},
					},
				},
			},
			{
				ID:   "if-node",
				Type: "if",
				If: &WorkflowIfNode{
					SourceNodeID: "start-node",
					Operator:     "equals",
					Value:        "go",
					TrueNodeID:   "loop-node",
					FalseNodeID:  "end-node",
				},
			},
			{
				ID:   "loop-node",
				Type: "loop",
				Loop: &WorkflowLoopNode{
					MaxIterations: 3,
					BodyNodeID:    "if-node",
					ExitNodeID:    "end-node",
				},
			},
			{ID: "end-node", Type: "end"},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "if-node"},
			{FromNodeID: "if-node", ToNodeID: "loop-node"},
			{FromNodeID: "if-node", ToNodeID: "end-node"},
			{FromNodeID: "loop-node", ToNodeID: "if-node"},
			{FromNodeID: "loop-node", ToNodeID: "end-node"},
		},
	}

	cloned := CloneWorkflowDefinition(original)
	if cloned == nil || cloned.Nodes[0].Start == nil {
		t.Fatalf("expected cloned workflow with start node payload, got %#v", cloned)
	}
	if cloned.Nodes[0].Start == original.Nodes[0].Start {
		t.Fatal("expected start payload to be deep cloned")
	}
	if &cloned.Nodes[0].Start.Inputs[0] == &original.Nodes[0].Start.Inputs[0] {
		t.Fatal("expected start input variable to be deep cloned")
	}
	if len(cloned.Nodes[0].Start.Inputs[0].Default) == 0 {
		t.Fatal("expected cloned default payload")
	}

	original.Nodes[0].Start.Inputs[0].Name = "changed"
	original.Nodes[0].Start.Inputs[0].Default[1] = 'X'
	if cloned.Nodes[0].Start.Inputs[0].Name != "filters" {
		t.Fatalf("cloned input name should not change, got %q", cloned.Nodes[0].Start.Inputs[0].Name)
	}
	if string(cloned.Nodes[0].Start.Inputs[0].Default) != `{"priority":"high"}` {
		t.Fatalf("cloned default payload should not change, got %s", string(cloned.Nodes[0].Start.Inputs[0].Default))
	}

	if cloned.Nodes[1].If == nil || cloned.Nodes[2].Loop == nil {
		t.Fatalf("expected if and loop payloads to be cloned: %#v", cloned.Nodes)
	}
	if cloned.Nodes[1].If == original.Nodes[1].If || cloned.Nodes[2].Loop == original.Nodes[2].Loop {
		t.Fatal("expected if/loop payload to be deep cloned")
	}
}
