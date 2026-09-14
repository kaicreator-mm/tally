package arch

import (
	"regexp"
	"testing"
)

// INV-7:只做有机器记录对应的功能。
// 字段黑名单出现即失败 —— 把最容易长回来的几个钉死。
// v3 澄清:限的是机器填不出的*跟踪字段*,不限用户创作内容(需求/文档/笔记)。
func TestNoManualFields(t *testing.T) {
	blacklist := []string{
		"estimated_hours", "actual_hours", "assignee", "priority",
		"story_points", "reviewer", "approver", "due_date",
	}
	content := readAllMigrations(t)
	for _, word := range blacklist {
		re := regexp.MustCompile(`(?i)\b` + word + `\b`)
		if re.MatchString(content) {
			t.Errorf("迁移中出现黑名单字段 %q(INV-7)", word)
		}
	}
}
