// v141_intent_loader 的单元测试:复用 v13_loader_test.go 的 git 仓库夹具,
// 验证 FrozenIntent 装载的 Git 快照语义(工作区不可见、历史修订可复装载、
// 同一提交确定性)、三类制品级失效的类型化拒收(不存在 / 内容非法 /
// 身份不一致)与基础设施级失败的区分(任务 P0-02 验收:给定 source commit,
// 装载器返回确定性 FrozenIntent 或类型化拒收)。

package artifacts

import (
	"errors"
	"testing"

	"tally/internal/intentevidence"
)

// intentYAML 渲染一份合法的 FrozenIntent 制品字节(Freeze → Render,与
// P0-08 confirm→freeze 事务将写入的形态同源)。
func intentYAML(t *testing.T, taskID string, taskType intentevidence.TaskType, sourceRef string, hints ...string) []byte {
	t.Helper()
	fi, err := intentevidence.Freeze(intentevidence.Facts{
		TaskID:    taskID,
		TaskType:  taskType,
		SourceRef: sourceRef,
		RiskHints: hints,
	})
	if err != nil {
		t.Fatalf("冻结夹具 %s: %v", taskID, err)
	}
	b, err := fi.Render()
	if err != nil {
		t.Fatalf("渲染夹具 %s: %v", taskID, err)
	}
	return b
}

// mustFreeze 冻结夹具事实,返回证据(用于断言 Revision / Equal)。
func mustFreeze(t *testing.T, taskID string, taskType intentevidence.TaskType, sourceRef string, hints ...string) intentevidence.FrozenIntent {
	t.Helper()
	fi, err := intentevidence.Freeze(intentevidence.Facts{
		TaskID:    taskID,
		TaskType:  taskType,
		SourceRef: sourceRef,
		RiskHints: hints,
	})
	if err != nil {
		t.Fatalf("冻结夹具 %s: %v", taskID, err)
	}
	return fi
}

// TestLoadIntentAtHead:从 HEAD 提交树装载 intent —— 内容与修订哈希绑定
// Git blob 实际内容、来源提交可追溯、无关路径视而不见、同提交重复装载
// 逐字段确定(AC:确定性 FrozenIntent)。
func TestLoadIntentAtHead(t *testing.T) {
	rf := initRepo(t)
	want := mustFreeze(t, "TASK-0023", intentevidence.TaskTypeBugFix, "GitHub Issue #142", "schema_migration")
	head := rf.commit(t, map[string]string{
		"src/main.go":                   "package main\n",
		".tally/intents/README.md":      "# 不是意图证据\n",
		".tally/intents/TASK-0023.yaml": string(intentYAML(t, "TASK-0023", intentevidence.TaskTypeBugFix, "GitHub Issue #142", "schema_migration")),
		".tally/intents/TASK-0024.yaml": string(intentYAML(t, "TASK-0024", intentevidence.TaskTypeMigration, "", "data_integrity")),
		".tally/tasks/TASK-0023.yaml":   taskYAML("TASK-0023", "无关的 v1.3 合同"),
	}, "intent 夹具提交")

	a, err := LoadIntent(rf.dir, "HEAD", "TASK-0023")
	if err != nil {
		t.Fatalf("装载: %v", err)
	}
	if a.SourceCommit != head {
		t.Errorf("来源提交应为 HEAD sha %s, got %s", head, a.SourceCommit)
	}
	if a.Path != ".tally/intents/TASK-0023.yaml" {
		t.Errorf("制品路径应由请求身份派生, got %q", a.Path)
	}
	fi := a.Intent
	if fi.TaskID() != "TASK-0023" || fi.TaskType() != intentevidence.TaskTypeBugFix {
		t.Errorf("冻结意图内容应原样装载, got %+v", fi)
	}
	if fi.SourceRef() != "GitHub Issue #142" {
		t.Errorf("source_ref 应原样装载, got %q", fi.SourceRef())
	}
	if hints := fi.RiskHints(); len(hints) != 1 || hints[0] != "schema_migration" {
		t.Errorf("risk_hints 应原样装载, got %v", hints)
	}
	if fi.Revision() != want.Revision() {
		t.Errorf("Revision 应绑定 Git blob 实际内容: want %s, got %s", want.Revision(), fi.Revision())
	}

	// 同一提交重复装载:逐字段相等、修订一致(确定性)。
	again, err := LoadIntent(rf.dir, head, "TASK-0023")
	if err != nil {
		t.Fatalf("复装载: %v", err)
	}
	if !again.Intent.Equal(fi) || again.Intent.Revision() != fi.Revision() || again.SourceCommit != a.SourceCommit {
		t.Errorf("同一提交复装载应确定, first=%+v second=%+v", fi, again.Intent)
	}

	// 按身份各取各的:TASK-0024 装到自己的 db_migration 证据,不串台。
	other, err := LoadIntent(rf.dir, "HEAD", "TASK-0024")
	if err != nil {
		t.Fatalf("装载 TASK-0024: %v", err)
	}
	if other.Intent.TaskID() != "TASK-0024" || other.Intent.TaskType() != intentevidence.TaskTypeMigration {
		t.Errorf("TASK-0024 应装载自身内容, got %+v", other.Intent)
	}
	if other.Intent.Equal(fi) {
		t.Error("两份不同证据不应相等")
	}
}

