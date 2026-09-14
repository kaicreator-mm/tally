package intake

import (
	"context"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tally/internal/airuntime"
)

// succeededResult 构造一个 SUCCEEDED 的 Runtime 结果(airuntime.FakeExecutor 预设)。
func succeededResult(out string) airuntime.Result {
	return airuntime.Result{
		OperationID:       "op-test",
		Status:            airuntime.StatusSucceeded,
		TerminationReason: airuntime.TerminationCompleted,
		Output:            json.RawMessage(out),
	}
}

// validBugDraft 是一份合法的 bug_fix 草案输出。
const validBugDraft = `{
  "task_type": "bug_fix",
  "summary": "修复 logout 后 session 仍有效的问题",
  "objective": "logout 之后旧凭证必须立即失效",
  "scope": {"allowed_paths": ["internal/auth"], "forbidden_paths": ["cmd/"]},
  "non_scope": ["不引入新的存储"],
  "constraints": ["保持键格式兼容"],
  "acceptance": [
    {"id": "AC-01", "statement": "WHEN 用户 logout,系统 shall 使其全部会话立即失效"},
    {"id": "AC-02", "statement": "logout 后用旧令牌访问受保护接口返回 401"}
  ],
  "risk_signals": ["涉及鉴权路径"]
}`

// 正例(AC1/AC4):一句意图(零手工结构化字段)→ 完整 bug_fix 草案;
// 送出的 Request 过域侧闸且形状完整(DECISION_LOCKS #17 / ADR-0002)。
func TestProposeDraftBugFix(t *testing.T) {
	fake := &airuntime.FakeExecutor{Result: succeededResult(validBugDraft)}
	p, err := Propose(context.Background(), fake, Intent{Text: "修复 logout 后 session 仍然有效的问题"})
	if err != nil {
		t.Fatalf("Propose 出错: %v", err)
	}
	dp, ok := p.(DraftProposal)
	if !ok {
		t.Fatalf("应返回 DraftProposal, got %T", p)
	}
	d := dp.Draft
	if d.TaskType != TaskTypeBugFix {
		t.Errorf("task_type = %q, want %q", d.TaskType, TaskTypeBugFix)
	}
	if d.Summary == "" || d.Objective == "" {
		t.Errorf("summary/objective 不得为空: %q / %q", d.Summary, d.Objective)
	}
	if len(d.Scope.AllowedPaths) == 0 || d.Scope.AllowedPaths[0] != "internal/auth" {
		t.Errorf("scope.allowed_paths 映射错误: %v", d.Scope.AllowedPaths)
	}
	if len(d.Acceptance) != 2 || d.Acceptance[0].ID != "AC-01" || d.Acceptance[1].ID != "AC-02" {
		t.Errorf("验收条款映射错误: %+v", d.Acceptance)
	}
	if len(d.RiskSignals) != 1 {
		t.Errorf("风险信号映射错误: %v", d.RiskSignals)
	}

	reqs := fake.Requests()
	if len(reqs) != 1 {
		t.Fatalf("应恰好发出一次推理调用, got %d", len(reqs))
	}
	req := reqs[0]
	if req.Operation != ProposeOperation {
		t.Errorf("operation 应为稳定 ID %q, got %q", ProposeOperation, req.Operation)
	}
	if req.Intent.Role == "" || req.Intent.Instructions == "" {
		t.Errorf("intent 必须显式声明角色与用途: %+v", req.Intent)
	}
	if req.Budget.MaxAttempts != 1 || req.Budget.MaxTokens <= 0 || req.Budget.DeadlineMs <= 0 || req.Budget.MaxCostCeilingUSD < 0 {
		t.Errorf("预算声明不完整: %+v", req.Budget)
	}
	if req.IdempotencyKey == "" || len(req.IdempotencyKey) > 200 {
		t.Errorf("幂等键非法: %q(len=%d)", req.IdempotencyKey, len(req.IdempotencyKey))
	}
	if err := req.Validate(); err != nil {
		t.Errorf("域构造的 Request 应过域侧闸: %v", err)
	}
	if !strings.Contains(req.Input.Prompt, "logout") {
		t.Errorf("提示词应含意图原文: %q", req.Input.Prompt)
	}
	for _, want := range []string{"bug_fix", "db_migration", "绝不发明验收"} {
		if !strings.Contains(req.Input.Prompt, want) {
			t.Errorf("提示词应声明规则 %q: %q", want, req.Input.Prompt)
		}
	}
	if !json.Valid(req.Input.OutputShape) || !strings.Contains(string(req.Input.OutputShape), "task_type") {
		t.Errorf("OutputShape 应是含 task_type 的合法 JSON: %s", req.Input.OutputShape)
	}
}

