package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"tally/internal/store"
)

// 指纹详情(视图 B 的执行侧):影响项目、关联条款(LEFT JOIN)、已验证修复、是否漂移、发生记录、模糊邻居。
//
// 关联条款来自 LEFT JOIN:指纹的失败经 build/deploy 回到 commit,再经 commit_clause 到条款。
// 没有需求层(推不出条款)时关联条款区不渲染,页面其余部分照常工作 —— 这正是它可以先上线的原因。

// FpEvent 是一条发生记录。
type FpEvent struct {
	Time       string
	Project    string
	SourceText string
	Commit     string // 短显;环境来源的失败没有 commit 链,为空
	Excerpt    string
}

// FpClause 是指纹关联到的一条条款(LEFT JOIN;无需求层时不渲染)。
type FpClause struct {
	ClauseID string
	Hash     string // 短显
}

// FingerprintPage 是指纹详情页数据。
type FingerprintPage struct {
	Page
	Fp               string
	Found            bool
	Count            int64
	FirstSeen        string
	LastSeen         string
	ProjectNames     string
	ProjectCount     int
	StatusText       string
	TagClass         string
	DiagnoseAttempts int32
	HasFix           bool
	FixRef           string
	FixVerifiedAt    string
	FixCombo         string
	Drifted          bool
	RelatedClauses   []FpClause // 关联条款(LEFT JOIN;无需求层为空,页面其余照常)
	Events           []FpEvent
	Neighbors        []FailureRow
}

// FingerprintDetail 渲染指纹详情(/fingerprint?fp=<精确指纹>)。
func (s *Server) FingerprintDetail(w http.ResponseWriter, r *http.Request) {
	fp := strings.TrimSpace(r.URL.Query().Get("fp"))
	data := FingerprintPage{Page: s.page(r, "指纹 "+shortHash(fp), "failures"), Fp: fp}
	if fp == "" || s.Exec == nil {
		s.Render.Render(w, http.StatusOK, "fingerprint", data)
		return
	}
	ctx := r.Context()

	st, err := s.Exec.GetFingerprintStat(ctx, fp)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			s.serverError(w, r, "读指纹索引", err)
			return
		}
		s.Render.Render(w, http.StatusOK, "fingerprint", data) // Found=false → 行动提示
		return
	}

	data.Found = true
	data.Count = st.OccurrenceCount
	data.FirstSeen = fmtTime(st.FirstSeen)
	data.LastSeen = fmtTime(st.LastSeen)
	data.ProjectNames = strings.Join(st.AffectedProjects, " ")
	data.ProjectCount = len(st.AffectedProjects)
	data.DiagnoseAttempts = st.DiagnoseAttempts
	data.StatusText, data.TagClass = classificationView(st.Status, false)
	data.HasFix = st.FixRef.Valid
	data.FixRef = st.FixRef.String
	data.FixVerifiedAt = fmtTime(st.FixVerifiedAt)
	data.FixCombo = comboBrief(st.FixVerifiedCombo)

	// 视图 B:漂移判定(修复验证后又复发);关联条款列(LEFT JOIN,无需求层为空)。
	if impact, ierr := s.Exec.ViewFingerprintImpact(ctx, fp); ierr != nil {
		slog.Error("读指纹影响面", "fp", fp, "err", ierr)
	} else {
		if len(impact) > 0 && impact[0].Drifted.Bool {
			data.Drifted = true
			data.StatusText, data.TagClass = classificationView(st.Status, true)
		}
		data.RelatedClauses = fpRelatedClauses(impact)
	}

	data.Events = s.fpEvents(ctx, fp)
	data.Neighbors = s.fpNeighbors(ctx, fp, st.FpFuzzy)

	s.Render.Render(w, http.StatusOK, "fingerprint", data)
}

// fpRelatedClauses 从影响面行收集非空关联条款(LEFT JOIN 无匹配为 NULL,跳过),去重保序。
// 没有需求层时返回空 —— 页面关联条款区整体不渲染,其余照常。
func fpRelatedClauses(impact []store.ViewFingerprintImpactRow) []FpClause {
	seen := map[string]bool{}
	var out []FpClause
	for _, r := range impact {
		if !r.RelatedClauseID.Valid || seen[r.RelatedClauseID.String] {
			continue
		}
		seen[r.RelatedClauseID.String] = true
		out = append(out, FpClause{ClauseID: r.RelatedClauseID.String, Hash: shortHash(r.RelatedClauseHash.String)})
	}
	return out
}

// fpEvents 取指纹的发生记录(新→旧),commit 短显。
func (s *Server) fpEvents(ctx context.Context, fp string) []FpEvent {
	evs, err := s.Exec.ListFailureEventsByFingerprint(ctx, fp)
	if err != nil {
		slog.Error("读指纹发生记录", "fp", fp, "err", err)
		return nil
	}
	out := make([]FpEvent, 0, len(evs))
	for _, ev := range evs {
		commit := ev.CommitSha
		if len(commit) > 6 {
			commit = commit[:6]
		}
		out = append(out, FpEvent{
			Time:       fmtTime(ev.OccurredAt),
			Project:    ev.ProjectID,
			SourceText: sourceText(ev.Source),
			Commit:     commit,
			Excerpt:    excerptFirstLine(ev.Excerpt.String),
		})
	}
	return out
}

// fpNeighbors 取同模糊指纹的邻居(排除自身,排行分前 5)—— 只作诊断线索,不能当答案。
func (s *Server) fpNeighbors(ctx context.Context, fp, fpFuzzy string) []FailureRow {
	nbs, err := s.Exec.ListFingerprintStatsByFuzzy(ctx, fpFuzzy)
	if err != nil {
		slog.Error("读模糊指纹邻居", "fp", fp, "err", err)
		return nil
	}
	var others []store.FingerprintStat
	for _, nb := range nbs {
		if nb.FpExact != fp {
			others = append(others, nb)
		}
	}
	if len(others) == 0 {
		return nil
	}
	excerpts, err := s.Exec.ListLatestFailureExcerpts(ctx)
	if err != nil {
		slog.Error("读邻居摘录", "fp", fp, "err", err)
		excerpts = nil
	}
	return topRanking(failureRows(others, excerpts), 5)
}
