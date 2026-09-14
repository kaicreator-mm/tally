package engineering

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

type TaskState string

const (
	TaskPending     TaskState = "pending"
	TaskReady       TaskState = "ready"
	TaskRunning     TaskState = "running"
	TaskBlocked     TaskState = "blocked"
	TaskInterrupted TaskState = "interrupted"
	TaskAccepted    TaskState = "accepted"
	TaskDone        TaskState = "done"
)

type AcceptanceState string

const (
	AcceptancePending  AcceptanceState = "pending"
	AcceptanceAccepted AcceptanceState = "accepted"
	AcceptanceRejected AcceptanceState = "rejected"
)

type TaskNode struct {
	ID                       ID              `json:"id"`
	Dependencies             []ID            `json:"dependencies,omitempty"`
	State                    TaskState       `json:"state"`
	BlockedReason            string          `json:"blocked_reason,omitempty"`
	ParallelEligible         bool            `json:"parallel_eligible"`
	RoleRequirement          string          `json:"role_requirement,omitempty"`
	Risk                     string          `json:"risk,omitempty"`
	EngineeringGate          string          `json:"engineering_gate,omitempty"`
	RuntimeRequirements      []string        `json:"runtime_requirements,omitempty"`
	EnvironmentRequirements  []string        `json:"environment_requirements,omitempty"`
	VerificationRequirements []string        `json:"verification_requirements,omitempty"`
	Acceptance               AcceptanceState `json:"acceptance"`
}
type TaskDAGProposal struct {
	RevisionID ID         `json:"revision_id"`
	Nodes      []TaskNode `json:"nodes"`
}
type ActiveTaskDAG struct {
	RevisionID ID         `json:"revision_id"`
	Nodes      []TaskNode `json:"nodes"`
}

var (
	ErrDAGEmpty              = errors.New("engineering: task DAG is empty")
	ErrDAGCycle              = errors.New("engineering: task DAG contains a cycle")
	ErrIllegalTaskTransition = errors.New("engineering: illegal task transition")
)

