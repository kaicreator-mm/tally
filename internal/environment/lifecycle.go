package environment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/core"
	"tally/internal/logpipe"
	"tally/internal/runner"
	"tally/internal/store"
)

// 下面三个接口都**在使用方定义**(见 docs/04-code-standards.md「接口」):
// runner 包提供实现(*runner.DockerRunner / *runner.SSHRunner 结构上满足),
// store 提供记录能力(*store.Queries 满足)。单测注入替身即可离线跑。

// DockerRunner 是 environment 需要的 docker 执行能力。
type DockerRunner interface {
	ComposeUp(ctx context.Context, file, project string) (runner.Result, error)
	ComposeDown(ctx context.Context, file, project string) (runner.Result, error)
	Pull(ctx context.Context, image string) (runner.Result, error)
	ComposeImages(ctx context.Context, file, project string) ([]string, error)
	ComposeLogs(ctx context.Context, file, project, service string) (runner.Result, error)
	ServerVersion(ctx context.Context) (string, error)
	ComposeVersion(ctx context.Context) (string, error)
}

// SSHRunner 是 environment 需要的远端执行能力。
type SSHRunner interface {
	Run(ctx context.Context, cmd runner.Command) (runner.Result, error)
}

// RunRecorder 是 environment 记录拉起事实的能力。事实面只 INSERT(INV-5)。
type RunRecorder interface {
	InsertEnvironmentRunOutcome(ctx context.Context, arg store.InsertEnvironmentRunOutcomeParams) (int64, error)
}

// environment_run.status 的合法取值(与迁移 0001 的 CHECK 约束一致)。
const (
	StatusUp       = "up"
	StatusFailed   = "failed"
	StatusTornDown = "torn_down"
)

// Manager 编排环境的拉起 / 推倒 / 重建。依赖全部注入,便于测试。
type Manager struct {
	Docker DockerRunner
	SSH    SSHRunner
	Store  RunRecorder
	Now    func() time.Time // 可注入;缺省 time.Now
}

// UpResult 是一次成功拉起的结果。
type UpResult struct {
	RunID      int64
	Combo      core.VersionCombo
	Health     HealthReport
	Definition string // 定义内容哈希,回显(INV-4)
}

func (m *Manager) now() time.Time {
	if m.Now != nil {
		return m.Now()
	}
	return time.Now()
}

// Up 按定义把环境拉起来:执行 → 健康检查 → 记 environment_run。
// 成功记一行 status=up(version_combo 非空);失败记一行 status=failed 并把原始输出交 logpipe。
func (m *Manager) Up(ctx context.Context, def Definition) (UpResult, error) {
	switch def.Kind {
	case KindDockerCompose:
		return m.upCompose(ctx, def)
	case KindSSHScript:
		return m.upSSH(ctx, def)
	default:
		return UpResult{}, fmt.Errorf("环境 %q 的 kind %q 不支持", def.Name, def.Kind)
	}
}

// upCompose 走 docker compose 拉起。
func (m *Manager) upCompose(ctx context.Context, def Definition) (UpResult, error) {
	started := m.now().UTC()
	file, project := def.ComposeFilePath(), def.ComposeProject()
	var raw strings.Builder

	// 预拉镜像,让 tag 落到本地,版本组合才记得到。拉取失败不致命(镜像可能已在本地)。
	if imgs, err := m.Docker.ComposeImages(ctx, file, project); err == nil {
		for _, img := range imgs {
			if res, perr := m.Docker.Pull(ctx, img); perr == nil && res.ExitCode != 0 {
				fmt.Fprintf(&raw, "pull %s:\n%s\n", img, res.Stderr)
			}
		}
	}

	res, err := m.Docker.ComposeUp(ctx, file, project)
	if err != nil {
		return UpResult{}, m.fail(ctx, def, started, dockerCombo(ctx, m, file, project, &raw),
			HealthReport{}, fmt.Errorf("docker compose up 执行失败: %w", err), raw.String())
	}
	fmt.Fprintf(&raw, "compose up:\n%s\n%s\n", res.Stdout, res.Stderr)
	if res.ExitCode != 0 {
		return UpResult{}, m.fail(ctx, def, started, dockerCombo(ctx, m, file, project, &raw),
			HealthReport{}, fmt.Errorf("docker compose up 退出码 %d", res.ExitCode), raw.String())
	}
	m.appendLogs(ctx, &raw, file, project) // 容器启动日志:健康检查失败时给 logpipe 更多线索

	combo := dockerCombo(ctx, m, file, project, &raw)

	fetchLogs := func(c context.Context, service string) (string, error) {
		lr, lerr := m.Docker.ComposeLogs(c, file, project, service)
		if lerr != nil {
			return "", lerr
		}
		return lr.Stdout + "\n" + lr.Stderr, nil
	}
	report, herr := WaitHealthy(ctx, def.HealthChecks, def.Wait, fetchLogs)
	if herr != nil {
		m.appendLogs(ctx, &raw, file, project)
		return UpResult{}, m.fail(ctx, def, started, combo, report,
			fmt.Errorf("健康检查中断: %w", herr), raw.String())
	}
	if !report.OK {
		m.appendLogs(ctx, &raw, file, project)
		return UpResult{}, m.fail(ctx, def, started, combo, report,
			fmt.Errorf("环境 %q 健康检查未通过", def.Name), raw.String())
	}

	runID, rerr := m.record(ctx, def, started, StatusUp, report, combo, nil)
	if rerr != nil {
		return UpResult{}, fmt.Errorf("记录环境拉起: %w", rerr)
	}
	return UpResult{RunID: runID, Combo: combo, Health: report, Definition: def.Hash}, nil
}

