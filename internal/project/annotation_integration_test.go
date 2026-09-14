//go:build integration

package project

import (
	"context"
	"testing"
	"time"

	"tally/internal/core"
)

// TestScanAnnotationsFromGitRepo:扫真实仓库的 @covers 注解,按 HEAD 时刻绑定当前条款版本。
func TestScanAnnotationsFromGitRepo(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	q := st.Queries()
	run := runID(t)
	project := core.ProjectID("proj-" + run)

	h1 := "sha256:" + run + "-req1"
	h2 := "sha256:" + run + "-req2sub"
	seedClauseVersion(t, q, string(project), "REQ-1", h1, 1, base, true)
	seedClauseVersion(t, q, string(project), "REQ-2.2", h2, 1, base, true)

	dir, repo := initTestRepo(t)
	src := "package x\n\n// @covers REQ-1\n// @covers REQ-2.2\nfunc TestA(t *testing.T) {}\n"
	writeCommit(t, repo, dir, "x_test.go", src, "test: 加测试", base.Add(1*time.Hour))
	// 另一个无注解的文件:不产生挂钩。
	writeCommit(t, repo, dir, "y.go", "package y\n", "feat: 无注解", base.Add(2*time.Hour))

	n, err := ScanAnnotations(ctx, q, project, dir)
	if err != nil {
		t.Fatalf("ScanAnnotations: %v", err)
	}
	if n != 2 {
		t.Errorf("应绑 2 条(REQ-1 与 REQ-2.2), got %d", n)
	}

	// test_clause 里应有 x_test.go → h1 与 x_test.go → h2。
	got := map[string]bool{}
	rows, err := q.ListTestClauses(ctx)
	if err != nil {
		t.Fatalf("列 test_clause: %v", err)
	}
	for _, r := range rows {
		got[r.TestName+"→"+r.ClauseContentHash] = true
	}
	if !got["x_test.go→"+h1] || !got["x_test.go→"+h2] {
		t.Errorf("test_clause 缺挂钩: %v", got)
	}
}

// TestClauseCoverage:当前条款里有多少被 @covers 引用。
func TestClauseCoverage(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	q := st.Queries()
	run := runID(t)
	project := core.ProjectID("proj-" + run)

	h1 := "sha256:" + run + "-c1"
	h2 := "sha256:" + run + "-c2"
	h3 := "sha256:" + run + "-c3"
	seedClauseVersion(t, q, string(project), "REQ-1", h1, 1, base, true)
	seedClauseVersion(t, q, string(project), "REQ-2", h2, 1, base, true)
	seedClauseVersion(t, q, string(project), "REQ-3", h3, 1, base, true)

	// 测试只覆盖 REQ-1 与 REQ-3,REQ-2 未覆盖。
	if _, err := BindTest(ctx, q, project, "t1", []string{"REQ-1", "REQ-3"}, base.Add(time.Hour)); err != nil {
		t.Fatalf("BindTest: %v", err)
	}

	cov, err := ClauseCoverage(ctx, q, project)
	if err != nil {
		t.Fatalf("ClauseCoverage: %v", err)
	}
	if cov.Total != 3 || cov.Covered != 2 {
		t.Errorf("覆盖率应 2/3, got %d/%d", cov.Covered, cov.Total)
	}
	if len(cov.Uncovered) != 1 || cov.Uncovered[0] != "REQ-2" {
		t.Errorf("未覆盖应是 [REQ-2], got %v", cov.Uncovered)
	}
	if cov.Rate() < 0.66 || cov.Rate() > 0.67 {
		t.Errorf("Rate 应约 0.667, got %f", cov.Rate())
	}
}

// TestClauseCoverageNoRequirementLayer:没有需求层(无当前条款)时 Total=0、Rate=0,不报错。
func TestClauseCoverageNoRequirementLayer(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	q := st.Queries()
	run := runID(t)
	project := core.ProjectID("proj-empty-" + run)

	cov, err := ClauseCoverage(ctx, q, project)
	if err != nil {
		t.Fatalf("空项目覆盖率不该报错: %v", err)
	}
	if cov.Total != 0 || cov.Rate() != 0 {
		t.Errorf("空项目应 0/0, got %d, rate %f", cov.Total, cov.Rate())
	}
}
