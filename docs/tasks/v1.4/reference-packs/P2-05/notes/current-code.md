# Current Tally Code Evidence

These excerpts come from the user-provided `tally.zip` baseline and are local authority.

## internal/workspace/manager.go:1-13

```text
    1: // Package workspace owns the disposable Git worktree lifecycle: it creates a
    2: // worktree at an exact base revision for one execution and removes it
    3: // idempotently afterwards — on success, failure or cancellation.
    4: //
    5: // Agent-generated executable changes never run in the owner primary worktree
    6: // (PRD v1.3 INV-EXEC-01, §13 Workspace, §21.5 Failure Diagnosis): the manager
    7: // refuses to place disposable worktrees inside the owner checkout and refuses
    8: // to destroy the owner primary worktree, whatever the caller claims about a
    9: // given path.
   10: package workspace
   11: 
   12: import (
   13: 	"context"
```

## internal/workspace/manager.go:23-63

```text
   23: )
   24: 
   25: // Sentinel errors. Match with errors.Is, never by string.
   26: var (
   27: 	// ErrPrimaryWorktree: the operation would treat the owner primary
   28: 	// worktree as disposable, either by creating state inside it or by
   29: 	// destroying it.
   30: 	ErrPrimaryWorktree = errors.New("workspace: owner primary worktree is not disposable")
   31: 	// ErrNotRepository: the given path is not a git repository checkout.
   32: 	ErrNotRepository = errors.New("workspace: not a git repository checkout")
   33: 	// ErrNotWorktree: the given path is not a checked-out git worktree
   34: 	// (no .git entry at all).
   35: 	ErrNotWorktree = errors.New("workspace: not a checked-out git worktree")
   36: 	// ErrUnknownRevision: the requested base revision does not resolve to
   37: 	// a commit in the owner repository.
   38: 	ErrUnknownRevision = errors.New("workspace: base revision does not resolve to a commit")
   39: )
   40: 
   41: // worktreeCleanupTimeout bounds the git calls of a teardown. Teardown must
   42: // complete even when the owning operation was cancelled, so it never
   43: // inherits the caller's cancellation — this timeout is the only bound.
   44: const worktreeCleanupTimeout = 30 * time.Second
   45: 
   46: // Manager creates and destroys disposable worktrees for one owner repository.
   47: type Manager struct {
   48: 	// repoDir is the owner repository checkout. It is the reference point
   49: 	// for every git operation and is never used as a disposable target.
   50: 	repoDir string
   51: 	// root is the Tally-owned directory that holds disposable worktrees,
   52: 	// guaranteed to lie outside repoDir.
   53: 	root string
   54: 	git  *gitClient
   55: }
   56: 
   57: // New returns a Manager for the repository checked out at repoDir. Disposable
   58: // worktrees are created under root, which must lie outside the owner
   59: // checkout; New fails with ErrPrimaryWorktree otherwise (INV-EXEC-01).
   60: func New(repoDir, root string) (*Manager, error) {
   61: 	absRepo, err := filepath.Abs(repoDir)
   62: 	if err != nil {
   63: 		return nil, fmt.Errorf("workspace: resolve repo dir: %w", err)
```

## internal/workspace/manager.go:74-121

