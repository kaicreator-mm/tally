package intakefreeze

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"tally/internal/intake"
	"tally/internal/intentevidence"
	"tally/internal/taskcontract"
)

// confirmedBase 返回一份能通过全链路校验的确认草案基线夹具:两个 MVP 任务
// 类型之一、非空摘要/目标、非空边界、至少一条验收、owner 确认保留的风险提示。
func confirmedBase() ConfirmedDraft {
	return ConfirmedDraft{
		TaskID:    "TASK-0042",
		Profile:   "standard",
		SourceRef: "GitHub Issue #142",
		Draft: intake.Draft{
			TaskType:  intake.TaskTypeBugFix,
			Summary:   "修复 logout 后 session 仍然有效的问题",
			Objective: "logout 必须使既有 session 立即失效",
			Scope: intake.Scope{
				AllowedPaths:   []string{"internal/session/**", "cmd/tallyhook/**"},
				ForbiddenPaths: []string{"internal/legacy/**"},
			},
			Acceptance: []intake.AcceptanceClause{
				{ID: "AC-01", Statement: "logout 后旧 session token 的请求返回 401"},
				{ID: "AC-02", Statement: "logout 后存储中不存在该 session 记录"},
			},
		},
		RiskHints: []string{"auth_flow", "session_storage"},
	}
}

// 确定性金测试(任务合同 AC):同一输入两次映射产出逐字段相等的结果;字段
// 一一对应;身份/修订只由内容决定;产物能被既有权威解析器原样重载出同一修订。
func TestMapHappyPathDeterministicGolden(t *testing.T) {
	in := confirmedBase()
	out1, err := Map(in)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	out2, err := Map(in)
	if err != nil {
		t.Fatalf("Map 第二次: %v", err)
	}
	if !reflect.DeepEqual(out1, out2) {
		t.Fatalf("同一输入两次映射结果不同:\n%+v\n%+v", out1, out2)
	}

	// FrozenIntent 侧:恰承载确认事实(任务身份/类型/来源/确认提示)。
	if out1.Intent.TaskID() != "TASK-0042" {
		t.Errorf("Intent.TaskID = %q", out1.Intent.TaskID())
	}
	if out1.Intent.TaskType() != intentevidence.TaskTypeBugFix {
		t.Errorf("Intent.TaskType = %q", out1.Intent.TaskType())
	}
	if out1.Intent.SourceRef() != "GitHub Issue #142" {
		t.Errorf("Intent.SourceRef = %q", out1.Intent.SourceRef())
	}
	if hints := out1.Intent.RiskHints(); !reflect.DeepEqual(hints, []string{"auth_flow", "session_storage"}) {
		t.Errorf("Intent.RiskHints = %v", hints)
	}
	if !strings.HasPrefix(out1.Intent.Revision(), "sha256:") {
		t.Errorf("Intent.Revision = %q,应为内容修订哈希", out1.Intent.Revision())
	}

	// TaskContract 侧:逐字段显式对应,无发明字段。
	task := out1.Task
	if task.ID != "TASK-0042" || task.Profile != "standard" {
		t.Errorf("Task.ID/Profile = %q/%q", task.ID, task.Profile)
	}
	if task.Title != in.Draft.Summary || task.Objective != in.Draft.Objective {
		t.Errorf("Task.Title/Objective = %q/%q", task.Title, task.Objective)
	}
	if !reflect.DeepEqual(task.Scope.AllowedPaths, []string{"internal/session/**", "cmd/tallyhook/**"}) ||
		!reflect.DeepEqual(task.Scope.ForbiddenPaths, []string{"internal/legacy/**"}) {
		t.Errorf("Task.Scope = %+v", task.Scope)
	}
	if len(task.Acceptance) != 2 {
		t.Fatalf("Task.Acceptance 条数 = %d", len(task.Acceptance))
	}
	for i, a := range task.Acceptance {
		want := in.Draft.Acceptance[i]
		if a.ID != want.ID || a.Statement != want.Statement {
			t.Errorf("Task.Acceptance[%d] = %+v,want id/statement 原样", i, a)
		}
		if a.IndependenceRequired != "" {
			t.Errorf("Task.Acceptance[%d].IndependenceRequired = %q,映射器不得发明治理属性", i, a.IndependenceRequired)
		}
		if task.Acceptance[i].Independence() != taskcontract.IndependenceNone {
			t.Errorf("未声明独立性应按 none(保守默认),got %q", task.Acceptance[i].Independence())
		}
	}
	// 草案没有的合同字段一律零值,不发明占位内容。
	if len(task.ProductRefs)+len(task.RequirementRefs)+len(task.ArchitectureRefs)+len(task.DependsOn)+len(task.DocumentationImpact) != 0 {
		t.Errorf("映射产物携带草案不存在的 refs/depends_on/documentation_impact: %+v", task)
	}
	// 哈希来自内容重算(taskcontract.Parse 同一纪律)。
	if task.Hash != task.RevisionHash() || !strings.HasPrefix(task.Hash, "sha256:") {
		t.Errorf("Task.Hash = %q,应为内容修订哈希", task.Hash)
	}

	// 与既有权威交叉金测试:产物经 YAML 往返后被 taskcontract.Parse 原样
	// 重载出同一内容修订(映射产物落进 .tally/tasks/ 制品族后可被既有装载
	// 路径消费 —— 生产可达性的纯函数面证据)。
	data, err := yaml.Marshal(task)
	if err != nil {
		t.Fatalf("YAML 编码: %v", err)
	}
	reloaded, err := taskcontract.Parse(data)
	if err != nil {
		t.Fatalf("taskcontract.Parse(映射产物): %v\n---\n%s", err, data)
	}
	if reloaded.Hash != task.Hash {
		t.Errorf("Parse(映射产物).Hash = %s,映射产物 Hash = %s,内容修订漂移", reloaded.Hash, task.Hash)
	}
}