// 正例(AC1/AC4):迁移意图 → db_migration 草案;source_ref 进提示词。
func TestProposeDraftMigrationWithSourceRef(t *testing.T) {
	fake := &airuntime.FakeExecutor{Result: succeededResult(`{
  "task_type": "db_migration",
  "summary": "给 users 表加 phone 列",
  "objective": "users 支持手机号",
  "scope": {"allowed_paths": ["migrations/", "internal/store"]},
  "acceptance": [{"id": "AC-01", "statement": "迁移在全新库上可重复执行且数据完整"}]
}`)}
	p, err := Propose(context.Background(), fake, Intent{Text: "给 users 表加手机号列", SourceRef: "GitHub Issue #142"})
	if err != nil {
		t.Fatalf("Propose 出错: %v", err)
	}
	dp, ok := p.(DraftProposal)
	if !ok {
		t.Fatalf("应返回 DraftProposal, got %T", p)
	}
	if dp.Draft.TaskType != TaskTypeMigration {
		t.Errorf("task_type = %q, want %q", dp.Draft.TaskType, TaskTypeMigration)
	}
	prompt := fake.Requests()[0].Input.Prompt
	if !strings.Contains(prompt, "GitHub Issue #142") {
		t.Errorf("提示词应含来源引用: %q", prompt)
	}
}

// 正例(AC3):意图含糊 → 纯澄清请求,绝不夹带验收。
func TestProposeClarificationOnly(t *testing.T) {
	fake := &airuntime.FakeExecutor{Result: succeededResult(
		`{"clarifications":[{"question":"缺陷在哪个登录入口?","candidates":["Web","移动端"]},{"question":"复现步骤是什么?"}]}`)}
	p, err := Propose(context.Background(), fake, Intent{Text: "登录有问题,修一下"})
	if err != nil {
		t.Fatalf("Propose 出错: %v", err)
	}
	cp, ok := p.(ClarificationProposal)
	if !ok {
		t.Fatalf("应返回 ClarificationProposal, got %T", p)
	}
	if len(cp.Questions) != 2 {
		t.Fatalf("问题数 = %d, want 2", len(cp.Questions))
	}
	if len(cp.Questions[0].Candidates) != 2 {
		t.Errorf("问题应带候选答案: %+v", cp.Questions[0])
	}
	if cp.Questions[1].Candidates != nil {
		t.Errorf("开放问题候选可为空: %+v", cp.Questions[1])
	}
}

// 负例(AC3 fail safe):一边给澄清一边给草案/验收 → 拒绝,不猜测。
func TestProposeRejectsClarificationWithDraft(t *testing.T) {
	fake := &airuntime.FakeExecutor{Result: succeededResult(`{
  "task_type": "bug_fix",
  "summary": "s",
  "objective": "o",
  "scope": {"allowed_paths": ["a"]},
  "acceptance": [{"id": "AC-01", "statement": "x"}],
  "clarifications": [{"question": "到底哪个入口?"}]
}`)}
	_, err := Propose(context.Background(), fake, Intent{Text: "登录有问题"})
	if !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("澄清与草案混出应 ErrInvalidProposal, got %v", err)
	}
}

