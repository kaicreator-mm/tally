package main

// P4-02:Codex runtime 经组合注册表(factory)的回归证明。
//
// P4-01 已证明 factory 产物与直接构造在描述符/Start 失败面逐字一致(结构性等
// 价);本文件补齐**行为级**回归:经 factory 拿到的生产 CodexRuntime 必须在完整
// 协议面(身份/取消/workspace/事件/结构化输出/resume/fork,PRD §35 全场景)与
// 组合面(claim → 候选固定 → 不可变 ChangeSet → 最终验证)上行为不变,且失败
// claim 绝不伪造候选(fail-safe 不因 factory 而变)。
//
// 测试驱动方式与 runtimes/codex 同构:现场构建一个只实现探针记录观测表面的
// 替身 codex CLI,真实经过进程边界/JSONL/取消/崩溃。替身刻意最小(只建模本回
// 归驱动的表面);深 CLI 表面(argv 顺序、clap 拒绝、警告权威化等)的权威探测
// 仍在 runtimes/codex 的测试与本回归共同运行的 runtimeconformance 裁决里。
//
// 边界:零生产代码改动 —— 本任务是回归证明,不是行为变更;runtimes/codex 保持
// 行为不变(PATH_OWNERSHIP:regression-only),Git/验证/验收权威零移入 runtime。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"tally/internal/agentruntime"
	"tally/internal/runstore"
	"tally/internal/verification"
	"tally/internal/workflow"
	"tally/runtimes/codex"
	"tally/test/runtimeconformance"
)

// ---- 替身 codex CLI(现场构建,进程级真实)----------------------------------

// factoryCodexLongMarker 是替身识别「长运行指令」的标记:conformance 的
// LongRunInstruction 注入点(阻塞直到被 Cancel/ctx/外部 kill 终止)。
const factoryCodexLongMarker = "tally-factory-long"

// factoryCodexBinPath 是现场构建的替身 CLI 路径(进程生命周期内构建一次)。
var (
	factoryCodexOnce   sync.Once
	factoryCodexBin    string
	factoryCodexBinErr error
)

// buildFactoryCodexBin 构建替身 codex CLI;失败 FailNow(不是跳过:替身构建
// 失败是测试自身缺陷,不是环境前提)。
func buildFactoryCodexBin(t *testing.T) string {
	t.Helper()
	factoryCodexOnce.Do(func() {
		dir, err := os.MkdirTemp("", "tally-factory-fake-codex-*")
		if err != nil {
			factoryCodexBinErr = fmt.Errorf("建临时目录: %w", err)
			return
		}
		src := filepath.Join(dir, "fake_codex_main.go")
		if err := os.WriteFile(src, []byte(factoryCodexSource), 0o644); err != nil {
			factoryCodexBinErr = fmt.Errorf("写替身源码: %w", err)
			return
		}
		exe := filepath.Join(dir, "fakecodex.exe")
		if out, err := exec.Command("go", "build", "-o", exe, src).CombinedOutput(); err != nil {
			factoryCodexBinErr = fmt.Errorf("构建替身 codex CLI: %v\n%s", err, out)
			return
		}
		factoryCodexBin = exe
	})
	if factoryCodexBinErr != nil {
		t.Fatalf("构建替身 codex CLI: %v", factoryCodexBinErr)
	}
	return factoryCodexBin
}

// factoryCodexRecord 是替身 exec 调用记录行的 Go 侧投影(workspace 钉定断言)。
type factoryCodexRecord struct {
	Mode    string   `json:"mode"`
	Thread  string   `json:"thread"`
	Cwd     string   `json:"cwd"`
	CwdFlag string   `json:"cwd_flag"`
	Argv    []string `json:"argv"`
}

// lastFactoryCodexRecord 读取替身记录文件的最后一条。
func lastFactoryCodexRecord(t *testing.T, path string) factoryCodexRecord {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取替身记录 %s: %v", path, err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[len(lines)-1]) == "" {
		t.Fatalf("替身记录为空: %s", path)
	}
	var rec factoryCodexRecord
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &rec); err != nil {
		t.Fatalf("解析替身记录: %v", err)
	}
	return rec
}

