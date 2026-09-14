package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"tally/internal/core"
	"tally/internal/logpipe"
	"tally/internal/project"
)

// runReindex 从 git + 事实面重算索引面(派生数据),绝不碰事实面(INV-5)。
//
//	tally reindex                         重算 trace_edge + 指纹统计(不需要 git)
//	tally reindex -repo <路径> -project <id>  额外从该 git 仓库重挂 trailer + 测试注解
//	tally reindex -check [-repo ... -project ...]  只检查不写:重算后与现有数据 diff,不一致非零退出
//
// -check 是 CI 定期跑的单向数据流自检:有人手改了索引表(绕过「事实→派生」的单向流),
// 重算就对不上,命令以非零退出码阻断。
func runReindex(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("reindex", flag.ContinueOnError)
	repo := fs.String("repo", "", "git 仓库路径;给了就从该仓库重挂 trailer + 测试注解")
	projID := fs.String("project", "", "项目 id(与 -repo 一起用:重挂归属的项目)")
	check := fs.Bool("check", false, "只检查不重算:重算后与现有数据 diff,不一致以非零退出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *repo != "" && *projID == "" {
		return fmt.Errorf("-repo 需要配合 -project 指定项目 id")
	}

	st, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer st.Close()

	opts := project.ReindexOptions{RepoPath: *repo, ProjectID: core.ProjectID(*projID)}

	if *check {
		diff, err := project.VerifyIndex(ctx, st, opts)
		if err != nil {
			return fmt.Errorf("检查索引面: %w", err)
		}
		if !diff.Consistent() {
			fmt.Print(diff.String())
			return fmt.Errorf("索引面与机器记录不一致:有人绕过了单向数据流")
		}
		fmt.Println("索引面一致:与 git + 事实面重算结果相符")
		return nil
	}

	rep, err := project.Reindex(ctx, st, opts)
	if err != nil {
		return fmt.Errorf("重算索引面: %w", err)
	}
	if *repo != "" {
		fmt.Printf("重挂 commit 挂钩 %d 条,测试挂钩 %d 条\n", rep.CommitHooks, rep.TestHooks)
	}
	fmt.Printf("重算追溯边 %d 条,指纹统计已从失败事件刷新\n", rep.TraceEdges)
	return nil
}

// runReindexFingerprints 用留存的原始日志重算历史指纹(RuleVersion 参与计算)。
//
// 事实面只追加(INV-5):本命令**绝不改写** failure_event。它读取带 raw_log_ref 的
// 历史事件,用当前规则重跑整条流水线,报告新旧精确指纹的差异 —— 让你知道规则升级
// 后哪些历史指纹漂移了、哪些没变。把结果落回可重建的索引面(fingerprint_stat)是
// reindex(T08)的职责;事实面永远保持原样,这正是为什么要留原始日志。
func runReindexFingerprints(ctx context.Context, _ []string) error {
	st, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer st.Close()

	rows, err := st.Queries().ListFailureEventsWithRaw(ctx)
	if err != nil {
		return fmt.Errorf("查询历史失败事件: %w", err)
	}
	if len(rows) == 0 {
		fmt.Println("没有带留存原始日志(raw_log_ref)的历史失败事件,无需重算。")
		fmt.Printf("当前规则版本: %s\n", logpipe.RuleVersion)
		return nil
	}

	fmt.Printf("当前规则版本: %s;重算 %d 条带原始日志的历史失败事件\n", logpipe.RuleVersion, len(rows))
	var changed, same, unreadable int
	for _, r := range rows {
		ref := r.RawLogRef.String
		raw, readErr := readRawLog(ref)
		if readErr != nil {
			unreadable++
			fmt.Printf("  #%-4d [%s] 读不到原始日志 %s: %v\n", r.ID, r.Source, ref, readErr)
			continue
		}
		// Process 对粗脱敏幂等,所以留存的是粗脱敏后的原始日志也能精确复现指纹。
		res := logpipe.Process(raw)
		if res.Exact != core.Fingerprint(r.FpExact) {
			changed++
			fmt.Printf("  #%-4d [%s] 指纹变化: %s(v%s) -> %s(v%s)\n",
				r.ID, r.Source, r.FpExact, r.PipelineVersion, res.Exact, logpipe.RuleVersion)
		} else {
			same++
			fmt.Printf("  #%-4d [%s] 指纹不变: %s\n", r.ID, r.Source, res.Exact)
		}
	}
	fmt.Printf("重算完成: 变化 %d, 不变 %d, 原始日志不可读 %d\n", changed, same, unreadable)
	return nil
}

// readRawLog 按 raw_log_ref 读原始日志。自部署单用户场景下 ref 是本地路径;
// 对象存储引用(s3:// 等)不在 T04 范围,明确报不支持,不假装读了个空串。
func readRawLog(ref string) (string, error) {
	if strings.Contains(ref, "://") {
		return "", fmt.Errorf("对象存储引用暂未支持(仅支持本地路径): %s", ref)
	}
	b, err := os.ReadFile(ref)
	if err != nil {
		return "", fmt.Errorf("读本地原始日志: %w", err)
	}
	return string(b), nil
}
