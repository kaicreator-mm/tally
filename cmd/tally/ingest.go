package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"tally/internal/core"
	"tally/internal/ingest"
	"tally/internal/jobs"
	"tally/internal/store"
)

// runIngestBuild 手动录入一次构建:tally ingest-build -project P -commit SHA [-toolchain X]
// [-log <日志文件>] [-status failed|passed] [-exit N] [-ci-url U] [-artifact D]
// [-raw-ref R] [-combo k=v,k=v]。
//
// 缺省读 stdin 当日志,方便管道接 CI 输出(CI 跑完构建把日志管道给本命令即完成录入)。
// 失败构建同时写 build_run 与 failure_event(接上 PRD 生产链的断点);成功构建只写 build_run。
func runIngestBuild(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("ingest-build", flag.ContinueOnError)
	project := fs.String("project", "", "项目 id(必填)")
	commit := fs.String("commit", "", "commit sha(必填:构建必须能回指 commit)")
	tool := fs.String("toolchain", "", "工具链名(npm/go/docker/compose/kamal;空 → 通用抽取)")
	logPath := fs.String("log", "", "构建日志文件路径(空 → 读 stdin)")
	status := fs.String("status", "failed", "构建结果:passed / failed")
	exit := fs.Int("exit", -1, "构建退出码(缺省:passed→0,failed→1)")
	ciURL := fs.String("ci-url", "", "CI 运行链接(可空)")
	artifact := fs.String("artifact", "", "成功时的制品 digest(可空)")
	rawRef := fs.String("raw-ref", "", "原始日志留存位置(缺省取 -log 路径;供规则改了重跑)")
	combo := fs.String("combo", "", "版本组合,逗号分隔的 k=v(如 go=1.23.0,docker=27.1)")
	workdir := fs.String("workdir", "", "复现上下文:项目代码所在目录(执行器用)")
	repro := fs.String("repro", "", "复现上下文:重跑那次构建的命令(诊断 G1/G2 用)")
	targets := fs.String("targets", "", "复现上下文:其余构建目标的完整命令,逗号分隔(G3 用)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *project == "" || *commit == "" {
		return fmt.Errorf("用法: tally ingest-build -project P -commit SHA [-toolchain X] [-log <文件>] [-status failed|passed]")
	}

	raw, err := readBuildLog(*logPath)
	if err != nil {
		return err
	}
	comboMap, err := parseCombo(*combo)
	if err != nil {
		return err
	}

	exitCode := *exit
	if exitCode < 0 {
		if *status == ingest.StatusPassed {
			exitCode = 0
		} else {
			exitCode = 1
		}
	}
	ref := *rawRef
	if ref == "" {
		ref = *logPath
	}
	now := time.Now().UTC()

	st, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer st.Close()

	runID, eventID, fp, err := ingest.IngestBuild(ctx, st, ingest.BuildInput{
		Input: ingest.Input{
			ProjectID:    *project,
			Raw:          raw,
			Toolchain:    *tool,
			RawLogRef:    ref,
			Combo:        comboMap,
			OccurredAt:   now,
			Workdir:      *workdir,
			Repro:        *repro,
			OtherTargets: splitTargets(*targets),
		},
		CommitSha:      *commit,
		CiRunURL:       *ciURL,
		Status:         *status,
		ExitCode:       exitCode,
		StartedAt:      now,
		FinishedAt:     now,
		ArtifactDigest: *artifact,
	})
	if err != nil {
		return err
	}
	fmt.Printf("build_run=%d\n", runID)
	if eventID != 0 {
		fmt.Printf("failure_event=%d(构建失败,已录失败事件)\n", eventID)
		maybeAutoDiagnose(ctx, st, *project, fp)
	} else {
		fmt.Println("构建成功,未录失败事件")
	}
	return nil
}

// maybeAutoDiagnose 在 TALLY_DIAGNOSE_AUTO=1 时,给刚录入的失败构建自动入队一次诊断
// (T11e,D-e1 默认关)。尽力而为:入队出错只提示,不让录入命令失败(失败事件已落库)。
// 与手动重诊断共用 worker 的同一条执行路径(D-e4),预算在 worker 侧守(D-e2)。
func maybeAutoDiagnose(ctx context.Context, st *store.Store, project string, fp core.Fingerprint) {
	if envOr("TALLY_DIAGNOSE_AUTO", "") != "1" || fp == "" {
		return
	}
	if _, err := jobs.New(st, nil).EnqueueDiagnose(ctx, project, string(fp)); err != nil {
		fmt.Printf("提示:自动诊断入队出错(failure_event 已录): %v\n", err)
		return
	}
	fmt.Println("自动诊断:已入队 diagnose 任务(TALLY_DIAGNOSE_AUTO=1)")
}

// readBuildLog 读构建日志:-log 为空时读 stdin。
func readBuildLog(path string) (string, error) {
	if path == "" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("读 stdin 日志: %w", err)
		}
		return string(b), nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读日志文件: %w", err)
	}
	return string(b), nil
}

// parseCombo 把 k=v,k=v 解成版本组合;空串 → nil(录入时记空对象)。
func parseCombo(s string) (core.VersionCombo, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	m := core.VersionCombo{}
	for _, kv := range strings.Split(s, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(kv), "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("解析 -combo:期望 k=v,得到 %q", kv)
		}
		m[k] = v
	}
	return m, nil
}

// splitTargets 把逗号分隔的「其余构建目标完整命令」解成列表;空串 → nil。
func splitTargets(s string) []string {
	var out []string
	for _, t := range strings.Split(s, ",") {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}