// factoryCodexSource 是替身 codex CLI 的完整源码。只实现本回归驱动的实测表面:
//
//	fakecodex --version
//	fakecodex exec [--json --skip-git-repo-check -C <dir> -s <mode> -m <model>
//	                --output-schema <file>] -o <file> [resume|fork <id>] <prompt>
//
// 行为(env 注入,经 codex.Config.Env 透传到被 spawn 进程):
//   - TALLY_FACTORY_CODEX_RECORD=<path>:每次 exec 追加一条 JSON 调用记录
//     (cwd + -C 值,供 WorkspaceEffect 探针断言 workspace 钉定);
//   - TALLY_FACTORY_CODEX_SCENARIO=fail:stderr + exit 7,不产出 -o 产物
//     (负例:真实失败必须仍判失败 claim);
//   - TALLY_FACTORY_CODEX_WRITE_PATH/_CONTENT:完成前向进程 cwd(适配器钉定的
//     workspace)写一个文件(组合回归的「实现产出」);
//   - 指令含 tally-factory-long:阻塞直到被 kill(取消/崩溃场景);
//   - resume 续用请求的 thread id;new/fork 落新 thread id(实测表面);
//   - 带 --output-schema 时最终消息为 {"answer":"ok"}(schema 形态),否则 "ok"。
const factoryCodexSource = `package main

import (
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		os.Exit(2)
	}
	if args[0] == "--version" {
		fmt.Println("codex-cli 0.0.0-factory-regression")
		return
	}
	if args[0] != "exec" {
		fmt.Fprintf(os.Stderr, "fake-codex: 不支持的子命令 %q\n", args[0])
		os.Exit(2)
	}

	// 旗标全部在 exec 层(子命令之前);进入 resume/fork 后不再接受旗标。
	var (
		cwdFlag, sandbox, model, schemaFile, outFile string
		mode, threadArg                              string
		prompt                                       []string
	)
	rest := args[1:]
	inSub := false
	for i := 0; i < len(rest); i++ {
		tok := rest[i]
		if !inSub && (tok == "resume" || tok == "fork") {
			mode, inSub = tok, true
			continue
		}
		if strings.HasPrefix(tok, "-") && len(tok) > 1 {
			if inSub {
				fmt.Fprintf(os.Stderr, "error: unexpected argument '%s' found\n", tok)
				os.Exit(2)
			}
			var val string
			switch tok {
			case "--json", "--skip-git-repo-check":
			case "-C", "--cd", "-s", "--sandbox", "-m", "--model", "--output-schema", "-o", "--output-last-message":
				if i+1 >= len(rest) {
					os.Exit(2)
				}
				i++
				val = rest[i]
			default:
				fmt.Fprintf(os.Stderr, "error: unexpected argument '%s' found\n", tok)
				os.Exit(2)
			}
			switch tok {
			case "-C", "--cd":
				cwdFlag = val
			case "-s", "--sandbox":
				sandbox = val
			case "-m", "--model":
				model = val
			case "--output-schema":
				schemaFile = val
			case "-o", "--output-last-message":
				outFile = val
			}
			continue
		}
		if inSub {
			if threadArg == "" {
				threadArg = tok
			} else {
				prompt = append(prompt, tok)
			}
		} else {
			prompt = append(prompt, tok)
		}
	}
	instr := strings.Join(prompt, " ")
	cwd, _ := os.Getwd()
	record(mode, threadArg, cwd, cwdFlag, sandbox, model, os.Args)

	if os.Getenv("TALLY_FACTORY_CODEX_SCENARIO") == "fail" {
		fmt.Fprintln(os.Stderr, "fake-codex: simulated codex exec failure")
		os.Exit(7)
	}

	// 实测表面:new/fork 落新 thread id,resume 续用请求的 thread id。
	threadID := newThreadID()
	if mode == "resume" {
		threadID = threadArg
	}
	emit(map[string]any{"type": "thread.started", "thread_id": threadID})
	emit(map[string]any{"type": "turn.started"})

	if strings.Contains(instr, "tally-factory-long") {
		time.Sleep(10 * time.Minute) // 由适配器 Cancel/ctx 或外部 kill 终止
		return
	}

	// 模拟实现产出:向进程 cwd(适配器 -C/cwd 钉定的 workspace)写配置文件。
	if p := os.Getenv("TALLY_FACTORY_CODEX_WRITE_PATH"); p != "" {
		target := p
		if !filepath.IsAbs(target) {
			target = filepath.Join(cwd, filepath.FromSlash(target))
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err == nil {
			_ = os.WriteFile(target, []byte(os.Getenv("TALLY_FACTORY_CODEX_WRITE_CONTENT")), 0o644)
		}
	}

	final := "ok"
	if schemaFile != "" {
		final = ` + "`" + `{"answer":"ok"}` + "`" + `
	}
	writeLastMessage(outFile, final)
	emit(map[string]any{"type": "turn.completed"})
}

func emit(v map[string]any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	fmt.Println(string(b))
}

func writeLastMessage(path, msg string) {
	if path == "" {
		return
	}
	_ = os.WriteFile(path, []byte(msg), 0o644)
}

func record(mode, threadArg, cwd, cwdFlag, sandbox, model string, argv []string) {
	path := os.Getenv("TALLY_FACTORY_CODEX_RECORD")
	if path == "" {
		return
	}
	b, err := json.Marshal(map[string]any{
		"mode": mode, "thread": threadArg, "cwd": cwd, "cwd_flag": cwdFlag,
		"sandbox": sandbox, "model": model, "argv": argv,
	})
	if err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(b, '\n'))
}

func newThreadID() string {
	var b [16]byte
	if _, err := crand.Read(b[:]); err != nil {
		return fmt.Sprintf("00000000-0000-0000-0000-%012d", time.Now().UnixNano()%1000000000000)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32])
}
`

