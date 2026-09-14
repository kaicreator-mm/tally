package main

// P4-01 的 runtime 组合注册表回归测试。
//
// 覆盖任务验收与 L3 Failure Handling:
//   - 既有 fake/codex 路径经 factory 与直接构造行为一致(描述符逐字一致、
//     CodexBin 确实进入适配器配置);组合流程行为本身由 v13/v14 既有回归
//     测试原样覆盖(TestV13VerticalPathEndToEnd 等);
//   - 未知 runtime 在任务执行前被拒绝:零 run 事实、零一次性 worktree;
//   - 注册缝可用:登记一个 builder 即可从同一缝选择新适配器(未来 DSH 走
//     同一路径),不产生第二执行协议。

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tally/internal/agentruntime"
	"tally/runtimes/codex"
	"tally/runtimes/fake"
)

// descriptorEquals 逐字段比较两个 RuntimeDescriptor(CapabilitySet 是 map,
// 结构体不可 == 比较;能力按排序序列表比较)。
func descriptorEquals(t *testing.T, got, want agentruntime.RuntimeDescriptor) {
	t.Helper()
	if got.Name != want.Name || got.Version != want.Version {
		t.Fatalf("descriptor name/version = %q/%q, want %q/%q", got.Name, got.Version, want.Name, want.Version)
	}
	gc, wc := got.Capabilities.Sorted(), want.Capabilities.Sorted()
	if len(gc) != len(wc) {
		t.Fatalf("capabilities = %v, want %v", gc, wc)
	}
	for i := range gc {
		if gc[i] != wc[i] {
			t.Fatalf("capabilities = %v, want %v", gc, wc)
		}
	}
}

// factory 经 factory 构造的适配器与组合根直接构造逐字一致(AC:既有
// fake/codex 路径行为不变)。
func TestRuntimeFactoryMatchesDirectConstruction(t *testing.T) {
	reg := newRuntimeRegistry()

	rt, err := reg.build(v13Config{Runtime: "fake"})
	if err != nil {
		t.Fatalf("build(fake): %v", err)
	}
	descriptorEquals(t, rt.Descriptor(), fake.New(fake.Config{}).Descriptor())

	rt, err = reg.build(v13Config{Runtime: "codex"})
	if err != nil {
		t.Fatalf("build(codex): %v", err)
	}
	descriptorEquals(t, rt.Descriptor(), codex.New(codex.Config{}).Descriptor())
}

// CodexBin 配置确实经 factory 进入适配器:用一个不存在的显式路径,factory
// 产物与直接构造的 Start 失败完全一致(requireFile 存在性检查先于任何进程
// 动作,全程零 spawn)。若 factory 弄丢 BinPath,适配器会改走发现流程,
// 失败形态不同。
func TestRuntimeFactoryPlumbsCodexBin(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-codex")

	reg := newRuntimeRegistry()
	viaFactory, err := reg.build(v13Config{Runtime: "codex", CodexBin: missing})
	if err != nil {
		t.Fatalf("build(codex, CodexBin): %v", err)
	}
	direct := codex.New(codex.Config{BinPath: missing})

	req := agentruntime.StartRequest{Workspace: t.TempDir()}
	_, errFactory := viaFactory.Start(context.Background(), req)
	_, errDirect := direct.Start(context.Background(), req)
	if errFactory == nil || errDirect == nil {
		t.Fatalf("不存在的 BinPath 应在 Start 处失败: factory=%v direct=%v", errFactory, errDirect)
	}
	if errFactory.Error() != errDirect.Error() {
		t.Fatalf("factory 产物与直接构造行为不一致:\n factory: %v\n direct:  %v", errFactory, errDirect)
	}
	if !strings.Contains(errFactory.Error(), "no-such-codex") {
		t.Fatalf("失败应指向显式 BinPath(证明配置已进入适配器): %v", errFactory)
	}
}

// Failure Handling:未知 runtime 在任务执行前被拒绝 —— 零 run 事实、
// 零一次性 worktree(fail-closed,绝不静默降级到缺省 runtime)。
func TestRuntimeFactoryRejectsUnknownBeforeExecution(t *testing.T) {
	repo := newV13Repo(t)
	cfg := v13ConfigFor(t, repo)
	cfg.Runtime = "magic"

	wf := &v13MemRunSink{}
	_, err := runV13Task(context.Background(), v13Deps{
		cfg:          cfg,
		workflowSink: wf,
		verifySink:   &v13MemVerifySink{},
	}, "TASK-1043")
	if err == nil || !strings.Contains(err.Error(), "未知 runtime") || !strings.Contains(err.Error(), "magic") {
		t.Fatalf("未知 runtime 应被拒绝并点名: %v", err)
	}
	// 任务执行前拒绝:没有任何 run 事实落 sink,一次性 worktree 根未被创建。
	if len(wf.begins) != 0 || len(wf.finishes) != 0 {
		t.Fatalf("拒绝执行不得落 run 事实: begins=%d finishes=%d", len(wf.begins), len(wf.finishes))
	}
	if _, serr := os.Stat(cfg.WorktreeRoot); !os.IsNotExist(serr) {
		t.Errorf("拒绝执行不得创建一次性 worktree 根: %v", serr)
	}
}

// stubRuntime 是注册缝的注入替身:只携带名字的协议值实现。
type stubRuntime struct{ name string }

func (s stubRuntime) Descriptor() agentruntime.RuntimeDescriptor {
	return agentruntime.RuntimeDescriptor{Name: s.name}
}

func (s stubRuntime) Start(context.Context, agentruntime.StartRequest) (agentruntime.Session, error) {
	return nil, errors.New("stubRuntime: 注册缝测试不应开启会话")
}

// 注册缝:登记一个 builder 即可按名字选择新适配器(v13/v14 组合流程零改动);
// 缺省名字与既有登记不受影响;错误提示按登记键序自述支持集合。
func TestRuntimeFactoryRegisterExtendsSeam(t *testing.T) {
	reg := newRuntimeRegistry()
	reg.register("stub", func(v13Config) (agentruntime.Runtime, error) {
		return stubRuntime{name: "stub"}, nil
	})

	rt, err := reg.build(v13Config{Runtime: "stub"})
	if err != nil {
		t.Fatalf("build(stub): %v", err)
	}
	if got := rt.Descriptor().Name; got != "stub" {
		t.Fatalf("runtime = %s, want stub", got)
	}

	// 既有登记与缺省不受新登记影响。
	dflt, err := reg.build(v13Config{})
	if err != nil || dflt.Descriptor().Name != fake.DescriptorName {
		t.Fatalf("缺省仍应为 fake: %v %v", dflt.Descriptor().Name, err)
	}

	// 支持集合由登记自述。
	_, err = reg.build(v13Config{Runtime: "magic"})
	if err == nil || !strings.Contains(err.Error(), "stub") {
		t.Fatalf("错误应列出登记的支持集(含 stub): %v", err)
	}
}
