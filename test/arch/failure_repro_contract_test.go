package arch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"tally/internal/command"
)

// P0-04 合同的机器检查(纯静态,不连库):
//
//	§21.6 Structured Repro:复现上下文的持久化不用 shell 字符串 + strings.Fields
//	表示,落结构化 CommandSpec JSON 与结构化 other targets。
//	INV-FACT-01 / INV-5:failure_event 只追加——迁移只加列、不回写存量行,
//	查询合同零修改路径;存量行经「结构化列 + 兼容列」双轨读取保持可读。
//
// 约束落在两处:migrations/0016_failure_command_spec.sql(结构化 schema)与
// internal/store/queries/failure_event.sql(结构化读写合同 + 兼容读取路径)。

const (
	failureReproMigration = "0016_failure_command_spec.sql"
	failureReproQueryPath = "../../internal/store/queries/failure_event.sql"
)

func readFailureReproQueries(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.FromSlash(failureReproQueryPath))
	if err != nil {
		t.Fatalf("读 %s: %v", failureReproQueryPath, err)
	}
	return string(b)
}

// failureSelectList 取指定查询的 SELECT 列清单(到 FROM 为止,已去注释)。
func failureSelectList(t *testing.T, name string) string {
	t.Helper()
	body := stripSQLComments(namedQueryBody(t, failureReproQueryPath, name))
	m := regexp.MustCompile(`(?is)SELECT\s+(.*?)\bFROM\b`).FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("%s 找不到 SELECT 列清单", name)
	}
	return m[1]
}

// hasColumn 判逗号列清单(不区分大小写)里是否有某列。
func hasColumn(list []string, col string) bool {
	for _, c := range list {
		if strings.EqualFold(c, col) {
			return true
		}
	}
	return false
}

// AC1:结构化复现表示就是 CommandSpec 的规范 JSON,字段集固定为
// argv/cwd/env/timeout/shell——落库形态与 internal/command 的线上形态一致,
// shell 恒显式输出(存储里 shell 语义始终可见,缺省即 false)。
func TestReproStorageEncodesCommandSpec(t *testing.T) {
	spec := command.CommandSpec{
		Argv:    []string{"go", "test", "./internal/jobs/..."},
		Cwd:     ".",
		Env:     map[string]string{"GOFLAGS": "-count=1"},
		Timeout: 120 * time.Second,
	}
	b, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("序列化 CommandSpec: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(b, &fields); err != nil {
		t.Fatalf("解析 CommandSpec JSON: %v", err)
	}
	for _, key := range []string{"argv", "cwd", "env", "timeout", "shell"} {
		if _, ok := fields[key]; !ok {
			t.Errorf("CommandSpec 存储 JSON 缺少 %q 字段: %s", key, b)
		}
	}
	if len(fields) != 5 {
		t.Errorf("CommandSpec 存储 JSON 字段集漂移(存储合同按这五个字段解释): %s", b)
	}
	// 往返不丢:落库读回仍是同一个 spec,不能从存储 JSON 构造出非法形态。
	var back command.CommandSpec
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("读回 CommandSpec JSON: %v", err)
	}
	if back.Cwd != spec.Cwd || back.Shell || back.Timeout != spec.Timeout ||
		len(back.Argv) != 3 || back.Argv[1] != "test" || back.Argv[2] != "./internal/jobs/..." ||
		back.Env["GOFLAGS"] != "-count=1" {
		t.Errorf("CommandSpec 往返失真: %s → %+v", b, back)
	}
}

// AC1:0016 把复现持久化合同落成结构化列,且带数据库层形状护栏。
// INV-FACT-01:只加列、给默认值,不回写存量行(存量行新列取 null/'[]' 哨兵默认)。
func TestFailureReproSchemaIsStructured(t *testing.T) {
	up := gooseUpSection(t, failureReproMigration)
	for _, col := range []struct{ name, dflt string }{
		{"repro_command", "'null'"},
		{"other_target_commands", "'[]'"},
	} {
		re := regexp.MustCompile(`(?is)ALTER\s+TABLE\s+failure_event\s+ADD\s+COLUMN\s+` +
			col.name + `\s+JSONB\s+NOT\s+NULL\s+DEFAULT\s+` + regexp.QuoteMeta(col.dflt) + `\s*;`)
		if !re.MatchString(up) {
			t.Errorf("0016 缺少 failure_event.%s JSONB NOT NULL DEFAULT %s(AC1:结构化复现列)", col.name, col.dflt)
		}
	}
	// 形状护栏:repro_command 只能是 null 哨兵或对象,禁止退化成 shell 字符串;
	// other_target_commands 必须是数组。护栏在库层,绕过应用层也写不进坏形状。
	if !regexp.MustCompile(`(?is)ALTER\s+TABLE\s+failure_event\s+ADD\s+CONSTRAINT\s+failure_event_repro_command_shape\s+` +
		`CHECK\s*\(\s*jsonb_typeof\s*\(\s*repro_command\s*\)\s+IN\s*\(\s*'null'\s*,\s*'object'\s*\)\s*\)`).MatchString(up) {
		t.Error("repro_command 缺少形状护栏(null 哨兵或对象):结构化合同可被字符串退化击穿")
	}
	if !regexp.MustCompile(`(?is)ALTER\s+TABLE\s+failure_event\s+ADD\s+CONSTRAINT\s+failure_event_other_target_commands_shape\s+` +
		`CHECK\s*\(\s*jsonb_typeof\s*\(\s*other_target_commands\s*\)\s*=\s*'array'\s*\)`).MatchString(up) {
		t.Error("other_target_commands 缺少形状护栏(必须是 JSON 数组)")
	}
}