// buildFactoryCodexRuntime 经注册表缝构建挂在替身 CLI 上的 CodexRuntime:构造
// 式与生产 builder 完全同形(只携带 CodexBin —— factory 面唯一的生产配置)。
// 替身行为开关经 t.Setenv 注入:runner 把 spec.Env 追加到继承环境之后,被
// spawn 的替身能看到测试进程环境;factory 面零改动。
func buildFactoryCodexRuntime(t *testing.T) agentruntime.Runtime {
	t.Helper()
	rt, err := newRuntimeRegistry().build(v13Config{Runtime: codex.DescriptorName, CodexBin: buildFactoryCodexBin(t)})
	if err != nil {
		t.Fatalf("factory build(codex): %v", err)
	}
	if got := rt.Descriptor().Name; got != codex.DescriptorName {
		t.Fatalf("factory 产物 = %s, want codex", got)
	}
	return rt
}

// ---- 回归 1:PRD §35 全场景 conformance 经 factory 产物 -----------------------

// TestCodexRuntimeThroughFactoryPassesConformance 对 factory 构建的
// CodexRuntime 跑完整 runtimeconformance 套件:start / cwd / execute / result /
// events / cancel / crash / workspace / usage / capability_truth,以及已声明
// 能力专项 cap:session.resume / cap:session.fork / cap:output.structured。
// 身份(描述符/能力真伪)、取消、workspace 钉定、事件语义全部经工厂产物验证。
func TestCodexRuntimeThroughFactoryPassesConformance(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	t.Setenv("TALLY_FACTORY_CODEX_RECORD", record)
	rt := buildFactoryCodexRuntime(t)
	codexRt := rt.(*codex.Runtime)

	rep := runtimeconformance.Run(t, rt, runtimeconformance.Options{
		Probes: runtimeconformance.Probes{
			LongRunInstruction: factoryCodexLongMarker + " run",
			CrashRun: func(ctx context.Context, sess agentruntime.Session, longRunInstruction string) (agentruntime.AgentRunResult, error) {
				cs, ok := sess.(*codex.Session)
				if !ok {
					return agentruntime.AgentRunResult{}, fmt.Errorf("crash probe: want *codex.Session, got %T", sess)
				}
				st, err := sess.Events(ctx)
				if err != nil {
					return agentruntime.AgentRunResult{}, err
				}
				defer func() { _ = st.Close() }()
				type outcome struct {
					res agentruntime.AgentRunResult
					err error
				}
				ch := make(chan outcome, 1)
				go func() {
					res, err := sess.Execute(ctx, agentruntime.ExecuteRequest{Instruction: longRunInstruction})
					ch <- outcome{res, err}
				}()
				ev, err := st.Next(ctx)
				if err != nil {
					return agentruntime.AgentRunResult{}, fmt.Errorf("等待 run.started: %w", err)
				}
				if ev.Kind != agentruntime.EventRunStarted {
					return agentruntime.AgentRunResult{}, fmt.Errorf("首事件应为 run.started, got %s", ev.Kind)
				}
				proc, ok := cs.RunProcess(ev.RunID)
				if !ok {
					return agentruntime.AgentRunResult{}, fmt.Errorf("in-flight run 无底层进程可注入崩溃")
				}
				if err := proc.Kill(); err != nil {
					return agentruntime.AgentRunResult{}, fmt.Errorf("注入进程崩溃: %w", err)
				}
				select {
				case out := <-ch:
					return out.res, out.err
				case <-ctx.Done():
					return agentruntime.AgentRunResult{}, ctx.Err()
				}
			},
			// fork 探针用 factory 产物自身的 Fork:证明工厂产物的派生语义。
			Fork: func(ctx context.Context, parent agentruntime.Session) (agentruntime.Session, error) {
				return codexRt.Fork(ctx, parent)
			},
			StructuredOutput: func(ctx context.Context, sess agentruntime.Session) error {
				cs, ok := sess.(*codex.Session)
				if !ok {
					return fmt.Errorf("structured probe: want *codex.Session, got %T", sess)
				}
				if err := cs.SetOutputSchema(`{"required":["answer"]}`); err != nil {
					return err
				}
				res, err := sess.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "conformance: structured"})
				if err != nil {
					return err
				}
				if res.Status != agentruntime.RunCompleted {
					return fmt.Errorf("符合 schema 的回合应完成: %s/%q", res.Status, res.Summary)
				}
				var parsed map[string]any
				if jerr := json.Unmarshal([]byte(res.Summary), &parsed); jerr != nil {
					return fmt.Errorf("最终消息不是 JSON: %q (%v)", res.Summary, jerr)
				}
				if _, ok := parsed["answer"]; !ok {
					return fmt.Errorf("最终消息缺少 schema required 属性 answer: %q", res.Summary)
				}
				return nil
			},
			WorkspaceEffect: func(ctx context.Context, sess agentruntime.Session) error {
				cs, ok := sess.(*codex.Session)
				if !ok {
					return fmt.Errorf("workspace probe: want *codex.Session, got %T", sess)
				}
				rec := lastFactoryCodexRecord(t, record)
				if rec.CwdFlag != cs.Workspace() || rec.Cwd != cs.Workspace() {
					return fmt.Errorf("替身观测到的执行目录 (-C=%q, cwd=%q) 不是提供的 workspace %q", rec.CwdFlag, rec.Cwd, cs.Workspace())
				}
				return nil
			},
		},
	})
	if !rep.OK() {
		t.Fatalf("factory 产物的 CodexRuntime conformance 未通过:\n%s", rep.FailureText())
	}
	// 能力专项必须真的跑过(声明即验证):身份面经工厂产物不变。
	for _, scenario := range []string{
		"cap:session.resume", "cap:session.fork", "cap:output.structured",
	} {
		if !rep.Ran(scenario) {
			t.Fatalf("场景 %s 未运行(声明了对应能力,必须验证)", scenario)
		}
	}
	// 未声明 session.isolation → canary 不运行(声明真实性不变)。
	if rep.Ran("cap:session.isolation") {
		t.Fatal("未声明 session.isolation,canary 场景不应运行")
	}
}

