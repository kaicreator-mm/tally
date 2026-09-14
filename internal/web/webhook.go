package web

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tally/internal/core"
	"tally/internal/ingest"
)

// webhook 尺寸上限(D-d4):请求体 12MB 封顶,其中 log 单条上限 10MB(PRD 同限)。
const (
	maxWebhookBodyBytes = 12 << 20
	maxWebhookLogBytes  = 10 << 20
)

// DiagnoseEnqueuer 是自动诊断的入队出口(T11e):按精确指纹入队 diagnose 任务。
// *jobs.Queue 天然满足;定义在消费方,web 不因此依赖队列的装配细节。
type DiagnoseEnqueuer interface {
	EnqueueDiagnose(ctx context.Context, project, fingerprint string) (int64, error)
}

// buildPayload 是 webhook 请求体(通用 JSON,不做 CI 平台专属适配;CI 侧一行脚本拼装)。
// 字段与 tally ingest-build 的 flag 一一对应,录入语义完全一致(D-d5)。
type buildPayload struct {
	Project      string            `json:"project"`
	Commit       string            `json:"commit"`
	Toolchain    string            `json:"toolchain"`
	Status       string            `json:"status"`    // passed / failed;空 → failed
	ExitCode     *int              `json:"exit_code"` // 空 → passed 取 0、failed 取 1
	CiURL        string            `json:"ci_url"`
	Artifact     string            `json:"artifact"`
	RawRef       string            `json:"raw_ref"` // 可空;缺省用留存路径
	Combo        map[string]string `json:"combo"`
	Workdir      string            `json:"workdir"`
	Repro        string            `json:"repro"`
	OtherTargets []string          `json:"other_targets"`
	Log          string            `json:"log"`
}

// webhookResponse 是 webhook 应答体。
type webhookResponse struct {
	BuildRun     int64  `json:"build_run"`
	FailureEvent int64  `json:"failure_event"` // 成功构建恒为 0
	RawLogRef    string `json:"raw_log_ref"`
}

// webhookBuild 处理 POST /webhook/build:token 校验 → 解 JSON → 存日志 → IngestBuild → 回 JSON。
//
// 鉴权(D-d1):单一静态 token(TALLY_WEBHOOK_TOKEN),常量时间比较,走 Authorization: Bearer。
// 未配置 token → 503(明确「未启用」);token 错 → 401。两者都非 2xx,
// INV-9 的行为式架构测试(未登录写请求必须被拒)不改而过(D-d2)。
// 本路由不在登录组内(token 就是它的登录),自然也不在 CSRF 组内(D-d3:非浏览器交互)。
func (d Deps) webhookBuild(w http.ResponseWriter, r *http.Request) {
	if d.WebhookToken == "" {
		writeWebhookJSON(w, http.StatusServiceUnavailable, "webhook 未启用:未配置 TALLY_WEBHOOK_TOKEN")
		return
	}
	if !webhookTokenOK(d.WebhookToken, r) {
		writeWebhookJSON(w, http.StatusUnauthorized, "token 缺失或不正确")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxWebhookBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeWebhookJSON(w, http.StatusRequestEntityTooLarge, "请求体超过 12MB 封顶")
		return
	}
	var p buildPayload
	if err := json.Unmarshal(body, &p); err != nil {
		writeWebhookJSON(w, http.StatusBadRequest, "请求体不是合法 JSON: "+err.Error())
		return
	}
	if len(p.Log) > maxWebhookLogBytes {
		writeWebhookJSON(w, http.StatusRequestEntityTooLarge, "log 超过 10MB 上限")
		return
	}
	if strings.TrimSpace(p.Project) == "" || strings.TrimSpace(p.Commit) == "" {
		writeWebhookJSON(w, http.StatusBadRequest, "project 与 commit 必填")
		return
	}
	if !safePathComponent(p.Project) || !safePathComponent(p.Commit) {
		writeWebhookJSON(w, http.StatusBadRequest, "project/commit 只允许字母数字与 . _ -(防路径穿越)")
		return
	}
	status := p.Status
	if status == "" {
		status = ingest.StatusFailed
	}
	if status != ingest.StatusPassed && status != ingest.StatusFailed {
		writeWebhookJSON(w, http.StatusBadRequest, "status 应是 passed/failed,得到 "+status)
		return
	}
	if d.IngestStore == nil {
		writeWebhookJSON(w, http.StatusServiceUnavailable, "webhook 未启用:server 未接数据库")
		return
	}

	// 原始日志留存(D-d4,PRD「原始日志留一份,规则改了能重跑」)。
	// 请求给了 raw_ref 就用它(CI 自己有留存时);否则落盘到 LogDir。
	ref := p.RawRef
	if ref == "" && d.LogDir != "" {
		ref, err = saveRawLog(d.LogDir, p.Project, p.Commit, time.Now().UTC(), p.Log)
		if err != nil {
			writeWebhookJSON(w, http.StatusInternalServerError, "留存原始日志出错: "+err.Error())
			return
		}
	}

	exitCode := 0
	if p.ExitCode != nil {
		exitCode = *p.ExitCode
	} else if status == ingest.StatusFailed {
		exitCode = 1
	}
	now := time.Now().UTC()
	runID, eventID, fp, err := ingest.IngestBuild(r.Context(), d.IngestStore, ingest.BuildInput{
		Input: ingest.Input{
			ProjectID:    p.Project,
			Raw:          p.Log,
			Toolchain:    p.Toolchain,
			RawLogRef:    ref,
			Combo:        core.VersionCombo(p.Combo),
			OccurredAt:   now,
			Workdir:      p.Workdir,
			Repro:        p.Repro,
			OtherTargets: p.OtherTargets,
		},
		CommitSha:      p.Commit,
		CiRunURL:       p.CiURL,
		Status:         status,
		ExitCode:       exitCode,
		StartedAt:      now,
		FinishedAt:     now,
		ArtifactDigest: p.Artifact,
	})
	if err != nil {
		writeWebhookJSON(w, http.StatusInternalServerError, "录入构建出错: "+err.Error())
		return
	}
	d.maybeAutoDiagnose(r.Context(), p.Project, fp)
	writeWebhookResult(w, webhookResponse{BuildRun: runID, FailureEvent: eventID, RawLogRef: ref})
}

