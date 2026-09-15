package controlplane

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

// TaskRequestFactory converts one frozen Active TaskDAG node into the already
// frozen v1.5.0 TaskExecutionRequest contract. Scheduling deliberately does not
// own runtime/environment policy or invent a second task contract.
type TaskRequestFactory interface {
	BuildTaskExecutionRequest(context.Context, ProjectSnapshot, engineering.TaskNode) (taskexecution.TaskExecutionRequest, error)
}

// TaskRequestFactoryFunc adapts a function to TaskRequestFactory.
type TaskRequestFactoryFunc func(context.Context, ProjectSnapshot, engineering.TaskNode) (taskexecution.TaskExecutionRequest, error)

func (f TaskRequestFactoryFunc) BuildTaskExecutionRequest(ctx context.Context, snapshot ProjectSnapshot, node engineering.TaskNode) (taskexecution.TaskExecutionRequest, error) {
	return f(ctx, snapshot, node)
}

// LocalScheduler is the v1.5.1 default application-local coordinator over the
// persisted Active TaskDAG. It owns no durable state of its own and uses
// Service.Dispatch as the only execution/acceptance path.
type LocalScheduler struct {
	Service       *Service
	Requests      TaskRequestFactory
	MaxConcurrent int

	runMu sync.Mutex
}

func NewLocalScheduler(service *Service, requests TaskRequestFactory, maxConcurrent int) (*LocalScheduler, error) {
	if service == nil {
		return nil, errors.New("controlplane: scheduler service is required")
	}
	if requests == nil {
		return nil, errors.New("controlplane: scheduler request factory is required")
	}
	if maxConcurrent <= 0 {
		return nil, errors.New("controlplane: scheduler max concurrency must be positive")
	}
	return &LocalScheduler{Service: service, Requests: requests, MaxConcurrent: maxConcurrent}, nil
}

// RecoverStartup converts durable RUNNING tasks into INTERRUPTED after a
// process restart. It never silently returns interrupted work to READY; an
// explicit RestartTask/task policy decision remains required by the frozen
// v1.5.0 restart semantics.
func (s *LocalScheduler) RecoverStartup(ctx context.Context, projectID engineering.ID) (ProjectSnapshot, error) {
	s.Service.mu.Lock()
	defer s.Service.mu.Unlock()

	snap, err := s.Service.Repo.Load(ctx, projectID)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	if snap.ActiveDAG == nil {
		return ProjectSnapshot{}, ErrDAGNotActive
	}

	dag := *snap.ActiveDAG
	changed := false
	for _, node := range append([]engineering.TaskNode(nil), dag.Nodes...) {
		if node.State != engineering.TaskRunning {
			continue
		}
		dag, err = dag.Transition(node.ID, engineering.TaskInterrupted, "")
		if err != nil {
			return ProjectSnapshot{}, err
		}
		if err := s.Service.record(&snap, "task.recovered_interrupted", node.ID, "scheduler_startup"); err != nil {
			return ProjectSnapshot{}, err
		}
		changed = true
	}
	dag = dag.RefreshReadiness()
	if !reflect.DeepEqual(*snap.ActiveDAG, dag) {
		changed = true
	}
	if !changed {
		return snap, nil
	}
	snap.ActiveDAG = &dag
	if err := s.Service.save(ctx, &snap); err != nil {
		return ProjectSnapshot{}, err
	}
	return snap, nil
}

// RunOnce refreshes persisted readiness and dispatches one deterministic local
// batch. If any READY node is not parallel-eligible, exactly one such node is
// dispatched and no scheduler-owned sibling overlaps it. Otherwise up to
// MaxConcurrent parallel-eligible nodes may execute concurrently.
func (s *LocalScheduler) RunOnce(ctx context.Context, projectID engineering.ID) ([]DispatchResult, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()

	snap, err := s.refreshReadiness(ctx, projectID)
	if err != nil {
		return nil, err
	}
	ready := snap.ActiveDAG.ReadyNodes()
	if len(ready) == 0 {
		return nil, nil
	}

	batch := selectSchedulerBatch(ready, s.MaxConcurrent)
	results := make([]DispatchResult, len(batch))
	errs := make([]error, len(batch))
	var wg sync.WaitGroup
	for i, node := range batch {
		i, node := i, node
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, buildErr := s.Requests.BuildTaskExecutionRequest(ctx, snap, node)
			if buildErr != nil {
				errs[i] = fmt.Errorf("controlplane: build execution request for %s: %w", node.ID, buildErr)
				return
			}
			if req.TaskID != node.ID {
				errs[i] = fmt.Errorf("controlplane: request factory returned task %s for DAG node %s", req.TaskID, node.ID)
				return
			}
			results[i], errs[i] = s.Service.Dispatch(ctx, projectID, req)
		}()
	}
	wg.Wait()

	var joined []error
	for _, err := range errs {
		if err != nil {
			joined = append(joined, err)
		}
	}
	return results, errors.Join(joined...)
}

// Run drains deterministic READY batches until the project is idle or a batch
// reports an execution/build failure. Operational task failures are persisted by
// Service.Dispatch before the error is returned.
func (s *LocalScheduler) Run(ctx context.Context, projectID engineering.ID) error {
	if _, err := s.RecoverStartup(ctx, projectID); err != nil {
		return err
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		results, err := s.RunOnce(ctx, projectID)
		if err != nil {
			return err
		}
		if len(results) == 0 {
			return nil
		}
	}
}

func (s *LocalScheduler) refreshReadiness(ctx context.Context, projectID engineering.ID) (ProjectSnapshot, error) {
	s.Service.mu.Lock()
	defer s.Service.mu.Unlock()

	snap, err := s.Service.Repo.Load(ctx, projectID)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	if snap.ActiveDAG == nil {
		return ProjectSnapshot{}, ErrDAGNotActive
	}
	refreshed := snap.ActiveDAG.RefreshReadiness()
	if reflect.DeepEqual(*snap.ActiveDAG, refreshed) {
		return snap, nil
	}
	snap.ActiveDAG = &refreshed
	if err := s.Service.record(&snap, "task_dag.readiness_refreshed", "", "local_scheduler"); err != nil {
		return ProjectSnapshot{}, err
	}
	if err := s.Service.save(ctx, &snap); err != nil {
		return ProjectSnapshot{}, err
	}
	return snap, nil
}

func selectSchedulerBatch(ready []engineering.TaskNode, max int) []engineering.TaskNode {
	for _, node := range ready {
		if !node.ParallelEligible {
			return []engineering.TaskNode{node}
		}
	}
	if len(ready) > max {
		ready = ready[:max]
	}
	return append([]engineering.TaskNode(nil), ready...)
}
