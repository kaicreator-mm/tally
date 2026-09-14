package arch

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// P3-05:outcome_event 契约(PRD §23 OutcomeLog / 决策锁 14:outcome 是
// append-only 事件捕获,不得创建可变 outcome 布尔列当权威;§23:v1.4 只做
// 耐久捕获,这些数据事后无法完整补采)。沿用本目录惯例做静态契约检查:
// 无真库可用,直接断言 migrations/0023_v14_outcome_event.sql、
// internal/store/queries/outcome_event.sql 与 sqlc 生成面的结构,把
// 「事件流不可改写」这条铁律写进测试。

const (
	outcomeMigrationPath = "../../migrations/0023_v14_outcome_event.sql"
	outcomeQueriesPath   = "../../internal/store/queries/outcome_event.sql"
	outcomeGeneratedPath = "../../internal/store/outcome_event.sql.go"
	outcomeModelsPath    = "../../internal/store/models.go"
	outcomeQuerierPath   = "../../internal/store/querier.go"
	outcomeDomainDir     = "../../internal/outcome"
)

// outcomeKinds 是 PRD §23 的最小事件集(AC1 封闭枚举,两侧共用)。
var outcomeKinds = []string{"accepted", "reverted", "hotfixed", "regression_found", "human_correction"}

func readOutcomeMigration(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.FromSlash(outcomeMigrationPath))
	if err != nil {
		t.Fatalf("读 %s: %v", outcomeMigrationPath, err)
	}
	return string(b)
}

func readOutcomeQueries(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.FromSlash(outcomeQueriesPath))
	if err != nil {
		t.Fatalf("读 %s: %v", outcomeQueriesPath, err)
	}
	return string(b)
}

func readOutcomeFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.FromSlash(path))
	if err != nil {
		t.Fatalf("读 %s: %v", path, err)
	}
	return string(b)
}

// AC1:event_kind 是封闭枚举 —— 0023 的 CHECK 限定为 PRD §23 的最小集,
// 五个 kind 逐一在列;internal/outcome 侧声明的 Kind 常量与之一一对应,
// 两侧之外没有第三个来源。
func TestOutcomeEventKindClosedEnum(t *testing.T) {
	content := readOutcomeMigration(t)
	if !strings.Contains(content, "-- +goose Up") || !strings.Contains(content, "-- +goose Down") {
		t.Errorf("%s 缺少 goose Up/Down 标记", outcomeMigrationPath)
	}
	re := regexp.MustCompile(`(?is)CHECK\s*\(event_kind\s+IN\s*\((.*?)\)\)`)
	m := re.FindStringSubmatch(content)
	if m == nil {
		t.Fatal("0023 缺少 event_kind 的封闭枚举 CHECK")
	}
	enumBody := normSQL(m[1])
	for _, kind := range outcomeKinds {
		if !strings.Contains(enumBody, "'"+kind+"'") {
			t.Errorf("event_kind CHECK 缺少 PRD §23 的 kind %q(AC1): %s", kind, enumBody)
		}
	}

	// 领域侧常量必须与 CHECK 完全一致(封闭枚举的两侧同源,无第三种值)。
	domain := readOutcomeFile(t, filepath.Join(outcomeDomainDir, "types.go"))
	kindConsts := regexp.MustCompile(`Kind\w+\s+Kind\s+=\s+"([^"]+)"`).FindAllStringSubmatch(domain, -1)
	got := make(map[string]bool, len(kindConsts))
	for _, kc := range kindConsts {
		got[kc[1]] = true
	}
	if len(got) != len(outcomeKinds) {
		t.Errorf("internal/outcome 声明了 %d 个 Kind 常量,应恰为 PRD §23 的 %d 个(AC1): %v", len(got), len(outcomeKinds), got)
	}
	for _, kind := range outcomeKinds {
		if !got[kind] {
			t.Errorf("internal/outcome 缺少 Kind 常量 %q(AC1 封闭枚举两侧同源)", kind)
		}
	}
}

// AC2:事件绑定 task/change(可得时必填,可空)与 timestamp(必填):
// task_id / changeset_id 为可空 TEXT,recorded_at TIMESTAMPTZ NOT NULL,
// 不由 DB 默认补时间。写入合同同样携带三个坐标。
func TestOutcomeEventBindingsAndTimestamp(t *testing.T) {
	body := createTableBody(t, readOutcomeMigration(t), "outcome_event")
	for _, want := range []struct{ name, typ string }{
		{"task_id", "TEXT"},
		{"changeset_id", "TEXT"},
		{"event_kind", "TEXT NOT NULL"},
		{"recorded_at", "TIMESTAMPTZ NOT NULL"},
	} {
		re := regexp.MustCompile(`(?is)\b` + want.name + `\s+` + want.typ + `\b`)
		if !re.MatchString(body) {
			t.Errorf("outcome_event 缺少列定义 %s %s(AC2)", want.name, want.typ)
		}
	}
	// 绑定「可得时必填」:两列可空, NULL 表示坐标缺失(0021 measurement 惯例)。
	for _, col := range []string{"task_id", "changeset_id"} {
		if regexp.MustCompile(`(?is)\b` + col + `\s+TEXT\s+NOT\s+NULL`).MatchString(body) {
			t.Errorf("outcome_event.%s 不应为 NOT NULL(AC2:绑定可得时必填,缺失以 NULL 表示)", col)
		}
	}

	insert := normSQL(sqlcQuery(t, readOutcomeQueries(t), "InsertOutcomeEvent"))
	if !strings.Contains(insert, "insert into outcome_event") {
		t.Fatalf("InsertOutcomeEvent 不是对 outcome_event 的 INSERT")
	}
	for _, col := range []string{"task_id", "changeset_id", "event_kind", "recorded_at"} {
		if !strings.Contains(insert, col) {
			t.Errorf("InsertOutcomeEvent 未写入 %s(AC2)", col)
		}
	}
}