```text
   74: 			ErrPrimaryWorktree, absRoot, absRepo)
   75: 	}
   76: 	return &Manager{repoDir: absRepo, root: absRoot, git: newGitClient()}, nil
   77: }
   78: 
   79: // Create checks out a new disposable worktree at exactly baseRevision. The
   80: // revision is resolved to a commit first; the worktree is created detached at
   81: // that exact commit, never at a moving ref.
   82: //
   83: // If creation fails — including when the context is cancelled while git runs
   84: // — any partially created state is removed before the error is returned, so a
   85: // failed Create never leaks a worktree.
   86: func (m *Manager) Create(ctx context.Context, baseRevision string) (*Worktree, error) {
   87: 	commit, err := m.git.resolveRevision(ctx, m.repoDir, baseRevision)
   88: 	if err != nil {
   89: 		return nil, err
   90: 	}
   91: 	path, err := m.reservePath(commit)
   92: 	if err != nil {
   93: 		return nil, err
   94: 	}
   95: 	if err := os.MkdirAll(m.root, 0o755); err != nil {
   96: 		return nil, fmt.Errorf("workspace: create disposable root %s: %w", m.root, err)
   97: 	}
   98: 	if err := m.git.worktreeAdd(ctx, m.repoDir, path, commit); err != nil {
   99: 		m.abandon(ctx, path)
  100: 		return nil, err
  101: 	}
  102: 	return &Worktree{Path: path, BaseCommit: commit, repoDir: m.repoDir, git: m.git}, nil
  103: }
  104: 
  105: // Cleanup destroys the worktree previously returned by Create.
  106: //
  107: // Cleanup is idempotent: repeated and concurrent calls converge to the same
  108: // state and a call on an already-removed worktree succeeds. It is equally
  109: // valid after success, after failure (including a worktree directory that
  110: // vanished underneath the manager, leaving stale git metadata) and after
  111: // cancellation: teardown deliberately does not inherit the caller's
  112: // cancellation, so a cancelled operation can always be torn down; internal
  113: // steps are bounded by worktreeCleanupTimeout instead.
  114: //
  115: // Cleanup refuses to destroy the owner primary worktree: if wt.Path is the
  116: // manager's owner checkout or the primary worktree of its repository, it
  117: // returns ErrPrimaryWorktree and touches nothing.
  118: func (m *Manager) Cleanup(ctx context.Context, wt *Worktree) error {
  119: 	if wt == nil {
  120: 		return nil
  121: 	}
```

## internal/workspace/manager.go:133-144

```text
  133: 	}
  134: 	wt.cleaned = true
  135: 	return nil
  136: }
  137: 
  138: // abandon best-effort removes a partially created worktree after a failed
  139: // Create. Errors are swallowed: the original creation error is what the
  140: // caller must see.
  141: func (m *Manager) abandon(ctx context.Context, path string) {
  142: 	_ = (&Worktree{Path: path, repoDir: m.repoDir, git: m.git}).remove(ctx)
  143: }
  144: 
```

## internal/workspace/manager.go:154-164

```text
  154: 
  155: 	mu      sync.Mutex
  156: 	cleaned bool
  157: }
  158: 
  159: // remove tears the worktree down. Steps are detached from the caller's
  160: // cancellation (bounded by worktreeCleanupTimeout) so teardown completes
  161: // after success, failure or cancellation alike.
  162: //
  163: // The directory is removed via `git worktree remove --force`; if git reports
  164: // failure the observed state is verified (worktree list plus directory
```

## internal/workflow/v13.go:1-7

```text
    1: // Package workflow 实现 P1 单 Task 垂直工作流内核(P1-39,PRD §17 P0/P1
    2: // vertical subset、§29 Debug 生命周期、§11.2.1 升级语义、§15 CodegenRun/ChangeSet)。
    3: //
    4: // 内核只做四件事,全部围绕**一个 Task 的顺序执行**:
    5: //
    6: //  1. Run 生命周期:一次 Task 执行 = 有限个顺序 CodegenRun;每个 Run 拥有一个
    7: //     一次性 Git worktree(P1-35)与一个 Runtime 会话(P1-04 协议),从 running
```

## internal/workflow/v13.go:10-24

```text
   10: //  2. 有限 retry/escalation 钩子:迭代次数上界与升级阈值来自 ExecutionPolicy
   11: //     (P1-03),内核只执行不定义;失败迭代达到阈值即升级边界 —— 有升级目标:
   12: //     当前 Run `status: escalated` 终止、新 Run 记 escalated_from(§11.2.1);
   13: //     无目标或目标非法:NEEDS_REVIEW(§29)。内核保证有限性:每 Run 迭代数有
   14: //     上界、升级目标 profile 不得重复(重复视为无目标)、升级链长度有绝对上限。
   15: //  3. 候选 ChangeSet 创建:到达候选固定点的 Run 经 FixCandidate 钩子固定
   16: //     恰好 0..1 个不可变候选快照(P1-05);crash/预算耗尽/升级终止的 Run
   17: //     不产生 ChangeSet(§11.2.1/§15.3)。
   18: //  4. 停止条件:策略列出的 stop condition 一旦被观测,立即终止当前 Run
   19: //     (无 ChangeSet)并把 Task 判为 NEEDS_REVIEW(§11.2/§29)。
   20: //
   21: // 刻意不存在的东西(AC4,§17「不得实现通用 workflow DSL」):Task DAG、
   22: // 就绪选择器、跨 Task 依赖、并行 Task 调度、通用编排语言。一个
   23: // TaskExecution 恰好驱动一个 Task,Execute 恰好可调用一次;任务状态由证据
   24: // 推导,内核不写 TaskExecutionState(§18.1),只产出可观测事实与结果值。
```

