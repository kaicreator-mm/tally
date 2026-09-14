package arch

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// P0-05:llm_usage 用量可用性契约(决策锁 16 / PRD §29 item 6-7 / INV-V14-12:
// 未知 token/成本用量不得抹掉「调用发生过」的事实,也不得把未知合成成
// 已上报的 0)。沿用本目录惯例做静态契约检查:无真库可用,直接断言
// migrations/0022_v14_measurement_usage.sql 与 internal/store/queries/
// measurement.sql 的结构,把「未上报 ≠ 0」这条铁律写进测试。

const (
	measurementUsageMigrationPath = "../../migrations/0022_v14_measurement_usage.sql"
	measurementQueriesPath        = "../../internal/store/queries/measurement.sql"
)

func readMeasurementUsageMigration(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.FromSlash(measurementUsageMigrationPath))
	if err != nil {
		t.Fatalf("读 %s: %v", measurementUsageMigrationPath, err)
	}
	return string(b)
}

func readMeasurementQueries(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.FromSlash(measurementQueriesPath))
	if err != nil {
		t.Fatalf("读 %s: %v", measurementQueriesPath, err)
	}
	return string(b)
}

// AC1:0022 以显式列 + 无歧义可空语义表示用量可用性。
// usage_reported BOOLEAN NOT NULL 回填 true(0021 时代只有真实上报过数值的
// 调用才会落行),随后 DROP DEFAULT:写入必须显式声明,缺省 true 会把未来的
// 未知调用伪装成已上报。
func TestMeasurementUsageReportedColumnExplicit(t *testing.T) {
	content := readMeasurementUsageMigration(t)
	if !strings.Contains(content, "-- +goose Up") || !strings.Contains(content, "-- +goose Down") {
		t.Errorf("%s 缺少 goose Up/Down 标记", measurementUsageMigrationPath)
	}
	if !regexp.MustCompile(`(?is)ALTER\s+TABLE\s+llm_usage\s+ADD\s+COLUMN\s+usage_reported\s+BOOLEAN\s+NOT\s+NULL\s+DEFAULT\s+true`).MatchString(content) {
		t.Errorf("0022 缺少 usage_reported BOOLEAN NOT NULL DEFAULT true(AC1:存量行如实回填为已上报)")
	}
	if !regexp.MustCompile(`(?is)ALTER\s+TABLE\s+llm_usage\s+ALTER\s+COLUMN\s+usage_reported\s+DROP\s+DEFAULT`).MatchString(content) {
		t.Errorf("0022 未在回填后 DROP usage_reported 的 DEFAULT(AC1:可用性必须显式写入,不得靠缺省值伪装)")
	}
}

// AC1:未知数值用 NULL 表示 —— token/cost 三列 DROP NOT NULL(0021 是 NOT NULL,
// 未知用量的调用因此无法落行);latency_micros 保持 NOT NULL(Tally 自观测墙钟,
// 与 provider 是否上报用量无关)。
func TestMeasurementUnknownValuesAreNullNotZero(t *testing.T) {
	content := readMeasurementUsageMigration(t)
	for _, col := range []string{"input_tokens", "output_tokens", "cost_micros"} {
		re := regexp.MustCompile(`(?is)ALTER\s+COLUMN\s+` + col + `\s+DROP\s+NOT\s+NULL`)
		if !re.MatchString(content) {
			t.Errorf("0022 缺少 %s DROP NOT NULL(AC1:NULL = 未观测,非 NULL = 已上报观测)", col)
		}
	}
	if regexp.MustCompile(`(?is)ALTER\s+COLUMN\s+latency_micros\b`).MatchString(content) {
		t.Errorf("0022 不应改动 latency_micros 的可空性(墙钟由 Tally 自观测,恒为已知)")
	}
}

// AC1+AC2:一致性 CHECK 把「显式标志」与「可空数值」锁成同一事实:
// 已上报 ⇒ 数值全部非 NULL;未上报 ⇒ 数值全部 NULL(把未知写成假 0 的通道
// 在数据库层被拒)。约束体含分组括号,提取到语句分号为止、剥掉括号后断言。
func TestMeasurementUsageReportedConsistencyCheck(t *testing.T) {
	content := readMeasurementUsageMigration(t)
	start := regexp.MustCompile(`(?s)ADD\s+CONSTRAINT\s+llm_usage_usage_reported_consistent\s+CHECK\s*\(`)
	loc := start.FindStringIndex(content)
	if loc == nil {
		t.Fatal("0022 缺少 llm_usage_usage_reported_consistent CHECK 约束")
	}
	rest := content[loc[1]:]
	end := strings.Index(rest, ";")
	if end < 0 {
		t.Fatal("0022 的 CHECK 约束语句没有终止分号")
	}
	body := normSQL(strings.NewReplacer("(", " ", ")", " ").Replace(rest[:end]))
	reportedArm := "usage_reported and input_tokens is not null and output_tokens is not null and cost_micros is not null"
	unknownArm := "not usage_reported and input_tokens is null and output_tokens is null and cost_micros is null"
	if !strings.Contains(body, reportedArm) {
		t.Errorf("CHECK 缺少「已上报 ⇒ 数值全部非 NULL」分支: %q", body)
	}
	if !strings.Contains(body, unknownArm) {
		t.Errorf("CHECK 缺少「未上报 ⇒ 数值全部 NULL」分支(AC2:未知不得携带数值): %q", body)
	}
}

