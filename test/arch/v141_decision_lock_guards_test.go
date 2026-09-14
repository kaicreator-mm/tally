package arch

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"tally/internal/engineeringpolicy"
	"tally/internal/outcome"
)

// A0-04 / DECISION_LOCKS「Machine enforcement index」:v1.4.1 特有读法的机器
// 守卫(检测意外的重复权威)。本文件不放宽任何既有 v14 守卫,只新增 v1.4.1
// 冻结读法(DL-02 与绑定解释 BI-01/BI-02)的可执行落点:
//
//	v141-DL02 只有一套核心 Coding Harness 协议(DL-02 / A-01):agentruntime
//	         的 Runtime/Session 接口是唯一的会话生命周期协议;被否决的
//	         ExecutionBackend 通用抽象不得复辟;别的包不得用另一套词汇复制
//	         「Descriptor()+Start」的协议开头形状(引用 agentruntime.* 的
//	         消费方窄化接口不在此列 —— 那是消费同一协议,不是第二个);
//	v141-BI01 (DL-08/A-04,绑定 PRD §17)engineeringpolicy 是运行时规则
//	         schema:治理元数据字段(status/version/policy_id/evidence_refs/
//	         created_from_cases/approved_by/approved_at/supersedes/scope)与
//	         治理生命周期词表(DRAFT/PROPOSED/ACTIVE/SUPERSEDED/REJECTED)
//	         不得进入该包的 struct/yaml/常量面;解析器行为面拒绝治理字段;
//	         数据库不得出现把策略声明为 runtime-active 的标志列 —— activation
//	         只能是 gitwriter 对 canonical YAML 的提交;
//	v141-BI02 (DL-12/A-06,绑定 PRD §27)Outcome kind 集合恰为五类:0023 的
//	         CHECK 枚举与 internal/outcome 的常量两侧都不得多出任何值;PRD §27
//	         的 workflow/integration 标签(rejected/escalated/integration_failed)
//	         与三个原始拼写(human_corrected/hotfix_required/regression_detected)
//	         不是 kind,解析必须 fail-safe 拒绝、绝不折叠;event_kind 列只属于
//	         outcome_event 一张表(没有平行的 case-history 事件库)。
//
// 无法自动化、仍由 review gate 把守的部分(任务合同实现要求 4):
//   - DL-02 的组合根事实:closedloop.Executor 适配器确实经 agentruntime.Runtime
//     路由、一次尝试一个会话(cmd/worker/v14.go)—— 接线级事实,静态 AST 面
//     只能守协议声明唯一性,接线正确性归 runtime conformance(P1-25 惯例);
//   - BI-01 的语义面:「runtime 只消费 ACTIVE」=「精确消费钉定提交里
//     .tally/policies/*.yaml 的 blob(artifacts.LoadPolicies)」这条数据流、
//     治理事实与运行时 schema 的分离落点(P3 lane)、以及「activation 是
//     gitwriter 提交」的流程不变量 —— P3 落地前只能靠 review;本文件先钉住
//     其反面(治理字段/治理词表/activation 标志列不得出现);
//   - BI-02 的完全面:平行事件库若刻意避开 event_kind 列名(换个词)则静态
//     扫描不可见 —— 列名唯一性是机器可守的下界,其余靠 review。

// ---- 共享夹具 -----------------------------------------------------------------

// v141RuntimeProtocolRoots 是协议唯一性守卫的扫描范围:控制面(internal)、
// 适配器(runtimes)与组合根(cmd)。只看非测试文件(生产边界)。
var v141RuntimeProtocolRoots = []string{
	repoInternal,
	filepath.Join("..", "..", "runtimes"),
	filepath.Join("..", "..", "cmd"),
}

// v141GoFile 是带路径的已解析生产源文件(违规信息要能指到具体文件)。
type v141GoFile struct {
	path string
	file *ast.File
}

// v141ParseRoots 解析 roots 下全部非测试 .go 文件为完整 AST(带路径)。
func v141ParseRoots(t *testing.T, roots ...string) []v141GoFile {
	t.Helper()
	var out []v141GoFile
	for _, root := range roots {
		for _, path := range goFiles(t, root, false) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatalf("解析 %s: %v", path, err)
			}
			out = append(out, v141GoFile{path: path, file: f})
		}
	}
	if len(out) == 0 {
		t.Fatalf("扫描范围 %v 下没有非测试 .go 文件(夹具失真)", roots)
	}
	return out
}