// AC3:UPDATE/DELETE 双重被拒 —— 0023 建语句级 append-only 触发器
// (0011 口径:0 行命中的改写语句本身也报错);0023 不移除/禁用任何触发器;
// 查询面与 sqlc 生成面不存在对 outcome_event 的任何 UPDATE/DELETE 路径。
func TestOutcomeEventAppendOnlyEnforced(t *testing.T) {
	content := readOutcomeMigration(t)
	if !regexp.MustCompile(`(?is)CREATE\s+TRIGGER\s+outcome_event_append_only\s+BEFORE\s+UPDATE\s+OR\s+DELETE\s+ON\s+outcome_event\s+FOR\s+EACH\s+STATEMENT\s+EXECUTE\s+FUNCTION\s+tally_prevent_modify\s*\(\)`).MatchString(content) {
		t.Errorf("0023 缺少 outcome_event 的语句级 append-only 触发器(AC3:INV-FACT-01,0011 口径)")
	}
	if regexp.MustCompile(`(?is)(DROP|DISABLE)\s+TRIGGER`).MatchString(content) {
		t.Errorf("0023 出现 DROP/DISABLE TRIGGER(AC3:append-only 防线不得移除)")
	}

	modifyRe := regexp.MustCompile(`(?is)\b(update\s+outcome_event\b|delete\s+from\s+outcome_event\b)`)
	for _, file := range queryFiles(t) {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("读 %s: %v", file, err)
		}
		if modifyRe.Match(b) {
			t.Errorf("%s: 对 outcome_event 的 UPDATE/DELETE 查询(AC3:outcome 事实只追加)", file)
		}
	}
	for _, gen := range []string{outcomeGeneratedPath, outcomeModelsPath, outcomeQuerierPath} {
		if modifyRe.MatchString(readOutcomeFile(t, gen)) {
			t.Errorf("%s: 生成面出现 outcome_event 的 UPDATE/DELETE 路径(AC3)", gen)
		}
	}
}

// AC4:sqlc 生成面由 query/schema 源产出 —— outcome_event.sql.go 含三条
// 查询的常量与方法;models.go 有 OutcomeEvent 模型;querier.go 接口暴露
// 同名方法(与 queries/outcome_event.sql 一一对应)。
func TestOutcomeEventSqlcGeneratedSurface(t *testing.T) {
	gen := readOutcomeFile(t, outcomeGeneratedPath)
	if !strings.Contains(gen, "Code generated by sqlc. DO NOT EDIT.") {
		t.Errorf("%s 不是 sqlc 生成文件(AC4)", outcomeGeneratedPath)
	}
	if !strings.Contains(gen, `// source: outcome_event.sql`) {
		t.Errorf("%s 未标注 source: outcome_event.sql(AC4:生成自 query 源)", outcomeGeneratedPath)
	}
	queries := readOutcomeQueries(t)
	for _, name := range []string{"InsertOutcomeEvent", "ListOutcomeEventsByTask", "ListOutcomeEventsByChangeset"} {
		if sqlcQuery(t, queries, name) == "" {
			t.Errorf("queries/outcome_event.sql 缺少查询 %s(AC4)", name)
		}
		// 生成面:查询常量与 Queries 方法一一在场。
		if !strings.Contains(gen, "name: "+name+" :") {
			t.Errorf("%s 缺少查询 %s(AC4)", outcomeGeneratedPath, name)
		}
		if !strings.Contains(readOutcomeFile(t, outcomeQuerierPath), name+"(ctx context.Context") {
			t.Errorf("querier.go 接口缺少 %s(AC4)", name)
		}
	}
	if !strings.Contains(readOutcomeFile(t, outcomeModelsPath), "type OutcomeEvent struct") {
		t.Errorf("models.go 缺少 OutcomeEvent 模型(AC4)")
	}
}

// AC5:没有派生 outcome 状态成为可变权威 —— 表只分类(event_kind)不裁决:
// 无 status 类可变状态列、无布尔 outcome 列;0023 只建 outcome_event 一张表
// (没有第二个 outcome 投影/状态表);查询面对 outcome_event 只有 INSERT/SELECT。
func TestOutcomeEventNoMutableStatusAuthority(t *testing.T) {
	content := readOutcomeMigration(t)
	body := createTableBody(t, content, "outcome_event")
	for _, banned := range []string{"status", "is_accepted", "is_reverted", "outcome_state"} {
		if regexp.MustCompile(`(?is)\b` + banned + `\b`).MatchString(body) {
			t.Errorf("outcome_event 出现可变状态/布尔列 %q(AC5,锁 14:outcome 只以事件捕获)", banned)
		}
	}
	if n := len(regexp.MustCompile(`(?is)CREATE\s+TABLE\s+\w+`).FindAllString(content, -1)); n != 1 {
		t.Errorf("0023 建了 %d 张表,应只建 outcome_event(AC5:不落派生 outcome 状态表)", n)
	}

	// 逐条语句正文检查(sqlcQuery 剥掉文档注释):只允许 INSERT/SELECT。
	queriesRaw := readOutcomeQueries(t)
	for _, name := range []string{"InsertOutcomeEvent", "ListOutcomeEventsByTask", "ListOutcomeEventsByChangeset"} {
		stmt := normSQL(sqlcQuery(t, queriesRaw, name))
		if regexp.MustCompile(`\b(update|delete from)\b`).MatchString(stmt) {
			t.Errorf("%s 出现 UPDATE/DELETE 语句(AC5:事件面只 INSERT/SELECT)", name)
		}
	}
}