// AC3:append-only 不因 0022 弱化 —— 0021 的行级触发器仍在,0022 不删除/禁用
// 触发器,也不引入 llm_usage 的 UPDATE/DELETE 数据路径;查询面同样只有
// INSERT/SELECT。
func TestMeasurementUsageAppendOnlyPreserved(t *testing.T) {
	content := readMeasurementUsageMigration(t)
	if regexp.MustCompile(`(?is)(DROP|DISABLE)\s+TRIGGER`).MatchString(content) {
		t.Errorf("0022 出现 DROP/DISABLE TRIGGER(AC3:append-only 触发器不得移除)")
	}
	all := readAllMigrations(t)
	if !regexp.MustCompile(`(?is)CREATE\s+TRIGGER\s+\S+\s+BEFORE\s+UPDATE\s+OR\s+DELETE\s+ON\s+llm_usage\b`).MatchString(all) {
		t.Errorf("llm_usage 缺少阻止 UPDATE/DELETE 的触发器(AC3:INV-FACT-01)")
	}
	if re := regexp.MustCompile(`(?is)\b(update\s+llm_usage\b|delete\s+from\s+llm_usage\b)`); re.MatchString(content) {
		t.Errorf("0022 出现 llm_usage 的 UPDATE/DELETE 数据路径(AC3:只追加)")
	}
	for _, file := range queryFiles(t) {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("读 %s: %v", file, err)
		}
		if regexp.MustCompile(`(?is)\b(update\s+llm_usage\b|delete\s+from\s+llm_usage\b)`).Match(b) {
			t.Errorf("%s: 对 llm_usage 的 UPDATE/DELETE 查询(AC3:度量事实只追加)", file)
		}
	}
}

// AC2+AC4:写入合同 —— InsertLLMUsage 必须携带 usage_reported,且不得在写侧
// 把 NULL 强转成 0(coalesce 一旦出现在 INSERT,就是把未知伪装成已上报 0 的
// 回头路);ListLLMUsageByTask 随行带回 usage_reported,读侧可逐行区分
// 「未知用量」与「观测到 0」。
func TestMeasurementUsageInsertAndListCarryAvailability(t *testing.T) {
	content := readMeasurementQueries(t)
	insert := normSQL(sqlcQuery(t, content, "InsertLLMUsage"))
	if !strings.Contains(insert, "insert into llm_usage") {
		t.Fatalf("InsertLLMUsage 不是对 llm_usage 的 INSERT")
	}
	if !strings.Contains(insert, "usage_reported") {
		t.Errorf("InsertLLMUsage 未写入 usage_reported(AC2:可用性必须在写入时显式落成)")
	}
	if strings.Contains(insert, "coalesce(") {
		t.Errorf("InsertLLMUsage 出现 coalesce(写侧把 NULL 强转 0 会把未知伪装成已上报,锁 16)")
	}
	list := normSQL(sqlcQuery(t, content, "ListLLMUsageByTask"))
	if !strings.Contains(list, "usage_reported") {
		t.Errorf("ListLLMUsageByTask 未返回 usage_reported(AC4:逐行未知/零不可区分)")
	}
}

// AC4:聚合合同 —— total/reported/missing 三列计数,sum 只累计已上报数值
// (未知行不进数值合计,也不会被当作 0 计入已上报);空合计 coalesce 为 0
// (空和),此时 calls_reported=0 且 calls_missing_usage 指明原因。
func TestMeasurementUsageAggregatesTotalReportedMissing(t *testing.T) {
	content := readMeasurementQueries(t)
	for _, name := range []string{"SumLLMCostByPlane", "SumLLMCostByPlaneForTask"} {
		q := normSQL(sqlcQuery(t, content, name))
		for _, want := range []string{
			"count(*)::bigint as calls_total",
			"count(*) filter (where usage_reported)::bigint as calls_reported",
			"count(*) filter (where not usage_reported)::bigint as calls_missing_usage",
			"coalesce(sum(cost_micros), 0)::bigint as cost_micros",
			"coalesce(sum(input_tokens), 0)::bigint as input_tokens",
			"coalesce(sum(output_tokens), 0)::bigint as output_tokens",
		} {
			if !strings.Contains(q, want) {
				t.Errorf("%s 缺少 %q(AC4:total/reported/missing 聚合)", name, want)
			}
		}
	}
}