// TestLoadIntentReadsCommitTreeNotWorkingTree:Git 是唯一来源(AC1 同纪律)。
// 装载读提交树快照:工作区未提交改动与未跟踪文件不可见;历史修订按提交可
// 复装载(working-tree divergence)。
func TestLoadIntentReadsCommitTreeNotWorkingTree(t *testing.T) {
	rf := initRepo(t)
	v1 := mustFreeze(t, "TASK-0023", intentevidence.TaskTypeBugFix, "issue v1")
	sha1 := rf.commit(t, map[string]string{
		".tally/intents/TASK-0023.yaml": string(intentYAML(t, "TASK-0023", intentevidence.TaskTypeBugFix, "issue v1")),
	}, "intent v1")

	// 未提交:工作区改成 v2,再加一个未跟踪的新 intent 文件。
	writeWorktreeFile(t, rf.dir, ".tally/intents/TASK-0023.yaml", string(intentYAML(t, "TASK-0023", intentevidence.TaskTypeBugFix, "v2 工作区未提交")))
	writeWorktreeFile(t, rf.dir, ".tally/intents/TASK-9999.yaml", string(intentYAML(t, "TASK-9999", intentevidence.TaskTypeBugFix, "未跟踪文件")))

	a, err := LoadIntent(rf.dir, "HEAD", "TASK-0023")
	if err != nil {
		t.Fatalf("装载 HEAD: %v", err)
	}
	if a.Intent.SourceRef() != "issue v1" || a.Intent.Revision() != v1.Revision() {
		t.Errorf("HEAD 装载应读提交内容 v1, 不看工作区, got %+v", a.Intent)
	}
	if _, err := LoadIntent(rf.dir, "HEAD", "TASK-9999"); !errors.Is(err, ErrIntentNotFound) {
		t.Errorf("未跟踪文件不应进入提交树, 应 ErrIntentNotFound, got %v", err)
	}

	// 提交 v2 后:历史修订 sha1 仍装载 v1,HEAD 装载 v2,修订互不漂移。
	v2 := mustFreeze(t, "TASK-0023", intentevidence.TaskTypeBugFix, "issue v2")
	sha2 := rf.commit(t, map[string]string{
		".tally/intents/TASK-0023.yaml": string(intentYAML(t, "TASK-0023", intentevidence.TaskTypeBugFix, "issue v2")),
	}, "intent v2")

	old, err := LoadIntent(rf.dir, sha1, "TASK-0023")
	if err != nil {
		t.Fatalf("装载历史修订: %v", err)
	}
	if old.SourceCommit != sha1 || old.Intent.Revision() != v1.Revision() {
		t.Errorf("历史修订应装载 v1 @ %s, got @%s rev=%s", sha1, old.SourceCommit, old.Intent.Revision())
	}
	newer, err := LoadIntent(rf.dir, "HEAD", "TASK-0023")
	if err != nil {
		t.Fatalf("装载新 HEAD: %v", err)
	}
	if newer.SourceCommit != sha2 || newer.Intent.Revision() != v2.Revision() {
		t.Errorf("HEAD 应装载 v2 @ %s, got @%s rev=%s", sha2, newer.SourceCommit, newer.Intent.Revision())
	}
	if v1.Revision() == v2.Revision() {
		t.Error("内容变化后修订哈希必须变化")
	}
}

// TestLoadIntentNotFound:制品不在提交树里是类型化拒收 ErrIntentNotFound;
// 打不开仓库是基础设施失败(不是 NotFound);请求身份本身非法按
// intentevidence 类型化错误拒收。
func TestLoadIntentNotFound(t *testing.T) {
	rf := initRepo(t)
	rf.commit(t, map[string]string{
		".tally/intents/TASK-0023.yaml": string(intentYAML(t, "TASK-0023", intentevidence.TaskTypeBugFix, "")),
	}, "intent")

	if _, err := LoadIntent(rf.dir, "HEAD", "TASK-0001"); !errors.Is(err, ErrIntentNotFound) {
		t.Errorf("未冻结任务的 intent 应 ErrIntentNotFound, got %v", err)
	}
	if _, err := LoadIntent(rf.dir+"/nope", "HEAD", "TASK-0023"); err == nil || errors.Is(err, ErrIntentNotFound) {
		t.Errorf("打不开的仓库是基础设施失败, 不应归类 ErrIntentNotFound, got %v", err)
	}
	for _, bad := range []string{"", "TASK 0023"} {
		if _, err := LoadIntent(rf.dir, "HEAD", bad); !errors.Is(err, intentevidence.ErrInvalidEvidence) {
			t.Errorf("请求身份 %q 非法应穿透 intentevidence 类型化错误, got %v", bad, err)
		}
	}
}

