package runtimeconformance

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"tally/internal/agentruntime"
)

// ---- 测试专用参考适配器 ---------------------------------------------------
//
// cfRuntime 是一个协议完整的参考实现,用于验证 harness 本身(测试 harness
// 的测试):协议合规的 runtime 必须全绿;故意违规的 runtime 必须被抓出。
// 它不承担 P1-24 参考适配器的职责,只服务本包测试。

// cfLongRun 与 Options.Probes.LongRunInstruction 配套:cfRuntime 收到这条
// 指令就让回合保持 in-flight,直到被 Cancel、ctx 取消或崩溃。
const cfLongRun = "conformance: 保持运行直到被取消或崩溃"

// cfOptions 控制 cfRuntime 的能力声明与故意违规点。
type cfOptions struct {
	caps              []agentruntime.Capability
	acceptNoWorkspace bool // 违规:跳过 CheckStart 闸,接受无 workspace 的 Start
	crashReportsOK    bool // 违规:进程崩溃上报为 RunCompleted
	crashAsError      bool // 合规变体:崩溃以 error 而非 RunFailed claim 上报
}

type cfRuntime struct {
	desc agentruntime.RuntimeDescriptor
	opt  cfOptions

	mu       sync.Mutex
	next     int
	sessions map[agentruntime.SessionID]*cfSession
}

func newCFRuntime(opt cfOptions) *cfRuntime {
	return &cfRuntime{
		desc: agentruntime.RuntimeDescriptor{
			Name:         "conformance-fake",
			Version:      "test",
			Capabilities: agentruntime.NewCapabilitySet(opt.caps...),
		},
		opt:      opt,
		sessions: map[agentruntime.SessionID]*cfSession{},
	}
}

func (r *cfRuntime) Descriptor() agentruntime.RuntimeDescriptor { return r.desc }

