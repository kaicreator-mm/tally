package exec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"tally/internal/core"
	"tally/internal/deploy"
	"tally/internal/environment"
	"tally/internal/ingest"
	"tally/internal/jobs"
	"tally/internal/store"
)

// FactStore 是执行端需要的事实面记录能力(只 INSERT,INV-5)。*store.Queries 同时满足两者。
type FactStore interface {
	environment.RunRecorder
	deploy.Recorder
}

// DiagnoseEnqueuer 是自动诊断的入队出口(T11e):按精确指纹入队 diagnose 任务。
// *jobs.Queue 天然满足;定义在消费方便于测试注入 fake。
type DiagnoseEnqueuer interface {
	EnqueueDiagnose(ctx context.Context, project, fingerprint string) (int64, error)
}

// Config 是执行端依赖。全部注入,便于测试:单测用 stub 记录器 + 假 docker/命令执行器离线验证,
// 集成测试接真 Postgres(写事实面)与真 docker。
type Config struct {
	Root   string // 内容仓库根目录;payload 里的定义路径相对它解析
	Store  FactStore
	DB     *store.Store // 失败录入用(ingest 需要 *store.Store);nil → 只记 environment_run/deploy_run,不录 failure_event(单测)
	Docker environment.DockerRunner
	NewSSH func(environment.SSHDef) environment.SSHRunner // nil → ssh-script 环境无法执行
	Exec   deploy.CommandRunner                           // 部署工具的本地执行
	Now    func() time.Time

	Diagnose DiagnoseDeps // 诊断执行体的依赖(KB/Executor/Runner);缺 → 诊断 job 报未接线

	// T11e 自动诊断(默认关,D-e1):失败录入产生 failure_event 后按指纹自动入队 diagnose,
	// 与手动重诊断共用同一 handler(D-e4)。开关由 TALLY_DIAGNOSE_AUTO 在装配处读入。
	DiagnoseAuto  bool
	DiagnoseQueue DiagnoseEnqueuer // nil → 即使开关开也不自动入队
}

// Handlers 返回执行端注册给队列的全部 job 处理器:
// 环境三件(拉起 / 推倒 / 重建)、部署与回滚、诊断。执行时按 payload 里的定义路径读最新定义,
// 调 environment / deploy 包真正执行 —— 成功失败都会由它们写对应的事实记录;
// 诊断按指纹装载 failure_event,交给 failure.Handle 跑一轮。
func Handlers(cfg Config) map[jobs.Kind]jobs.Handler {
	return map[jobs.Kind]jobs.Handler{
		jobs.KindEnvironmentUp:      cfg.handleEnv("up"),
		jobs.KindEnvironmentDown:    cfg.handleEnv("down"),
		jobs.KindEnvironmentRebuild: cfg.handleEnv("rebuild"),
		jobs.KindDeploy:             cfg.handleDeploy(),
		jobs.KindDeployRollback:     cfg.handleRollback(),
		jobs.KindDiagnose:           cfg.handleDiagnose(),
	}
}

