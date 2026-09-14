package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/core"
	"tally/internal/logpipe"
	"tally/internal/store"
)

// Recorder 是 deploy 需要的记账能力。事实面只 INSERT(INV-5)。
// *store.Queries 满足;单测注入替身。
type Recorder interface {
	InsertDeployRun(ctx context.Context, arg store.InsertDeployRunParams) (int64, error)
}

// deploy_run.status 的取值。没有 CHECK 约束,但全项目就这四个。
const (
	StatusDeployed       = "deployed"
	StatusFailed         = "failed"
	StatusRolledBack     = "rolled_back"
	StatusRollbackFailed = "rollback_failed"
)

// Deployer 编排一次部署 / 回滚:调外部工具 → 读退出码 → 记一行 deploy_run。
//
// 不实现部署逻辑(铁律 6):零停机、健康等待、滚动更新全在工具里,
// 本包只转发命令、读退出码、记账。依赖全部注入,便于测试。
type Deployer struct {
	Exec  CommandRunner
	Store Recorder
	Now   func() time.Time // 可注入;缺省 time.Now
}

// Result 是一次部署 / 回滚的结果。
type Result struct {
	RunID    int64
	Status   string // deployed / failed / rolled_back / rollback_failed
	ExitCode int
	Combo    core.VersionCombo
}

func (d *Deployer) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

// Deploy 把定义里的制品部到目标环境:执行工具 → 记一行 deploy_run。
// 成功记 status=deployed;失败记 status=failed,原始输出交 logpipe(source=deploy)。
func (d *Deployer) Deploy(ctx context.Context, def Definition) (Result, error) {
	return d.run(ctx, def, false)
}

// run 是部署 / 回滚的共用骨架。rollback=true 时走工具的回滚命令与对应状态。
func (d *Deployer) run(ctx context.Context, def Definition, rollback bool) (Result, error) {
	if err := def.Validate(); err != nil {
		return Result{}, err
	}
	tl, err := newTool(def)
	if err != nil {
		return Result{}, err
	}
	cmd := tl.deployCmd(def)
	okStatus, failStatus := StatusDeployed, StatusFailed
	if rollback {
		cmd = tl.rollbackCmd(def)
		okStatus, failStatus = StatusRolledBack, StatusRollbackFailed
	}

	started := d.now().UTC()
	res, runErr := d.Exec.Run(ctx, cmd) // 只转发,读退出码;不等不重试
	raw := res.Stdout + "\n" + res.Stderr

	ok := runErr == nil && res.ExitCode == 0
	cause := runErr
	if runErr == nil && !ok {
		cause = fmt.Errorf("%s %s 退出码 %d", tl.name(), actionName(rollback), res.ExitCode)
	}

	combo := core.VersionCombo{
		"artifact_digest": string(def.Digest),
		"tool":            tl.name(),
		"target_env":      def.TargetEnv,
	}

	status := okStatus
	var lp *logpipeSummary
	var processed logpipe.Result
	if !ok {
		status = failStatus
		processed = logpipe.Process(raw) // ← 失败输出在这里进 logpipe
		lp = &logpipeSummary{
			Source:     "deploy", // 与构建/环境失败区分(PRD 第二部分规则 ⑤)
			Excerpt:    processed.Excerpt,
			Normalized: processed.Normalized,
			ExactFP:    string(processed.Exact),
			FuzzyFP:    string(processed.Fuzzy),
		}
	}

	runID, recErr := d.record(ctx, def, tl.name(), started, status, res.ExitCode, runErr == nil, ok, lp, combo)
	result := Result{RunID: runID, Status: status, ExitCode: res.ExitCode, Combo: combo}
	if recErr != nil {
		if cause == nil {
			return result, fmt.Errorf("记录部署: %w", recErr)
		}
		return result, fmt.Errorf("%w;另外记录部署事实出错: %w", cause, recErr)
	}
	if !ok {
		return result, &FailureError{
			RunID:      runID,
			Project:    def.Project,
			TargetEnv:  def.TargetEnv,
			Status:     status,
			Combo:      combo,
			OccurredAt: d.now().UTC(),
			Processed:  processed,
			RawBytes:   int64(len(raw)),
			Cause:      cause,
		}
	}
	return result, nil
}

