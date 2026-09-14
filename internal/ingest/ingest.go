package ingest

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"tally/internal/core"
	"tally/internal/failure"
	"tally/internal/logpipe"
	"tally/internal/logpipe/toolchain"
	"tally/internal/store"
)

// sources 是 failure_event.source 的合法取值(与迁移 0001 的 CHECK 约束一致)。
// 在送进 failure.RecordEvent 之前早一步校验,报错比数据库约束更清楚。
var sources = map[string]bool{"build": true, "deploy": true, "environment": true}

// Input 是一次失败录入(与来源无关)。
//
// 两种用法:
//   - 给 Raw(+ 可选 Toolchain):本包跑 logpipe 算指纹(构建录入走这条)。
//   - 给 Processed:业务包已经跑过 logpipe(部署 / 环境失败在各自包内就算好了指纹),
//     直接透传,不重复处理 —— 保证 failure_event 与各自表 health_check_result 里的指纹一致。
type Input struct {
	ProjectID string
	Source    string // build / deploy / environment
	// 三个来源外键恰好一个非零(failure.RecordEvent 强制,否则不碰库)。
	BuildRunID       int64
	DeployRunID      int64
	EnvironmentRunID int64

	Raw       string // 原始日志:Processed 为空时过 logpipe 算指纹
	Toolchain string // 工具链名,选抽取器;空或 unknown → 通用抽取

	// Processed 可选:业务包已算好的 logpipe 产出。非 nil 时忽略 Raw / Toolchain。
	Processed *logpipe.Result
	// RawLogBytes 可选:原始日志字节数。0 → 取 len(Raw)(Processed 路径调用方自行给)。
	RawLogBytes int64

	Combo      core.VersionCombo // 失败当时的版本组合;nil → 记空对象
	OccurredAt time.Time         // 失败发生时刻(由调用方给,录入不读时钟以外的随机源)
	RawLogRef  string            // 原始日志留存位置(供规则改了重跑;PRD 第三部分)

	// 复现上下文的权威来源(T11b):录入时由知道怎么重现的一方(CI/人)携带,
	// 随失败持久化进 failure_event,诊断时装载出来跑真的 G1/G2/G3。
	// 部署/环境失败本来就没有可复现命令,留空 → Handle 跳过 G1/G2(D3 语义保留)。
	Workdir      string   // 项目代码所在目录(执行器用)
	Repro        string   // 复现命令:重跑那次构建
	OtherTargets []string // 其余构建目标的完整构建命令(G3:不受影响)
}