// resolvePath 把仓库相对的定义路径解析成绝对路径,拒绝越出 Root 的穿越与绝对路径
// (payload 来自界面表单,是不可信输入)。
func (cfg Config) resolvePath(rel string) (string, error) {
	// 绝对路径形态(POSIX 前导 / 、Windows 盘符)一律拒绝:定义路径必须是仓库相对路径。
	if strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, `\`) || strings.ContainsRune(rel, ':') {
		return "", fmt.Errorf("定义路径 %q 必须是仓库相对路径", rel)
	}
	root := filepath.Clean(cfg.Root)
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if abs == root || !strings.HasPrefix(abs, root+string(filepath.Separator)) {
		return "", fmt.Errorf("定义路径 %q 越出内容仓库边界", rel)
	}
	return abs, nil
}

// guardKnownProject 是执行端的 ghost-project guard(PRD §33.3 Derived Index):
// 环境 up/down/rebuild 与部署/回滚都会写事实面(environment_run / deploy_run,
// INV-5 只追加),执行前先查派生项目注册表(project_index,Git 注册表的派生索引面,
// 不是作者源)—— 未知项目动作不执行、一行事实都不落;注册表查询出错按 fail-closed
// 拒绝执行(宁可不做,不写来历不明的事实)。DB 未接(Config.DB 注释的离线单测装配)
// 时无注册表可查,保持既有离线语义;生产装配(cmd/worker)恒接库。
func (cfg Config) guardKnownProject(ctx context.Context, project string) error {
	if cfg.DB == nil {
		return nil
	}
	known, err := cfg.DB.Queries().ProjectIndexExists(ctx, project)
	if err != nil {
		return fmt.Errorf("查询项目注册表出错,按 fail-closed 拒绝执行: %w", err)
	}
	if !known {
		return fmt.Errorf("未知项目 %q(不在派生项目注册表 project_index 中),动作不执行、事实不落库", project)
	}
	return nil
}

// handleEnv 是拉起 / 推倒 / 重建的公共处理器。
//
// 定义在执行时刻重新读取解析:内容一变哈希就变,新旧拉起记录自然分开(INV-4)。
// Manager.Up/Rebuild 成功失败都写一行 environment_run;Down 成功后写一行 status=torn_down。
func (cfg Config) handleEnv(action string) jobs.Handler {
	return func(ctx context.Context, raw json.RawMessage) error {
		var p jobs.EnvActionPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("环境动作 payload: %w", err)
		}
		if p.Def == "" {
			return errors.New("环境动作缺少 def(环境定义仓库相对路径)")
		}
		path, err := cfg.resolvePath(p.Def)
		if err != nil {
			return err
		}
		def, err := environment.ParseDefinition(path)
		if err != nil {
			return err
		}
		// 注册级硬校验:缺健康检查的定义直接拒绝执行,不是警告。
		if err := environment.Validate(def); err != nil {
			return fmt.Errorf("环境定义校验不过: %w", err)
		}
		// 动作要写 environment_run(事实面):先验证项目身份(§33.3)再执行。
		if err := cfg.guardKnownProject(ctx, def.Project); err != nil {
			return err
		}
		m := &environment.Manager{Docker: cfg.Docker, Store: cfg.Store, Now: cfg.Now}
		if def.Kind == environment.KindSSHScript {
			if cfg.NewSSH == nil {
				return errors.New("未配置 SSH 执行能力,无法执行 ssh-script 环境")
			}
			m.SSH = cfg.NewSSH(def.SSH)
		}

		switch action {
		case "up":
			res, uerr := m.Up(ctx, def)
			if uerr != nil {
				cfg.ingestEnvFailure(ctx, uerr)
				return fmt.Errorf("拉起环境 %s/%s: %w", def.Project, def.Name, uerr)
			}
			slog.Info("环境已拉起", "project", def.Project, "env", def.Name,
				"run", res.RunID, "definition", res.Definition)
		case "down":
			if derr := m.Down(ctx, def); derr != nil {
				return fmt.Errorf("推倒环境 %s/%s: %w", def.Project, def.Name, derr)
			}
			slog.Info("环境已推倒", "project", def.Project, "env", def.Name)
		case "rebuild":
			res, rerr := m.Rebuild(ctx, def)
			if rerr != nil {
				cfg.ingestEnvFailure(ctx, rerr)
				return fmt.Errorf("重建环境 %s/%s: %w", def.Project, def.Name, rerr)
			}
			slog.Info("环境已重建", "project", def.Project, "env", def.Name, "run", res.RunID)
		default:
			return fmt.Errorf("未知环境动作 %q", action)
		}
		return nil
	}
}

// ingestEnvFailure / ingestDeployFailure:把业务包返回的结构化失败错误录进 failure_event
// (D2:编排层统一录入,业务包不导入 ingest)。
//
// 尽力而为:cfg.DB 未接(单测)或错误不是结构化失败(如推倒失败、校验不过)时静默跳过;
// 录入出错只记日志,不覆盖原始失败 —— 动作失败的事实(environment_run / deploy_run)已由业务包记下。
// 录入成功后,若开关打开(T11e)顺带给该指纹自动入队一次诊断。
func (cfg Config) ingestEnvFailure(ctx context.Context, err error) {
	if cfg.DB == nil {
		return
	}
	var uerr *environment.UpError
	if !errors.As(err, &uerr) {
		return
	}
	if _, fp, ingErr := ingest.RecordFailure(ctx, cfg.DB, ingest.Input{
		ProjectID:        uerr.Project,
		Source:           "environment",
		EnvironmentRunID: uerr.RunID,
		Processed:        &uerr.Processed,
		RawLogBytes:      uerr.RawBytes,
		Combo:            uerr.Combo,
		OccurredAt:       uerr.OccurredAt,
	}); ingErr != nil {
		slog.Error("录入环境失败事件出错(environment_run 已记)", "run", uerr.RunID, "err", ingErr)
	} else {
		cfg.maybeAutoDiagnose(ctx, uerr.Project, fp)
	}
}

func (cfg Config) ingestDeployFailure(ctx context.Context, err error) {
	if cfg.DB == nil {
		return
	}
	var ferr *deploy.FailureError
	if !errors.As(err, &ferr) {
		return
	}
	if _, fp, ingErr := ingest.RecordFailure(ctx, cfg.DB, ingest.Input{
		ProjectID:   ferr.Project,
		Source:      "deploy",
		DeployRunID: ferr.RunID,
		Processed:   &ferr.Processed,
		RawLogBytes: ferr.RawBytes,
		Combo:       ferr.Combo,
		OccurredAt:  ferr.OccurredAt,
	}); ingErr != nil {
		slog.Error("录入部署失败事件出错(deploy_run 已记)", "run", ferr.RunID, "err", ingErr)
	} else {
		cfg.maybeAutoDiagnose(ctx, ferr.Project, fp)
	}
}

// maybeAutoDiagnose 在开关打开时(TALLY_DIAGNOSE_AUTO=1,装配处读入 DiagnoseAuto),
// 给刚录入的失败自动入队一次 diagnose(T11e)。
//
// 尽力而为:入队出错只记日志,不影响失败录入的结果(failure_event 已落库)。
// 预算不在入队侧预检(D-e2):worker 执行时 failure.Handle 对终态/超预算指纹零 LLM 短路。
func (cfg Config) maybeAutoDiagnose(ctx context.Context, project string, fp core.Fingerprint) {
	if !cfg.DiagnoseAuto || cfg.DiagnoseQueue == nil || fp == "" {
		return
	}
	if _, err := cfg.DiagnoseQueue.EnqueueDiagnose(ctx, project, string(fp)); err != nil {
		slog.Warn("自动诊断入队出错(失败录入不受影响)", "project", project, "fingerprint", fp, "err", err)
	}
}

// handleDeploy 是部署处理器:按 payload 指定的部署定义(执行时刻重新解析,制品 digest 以定义
// 为准)调工具部署,并记一行 deploy_run(deployed / failed)。部署逻辑在工具里,这里只转发 +
// 记账(铁律 6)。
func (cfg Config) handleDeploy() jobs.Handler {
	return func(ctx context.Context, raw json.RawMessage) error {
		var p jobs.DeployPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("部署 payload: %w", err)
		}
		if p.Def == "" {
			return errors.New("部署缺少 def(部署定义仓库相对路径)")
		}
		path, err := cfg.resolvePath(p.Def)
		if err != nil {
			return err
		}
		def, err := deploy.ParseDefinition(path) // 解析即校验(制品 digest / 工具 / config 必填)
		if err != nil {
			return err
		}
		// 部署要写 deploy_run(事实面):先验证项目身份(§33.3)再执行。
		if err := cfg.guardKnownProject(ctx, def.Project); err != nil {
			return err
		}

		d := &deploy.Deployer{Exec: cfg.Exec, Store: cfg.Store, Now: cfg.Now}
		res, derr := d.Deploy(ctx, def)
		if derr != nil {
			// 失败行已由 Deployer 记下(failed);这里把失败录进 failure_event,job 标失败供排查。
			cfg.ingestDeployFailure(ctx, derr)
			return fmt.Errorf("部署 %s/%s 制品 %s: %w", def.Project, def.TargetEnv, def.Digest, derr)
		}
		slog.Info("部署完成", "project", def.Project, "env", def.TargetEnv,
			"digest", string(def.Digest), "run", res.RunID)
		return nil
	}
}

// handleRollback 是部署回滚处理器:解析部署定义,覆盖 target_env / artifact_digest 为 payload
// 指定的目标,调工具回滚并记一行 deploy_run(rolled_back / rollback_failed)。
// 回滚逻辑在工具里,这里只转发 + 记账(铁律 6)。
func (cfg Config) handleRollback() jobs.Handler {
	return func(ctx context.Context, raw json.RawMessage) error {
		var p jobs.DeployRollbackPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("回滚 payload: %w", err)
		}
		if p.Def == "" {
			return errors.New("回滚缺少 def(部署定义仓库相对路径)")
		}
		path, err := cfg.resolvePath(p.Def)
		if err != nil {
			return err
		}
		def, err := deploy.ParseDefinition(path)
		if err != nil {
			return err
		}
		// 回滚要写 deploy_run(事实面):先验证项目身份(§33.3)再执行。
		if err := cfg.guardKnownProject(ctx, def.Project); err != nil {
			return err
		}
		if p.TargetEnv != "" {
			def.TargetEnv = p.TargetEnv
		}
		if p.Digest != "" {
			def.Digest = core.ArtifactDigest(p.Digest)
		}
		if err := def.Validate(); err != nil {
			return fmt.Errorf("部署定义校验不过: %w", err)
		}

		d := &deploy.Deployer{Exec: cfg.Exec, Store: cfg.Store, Now: cfg.Now}
		res, rerr := d.Rollback(ctx, def)
		if rerr != nil {
			// 失败行已由 Deployer 记下(rollback_failed);这里把失败录进 failure_event,job 标失败供排查。
			cfg.ingestDeployFailure(ctx, rerr)
			return fmt.Errorf("回滚 %s/%s 到 %s: %w", def.Project, def.TargetEnv, def.Digest, rerr)
		}
		slog.Info("回滚完成", "project", def.Project, "env", def.TargetEnv,
			"digest", string(def.Digest), "run", res.RunID)
		return nil
	}
}
