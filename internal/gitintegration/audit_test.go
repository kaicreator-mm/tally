// audit_test.go 拥有集成事实列举的测试(P5-05):真实临时仓库上,P5-02 落地
// 的集成分支被完整列举,空仓库零行,输出按 ref 名升序。
package gitintegration

import (
	"context"
	"strings"
	"testing"
)

// 集成分支经 P5-02 具名操作落地(写面只用既有具名操作),再经本审计列举。
func TestListIntegrationRefs(t *testing.T) {
	ctx := context.Background()
	dir := initRepo(t)
	base := seedCommit(t, dir)
	cs, fact := acceptedCandidate(t, base, "audit-1")
	svc := newTestService(t, t.TempDir())

	// 空集合:尚无集成事实,不是错误。
	refs, err := ListIntegrationRefs(ctx, dir)
	if err != nil || len(refs) != 0 {
		t.Fatalf("无集成分支时应零行: %v %+v", err, refs)
	}

	res, err := svc.EnsureIntegrationBranch(context.Background(), IntegrationBranchRequest{
		ChangeSet:      cs,
		Acceptance:     fact,
		RepositoryPath: dir,
		Authorization:  testAuth(t, expectedBranchRef(t, cs)),
	})
	if err != nil || !res.Created {
		t.Fatalf("EnsureIntegrationBranch: created=%v err=%v", res.Created, err)
	}

	refs, err = ListIntegrationRefs(ctx, dir)
	if err != nil {
		t.Fatalf("ListIntegrationRefs: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("应列举恰 1 条集成分支,得到 %d", len(refs))
	}
	if refs[0].Ref != res.Branch {
		t.Fatalf("ref 名不一致: %s vs %s", refs[0].Ref, res.Branch)
	}
	if refs[0].Commit != base {
		t.Fatalf("ref 指向 %q, want 基线 %q", refs[0].Commit, base)
	}
	if strings.TrimSpace(refs[0].Subject) == "" {
		t.Fatal("应携带提交主题(观测值原样)")
	}
	if strings.TrimSpace(refs[0].Date) == "" {
		t.Fatal("应携带提交时刻")
	}

	// 第二条分支:输出按 ref 名升序(确定性)。
	cs2, fact2 := acceptedCandidate(t, base, "audit-2")
	res2, err := svc.EnsureIntegrationBranch(context.Background(), IntegrationBranchRequest{
		ChangeSet:      cs2,
		Acceptance:     fact2,
		RepositoryPath: dir,
		Authorization:  testAuth(t, expectedBranchRef(t, cs2)),
	})
	if err != nil || !res2.Created {
		t.Fatalf("EnsureIntegrationBranch(2): created=%v err=%v", res2.Created, err)
	}
	refs, err = ListIntegrationRefs(ctx, dir)
	if err != nil || len(refs) != 2 {
		t.Fatalf("应列举 2 条: %v %d", err, len(refs))
	}
	if refs[0].Ref >= refs[1].Ref {
		t.Fatalf("列举应按 ref 名升序: %s vs %s", refs[0].Ref, refs[1].Ref)
	}
}

// 非仓库路径 fail-safe。
func TestListIntegrationRefsNotARepo(t *testing.T) {
	if _, err := ListIntegrationRefs(context.Background(), t.TempDir()); err == nil {
		t.Fatal("非仓库应报错")
	}
}
