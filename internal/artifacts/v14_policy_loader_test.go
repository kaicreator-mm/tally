// v14_policy_loader 的单元测试:复用 v13_loader_test.go 的 git 仓库夹具,
// 验证 EngineeringPolicy 装载的 Git 快照语义(工作区不可见、历史修订可复装载、
// AC1)、无效策略的逐文件确定性拒收(AC2)、重复策略 id 的整体拒绝(AC3)、
// 修订哈希绑定 Git 实际内容(防篡改/防漂移),以及与 v1.3 task/verification
// 装载互不干扰(AC4)。

package artifacts

import (
	"strings"
	"testing"

	"tally/internal/engineeringpolicy"
)

// policyYAML 生成一份合法的 gate EngineeringPolicy 夹具(通过
// engineeringpolicy.Parse + Validate)。
func policyYAML(id, rationale string) string {
	return `id: ` + id + `
enforcement: gate
rationale: ` + rationale + `
trigger:
  risk_signals:
    - schema_migration
  paths:
    - "migrations/**"
requires:
  - fresh_database
  - data_integrity
gate_condition:
  all_of:
    - fresh_database
    - data_integrity
bundle_tags:
  - database
`
}

// guidancePolicyYAML 合法的 guidance 策略:不带 gate_condition(PRD §8.2,
// guidance 无阻断权)。
const guidancePolicyYAML = `id: testing_guidance
enforcement: guidance
rationale: 鼓励为变更补测试
trigger:
  paths:
    - "src/**"
requires:
  - unit_tests_cover_change
`

// 下列夹具各有一种确定性非法形态,engineeringpolicy.Parse 必拒(AC2)。
const (
	// gate 无 gate_condition:无条件门禁等于静默放行。
	gateWithoutConditionYAML = `id: bad_gate
enforcement: gate
requires:
  - fresh_database
`
	// guidance 携带 gate_condition:guidance 在 v1.4 拿不到阻断权。
	guidanceWithGateYAML = `id: bad_guidance
enforcement: guidance
requires:
  - fresh_database
gate_condition:
  all_of:
    - fresh_database
`
	// gate_condition 引用 requires[] 之外的键:门禁不得要求未事先告知的条件。
	gateUnknownKeyYAML = `id: unknown_key_gate
enforcement: gate
requires:
  - fresh_database
gate_condition:
  all_of:
    - data_integrity
`
	// schema 之外的未知字段。
	unknownFieldPolicyYAML = `id: unknown_field_policy
enforcement: gate
requires:
  - fresh_database
gate_condition:
  all_of:
    - fresh_database
made_up_field: true
`
	// 执行策略字段:决策锁 6,ExecutionPolicy != EngineeringPolicy。
	execFieldPolicyYAML = `id: exec_field_policy
enforcement: gate
requires:
  - fresh_database
gate_condition:
  all_of:
    - fresh_database
network:
  mode: offline
`
	// 声称 revision:revision 只能从内容观测(schema 里 Revision 是 yaml:"-",
	// desired 声明不得冒充观测事实)。
	claimedRevisionPolicyYAML = `id: claimed_revision_policy
enforcement: gate
revision: sha256:deadbeef
requires:
  - fresh_database
gate_condition:
  all_of:
    - fresh_database
`
	// 多文档与空文档。
	multiDocPolicyYAML = `id: policy_a
enforcement: guidance
---
id: policy_b
enforcement: guidance
`
	emptyPolicyYAML = ``
)

// policyRejectedPaths 提取策略装载拒收清单里的路径。
func policyRejectedPaths(snap *PolicySnapshot) []string {
	paths := make([]string, 0, len(snap.Rejected))
	for _, r := range snap.Rejected {
		paths = append(paths, r.Path)
	}
	return paths
}

// policyPaths 提取有效策略制品的路径。
func policyPaths(policies []PolicyArtifact) []string {
	paths := make([]string, 0, len(policies))
	for _, a := range policies {
		paths = append(paths, a.Path)
	}
	return paths
}