// 排版变体(切片顺序)不改变身份/修订:同一内容的条目换位是排版,不是内容
// 差异(与 Freeze/RevisionHash 的规范化同一纪律)。
func TestMapCanonicalizesOrderingVariants(t *testing.T) {
	base, err := Map(confirmedBase())
	if err != nil {
		t.Fatalf("Map 基线: %v", err)
	}
	in := confirmedBase()
	in.RiskHints = []string{"session_storage", "auth_flow", "auth_flow"} // 换序 + 重复
	in.Draft.Scope.AllowedPaths = []string{"cmd/tallyhook/**", "internal/session/**"}
	in.Draft.Acceptance = []intake.AcceptanceClause{
		{ID: "AC-02", Statement: "logout 后存储中不存在该 session 记录"},
		{ID: "AC-01", Statement: "logout 后旧 session token 的请求返回 401"},
	}
	out, err := Map(in)
	if err != nil {
		t.Fatalf("Map 变体: %v", err)
	}
	if out.Intent.Revision() != base.Intent.Revision() {
		t.Errorf("提示换序/去重后 Intent.Revision 改变:%s != %s", out.Intent.Revision(), base.Intent.Revision())
	}
	if out.Task.Hash != base.Task.Hash {
		t.Errorf("scope/acceptance 换序后 Task.Hash 改变:%s != %s", out.Task.Hash, base.Task.Hash)
	}
}

// 只有 owner 确认保留的提示进入冻结证据;草案的候选风险信号只是 advisory,
// 绝不因确认而自动升级为证据(DL-06 / L3 pack「Do」)。
func TestMapFreezesOnlyConfirmedRiskHints(t *testing.T) {
	in := confirmedBase()
	in.Draft.RiskSignals = []string{"data_loss", "schema_migration"} // 推理候选,未被确认
	out, err := Map(in)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	hints := out.Intent.RiskHints()
	for _, candidate := range in.Draft.RiskSignals {
		for _, h := range hints {
			if h == candidate {
				t.Errorf("未确认的候选风险信号 %q 泄入冻结证据", candidate)
			}
		}
	}
	// 确认的提示以 intent:<task-id> basis 投影为风险信号候选(P0-01 语义原样)。
	for _, s := range out.Intent.RiskSignalCandidates() {
		if len(s.Basis) != 1 || s.Basis[0] != "intent:TASK-0042" {
			t.Errorf("风险信号候选 basis = %v,应为 [intent:TASK-0042]", s.Basis)
		}
	}
}

