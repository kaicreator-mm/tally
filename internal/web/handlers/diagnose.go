package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"tally/internal/failure"
	"tally/internal/jobs"
)

// 诊断状态 + 触发重诊断。
//
// 状态是算出来的(INV-3):第几轮(diagnose_attempts)/ 五分类(fingerprint_stat.status)都从索引读。
// 触发重诊断只入队 jobs.KindDiagnose(INV-5:界面不碰事实面),worker 执行时才写新记录。
//
// 三轮预算由后端守(PRD 第四部分规则 ⑤):入队前用 failure.ConsultBudget 判,
// 已达上限 / 已有定论一律拒绝入队 —— 界面只显示结果,不提供任何绕过。

// diagnoseMaxAttempts 是展示用的诊断轮数上限(与 failure 包的 maxAttempts 同源语义)。
const diagnoseMaxAttempts = 3

// DiagnosePage 是诊断状态页数据。
type DiagnosePage struct {
	Page
	Fp            string
	Found         bool
	StatusText    string // 五分类当前文案
	TagClass      string
	Attempts      int32
	RoundText     string // 已诊断 N / 3 轮
	BudgetState   string // 可再诊断 / 已达上限 / 已有定论
	CanDiagnose   bool   // 是否显示「触发重诊断」按钮
	ConcludedText string // 终态出口文案(已有定论时)
	Project       string // 回显用:受影响项目之一
}

// DiagnoseStatus 渲染诊断状态页(/diagnose?fp=<精确指纹>)。
func (s *Server) DiagnoseStatus(w http.ResponseWriter, r *http.Request) {
	fp := strings.TrimSpace(r.URL.Query().Get("fp"))
	data := DiagnosePage{Page: s.page(r, "诊断 "+shortHash(fp), "failures"), Fp: fp}
	if fp == "" || s.Exec == nil {
		s.Render.Render(w, http.StatusOK, "diagnose", data)
		return
	}
	st, err := s.Exec.GetFingerprintStat(r.Context(), fp)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			s.serverError(w, r, "读指纹索引", err)
			return
		}
		s.Render.Render(w, http.StatusOK, "diagnose", data) // Found=false → 行动提示
		return
	}

	data.Found = true
	data.Attempts = st.DiagnoseAttempts
	data.RoundText = diagnoseRoundText(st.DiagnoseAttempts)
	data.StatusText, data.TagClass = classificationView(st.Status, false)
	if len(st.AffectedProjects) > 0 {
		data.Project = st.AffectedProjects[0]
	}

	// 预算判定(确定性,不触 LLM):终态短路 > 三轮上限 > 放行。
	decision := failure.ConsultBudget(true, failure.Stat{
		Status:           st.Status,
		DiagnoseAttempts: int(st.DiagnoseAttempts),
		AffectedProjects: st.AffectedProjects,
		FixRef:           st.FixRef.String,
	})
	switch {
	case decision.Concluded != nil:
		data.BudgetState = "已有定论"
		data.ConcludedText, _ = classificationView(string(*decision.Concluded), false)
		data.CanDiagnose = false
	case decision.Archived:
		data.BudgetState = "已达上限"
		data.CanDiagnose = false
	default:
		data.BudgetState = "可再诊断"
		data.CanDiagnose = true
	}
	s.Render.Render(w, http.StatusOK, "diagnose", data)
}

// DiagnoseTrigger 触发重诊断:校验三轮预算后只入队一个 diagnose job(PRG 重定向回去)。
//
// 后端守预算:已达上限 / 已有定论都拒绝入队。即便绕过界面手工 POST,也过不了这道闸。
func (s *Server) DiagnoseTrigger(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单解析失败", http.StatusBadRequest)
		return
	}
	fp := strings.TrimSpace(r.FormValue("fp"))
	back := "/diagnose?fp=" + fp
	if fp == "" {
		redirectFlash(w, r, "/failures", "bad", "缺少指纹,无法触发诊断")
		return
	}
	if s.Exec == nil {
		redirectFlash(w, r, back, "bad", "执行数据未接入,无法判断诊断预算")
		return
	}
	st, err := s.Exec.GetFingerprintStat(r.Context(), fp)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			redirectFlash(w, r, back, "bad", "指纹尚未索引,先跑一次构建让失败入库")
			return
		}
		s.serverError(w, r, "读指纹索引", err)
		return
	}
	decision := failure.ConsultBudget(true, failure.Stat{
		Status:           st.Status,
		DiagnoseAttempts: int(st.DiagnoseAttempts),
		AffectedProjects: st.AffectedProjects,
		FixRef:           st.FixRef.String,
	})
	switch {
	case decision.Concluded != nil:
		text, _ := classificationView(string(*decision.Concluded), false)
		redirectFlash(w, r, back, "warn", "已有定论("+text+"),不再诊断")
		return
	case decision.Archived:
		redirectFlash(w, r, back, "warn", "已达上限(3 轮),不再诊断")
		return
	}
	project := strings.TrimSpace(r.FormValue("project"))
	s.enqueue(w, r, jobs.KindDiagnose, jobs.DiagnosePayload{Fingerprint: fp, Project: project}, back)
}

// diagnoseRoundText 渲染「已诊断 N / 3 轮」。
func diagnoseRoundText(attempts int32) string {
	return "已诊断 " + strconv.Itoa(int(attempts)) + " / " + strconv.Itoa(diagnoseMaxAttempts) + " 轮"
}