// ---- 回归 2:组合面 —— factory 选择的 codex 走完 CodegenRun→ChangeSet→验证 ----

// TestV13CodexRuntimeThroughFactoryProducesChangeSet 走生产注册表
// (workerRuntimes,deps.runtime=nil 的生产路径):cfg.Runtime="codex" 经
// factory 选出 CodexRuntime,替身在 worktree 内产出实现文件,垂直路径必须与
// fake 路径同构地固定不可变 ChangeSet 并通过最终验证;runtime 身份 "codex"
// 必须如实落入 run 事实。
func TestV13CodexRuntimeThroughFactoryProducesChangeSet(t *testing.T) {
	repo := newV13Repo(t)
	cfg := v13ConfigFor(t, repo)
	cfg.Runtime = "codex"
	cfg.CodexBin = buildFactoryCodexBin(t)
	// 替身的「实现产出」经继承环境注入(生产 builder 只携带 CodexBin ——
	// factory 面零改动);写进进程 cwd = 适配器钉定的一次性 worktree。
	t.Setenv("TALLY_FACTORY_CODEX_WRITE_PATH", "internal/widget/greeting.go")
	t.Setenv("TALLY_FACTORY_CODEX_WRITE_CONTENT", "package widget\n")

	wf := &v13MemRunSink{}
	vs := &v13MemVerifySink{}
	res, err := runV13Task(context.Background(), v13Deps{
		cfg:          cfg,
		workflowSink: wf,
		verifySink:   vs,
	}, "TASK-1043")
	if err != nil {
		t.Fatalf("runV13Task: %v", err)
	}

	// 垂直路径走完:ChangeSet 固定,内容来自替身产出(经 workspace 钉定落入
	// 一次性 worktree,再被候选固定钩子收取)。
	if res.Codegen.Status != workflow.TaskCompleted || len(res.Codegen.Runs) != 1 {
		t.Fatalf("codegen 侧终态 = %s/%d run, want completed/1", res.Codegen.Status, len(res.Codegen.Runs))
	}
	cs := res.Codegen.ChangeSet
	if cs == nil {
		t.Fatal("ChangeSet = nil, want 固定候选")
	}
	found := false
	for _, f := range cs.Files() {
		if f.Path == "internal/widget/greeting.go" && strings.Contains(f.Content, "package widget") {
			found = true
		}
	}
	if !found {
		t.Fatalf("候选缺少替身实现产出: %v", cs.Files())
	}

	// 身份:factory 选择的 runtime 名字如实进入 run 事实(不因 factory 变形)。
	if len(wf.begins) != 1 {
		t.Fatalf("run 事实数 = %d, want 1", len(wf.begins))
	}
	if got := wf.begins[0].RuntimeName; got != "codex" {
		t.Fatalf("RunBegin.RuntimeName = %q, want codex(经 factory 的身份归因)", got)
	}
	if wf.finishes[0].Status != runstore.RunCompleted {
		t.Fatalf("RunFinish = %+v, want completed", wf.finishes[0])
	}

	// 最终验证:visible binding(go version)通过 → verified;codex 路径的
	// 验证/验收权威仍在验证编排侧,零移入 runtime。
	if res.Verify == nil || res.Verify.Status != verification.FinalVerified {
		t.Fatalf("Verification = %+v, want verified", res.Verify)
	}

	// owner 主 worktree 零污染;一次性 worktree 全部回收(既有纪律不变)。
	if dirty := v13GitOut(t, repo, "status", "--porcelain"); dirty != "" {
		t.Errorf("主 worktree 被污染: %q", dirty)
	}
	if entries, err := os.ReadDir(cfg.WorktreeRoot); err != nil || len(entries) != 0 {
		t.Errorf("一次性根目录应清空: entries=%d err=%v", len(entries), err)
	}
}