// upSSH 走 SSH 脚本拉起。
func (m *Manager) upSSH(ctx context.Context, def Definition) (UpResult, error) {
	if m.SSH == nil {
		return UpResult{}, errors.New("未配置 SSH 执行器,无法拉起 ssh-script 环境")
	}
	started := m.now().UTC()
	var raw strings.Builder
	combo := m.sshCombo(ctx, def, &raw)

	res, err := m.SSH.Run(ctx, runner.Command{Argv: []string{def.Script}})
	if err != nil {
		return UpResult{}, m.fail(ctx, def, started, combo, HealthReport{},
			fmt.Errorf("SSH 执行失败: %w", err), raw.String())
	}
	fmt.Fprintf(&raw, "ssh script:\n%s\n%s\n", res.Stdout, res.Stderr)
	if res.ExitCode != 0 {
		return UpResult{}, m.fail(ctx, def, started, combo, HealthReport{},
			fmt.Errorf("SSH 脚本退出码 %d", res.ExitCode), raw.String())
	}

	report, herr := WaitHealthy(ctx, def.HealthChecks, def.Wait, m.sshLogFetcher(def))
	if herr != nil {
		return UpResult{}, m.fail(ctx, def, started, combo, report,
			fmt.Errorf("健康检查中断: %w", herr), raw.String())
	}
	if !report.OK {
		return UpResult{}, m.fail(ctx, def, started, combo, report,
			fmt.Errorf("环境 %q 健康检查未通过", def.Name), raw.String())
	}

	runID, rerr := m.record(ctx, def, started, StatusUp, report, combo, nil)
	if rerr != nil {
		return UpResult{}, fmt.Errorf("记录环境拉起: %w", rerr)
	}
	return UpResult{RunID: runID, Combo: combo, Health: report, Definition: def.Hash}, nil
}

// Down 推倒环境,成功后记一行 status=torn_down(INV-5 只追加)。
//
// docker-compose 用 `down --volumes --remove-orphans`:卷、网络、孤儿容器一并清掉,不残留。
// ssh-script 跑 teardown 脚本(若定义);无 teardown 脚本时仍记 torn_down(调用方意图即推倒)。
func (m *Manager) Down(ctx context.Context, def Definition) error {
	switch def.Kind {
	case KindDockerCompose:
		res, err := m.Docker.ComposeDown(ctx, def.ComposeFilePath(), def.ComposeProject())
		if err != nil {
			return fmt.Errorf("docker compose down 执行失败: %w", err)
		}
		if res.ExitCode != 0 {
			return fmt.Errorf("docker compose down 退出码 %d: %s", res.ExitCode, strings.TrimSpace(res.Stderr))
		}
		return m.recordTornDown(ctx, def)
	case KindSSHScript:
		if def.Teardown != "" {
			if m.SSH == nil {
				return errors.New("未配置 SSH 执行器,无法推倒 ssh-script 环境")
			}
			res, err := m.SSH.Run(ctx, runner.Command{Argv: []string{def.Teardown}})
			if err != nil {
				return fmt.Errorf("SSH 推倒脚本执行失败: %w", err)
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("SSH 推倒脚本退出码 %d: %s", res.ExitCode, strings.TrimSpace(res.Stderr))
			}
		}
		return m.recordTornDown(ctx, def)
	default:
		return fmt.Errorf("环境 %q 的 kind %q 不支持", def.Name, def.Kind)
	}
}

