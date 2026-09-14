package gitintegration

// Git 权限策略(P5-04)的测试:权限矩阵是确定性表测试(同一输入永远同一
// 决策,不依赖 Git/时间/顺序);结构性边界(runtime 无法请求 L3)用矩阵行、
// 令牌伪造/误绑失败路径与导入面扫描共同钉住。涉及真实操作失败路径的用例
// 遵循 service_test.go 的真实临时 Git 仓库约定。

import (
	"context"
	"errors"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testAuth 为目标 ref 签发一枚 Tally 域 L3 授权令牌(测试管线行为用;
// 未声明受保护分支 —— 审计命名空间与普通 ref 的令牌都经此获得)。
func testAuth(t *testing.T, target string) Authorization {
	t.Helper()
	d, err := NewDecider(nil)
	if err != nil {
		t.Fatalf("NewDecider: %v", err)
	}
	auth, err := d.Authorize(TallyCaller("test"), Request{Level: LevelIntegration, TargetRef: target}, Approval{})
	if err != nil {
		t.Fatalf("Authorize %s: %v", target, err)
	}
	return auth
}

func TestPermissionMatrix(t *testing.T) {
	const (
		plainTarget    = "refs/heads/target"
		protectedMains = "refs/heads/main"
		auditTarget    = "refs/heads/tally/integration/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	)
	validApproval := Approval{TargetRef: protectedMains, Basis: "governance:approved", ApprovedAt: time.Now().UTC()}

	d, err := NewDecider([]string{protectedMains})
	if err != nil {
		t.Fatalf("NewDecider: %v", err)
	}

	cases := []struct {
		name      string
		caller    Caller
		level     Level
		target    string
		approval  Approval
		wantAllow bool
		wantBasis string // wantAllow 时断言精确 basis;拒绝时忽略
		wantErr   error  // wantAllow 为 false 时的期望 sentinel
	}{
		// ---- Tally 域:L0-L2 在天花板内;L3 非受保护目标走受控集成流程。 ----
		{name: "tally L0 read", caller: TallyCaller("worker"), level: LevelRead, wantAllow: true, wantBasis: basisWithinCeiling},
		{name: "tally L1 workspace", caller: TallyCaller("worker"), level: LevelWorkspace, wantAllow: true, wantBasis: basisWithinCeiling},
		{name: "tally L2 commit", caller: TallyCaller("worker"), level: LevelCommit, wantAllow: true, wantBasis: basisWithinCeiling},
		{name: "tally L3 unprotected", caller: TallyCaller("integration-step"), level: LevelIntegration, target: plainTarget, wantAllow: true, wantBasis: basisControlledIntegration},
		{name: "tally L3 unprotected ignores unrelated approval", caller: TallyCaller("integration-step"), level: LevelIntegration, target: plainTarget, approval: validApproval, wantAllow: true, wantBasis: basisControlledIntegration},
		{name: "tally L3 audit namespace (branch creation)", caller: TallyCaller("integration-step"), level: LevelIntegration, target: auditTarget, wantAllow: true, wantBasis: basisControlledIntegration},

		// ---- Tally 域 L3 受保护分支:必须携带与目标匹配、完整的策略批准。 ----
		{name: "tally L3 protected without approval", caller: TallyCaller("integration-step"), level: LevelIntegration, target: protectedMains, wantErr: ErrProtectedTargetDenied},
		{name: "tally L3 protected zero approval", caller: TallyCaller("integration-step"), level: LevelIntegration, target: protectedMains, approval: Approval{}, wantErr: ErrProtectedTargetDenied},
		{name: "tally L3 protected approval without basis", caller: TallyCaller("integration-step"), level: LevelIntegration, target: protectedMains, approval: Approval{TargetRef: protectedMains, ApprovedAt: time.Now().UTC()}, wantErr: ErrProtectedTargetDenied},
		{name: "tally L3 protected approval without time", caller: TallyCaller("integration-step"), level: LevelIntegration, target: protectedMains, approval: Approval{TargetRef: protectedMains, Basis: "governance:approved"}, wantErr: ErrProtectedTargetDenied},
		{name: "tally L3 protected approval mismatched target", caller: TallyCaller("integration-step"), level: LevelIntegration, target: protectedMains, approval: Approval{TargetRef: plainTarget, Basis: "governance:approved", ApprovedAt: time.Now().UTC()}, wantErr: ErrProtectedTargetDenied},
		{name: "tally L3 protected with valid approval", caller: TallyCaller("integration-step"), level: LevelIntegration, target: protectedMains, approval: validApproval, wantAllow: true, wantBasis: basisProtectedApprovedPrefix + "governance:approved"},

		// ---- runtime:天花板 L1;L2/L3 是结构拒绝,任何批准不可覆盖。 ----
		{name: "runtime L0 read", caller: RuntimeCaller("codex"), level: LevelRead, wantAllow: true, wantBasis: basisWithinCeiling},
		{name: "runtime L1 workspace", caller: RuntimeCaller("codex"), level: LevelWorkspace, wantAllow: true, wantBasis: basisWithinCeiling},
		{name: "runtime L2 commit denied (DL-03)", caller: RuntimeCaller("codex"), level: LevelCommit, wantErr: ErrRuntimeAuthorityExceeded},
		{name: "runtime L3 unprotected denied", caller: RuntimeCaller("codex"), level: LevelIntegration, target: plainTarget, wantErr: ErrRuntimeAuthorityExceeded},
		{name: "runtime L3 protected denied even with valid approval", caller: RuntimeCaller("codex"), level: LevelIntegration, target: protectedMains, approval: validApproval, wantErr: ErrRuntimeAuthorityExceeded},

		// ---- 非法输入:失败安全,绝不猜。 ----
		{name: "unknown caller kind", caller: Caller{Kind: CallerKind(9)}, level: LevelRead, wantErr: ErrInvalidPermissionRequest},
		{name: "level below L0", caller: TallyCaller("worker"), level: Level(-1), wantErr: ErrInvalidPermissionRequest},
		{name: "level 4 (L4 Remote out of scope)", caller: TallyCaller("worker"), level: Level(4), wantErr: ErrInvalidPermissionRequest},
		{name: "level 5 (L5 Destructive out of scope)", caller: TallyCaller("worker"), level: Level(5), wantErr: ErrInvalidPermissionRequest},
		{name: "far out-of-range level", caller: TallyCaller("worker"), level: Level(100), wantErr: ErrInvalidPermissionRequest},
		{name: "L3 without target ref", caller: TallyCaller("worker"), level: LevelIntegration, wantErr: ErrInvalidPermissionRequest},
		{name: "L3 relative ref target", caller: TallyCaller("worker"), level: LevelIntegration, target: "master", wantErr: ErrInvalidPermissionRequest},
		{name: "L3 traversal-ish ref target", caller: TallyCaller("worker"), level: LevelIntegration, target: "refs/heads/a..b", wantErr: ErrInvalidPermissionRequest},
	}

	for _, tc := range cases {
		got, err := d.Evaluate(tc.caller, Request{Level: tc.level, TargetRef: tc.target}, tc.approval)
		if tc.wantAllow {
			if err != nil {
				t.Fatalf("%s: Evaluate returned %v, want allow", tc.name, err)
			}
			if !got.Allowed || got.Level != tc.level || got.Basis != tc.wantBasis {
				t.Fatalf("%s: decision = %+v, want allowed with basis %q", tc.name, got, tc.wantBasis)
			}
			// 确定性:同一输入重复求值必须得到同一决策。
			again, err := d.Evaluate(tc.caller, Request{Level: tc.level, TargetRef: tc.target}, tc.approval)
			if err != nil || again != got {
				t.Fatalf("%s: Evaluate not deterministic: %+v vs %+v (err=%v)", tc.name, again, got, err)
			}
			continue
		}
		if !errors.Is(err, tc.wantErr) {
			t.Fatalf("%s: got %v, want %v", tc.name, err, tc.wantErr)
		}
		if got.Allowed {
			t.Fatalf("%s: decision projection claims allowed on denial: %+v", tc.name, got)
		}
	}

	// 组合根把审计 ref 声明为受保护是合法的防御纵深:此时它同样要求批准。
	dAudit, err := NewDecider([]string{auditTarget})
	if err != nil {
		t.Fatalf("NewDecider(audit): %v", err)
	}
	if !dAudit.IsProtected(auditTarget) {
		t.Fatal("declared audit ref must be protected")
	}
	if _, err := dAudit.Evaluate(TallyCaller("integration-step"), Request{Level: LevelIntegration, TargetRef: auditTarget}, Approval{}); !errors.Is(err, ErrProtectedTargetDenied) {
		t.Fatalf("declared-protected audit ref without approval: got %v, want ErrProtectedTargetDenied", err)
	}
}

// TestRuntimeCallerNeverObtainsIntegrationAuthority 是任务验收标准的直接证明:
// runtime 形态的调用方带着 L3 请求走到权限层,无论携带什么批准事实,
// 都被结构拒绝,拿不到任何 Authorization 令牌 —— 没有令牌就没有可调用的
// L3 操作(EnsureIntegrationBranch / Merge 都要求令牌)。
func TestRuntimeCallerNeverObtainsIntegrationAuthority(t *testing.T) {
	d, err := NewDecider([]string{"refs/heads/main"})
	if err != nil {
		t.Fatalf("NewDecider: %v", err)
	}
	targets := []string{"refs/heads/target", "refs/heads/main"}
	for _, target := range targets {
		for _, approval := range []Approval{
			{},
			{TargetRef: target, Basis: "governance:attempted", ApprovedAt: time.Now().UTC()},
		} {
			auth, err := d.Authorize(RuntimeCaller("codex"), Request{Level: LevelIntegration, TargetRef: target}, approval)
			if !errors.Is(err, ErrRuntimeAuthorityExceeded) {
				t.Fatalf("runtime L3 %s (approval=%+v): got %v, want ErrRuntimeAuthorityExceeded", target, approval, err)
			}
			if auth.valid() {
				t.Fatalf("runtime L3 %s: obtained an authorization token %+v", target, auth)
			}
			if !errors.Is(requireAuthorization(auth, target), ErrPermissionRequired) {
				t.Fatalf("runtime L3 %s: zero token must be rejected at the operation entry", target)
			}
		}
	}
}

// TestAuthorizationTokenLifecycle 钉住令牌语义:只有 L3 经 Authorize 签发、
// 零值令牌在操作入口被拒(ErrPermissionRequired)、令牌不可跨目标重用
// (ErrPermissionDenied)。
func TestAuthorizationTokenLifecycle(t *testing.T) {
	d, err := NewDecider(nil)
	if err != nil {
		t.Fatalf("NewDecider: %v", err)
	}
	if _, err := d.Authorize(TallyCaller("worker"), Request{Level: LevelCommit}, Approval{}); !errors.Is(err, ErrInvalidPermissionRequest) {
		t.Fatalf("Authorize on L2: got %v, want ErrInvalidPermissionRequest", err)
	}

	auth, err := d.Authorize(TallyCaller("worker"), Request{Level: LevelIntegration, TargetRef: "refs/heads/target"}, Approval{})
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	if err := requireAuthorization(auth, "refs/heads/target"); err != nil {
		t.Fatalf("bound token rejected: %v", err)
	}
	if err := requireAuthorization(Authorization{}, "refs/heads/target"); !errors.Is(err, ErrPermissionRequired) {
		t.Fatalf("zero token: got %v, want ErrPermissionRequired", err)
	}
	if err := requireAuthorization(auth, "refs/heads/other"); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("misbound token: got %v, want ErrPermissionDenied", err)
	}
}

func TestNewDeciderValidation(t *testing.T) {
	cases := []struct {
		name string
		refs []string
	}{
		{name: "relative ref", refs: []string{"master"}},
		{name: "empty ref", refs: []string{""}},
		{name: "option-shaped ref", refs: []string{"refs/heads/-evil"}},
		{name: "dotdot ref", refs: []string{"refs/heads/a..b"}},
		{name: "lock suffix ref", refs: []string{"refs/heads/x.lock"}},
		{name: "space in ref", refs: []string{"refs/heads/a b"}},
		{name: "duplicate refs", refs: []string{"refs/heads/main", "refs/heads/main"}},
	}
	for _, tc := range cases {
		if _, err := NewDecider(tc.refs); !errors.Is(err, ErrInvalidPermissionRequest) {
			t.Fatalf("%s: got %v, want ErrInvalidPermissionRequest", tc.name, err)
		}
	}
	// nil/空受保护集合合法:组合根可以不声明受保护分支。
	d, err := NewDecider(nil)
	if err != nil {
		t.Fatalf("NewDecider(nil): %v", err)
	}
	if d.IsProtected("refs/heads/main") {
		t.Fatal("undeclared ref must not be protected")
	}
	// nil 接收者失败安全:IsProtected 恒 false,但 Evaluate 拒绝。
	var nilDecider *Decider
	if nilDecider.IsProtected("refs/heads/main") {
		t.Fatal("nil decider must not report protection")
	}
	if _, err := nilDecider.Evaluate(TallyCaller("worker"), Request{Level: LevelRead}, Approval{}); !errors.Is(err, ErrInvalidPermissionRequest) {
		t.Fatalf("nil decider Evaluate: got %v, want ErrInvalidPermissionRequest", err)
	}
}

// TestMergeRefusesMissingAuthorization:L3 合并在没有授权令牌时于触碰任何
// 仓库状态之前被拒(负路径;同时是「无令牌即无操作」的证据)。
func TestMergeRefusesMissingAuthorization(t *testing.T) {
	f := newMergeFixture(t, "pm-1",
		map[string]string{"g.txt": "ours"},
		map[string]string{"f.txt": "theirs"},
	)
	svc := newTestService(t, t.TempDir())

	_, err := svc.Merge(context.Background(), MergeRequest{
		ChangeSet:            f.cs,
		Acceptance:           f.fact,
		RepositoryPath:       f.dir,
		TargetRef:            "refs/heads/target",
		ExpectedTarget:       f.target,
		ExpectedBranchCommit: f.tip,
		// Authorization: 刻意缺席。
	})
	if !errors.Is(err, ErrPermissionRequired) {
		t.Fatalf("got %v, want ErrPermissionRequired", err)
	}
	if !refPointsAt(t, f.dir, "refs/heads/target", f.target) || !refPointsAt(t, f.dir, f.ref, f.tip) {
		t.Fatal("refs moved despite missing authorization")
	}
}

// TestMergeRefusesMisboundAuthorization:绑定到别的目标的令牌不可重用。
func TestMergeRefusesMisboundAuthorization(t *testing.T) {
	f := newMergeFixture(t, "pm-2",
		map[string]string{"g.txt": "ours"},
		map[string]string{"f.txt": "theirs"},
	)
	svc := newTestService(t, t.TempDir())

	_, err := svc.Merge(context.Background(), MergeRequest{
		ChangeSet:            f.cs,
		Acceptance:           f.fact,
		RepositoryPath:       f.dir,
		TargetRef:            "refs/heads/target",
		ExpectedTarget:       f.target,
		ExpectedBranchCommit: f.tip,
		Authorization:        testAuth(t, "refs/heads/other"),
	})
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("got %v, want ErrPermissionDenied", err)
	}
	if !refPointsAt(t, f.dir, "refs/heads/target", f.target) || !refPointsAt(t, f.dir, f.ref, f.tip) {
		t.Fatal("refs moved despite misbound authorization")
	}
}

// TestProtectedBranchMergeRequiresApproval 端到端证明「受保护分支要求
// 策略批准的本地集成」:未批准 → 拿不到令牌 → 合并结构上不可达;
// 批准后 → 令牌签发 → 合并按既有受控管线成功。
func TestProtectedBranchMergeRequiresApproval(t *testing.T) {
	f := newMergeFixture(t, "pm-3",
		map[string]string{"g.txt": "ours"},
		map[string]string{"f.txt": "theirs"},
	)
	const protected = "refs/heads/target"
	d, err := NewDecider([]string{protected})
	if err != nil {
		t.Fatalf("NewDecider: %v", err)
	}

	// 未批准:决策拒绝,没有令牌,合并不可达。
	if _, err := d.Authorize(TallyCaller("integration-step"), Request{Level: LevelIntegration, TargetRef: protected}, Approval{}); !errors.Is(err, ErrProtectedTargetDenied) {
		t.Fatalf("unapproved protected merge: got %v, want ErrProtectedTargetDenied", err)
	}

	// 策略批准后:放行 + 令牌,合并沿既有管线成功。
	auth, err := d.Authorize(TallyCaller("integration-step"), Request{Level: LevelIntegration, TargetRef: protected},
		Approval{TargetRef: protected, Basis: "governance:approved", ApprovedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("approved protected merge: %v", err)
	}
	svc := newTestService(t, t.TempDir())
	res, err := svc.Merge(context.Background(), MergeRequest{
		ChangeSet:            f.cs,
		Acceptance:           f.fact,
		RepositoryPath:       f.dir,
		TargetRef:            protected,
		ExpectedTarget:       f.target,
		ExpectedBranchCommit: f.tip,
		Authorization:        auth,
	})
	if err != nil {
		t.Fatalf("Merge with approved authorization: %v", err)
	}
	if !refPointsAt(t, f.dir, protected, res.Commit) {
		t.Fatal("protected target did not advance to merge commit")
	}
}

// TestEnsureIntegrationBranchRefusesMissingAuthorization:分支创建同样是
// L3 操作,没有授权令牌即在触碰仓库前被拒。
func TestEnsureIntegrationBranchRefusesMissingAuthorization(t *testing.T) {
	dir := initRepo(t)
	base := seedCommit(t, dir)
	cs, fact := acceptedCandidate(t, base, "pm-4")
	svc := newTestService(t, t.TempDir())

	_, err := svc.EnsureIntegrationBranch(context.Background(), IntegrationBranchRequest{
		ChangeSet:      cs,
		Acceptance:     fact,
		RepositoryPath: dir,
		// Authorization: 刻意缺席。
	})
	if !errors.Is(err, ErrPermissionRequired) {
		t.Fatalf("got %v, want ErrPermissionRequired", err)
	}
	if refExists(t, dir, expectedBranchRef(t, cs)) {
		t.Fatal("branch was created despite missing authorization")
	}
	if h := runTestGit(t, dir, "rev-parse", "HEAD"); strings.TrimSpace(h) != base {
		t.Fatalf("HEAD moved to %q", h)
	}
}

// TestRuntimePackagesNeverImportGitIntegration 把「runtime 一侧到不了集成层」
// 钉成机器规则:internal/agentruntime 与 runtures/** 的非测试 Go 文件不得
// 导入本包(权限决策与 L3 操作只在 Tally 域/组合根可达);反向,本包是
// runtime 中立的语义层,不导入任何 runtime 侧包。
func TestRuntimePackagesNeverImportGitIntegration(t *testing.T) {
	importsOf := func(dir string) map[string]bool {
		t.Helper()
		out := map[string]bool{}
		err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			for _, imp := range f.Imports {
				out[strings.Trim(imp.Path.Value, `"`)] = true
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
		return out
	}

	const integrationPkg = "tally/internal/gitintegration"
	for _, dir := range []string{
		filepath.Join("..", "..", "internal", "agentruntime"),
		filepath.Join("..", "..", "runtimes"),
	} {
		for imp := range importsOf(dir) {
			if imp == integrationPkg {
				t.Errorf("%s imports %s: runtime 一侧不得触达 Git 集成层(P5-04 权限边界)", dir, integrationPkg)
			}
		}
	}

	own := importsOf(".")
	for _, imp := range []string{"tally/internal/agentruntime", "tally/internal/airuntime", "tally/internal/llm"} {
		if own[imp] {
			t.Errorf("gitintegration imports %s: 集成层必须 runtime 中立(P5-04 权限边界)", imp)
		}
	}
}
