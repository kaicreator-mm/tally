package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"tally/internal/core"
	"tally/internal/project"
)

// 追溯视图(PRD 第六部分):全部从机器记录算出,只读。
//   视图 A:某条款的实现累计失败 —— 失败集中在一条需求上,多半是需求没写清。
//   视图 B:某指纹的影响面 —— 关联条款列 LEFT JOIN:无需求层时该列空,页面其余照常。
//
// 两页都需要 s.Trace(*store.Queries,project 包的函数签名);nil → 空状态行动提示。

// ViewAPage 是视图 A(条款累计失败)数据。
type ViewAPage struct {
	Page
	Project string
	Clause  string
	Found   bool

	ImplCommits     int64 // 实现提交数
	BuildFailures   int64 // 构建失败数
	DeployFailures  int64 // 部署失败数
	SameFingerprint int64 // 同一指纹最高复发次数
	ChangeCount     int64 // 条款变更次数
	AgeDays         int   // 条款年龄(天)
	FirstSigned     string

	Failures []ViewAFailure // 失败构成(按指纹)
}

// ViewAFailure 是视图 A 失败构成的一项。
type ViewAFailure struct {
	Fp             string
	Count          int64
	Classification string // 处置分类(fingerprint_stat.status;未归类为 unresolved)
}

// ViewA 渲染视图 A(/views/a?project=<p>&clause=<c>)。
func (s *Server) ViewA(w http.ResponseWriter, r *http.Request) {
	projectID := strings.TrimSpace(r.URL.Query().Get("project"))
	clause := strings.TrimSpace(r.URL.Query().Get("clause"))
	data := ViewAPage{Page: s.page(r, "视图 A "+clause, "coverage"), Project: projectID, Clause: clause}
	if projectID == "" || clause == "" || s.Trace == nil {
		s.Render.Render(w, http.StatusOK, "view_a", data)
		return
	}
	report, err := project.ClauseImpl(r.Context(), s.Trace, core.ProjectID(projectID), clause)
	if err != nil {
		if errors.Is(err, project.ErrClauseNotFound) {
			s.Render.Render(w, http.StatusOK, "view_a", data) // Found=false → 行动提示
			return
		}
		s.serverError(w, r, "算视图 A", err)
		return
	}
	data.Found = true
	data.ImplCommits = report.ImplCommits
	data.BuildFailures = report.BuildFailures
	data.DeployFailures = report.DeployFailures
	data.SameFingerprint = report.SameFingerprint
	data.ChangeCount = report.ChangeCount
	data.AgeDays = report.AgeDays
	data.FirstSigned = report.FirstSignedAt.Local().Format("2006-01-02")
	for _, f := range report.Failures {
		data.Failures = append(data.Failures, ViewAFailure{
			Fp:             string(f.Fingerprint),
			Count:          f.Count,
			Classification: f.Classification,
		})
	}
	s.Render.Render(w, http.StatusOK, "view_a", data)
}

// ViewBPage 是视图 B(指纹影响面)数据。
type ViewBPage struct {
	Page
	Fp            string
	Found         bool
	Occurrences   int64
	Projects      string
	ProjectCount  int
	StatusText    string
	TagClass      string
	HasFix        bool
	FixRef        string
	VerifiedAt    string
	Drifted       bool
	RelatedClause []ViewBClause // 关联条款(LEFT JOIN;无需求层时为空)
}

// ViewBClause 是视图 B 里指纹关联到的一条条款。
type ViewBClause struct {
	ClauseID string
	Hash     string // 短显
}

// ViewB 渲染视图 B(/views/b?fp=<精确指纹>)。
func (s *Server) ViewB(w http.ResponseWriter, r *http.Request) {
	fp := strings.TrimSpace(r.URL.Query().Get("fp"))
	data := ViewBPage{Page: s.page(r, "视图 B "+shortHash(fp), "failures"), Fp: fp}
	if fp == "" || s.Trace == nil {
		s.Render.Render(w, http.StatusOK, "view_b", data)
		return
	}
	impact, err := project.FingerprintImpactView(r.Context(), s.Trace, core.Fingerprint(fp))
	if err != nil {
		if errors.Is(err, project.ErrFingerprintNotFound) {
			s.Render.Render(w, http.StatusOK, "view_b", data) // Found=false → 行动提示
			return
		}
		s.serverError(w, r, "算视图 B", err)
		return
	}
	data.Found = true
	data.Occurrences = impact.Occurrences
	data.Projects = strings.Join(impact.AffectedProjects, " ")
	data.ProjectCount = impact.AffectedProjectCount
	data.StatusText, data.TagClass = classificationView(impact.Status, impact.Drifted)
	data.Drifted = impact.Drifted
	data.HasFix = impact.HasFix
	data.FixRef = impact.FixRef
	data.VerifiedAt = verifiedAtText(impact.VerifiedAt)
	for _, rc := range impact.RelatedClauses {
		data.RelatedClause = append(data.RelatedClause, ViewBClause{
			ClauseID: rc.ClauseID,
			Hash:     shortHash(string(rc.ContentHash)),
		})
	}
	if len(data.RelatedClause) == 0 {
		slog.Debug("视图 B 无关联条款(LEFT JOIN 无需求层)", "fp", fp)
	}
	s.Render.Render(w, http.StatusOK, "view_b", data)
}