func (r *cfRuntime) Start(ctx context.Context, req agentruntime.StartRequest) (agentruntime.Session, error) {
	if !r.opt.acceptNoWorkspace {
		if err := agentruntime.CheckStart(r.desc, req); err != nil {
			return nil, err
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if req.ResumeOf != "" {
		s, ok := r.sessions[req.ResumeOf]
		if !ok {
			return nil, fmt.Errorf("cfRuntime: 未知会话 %s", req.ResumeOf)
		}
		// 续用既有会话:重新打开协议句柄,逻辑会话(含既有 runs)不变。
		s.mu.Lock()
		s.closed = false
		s.subs = map[*cfStream]struct{}{}
		s.mu.Unlock()
		return s, nil
	}
	r.next++
	s := &cfSession{
		rt:   r,
		id:   agentruntime.SessionID(fmt.Sprintf("cf-sess-%d", r.next)),
		runs: map[agentruntime.RunID]*cfRun{},
		subs: map[*cfStream]struct{}{},
	}
	r.sessions[s.id] = s
	return s, nil
}

// cfRun 是会话内一次运行的测试实现。
type cfRun struct {
	id     agentruntime.RunID
	cancel chan struct{}
	crash  chan struct{}
	done   chan struct{} // 终态落定后关闭

	mu     sync.Mutex
	result agentruntime.AgentRunResult
}

func (r *cfRun) finished() bool {
	select {
	case <-r.done:
		return true
	default:
		return false
	}
}

type cfSession struct {
	rt *cfRuntime
	id agentruntime.SessionID

	mu      sync.Mutex
	closed  bool
	nextRun int
	seq     int64
	runs    map[agentruntime.RunID]*cfRun
	subs    map[*cfStream]struct{}
}

func (s *cfSession) Execute(ctx context.Context, req agentruntime.ExecuteRequest) (agentruntime.AgentRunResult, error) {
	if err := req.Validate(); err != nil {
		return agentruntime.AgentRunResult{}, err
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return agentruntime.AgentRunResult{}, agentruntime.ErrSessionClosed
	}
	s.nextRun++
	run := &cfRun{
		id:     agentruntime.RunID(fmt.Sprintf("cf-run-%d", s.nextRun)),
		cancel: make(chan struct{}),
		crash:  make(chan struct{}),
		done:   make(chan struct{}),
	}
	s.runs[run.id] = run
	s.mu.Unlock()

	s.emit(run, agentruntime.EventRunStarted, "")

	if req.Instruction != cfLongRun {
		return s.finish(run, agentruntime.RunCompleted, req.Instruction), nil
	}
	// 长指令:保持 in-flight 直到取消/崩溃。
	var status agentruntime.RunStatus
	var summary string
	select {
	case <-ctx.Done():
		status, summary = agentruntime.RunCancelled, "被取消的 ctx 终止"
	case <-run.cancel:
		status, summary = agentruntime.RunCancelled, "被 Cancel 终止"
	case <-run.crash:
		if s.rt.opt.crashReportsOK {
			return s.finish(run, agentruntime.RunCompleted, "崩溃误报为完成"), nil
		}
		if s.rt.opt.crashAsError {
			s.emit(run, agentruntime.EventRunFailed, "进程崩溃")
			close(run.done)
			return agentruntime.AgentRunResult{}, fmt.Errorf("cfRuntime: run %s 底层进程崩溃", run.id)
		}
		status, summary = agentruntime.RunFailed, "进程崩溃"
	}
	return s.finish(run, status, summary), nil
}

// finish 落终态:发语义事件、用量事件与终态事件,记录 claim 快照。
func (s *cfSession) finish(run *cfRun, status agentruntime.RunStatus, summary string) agentruntime.AgentRunResult {
	s.emit(run, agentruntime.EventAgentMessage, summary)
	s.emit(run, agentruntime.EventUsageReported, "")
	terminal := agentruntime.EventRunCompleted
	switch status {
	case agentruntime.RunFailed:
		terminal = agentruntime.EventRunFailed
	case agentruntime.RunCancelled:
		terminal = agentruntime.EventRunCancelled
	}
	s.emit(run, terminal, "")

	res := agentruntime.AgentRunResult{
		RunID:     run.id,
		SessionID: s.id,
		Status:    status,
		Summary:   summary,
		Usage:     agentruntime.Usage{InputTokens: 11, OutputTokens: 7},
	}
	run.mu.Lock()
	run.result = res
	run.mu.Unlock()
	close(run.done)
	return res
}

// emit 广播一条事件(非阻塞投递,缓冲满即丢)。
func (s *cfSession) emit(run *cfRun, kind agentruntime.EventKind, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	ev := agentruntime.Event{
		Kind:      kind,
		SessionID: s.id,
		RunID:     run.id,
		Seq:       s.seq,
		Time:      time.Now(),
		Text:      text,
	}
	for st := range s.subs {
		select {
		case st.ch <- ev:
		default:
		}
	}
}

// awaitInFlight 等待出现一个未到终态的 run(测试侧确定性同步)。
func (s *cfSession) awaitInFlight(d time.Duration) (*cfRun, error) {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		var cand *cfRun
		for _, r := range s.runs {
			if !r.finished() {
				cand = r
				break
			}
		}
		s.mu.Unlock()
		if cand != nil {
			return cand, nil
		}
		time.Sleep(time.Millisecond)
	}
	return nil, fmt.Errorf("cfRuntime: 等待 in-flight run 超时")
}

func (s *cfSession) Cancel(ctx context.Context, id agentruntime.RunID) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return agentruntime.ErrSessionClosed
	}
	run, ok := s.runs[id]
	s.mu.Unlock()
	if !ok {
		return agentruntime.ErrRunNotFound
	}
	select {
	case <-run.cancel: // 已取消,幂等
	default:
		close(run.cancel)
	}
	return nil
}

func (s *cfSession) Events(ctx context.Context) (agentruntime.EventStream, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, agentruntime.ErrSessionClosed
	}
	if err := agentruntime.CheckEvents(s.rt.desc); err != nil {
		return nil, err
	}
	st := &cfStream{sess: s, ch: make(chan agentruntime.Event, 128)}
	s.subs[st] = struct{}{}
	return st, nil
}

func (s *cfSession) Result(ctx context.Context, id agentruntime.RunID) (agentruntime.AgentRunResult, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return agentruntime.AgentRunResult{}, agentruntime.ErrSessionClosed
	}
	run, ok := s.runs[id]
	s.mu.Unlock()
	if !ok {
		return agentruntime.AgentRunResult{}, agentruntime.ErrRunNotFound
	}
	run.mu.Lock()
	defer run.mu.Unlock()
	return run.result, nil
}