// ---- v141-DL02:只有一套核心 runtime 协议 --------------------------------------

// v141-1(DL-02/A-01):协议唯一。三重落点:
//
//	(a) ExecutionBackend(A-01 已否决的通用执行后端抽象)不得以任何形态复辟;
//	(b) 导出接口 Runtime/Session 只属于 internal/agentruntime(runtimes/codex
//	    的 Runtime、store 的 Session 等具体类型是适配器/行模型,不是协议声明);
//	(c) 别的包不得在完全不引用 agentruntime 词汇的前提下复制协议开头形状
//	    (零参 Descriptor + 带参 Start)—— 那是用另一套词汇造的第二协议。
func TestV141SingleCoreRuntimeProtocol(t *testing.T) {
	agentruntimePrefix := filepath.Join(repoInternal, "agentruntime") + string(filepath.Separator)
	inAgentruntime := func(path string) bool { return strings.HasPrefix(path, agentruntimePrefix) }

	for _, gf := range v141ParseRoots(t, v141RuntimeProtocolRoots...) {
		for _, decl := range gf.file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				// (a) 被否决的抽象不得复辟(接口或具体类型都算)。
				if ts.Name.Name == "ExecutionBackend" {
					t.Errorf("%s 声明 %s:A-01 已否决通用 ExecutionBackend 抽象,协议唯一入口是 agentruntime.Runtime(DL-02)", gf.path, ts.Name.Name)
				}
				iface, ok := ts.Type.(*ast.InterfaceType)
				if !ok || inAgentruntime(gf.path) {
					continue // 非接口声明不是协议;协议本体所在的包自外于本守卫
				}
				// (b) 协议名唯一。
				if ast.IsExported(ts.Name.Name) && (ts.Name.Name == "Runtime" || ts.Name.Name == "Session") {
					t.Errorf("%s 声明接口 %s:核心会话协议唯一(DL-02),接口名 Runtime/Session 只属于 internal/agentruntime", gf.path, ts.Name.Name)
				}
				// (c) 协议形状唯一(换名不换形也算第二协议)。
				if ast.IsExported(ts.Name.Name) && v141ProtocolShapeCopy(iface) {
					t.Errorf("%s 的接口 %s 用另一套词汇复制了协议开头形状(Descriptor+Start)(DL-02:第二核心协议)", gf.path, ts.Name.Name)
				}
			}
		}
	}
}

// v141ProtocolShapeCopy 报告接口是否在完全不引用 agentruntime 词汇的前提下
// 复制了协议开头形状(PRD §10.2:Descriptor 自述 + Start 开会话)。引用
// agentruntime.* 的窄化接口是消费方视角的同一协议,不算第二协议。
func v141ProtocolShapeCopy(iface *ast.InterfaceType) bool {
	var hasDescriptor, hasStart bool
	for _, field := range iface.Methods.List {
		if len(field.Names) != 1 {
			continue // 内嵌接口不构成方法形状
		}
		ft, ok := field.Type.(*ast.FuncType)
		if !ok {
			continue
		}
		switch field.Names[0].Name {
		case "Descriptor":
			hasDescriptor = ft.Params == nil || len(ft.Params.List) == 0
		case "Start":
			hasStart = ft.Params != nil && len(ft.Params.List) > 0
		}
	}
	if !hasDescriptor || !hasStart {
		return false
	}
	refsAgentruntime := false
	ast.Inspect(iface, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "agentruntime" {
				refsAgentruntime = true
			}
		}
		return true
	})
	return !refsAgentruntime
}

// ---- v141-BI01:engineeringpolicy 排除治理字段 ----------------------------------

// v141GovernanceYAMLFields 是 PRD §17 / BI-01 的治理元数据字段词汇(yaml 名)。
var v141GovernanceYAMLFields = []string{
	"status", "version", "policy_id",
	"evidence_refs", "created_from_cases",
	"approved_by", "approved_at", "supersedes", "scope",
}