// ---- 回归 3(负例):失败 claim 绝不伪造候选 ----------------------------------

// TestV13CodexRuntimeFailedClaimNeverFixesCandidate:替身以非零退出失败,经
// factory 的 codex 产出 RunFailed claim;内核按失败迭代收束 —— 零 ChangeSet、
// 零最终验证,run 终态非 completed。信任不变量(claim ≠ 事实)经工厂路径不变。
func TestV13CodexRuntimeFailedClaimNeverFixesCandidate(t *testing.T) {
	repo := newV13Repo(t)
	cfg := v13ConfigFor(t, repo)
	cfg.Runtime = "codex"
	cfg.CodexBin = buildFactoryCodexBin(t)
	t.Setenv("TALLY_FACTORY_CODEX_SCENARIO", "fail")

	wf := &v13MemRunSink{}
	res, err := runV13Task(context.Background(), v13Deps{
		cfg:          cfg,
		workflowSink: wf,
		verifySink:   &v13MemVerifySink{},
	}, "TASK-1043")
	if err != nil {
		t.Fatalf("领域内失败是正常结果(job 成功),不应返回 error: %v", err)
	}
	if res.Codegen.ChangeSet != nil || len(wf.changeSets) != 0 {
		t.Fatalf("失败 claim 不得固定候选: ChangeSet=%v sink=%d", res.Codegen.ChangeSet, len(wf.changeSets))
	}
	if res.Verify != nil {
		t.Fatalf("无固定候选不得进入最终验证: %+v", res.Verify)
	}
	if len(wf.finishes) == 0 || wf.finishes[0].Status == runstore.RunCompleted {
		t.Fatalf("run 应以失败/升级收束: finishes=%+v", wf.finishes)
	}
}