func (s *cfSession) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return agentruntime.ErrSessionClosed
	}
	s.closed = true
	for st := range s.subs {
		close(st.ch)
	}
	s.subs = map[*cfStream]struct{}{}
	return nil
}

type cfStream struct {
	sess *cfSession
	ch   chan agentruntime.Event

	mu          sync.Mutex
	closed      bool
	sawTerminal bool
}

func (st *cfStream) Next(ctx context.Context) (agentruntime.Event, error) {
	st.mu.Lock()
	closed, done := st.closed, st.sawTerminal
	st.mu.Unlock()
	if closed || done {
		return agentruntime.Event{}, agentruntime.ErrEventStreamClosed
	}
	select {
	case <-ctx.Done():
		return agentruntime.Event{}, ctx.Err()
	case ev, ok := <-st.ch:
		if !ok {
			return agentruntime.Event{}, agentruntime.ErrEventStreamClosed
		}
		if ev.Kind.Terminal() {
			st.mu.Lock()
			st.sawTerminal = true
			st.mu.Unlock()
		}
		return ev, nil
	}
}

func (st *cfStream) Close() error {
	st.mu.Lock()
	if st.closed {
		st.mu.Unlock()
		return nil
	}
	st.closed = true
	st.mu.Unlock()
	st.sess.mu.Lock()
	delete(st.sess.subs, st)
	st.sess.mu.Unlock()
	return nil
}

// ---- 参考探针实现 ----------------------------------------------------------

// cfCrashRun 后台执行长指令,等其 in-flight 后注入进程崩溃,并原样返回
// Execute 的返回值(与 Probes.CrashRun 契约一致)。
func cfCrashRun(ctx context.Context, sess agentruntime.Session, longRun string) (agentruntime.AgentRunResult, error) {
	type outcome struct {
		res agentruntime.AgentRunResult
		err error
	}
	ch := make(chan outcome, 1)
	go func() {
		res, err := sess.Execute(ctx, agentruntime.ExecuteRequest{Instruction: longRun})
		ch <- outcome{res: res, err: err}
	}()
	cs, ok := sess.(*cfSession)
	if !ok {
		return agentruntime.AgentRunResult{}, fmt.Errorf("cfCrashRun: 期望 *cfSession,得到 %T", sess)
	}
	run, err := cs.awaitInFlight(5 * time.Second)
	if err != nil {
		return agentruntime.AgentRunResult{}, err
	}
	select {
	case <-run.crash: // 已崩溃,幂等
	default:
		close(run.crash)
	}
	select {
	case out := <-ch:
		return out.res, out.err
	case <-ctx.Done():
		return agentruntime.AgentRunResult{}, ctx.Err()
	}
}

// cfFork 从 parent 派生全新会话(新 id、不共享 runs)。
func cfFork(ctx context.Context, parent agentruntime.Session) (agentruntime.Session, error) {
	p, ok := parent.(*cfSession)
	if !ok {
		return nil, fmt.Errorf("cfFork: 期望 *cfSession,得到 %T", parent)
	}
	p.rt.mu.Lock()
	defer p.rt.mu.Unlock()
	p.rt.next++
	s := &cfSession{
		rt:   p.rt,
		id:   agentruntime.SessionID(fmt.Sprintf("cf-sess-%d", p.rt.next)),
		runs: map[agentruntime.RunID]*cfRun{},
		subs: map[*cfStream]struct{}{},
	}
	p.rt.sessions[s.id] = s
	return s, nil
}

// cfStructuredOutput 驱动一次结构化输出往返;参考实现视为 schema 强制通过。
func cfStructuredOutput(ctx context.Context, sess agentruntime.Session) error {
	_, err := sess.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "conformance: 以 JSON 回复 {\"ok\":true}"})
	return err
}

// cfApprovalDeny 驱动一次 deny 审批;参考实现的 deny 语义视为阻止生效。
func cfApprovalDeny(ctx context.Context, sess agentruntime.Session) error {
	_, err := sess.Execute(ctx, agentruntime.ExecuteRequest{Instruction: "conformance: 请求审批并等待 deny"})
	return err
}

// cfWorkspaceEffect 参考实现不触达文件系统,效果探针恒通过。
func cfWorkspaceEffect(ctx context.Context, sess agentruntime.Session) error { return nil }

// ---- 测试脚手架 -------------------------------------------------------------