// v141GovernanceLifecycleValues 是治理生命周期词表(BI-01);值比较不区分
// 大小写 —— 治理状态不会因为写成小写就变成运行时 schema 的一部分。
var v141GovernanceLifecycleValues = []string{"draft", "proposed", "active", "superseded", "rejected"}

// v141YAMLKey 提取结构体字段 tag 的 yaml 键名(无 tag / "-" / inline 返回空)。
func v141YAMLKey(tag *ast.BasicLit) string {
	if tag == nil || tag.Kind != token.STRING {
		return ""
	}
	key := reflect.StructTag(strings.Trim(tag.Value, "`")).Get("yaml")
	name := strings.SplitN(key, ",", 2)[0]
	if name == "" || name == "-" {
		return ""
	}
	return name
}

// v141-2a(BI-01/DL-08 结构面):engineeringpolicy 包的任何 struct 字段与
// yaml 键不得携带治理词汇,常量面不得出现治理生命周期词表 —— 治理元数据
// 归独立治理事实(P3 lane),运行时规则 schema 保持窄(PRD §8 + §17 经 BI-01)。
func TestV141PolicySchemaExcludesGovernanceFields(t *testing.T) {
	files := v14ParsePackage(t, filepath.Join(repoInternal, "engineeringpolicy"))
	policyFound := false
	for _, f := range files {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			if gd.Tok == token.TYPE {
				for _, spec := range gd.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if ts.Name.Name == "Policy" {
						policyFound = true
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						continue
					}
					for _, field := range st.Fields.List {
						for _, name := range field.Names {
							lower := strings.ToLower(name.Name)
							for _, vocab := range v141GovernanceYAMLFields {
								if strings.Contains(lower, strings.ReplaceAll(vocab, "_", "")) {
									t.Errorf("engineeringpolicy.%s 的字段 %s 命中治理词汇 %q(BI-01/DL-08:治理元数据不得进入运行时规则 schema)", ts.Name.Name, name.Name, vocab)
								}
							}
						}
						if key := v141YAMLKey(field.Tag); key != "" {
							for _, vocab := range v141GovernanceYAMLFields {
								if strings.Contains(strings.ToLower(key), vocab) {
									t.Errorf("engineeringpolicy.%s 的 yaml 键 %q 命中治理词汇 %q(BI-01/DL-08)", ts.Name.Name, key, vocab)
								}
							}
						}
					}
				}
			}
			if gd.Tok == token.CONST {
				for _, spec := range gd.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok || len(vs.Names) == 0 {
						continue
					}
					for _, val := range vs.Values {
						bl, ok := val.(*ast.BasicLit)
						if !ok || bl.Kind != token.STRING {
							continue
						}
						unquoted := strings.ToLower(strings.Trim(bl.Value, "`\""))
						for _, life := range v141GovernanceLifecycleValues {
							if unquoted == life {
								t.Errorf("engineeringpolicy 常量 %s = %s:治理生命周期值 %q 不得进入运行时策略包(BI-01/DL-08)", vs.Names[0].Name, bl.Value, life)
							}
						}
					}
				}
			}
		}
	}
	if !policyFound {
		t.Fatal("engineeringpolicy 下找不到 Policy 类型(夹具失真)")
	}
}

// v141-2b(BI-01/DL-08 行为面,负例):解析器必须拒绝一切治理字段。严格模式
// (KnownFields)当前按未知字段拒绝;本守卫把 BI-01 点名的字段钉成显式期望,
// 防止未来有人给 schema 加上这些名字让治理元数据「变合法」地流进运行时策略。
func TestV141PolicyParseRejectsGovernanceFields(t *testing.T) {
	// 每个字段配一份「其余部分合法」的 gate 策略骨架,单独注入被禁字段。
	inject := func(field string) string {
		return `
id: policy-a004-governance-probe
enforcement: gate
rationale: probe
trigger:
  risk_signals:
    - schema_migration
requires:
  - fresh_database
gate_condition:
  all_of:
    - fresh_database
` + field + "\n"
	}
	fields := []string{
		"status: ACTIVE",
		"status: draft",
		"version: 2",
		"policy_id: policy-123",
		"evidence_refs: [case-1]",
		"created_from_cases: [case-9]",
		"approved_by: [alice]",
		"approved_at: 2026-01-01T00:00:00Z",
		"supersedes: [policy-old]",
		"scope: [repo]",
	}
	for _, field := range fields {
		if _, err := engineeringpolicy.Parse([]byte(inject(field))); !errors.Is(err, engineeringpolicy.ErrInvalidPolicy) {
			t.Errorf("策略 YAML 携带治理字段 %q 未被拒绝: err = %v(BI-01/DL-08:治理元数据不得进入运行时策略)", field, err)
		}
	}
}

