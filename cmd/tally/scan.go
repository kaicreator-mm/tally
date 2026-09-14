package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tally/internal/logpipe"
	"tally/internal/logpipe/toolchain"
)

// runScan 批量扫一个目录下的历史日志,逐个跑流水线,输出碰撞率实验结果
// (事件总数、去重后指纹数、碰撞率、各工具链分布)。这是 T04 的碰撞率实验:
// 先只做统计,不做修复。
//
// 工具链默认按「相对扫描根的第一段路径」推断(testdata 按工具链分目录),
// 可用 -toolchain 强制所有文件用同一抽取器;推断不出就退回通用抽取(记为 unknown)。
func runScan(_ context.Context, args []string) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	tool := fs.String("toolchain", "", "强制所有文件用该工具链抽取器(默认按路径推断)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root := fs.Arg(0)
	if root == "" {
		return fmt.Errorf("用法: tally scan [-toolchain X] <日志目录>")
	}

	stats, err := scanDir(root, *tool)
	if err != nil {
		return err
	}
	printStats(stats)
	return nil
}

// scanDir 遍历 root 下所有 .log,跑流水线累加统计。拆出来便于复用与测试接线。
func scanDir(root, forcedToolchain string) (*logpipe.Stats, error) {
	stats := logpipe.NewStats()
	count := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".log" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		tc := forcedToolchain
		if tc == "" {
			tc = inferToolchain(rel)
		}
		var res logpipe.Result
		if e, ok := toolchain.ByName(tc); ok {
			res = logpipe.ProcessWith(string(b), e.Extract)
		} else {
			res = logpipe.Process(string(b)) // 未知工具链退回通用抽取
			tc = "unknown"
		}
		stats.Add(tc, res.Exact)
		count++
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("扫描 %s: %w", root, err)
	}
	if count == 0 {
		return nil, fmt.Errorf("%s 下没有 .log 文件", root)
	}
	return stats, nil
}

// inferToolchain 用相对路径第一段当工具链名。单文件(无目录层级)记为 unknown。
func inferToolchain(rel string) string {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) > 1 {
		return parts[0]
	}
	return "unknown"
}

// printStats 打印碰撞率实验结果。
func printStats(s *logpipe.Stats) {
	fmt.Println("扫描结果")
	fmt.Printf("  事件总数:     %d\n", s.Total)
	fmt.Printf("  去重后指纹数: %d\n", s.Distinct())
	fmt.Printf("  碰撞率:       %.1f%%\n", s.CollisionRate()*100)
	fmt.Println("各工具链分布:")
	for _, ts := range s.ByToolchain() {
		fmt.Printf("  %-8s 事件 %3d  去重 %3d  碰撞率 %5.1f%%\n",
			ts.Name, ts.Total, ts.Distinct, ts.CollisionRate*100)
	}
}