## internal/workflow/v13.go:72-82

```text
   72: // TaskStatus 是一次 Task 顺序执行的任务级结果。它只是内核的返回值,不是
   73: // TaskExecutionState 写入(§18.1:状态由证据推导,Agent/内核都不得直写)。
   74: type TaskStatus string
   75: 
   76: const (
   77: 	// TaskCompleted 表示某个 Run 固定了候选 ChangeSet,执行成功收束。
   78: 	TaskCompleted TaskStatus = "completed"
   79: 	// TaskNeedsReview 表示重试/升级阶梯耗尽、升级不可用、Run 中途夭折或触发
   80: 	// 停止条件 —— 按 §29 进入 NEEDS_REVIEW 等待人裁决。
   81: 	TaskNeedsReview TaskStatus = "needs_review"
   82: 	// TaskCancelled 表示调用方取消了执行。
```

## internal/workflow/v13.go:105-117

```text
  105: 	Task taskcontract.TaskContract
  106: 	// Role 是本执行的逻辑角色(PRD §9;如 "coding")。
  107: 	Role string
  108: 	// Policy 是执行边界的唯一权威(PRD §11.2),必须过 Validate。
  109: 	Policy policy.ExecutionPolicy
  110: 	// BaseRevision 所有 Run 的共同 base revision(每次执行从同一基线出发;
  111: 	// ChangeSet 不跨 Run 累积,§15.3)。
  112: 	BaseRevision string
  113: 	// 首个 Run 的归因坐标(§11.2.1:runtime/model/behavior_revision 是 Run 内
  114: 	// 单值事实;升级 Run 的对应值来自 EscalationTarget)。
  115: 	Profile          string
  116: 	Runtime          agentruntime.Runtime
  117: 	Model            string
```

## internal/workflow/v13.go:120-130

```text
  120: 	// (§15.2;内核不解释,只为 RunSink 透传)。
  121: 	VerificationRevision string
  122: 	ContextPackHash      string
  123: 	// Worktrees 提供 Run 的一次性源码 worktree(P1-35)。
  124: 	Worktrees *workspace.RunTable
  125: 	// Sink 接收 Run 生命周期与 ChangeSet 事实(P1-33 的进程内适配点)。
  126: 	Sink RunSink
  127: 	// Hooks 是确定性集成钩子;FixCandidate 必填,其余可空(语义见 Hooks)。
  128: 	Hooks Hooks
  129: 	// Now 返回事实时刻;nil 则用 time.Now。测试注入固定时钟。
  130: 	Now func() time.Time
```

## internal/runstore/changeset.go:94-183