// TestLoadPoliciesAtHead:一次装载读全策略快照 —— 有效策略入列、无效策略逐文件
// 拒收、无关路径视而不见、来源提交可追溯、输出确定有序(AC1/AC2)。
func TestLoadPoliciesAtHead(t *testing.T) {
	rf := initRepo(t)
	head := rf.commit(t, map[string]string{
		"src/main.go":                           "package main\n",
		".tally/README.md":                      "# 不是策略\n",
		".tally/policies/notes.txt":             "不是 yaml",
		".tally/policies/POLICY-0001.yml":       "扩展名不对,不属于制品集\n",
		".tally/policies/sub/POLICY-0002.yaml":  "子目录里的文件不属于扁平制品集\n",
		".tally/policies/migration_gate.yaml":   policyYAML("migration_gate", "迁移必须全量验证"),
		".tally/policies/testing_guidance.yaml": guidancePolicyYAML,
		".tally/policies/broken.yaml":           gateWithoutConditionYAML,
	}, "策略夹具提交")

	snap, err := LoadPolicies(rf.dir, "HEAD")
	if err != nil {
		t.Fatalf("装载: %v", err)
	}
	if snap.SourceCommit != head {
		t.Errorf("来源提交应为 HEAD sha %s, got %s", head, snap.SourceCommit)
	}

	// 有效策略两份,按 Path 排序;无关路径(src/、README、notes.txt、.yml、子目录)不入列。
	got := policyPaths(snap.Policies)
	want := []string{".tally/policies/migration_gate.yaml", ".tally/policies/testing_guidance.yaml"}
	if len(got) != len(want) {
		t.Fatalf("有效策略应为 %v, got %v", want, got)
	}
	for i, p := range want {
		if got[i] != p {
			t.Errorf("策略[%d] 应为 %s, got %s", i, p, got[i])
		}
	}
	gate := snap.Policies[0].Policy
	if gate.ID != "migration_gate" || gate.Enforcement != "gate" || len(gate.GateCondition.AllOf) != 2 {
		t.Errorf("策略内容应原样装载, got %+v", gate)
	}
	if !strings.HasPrefix(gate.Revision, "sha256:") {
		t.Errorf("装载应填充内容修订哈希, got %q", gate.Revision)
	}

	// 无效策略逐文件拒收:带类型、理由可读、且是 Parse 校验链的确定性文案。
	if len(snap.Rejected) != 1 {
		t.Fatalf("应恰好拒收 1 份, got %v", policyRejectedPaths(snap))
	}
	r := snap.Rejected[0]
	if r.Path != ".tally/policies/broken.yaml" || r.Kind != KindPolicy {
		t.Errorf("拒收应指向 broken.yaml 且类型为 %s, got %+v", KindPolicy, r)
	}
	if !strings.HasPrefix(r.Reason, "engineeringpolicy: invalid policy") {
		t.Errorf("拒收理由应来自确定性校验链, got %q", r.Reason)
	}
}

// TestLoadPoliciesRejectsInvalidDeterministically:每种非法形态都被确定性拒收
// 于装载结果之外(不返回 error,不污染同快照里的有效策略)(AC2)。
func TestLoadPoliciesRejectsInvalidDeterministically(t *testing.T) {
	cases := map[string]string{
		"gate无条件":     gateWithoutConditionYAML,
		"guidance带门禁": guidanceWithGateYAML,
		"门禁引用未知键":     gateUnknownKeyYAML,
		"未知字段":        unknownFieldPolicyYAML,
		"执行策略字段":      execFieldPolicyYAML,
		"声称revision":  claimedRevisionPolicyYAML,
		"多文档":         multiDocPolicyYAML,
		"空文档":         emptyPolicyYAML,
	}
	for name, bad := range cases {
		t.Run(name, func(t *testing.T) {
			rf := initRepo(t)
			rf.commit(t, map[string]string{
				".tally/policies/good.yaml": policyYAML("good_gate", "好策略"),
				".tally/policies/bad.yaml":  bad,
			}, "坏策略夹具")

			snap, err := LoadPolicies(rf.dir, "HEAD")
			if err != nil {
				t.Fatalf("内容失效是逐文件拒收, 不是装载失败: %v", err)
			}
			if got := policyPaths(snap.Policies); len(got) != 1 || got[0] != ".tally/policies/good.yaml" {
				t.Errorf("有效策略不应被坏策略牵连, got %v", got)
			}
			if len(snap.Rejected) != 1 || snap.Rejected[0].Path != ".tally/policies/bad.yaml" {
				t.Fatalf("应恰好拒收 bad.yaml, got %v", policyRejectedPaths(snap))
			}
			if snap.Rejected[0].Kind != KindPolicy {
				t.Errorf("拒收应带类型 %s, got %q", KindPolicy, snap.Rejected[0].Kind)
			}
			if !strings.HasPrefix(snap.Rejected[0].Reason, "engineeringpolicy: invalid policy") {
				t.Errorf("拒收理由应是确定性校验文案, got %q", snap.Rejected[0].Reason)
			}
		})
	}
}