// 映射必须与直接调用 P0-01 冻结入口产出同一份证据:Draft→Facts 的翻译不引入
// 任何额外规范化或语义改写。
func TestMapIntentMatchesDirectFreeze(t *testing.T) {
	in := confirmedBase()
	out, err := Map(in)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	direct, err := intentevidence.Freeze(intentevidence.Facts{
		TaskID:    in.TaskID,
		TaskType:  intentevidence.TaskTypeBugFix,
		SourceRef: in.SourceRef,
		RiskHints: in.RiskHints,
	})
	if err != nil {
		t.Fatalf("intentevidence.Freeze: %v", err)
	}
	if !out.Intent.Equal(direct) || out.Intent.Revision() != direct.Revision() {
		t.Errorf("映射产物 Intent 与直接 Freeze 不一致:%+v vs %+v", out.Intent, direct)
	}
}

// 负路径:治理输入非法(profile 不是显式治理决策能兜底的形状)一律类型化拒绝。
func TestMapRejectsInvalidProfile(t *testing.T) {
	for _, p := range []string{"", "Standard", "hyper"} {
		in := confirmedBase()
		in.Profile = p
		if _, err := Map(in); !errors.Is(err, ErrInvalidProfile) {
			t.Errorf("profile %q: err = %v, want ErrInvalidProfile", p, err)
		}
	}
	// 首尾空白是排版(与 taskcontract.Parse normalize 同纪律),折叠后合法。
	in := confirmedBase()
	in.Profile = " strict "
	if _, err := Map(in); err != nil {
		t.Errorf("profile 首尾空白应按排版折叠: %v", err)
	}
}

// 负路径:任务类型不在 MVP 两个值内(含大小写变体与空)一律 fail-safe 拒绝,
// 不猜测归类(AC4 同款;映射层显式翻译是词表漂移的编译失败点)。
func TestMapRejectsUnsupportedTaskType(t *testing.T) {
	for _, tt := range []string{"", "BUG_FIX", "Bug_Fix", "bugfix", "refactor"} {
		in := confirmedBase()
		in.Draft.TaskType = intake.TaskType(tt)
		if _, err := Map(in); !errors.Is(err, ErrUnsupportedTaskType) {
			t.Errorf("task_type %q: err = %v, want ErrUnsupportedTaskType", tt, err)
		}
	}
}

// 负路径:non_scope / constraints 在冻结工程制品 schema 里没有归宿,任何条目
// (含纯空白条目)都必须先由确认路径解析,映射器绝不代 owner 决定丢弃。
func TestMapRejectsUnrepresentableDraftSemantics(t *testing.T) {
	mk := func(name string, vals []string) ConfirmedDraft {
		in := confirmedBase()
		switch name {
		case "non_scope":
			in.Draft.NonScope = vals
		case "constraints":
			in.Draft.Constraints = vals
		}
		return in
	}
	cases := []struct {
		name string
		vals []string
	}{
		{"一条实质内容", []string{"不改动导出 API"}},
		{"多条实质内容", []string{"兼容 Go 1.21", "不动数据库 schema"}},
		{"纯空白条目也算未解析", []string{"   "}},
	}
	for _, field := range []string{"non_scope", "constraints"} {
		for _, tc := range cases {
			in := mk(field, tc.vals)
			_, err := Map(in)
			if !errors.Is(err, ErrUnsupportedDraftSemantics) {
				t.Errorf("%s(%s): err = %v, want ErrUnsupportedDraftSemantics", field, tc.name, err)
				continue
			}
			if !strings.Contains(err.Error(), field) {
				t.Errorf("%s(%s): 错误信息应点名字段,got %q", field, tc.name, err.Error())
			}
		}
	}
}

// 负路径:身份/提示的 fail-closed 校验委托 P0-01 冻结入口,intentevidence 的
// 类型化哨兵必须可 errors.Is 透传(包装不吞类型)。
func TestMapSurfacesEvidenceValidationFailures(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ConfirmedDraft)
	}{
		{"缺少任务身份", func(c *ConfirmedDraft) { c.TaskID = "" }},
		{"任务身份含空白", func(c *ConfirmedDraft) { c.TaskID = "TASK 0042" }},
		{"确认提示含空白条目", func(c *ConfirmedDraft) { c.RiskHints = []string{"auth_flow", "   "} }},
	}
	for _, tc := range cases {
		in := confirmedBase()
		tc.mutate(&in)
		if _, err := Map(in); !errors.Is(err, intentevidence.ErrInvalidEvidence) {
			t.Errorf("%s: err = %v, want intentevidence.ErrInvalidEvidence", tc.name, err)
		}
	}
}