// v141ActivationColumnRe 是「把策略声明为 runtime-active」的标志列词表
// (BI-01:activation 只能是 gitwriter 提交 canonical YAML,数据库不持有开关)。
var v141ActivationColumnRe = regexp.MustCompile(`(?i)\b(is_active|runtime_active|is_runtime_active|activation|active)\b`)

// v141PolicyActivationViolation 在迁移 SQL 里找出策略命名表上的 activation
// 标志列(CREATE TABLE 列块与 ALTER TABLE ADD COLUMN 两个落点)。返回违规
// 描述;空切片 = 无违规。治理事实表的 status 列(单独治理事实,P3 lane)
// 不在词表内 —— BI-01 只禁 activation 标志,不禁治理投影自身的生命周期。
func v141PolicyActivationViolation(migrationsSQL string) []string {
	var violations []string
	createRe := regexp.MustCompile(`(?is)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z0-9_.]+)\s*\(`)
	for _, m := range createRe.FindAllStringSubmatch(migrationsSQL, -1) {
		table := m[1]
		if !strings.Contains(strings.ToLower(table), "polic") {
			continue
		}
		bodyRe := regexp.MustCompile(`(?is)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?` + regexp.QuoteMeta(table) + `\s*\((.*?)\n\);`)
		body := bodyRe.FindStringSubmatch(migrationsSQL)
		if body == nil {
			continue
		}
		if v141ActivationColumnRe.MatchString(body[1]) {
			violations = append(violations, "CREATE TABLE "+table+" 携带 activation 标志列")
		}
	}
	alterRe := regexp.MustCompile(`(?is)ALTER\s+TABLE\s+(?:IF\s+EXISTS\s+)?([a-zA-Z0-9_.]+)\s+([^;]*);`)
	for _, m := range alterRe.FindAllStringSubmatch(migrationsSQL, -1) {
		table := m[1]
		if !strings.Contains(strings.ToLower(table), "polic") {
			continue
		}
		if v141ActivationColumnRe.MatchString(m[2]) {
			violations = append(violations, "ALTER TABLE "+table+" 增加 activation 标志列")
		}
	}
	return violations
}

// v141-2c(BI-01/DL-08 持久化面):数据库不得把策略声明为 runtime-active。
func TestV141NoDatabasePolicyActivationFlag(t *testing.T) {
	if v := v141PolicyActivationViolation(readAllMigrations(t)); len(v) > 0 {
		t.Fatalf("迁移出现策略 activation 标志(BI-01:activation 只能是 gitwriter 提交 canonical YAML): %v", v)
	}
}

// v141-2c 负例(守卫必须真的会响):合成迁移上的 activation 标志必须被
// 识别,而治理事实表的 status 列(BI-01 允许的分离投影)必须不误报。
func TestV141PolicyActivationGuardDetectsViolations(t *testing.T) {
	dirtyCreate := "CREATE TABLE engineering_policy (\n  id TEXT,\n  is_active BOOLEAN NOT NULL\n);"
	if v := v141PolicyActivationViolation(dirtyCreate); len(v) == 0 {
		t.Fatal("守卫未识别 CREATE TABLE 上的 is_active 标志(负例:守卫必须会响)")
	}
	dirtyAlter := "ALTER TABLE policy_governance ADD COLUMN runtime_active BOOLEAN NOT NULL DEFAULT false;"
	if v := v141PolicyActivationViolation(dirtyAlter); len(v) == 0 {
		t.Fatal("守卫未识别 ALTER TABLE 增加的 runtime_active 标志(负例:守卫必须会响)")
	}
	cleanGovernance := "CREATE TABLE policy_governance (\n  policy_id TEXT,\n  status TEXT NOT NULL\n);"
	if v := v141PolicyActivationViolation(cleanGovernance); len(v) != 0 {
		t.Fatalf("治理事实表的 status 列被误报为 activation 标志(BI-01 允许治理投影自带生命周期): %v", v)
	}
}

