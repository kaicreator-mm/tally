//go:build integration

// P5-06 — Local Git control-plane gate(P5 lane 的 release gate)。
//
// 在 drift / conflict / protected branch / concurrent writer 条件下验证
// L0–L3 权限阶梯与本地集成控制面。全部在真实临时仓库上离线执行:
//
//	受保护分支    Decider 声明受保护 ref:无批准 → ErrProtectedTargetDenied;
//	            带批准 → 令牌签发(AC 矩阵)
//	运行时天花板  RuntimeCaller 请求 L3 → ErrRuntimeAuthorityExceeded
//	            (即使携带有效批准;P5-04 结构执行点)
//	集成主链      EnsureIntegrationBranch(确定性分支,CAS create)→
//	            Integrate(候选内容经唯一写边界 CAS 提交到分支)→
//	            Merge(确定性双亲合并提交,目标 ref CAS 推进)
//	漂移/并发写   持过期基线的 Integrate → ErrStaleBase(败者零部分写入,
//	            HEAD 不动)—— 预检 + gitwriter 锁内二次校验的双层 CAS
//	冲突族        已合并重放 ErrAlreadyMerged;分支漂移
//	            ErrIntegrationBranchExists(P5-02 CAS create);真三方可检
//	            冲突由 merge_test.go 包级 12 测试覆盖(本门禁组合 e2e 面)
//	审计列举      ListIntegrationRefs(P5-05)可见治理命名空间分支
//
// AC1:门禁零远程凭证/零 push —— 仓库无任何 remote 配置,全部操作离线。
package e2e

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"

	"tally/internal/changeset"
	"tally/internal/gitintegration"
	"tally/internal/gitwriter"
)

const p506TargetRef = "refs/heads/main"

// p506Fixture:真实临时仓库(PlainInit + 首提交)+ 候选(文件内容差量)+
// 验收事实 + 服务(主工作树与集成仓库分离)+ 权限决策点(main 受保护)。
type p506Fixture struct {
	t       *testing.T
	repo    string
	base    string
	w       *gitwriter.Writer
	svc     *gitintegration.Service
	decider *gitintegration.Decider
	cand    *changeset.ChangeSet
	fact    gitintegration.AcceptanceFact
}

func newP506Fixture(t *testing.T) *p506Fixture {
	t.Helper()
	dir := t.TempDir()
	if _, err := git.PlainInit(dir, false); err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	w, err := gitwriter.New(dir)
	if err != nil {
		t.Fatalf("gitwriter.New: %v", err)
	}
	baseRes, err := w.Commit(context.Background(), gitwriter.CommitRequest{
		Message: "tally: p506 base",
		Files:   []gitwriter.FileUpdate{{Path: "app/settings.go", Content: []byte("mode = 1\n")}},
	})
	if err != nil {
		t.Fatalf("首提交: %v", err)
	}
	// 显式创建受保护目标 ref(PlainInit 默认分支名不定;合并走 ref 级 CAS,
	// 无需检出)。
	if _, err := gitwriter.RunGit(context.Background(), dir, "", "update-ref", "refs/heads/main", baseRes.Commit); err != nil {
		t.Fatalf("创建 main ref: %v", err)
	}
	svc, err := gitintegration.New(gitintegration.Config{PrimaryWorktree: t.TempDir()})
	if err != nil {
		t.Fatalf("gitintegration.New: %v", err)
	}
	decider, err := gitintegration.NewDecider([]string{p506TargetRef})
	if err != nil {
		t.Fatalf("NewDecider: %v", err)
	}
	cs, err := changeset.New(changeset.Spec{
		Project:      "p506-demo",
		BaseRevision: baseRes.Commit,
		Files: []changeset.ChangedFile{{
			Path:    "app/settings.go",
			Content: "mode = 2\n",
		}},
		ProducerRun: "run-p506",
	})
	if err != nil {
		t.Fatalf("changeset.New: %v", err)
	}
	fact := gitintegration.AcceptanceFact{
		ChangeSetID: cs.ID(),
		AcceptedAt:  time.Now().UTC(),
		Basis:       "approval-record-p506",
	}
	return &p506Fixture{t: t, repo: dir, base: baseRes.Commit, w: w, svc: svc, decider: decider, cand: cs, fact: fact}
}

