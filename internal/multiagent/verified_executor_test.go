package multiagent

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tally/internal/agentruntime"
	"tally/internal/artifacts"
	"tally/internal/engineering"
	"tally/internal/gitwriter"
	"tally/internal/intake"
	"tally/internal/intakefreeze"
	"tally/internal/store"
	"tally/internal/taskexecution"
	"tally/internal/verification"
)

// verified_executor_test.go 钉住 v1.5.4 角色执行的 enforcement 合同:
//   - reviewer/verifier/specialist 是只读角色:必须先有冻结候选、运行期漂移
//     被拒绝,且没有角色指令的裸请求被拒收(不得绕过 enforcement);
//   - implementer/repairer 走原样委托执行(现状行为不变),产物经既有
//     FreezeWorktree 冻结为候选,不经验证权威。

const verifiedTaskID = "TASK-0042"

type verifiedFixture struct {
	repoDir   string
	freezeSHA string
	workspace string
}

// newVerifiedFixture 走生产冻结路径构造可执行夹具:注册表基线提交 →
// intakefreeze 冻结三件套 → FreezeCommit 单提交落库 → 在冻结提交上挂一个
// 可丢弃链接 worktree 作为角色执行 workspace。
func newVerifiedFixture(t *testing.T) verifiedFixture {
	t.Helper()
	repoDir := t.TempDir()
	gitT(t, repoDir, "init")
	gitT(t, repoDir, "config", "user.name", "verified-test")
	gitT(t, repoDir, "config", "user.email", "verified@test.local")
	writeT(t, filepath.Join(repoDir, ".tally", "project.yaml"), "project: verified-proj\nentities:\n  - id: server\n    type: service\n    path: cmd/server\n")
	gitT(t, repoDir, "add", ".")
	gitT(t, repoDir, "commit", "-m", "base: registry")
	base := strings.TrimSpace(gitT(t, repoDir, "rev-parse", "HEAD"))

	frozen, err := intakefreeze.Freeze(intakefreeze.ConfirmedDraft{
		TaskID:    verifiedTaskID,
		Profile:   "standard",
		SourceRef: "GitHub Issue #142",
		Draft: intake.Draft{
			TaskType:  intake.TaskTypeBugFix,
			Summary:   "verified executor fixture",
			Objective: "logout must invalidate the existing session",
			Scope: intake.Scope{
				AllowedPaths:   []string{"internal/session/**"},
				ForbiddenPaths: []string{"internal/legacy/**"},
			},
			Acceptance: []intake.AcceptanceClause{
				{ID: "AC-01", Statement: "requests with the old session token return 401"},
			},
		},
		RiskHints: []string{"auth_session"},
	}, intakefreeze.VerificationPlan{
		Evidence: []intakefreeze.EvidenceDeclaration{{
			AcceptanceID: "AC-01",
			Kind:         verification.EvidenceTest,
			Path:         "internal/session/logout_test.go",
		}},
	})
	if err != nil {
		t.Fatalf("intakefreeze.Freeze: %v", err)
	}
	res, err := artifacts.FreezeCommit(context.Background(), repoDir, artifacts.FreezeCommitRequest{
		ExpectedBase: base,
		Artifacts:    frozen,
		Message:      "freeze: " + verifiedTaskID,
		Signature:    &gitwriter.Signature{Name: "verified-test", Email: "verified@test.local"},
	})
	if err != nil {
		t.Fatalf("FreezeCommit: %v", err)
	}

	workspace := filepath.Join(t.TempDir(), "workspace")
	gitT(t, repoDir, "worktree", "add", "--detach", workspace, res.Commit)
	if _, err := artifacts.LoadBundle(repoDir, res.Commit, verifiedTaskID); err != nil {
		t.Fatalf("LoadBundle at freeze commit: %v", err)
	}
	return verifiedFixture{repoDir: repoDir, freezeSHA: res.Commit, workspace: workspace}
}