// 负例(AC4 fail safe):任务类型越界(含缺类)→ ErrUnsupportedTaskType,
// 不静默降级、不发明支持的类型。
func TestProposeRejectsUnsupportedTaskType(t *testing.T) {
	cases := []struct {
		name string
		out  string
	}{
		{"越界类型", `{"task_type":"refactor","summary":"s","objective":"o","scope":{"allowed_paths":["a"]},"acceptance":[{"id":"AC-01","statement":"x"}]}`},
		{"缺失类型", `{"summary":"s","objective":"o","scope":{"allowed_paths":["a"]},"acceptance":[{"id":"AC-01","statement":"x"}]}`},
		{"空澄清视为草案分支", `{"clarifications":[]}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fake := &airuntime.FakeExecutor{Result: succeededResult(c.out)}
			_, err := Propose(context.Background(), fake, Intent{Text: "重构一下模块"})
			if !errors.Is(err, ErrUnsupportedTaskType) {
				t.Fatalf("应 ErrUnsupportedTaskType, got %v", err)
			}
		})
	}
}

// 负例(fail safe):形状非法的输出一律 ErrInvalidProposal —— 缺项、空白条款、
// 重复 id、未知字段、尾随内容、非法 JSON,绝不 best-effort 猜测。
func TestProposeRejectsInvalidShapes(t *testing.T) {
	base := map[string]string{
		"task_type":  `"db_migration"`,
		"summary":    `"s"`,
		"objective":  `"o"`,
		"scope":      `{"allowed_paths":["a"]}`,
		"acceptance": `[{"id":"AC-01","statement":"x"}]`,
	}
	build := func(overrides map[string]string) string {
		fields := map[string]string{}
		for k, v := range base {
			fields[k] = v
		}
		for k, v := range overrides {
			if v == "" {
				delete(fields, k)
				continue
			}
			fields[k] = v
		}
		var parts []string
		for k, v := range fields {
			parts = append(parts, `"`+k+`":`+v)
		}
		return "{" + strings.Join(parts, ",") + "}"
	}
	cases := []struct {
		name string
		out  string
	}{
		{"缺 summary", build(map[string]string{"summary": ""})},
		{"缺 objective", build(map[string]string{"objective": ""})},
		{"allowed_paths 为空", build(map[string]string{"scope": `{"allowed_paths":[]}`})},
		{"allowed_paths 含空白条目", build(map[string]string{"scope": `{"allowed_paths":["a","  "]}`})},
		{"forbidden_paths 含空白条目", build(map[string]string{"scope": `{"allowed_paths":["a"],"forbidden_paths":[" "]}`})},
		{"验收为空", build(map[string]string{"acceptance": `[]`})},
		{"验收 id 重复", build(map[string]string{"acceptance": `[{"id":"AC-01","statement":"x"},{"id":"AC-01","statement":"y"}]`})},
		{"验收缺 statement", build(map[string]string{"acceptance": `[{"id":"AC-01"}]`})},
		{"non_scope 含空白条目", build(map[string]string{"non_scope": `[" "]`})},
		{"constraints 含空白条目", build(map[string]string{"constraints": `[""]`})},
		{"risk_signals 含空白条目", build(map[string]string{"risk_signals": `["  "]`})},
		{"澄清缺 question", `{"clarifications":[{"candidates":["a"]}]}`},
		{"澄清候选空白", `{"clarifications":[{"question":"q","candidates":["a"," "]}]}`},
		{"未知字段", `{"task_type":"bug_fix","unexpected":1}`},
		{"尾随内容", validBugDraft + " trailing"},
		{"非法 JSON", `{not json`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fake := &airuntime.FakeExecutor{Result: succeededResult(c.out)}
			_, err := Propose(context.Background(), fake, Intent{Text: "修 bug"})
			if !errors.Is(err, ErrInvalidProposal) {
				t.Fatalf("应 ErrInvalidProposal, got %v", err)
			}
		})
	}
}

// 负例:Runtime 终态逐一映射到可区分的类型化错误,不折叠成一种"失败"
// (信任不变量:FAIL 与 ERROR/拿不准端到端可区分)。
func TestProposeStatusMapping(t *testing.T) {
	cases := []struct {
		name   string
		status airuntime.Status
		want   error
	}{
		{"uncertain", airuntime.StatusUncertain, ErrProposeUncertain},
		{"cancelled", airuntime.StatusCancelled, ErrProposeCancelled},
		{"failed", airuntime.StatusFailed, ErrProposeFailed},
		{"未知终态按拿不准", airuntime.Status("WEIRD"), ErrProposeUncertain},
	}
	all := []error{ErrProposeUncertain, ErrProposeCancelled, ErrProposeFailed}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fake := &airuntime.FakeExecutor{Result: airuntime.Result{
				OperationID:       "op-1",
				Status:            c.status,
				TerminationReason: airuntime.TerminationProviderFailure,
			}}
			_, err := Propose(context.Background(), fake, Intent{Text: "修 bug"})
			if !errors.Is(err, c.want) {
				t.Fatalf("status=%s 应归 %v, got %v", c.status, c.want, err)
			}
			for _, other := range all {
				if other != c.want && errors.Is(err, other) {
					t.Errorf("status=%s 误命中 %v:终态必须可区分", c.status, other)
				}
			}
		})
	}
}

// 负例:传输层失败(error 返回,无 Operation 结果)如实上抛。
func TestProposeTransportError(t *testing.T) {
	fake := &airuntime.FakeExecutor{Err: errors.New("网络不可达")}
	if _, err := Propose(context.Background(), fake, Intent{Text: "修 bug"}); err == nil {
		t.Fatal("传输层失败应返回错误")
	}
}

// 负例:SUCCEEDED 但输出为空 → 明确报错,不返回空草案假装成功。
func TestProposeSucceededWithoutOutput(t *testing.T) {
	fake := &airuntime.FakeExecutor{Result: airuntime.Result{Status: airuntime.StatusSucceeded}}
	if _, err := Propose(context.Background(), fake, Intent{Text: "修 bug"}); err == nil {
		t.Fatal("SUCCEEDED 但无输出应报错")
	}
}

// 负例(AC1 的唯一必填项):意图为空/超长直接拒绝,不发推理调用。
func TestProposeValidatesIntent(t *testing.T) {
	fake := &airuntime.FakeExecutor{Result: succeededResult(validBugDraft)}
	if _, err := Propose(context.Background(), fake, Intent{Text: "   "}); !errors.Is(err, ErrEmptyIntent) {
		t.Fatalf("空白意图应 ErrEmptyIntent, got %v", err)
	}
	if _, err := Propose(context.Background(), fake, Intent{Text: strings.Repeat("x", maxIntentLen+1)}); !errors.Is(err, ErrIntentTooLong) {
		t.Fatalf("超长意图应 ErrIntentTooLong, got %v", err)
	}
	if n := len(fake.Requests()); n != 0 {
		t.Errorf("被拒输入不得到达传输, got %d 次调用", n)
	}
	if _, err := Propose(context.Background(), nil, Intent{Text: "修 bug"}); !errors.Is(err, ErrNoExecutor) {
		t.Fatalf("未注入 Executor 应 ErrNoExecutor, got %v", err)
	}
}

// 幂等键稳定性(契约 Idempotency-Key):同一意图(含来源)复用同一键;
// 意图或来源一变,键随之变(新逻辑调用)。
func TestProposeIdempotencyKeyStable(t *testing.T) {
	fake := &airuntime.FakeExecutor{Result: succeededResult(validBugDraft)}
	in := Intent{Text: "修复 X", SourceRef: "Jira ABC-1"}
	for i := 0; i < 2; i++ {
		if _, err := Propose(context.Background(), fake, in); err != nil {
			t.Fatalf("Propose 第 %d 次: %v", i+1, err)
		}
	}
	reqs := fake.Requests()
	if reqs[0].IdempotencyKey == "" || reqs[0].IdempotencyKey != reqs[1].IdempotencyKey {
		t.Errorf("同一意图应复用同一幂等键, got %q vs %q", reqs[0].IdempotencyKey, reqs[1].IdempotencyKey)
	}

	changed := in
	changed.Text = "修复 Y"
	if _, err := Propose(context.Background(), fake, changed); err != nil {
		t.Fatalf("Propose 意图变化后: %v", err)
	}
	if got := fake.Requests()[2].IdempotencyKey; got == reqs[0].IdempotencyKey {
		t.Errorf("意图变化应产生新幂等键, got %q", got)
	}

	refChanged := in
	refChanged.SourceRef = "Jira ABC-2"
	if _, err := Propose(context.Background(), fake, refChanged); err != nil {
		t.Fatalf("Propose 来源变化后: %v", err)
	}
	if got := fake.Requests()[3].IdempotencyKey; got == reqs[0].IdempotencyKey {
		t.Errorf("来源变化应产生新幂等键, got %q", got)
	}
}

// 脱敏闸:意图里混进疑似密钥 → 本地闸拒绝,内容不到达传输。
func TestProposeRedactedIntentRejected(t *testing.T) {
	fake := &airuntime.FakeExecutor{Result: succeededResult(validBugDraft)}
	_, err := Propose(context.Background(), fake, Intent{Text: "把密钥 sk-abcdefghijklmnopqrst 换掉"})
	if err == nil {
		t.Fatal("疑似密钥应被拒绝")
	}
	if n := len(fake.Requests()); n != 0 {
		t.Errorf("被拒内容不得到达传输, got %d 次调用", n)
	}
}

// AC5 机器钉子(仿 test/arch 的消费域包扫描):本包声明面(类型/函数/变量/
// 字段/struct tag)不得出现 provider/model/tier/凭证/供应商名 —— 路由属于
// 公共 AI Runtime,域只声明 operation/intent/input/budget + 幂等键。
func TestNoForbiddenVocabularyInIntake(t *testing.T) {
	forbidden := map[string]string{
		"provider":   "provider 路由不得进入 intake:属于公共 AI Runtime(AC5)",
		"model":      "模型选择不得进入 intake:属于公共 AI Runtime(AC5)",
		"tier":       "tier 不得进入 intake:Runtime 从 rollout/policy 决定(AC5)",
		"baseurl":    "端点配置不得进入 intake:属于传输层(AC5)",
		"apikey":     "凭证不得进入 intake:凭证属于传输层(AC5)",
		"api_key":    "凭证不得进入 intake:凭证属于传输层(AC5)",
		"openai":     "供应商名不得进入 intake(AC5)",
		"anthropic":  "供应商名不得进入 intake(AC5)",
		"httpclient": "legacy llm 客户端类型不得回到 intake:一律 airuntime.Executor",
		"proposer":   "legacy llm.Proposer 不得回到 intake:一律 airuntime.Executor",
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("读包目录: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, filepath.Join(".", e.Name()), nil, 0)
		if err != nil {
			t.Fatalf("解析 %s: %v", e.Name(), err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			var ids []string
			switch decl := n.(type) {
			case *ast.TypeSpec:
				ids = append(ids, decl.Name.Name)
			case *ast.FuncDecl:
				ids = append(ids, decl.Name.Name)
			case *ast.ValueSpec:
				for _, name := range decl.Names {
					ids = append(ids, name.Name)
				}
			case *ast.Field:
				for _, name := range decl.Names {
					ids = append(ids, name.Name)
				}
				if decl.Tag != nil {
					ids = append(ids, strings.Trim(decl.Tag.Value, "`\""))
				}
			}
			for _, id := range ids {
				lower := strings.ToLower(id)
				for frag, msg := range forbidden {
					if strings.Contains(lower, frag) {
						t.Errorf("%s 标识符/标签 %q 命中禁词: %s", e.Name(), id, msg)
					}
				}
			}
			return true
		})
	}
}

// 草案类型面上没有任何判定/冻结字段(AC2): Proposal 永远是 DRAFT,
// 冻结归既有合同路径(INV-02),本包类型面不允许出现权威语义。
func TestDraftHasNoAuthorityFields(t *testing.T) {
	forbidden := map[string]bool{
		"verified": true, "frozen": true, "approved": true, "accepted": true,
		"status": true, "verdict": true, "passed": true, "valid": true,
	}
	var check func(rt reflect.Type)
	check = func(rt reflect.Type) {
		switch rt.Kind() {
		case reflect.Slice, reflect.Ptr, reflect.Array:
			check(rt.Elem())
			return
		case reflect.Struct:
			for i := 0; i < rt.NumField(); i++ {
				f := rt.Field(i)
				name := strings.ToLower(strings.Split(f.Tag.Get("json"), ",")[0])
				if forbidden[name] {
					t.Errorf("草案结构出现判定/冻结字段 %q(AC2:提案在冻结前不作数)", f.Name)
				}
				check(f.Type)
			}
		}
	}
	check(reflect.TypeOf(Draft{}))
	check(reflect.TypeOf(Clarification{}))
}
