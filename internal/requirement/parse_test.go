package requirement

import (
	"strings"
	"testing"

	"tally/internal/core"
)

// ParseEARS 正例:EARS 五种形态 + 复合前置都要解析对,并满足 ruleset。
func TestParseEARS(t *testing.T) {
	tests := []struct {
		name       string
		stmt       string
		system     string
		trigKind   EARSKind // 空 = 无触发
		preKinds   []EARSKind
		nResponses int
	}{
		{"常态", "系统 shall 校验所有必填字段", "系统", "", nil, 1},
		{"事件 WHEN", "WHEN 用户点击保存, 系统 shall 将草稿写入 git", "系统", KindWhen, nil, 1},
		{"状态 WHILE", "WHILE 飞行模式, 系统 shall 缓存外发消息", "系统", "", []EARSKind{KindWhile}, 1},
		{"可选 WHERE", "WHERE 启用了加密, 系统 shall 使用 TLS 1.3", "系统", "", []EARSKind{KindWhere}, 1},
		{"异常 IF-THEN", "IF 磁盘已满, THEN 系统 shall 返回 507", "系统", KindIf, nil, 1},
		{"复合前置+多响应", "WHILE 离线, WHEN 收到消息, 系统 shall 入队; 并提示用户", "系统", KindWhen, []EARSKind{KindWhile}, 2},
		{"英文冠词", "WHEN the user saves, the tally server shall persist the draft", "tally server", KindWhen, nil, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEARS(tt.stmt)
			if err != nil {
				t.Fatalf("ParseEARS(%q) 出错: %v", tt.stmt, err)
			}
			if got.System != tt.system {
				t.Errorf("系统名 = %q, 期望 %q", got.System, tt.system)
			}
			if tt.trigKind == "" {
				if got.Trigger != nil {
					t.Errorf("不应有触发, got %+v", got.Trigger)
				}
			} else {
				if got.Trigger == nil || got.Trigger.Kind != tt.trigKind {
					t.Errorf("触发 = %+v, 期望 %s", got.Trigger, tt.trigKind)
				}
			}
			if len(got.Preconditions) != len(tt.preKinds) {
				t.Errorf("前置数 = %d, 期望 %d (%+v)", len(got.Preconditions), len(tt.preKinds), got.Preconditions)
			}
			for i, k := range tt.preKinds {
				if i < len(got.Preconditions) && got.Preconditions[i].Kind != k {
					t.Errorf("前置[%d] = %s, 期望 %s", i, got.Preconditions[i].Kind, k)
				}
			}
			if len(got.Responses) != tt.nResponses {
				t.Errorf("响应数 = %d (%v), 期望 %d", len(got.Responses), got.Responses, tt.nResponses)
			}
		})
	}
}

// ParseEARS 负例:违反 ruleset 的语句必须被拒。
func TestParseEARSRejects(t *testing.T) {
	tests := []struct {
		name, stmt, want string
	}{
		{"缺 shall", "系统 校验字段", "shall"},
		{"两个 shall(不唯一系统名)", "系统 shall A; 系统 shall B", "shall"},
		{"缺响应", "WHEN 用户点击, 系统 shall ", "响应"},
		{"两个触发", "WHEN a, IF b, 系统 shall c", "触发"},
		{"缺系统名", "WHEN 用户点击, shall 写入", "系统名"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseEARS(tt.stmt)
			if err == nil {
				t.Fatalf("ParseEARS(%q) 应报错", tt.stmt)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("错误 %q 未提到 %q", err.Error(), tt.want)
			}
		})
	}
}

const sampleDoc = `# 草稿管理

## REQ-1 保存草稿

WHEN 用户点击保存, 系统 shall 将草稿写入 git。

验收: 自动化测试
- 点击保存后仓库出现新提交

## REQ-2 草稿加密

WHERE 启用了加密, 系统 shall 对草稿内容加密存储。

验收: 指标
- 静态扫描测到明文即失败

## 非目标

- 不做云端同步
- 不做多人协同

## 术语表

- 草稿: 未签署的需求文本
- 签署: 冻结某版需求并记内容哈希
`

// Parse 正例:整份文档解析出条款、非目标、术语表。
func TestParseDocument(t *testing.T) {
	doc := Parse(sampleDoc)
	if doc.Title != "草稿管理" {
		t.Errorf("标题 = %q", doc.Title)
	}
	if len(doc.Clauses) != 2 {
		t.Fatalf("条款数 = %d, 期望 2", len(doc.Clauses))
	}
	c1 := doc.Clauses[0]
	if c1.ID != "REQ-1" || c1.Title != "保存草稿" {
		t.Errorf("REQ-1 解析错: %+v", c1)
	}
	if !strings.Contains(c1.Statement, "WHEN 用户点击保存") {
		t.Errorf("REQ-1 语句 = %q", c1.Statement)
	}
	if c1.Mechanism != MechAutoTest {
		t.Errorf("REQ-1 机制 = %q", c1.Mechanism)
	}
	if len(c1.Criteria) != 1 {
		t.Errorf("REQ-1 验收条件数 = %d", len(c1.Criteria))
	}
	if len(doc.NonGoals) != 2 {
		t.Errorf("非目标数 = %d", len(doc.NonGoals))
	}
	if len(doc.Glossary) != 2 || doc.Glossary[0].Name != "草稿" {
		t.Errorf("术语表 = %+v", doc.Glossary)
	}
}

// ContentHash:确定性 + 内容敏感 + 排版不敏感。
func TestContentHash(t *testing.T) {
	c := Clause{ID: "REQ-1", Statement: "系统 shall A", Mechanism: MechAutoTest, Criteria: []string{"x"}}
	h1 := ContentHash(c)
	if ContentHash(c) != h1 {
		t.Error("相同内容哈希不一致(不确定)")
	}

	// 排版差异(多余空白)不改变哈希
	c2 := c
	c2.Statement = "  系统   shall  A "
	if ContentHash(c2) != h1 {
		t.Error("纯排版差异不应改变内容哈希")
	}

	// 真实内容变化必须改变哈希
	c3 := c
	c3.Statement = "系统 shall B"
	if ContentHash(c3) == h1 {
		t.Error("内容变化后哈希未改变")
	}
	if !strings.HasPrefix(string(h1), "sha256:") {
		t.Errorf("哈希格式 = %q", h1)
	}
	var _ core.ClauseHash = h1
}