// TestLoadIntentRejectsInvalidDeterministically:提交树里的 intent 内容非法
// (未知字段、声明 revision、空/多文档、非映射、词表外 task_type、身份含
// 空白)一律类型化拒收 ErrIntentInvalid,intentevidence 哨兵穿透可判。
func TestLoadIntentRejectsInvalidDeterministically(t *testing.T) {
	cases := map[string]string{
		"未知字段":       "task_id: TASK-0023\ntask_type: bug_fix\nmade_up_field: true\n",
		"声明revision": "task_id: TASK-0023\ntask_type: bug_fix\nrevision: sha256:deadbeef\n",
		"空文档":        "",
		"多文档":        "task_id: TASK-0023\ntask_type: bug_fix\n---\ntask_id: TASK-0024\ntask_type: bug_fix\n",
		"非映射":        "- a\n- b\n",
		"词表外类型":      "task_id: TASK-0023\ntask_type: hotfix\n",
		"身份含空白":      "task_id: \"TASK 0023\"\ntask_type: bug_fix\n",
	}
	for name, bad := range cases {
		t.Run(name, func(t *testing.T) {
			rf := initRepo(t)
			rf.commit(t, map[string]string{
				".tally/intents/TASK-0023.yaml": bad,
			}, "坏 intent 夹具")

			a, err := LoadIntent(rf.dir, "HEAD", "TASK-0023")
			if a != nil {
				t.Errorf("非法制品不得返回装载结果, got %+v", a)
			}
			if !errors.Is(err, ErrIntentInvalid) {
				t.Fatalf("应类型化拒收 ErrIntentInvalid, got %v", err)
			}
			if errors.Is(err, ErrIntentNotFound) || errors.Is(err, ErrIntentIdentityMismatch) {
				t.Errorf("内容失效不应误判为不存在/身份不一致, got %v", err)
			}
			if !errors.Is(err, intentevidence.ErrInvalidEvidence) && !errors.Is(err, intentevidence.ErrUnsupportedTaskType) {
				t.Errorf("intentevidence 类型化哨兵应穿透, got %v", err)
			}
		})
	}
}

// TestLoadIntentRejectsIdentityMismatch:制品内容声明的 task_id 与请求/路径
// 身份不一致时按身份漂移拒收 —— 单制品装载没有同侪可对照,任何漂移都
// fail-closed,不猜真身。
func TestLoadIntentRejectsIdentityMismatch(t *testing.T) {
	rf := initRepo(t)
	rf.commit(t, map[string]string{
		".tally/intents/TASK-0001.yaml": string(intentYAML(t, "TASK-0002", intentevidence.TaskTypeMigration, "放错文件的证据")),
	}, "身份漂移夹具")

	a, err := LoadIntent(rf.dir, "HEAD", "TASK-0001")
	if a != nil {
		t.Errorf("身份不一致不得返回装载结果, got %+v", a)
	}
	if !errors.Is(err, ErrIntentIdentityMismatch) {
		t.Fatalf("应类型化拒收 ErrIntentIdentityMismatch, got %v", err)
	}
	if errors.Is(err, ErrIntentInvalid) || errors.Is(err, ErrIntentNotFound) {
		t.Errorf("身份漂移不应误判为内容非法/不存在, got %v", err)
	}
}

// TestLoadIntentUnresolvableRevision:revision 解析不了是基础设施失败,
// 不是三类制品级拒收中的任何一种。
func TestLoadIntentUnresolvableRevision(t *testing.T) {
	rf := initRepo(t)
	rf.commit(t, map[string]string{
		".tally/intents/TASK-0023.yaml": string(intentYAML(t, "TASK-0023", intentevidence.TaskTypeBugFix, "")),
	}, "intent")

	_, err := LoadIntent(rf.dir, "definitely-not-a-revision", "TASK-0023")
	if err == nil {
		t.Fatal("解析不了的 revision 应报错")
	}
	if errors.Is(err, ErrIntentNotFound) || errors.Is(err, ErrIntentInvalid) || errors.Is(err, ErrIntentIdentityMismatch) {
		t.Errorf("基础设施失败不应误判为制品级拒收, got %v", err)
	}
}