// FailureError 是部署 / 回滚失败返回的结构化错误:带上刚记下的 deploy_run id、失败当时的
// 版本组合与 logpipe 指纹产出,供编排层(exec / CLI)用 errors.As 取出后录进 failure_event
// (业务包不导入 ingest,方向是 exec → ingest → deploy;见 docs/01-architecture.md)。
//
// Error() 文本与结构化之前完全一致(cause;指纹 X;脱敏片段: Y),既有断言不受影响。
type FailureError struct {
	RunID      int64 // deploy_run(failed / rollback_failed)的行 id:failure_event 的来源外键
	Project    string
	TargetEnv  string
	Status     string            // failed / rollback_failed(与 deploy_run.status 一致)
	Combo      core.VersionCombo // 失败当时的版本组合(与 deploy_run.version_combo 同源)
	OccurredAt time.Time         // 失败发生时刻
	Processed  logpipe.Result    // logpipe 产出(指纹与 health_check_result 里的一致)
	RawBytes   int64             // 原始输出字节数
	Cause      error
}

func (e *FailureError) Error() string {
	return fmt.Sprintf("%v;指纹 %s;脱敏片段: %s", e.Cause, e.Processed.Exact, oneLine(e.Processed.Excerpt))
}

func (e *FailureError) Unwrap() error { return e.Cause }

// record 记一行 deploy_run(只 INSERT,INV-5)。字段齐全:
// digest、目标、工具、结果、退出码、健康检查结果、版本组合。
func (d *Deployer) record(ctx context.Context, def Definition, toolName string, started time.Time,
	status string, exitCode int, ran, ok bool, lp *logpipeSummary, combo core.VersionCombo) (int64, error) {
	hcJSON, err := json.Marshal(healthJSON{OK: ok, By: toolName, Logpipe: lp})
	if err != nil {
		return 0, fmt.Errorf("编码健康检查结果: %w", err)
	}
	comboJSON, err := json.Marshal(combo)
	if err != nil {
		return 0, fmt.Errorf("编码版本组合: %w", err)
	}
	return d.Store.InsertDeployRun(ctx, store.InsertDeployRunParams{
		ProjectID:         def.Project,
		ArtifactDigest:    string(def.Digest), // NOT NULL:没有 digest 的部署到不了这里
		BuildRunID:        pgtype.Int8{Int64: def.BuildRun, Valid: def.BuildRun != 0},
		TargetEnv:         def.TargetEnv,
		Tool:              toolName,
		StartedAt:         pgtype.Timestamptz{Time: started, Valid: true},
		FinishedAt:        pgtype.Timestamptz{Time: d.now().UTC(), Valid: true},
		Status:            status,
		ExitCode:          pgtype.Int4{Int32: int32(exitCode), Valid: ran}, // 没跑完 → NULL
		HealthCheckResult: hcJSON,
		VersionCombo:      comboJSON,
	})
}

// healthJSON 是 deploy_run.health_check_result 的结构。
// 健康检查由外部工具执行,本包只把工具退出码翻译成 ok/不 ok(INV:不等不重试)。
type healthJSON struct {
	OK      bool            `json:"ok"`
	By      string          `json:"by"`                // 由哪个工具的内置健康检查背书
	Logpipe *logpipeSummary `json:"logpipe,omitempty"` // 仅失败时:logpipe 处理结果
}

// logpipeSummary 是失败时存进 health_check_result 的 logpipe 产出。
type logpipeSummary struct {
	Source     string `json:"source"` // 固定 "deploy":失败来源
	Excerpt    string `json:"excerpt"`
	Normalized string `json:"normalized"`
	ExactFP    string `json:"exact_fp"`
	FuzzyFP    string `json:"fuzzy_fp"`
}

func actionName(rollback bool) string {
	if rollback {
		return "rollback"
	}
	return "deploy"
}

// oneLine 把多行片段压成一行并截断,用于 error 文本(内容已脱敏)。
func oneLine(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " | "))
	const max = 200
	if len(s) > max {
		s = s[:max] + "…"
	}
	return s
}