// ---- v141-BI02:Outcome kind 集合恰为五类 --------------------------------------

// v141PRD27NonKindLabels 是 PRD §27 出现、但按 BI-02 不属于 Outcome kind 集
// 的标签:前三个保持 workflow/attention/integration 事实(runstore/attention
// 投影);后三个是 §27 的原始拼写,只能按 BI-02 映射到既有 kind,不得成为
// 新 kind。
var v141PRD27NonKindLabels = []string{
	"rejected", "escalated", "integration_failed",
	"human_corrected", "hotfix_required", "regression_detected",
}

// v141CheckKindValues 提取迁移文本里 event_kind CHECK 枚举体的全部引号值
// (无 CHECK 时返回 nil)。
func v141CheckKindValues(migrationSQL string) []string {
	re := regexp.MustCompile(`(?is)CHECK\s*\(event_kind\s+IN\s*\((.*?)\)\)`)
	m := re.FindStringSubmatch(migrationSQL)
	if m == nil {
		return nil
	}
	var vals []string
	for _, pair := range regexp.MustCompile(`'([^']+)'`).FindAllStringSubmatch(normSQL(m[1]), -1) {
		vals = append(vals, pair[1])
	}
	return vals
}

// v141EventKindTables 迁移里定义了 event_kind 列的全部表名(平行事件库检测)。
func v141EventKindTables(migrationsSQL string) map[string]bool {
	tables := map[string]bool{}
	re := regexp.MustCompile(`(?is)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z0-9_.]+)\s*\((.*?)\n\);`)
	for _, m := range re.FindAllStringSubmatch(migrationsSQL, -1) {
		if regexp.MustCompile(`(?i)\bevent_kind\b`).MatchString(m[2]) {
			tables[m[1]] = true
		}
	}
	return tables
}

// v141-3(BI-02/DL-12):kind 集合恰为五类、§27 标签不渗入、event_kind 列
// 只属于 outcome_event 一张表。既有 AC1(0023 CHECK 五类在场 + 领域常量恰五)
// 之上,v1.4.1 读法再钉:CHECK 侧同样封闭(多一个值就是 DL-12 的扩张)。
func TestV141OutcomeKindSetStaysFive(t *testing.T) {
	checkVals := v141CheckKindValues(readOutcomeMigration(t))
	if checkVals == nil {
		t.Fatal("0023 缺少 event_kind 的封闭枚举 CHECK(夹具失真)")
	}
	checkSet := map[string]bool{}
	for _, v := range checkVals {
		checkSet[v] = true
	}
	for _, kind := range outcomeKinds {
		if !checkSet[kind] {
			t.Errorf("event_kind CHECK 缺少 kind %q(BI-02/DL-12)", kind)
		}
	}
	if len(checkSet) != len(outcomeKinds) {
		t.Errorf("event_kind CHECK 有 %d 个值,应恰为五类(BI-02/DL-12:kind 集合冻结,扩张需 ADR)", len(checkSet))
	}

	// 领域侧常量值:恰为五类,且 §27 非 kind 标签一个都不在。
	domain := readOutcomeFile(t, filepath.Join(outcomeDomainDir, "types.go"))
	kindVals := map[string]bool{}
	for _, kc := range regexp.MustCompile(`Kind\w+\s+Kind\s+=\s+"([^"]+)"`).FindAllStringSubmatch(domain, -1) {
		kindVals[kc[1]] = true
	}
	for _, bad := range v141PRD27NonKindLabels {
		if kindVals[bad] {
			t.Errorf("internal/outcome 出现 §27 非 kind 常量值 %q(BI-02:rejected/escalated/integration_failed 是 workflow/integration 事实;原始拼写只能映射不得新增)", bad)
		}
	}
	if len(kindVals) != len(outcomeKinds) {
		t.Errorf("internal/outcome 有 %d 个 Kind 常量,应恰为五类(BI-02/DL-12): %v", len(kindVals), kindVals)
	}

	// 查询面与 sqlc 生成面:§27 非 kind 标签不得作为 kind 字面量渗入语句。
	queries := readOutcomeQueries(t)
	for _, name := range []string{"InsertOutcomeEvent", "ListOutcomeEventsByTask", "ListOutcomeEventsByChangeset"} {
		stmt := strings.ToLower(normSQL(sqlcQuery(t, queries, name)))
		for _, bad := range v141PRD27NonKindLabels {
			if strings.Contains(stmt, "'"+bad+"'") {
				t.Errorf("%s 的语句正文出现 §27 非 kind 字面量 %q(BI-02)", name, bad)
			}
		}
	}
	gen := strings.ToLower(readOutcomeFile(t, outcomeGeneratedPath))
	for _, bad := range v141PRD27NonKindLabels {
		if strings.Contains(gen, `"`+bad+`"`) {
			t.Errorf("sqlc 生成面出现 §27 非 kind 字面量 %q(BI-02)", bad)
		}
	}

	// event_kind 列只属于 outcome_event 一张表(没有平行的 case-history 事件库)。
	tables := v141EventKindTables(readAllMigrations(t))
	if !tables["outcome_event"] || len(tables) != 1 {
		t.Errorf("定义 event_kind 的表 = %v,应只有 outcome_event(BI-02:不得另建平行事件库)", tables)
	}
}