```text
   94: 		out = append(out, cs)
   95: 	}
   96: 	return out, nil
   97: }
   98: 
   99: // AppendSupersededFact 追加一条取代事实(append-only,PRD §15.3,AC1):
  100: // old 被 new 取代。同一事实(同 old/new/reason/时刻)重复追加按 SQL 合同
  101: // 幂等忽略;不同时刻各自成行。
  102: func (s *Store) AppendSupersededFact(ctx context.Context, f changeset.SupersededFact) error {
  103: 	if err := validateSupersededFact(f); err != nil {
  104: 		return err
  105: 	}
  106: 	if err := s.q.InsertChangeSetSupersededFact(ctx, store.InsertChangeSetSupersededFactParams{
  107: 		OldChangesetID: string(f.Old),
  108: 		NewChangesetID: string(f.New),
  109: 		Reason:         string(f.Reason),
  110: 		SupersededAt:   tsPtr(f.At),
  111: 	}); err != nil {
  112: 		return fmt.Errorf("runstore: 追加取代事实 %s→%s: %w", f.Old, f.New, err)
  113: 	}
  114: 	return nil
  115: }
  116: 
  117: // ChangeSetSupersessions 返回某快照参与的取代事实(作为 old 或 new),
  118: // 按发生时刻定序:派生 superseded_by / 取代链的投影源。
  119: func (s *Store) ChangeSetSupersessions(ctx context.Context, id changeset.ChangeSetID) ([]changeset.SupersededFact, error) {
  120: 	rows, err := s.q.ListChangeSetSupersessionsForChangeSet(ctx, string(id))
  121: 	if err != nil {
  122: 		return nil, fmt.Errorf("runstore: 查询 changeset %s 的取代事实: %w", id, err)
  123: 	}
  124: 	out := make([]changeset.SupersededFact, 0, len(rows))
  125: 	for _, r := range rows {
  126: 		f, err := supersededFactFromRow(r.OldChangesetID, r.NewChangesetID, r.Reason, r.SupersededAt)
  127: 		if err != nil {
  128: 			return nil, err
  129: 		}
  130: 		out = append(out, f)
  131: 	}
  132: 	return out, nil
  133: }
  134: 
  135: // AllChangeSetSupersessions 返回全部取代事实(排序确定):
  136: // 取代投影整体重建 / 对账用(INV-STATE-01 可重建)。
  137: func (s *Store) AllChangeSetSupersessions(ctx context.Context) ([]changeset.SupersededFact, error) {
  138: 	rows, err := s.q.ListChangeSetSupersessions(ctx)
  139: 	if err != nil {
  140: 		return nil, fmt.Errorf("runstore: 查询全部取代事实: %w", err)
  141: 	}
  142: 	out := make([]changeset.SupersededFact, 0, len(rows))
  143: 	for _, r := range rows {
  144: 		f, err := supersededFactFromRow(r.OldChangesetID, r.NewChangesetID, r.Reason, r.SupersededAt)
  145: 		if err != nil {
  146: 			return nil, err
  147: 		}
  148: 		out = append(out, f)
  149: 	}
  150: 	return out, nil
  151: }
  152: 
  153: func supersededFactFromRow(oldID, newID, reason string, at pgtype.Timestamptz) (changeset.SupersededFact, error) {
  154: 	f, err := changeset.NewSupersededFact(
  155: 		changeset.ChangeSetID(oldID),
  156: 		changeset.ChangeSetID(newID),
  157: 		changeset.SupersedeReason(reason),
  158: 		tsVal(at),
  159: 	)
  160: 	if err != nil {
  161: 		return changeset.SupersededFact{}, fmt.Errorf("runstore: 重建取代事实 %s→%s(%s): %w", oldID, newID, reason, err)
  162: 	}
  163: 	return f, nil
  164: }
  165: 
  166: func validateSupersededFact(f changeset.SupersededFact) error {
  167: 	switch {
  168: 	case f.Old == "" || f.New == "":
  169: 		return fmt.Errorf("%w: old/new 为空", changeset.ErrBadSupersede)
  170: 	case f.Old == f.New:
  171: 		return fmt.Errorf("%w: old 与 new 相同", changeset.ErrBadSupersede)
  172: 	case f.At.IsZero():
  173: 		return fmt.Errorf("%w: 时刻为零值", changeset.ErrBadSupersede)
  174: 	}
  175: 	switch f.Reason {
  176: 	case changeset.SupersedeEscalation, changeset.SupersedeRework, changeset.SupersedeSpecDefect:
  177: 	default:
  178: 		return fmt.Errorf("%w: 未知原因 %q", changeset.ErrBadSupersede, f.Reason)
  179: 	}
  180: 	return nil
  181: }
  182: 
  183: // changeSetFromRow 从行重建领域快照并核对内容身份(AC3/AC4)。
```