// allCapabilities 返回全部会触发能力专项场景的能力。
func allCapabilities() []agentruntime.Capability {
	return []agentruntime.Capability{
		agentruntime.CapEventStreaming,
		agentruntime.CapSessionResume,
		agentruntime.CapSessionFork,
		agentruntime.CapOutputStructured,
		agentruntime.CapToolApproval,
		agentruntime.CapSessionIsolation,
	}
}

// fullProbes 返回与 allCapabilities 配套的完整注入点。
func fullProbes() Probes {
	return Probes{
		LongRunInstruction: cfLongRun,
		CrashRun:           cfCrashRun,
		Fork:               cfFork,
		StructuredOutput:   cfStructuredOutput,
		ApprovalDeny:       cfApprovalDeny,
		WorkspaceEffect:    cfWorkspaceEffect,
	}
}

// coreScenarioNames 是全部核心场景名(对应 PRD §35 通用清单)。
var coreScenarioNames = []string{
	"start", "cwd", "execute", "result", "events",
	"cancel", "crash", "workspace", "usage", "capability_truth",
}

// capScenarioNames 是全部能力专项场景名(名 = "cap:" + 能力标识)。
var capScenarioNames = []string{
	"cap:" + string(agentruntime.CapSessionResume),
	"cap:" + string(agentruntime.CapSessionFork),
	"cap:" + string(agentruntime.CapOutputStructured),
	"cap:" + string(agentruntime.CapToolApproval),
	"cap:" + string(agentruntime.CapSessionIsolation),
}

// failed 报告某场景已执行且失败。
func failed(rep *Report, name string) bool {
	for _, c := range rep.Checks {
		if c.Name == name {
			return len(c.Failures) > 0
		}
	}
	return false
}

// ---- harness 正向验证:合规 runtime 全绿 -------------------------------------

func TestHarnessConformingRuntimePassesAllScenarios(t *testing.T) {
	rep := Run(t, newCFRuntime(cfOptions{caps: allCapabilities()}), Options{Probes: fullProbes()})
	if !rep.OK() {
		t.Fatalf("协议合规的 runtime 应全部通过: %s", rep.FailureText())
	}
	for _, name := range append(append([]string{}, coreScenarioNames...), capScenarioNames...) {
		if !rep.Ran(name) {
			t.Errorf("声明全部能力时应执行场景 %s", name)
		}
	}
}

func TestHarnessCrashReportedAsErrorIsConformant(t *testing.T) {
	// 崩溃的第二条合法上报路径:Execute 返回 error 而非 RunFailed claim。
	rep := Run(t, newCFRuntime(cfOptions{caps: allCapabilities(), crashAsError: true}), Options{Probes: fullProbes()})
	if !rep.OK() {
		t.Fatalf("以 error 上报崩溃也应通过: %s", rep.FailureText())
	}
}

func TestHarnessCapabilityScenariosOnlyRunWhenDeclared(t *testing.T) {
	// AC2:能力专项场景只按声明门控。零能力的 runtime 走核心场景的拒绝路径,
	// 且不得执行任何 cap:* 场景。
	rep := Run(t, newCFRuntime(cfOptions{}), Options{Probes: fullProbes()})
	if !rep.OK() {
		t.Fatalf("零能力 runtime 的核心场景(拒绝路径)应通过: %s", rep.FailureText())
	}
	for _, name := range capScenarioNames {
		if rep.Ran(name) {
			t.Errorf("未声明能力时不应执行场景 %s(AC2)", name)
		}
	}
	for _, name := range []string{"events", "capability_truth", "cancel"} {
		if !rep.Ran(name) {
			t.Errorf("核心场景 %s 应始终执行", name)
		}
	}
}

func TestHarnessPartialCapabilitiesGatesCorrectly(t *testing.T) {
	// 只声明 event.streaming:events 走正向路径,resume 走拒绝路径,cap:* 不执行。
	rep := Run(t, newCFRuntime(cfOptions{caps: []agentruntime.Capability{agentruntime.CapEventStreaming}}), Options{Probes: fullProbes()})
	if !rep.OK() {
		t.Fatalf("只声明 event.streaming 的合规 runtime 应通过: %s", rep.FailureText())
	}
	if rep.Ran(capScenarioNames[0]) {
		t.Error("未声明 session.resume 时不应执行 cap:session.resume")
	}
}

