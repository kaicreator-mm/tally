package handlers

import (
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"tally/internal/store"
)

// 失败页:按指纹归并(一个指纹一行,不是 N 次 N 行)+ 修复排行 + 失败时间线 × 依赖升级对齐。
// 数据来自 fingerprint_stat(索引面,可随时重算)与 failure_event/dependency_change(事实面,只读)。

// FailureRow 是一个指纹的归并行。
type FailureRow struct {
	Fp           string
	Excerpt      string // 最近一次的脱敏片段首行
	Source       string
	SourceText   string
	Count        int64
	Projects     int
	ProjectNames string
	Rank         int64 // 出现次数 × 影响项目数
	LastSeen     string
	StatusText   string
	TagClass     string
}

// FailuresPage 是失败页数据。
type FailuresPage struct {
	Page
	Rows      []FailureRow  // 归并列表,最近出现在前
	Ranking   []FailureRow  // 修复排行,排行分降序取前 10
	Timeline  []TimelineDay // 近 90 天按天失败 × 依赖刻度
	AxisFirst string
	AxisLast  string
	Window    string
}

// ListFailures 渲染失败页。任一读源出错 → 该区块落空,不 500。
func (s *Server) ListFailures(w http.ResponseWriter, r *http.Request) {
	data := FailuresPage{Page: s.page(r, "失败", "failures")}
	data.Window = "近 90 天"
	if s.Exec != nil {
		ctx := r.Context()
		stats, err1 := s.Exec.ListFingerprintStats(ctx)
		excerpts, err2 := s.Exec.ListLatestFailureExcerpts(ctx)
		if err1 != nil || err2 != nil {
			slog.Error("读失败归并", "stats", err1, "excerpts", err2)
		} else {
			data.Rows = failureRows(stats, excerpts)
			data.Ranking = topRanking(data.Rows, 10)
		}
		days, err3 := s.Exec.CountFailuresByDay(ctx)
		deps, err4 := s.Exec.ListDependencyChangesRecent(ctx)
		if err3 != nil || err4 != nil {
			slog.Error("读失败时间线", "days", err3, "deps", err4)
		} else {
			data.Timeline = buildTimeline(days, deps, time.Now(), timelineWindowDays)
			if n := len(data.Timeline); n > 0 {
				data.AxisFirst = data.Timeline[0].Full
				data.AxisLast = data.Timeline[n-1].Full
			}
		}
	}
	s.Render.Render(w, http.StatusOK, "failures", data)
}

/* ---------- 归并与排行(纯函数,单测钉住) ---------- */

// failureRows 把指纹索引行与最近摘录拼成归并行。
// 漂移:已修复(fixed)但末次出现晚于修复验证时刻 —— 修复后又复发,标「修复可能过期」。
func failureRows(stats []store.FingerprintStat, excerpts []store.ListLatestFailureExcerptsRow) []FailureRow {
	exc := make(map[string]store.ListLatestFailureExcerptsRow, len(excerpts))
	for _, e := range excerpts {
		exc[e.FpExact] = e
	}
	rows := make([]FailureRow, 0, len(stats))
	for _, st := range stats {
		drifted := st.Status == "fixed" &&
			st.FixVerifiedAt.Valid && st.LastSeen.Valid &&
			st.LastSeen.Time.After(st.FixVerifiedAt.Time)
		text, class := classificationView(st.Status, drifted)
		e := exc[st.FpExact]
		rows = append(rows, FailureRow{
			Fp:           st.FpExact,
			Excerpt:      excerptFirstLine(e.Excerpt.String),
			Source:       e.Source,
			SourceText:   sourceText(e.Source),
			Count:        st.OccurrenceCount,
			Projects:     len(st.AffectedProjects),
			ProjectNames: strings.Join(st.AffectedProjects, " "),
			Rank:         rankValue(st.OccurrenceCount, len(st.AffectedProjects)),
			LastSeen:     fmtTime(st.LastSeen),
			StatusText:   text,
			TagClass:     class,
		})
	}
	return rows
}

// topRanking 取排行分降序前 n(同分按指纹,序稳定)—— 该先修哪个。
func topRanking(rows []FailureRow, n int) []FailureRow {
	sorted := make([]FailureRow, len(rows))
	copy(sorted, rows)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Rank != sorted[j].Rank {
			return sorted[i].Rank > sorted[j].Rank
		}
		return sorted[i].Fp < sorted[j].Fp
	})
	if len(sorted) > n {
		sorted = sorted[:n]
	}
	return sorted
}
