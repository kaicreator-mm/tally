package deps

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	"tally/internal/command"
	"tally/internal/core"
	"tally/internal/store"
)

// reproRuns 是退休重验的运行次数:与 failure G1 同标准,连跑 5 次判定复现是否稳定。
// 不在 deps 导入 failure(业务包互不导入),故在此镜像该常量与语义(D-c3)。
const reproRuns = 5

// FixCandidate 是退休判定所需的一条已验证修复的最小视图(在使用方定义,
// 业务包互不导入)。由调用方从 knowledge.Entry 转换而来。
type FixCandidate struct {
	Fingerprint   core.Fingerprint
	VerifiedCombo core.VersionCombo
	Retired       bool // 已退休(RETIRED.md):不重验(D-c6)
}

// ReproRunner 跑一次复现命令,返回是否绿(退出码 0)。绿 = 失败没复现;红 = 复现了。
// 真实执行器在 cmd 里接 runner.LocalRunner;单测注入确定性 fake。
type ReproRunner interface {
	Run(ctx context.Context, workdir, command string) (green bool, err error)
}

// Retirer 给一条修复打退休标记(写 RETIRED.md + git commit),返回提交哈希。
// 实现是 knowledge.Store(签名同形,结构满足);与 MarkStale 同构,记录保留不删。
type Retirer interface {
	Retire(ctx context.Context, fp core.Fingerprint, reason string) (string, error)
}

// RetireStore 是退休检查对数据库的期望(在使用方定义):从事实面装载复现上下文,
// 在索引面清掉失效修复。实现是 storeAdapter(包 *store.Store),调用方注入。
type RetireStore interface {
	// LoadReproContext 取(项目, 指纹)最近一次失败的 workdir / repro(D-c4)。
	// 无对应事件返回空串(不报错)——调用方据此跳过,不猜。
	LoadReproContext(ctx context.Context, project string, fp core.Fingerprint) (workdir, repro string, err error)
	// ClearFingerprintStatFix 清索引面 fix 相关列(D-c5):状态回 unresolved,重新可诊断。
	ClearFingerprintStatFix(ctx context.Context, fp core.Fingerprint) error
}

// RetireOptions 是一次退休检查的全部输入。
type RetireOptions struct {
	ProjectID string         // 发生依赖升级的项目
	Changed   []string       // 本次变更的包(来自 Collect)
	Fixes     []FixCandidate // 已验证修复(调用方从 knowledge.ListVerified 转换)
	Runner    ReproRunner
	Retirer   Retirer
	Store     RetireStore
}

// Report 是一次退休检查的结果(给人看的摘要)。
type Report struct {
	Retired    []core.Fingerprint // 不再复现 → 已退休
	StillRepro []core.Fingerprint // 仍稳定复现 → 不动
	Skipped    []core.Fingerprint // 缺复现上下文 / 已退休 → 跳过
}

