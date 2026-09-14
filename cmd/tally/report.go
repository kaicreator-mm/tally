package main

// report.go(P0-08)把 P1 度量基线报告接成操作员 CLI 只读路径:
//
//	tally report baseline [-task ID]
//
// 报告组装与呈现逻辑在 internal/measurement(BuildBaselineReport /
// RenderBaselineReport,纯函数、确定性呈现);本文件只负责从真实存储历史
// 装载事实投影:
//
//   - 任务枚举:llm_usage / human_effort 两张事实表里出现过 task_id 的任务
//     (读侧 SQL,只 SELECT);
//   - 事实装载:经 measurement.Store 逐任务读 human_effort / llm_usage 行;
//   - 维度缺口显式化(AC2):phase span / 里程碑 / binding / acceptance /
//     first-pass 在当前存储合同(migrations/0021)下没有落库面,本命令不
//     伪造这些维度的数据 —— 相关读数在报告中呈现为 insufficient(n=0),
//     命令头部的来源说明逐项列出「无持久化来源」的维度,不静默当作 0;
//   - 任务外控制面调用(llm_usage.task_id 为空)不属于任何任务投影,不计入
//     报告,但条数显式告知,不静默丢弃;
//   - AC4(无写侧生命周期突变):本命令只执行 SELECT,不触碰任何任务状态、
//     不追加任何事实行 —— 报告是纯读路径。

import (
	"context"
	"flag"
	"fmt"
	"time"

	"tally/internal/measurement"
	"tally/internal/store"
)

// runReport 是 report 命令族入口(当前只有 baseline,后续读侧报表挂这里)。
func runReport(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("用法: tally report baseline [-task ID]")
	}
	switch args[0] {
	case "baseline":
		return runReportBaseline(ctx, args[1:])
	default:
		return fmt.Errorf("未知 report 子命令: %s(可用: baseline)", args[0])
	}
}

// runReportBaseline 从真实存储历史产出基线报告(P0-08 AC1)。
//
//	tally report baseline              全部有度量事实的任务
//	tally report baseline -task ID     单任务(该任务无事实行时报告如实全不足)
func runReportBaseline(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("report baseline", flag.ContinueOnError)
	taskID := fs.String("task", "", "限定单个任务 id(空 = 全部有度量事实的任务)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("report baseline 不接受位置参数: %q", fs.Arg(0))
	}

	st, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer st.Close()

	// 任务外控制面调用不计入任何任务投影,但必须显式告知存在多少条(AC2)。
	tasklessCalls, err := countTasklessUsage(ctx, st)
	if err != nil {
		return err
	}

	var taskIDs []string
	if *taskID != "" {
		taskIDs = []string{*taskID}
	} else {
		taskIDs, err = listTasksWithFacts(ctx, st)
		if err != nil {
			return err
		}
	}

	ms := measurement.New(st.Queries())
	tasks := make([]measurement.TaskBaseline, 0, len(taskIDs))
	for _, id := range taskIDs {
		efforts, err := ms.HumanEffortForTask(ctx, id)
		if err != nil {
			return err
		}
		usages, err := ms.LLMUsageForTask(ctx, id)
		if err != nil {
			return err
		}
		tasks = append(tasks, measurement.TaskBaseline{
			TaskID:  id,
			Efforts: efforts,
			Usages:  usages,
			// Spans/Milestones/Bindings/Acceptances/FirstPass 无持久化来源,
			// 保持缺失(见文件头注释;AC2:缺失显式,不补 0)。
		})
	}

	rep, err := measurement.BuildBaselineReport(tasks, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("组装基线报告: %w", err)
	}

	printBaselineSources(taskIDs, tasklessCalls)
	fmt.Print(measurement.RenderBaselineReport(rep))
	return nil
}

// printBaselineSources 打印报告的来源说明:覆盖哪些任务、哪些维度有持久化
// 事实、哪些维度当前没有存储面(缺失必须显式,不留给读者猜,AC2/AC3)。
func printBaselineSources(taskIDs []string, tasklessCalls int) {
	fmt.Printf("报告来源:任务 %d 个(度量事实表中出现过 task_id 的任务)\n", len(taskIDs))
	if len(taskIDs) == 0 {
		fmt.Println("  存储中尚无任何带任务归属的度量事实(human_effort / llm_usage);全部读数将呈现为样本不足。")
	}
	fmt.Println("维度来源:human_effort / llm_usage(持久化事实,migrations/0021)")
	fmt.Println("  无持久化来源的维度(呈现为 insufficient,不补 0):phase spans、milestones、contract bindings、acceptance independence、first-pass")
	if tasklessCalls > 0 {
		fmt.Printf("  另有 %d 条任务外控制面调用(llm_usage.task_id 为空)不属于任何任务投影,不计入本报告\n", tasklessCalls)
	}
	fmt.Println()
}

// listTasksWithFacts 枚举在度量事实表里出现过 task_id 的任务(去重、按 id
// 排序)。P0-08 的读侧枚举,保持只 SELECT,不进 sqlc 写面。
func listTasksWithFacts(ctx context.Context, st *store.Store) ([]string, error) {
	rows, err := st.Pool().Query(ctx, `
		SELECT task_id FROM (
			SELECT task_id FROM human_effort WHERE task_id <> ''
			UNION
			SELECT task_id FROM llm_usage WHERE task_id IS NOT NULL AND task_id <> ''
		) t
		ORDER BY task_id`)
	if err != nil {
		return nil, fmt.Errorf("枚举有度量事实的任务: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("读取任务 id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("枚举有度量事实的任务: %w", err)
	}
	return ids, nil
}

// countTasklessUsage 统计任务外控制面调用数(llm_usage.task_id IS NULL)。
func countTasklessUsage(ctx context.Context, st *store.Store) (int, error) {
	var n int
	if err := st.Pool().QueryRow(ctx,
		`SELECT count(*) FROM llm_usage WHERE task_id IS NULL`).Scan(&n); err != nil {
		return 0, fmt.Errorf("统计任务外控制面调用: %w", err)
	}
	return n, nil
}