func gitT(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out, errBuf strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(errBuf.String()))
	}
	return out.String()
}

func writeT(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// workspaceChange 模拟角色在可丢弃 worktree 里留下的内容变更。
func workspaceChange(t *testing.T, req taskexecution.TaskExecutionRequest, name, content string) {
	t.Helper()
	writeT(t, filepath.Join(req.Workspace.Path, name), content)
}

type recordingDelegate struct {
	calls     int
	lastReq   taskexecution.TaskExecutionRequest
	onExecute func(req taskexecution.TaskExecutionRequest)
}

func (d *recordingDelegate) Execute(ctx context.Context, req taskexecution.TaskExecutionRequest) (taskexecution.TaskExecutionResult, error) {
	d.calls++
	d.lastReq = req
	if d.onExecute != nil {
		d.onExecute(req)
	}
	return taskexecution.TaskExecutionResult{
		Status:      taskexecution.ExecutionCompleted,
		Workspace:   req.Workspace,
		Runtime:     req.Runtime,
		Environment: req.Environment,
		AgentClaim:  agentruntime.AgentRunResult{RunID: "run-1", SessionID: "session-1", Status: agentruntime.RunCompleted, Summary: "stub run"},
	}, nil
}

func (d *recordingDelegate) Cancel(id engineering.ID) error { return nil }

func newVerifiedExecutor(t *testing.T, fx verifiedFixture, delegate taskexecution.AgentExecutor) *VerifiedRoleExecutor {
	t.Helper()
	executor, err := NewVerifiedRoleExecutor(VerifiedRoleExecutorConfig{
		Delegate:     delegate,
		Repo:         fx.repoDir,
		WorktreeRoot: filepath.Join(t.TempDir(), "verify-worktrees"),
		EnvName:      "verified-test",
		Runtime:      agentruntime.RuntimeDescriptor{Name: "test", Version: "1"},
		Queries:      &store.Queries{},
	})
	if err != nil {
		t.Fatalf("NewVerifiedRoleExecutor: %v", err)
	}
	return executor
}

func verifiedRequest(fx verifiedFixture, instruction string) taskexecution.TaskExecutionRequest {
	return taskexecution.TaskExecutionRequest{
		TaskID:             engineering.ID(verifiedTaskID),
		FrozenTaskContract: engineering.ArtifactRef{ID: engineering.ID(verifiedTaskID + ":contract"), Kind: "task_contract", URI: "tally://task/" + verifiedTaskID + "/contract"},
		Runtime:            taskexecution.RuntimeRef{ID: "rt-test"},
		Environment:        taskexecution.EnvironmentRef{Kind: taskexecution.EnvironmentLocal, Target: fx.workspace},
		Workspace: taskexecution.WorkspaceRef{
			Environment:  taskexecution.EnvironmentLocal,
			Target:       fx.workspace,
			Identity:     "ws-verified",
			Path:         fx.workspace,
			BaseRevision: fx.freezeSHA,
		},
		Policy:      taskexecution.ExecutionPolicy{FallbackToFresh: true},
		Instruction: instruction,
	}
}

// 生产者角色:委托执行原样保留(现状行为不变),产物冻结为候选,不经验证权威。
func TestVerifiedRoleExecutorProducerRolesDelegateUnchangedAndFreezeCandidate(t *testing.T) {
	for _, role := range []Role{RoleImplementer, RoleRepairer} {
		t.Run(string(role), func(t *testing.T) {
			fx := newVerifiedFixture(t)
			delegate := &recordingDelegate{onExecute: func(req taskexecution.TaskExecutionRequest) {
				workspaceChange(t, req, "internal/session/feature.txt", "produced\n")
			}}
			executor := newVerifiedExecutor(t, fx, delegate)

			instruction := "Implement " + verifiedTaskID + ".\n\n" + roleDirective(role, "")
			result, err := executor.Execute(context.Background(), verifiedRequest(fx, instruction))
			if err != nil {
				t.Fatalf("Execute(%s): %v", role, err)
			}
			if delegate.calls != 1 {
				t.Fatalf("delegate calls = %d, want 1", delegate.calls)
			}
			if delegate.lastReq.Instruction != instruction {
				t.Errorf("producer instruction mutated:\n got %q\nwant %q", delegate.lastReq.Instruction, instruction)
			}
			if result.Status != taskexecution.ExecutionCompleted {
				t.Errorf("status = %q, want completed", result.Status)
			}
			if result.CandidateRef == "" {
				t.Error("producer run must freeze a candidate (empty CandidateRef)")
			}
			if len(result.EvidenceRefs) != 0 {
				t.Errorf("producer run must not produce verification evidence, got %v", result.EvidenceRefs)
			}
		})
	}
}

// 只读角色运行期漂移被拒绝:候选工作区在评审/验证期间被改动即 fail-closed。
func TestVerifiedRoleExecutorRejectsReadOnlyRoleMutation(t *testing.T) {
	fx := newVerifiedFixture(t)
	// 先落一份候选状态(等价于实现角色留下的产物),只读角色才有可评审对象。
	writeT(t, filepath.Join(fx.workspace, "internal", "session", "candidate.txt"), "candidate\n")

	delegate := &recordingDelegate{onExecute: func(req taskexecution.TaskExecutionRequest) {
		workspaceChange(t, req, "internal/session/drift.txt", "mutated during review\n")
	}}
	executor := newVerifiedExecutor(t, fx, delegate)

	_, err := executor.Execute(context.Background(), verifiedRequest(fx, roleDirective(RoleReviewer, "")))
	if !errors.Is(err, ErrReadOnlyRoleMutatedCandidate) {
		t.Fatalf("reviewer drift err = %v, want ErrReadOnlyRoleMutatedCandidate", err)
	}
	if delegate.calls != 1 {
		t.Fatalf("delegate calls = %d, want 1", delegate.calls)
	}
}

// 只读角色必须存在冻结候选:干净 workspace 上 reviewer/verifier/specialist
// 在委托执行前就被拒绝,不能以裸方式运行。
func TestVerifiedRoleExecutorRequiresFrozenCandidateForReadOnlyRoles(t *testing.T) {
	for _, role := range []Role{RoleReviewer, RoleVerifier, RoleSpecialist} {
		t.Run(string(role), func(t *testing.T) {
			fx := newVerifiedFixture(t)
			delegate := &recordingDelegate{}
			executor := newVerifiedExecutor(t, fx, delegate)

			specialist := ""
			if role == RoleSpecialist {
				specialist = "performance"
			}
			_, err := executor.Execute(context.Background(), verifiedRequest(fx, roleDirective(role, specialist)))
			if !errors.Is(err, ErrNoFrozenCandidate) {
				t.Fatalf("%s on clean workspace err = %v, want ErrNoFrozenCandidate", role, err)
			}
			if delegate.calls != 0 {
				t.Fatalf("%s ran without a frozen candidate (delegate calls = %d)", role, delegate.calls)
			}
		})
	}
}

// 无角色指令的裸请求被拒收:enforcement 不能被绕过。
func TestVerifiedRoleExecutorRejectsRequestWithoutRoleDirective(t *testing.T) {
	fx := newVerifiedFixture(t)
	writeT(t, filepath.Join(fx.workspace, "candidate.txt"), "candidate\n")
	delegate := &recordingDelegate{}
	executor := newVerifiedExecutor(t, fx, delegate)

	_, err := executor.Execute(context.Background(), verifiedRequest(fx, "raw instruction without a role directive"))
	if err == nil || !strings.Contains(err.Error(), "without a v1.5.4 role directive") {
		t.Fatalf("unmarked instruction err = %v, want role-directive rejection", err)
	}
	if delegate.calls != 0 {
		t.Fatalf("unmarked request reached the delegate (calls = %d)", delegate.calls)
	}
}
