package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"tally/internal/core"
	"tally/internal/knowledge"
	"tally/internal/store"
)

// 已验证修复库浏览(只读,无任何编辑 / 新增入口,INV-2):
// 条目只能经失败处理流程(四道闸验证)入库,界面读它、绝不写它。
//
// 双层区分(见 docs/06-ui-spec.md ④):验证库条目用 --verified 绿 +「已验证」标(badge-verified),
// 与 Wiki 笔记层(--stale 赭石 +「笔记·未验证」,badge-note)在颜色与徽章上明确可分。

// KnowledgeSource 是验证库的只读浏览来源(使用方定义接口)。*knowledge.Store 满足它。
// nil → 验证库页渲染空状态行动提示。
type KnowledgeSource interface {
	ListVerified(ctx context.Context) ([]knowledge.Entry, error)
}

// FixEntry 是页面上的一条已验证修复(纯展示)。
type FixEntry struct {
	Fp         string
	Repro      string
	Summary    string
	Gates      string
	VerifiedAt string
	Combo      string
	Stale      bool
	Projects   string
	Count      int64
}

// KnowledgePage 是验证库浏览页数据。结构体里没有任何表单字段 —— 模板不渲染编辑入口。
type KnowledgePage struct {
	Page
	Entries []FixEntry
}

// ListKnowledge 渲染已验证修复库(按指纹浏览,只读)。
func (s *Server) ListKnowledge(w http.ResponseWriter, r *http.Request) {
	data := KnowledgePage{Page: s.page(r, "已验证修复", "knowledge")}
	if s.Knowledge != nil {
		entries, err := s.Knowledge.ListVerified(r.Context())
		if err != nil {
			slog.Error("读验证库", "err", err)
		} else {
			data.Entries = s.fixEntries(r.Context(), entries)
		}
	}
	s.Render.Render(w, http.StatusOK, "knowledge", data)
}

// fixEntries 把验证库条目与指纹索引拼成页面行:索引补上影响项目 / 出现次数 / 漂移(修复后又复发)。
func (s *Server) fixEntries(ctx context.Context, entries []knowledge.Entry) []FixEntry {
	statByFp := map[string]store.FingerprintStat{}
	if s.Exec != nil {
		stats, err := s.Exec.ListFingerprintStats(ctx)
		if err != nil {
			slog.Error("读指纹索引(验证库漂移判定)", "err", err)
		} else {
			for _, st := range stats {
				statByFp[st.FpExact] = st
			}
		}
	}
	out := make([]FixEntry, 0, len(entries))
	for _, e := range entries {
		st := statByFp[e.Fingerprint]
		drifted := st.Status == "fixed" && st.FixVerifiedAt.Valid && st.LastSeen.Valid &&
			st.LastSeen.Time.After(st.FixVerifiedAt.Time)
		out = append(out, FixEntry{
			Fp:         e.Fingerprint,
			Repro:      e.Repro,
			Summary:    e.Summary,
			Gates:      e.Gates,
			VerifiedAt: verifiedAtText(e.VerifiedAt),
			Combo:      comboText(e.VerifiedCombo),
			Stale:      e.MarkedStale || drifted,
			Projects:   strings.Join(st.AffectedProjects, " "),
			Count:      st.OccurrenceCount,
		})
	}
	return out
}

// comboText 把版本组合压成一行供验证库条目展示。
// 排序与截断规则与 comboBrief 同源(见 formatComboMap),空组合落空串。
func comboText(combo core.VersionCombo) string {
	return formatComboMap(combo)
}
