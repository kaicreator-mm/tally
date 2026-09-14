// Package candidateverify(P2-01)把 v1.3 worker 垂直路径的「最终候选验证组成」
// 抽成可复用的内部服务:immutable ChangeSet 物化进一次性验证 workspace +
// verification.Orchestrator 最终编排(visible 冻结协议 + required gates +
// held-out 注入),供 v1.3 worker 与 v1.4 outer loop 复用(PRD §12-16)。
//
// 边界(与 AGENTS.md §5/§12 一致):
//   - 输入是精确的不可变上下文:冻结 VerificationContract(验证权威)+ immutable
//     ChangeSet(verdict 绑定对象)。类型层已保证不可变(Frozen 深拷贝只读;
//     ChangeSet 字段全不导出),服务绝不改写、绝不二次裁决(决策锁 8/9)。
//   - 本包只做组成(materialize + compose + run),不发明验证语义:FAIL 与 ERROR
//     的四元组判定归 internal/verification 编排器;evidence append-only 归
//     RunSink 端口实现(INV-FACT-01)。
//   - 候选执行与验证只发生在一次性 Git worktree 里,owner 主 worktree 零写入
//     (INV-EXEC-01)。
//   - 契约含 held_out binding 时接 holdout 通道(Tally 控制文件系统根,决策锁
//     10/11);通道根未接线即拒绝 —— 绝不静默降级为只跑 visible(INV-VER-05)。
//     held-out 前提(runtime session.isolation 能力协商)由调用方在执行前强制。
package candidateverify

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/holdout"
	"tally/internal/policy"
	"tally/internal/runner"
	"tally/internal/runstore"
	"tally/internal/store"
	"tally/internal/verification"
	"tally/internal/workspace"
)

// Config 是服务的生产装配面:owner 内容仓库(只读)、一次性 worktree 根、
// holdout 通道根、验证环境稳定名与 evidence 落库 provenance 通道。
type Config struct {
	// Repo 是 owner 内容仓库(owner 主 worktree,只读:worktree 走 Git 对象库)。
	Repo string
	// WorktreeRoot 是一次性验证 workspace 的父目录;必须落在 owner checkout
	// 之外(workspace.New 把关)。
	WorktreeRoot string
	// HoldoutRoot 是 Tally 控制的 holdout 文件系统根(决策锁 11);契约不含
	// held_out binding 时不读这个字段,空值安全。
	HoldoutRoot string
	// EnvName 是验证环境的稳定名称(与 EnvironmentIdentity 哈希合成环境身份)。
	EnvName string
	// Recorder 是 evidence 落库 provenance;必须是 fact-capable 通道
	// (INV-AUTH-01/02,由编排器构造时校验)。例如 worker v1.3 用
	// evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally-worker/v13"}。
	Recorder evidence.Recorder
}

// Deps 是注入缝(nil = 生产路径):Queries 为生产事实查询集(verification_run /
// evidence 落库);Sink 覆盖缺省 runstore 适配器(测试内存事实面)。
type Deps struct {
	// Queries 是生产事实查询集;与 Sink 皆 nil = 无处落库,Verify 拒绝执行。
	Queries *store.Queries
	// Sink 覆盖缺省的 RunSink→runstore 适配器(测试注入缝)。
	Sink verification.RunSink
}

// Request 是一次最终验证的输入:精确的不可变上下文。
type Request struct {
	// Contract 是冻结的 VerificationContract(验证权威;作者侧契约不可执行)。
	Contract *verification.Frozen
	// ChangeSet 是被验证的 immutable ChangeSet(verdict 绑定对象,§8.7.1;
	// 进入验证后不可变 —— 修复语义是新 Run + 新 ChangeSet,决策锁 8)。
	ChangeSet *changeset.ChangeSet
	// Runtime 是执行后端身份(观测坐标:verification_run.runtime 列 +
	// version combo)。只作归因,不改变编排行为。
	Runtime string
}

// Service 对固定候选做最终验证的可复用组成服务:候选 materialize 进一次性验证
// workspace(不改 owner checkout),visible binding 与 gate 在其内执行;契约含
// held_out binding 时接 holdout 通道。VerificationRun / evidence 事实经
// verification.RunSink 端口落库。一个实例可服务多次验证(每次 Verify 恰好一行
// VerificationRun)。
type Service struct {
	cfg  Config
	deps Deps
}

// New 构造服务。装配错误(依赖皆缺)推迟到 Verify 时报告,与执行面同一错误通道。
func New(cfg Config, deps Deps) *Service {
	return &Service{cfg: cfg, deps: deps}
}