// 负路径:映射产物必须先通过既有权威的全部确定性校验(结构 + 静态准入形态,
// 含 scope 自相矛盾)才允许离开本包 —— invalid drafts fail before artifact commit。
func TestMapRejectsInvalidContractSemantics(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ConfirmedDraft)
	}{
		{"目标为空", func(c *ConfirmedDraft) { c.Draft.Objective = "   " }},
		{"允许边界为空", func(c *ConfirmedDraft) { c.Draft.Scope.AllowedPaths = nil }},
		{"验收条款 id 重复", func(c *ConfirmedDraft) {
			c.Draft.Acceptance[1].ID = "AC-01"
		}},
		{"条款 id 不符合 AC-<数字> 形态", func(c *ConfirmedDraft) {
			c.Draft.Acceptance[0].ID = "AC1"
		}},
		{"任务身份不符合 TASK-<数字> 形态", func(c *ConfirmedDraft) {
			c.TaskID = "TASK-0042-bis"
		}},
		{"scope 自相矛盾", func(c *ConfirmedDraft) {
			c.Draft.Scope.ForbiddenPaths = []string{"internal/session"}
		}},
	}
	for _, tc := range cases {
		in := confirmedBase()
		tc.mutate(&in)
		if _, err := Map(in); !errors.Is(err, ErrInvalidContract) {
			t.Errorf("%s: err = %v, want ErrInvalidContract", tc.name, err)
		}
	}
}

// 输入隔离:Map 不得修改确认草案,也不得保留其切片引用 —— 映射之后再改动
// 输入,产物(含内容哈希)纹丝不动。
func TestMapDoesNotMutateInput(t *testing.T) {
	in := confirmedBase()
	snapshot := confirmedBase()
	out, err := Map(in)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	wantHash, wantRevision := out.Task.Hash, out.Intent.Revision()

	// 先证明 Map 没有动输入草案本身。
	if !reflect.DeepEqual(in.Draft, snapshot.Draft) {
		t.Fatalf("Map 修改了输入草案本身")
	}

	// 再就地篡改输入的每一个切片字段,证明产物不随之漂移。
	in.Draft.Scope.AllowedPaths[0] = "hijacked/**"
	in.Draft.Scope.ForbiddenPaths[0] = "hijacked/**"
	in.Draft.Acceptance[0].Statement = "hijacked"
	in.RiskHints[0] = "hijacked"

	if out.Task.Hash != wantHash || out.Intent.Revision() != wantRevision {
		t.Fatalf("产物在输入被篡改后漂移: hash/revision = %s/%s", out.Task.Hash, out.Intent.Revision())
	}
	if out.Task.Scope.AllowedPaths[0] != "internal/session/**" {
		t.Errorf("产物 scope 与输入切片共享底层数组:%q", out.Task.Scope.AllowedPaths[0])
	}
	if out.Task.Acceptance[0].Statement != snapshot.Draft.Acceptance[0].Statement {
		t.Errorf("产物 acceptance 与输入切片共享底层数组:%q", out.Task.Acceptance[0].Statement)
	}
	if out.Intent.RiskHints()[0] != "auth_flow" {
		t.Errorf("产物提示与输入切片共享底层数组:%q", out.Intent.RiskHints()[0])
	}
}

// 负路径补充:校验顺序固定(确定性失败)——治理输入与身份同时非法时,
// 报最前面的 profile 失败。
func TestMapFailureOrderIsDeterministic(t *testing.T) {
	in := confirmedBase()
	in.Profile = "hyper"
	in.TaskID = ""
	in.Draft.TaskType = "refactor"
	if _, err := Map(in); !errors.Is(err, ErrInvalidProfile) {
		t.Errorf("校验顺序应固定且 profile 最先: err = %v", err)
	}
}