// RecordFailure 录入一次失败:选抽取器跑 logpipe(或透传 Processed)→ 组 failure.Event
// → failure.RecordEvent(同事务写 failure_event + 累加 fingerprint_stat)。
// 返回事件 id 与精确指纹(指纹一并返回,供调用方按需入队诊断,T11e/D-e3)。
//
// 只对失败录入:成功构建只写 build_run、不进 failure_event(PRD:只在失败的时候收)。
//
// 项目身份校验(PRD §33.3 Derived Index,ghost-project guard):落库前先查派生项目
// 注册表(project_index,Git 注册表的派生索引面,不是作者源)。未知项目在事实 INSERT
// 之前拒收;注册表本身不可查(未接存储/查询出错)按 fail-closed 拒收新事实。
// 已落库的历史事实不受影响(本函数只 INSERT,INV-5)。
func RecordFailure(ctx context.Context, s *store.Store, in Input) (eventID int64, fp core.Fingerprint, err error) {
	if strings.TrimSpace(in.ProjectID) == "" {
		return 0, "", errors.New("录入失败事件:缺少 project_id")
	}
	if !sources[in.Source] {
		return 0, "", fmt.Errorf("录入失败事件:未知来源 %q(支持 build/deploy/environment)", in.Source)
	}
	// 来源外键计数与 failure.RecordEvent 同一确定性校验:提前到注册表查询之前,
	// 未接库的调用也在碰任何数据库语义前拿到同一个明确错误。
	srcs := 0
	for _, id := range []int64{in.BuildRunID, in.DeployRunID, in.EnvironmentRunID} {
		if id != 0 {
			srcs++
		}
	}
	if srcs != 1 {
		return 0, "", fmt.Errorf("录入失败事件:三个来源外键必须恰好一个非空, got %d", srcs)
	}
	if err := ensureKnownProject(ctx, s, in.ProjectID); err != nil {
		return 0, "", err
	}

	processed := in.Processed
	if processed == nil {
		r := process(in.Raw, in.Toolchain)
		processed = &r
	}
	rawBytes := in.RawLogBytes
	if rawBytes == 0 {
		rawBytes = int64(len(in.Raw))
	}
	combo := in.Combo
	if combo == nil {
		combo = core.VersionCombo{}
	}

	ev := failure.Event{
		ProjectID:        in.ProjectID,
		Source:           in.Source,
		BuildRunID:       in.BuildRunID,
		DeployRunID:      in.DeployRunID,
		EnvironmentRunID: in.EnvironmentRunID,
		RawLogRef:        in.RawLogRef,
		RawLogBytes:      rawBytes,
		Excerpt:          processed.Excerpt,
		Normalized:       processed.Normalized,
		PipelineVersion:  logpipe.RuleVersion,
		Exact:            processed.Exact,
		Fuzzy:            processed.Fuzzy,
		Combo:            combo,
		OccurredAt:       in.OccurredAt,
		// 复现上下文(P0-12 结构化合同):录入边界的旧文本输入在此换成结构化
		// CommandSpec / OtherTarget;strings.Fields 的旧执行语义只活在
		// failure.LegacyReproCommand 这一明确命名的兼容装载器里。
		Repro:        failure.LegacyReproCommand(in.Workdir, in.Repro),
		OtherTargets: legacyOtherTargets(in.Workdir, in.OtherTargets),
	}
	id, err := failure.RecordEvent(ctx, s, ev)
	if err != nil {
		return 0, "", fmt.Errorf("录入失败事件: %w", err)
	}
	return id, processed.Exact, nil
}

// ensureKnownProject 是 ghost-project guard 原语(PRD §33.3):项目在派生项目注册表
// (project_index,由 Git 注册表派生、可整体重建,INV-STATE-01)中才放行新事实。
// PostgreSQL 不是作者源 —— 项目身份的权威在 Git 注册表,本表只是 DB 侧的派生判定面;
// 判定面不可查(未接存储/查询出错)时按 fail-closed 拒收新事实,绝不放行。
func ensureKnownProject(ctx context.Context, s *store.Store, projectID string) error {
	if s == nil {
		return errors.New("录入失败事件:项目注册表不可查(未接存储),按 fail-closed 拒收新事实")
	}
	known, err := s.Queries().ProjectIndexExists(ctx, projectID)
	if err != nil {
		return fmt.Errorf("录入失败事件:查询项目注册表出错,按 fail-closed 拒收新事实: %w", err)
	}
	if !known {
		return fmt.Errorf("录入失败事件:未知项目 %q(不在派生项目注册表 project_index 中),事实拒收", projectID)
	}
	return nil
}

// legacyOtherTargets 把录入边界的旧文本目标(其余构建目标的完整构建命令)换成结构化
// OtherTarget:Target 用原文作标签,Command 按旧执行语义(workdir + strings.Fields)
// 重建,与 Repro 的兼容装载同源。空/nil 输入保持 nil(Event 侧空数组语义不变)。
func legacyOtherTargets(workdir string, targets []string) []failure.OtherTarget {
	if len(targets) == 0 {
		return nil
	}
	out := make([]failure.OtherTarget, 0, len(targets))
	for _, t := range targets {
		out = append(out, failure.OtherTarget{Target: t, Command: failure.LegacyReproCommand(workdir, t)})
	}
	return out
}

// process 按工具链名选抽取器跑完整流水线;空名或未知工具链退回通用抽取。
// 与 cmd/tally scan 的选法同源(toolchain.ByName),保证同日志同指纹(INV-8)。
func process(raw, tool string) logpipe.Result {
	if e, ok := toolchain.ByName(tool); ok {
		return logpipe.ProcessWith(raw, e.Extract)
	}
	return logpipe.Process(raw)
}
