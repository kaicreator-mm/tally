package multiagent

import (
	"reflect"
	"testing"

	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

func schedulerDAG(t *testing.T) engineering.ActiveTaskDAG {
	t.Helper()
	dag, err := engineering.ActivateTaskDAG(engineering.TaskDAGProposal{RevisionID: "dag", Nodes: []engineering.TaskNode{
		{ID: "a", ParallelEligible: true},
		{ID: "b", ParallelEligible: true},
		{ID: "c", Dependencies: []engineering.ID{"a"}, ParallelEligible: true},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return dag
}

func schedulerRequest(id engineering.ID, path string) taskexecution.TaskExecutionRequest {
	req := testRequest(id)
	req.Workspace.Path = path
	return req
}

func TestSelectReadyAllowsBoundedNonConflictingParallelTasks(t *testing.T) {
	dag := schedulerDAG(t)
	got := SelectReady(dag, map[engineering.ID]taskexecution.TaskExecutionRequest{
		"a": schedulerRequest("a", "/tmp/a"),
		"b": schedulerRequest("b", "/tmp/b"),
		"c": schedulerRequest("c", "/tmp/c"),
	}, nil, 2)
	if !reflect.DeepEqual(got, []engineering.ID{"a", "b"}) {
		t.Fatalf("selected=%v", got)
	}
}

func TestSelectReadyRejectsConflictAndDependencyRace(t *testing.T) {
	dag := schedulerDAG(t)
	policies := map[engineering.ID]TaskPolicy{
		"a": {ConflictKeys: []string{"file:shared.go"}},
		"b": {ConflictKeys: []string{"file:shared.go"}},
	}
	got := SelectReady(dag, map[engineering.ID]taskexecution.TaskExecutionRequest{
		"a": schedulerRequest("a", "/tmp/a"),
		"b": schedulerRequest("b", "/tmp/b"),
		"c": schedulerRequest("c", "/tmp/c"),
	}, policies, 3)
	if !reflect.DeepEqual(got, []engineering.ID{"a"}) {
		t.Fatalf("selected=%v", got)
	}
}

func TestSelectReadyEnforcesConcurrencyLimit(t *testing.T) {
	dag := schedulerDAG(t)
	got := SelectReady(dag, map[engineering.ID]taskexecution.TaskExecutionRequest{
		"a": schedulerRequest("a", "/tmp/a"),
		"b": schedulerRequest("b", "/tmp/b"),
	}, nil, 1)
	if !reflect.DeepEqual(got, []engineering.ID{"a"}) {
		t.Fatalf("selected=%v", got)
	}
}
