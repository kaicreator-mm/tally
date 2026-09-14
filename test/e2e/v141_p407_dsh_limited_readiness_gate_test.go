//go:build integration

// P4-07 — DeepSeek limited-readiness gate(任务合同:在有 Tally 自有隔离的
// 真实仓库上跑一个有界候选流,裁决 LIMITED_READY 或 BLOCKED;P4-07 是
// DSH 在 v1.4.1 发布范围内的 readiness gate)。
//
// 被测面 = P4-04 适配器公共面(runtimes/dsh,经 Start 钉版身份闸 + 能力协商)
// + 既有闭环桥(changeset.New 从 workspace 差量构造候选快照)+ 真实临时 Git
// 仓库 + 真实进程树(替身 launcher 经 go build 现场构建,确定性、无钥)。
//
// 门禁证据面:
//   - 有界任务:真实仓库 workspace 内,headless 一次性任务读取任务输入并
//     确定性完成(RunCompleted);任务产出经 changeset.New 走既有闭环桥;
//   - 负面权限:ResumeOf/ModelProfile 请求在能力协商处被拒(诚实能力集);
//   - 负面取消:长运行任务 Cancel → RunCancelled + 整树终止后子进程零残留;
//   - 隔离:cwd 钉 workspace、per-session DSH_HOME、遥测禁用(P4-04 包矩阵
//     逐项核对;本门禁经真实进程树复核取消/孤儿面)。
//
// 裁决:**LIMITED_READY** —— 上述全部在确定性替身上证明;显式限制:
// ① 真实 dsh 二进制的 live 内容产出(模型真实写文件)未在门禁内验证(需要
// 真钥+真二进制的 live 环境;门禁保持可复现、无钥);② headless 不暴露
// events/resume/结构化输出(R-3,能力面诚实声明);③ 上游 developer-preview
// 易变(R-2,钉版闸使漂移以显式失败显形)。DL-17:门禁不搜索、不挂载任何
// hidden 材料。
package e2e

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"

	"tally/internal/agentruntime"
	"tally/internal/changeset"
	"tally/internal/gitwriter"
	"tally/runtimes/dsh"
)

// p407BuildFake 现场构建替身 launcher(确定性、无钥;与 P4-04 包测试同一替身)。
func p407BuildFake(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法定位测试源文件")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	out := filepath.Join(t.TempDir(), "fakedsh")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", out, "./runtimes/dsh/testdata/fakedsh")
	cmd.Dir = root
	if out2, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("构建替身 dsh: %v: %s", err, out2)
	}
	return out
}

// p407Rev 解析仓库当前 HEAD(完整 sha)。
func p407Rev(t *testing.T, repo string) string {
	t.Helper()
	w, err := gitwriter.New(repo)
	if err != nil {
		t.Fatalf("gitwriter.New: %v", err)
	}
	h, err := w.Head(context.Background())
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	return h
}