// TestLoadPoliciesReadsCommitTreeNotWorkingTree:Git 是唯一来源(AC1)。装载读
// 提交树快照:工作区未提交改动与未跟踪文件不可见;历史修订按提交可复装载。
func TestLoadPoliciesReadsCommitTreeNotWorkingTree(t *testing.T) {
	rf := initRepo(t)
	sha1 := rf.commit(t, map[string]string{
		".tally/policies/migration_gate.yaml": policyYAML("migration_gate", "迁移 v1"),
	}, "策略 v1")

	// 未提交:把工作区改成 v2,再加一个未跟踪的新策略文件。
	writeWorktreeFile(t, rf.dir, ".tally/policies/migration_gate.yaml", policyYAML("migration_gate", "迁移 v2 工作区未提交"))
	writeWorktreeFile(t, rf.dir, ".tally/policies/untracked.yaml", policyYAML("untracked", "未跟踪文件"))

	snap, err := LoadPolicies(rf.dir, "HEAD")
	if err != nil {
		t.Fatalf("装载 HEAD: %v", err)
	}
	if len(snap.Policies) != 1 {
		t.Fatalf("未跟踪文件不应入快照, got %v", policyPaths(snap.Policies))
	}
	if got := snap.Policies[0].Policy.Rationale; got != "迁移 v1" {
		t.Errorf("HEAD 装载应读提交内容 v1, 不看工作区, got %q", got)
	}

	// 提交 v2 后:历史修订 sha1 仍装载 v1,HEAD 装载 v2。
	sha2 := rf.commit(t, map[string]string{
		".tally/policies/migration_gate.yaml": policyYAML("migration_gate", "迁移 v2"),
	}, "策略 v2")

	old, err := LoadPolicies(rf.dir, sha1)
	if err != nil {
		t.Fatalf("装载历史修订: %v", err)
	}
	if got := old.Policies[0].Policy.Rationale; got != "迁移 v1" {
		t.Errorf("历史修订应装载 v1, got %q", got)
	}
	if old.SourceCommit != sha1 {
		t.Errorf("历史装载的来源提交应为 %s, got %s", sha1, old.SourceCommit)
	}
	newer, err := LoadPolicies(rf.dir, "HEAD")
	if err != nil {
		t.Fatalf("装载新 HEAD: %v", err)
	}
	if got := newer.Policies[0].Policy.Rationale; got != "迁移 v2" {
		t.Errorf("HEAD 应装载 v2, got %q", got)
	}
	if newer.SourceCommit != sha2 {
		t.Errorf("来源提交应为 %s, got %s", sha2, newer.SourceCommit)
	}
}

// TestLoadPoliciesRejectsDuplicateIDs:同一快照里同一策略 id 被多个文件声明时,
// 冲突组整体拒绝 —— 当前投影必须唯一,不按文件名或内容猜真身(AC3)。
func TestLoadPoliciesRejectsDuplicateIDs(t *testing.T) {
	rf := initRepo(t)
	rf.commit(t, map[string]string{
		".tally/policies/migration_gate.yaml":     policyYAML("migration_gate", "真身"),
		".tally/policies/migration_gate.dup.yaml": policyYAML("migration_gate", "重复声明"),
		".tally/policies/testing_guidance.yaml":   guidancePolicyYAML,
	}, "id 冲突夹具")

	snap, err := LoadPolicies(rf.dir, "HEAD")
	if err != nil {
		t.Fatalf("装载: %v", err)
	}
	got := policyPaths(snap.Policies)
	if len(got) != 1 || got[0] != ".tally/policies/testing_guidance.yaml" {
		t.Fatalf("id 冲突的策略应整体拒绝,其余照常装载, got %v", got)
	}
	wantRejected := []string{
		".tally/policies/migration_gate.dup.yaml",
		".tally/policies/migration_gate.yaml",
	}
	gotRejected := policyRejectedPaths(snap)
	if len(gotRejected) != len(wantRejected) {
		t.Fatalf("拒收清单应为 %v, got %v", wantRejected, gotRejected)
	}
	for _, p := range wantRejected {
		if !containsString(gotRejected, p) {
			t.Errorf("拒收清单缺 %s, got %v", p, gotRejected)
		}
	}
	for _, r := range snap.Rejected {
		if !strings.Contains(r.Reason, "策略 id migration_gate 由多个文件声明") {
			t.Errorf("拒收理由应点名冲突 id, got %q", r.Reason)
		}
	}
}