// Verify 对精确的 (Frozen Contract, immutable ChangeSet) 做一次最终验证编排。
// 返回 error 仅表示装配/基础设施故障;领域内的 fail / needs_review 是正常结论
// (FinalOutcome.Status,事实已落库)。成功/失败/取消一律清理一次性 workspace。
func (s *Service) Verify(ctx context.Context, req Request) (*verification.FinalOutcome, error) {
	switch {
	case req.Contract == nil:
		return nil, errors.New("candidateverify: 缺少冻结 VerificationContract")
	case req.ChangeSet == nil:
		return nil, errors.New("candidateverify: 缺少 immutable ChangeSet")
	case strings.TrimSpace(req.Runtime) == "":
		return nil, errors.New("candidateverify: 缺少 runtime 身份")
	}

	mgr, err := workspace.New(s.cfg.Repo, s.cfg.WorktreeRoot)
	if err != nil {
		return nil, fmt.Errorf("candidateverify: 建 workspace manager: %w", err)
	}
	vw, err := materializeCandidate(ctx, mgr, req.ChangeSet)
	if err != nil {
		return nil, err
	}
	// 成功/失败/取消一律清理;清理不继承调用方取消(workspace.Manager 纪律)。
	defer func() {
		if err := mgr.Cleanup(context.WithoutCancel(ctx), vw); err != nil {
			slog.Warn("candidateverify: 清理验证 workspace 失败", "path", vw.Path, "err", err)
		}
	}()

	var reader holdout.VerificationReader
	var boundary verification.HoldoutRunFunc
	if ContractHasHeldOut(req.Contract) {
		hs, err := holdout.NewFilesystemStore(s.cfg.HoldoutRoot, s.cfg.Repo, s.cfg.WorktreeRoot)
		if err != nil {
			return nil, fmt.Errorf("candidateverify: 打开 holdout store: %w", err)
		}
		reader = hs
		boundary = holdoutBoundary(runner.LocalRunner{})
	}

	vsink := s.deps.Sink
	if vsink == nil {
		if s.deps.Queries == nil {
			return nil, errors.New("candidateverify: 缺少事实存储且未注入 verification sink")
		}
		vsink = NewRunStoreSink(runstore.New(s.deps.Queries), req.Contract.Revision())
	}

	orch, err := verification.NewOrchestrator(verification.OrchestratorConfig{
		Runner:        runner.LocalRunner{},
		Policy:        policy.DefaultExecutionPolicy(),
		HoldoutReader: reader,
		HoldoutRun:    boundary,
		Sink:          vsink,
		// Recorder 是验证事实的 provenance 通道:本服务的全部 evidence 都由
		// 冻结协议执行产生,通道为 fact-capable 的 verification_run(INV-AUTH-01/02)。
		Recorder: s.cfg.Recorder,
	})
	if err != nil {
		return nil, fmt.Errorf("candidateverify: 构造最终验证编排器: %w", err)
	}

	// 环境身份:本地 worker 执行面(OS/arch/工具链语义字段,§8.7.1)。
	env := verification.EnvironmentIdentity{
		OS:        goruntime.GOOS,
		Arch:      goruntime.GOARCH,
		Toolchain: goruntime.Version(),
	}
	combo := core.VersionCombo{"go": goruntime.Version(), "runtime": req.Runtime}

	outcome, err := orch.Run(ctx, verification.FinalRunRequest{
		Project:          req.ChangeSet.Project(),
		Contract:         req.Contract,
		ChangeSet:        req.ChangeSet,
		EnvironmentName:  s.cfg.EnvName,
		Environment:      env,
		VersionCombo:     combo,
		Runtime:          req.Runtime,
		Workspace:        vw.Path,
		CandidateFixedAt: req.ChangeSet.CreatedAt(),
	})
	if err != nil {
		return nil, fmt.Errorf("candidateverify: 最终验证执行失败: %w", err)
	}
	return outcome, nil
}

// ContractHasHeldOut 报告冻结契约是否含 held_out binding。调用方用它强制
// held-out 前提(holdout 通道接线 + runtime session.isolation 能力声明);
// 服务自己用它决定是否接 holdout 通道。
func ContractHasHeldOut(f *verification.Frozen) bool {
	c := f.Contract()
	for _, acc := range c.Acceptances {
		for _, b := range acc.EvidenceBindings {
			if b.VisibilityPolicy == verification.VisibilityHeldOut {
				return true
			}
		}
	}
	return false
}

// materializeCandidate 把不可变 ChangeSet 物化成一次性验证 workspace:改动文件
// 非空 → 在 base revision 上照写改后完整内容(core.FileChange 同一约定:照写,
// 不引 patch 引擎);快照只有 candidate revision(实现 Run 已提交)→ 直接在该
// revision 上开 worktree。越出 worktree 的路径一律拒绝(INV-EXEC-01)。
func materializeCandidate(ctx context.Context, mgr *workspace.Manager, cs *changeset.ChangeSet) (*workspace.Worktree, error) {
	rev := cs.BaseRevision()
	if len(cs.Files()) == 0 && strings.TrimSpace(cs.CandidateRevision()) != "" {
		rev = cs.CandidateRevision()
	}
	wt, err := mgr.Create(ctx, rev)
	if err != nil {
		return nil, fmt.Errorf("candidateverify: 建验证 workspace: %w", err)
	}
	for _, f := range cs.Files() {
		if err := guardWorktreeRelPath(f.Path); err != nil {
			return nil, err
		}
		p := filepath.Join(wt.Path, filepath.FromSlash(filepath.Clean(f.Path)))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return nil, fmt.Errorf("candidateverify: 物化候选(建目录)%s: %w", f.Path, err)
		}
		if err := os.WriteFile(p, []byte(f.Content), 0o644); err != nil {
			return nil, fmt.Errorf("candidateverify: 物化候选 %s: %w", f.Path, err)
		}
	}
	return wt, nil
}

// guardWorktreeRelPath 拒绝越出一次性 worktree 的仓库相对候选路径(空、绝对、
// .. 穿越)。
func guardWorktreeRelPath(p string) error {
	rel := filepath.Clean(filepath.FromSlash(p))
	if p == "" || rel == "." || rel == ".." || filepath.IsAbs(rel) ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("候选路径 %q 越出一次性 worktree,拒绝(INV-EXEC-01)", p)
	}
	return nil
}