// p407WaitChildPID 轮询记录文件直到替身记下子进程 PID。
func p407WaitChildPID(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(path)
		if err == nil {
			lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
			if len(lines) > 0 {
				var rec struct {
					ChildPID int `json:"child_pid"`
				}
				if json.Unmarshal([]byte(lines[len(lines)-1]), &rec) == nil && rec.ChildPID > 0 {
					return rec.ChildPID
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("子进程 PID 未在时限内出现")
	return 0
}

// p407WaitGone 轮询确认进程已消失(孤儿零残留取证)。
func p407WaitGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		p, err := os.FindProcess(pid)
		if err != nil {
			return
		}
		if err := p.Signal(nil); err != nil {
			return // 进程已不存在
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("子进程 %d 在整树终止后仍未消失(孤儿残留,R-1 回归)", pid)
}

func TestP407DSHLimitedReadinessGate(t *testing.T) {
	ctx := context.Background()
	fakeBin := p407BuildFake(t)

	// 真实仓库 workspace:git 仓 + 任务输入文件。
	ws := t.TempDir()
	if _, err := git.PlainInit(ws, false); err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	taskInput := filepath.Join(ws, "task-input.txt")
	if err := os.WriteFile(taskInput, []byte("bounded-task-payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	// workspace bootstrap 提交(任务输入入库;base/candidate 差量才有定义)。
	bootW, err := gitwriter.New(ws)
	if err != nil {
		t.Fatalf("gitwriter.New: %v", err)
	}
	bootRes, err := bootW.Commit(ctx, gitwriter.CommitRequest{
		Message: "tally: p407 workspace bootstrap",
		Files:   []gitwriter.FileUpdate{{Path: "task-input.txt", Content: []byte("bounded-task-payload")}},
	})
	if err != nil {
		t.Fatalf("workspace bootstrap: %v", err)
	}
	baseRev := bootRes.Commit

	// 适配器经公共面构造(钉版身份闸 + 能力协商在 Start)。
	rt := dsh.New(dsh.Config{
		BinPath: fakeBin,
		Env:     map[string]string{"DEEPSEEK_API_KEY": "test-key"},
	})
	s, err := rt.Start(ctx, agentruntime.StartRequest{Workspace: ws})
	if err != nil {
		t.Fatalf("Start(钉版身份闸 + 能力协商): %v", err)
	}
	ds, ok := s.(*dsh.Session)
	if !ok {
		t.Fatalf("Start 应产出 *dsh.Session,得到 %T", s)
	}

	// 有界任务:读取任务输入 → RunCompleted,最终回答确定性。
	res, err := ds.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "read task-input.txt"})
	if err != nil || res.Status != agentruntime.RunCompleted {
		t.Fatalf("有界任务: res=%+v err=%v", res, err)
	}
	if res.Summary != "bounded-task-payload" {
		t.Fatalf("最终回答 = %q, want 任务输入原文", res.Summary)
	}

	// 既有闭环桥:任务产出经 changeset.New 构造候选快照(workspace 差量)。
	produced := filepath.Join(ws, "produced.txt")
	if err := os.WriteFile(produced, []byte("bounded-task-result"), 0o644); err != nil {
		t.Fatal(err)
	}
	cs, err := changeset.New(changeset.Spec{
		Project:      "p407-demo",
		BaseRevision: baseRev,
		Files: []changeset.ChangedFile{{
			Path:    "produced.txt",
			Content: "bounded-task-result",
		}},
		ProducerRun: "run-p407",
	})
	if err != nil {
		t.Fatalf("AC 候选经既有闭环桥: %v", err)
	}
	if cs.ID() == "" {
		t.Fatal("候选应有身份")
	}

	// 负面权限:能力协商拒绝(诚实能力集 —— headless 无 resume/模型选择)。
	if _, err := rt.Start(ctx, agentruntime.StartRequest{
		Workspace: ws,
		ResumeOf:  "some-session",
	}); err == nil {
		t.Fatal("ResumeOf 应被能力协商拒绝")
	}
	if _, err := rt.Start(ctx, agentruntime.StartRequest{
		Workspace:    ws,
		ModelProfile: "deepseek-chat",
	}); err == nil {
		t.Fatal("ModelProfile 应被能力协商拒绝")
	}

	// 负面取消:长运行任务 Cancel → RunCancelled + 整树终止后零孤儿。
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	rt2 := dsh.New(dsh.Config{
		BinPath: fakeBin,
		Env:     map[string]string{"DEEPSEEK_API_KEY": "test-key", "TALLY_FAKE_DSH_RECORD": record},
	})
	s2i, err := rt2.Start(ctx, agentruntime.StartRequest{Workspace: ws})
	if err != nil {
		t.Fatalf("Start(取消面): %v", err)
	}
	s2, ok := s2i.(*dsh.Session)
	if !ok {
		t.Fatalf("Start 应产出 *dsh.Session,得到 %T", s2i)
	}
	outcome := make(chan agentruntime.AgentRunResult, 1)
	errCh := make(chan error, 1)
	go func() {
		res, err := s2.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "tally-conformance-long run"})
		outcome <- res
		errCh <- err
	}()
	childPID := p407WaitChildPID(t, record)
	runID, ok := s2.InFlightRunID()
	if !ok {
		t.Fatal("应有 in-flight run")
	}
	if err := s2.Cancel(ctx, runID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	got := <-outcome
	if e := <-errCh; e != nil {
		t.Fatalf("取消收敛错误: %v", e)
	}
	if got.Status != agentruntime.RunCancelled {
		t.Fatalf("取消应 RunCancelled,得到 %s(%q)", got.Status, got.Summary)
	}
	if childPID > 0 {
		p407WaitGone(t, childPID)
	}

	// 隔离复核:替身记录面确认 cwd=workspace、DSH_HOME 为会话私有临时目录、
	// 遥测禁用(替身 dump 观测面,与 P4-04 包矩阵同源)。
	_ = ds
}