// AC2:存量行保持可读——0016 不删不改 0013 的旧复现三列,兼容读取的来源必须
// 原地保留;Down 只回收 0016 自己加的结构化列。
func TestFailureReproMigrationKeepsLegacyColumnsReadable(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoMigrations, failureReproMigration))
	if err != nil {
		t.Fatalf("读 %s: %v", failureReproMigration, err)
	}
	content := string(b)
	if !strings.Contains(content, "-- +goose Up") || !strings.Contains(content, "-- +goose Down") {
		t.Fatal("0016 缺少 goose Up/Down 标记")
	}
	if regexp.MustCompile(`(?i)DROP\s+COLUMN\s+(workdir|repro|other_targets)\b`).MatchString(content) {
		t.Error("0016 删除了旧复现列 workdir/repro/other_targets(存量行兼容读取的唯一来源,AC2)")
	}
	for _, col := range []string{"repro_command", "other_target_commands"} {
		re := regexp.MustCompile(`(?is)--\s*\+goose\s+Down.*DROP\s+COLUMN\s+(?:IF\s+EXISTS\s+)?` + col + `\b`)
		if !re.MatchString(content) {
			t.Errorf("0016 Down 未回收自己加的列 %s", col)
		}
	}
}

// AC1 合同面 + AC2 合同面:新写入只落结构化两列,旧文本三列退出写入;
// 读取按「结构化 + 兼容」双轨返回,repro_command 为 null 的存量行才能经
// 兼容路径(strings.Fields(repro) + workdir)装载。
func TestFailureQueriesCarryStructuredReproContract(t *testing.T) {
	// 写入合同:结构化列必须出现,旧文本列必须缺席。
	insert := stripSQLComments(namedQueryBody(t, failureReproQueryPath, "InsertFailureEvent"))
	m := regexp.MustCompile(`(?is)INSERT\s+INTO\s+failure_event\s*\((.*?)\)\s*VALUES`).FindStringSubmatch(insert)
	if m == nil {
		t.Fatal("InsertFailureEvent 找不到列清单")
	}
	cols := splitPKColumns(m[1])
	for _, col := range []string{"repro_command", "other_target_commands"} {
		if !hasColumn(cols, col) {
			t.Errorf("InsertFailureEvent 列清单缺少结构化列 %s(AC1:新写入落结构化复现)", col)
		}
	}
	for _, col := range []string{"workdir", "repro", "other_targets"} {
		if hasColumn(cols, col) {
			t.Errorf("InsertFailureEvent 仍在写旧文本列 %s(0016 起新写入只落结构化复现)", col)
		}
	}

	// 诊断装载(T10c):结构化与兼容列必须同时返回,存量行才可经兼容路径重建。
	diag := failureSelectList(t, "GetLatestFailureEventByFingerprint")
	for _, col := range []string{"repro_command", "other_target_commands", "workdir", "repro", "other_targets"} {
		if !regexp.MustCompile(`(?i)\b` + col + `\b`).MatchString(diag) {
			t.Errorf("GetLatestFailureEventByFingerprint 的 SELECT 缺少 %s(AC2:双轨读取,存量行可读)", col)
		}
	}

	// 退休检查(D-c4/T11c):结构化 repro_command + 兼容 workdir/repro 双轨。
	retire := failureSelectList(t, "GetLatestFailureEventByProjectAndFingerprint")
	for _, col := range []string{"repro_command", "workdir", "repro"} {
		if !regexp.MustCompile(`(?i)\b` + col + `\b`).MatchString(retire) {
			t.Errorf("GetLatestFailureEventByProjectAndFingerprint 的 SELECT 缺少 %s(AC2:双轨读取,存量行可读)", col)
		}
	}
}

// AC3:failure_event 的查询合同零修改路径——文件里不出现任何
// UPDATE/DELETE(INV-5 / INV-FACT-01,注释不计)。
func TestFailureEventQueriesAppendOnly(t *testing.T) {
	body := stripSQLComments(readFailureReproQueries(t))
	re := regexp.MustCompile(`(?is)\b(?:UPDATE\s+failure_event\b|DELETE\s+FROM\s+failure_event\b)`)
	if re.MatchString(body) {
		t.Error("failure_event.sql 出现 UPDATE/DELETE(事实面只追加,AC3)")
	}
}