// TestLoadPoliciesRevisionBindsToGitContent:装载结果的 Revision 由 Git blob 的
// 实际内容推导,声明/陈旧 revision 无法经装载进入下游 —— 内容被篡改必然得到不同
// revision,历史修订的 revision 不随 HEAD 漂移。
func TestLoadPoliciesRevisionBindsToGitContent(t *testing.T) {
	rf := initRepo(t)
	sha1 := rf.commit(t, map[string]string{
		".tally/policies/migration_gate.yaml": policyYAML("migration_gate", "迁移 v1"),
	}, "策略 v1")

	v1, err := LoadPolicies(rf.dir, sha1)
	if err != nil {
		t.Fatalf("装载 v1: %v", err)
	}
	rev1 := v1.Policies[0].Policy.Revision
	if !strings.HasPrefix(rev1, "sha256:") {
		t.Fatalf("revision 应形如 sha256:<hex>, got %q", rev1)
	}

	// 篡改内容(同 id,改 rationale)后提交:v2 的 revision 必须不同,
	// 且等于 v2 内容重算的修订哈希;旧提交的 revision 不漂移。
	sha2 := rf.commit(t, map[string]string{
		".tally/policies/migration_gate.yaml": policyYAML("migration_gate", "迁移 v2 被篡改"),
	}, "策略 v2")

	v2, err := LoadPolicies(rf.dir, "HEAD")
	if err != nil {
		t.Fatalf("装载 v2: %v", err)
	}
	rev2 := v2.Policies[0].Policy.Revision
	if v2.SourceCommit != sha2 {
		t.Errorf("v2 装载的来源提交应为 %s, got %s", sha2, v2.SourceCommit)
	}
	if rev2 == rev1 {
		t.Errorf("内容被篡改后 revision 必须变化, v1=%s v2=%s", rev1, rev2)
	}
	recomputedPolicy, err := engineeringpolicy.Parse([]byte(policyYAML("migration_gate", "迁移 v2 被篡改")))
	if err != nil {
		t.Fatalf("重算 v2 哈希: %v", err)
	}
	recomputed := recomputedPolicy.Revision
	if rev2 != recomputed {
		t.Errorf("装载的 revision 应绑定 Git 实际内容: want %s, got %s", recomputed, rev2)
	}

	reloadedV1, err := LoadPolicies(rf.dir, sha1)
	if err != nil {
		t.Fatalf("复装载 v1: %v", err)
	}
	if got := reloadedV1.Policies[0].Policy.Revision; got != rev1 {
		t.Errorf("历史修订的 revision 不应漂移: want %s, got %s", rev1, got)
	}
}

// TestLoadPoliciesUnresolvableRevision:仓库打不开/revision 解析不了是装载失败,
// 不是空快照。
func TestLoadPoliciesUnresolvableRevision(t *testing.T) {
	rf := initRepo(t)
	rf.commit(t, map[string]string{
		".tally/policies/migration_gate.yaml": policyYAML("migration_gate", "迁移"),
	}, "策略")

	if _, err := LoadPolicies(rf.dir, "definitely-not-a-revision"); err == nil {
		t.Fatal("解析不了的 revision 应报错")
	}
	if _, err := LoadPolicies(rf.dir+"/nope", "HEAD"); err == nil {
		t.Fatal("打不开的仓库应报错")
	}
}

// TestLoadAndLoadPoliciesIndependent:v1.4 策略装载与 v1.3 task/verification
// 装载互不干扰(AC4):v13 Load 对 .tally/policies 视而不见,LoadPolicies 对
// tasks/verifications 视而不见,同一仓库两条路径各自装载成功。
func TestLoadAndLoadPoliciesIndependent(t *testing.T) {
	rf := initRepo(t)
	rf.commit(t, map[string]string{
		ManifestPath:                          manifestYAML,
		".tally/tasks/TASK-0001.yaml":         taskYAML("TASK-0001", "目标一"),
		".tally/verifications/TASK-0001.yaml": verificationYAML("TASK-0001", 1),
		".tally/policies/migration_gate.yaml": policyYAML("migration_gate", "迁移必须全量验证"),
	}, "混合制品提交")

	v13, err := Load(rf.dir, "HEAD")
	if err != nil {
		t.Fatalf("v13 装载: %v", err)
	}
	if got := pathsOf(v13.Tasks); len(got) != 1 || got[0] != ".tally/tasks/TASK-0001.yaml" {
		t.Errorf("v13 装载应只有 TASK-0001, got %v", got)
	}
	if len(v13.Verifications) != 1 {
		t.Errorf("v13 装载应只有 1 份验证契约, got %d", len(v13.Verifications))
	}
	if len(v13.Rejected) != 0 {
		t.Errorf("v13 装载不应把策略文件当坏合同拒收, got %v", rejectedPaths(v13))
	}

	pol, err := LoadPolicies(rf.dir, "HEAD")
	if err != nil {
		t.Fatalf("v14 策略装载: %v", err)
	}
	if got := policyPaths(pol.Policies); len(got) != 1 || got[0] != ".tally/policies/migration_gate.yaml" {
		t.Errorf("v14 装载应只有 migration_gate, got %v", got)
	}
	if len(pol.Rejected) != 0 {
		t.Errorf("v14 装载不应把 task/verification 当坏策略拒收, got %v", policyRejectedPaths(pol))
	}
}