// authorizeTally 以 Tally 域调用方申请目标 ref 的 L3 授权(受保护目标自动
// 消费批准事实;approvalBasis 为空 = 不携带批准)。
func (f *p506Fixture) authorizeTally(targetRef, approvalBasis string) (gitintegration.Authorization, error) {
	var approval gitintegration.Approval
	if approvalBasis != "" {
		approval = gitintegration.Approval{TargetRef: targetRef, Basis: approvalBasis, ApprovedAt: time.Now().UTC()}
	}
	return f.decider.Authorize(gitintegration.TallyCaller("p506-gate"),
		gitintegration.Request{Level: gitintegration.LevelIntegration, TargetRef: targetRef}, approval)
}

func (f *p506Fixture) head() string {
	f.t.Helper()
	h, err := f.w.Head(context.Background())
	if err != nil {
		f.t.Fatalf("Head: %v", err)
	}
	return h
}

func TestP506LocalGitControlPlaneGate(t *testing.T) {
	ctx := context.Background()
	f := newP506Fixture(t)

	// AC1:门禁零远程凭证/零 push —— 仓库无任何 remote 配置,全部操作离线。
	repoObj, err := git.PlainOpen(f.repo)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}
	remotes, err := repoObj.Remotes()
	if err != nil {
		t.Fatalf("读 remotes: %v", err)
	}
	if len(remotes) != 0 {
		t.Fatalf("门禁夹具不得配置 remote(AC1)")
	}

	// 受保护分支面:无批准拒绝、带批准签发。
	if _, err := f.authorizeTally(p506TargetRef, ""); !errors.Is(err, gitintegration.ErrProtectedTargetDenied) {
		t.Fatalf("受保护目标无批准应 ErrProtectedTargetDenied,得到 %v", err)
	}
	tokMain, err := f.authorizeTally(p506TargetRef, "gov-approval-p506")
	if err != nil {
		t.Fatalf("带批准应签发令牌: %v", err)
	}

	// 运行时天花板:RuntimeCaller 请求 L3 → 拒绝(即使携带有效批准)。
	if _, err := f.decider.Authorize(gitintegration.RuntimeCaller("fake-runtime"),
		gitintegration.Request{Level: gitintegration.LevelIntegration, TargetRef: p506TargetRef},
		gitintegration.Approval{TargetRef: p506TargetRef, Basis: "gov-approval-p506", ApprovedAt: time.Now().UTC()},
	); !errors.Is(err, gitintegration.ErrRuntimeAuthorityExceeded) {
		t.Fatalf("runtime 请求 L3 应 ErrRuntimeAuthorityExceeded,得到 %v", err)
	}

	// 集成分支(确定性命名;令牌绑定分支 ref;非受保护命名空间)。
	branchRef, err := gitintegration.IntegrationBranchRefFor(f.cand.ID())
	if err != nil {
		t.Fatalf("IntegrationBranchRefFor: %v", err)
	}
	tokBranch, err := f.authorizeTally(branchRef, "")
	if err != nil {
		t.Fatalf("分支令牌应签发(非受保护命名空间): %v", err)
	}
	branchRes, err := f.svc.EnsureIntegrationBranch(ctx, gitintegration.IntegrationBranchRequest{
		ChangeSet:      f.cand,
		Acceptance:     f.fact,
		RepositoryPath: f.repo,
		Authorization:  tokBranch,
	})
	if err != nil || !branchRes.Created {
		t.Fatalf("EnsureIntegrationBranch: created=%v err=%v", branchRes.Created, err)
	}
	if branchRes.Base != f.base {
		t.Fatalf("分支应植根候选基线: %q", branchRes.Base)
	}

	// 检出集成分支,让 Integrate 的提交落在分支上(合并才有内容可并)。
	if _, err := gitwriter.RunGit(ctx, f.repo, "", "checkout", "-q", strings.TrimPrefix(branchRef, "refs/heads/")); err != nil {
		t.Fatalf("checkout 集成分支: %v", err)
	}

	// 集成主链:候选内容经唯一写边界 CAS 提交到分支。
	res, err := f.svc.Integrate(ctx, gitintegration.IntegrateRequest{
		ChangeSet:      f.cand,
		Acceptance:     f.fact,
		RepositoryPath: f.repo,
		ExpectedBase:   f.base,
	})
	if err != nil {
		t.Fatalf("Integrate: %v", err)
	}
	if res.Commit == "" || f.head() != res.Commit {
		t.Fatalf("集成提交应推进分支 HEAD: commit=%q head=%q", res.Commit, f.head())
	}

	// 漂移/并发写败者:持过期基线的第二次集成 → ErrStaleBase,HEAD 不动
	//(预检 + gitwriter 锁内二次校验的双层 CAS;败者零部分写入)。
	_, err = f.svc.Integrate(ctx, gitintegration.IntegrateRequest{
		ChangeSet:      f.cand,
		Acceptance:     f.fact,
		RepositoryPath: f.repo,
		ExpectedBase:   f.base,
	})
	if !errors.Is(err, gitintegration.ErrStaleBase) {
		t.Fatalf("漂移应为 ErrStaleBase,得到 %v", err)
	}
	if f.head() != res.Commit {
		t.Fatalf("漂移失败后 HEAD 不得移动: %q", f.head())
	}

	// 合并:集成分支合入受保护 main(经批准令牌;目标 ref CAS 推进)。
	mergeRes, err := f.svc.Merge(ctx, gitintegration.MergeRequest{
		ChangeSet:            f.cand,
		Acceptance:           f.fact,
		RepositoryPath:       f.repo,
		TargetRef:            p506TargetRef,
		ExpectedTarget:       f.base,
		ExpectedBranchCommit: res.Commit,
		Authorization:        tokMain,
	})
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if mergeRes.Commit == "" {
		t.Fatalf("合并应产出确定性提交")
	}

	// 已合并重放:再次合并 → ErrAlreadyMerged。
	_, err = f.svc.Merge(ctx, gitintegration.MergeRequest{
		ChangeSet:            f.cand,
		Acceptance:           f.fact,
		RepositoryPath:       f.repo,
		TargetRef:            p506TargetRef,
		ExpectedTarget:       f.head(),
		ExpectedBranchCommit: res.Commit,
		Authorization:        tokMain,
	})
	if !errors.Is(err, gitintegration.ErrAlreadyMerged) {
		t.Fatalf("已合并重放应 ErrAlreadyMerged,得到 %v", err)
	}

	// 分支漂移:同一确定性名重新创建(分支现指向集成提交,偏离 base)→
	// ErrIntegrationBranchExists。
	if _, err := f.svc.EnsureIntegrationBranch(ctx, gitintegration.IntegrationBranchRequest{
		ChangeSet:      f.cand,
		Acceptance:     f.fact,
		RepositoryPath: f.repo,
		Authorization:  tokBranch,
	}); !errors.Is(err, gitintegration.ErrIntegrationBranchExists) {
		t.Fatalf("分支漂移应 ErrIntegrationBranchExists,得到 %v", err)
	}

	// 审计列举(P5-05):治理命名空间的分支可见。
	refs, err := gitintegration.ListIntegrationRefs(ctx, f.repo)
	if err != nil || len(refs) == 0 {
		t.Fatalf("集成审计列举应可见分支: %v %d", err, len(refs))
	}
}