// maybeAutoDiagnose 在开关打开时(Deps.DiagnoseAuto,来自 TALLY_DIAGNOSE_AUTO=1),
// 给刚录入的失败自动入队一次 diagnose(T11e,D-e1 默认关)。成功构建 fp 为空,天然不入队。
//
// 尽力而为:入队出错只记日志,不让录入应答失败(failure_event 已落库);
// 预算不在入队侧预检(D-e2),worker 执行时 failure.Handle 对超预算指纹零 LLM 短路;
// 与手动重诊断共用同一 handler(D-e4)。
func (d Deps) maybeAutoDiagnose(ctx context.Context, project string, fp core.Fingerprint) {
	if !d.DiagnoseAuto || d.DiagnoseQueue == nil || fp == "" {
		return
	}
	if _, err := d.DiagnoseQueue.EnqueueDiagnose(ctx, project, string(fp)); err != nil {
		slog.Warn("自动诊断入队出错(webhook 录入不受影响)", "project", project, "fingerprint", fp, "err", err)
	}
}

// webhookTokenOK 常量时间比较请求 token 与配置 token(D-d1)。
func webhookTokenOK(want string, r *http.Request) bool {
	const prefix = "Bearer "
	got := r.Header.Get("Authorization")
	if !strings.HasPrefix(got, prefix) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(strings.TrimSpace(got[len(prefix):])), []byte(want)) == 1
}

// saveRawLog 把原始日志落盘,返回留存路径(记进 raw_log_ref):
// <LogDir>/<项目>/<UTC时刻>-<commit 前 12 位>.log。时刻精确到纳秒,避免同秒碰撞。
func saveRawLog(dir, project, commit string, at time.Time, log string) (string, error) {
	sub := filepath.Join(dir, project)
	if err := os.MkdirAll(sub, 0o755); err != nil {
		return "", fmt.Errorf("建日志目录: %w", err)
	}
	short := commit
	if len(short) > 12 {
		short = short[:12]
	}
	name := at.Format("20060102T150405.000000000Z") + "-" + short + ".log"
	path := filepath.Join(sub, name)
	if err := os.WriteFile(path, []byte(log), 0o644); err != nil {
		return "", fmt.Errorf("写日志文件: %w", err)
	}
	return path, nil
}

// safePathComponent 判定 s 可安全用作路径成分:只允许字母数字与 . _ -,拒绝空/./..。
// project/commit 来自网络输入,落盘前必须过这一关(防路径穿越)。
func safePathComponent(s string) bool {
	if s == "" || s == "." || s == ".." {
		return false
	}
	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '.' || c == '-' || c == '_':
		default:
			return false
		}
	}
	return true
}

// writeWebhookJSON 写错误应答 {"error": msg}。
func writeWebhookJSON(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// writeWebhookResult 写成功应答(200)。
func writeWebhookResult(w http.ResponseWriter, res webhookResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}