// recordTornDown 记一行 status=torn_down(INV-5 只追加)。
// version_combo 只记 kind(推倒时不采集镜像版本);health_check_result 为 null(推倒无健康检查)。
func (m *Manager) recordTornDown(ctx context.Context, def Definition) error {
	comboJSON, err := json.Marshal(core.VersionCombo{"kind": string(def.Kind)})
	if err != nil {
		return fmt.Errorf("编码版本组合: %w", err)
	}
	now := m.now().UTC()
	_, err = m.Store.InsertEnvironmentRunOutcome(ctx, store.InsertEnvironmentRunOutcomeParams{
		ProjectID:         def.Project,
		EnvName:           def.Name,
		DefinitionHash:    def.Hash,
		StartedAt:         pgtype.Timestamptz{Time: now, Valid: true},
		FinishedAt:        pgtype.Timestamptz{Time: now, Valid: true},
		Status:            StatusTornDown,
		HealthCheckResult: nil, // 推倒无健康检查
		VersionCombo:      comboJSON,
	})
	if err != nil {
		return fmt.Errorf("记录环境推倒: %w", err)
	}
	return nil
}

// Rebuild 推倒重建:先 Down 再 Up。坏了就推倒重建,不修(PRD 第一部分)。
func (m *Manager) Rebuild(ctx context.Context, def Definition) (UpResult, error) {
	if err := m.Down(ctx, def); err != nil {
		return UpResult{}, fmt.Errorf("重建前推倒失败: %w", err)
	}
	return m.Up(ctx, def)
}

// fail 处理拉起失败:把原始输出交给 logpipe 入口(T04 已完成)产出脱敏片段与指纹,
// 记一行 status=failed(带上处理结果),再返回带 runID 与指纹的结构化 *UpError。
func (m *Manager) fail(ctx context.Context, def Definition, started time.Time, combo core.VersionCombo,
	report HealthReport, cause error, rawOutput string) error {
	processed := logpipe.Process(rawOutput) // ← 原始输出在这里进 logpipe
	if combo == nil {
		combo = core.VersionCombo{"kind": string(def.Kind)}
	}
	lp := &logpipeSummary{
		Excerpt:    processed.Excerpt,
		Normalized: processed.Normalized,
		ExactFP:    string(processed.Exact),
		FuzzyFP:    string(processed.Fuzzy),
	}
	report.OK = false
	runID, recErr := m.record(ctx, def, started, StatusFailed, report, combo, lp)
	if recErr != nil {
		return fmt.Errorf("%w;另外记录失败事实出错: %w", cause, recErr)
	}
	return &UpError{
		RunID:      runID,
		Project:    def.Project,
		EnvName:    def.Name,
		Combo:      combo,
		OccurredAt: m.now().UTC(),
		Processed:  processed,
		RawBytes:   int64(len(rawOutput)),
		Cause:      cause,
	}
}

// UpError 是拉起失败返回的结构化错误:带上刚记下的 environment_run id、失败当时的版本组合
// 与 logpipe 指纹产出,供编排层(exec / CLI)用 errors.As 取出后录进 failure_event
// (业务包不导入 ingest,方向是 exec → ingest → environment;见 docs/01-architecture.md)。
//
// Error() 文本与结构化之前完全一致(cause;指纹 X;脱敏片段: Y),既有断言不受影响。
type UpError struct {
	RunID      int64 // environment_run(status=failed)的行 id:failure_event 的来源外键
	Project    string
	EnvName    string
	Combo      core.VersionCombo // 失败当时的版本组合(与 environment_run.version_combo 同源)
	OccurredAt time.Time         // 失败发生时刻
	Processed  logpipe.Result    // logpipe 产出(指纹与 health_check_result 里的一致)
	RawBytes   int64             // 原始输出字节数
	Cause      error
}

func (e *UpError) Error() string {
	return fmt.Sprintf("%v;指纹 %s;脱敏片段: %s", e.Cause, e.Processed.Exact, oneLine(e.Processed.Excerpt))
}

func (e *UpError) Unwrap() error { return e.Cause }

// healthResultJSON 是 environment_run.health_check_result 的结构。
type healthResultJSON struct {
	OK      bool            `json:"ok"`
	Results []CheckResult   `json:"results"`
	Logpipe *logpipeSummary `json:"logpipe,omitempty"` // 仅失败时:logpipe 处理结果
}

