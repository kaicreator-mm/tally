package multiagent

import (
	"sort"
	"strings"

	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

// SelectReady returns a deterministic, bounded set of ready tasks that are
// safe to execute together. DAG readiness remains authoritative; this function
// only adds parallel-eligibility and conflict analysis.
func SelectReady(dag engineering.ActiveTaskDAG, requests map[engineering.ID]taskexecution.TaskExecutionRequest, policies map[engineering.ID]TaskPolicy, limit int) []engineering.ID {
	if limit < 1 {
		limit = 1
	}
	ready := dag.ReadyNodes()
	sort.Slice(ready, func(i, j int) bool { return ready[i].ID < ready[j].ID })
	selected := make([]engineering.ID, 0, limit)
	used := map[string]bool{}
	for _, node := range ready {
		req, ok := requests[node.ID]
		if !ok {
			continue
		}
		if len(selected) > 0 && !node.ParallelEligible {
			continue
		}
		if len(selected) == 1 {
			first, _ := findTaskNode(dag, selected[0])
			if !first.ParallelEligible {
				break
			}
		}
		keys := conflictKeys(req, policies[node.ID])
		if overlaps(used, keys) {
			continue
		}
		selected = append(selected, node.ID)
		for _, key := range keys {
			used[key] = true
		}
		if len(selected) == limit {
			break
		}
	}
	return selected
}

func conflictKeys(req taskexecution.TaskExecutionRequest, policy TaskPolicy) []string {
	keys := append([]string(nil), policy.Normalize().ConflictKeys...)
	if path := strings.TrimSpace(req.Workspace.Path); path != "" {
		keys = append(keys, "workspace-path:"+path)
	} else if identity := strings.TrimSpace(req.Workspace.Identity); identity != "" {
		keys = append(keys, "workspace:"+string(req.Workspace.Environment)+":"+req.Workspace.Target+":"+identity)
	}
	return uniqueStrings(keys)
}

func overlaps(used map[string]bool, keys []string) bool {
	for _, key := range keys {
		if used[key] {
			return true
		}
	}
	return false
}

func findTaskNode(dag engineering.ActiveTaskDAG, id engineering.ID) (engineering.TaskNode, bool) {
	for _, node := range dag.Nodes {
		if node.ID == id {
			return node, true
		}
	}
	return engineering.TaskNode{}, false
}
