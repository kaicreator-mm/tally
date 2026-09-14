package exec

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"tally/internal/airuntime"
	"tally/internal/command"
	"tally/internal/core"
	"tally/internal/failure"
	"tally/internal/jobs"
)

// DiagnoseDeps 是诊断执行体的依赖,全部注入(测试接 fake,worker 接真实现)。
// 任一缺失 → 诊断 job 报「依赖未接线」而失败(注册照旧,缺的是运行条件)。
type DiagnoseDeps struct {
	Executor airuntime.Executor     // 通用推理只提案(INV-1);判定在 failure 的四道闸
	Runner   failure.Runner         // 跑复现 / 验证的执行器
	KB       failure.KnowledgeBase  // knowledge.Store:精确命中零 LLM
	SpecSink failure.SpecDefectSink // 可不注入:未注入时 spec_defect 出口不可达
}

// handleDiagnose 是诊断 job 的执行体:按精确指纹装载最近一次 failure_event,
// 重组 failure.Event,交给 failure.Handle 跑一轮「查库 → 预算 → 复现 → 提案 → 四道闸 → 归类」。
//
// 复现上下文(T11b)随失败录入持久化在 failure_event,装载时一并带出;
// 录入方没给 Repro 时 Handle 仍跳过 G1/G2(D3 语义保留)。
func (cfg Config) handleDiagnose() jobs.Handler {
	return func(ctx context.Context, raw json.RawMessage) error {
		var p jobs.DiagnosePayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("诊断 payload: %w", err)
		}
		if p.Fingerprint == "" {
			return errors.New("诊断缺少 fingerprint(精确指纹)")
		}
		if cfg.DB == nil || cfg.Diagnose.KB == nil || cfg.Diagnose.Executor == nil || cfg.Diagnose.Runner == nil {
			return errors.New("诊断依赖未接线(需要 DB / KB / Executor / Runner)")
		}

		ev, err := cfg.loadDiagnoseEvent(ctx, core.Fingerprint(p.Fingerprint))
		if err != nil {
			return err
		}

		concl, err := failure.Handle(ctx, cfg.failureDeps(), ev)
		if err != nil {
			return fmt.Errorf("诊断 %s: %w", p.Fingerprint, err)
		}
		slog.Info("诊断完成", "fingerprint", p.Fingerprint, "project", ev.ProjectID,
			"conclusion", describeConclusion(concl))
		return nil
	}
}

// loadDiagnoseEvent 按精确指纹取最近一次 failure_event,重组诊断输入。
// 无对应事件 → 报错(先录入再诊断,不让诊断凭空跑)。
func (cfg Config) loadDiagnoseEvent(ctx context.Context, fp core.Fingerprint) (failure.Event, error) {
	row, err := cfg.DB.Queries().GetLatestFailureEventByFingerprint(ctx, string(fp))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return failure.Event{}, fmt.Errorf("指纹 %s 无 failure_event(先录入失败再诊断)", fp)
		}
		return failure.Event{}, fmt.Errorf("装载 failure_event: %w", err)
	}
	combo := core.VersionCombo{}
	if len(row.VersionCombo) > 0 {
		if err := json.Unmarshal(row.VersionCombo, &combo); err != nil {
			return failure.Event{}, fmt.Errorf("解码 version_combo: %w", err)
		}
	}
	// 复现上下文(T11b):录入时持久化,装载出来跑真的 G1/G2/G3。
	// 0016 双轨合同(P0-04/P0-12):repro_command 是 JSON 对象 → 结构化 CommandSpec;
	// 是 null → 存量行按 LegacyReproCommand(workdir, repro) 重建旧执行语义;
	// other_target_commands 是 [{target,command}] → 结构化其余目标;空数组保持 nil。
	repro := command.CommandSpec{}
	if trimmed := bytes.TrimSpace(row.ReproCommand); len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null")) {
		if err := json.Unmarshal(trimmed, &repro); err != nil {
			return failure.Event{}, fmt.Errorf("解码 repro_command: %w", err)
		}
	} else {
		repro = failure.LegacyReproCommand(row.Workdir, row.Repro)
	}
	var targets []failure.OtherTarget
	if trimmed := bytes.TrimSpace(row.OtherTargetCommands); len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("[]")) {
		if err := json.Unmarshal(trimmed, &targets); err != nil {
			return failure.Event{}, fmt.Errorf("解码 other_target_commands: %w", err)
		}
	}
	return failure.Event{
		ProjectID:        row.ProjectID,
		Source:           row.Source,
		BuildRunID:       row.BuildRunID.Int64,
		DeployRunID:      row.DeployRunID.Int64,
		EnvironmentRunID: row.EnvironmentRunID.Int64,
		RawLogRef:        row.RawLogRef.String,
		RawLogBytes:      row.RawLogBytes.Int64,
		Excerpt:          row.Excerpt.String,
		Normalized:       row.Normalized.String,
		PipelineVersion:  row.PipelineVersion,
		Exact:            core.Fingerprint(row.FpExact),
		Fuzzy:            core.Fingerprint(row.FpFuzzy),
		Combo:            combo,
		OccurredAt:       row.OccurredAt.Time,
		Repro:            repro,
		OtherTargets:     targets,
	}, nil
}

// failureDeps 组装 failure.Handle 的依赖:Stats 包 fingerprint_stat 索引面,Now 用执行端时钟。
func (cfg Config) failureDeps() failure.Deps {
	return failure.Deps{
		Executor: cfg.Diagnose.Executor,
		Runner:   cfg.Diagnose.Runner,
		KB:       cfg.Diagnose.KB,
		Stats:    failure.NewStatStore(cfg.DB.Queries()),
		SpecSink: cfg.Diagnose.SpecSink,
		Now:      cfg.Now,
	}
}

// describeConclusion 把一轮结论压成日志用的一行。
func describeConclusion(c failure.Conclusion) string {
	switch {
	case c.SkippedRepro:
		return "跳过:缺复现上下文"
	case c.Archived:
		return "归档:三轮预算用尽"
	case c.Paused:
		return "暂停:修复改到应用源码"
	case c.Kind != "":
		return string(c.Kind)
	default:
		return "本轮无出口"
	}
}