// v141-4(BI-02 行为面,负例):§27 标签与拼写变体必须被 fail-safe 拒绝,
// 绝不折叠进任何已知 kind(未知的 outcome 不是「大概是 accepted」)。
func TestV141OutcomeParseKindRejectsPRD27Labels(t *testing.T) {
	// 夹具同源校验:五类必须可解析(守卫词表与 PRD §23 同源)。
	for _, kind := range outcomeKinds {
		got, err := outcome.ParseKind(kind)
		if err != nil || string(got) != kind {
			t.Fatalf("ParseKind(%q): err = %v, got = %q(夹具失真)", kind, err, got)
		}
	}
	for _, bad := range v141PRD27NonKindLabels {
		if _, err := outcome.ParseKind(bad); !errors.Is(err, outcome.ErrUnknownKind) {
			t.Errorf("ParseKind(%q) 未拒绝: err = %v(BI-02:§27 workflow/integration 标签与原始拼写不是 outcome kind)", bad, err)
		}
	}
	for _, bad := range []string{"", "Accepted"} {
		if _, err := outcome.ParseKind(bad); !errors.Is(err, outcome.ErrUnknownKind) {
			t.Errorf("ParseKind(%q) 未拒绝: err = %v(BI-02:kind 必须显式写入,无默认 outcome、大小写不折叠)", bad, err)
		}
	}
}

// v141-3 负例(守卫必须真的会响,不触碰受保护的 0023 迁移):CHECK 枚举里
// 多出的值、平行事件表的 event_kind 列,都必须被提取器识别。
func TestV141OutcomeGuardsDetectViolations(t *testing.T) {
	extra := "CHECK (event_kind IN\n  ('accepted', 'reverted', 'hotfixed', 'regression_found',\n   'human_correction', 'rejected'))"
	vals := v141CheckKindValues("CREATE TABLE outcome_event (\n  event_kind TEXT NOT NULL " + extra + "\n);")
	found := false
	for _, v := range vals {
		if v == "rejected" {
			found = true
		}
	}
	if !found || len(vals) != 6 {
		t.Fatalf("守卫未从 CHECK 枚举体提取出多出的值(负例:守卫必须会响): %v", vals)
	}
	parallel := "CREATE TABLE outcome_case_history (\n  id TEXT,\n  event_kind TEXT NOT NULL\n);"
	if got := v141EventKindTables(parallel); !got["outcome_case_history"] {
		t.Fatal("守卫未识别平行事件表的 event_kind 列(负例:守卫必须会响)")
	}
}