// logpipeSummary 是失败时存进 health_check_result 的 logpipe 产出。
type logpipeSummary struct {
	Excerpt    string `json:"excerpt"`
	Normalized string `json:"normalized"`
	ExactFP    string `json:"exact_fp"`
	FuzzyFP    string `json:"fuzzy_fp"`
}

// record 记一行 environment_run(只 INSERT,INV-5)。lp 非空表示这是一次失败。
func (m *Manager) record(ctx context.Context, def Definition, started time.Time, status string,
	report HealthReport, combo core.VersionCombo, lp *logpipeSummary) (int64, error) {
	hcJSON, err := json.Marshal(healthResultJSON{OK: report.OK, Results: report.Results, Logpipe: lp})
	if err != nil {
		return 0, fmt.Errorf("编码健康检查结果: %w", err)
	}
	comboJSON, err := json.Marshal(combo)
	if err != nil {
		return 0, fmt.Errorf("编码版本组合: %w", err)
	}
	return m.Store.InsertEnvironmentRunOutcome(ctx, store.InsertEnvironmentRunOutcomeParams{
		ProjectID:         def.Project,
		EnvName:           def.Name,
		DefinitionHash:    def.Hash,
		StartedAt:         pgtype.Timestamptz{Time: started, Valid: true},
		FinishedAt:        pgtype.Timestamptz{Time: m.now().UTC(), Valid: true},
		Status:            status,
		HealthCheckResult: hcJSON,
		VersionCombo:      comboJSON,
	})
}

// appendLogs 把 compose 全部服务日志追加进 raw(失败时给 logpipe 用)。取不到不致命。
func (m *Manager) appendLogs(ctx context.Context, raw *strings.Builder, file, project string) {
	if res, err := m.Docker.ComposeLogs(ctx, file, project, ""); err == nil {
		fmt.Fprintf(raw, "compose logs:\n%s\n%s\n", res.Stdout, res.Stderr)
	}
}

// dockerCombo 组装 docker-compose 的版本组合:运行时版本(docker + compose)+ 各镜像 tag。
// 采集不到的项跳过(不致命),但 kind 一定在 —— version_combo 必非空。
func dockerCombo(ctx context.Context, m *Manager, file, project string, raw *strings.Builder) core.VersionCombo {
	combo := core.VersionCombo{"kind": string(KindDockerCompose)}
	if v, err := m.Docker.ServerVersion(ctx); err == nil {
		combo["docker"] = v
	} else {
		fmt.Fprintf(raw, "docker version: %v\n", err)
	}
	if v, err := m.Docker.ComposeVersion(ctx); err == nil {
		combo["compose"] = v
	} else {
		fmt.Fprintf(raw, "compose version: %v\n", err)
	}
	if imgs, err := m.Docker.ComposeImages(ctx, file, project); err == nil {
		for i, img := range imgs {
			combo[fmt.Sprintf("image:%d", i)] = img // 镜像 tag(如 alpine:latest)
		}
	} else {
		fmt.Fprintf(raw, "compose images: %v\n", err)
	}
	return combo
}

// sshCombo 组装 ssh-script 的版本组合:host + 远端 OS 版本(uname)。
func (m *Manager) sshCombo(ctx context.Context, def Definition, raw *strings.Builder) core.VersionCombo {
	combo := core.VersionCombo{"kind": string(KindSSHScript), "host": def.SSH.Host}
	if res, err := m.SSH.Run(ctx, runner.Command{Argv: []string{"uname -sr"}}); err == nil && res.ExitCode == 0 {
		combo["os"] = strings.TrimSpace(res.Stdout)
	} else if err != nil {
		fmt.Fprintf(raw, "uname: %v\n", err)
	}
	return combo
}

// sshLogFetcher 为 ssh-script 的 log-pattern 检查提供日志:跑 def.LogCommand 取其输出。
// 未定义 LogCommand 时返回 nil(该检查会以「无日志来源」失败)。
func (m *Manager) sshLogFetcher(def Definition) LogFetcher {
	if def.LogCommand == "" {
		return nil
	}
	return func(ctx context.Context, _ string) (string, error) {
		res, err := m.SSH.Run(ctx, runner.Command{Argv: []string{def.LogCommand}})
		if err != nil {
			return "", err
		}
		return res.Stdout + "\n" + res.Stderr, nil
	}
}

// SSHRunnerConfig 把定义里的 SSHDef 转成 runner.SSHConfig。
func SSHRunnerConfig(s SSHDef) runner.SSHConfig {
	host := s.Host
	if s.Port != 0 {
		host = net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	}
	return runner.SSHConfig{Host: host, User: s.User, Password: s.Password, KeyFile: s.KeyFile}
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
