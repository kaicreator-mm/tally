package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

	"tally/internal/command"
	"tally/internal/core"
	"tally/internal/deps"
	"tally/internal/knowledge"
	"tally/internal/runner"
	"tally/internal/store"
)

// runDeps 采集依赖升级事实,顺带跑修复退休检查:tally deps -project P -repo PATH
// [-from REV] [-to REV] [-content-repo PATH] [-no-retire]。
//
// 采集:遍历 (from, to] 区间的提交,逐个与父提交比锁文件(go.mod / package-lock.json),
// 变更写进 dependency_change(事实面只 INSERT;唯一索引去重,重跑安全)。
// -from 缺省 = 从仓库开头全量;-to 缺省 = HEAD。
//
// 退休检查(T11c,D-c1):采集完顺带重验本次变更碰过的已验证修复 —— 重跑其复现命令,
// 不再复现的退休(写 RETIRED.md + 清索引面)。-no-retire 可关;-content-repo 给知识库仓库。
func runDeps(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("deps", flag.ContinueOnError)
	project := fs.String("project", "", "项目 id(必填)")
	repoPath := fs.String("repo", "", "本地 git 仓库路径(必填)")
	from := fs.String("from", "", "起始 rev(不含;上次采到这里;空 = 从仓库开头)")
	to := fs.String("to", "", "结束 rev(含;空 = HEAD)")
	contentRepo := fs.String("content-repo", envOr("TALLY_CONTENT_REPO", "./tally-content"), "知识库所在 git 仓库(退休检查用)")
	noRetire := fs.Bool("no-retire", false, "只采集,不跑修复退休检查")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *project == "" || *repoPath == "" {
		return fmt.Errorf("用法: tally deps -project P -repo PATH [-from REV] [-to REV] [-content-repo PATH] [-no-retire]")
	}

	st, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer st.Close()

	changed, n, err := deps.Collect(ctx, st, deps.Options{
		ProjectID: *project,
		RepoPath:  *repoPath,
		From:      *from,
		To:        *to,
	})
	if err != nil {
		return err
	}
	fmt.Printf("采集完成:%d 条新依赖变更\n", n)

	if *noRetire {
		return nil
	}
	return runRetire(ctx, st, *project, *contentRepo, changed)
}

// runRetire 跑修复退休检查:本次变更的包 × 已验证修复 → 重跑复现 → 不再复现的退休。
// 无新变更 → 无事可做(重跑安全)。知识库打不开 / 没有已验证修复时如实降级,不挡采集。
func runRetire(ctx context.Context, st *store.Store, project, contentRepo string, changed []string) error {
	if len(changed) == 0 {
		return nil
	}
	kb := knowledge.NewStore(contentRepo, st.Queries())
	fixes, err := candidatesFromKnowledge(ctx, kb)
	if err != nil {
		// 知识库不可用(如内容仓库未初始化):退休检查跳过,采集结果不受影响。
		fmt.Printf("退休检查跳过:知识库不可用(%v)\n", err)
		return nil
	}
	if len(fixes) == 0 {
		return nil // 还没有已验证修复,谈不上退休
	}

	rep, err := deps.RetirePass(ctx, deps.RetireOptions{
		ProjectID: project,
		Changed:   changed,
		Fixes:     fixes,
		Runner:    localReproRunner{local: runner.LocalRunner{}},
		Retirer:   kb,
		Store:     deps.NewRetireStore(st),
	})
	if err != nil {
		return err
	}
	fmt.Printf("退休检查:退休 %d,仍复现 %d,跳过 %d\n", len(rep.Retired), len(rep.StillRepro), len(rep.Skipped))
	return nil
}

// candidatesFromKnowledge 列知识库已验证修复,转成退休判定视图(业务包互不导入,
// cmd 做编排:knowledge.Entry → deps.FixCandidate)。
func candidatesFromKnowledge(ctx context.Context, kb *knowledge.Store) ([]deps.FixCandidate, error) {
	entries, err := kb.ListVerified(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]deps.FixCandidate, 0, len(entries))
	for _, e := range entries {
		out = append(out, deps.FixCandidate{
			Fingerprint:   core.Fingerprint(e.Fingerprint),
			VerifiedCombo: e.VerifiedCombo,
			Retired:       e.Retired,
		})
	}
	return out, nil
}

// localReproRunner 把 runner.Runner 适配成 deps.ReproRunner:跑复现命令,退出码 0 = 绿。
//
// 执行走结构化 CommandSpec(P0-15,PRD §21.6):历史失败事件的 repro 仍是 TEXT,
// 在此桥接成 argv 并装配 spec 后交 runner 执行;shell 恒 false(§32:shell 语义
// 默认拒绝),timeout 未设沿用现状(0 = 由执行策略决定)。
type localReproRunner struct{ local runner.Runner }

func (r localReproRunner) Run(ctx context.Context, workdir, repro string) (bool, error) {
	spec, err := reproSpec(workdir, repro)
	if err != nil {
		return false, err
	}
	if spec.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, spec.Timeout)
		defer cancel()
	}
	res, err := r.local.Run(ctx, toRunnerCommand(spec))
	if err != nil {
		return false, err
	}
	return res.ExitCode == 0, nil
}

// reproSpec 把历史 TEXT 形态的 repro 装配成结构化命令:
// argv 来自确定性桥接拆分,cwd = 失败事件记录的 workdir,env 无追加,
// timeout = 0(未设,沿用现状)。拆不出合法 argv → 报错,不静默执行空命令。
func reproSpec(workdir, repro string) (command.CommandSpec, error) {
	argv, err := legacyReproArgv(repro)
	if err != nil {
		return command.CommandSpec{}, err
	}
	spec := command.CommandSpec{Argv: argv, Cwd: workdir, Shell: false}
	if err := spec.Validate(); err != nil {
		return command.CommandSpec{}, fmt.Errorf("repro %q: %w", repro, err)
	}
	return spec, nil
}

// legacyReproArgv 把历史 TEXT repro 桥接为 argv:只认空白分隔与成对的 "/' 引号
// (引号内空白保持字面)。不做变量展开、glob、管道等任何 shell 语义 ——
// 引号只保证含空格的参数完整,不是 shell。引号不闭合是数据缺陷:报错,不猜。
func legacyReproArgv(repro string) ([]string, error) {
	var (
		argv  []string
		arg   strings.Builder
		inArg bool
		quote rune // 当前未闭合的引号字符;0 = 不在引号内
	)
	flush := func() {
		if inArg {
			argv = append(argv, arg.String())
			arg.Reset()
			inArg = false
		}
	}
	for _, r := range repro {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				arg.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote, inArg = r, true
		case unicode.IsSpace(r):
			flush()
		default:
			arg.WriteRune(r)
			inArg = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("repro 引号未闭合: %q", repro)
	}
	flush()
	return argv, nil
}

// toRunnerCommand 把 CommandSpec 映射成执行器输入:argv→Argv,cwd→Dir,
// env→追加环境变量(键序固定,执行可复现);timeout 不进 Command,
// 由调用方按 spec.Timeout 经 ctx 预算(见 Run)。
func toRunnerCommand(spec command.CommandSpec) runner.Command {
	keys := make([]string, 0, len(spec.Env))
	for k := range spec.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	env := make([]string, 0, len(keys))
	for _, k := range keys {
		env = append(env, k+"="+spec.Env[k])
	}
	return runner.Command{Argv: spec.Argv, Dir: spec.Cwd, Env: env}
}

// envOr 返回环境变量值;空 → def。
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