// RetirePass 跑一轮退休检查(D-c1:采集完依赖变更顺带触发):
// 变更包 × 已验证修复 → 相关的重跑复现命令 → 不再复现的退休。
//
// 全流程零 LLM(铁律 1):相关性与复现稳定性都是确定性判定。
// 无变更包 → 直接返回(没什么可查)。
func RetirePass(ctx context.Context, opts RetireOptions) (Report, error) {
	var rep Report
	if len(opts.Changed) == 0 {
		return rep, nil
	}
	changed := make(map[string]bool, len(opts.Changed))
	for _, p := range opts.Changed {
		changed[p] = true
	}

	for _, fix := range opts.Fixes {
		if fix.Retired {
			// 已退休不重验(D-c6):记录保留,退出使用。
			rep.Skipped = append(rep.Skipped, fix.Fingerprint)
			continue
		}
		touched := touchedPackages(fix.VerifiedCombo, changed)
		if len(touched) == 0 {
			continue // 验证组合没碰本次变更的包 → 无关
		}

		workdir, repro, err := opts.Store.LoadReproContext(ctx, opts.ProjectID, fix.Fingerprint)
		if err != nil {
			return rep, fmt.Errorf("装载复现上下文 %s: %w", fix.Fingerprint, err)
		}
		if workdir == "" || repro == "" {
			// 旧失败没带上下文:跳过并说明,不猜(D-c4)。
			slog.Info("退休检查跳过:缺复现上下文", "fingerprint", fix.Fingerprint,
				"project", opts.ProjectID, "changed", touched)
			rep.Skipped = append(rep.Skipped, fix.Fingerprint)
			continue
		}

		stable, err := reproStable(ctx, opts.Runner, workdir, repro)
		if err != nil {
			return rep, fmt.Errorf("重验复现 %s: %w", fix.Fingerprint, err)
		}
		if stable {
			rep.StillRepro = append(rep.StillRepro, fix.Fingerprint) // 仍复现 → 不动
			continue
		}

		// 不再复现 → 退休:先清索引面(让同指纹再失败能重新诊断),再写退休标记。
		if err := opts.Store.ClearFingerprintStatFix(ctx, fix.Fingerprint); err != nil {
			return rep, fmt.Errorf("清索引面修复 %s: %w", fix.Fingerprint, err)
		}
		reason := fmt.Sprintf("依赖升级变更了 %v,验证过的复现命令不再复现", touched)
		if _, err := opts.Retirer.Retire(ctx, fix.Fingerprint, reason); err != nil {
			return rep, fmt.Errorf("退休 %s: %w", fix.Fingerprint, err)
		}
		slog.Info("修复退休", "fingerprint", fix.Fingerprint, "project", opts.ProjectID, "changed", touched)
		rep.Retired = append(rep.Retired, fix.Fingerprint)
	}
	return rep, nil
}

// touchedPackages 返回验证组合里被本次变更碰过的包(确定序)。空 = 无关(D-c2:
// 只要升级碰过修复依赖过的组件,就值得重验;版本比对交给运行时)。
func touchedPackages(combo core.VersionCombo, changed map[string]bool) []string {
	var out []string
	for pkg := range combo {
		if changed[pkg] {
			out = append(out, pkg)
		}
	}
	sort.Strings(out)
	return out
}

// reproStable 连跑复现命令 reproRuns 次,判定是否仍稳定复现(G1 语义,同 failure.ConfirmRepro):
// 全红 = 仍复现(true);有绿 = 不再复现(false)。
func reproStable(ctx context.Context, r ReproRunner, workdir, command string) (bool, error) {
	reds := 0
	for i := 0; i < reproRuns; i++ {
		green, err := r.Run(ctx, workdir, command)
		if err != nil {
			return false, fmt.Errorf("第 %d 次: %w", i+1, err)
		}
		if !green {
			reds++
		}
	}
	return reds == reproRuns, nil
}

// storeAdapter 用 *store.Store 实现 RetireStore。
type storeAdapter struct{ s *store.Store }

// NewRetireStore 把 *store.Store 适配成退休检查用的 RetireStore。
func NewRetireStore(s *store.Store) RetireStore { return storeAdapter{s: s} }

func (a storeAdapter) LoadReproContext(ctx context.Context, project string, fp core.Fingerprint) (string, string, error) {
	row, err := a.s.Queries().GetLatestFailureEventByProjectAndFingerprint(ctx,
		store.GetLatestFailureEventByProjectAndFingerprintParams{
			ProjectID: project,
			FpExact:   string(fp),
		})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", nil // 无对应失败事件 → 无上下文
		}
		return "", "", err
	}
	// 0016 双轨合同:结构化行优先(repro_command 是 CommandSpec JSON);存量行
	// (repro_command 为 JSON null)按旧文本两列原样返回。接口仍是 (workdir, repro)
	// 字符串,cmd/tally 侧 P0-15 的 quote-aware 桥接负责还原 argv。
	if trimmed := bytes.TrimSpace(row.ReproCommand); len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null")) {
		var spec command.CommandSpec
		if err := json.Unmarshal(trimmed, &spec); err != nil {
			return "", "", fmt.Errorf("解码 repro_command: %w", err)
		}
		if len(spec.Argv) == 0 {
			return "", "", nil
		}
		return spec.Cwd, strings.Join(spec.Argv, " "), nil
	}
	return row.Workdir, row.Repro, nil
}

func (a storeAdapter) ClearFingerprintStatFix(ctx context.Context, fp core.Fingerprint) error {
	return a.s.Queries().ClearFingerprintStatFix(ctx, string(fp))
}