func ActivateTaskDAG(p TaskDAGProposal) (ActiveTaskDAG, error) {
	if err := p.RevisionID.Validate("task DAG revision"); err != nil {
		return ActiveTaskDAG{}, err
	}
	if len(p.Nodes) == 0 {
		return ActiveTaskDAG{}, ErrDAGEmpty
	}
	nodes := cloneNodes(p.Nodes)
	byID := map[ID]int{}
	for i := range nodes {
		n := &nodes[i]
		if err := n.ID.Validate("task node"); err != nil {
			return ActiveTaskDAG{}, err
		}
		if _, ok := byID[n.ID]; ok {
			return ActiveTaskDAG{}, fmt.Errorf("engineering: duplicate task node %s", n.ID)
		}
		byID[n.ID] = i
		n.State = TaskPending
		n.Acceptance = AcceptancePending
		n.BlockedReason = ""
		seen := map[ID]bool{}
		for _, dep := range n.Dependencies {
			if dep == n.ID || seen[dep] {
				return ActiveTaskDAG{}, fmt.Errorf("engineering: invalid dependency %s -> %s", n.ID, dep)
			}
			seen[dep] = true
		}
	}
	for _, n := range nodes {
		for _, dep := range n.Dependencies {
			if _, ok := byID[dep]; !ok {
				return ActiveTaskDAG{}, fmt.Errorf("engineering: task %s missing dependency %s", n.ID, dep)
			}
		}
	}
	if hasCycle(nodes, byID) {
		return ActiveTaskDAG{}, ErrDAGCycle
	}
	return (ActiveTaskDAG{RevisionID: p.RevisionID, Nodes: nodes}).RefreshReadiness(), nil
}
func hasCycle(nodes []TaskNode, byID map[ID]int) bool {
	state := map[ID]uint8{}
	var visit func(ID) bool
	visit = func(id ID) bool {
		if state[id] == 1 {
			return true
		}
		if state[id] == 2 {
			return false
		}
		state[id] = 1
		for _, dep := range nodes[byID[id]].Dependencies {
			if visit(dep) {
				return true
			}
		}
		state[id] = 2
		return false
	}
	for _, n := range nodes {
		if state[n.ID] == 0 && visit(n.ID) {
			return true
		}
	}
	return false
}
func (d ActiveTaskDAG) RefreshReadiness() ActiveTaskDAG {
	out := ActiveTaskDAG{RevisionID: d.RevisionID, Nodes: cloneNodes(d.Nodes)}
	states := map[ID]TaskState{}
	for _, n := range out.Nodes {
		states[n.ID] = n.State
	}
	for i := range out.Nodes {
		n := &out.Nodes[i]
		if n.State == TaskDone || n.State == TaskAccepted || n.State == TaskRunning {
			continue
		}
		var unmet []ID
		for _, dep := range n.Dependencies {
			if states[dep] != TaskDone {
				unmet = append(unmet, dep)
			}
		}
		if len(unmet) == 0 {
			n.State = TaskReady
			n.BlockedReason = ""
		} else {
			sort.Slice(unmet, func(i, j int) bool { return unmet[i] < unmet[j] })
			parts := make([]string, len(unmet))
			for j, id := range unmet {
				parts[j] = string(id)
			}
			n.State = TaskBlocked
			n.BlockedReason = "waiting for: " + strings.Join(parts, ",")
		}
	}
	return out
}
func (d ActiveTaskDAG) ReadyNodes() []TaskNode {
	var out []TaskNode
	for _, n := range d.Nodes {
		if n.State == TaskReady {
			out = append(out, cloneNode(n))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func (d ActiveTaskDAG) ParallelReady() []TaskNode {
	var out []TaskNode
	for _, n := range d.ReadyNodes() {
		if n.ParallelEligible {
			out = append(out, n)
		}
	}
	return out
}
func (d ActiveTaskDAG) Transition(id ID, to TaskState, reason string) (ActiveTaskDAG, error) {
	out := ActiveTaskDAG{RevisionID: d.RevisionID, Nodes: cloneNodes(d.Nodes)}
	idx := -1
	for i := range out.Nodes {
		if out.Nodes[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return d, fmt.Errorf("engineering: unknown task %s", id)
	}
	from := out.Nodes[idx].State
	if !legalTaskTransition(from, to) {
		return d, fmt.Errorf("%w: %s -> %s", ErrIllegalTaskTransition, from, to)
	}
	if to == TaskReady {
		r := out.RefreshReadiness()
		for _, n := range r.Nodes {
			if n.ID == id && n.State != TaskReady {
				return d, fmt.Errorf("%w: dependencies not satisfied", ErrIllegalTaskTransition)
			}
		}
	}
	if to == TaskBlocked && strings.TrimSpace(reason) == "" {
		return d, errors.New("engineering: blocked transition requires reason")
	}
	out.Nodes[idx].State = to
	out.Nodes[idx].BlockedReason = ""
	if to == TaskBlocked {
		out.Nodes[idx].BlockedReason = reason
	}
	if to == TaskAccepted || to == TaskDone {
		out.Nodes[idx].Acceptance = AcceptanceAccepted
	}
	return out.RefreshReadiness(), nil
}
func legalTaskTransition(from, to TaskState) bool {
	switch from {
	case TaskPending:
		return to == TaskReady || to == TaskBlocked
	case TaskBlocked:
		return to == TaskReady || to == TaskBlocked
	case TaskReady:
		return to == TaskRunning || to == TaskBlocked
	case TaskRunning:
		return to == TaskInterrupted || to == TaskBlocked || to == TaskAccepted
	case TaskInterrupted:
		return to == TaskReady || to == TaskBlocked
	case TaskAccepted:
		return to == TaskDone || to == TaskBlocked
	}
	return false
}
func cloneNodes(in []TaskNode) []TaskNode {
	out := make([]TaskNode, len(in))
	for i, n := range in {
		out[i] = cloneNode(n)
	}
	return out
}
func cloneNode(n TaskNode) TaskNode {
	n.Dependencies = append([]ID(nil), n.Dependencies...)
	n.RuntimeRequirements = append([]string(nil), n.RuntimeRequirements...)
	n.EnvironmentRequirements = append([]string(nil), n.EnvironmentRequirements...)
	n.VerificationRequirements = append([]string(nil), n.VerificationRequirements...)
	return n
}