// ---- harness 反向验证:违规 runtime 必须被抓出 --------------------------------

// recordingTB 是 harnessTB 的记录型替身:把失败收进切片而不是写进真实测试
// 输出。负向测试用它在「预期违规被抓出」的同时保持测试本身绿色。
type recordingTB struct {
	failures []string
}

func (r *recordingTB) Helper() {}

func (r *recordingTB) Errorf(format string, args ...any) {
	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}

func (r *recordingTB) FailNow() {
	panic("recordingTB: 意外的 FailNow(负向场景不应走 fatal 路径)")
}

// runSingle 以记录型替身执行单个场景,返回只含该场景的 Report。
func runSingle(t *testing.T, name string, rt agentruntime.Runtime, opts Options) *Report {
	t.Helper()
	rep := &Report{RuntimeName: rt.Descriptor().Name}
	for _, sc := range applicableScenarios(rt.Descriptor().Capabilities) {
		if sc.name == name {
			runOne(&recordingTB{}, rep, rt, opts, sc)
			return rep
		}
	}
	t.Fatalf("场景 %s 不存在或对当前能力声明不适用", name)
	return nil
}

func TestHarnessCatchesWorkspaceGateViolation(t *testing.T) {
	rep := runSingle(t, "workspace", newCFRuntime(cfOptions{acceptNoWorkspace: true}), Options{Probes: fullProbes()})
	if rep.OK() {
		t.Fatal("越过 INV-EXEC-01 workspace 闸的 runtime 必须被抓出")
	}
	if !failed(rep, "workspace") {
		t.Fatalf("workspace 场景应失败,得到: %s", rep.FailureText())
	}
}

func TestHarnessCatchesCrashReportedAsSuccess(t *testing.T) {
	rt := newCFRuntime(cfOptions{caps: allCapabilities(), crashReportsOK: true})
	rep := runSingle(t, "crash", rt, Options{Probes: fullProbes()})
	if rep.OK() {
		t.Fatal("进程崩溃上报为完成必须被抓出")
	}
	if !failed(rep, "crash") {
		t.Fatalf("crash 场景应失败,得到: %s", rep.FailureText())
	}
}

func TestHarnessCatchesUndeclaredResumeAccepted(t *testing.T) {
	// 违规:未声明 session.resume 却放行 ResumeOf(未走 CheckStart 协商)。
	rt := newCFRuntime(cfOptions{acceptNoWorkspace: true})
	rep := runSingle(t, "capability_truth", rt, Options{Probes: fullProbes()})
	if rep.OK() {
		t.Fatal("未声明 session.resume 却接受 ResumeOf 必须被抓出")
	}
	if !failed(rep, "capability_truth") {
		t.Fatalf("capability_truth 场景应失败,得到: %s", rep.FailureText())
	}
}

func TestHarnessFailsWhenDeclaredCapabilityUnverifiable(t *testing.T) {
	// 声明 session.fork 但不提供 Fork 注入点:声明不可验证 = 能力声明不真实。
	rt := newCFRuntime(cfOptions{caps: []agentruntime.Capability{agentruntime.CapSessionFork}})
	rep := runSingle(t, "cap:"+string(agentruntime.CapSessionFork), rt, Options{Probes: Probes{LongRunInstruction: cfLongRun, CrashRun: cfCrashRun}})
	if rep.OK() {
		t.Fatal("声明 session.fork 而不提供验证探针必须失败")
	}
	if !failed(rep, "cap:"+string(agentruntime.CapSessionFork)) {
		t.Fatalf("cap:session.fork 场景应失败,得到: %s", rep.FailureText())
	}
}

func TestHarnessFailsWhenCrashProbesMissing(t *testing.T) {
	// crash 是 PRD §35 必测项:缺注入点 = 不可验证 = 失败;
	// cancel 的 in-flight 部分同理需要 LongRunInstruction。
	rt := newCFRuntime(cfOptions{})
	for _, name := range []string{"crash", "cancel"} {
		rep := runSingle(t, name, rt, Options{})
		if rep.OK() {
			t.Fatalf("%s 场景在缺少注入点时必须失败", name)
		}
		if !failed(rep, name) {
			t.Fatalf("%s 场景应失败,得到: %s", name, rep.FailureText())
		}
	}
}
