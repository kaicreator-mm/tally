package arch

import (
	"os"
	"regexp"
	"testing"
)

// 事实面五张表:只 INSERT,不 UPDATE 不 DELETE(INV-5)。
var factTables = []string{"build_run", "deploy_run", "environment_run", "failure_event", "dependency_change"}

// INV-5:sqlc 查询里不允许对事实面表的 UPDATE/DELETE。
func TestNoUpdateDeleteOnFacts(t *testing.T) {
	for _, file := range queryFiles(t) {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("读 %s: %v", file, err)
		}
		for _, tbl := range factTables {
			re := regexp.MustCompile(`(?is)\b(UPDATE\s+` + tbl + `|DELETE\s+FROM\s+` + tbl + `)\b`)
			if re.Match(b) {
				t.Errorf("%s: 对事实面表 %s 的 UPDATE/DELETE 查询(INV-5 事实面只追加)", file, tbl)
			}
		}
	}
}

// INV-5 双保险:每张事实面表在数据库层有阻止 UPDATE/DELETE 的触发器。
func TestFactsHaveAppendOnlyTrigger(t *testing.T) {
	content := readAllMigrations(t)
	for _, tbl := range factTables {
		re := regexp.MustCompile(`(?is)CREATE\s+TRIGGER\s+\S+\s+BEFORE\s+UPDATE\s+OR\s+DELETE\s+ON\s+` + tbl + `\b`)
		if !re.MatchString(content) {
			t.Errorf("事实面表 %s 缺少阻止 UPDATE/DELETE 的触发器(INV-5 双保险)", tbl)
		}
	}
}
