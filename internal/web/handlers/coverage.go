package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"tally/internal/core"
	"tally/internal/project"
	"tally/internal/requirement"
)

// 条款覆盖率 + 「依据已变更」标记(INV-4)。
//
// 覆盖率:签署的当前条款里有多少被测试 @covers 引用(project.ClauseCoverage)。
// 依据已变更:下游引用(commit trailer / 测试注解)记的是条款某历史版本的 content_hash;
// 条款内容一变哈希就变,旧引用对不上当前版 → requirement.CheckReference 判「依据已变更」。
// 失效是哈希比对的自然结果,不需要一张失效表、不需要传播逻辑。
//
// 只读;需要 s.Trace(*store.Queries);nil → 空状态行动提示。
// 一个页面两用:无 project 参数 → 项目索引;有 project → 该项目详情。

// CoverageProject 是覆盖率索引页的一个项目。
type CoverageProject struct {
	Slug    string
	Total   int
	Covered int
	RatePct int // 0..100
}

// ChangedRef 是一条「依据已变更」的下游引用。
type ChangedRef struct {
	Kind       string // commit / test
	RefHash    string // 短显
	ClauseID   string
	StatusText string // requirement.RefStatus.String() = 「依据已变更」
}

// UncoveredRow 是一条未被测试覆盖的条款。
type UncoveredRow struct {
	ClauseID string
}

// CoveragePage 是覆盖率页数据(索引 / 详情共用;有 project 参数时 Detail=true)。
type CoveragePage struct {
	Page
	Detail    bool
	Project   string
	Projects  []CoverageProject // 索引模式
	Total     int
	Covered   int
	RatePct   int
	Uncovered []UncoveredRow
	Changed   []ChangedRef
}

// Coverage 渲染条款覆盖率页(/coverage 或 /coverage?project=<p>)。
func (s *Server) Coverage(w http.ResponseWriter, r *http.Request) {
	projectID := strings.TrimSpace(r.URL.Query().Get("project"))
	if projectID == "" {
		data := CoveragePage{Page: s.page(r, "条款覆盖率", "coverage")}
		if s.Trace != nil {
			data.Projects = s.coverageIndex(r)
		}
		s.Render.Render(w, http.StatusOK, "coverage", data)
		return
	}

	data := CoveragePage{Page: s.page(r, "覆盖率 "+projectID, "coverage"), Detail: true, Project: projectID}
	if s.Trace == nil {
		s.Render.Render(w, http.StatusOK, "coverage", data)
		return
	}
	ctx := r.Context()
	cov, err := project.ClauseCoverage(ctx, s.Trace, core.ProjectID(projectID))
	if err != nil {
		s.serverError(w, r, "算覆盖率", err)
		return
	}
	data.Total = cov.Total
	data.Covered = cov.Covered
	data.RatePct = int(cov.Rate() * 100)
	for _, id := range cov.Uncovered {
		data.Uncovered = append(data.Uncovered, UncoveredRow{ClauseID: id})
	}
	data.Changed = s.changedReferences(ctx, projectID)
	s.Render.Render(w, http.StatusOK, "coverage", data)
}

// coverageIndex 列出有需求层的项目及各自覆盖率。
func (s *Server) coverageIndex(r *http.Request) []CoverageProject {
	slugs, err := s.Trace.ListClauseProjects(r.Context())
	if err != nil {
		slog.Error("列需求项目", "err", err)
		return nil
	}
	var out []CoverageProject
	for _, slug := range slugs {
		cp := CoverageProject{Slug: slug}
		if cov, cerr := project.ClauseCoverage(r.Context(), s.Trace, core.ProjectID(slug)); cerr != nil {
			slog.Error("算覆盖率", "project", slug, "err", cerr)
		} else {
			cp.Total = cov.Total
			cp.Covered = cov.Covered
			cp.RatePct = int(cov.Rate() * 100)
		}
		out = append(out, cp)
	}
	return out
}

// changedReferences 列出本项目「依据已变更」的下游引用:条款哈希对不上当前版(INV-4)。
// 判定用 requirement.CheckReference —— 这就是 INV-4 要的全部机制。
func (s *Server) changedReferences(ctx context.Context, projectID string) []ChangedRef {
	currents, err := s.Trace.ListCurrentClauseVersions(ctx, projectID)
	if err != nil {
		slog.Error("读当前条款哈希", "project", projectID, "err", err)
		return nil
	}
	currentHash := make(map[string]core.ClauseHash, len(currents))
	for _, c := range currents {
		currentHash[c.ClauseID] = core.ClauseHash(c.ContentHash)
	}
	refs, err := s.Trace.ListClauseReferences(ctx, projectID)
	if err != nil {
		slog.Error("读下游引用", "project", projectID, "err", err)
		return nil
	}
	var out []ChangedRef
	for _, ref := range refs {
		cur, ok := currentHash[ref.ClauseID]
		if !ok {
			continue // 条款已不在当前版本集(被删):无当前基准可比,交给覆盖率视图呈现
		}
		// 引用的哈希对不上当前版 → 依据已变更(INV-4 的唯一判定)。
		if requirement.CheckReference(core.ClauseHash(ref.RefHash), cur) != requirement.RefChanged {
			continue
		}
		out = append(out, ChangedRef{
			Kind:       ref.Kind,
			RefHash:    shortHash(ref.RefHash),
			ClauseID:   ref.ClauseID,
			StatusText: requirement.RefChanged.String(),
		})
	}
	return out
}
