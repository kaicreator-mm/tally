// v141_bundle_loader 的单元测试:复用 v13/v141 夹具与真实的冻结准入+冻结提交
// 生产路径(P0-07/P0-08),钉住同源 bundle 装载的合同 ——
//   - 单一来源:四类权威制品(intent/task/verification/policies)解析自同一
//     pinned 提交,内容与冻结值/独立装载器逐项相等(DL-04,任务 AC);
//   - 身份:bundle 身份 = 一个来源提交 + 全部装载修订的确定性纯函数;换任何
//     一份制品、换来源提交,身份必变(任务 AC);
//   - divergent commits:pinned 提交缺哪份制品就类型化拒收,绝不自动升级到
//     HEAD 或任何其他提交补读(任务 Verification:divergent commits);
//   - 身份漂移/内容非法 fail-closed;基础设施失败与制品级拒收分类清晰。

package artifacts

import (
	"context"
	"errors"
	"testing"

	"tally/internal/engineeringpolicy"
	"tally/internal/intentevidence"
)

// bundlePolicyV1 / bundlePolicyV2 是 bundle 测试的两代策略内容(同一 id,不同
// rationale → 不同内容修订):冻结后换版策略,用于 divergent commits 场景。
const bundlePolicyID = "db_gate"

func bundlePolicyYAML(rationale string) string {
	return policyYAML(bundlePolicyID, rationale)
}

// mustParsePolicy 解析策略夹具,取其内容修订(观测值对照用)。
func mustParsePolicy(t *testing.T, yaml string) engineeringpolicy.Policy {
	t.Helper()
	p, err := engineeringpolicy.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("解析策略夹具: %v", err)
	}
	return p
}

// TestLoadBundleAtFreezeCommit:经生产冻结提交(FreezeCommit)落库的任务,
// bundle 从**同一个**提交把四类权威制品装载回来 —— 三件套与冻结值逐项相等,
// 策略与独立装载器相等,来源提交唯一,bundle 身份确定(任务 AC:同源装载)。
func TestLoadBundleAtFreezeCommit(t *testing.T) {
	rf := initRepo(t)
	polV1 := mustParsePolicy(t, bundlePolicyYAML("冻结时策略"))
	base := rf.commit(t, map[string]string{
		ManifestPath:                   manifestYAML,
		".tally/policies/db_gate.yaml": bundlePolicyYAML("冻结时策略"),
	}, "base: 注册表 + 策略")
	fa := freezeArtifacts(t, "TASK-0042")
	res, err := FreezeCommit(context.Background(), rf.dir, FreezeCommitRequest{
		ExpectedBase: base,
		Artifacts:    fa,
		Message:      "freeze: 冻结 TASK-0042",
		Signature:    &freezeSig,
	})
	if err != nil {
		t.Fatalf("FreezeCommit: %v", err)
	}

	b, err := LoadBundle(rf.dir, res.Commit, "TASK-0042")
	if err != nil {
		t.Fatalf("LoadBundle: %v", err)
	}
	if b.TaskID != "TASK-0042" || b.SourceCommit != res.Commit {
		t.Errorf("bundle 身份轴 = (%s, %s),want (TASK-0042, %s)", b.TaskID, b.SourceCommit, res.Commit)
	}

	// 三件套与冻结值逐项相等:内容重算的修订/哈希必须回到冻结时刻的值。
	if b.Intent.Path != ".tally/intents/TASK-0042.yaml" || b.Intent.SourceCommit != res.Commit {
		t.Errorf("IntentArtifact = %+v", b.Intent)
	}
	if !b.Intent.Intent.Equal(fa.Intent) || b.Intent.Intent.Revision() != fa.Intent.Revision() {
		t.Errorf("intent 应与冻结值相等, got rev=%s want rev=%s", b.Intent.Intent.Revision(), fa.Intent.Revision())
	}
	if b.Task.Path != TasksDir+"/TASK-0042.yaml" || b.Task.Contract.Hash != fa.Task.Hash {
		t.Errorf("task 应与冻结值相等, got hash=%s want %s", b.Task.Contract.Hash, fa.Task.Hash)
	}
	if b.Verification.Path != VerificationsDir+"/TASK-0042.yaml" ||
		b.Verification.Frozen.Hash() != fa.Verification.Hash() ||
		b.Verification.Frozen.Revision() != fa.Verification.Revision() {
		t.Errorf("verification 应与冻结值相等, got hash=%s rev=%d want hash=%s rev=%d",
			b.Verification.Frozen.Hash(), b.Verification.Frozen.Revision(), fa.Verification.Hash(), fa.Verification.Revision())
	}

	// 策略来自同一提交,与独立装载器逐项相等;无效制品没有。
	if n := len(b.Policies.Policies); n != 1 {
		t.Fatalf("bundle 应装载 1 条策略, got %d", n)
	}
	if b.Policies.Policies[0].Policy.Revision != polV1.Revision {
		t.Errorf("policy 修订应绑定本提交内容: got %s want %s", b.Policies.Policies[0].Policy.Revision, polV1.Revision)
	}
	if n := len(b.Policies.Rejected); n != 0 {
		t.Errorf("不应有被拒策略, got %v", b.Policies.Rejected)
	}
	lp, err := LoadPolicies(rf.dir, res.Commit)
	if err != nil {
		t.Fatalf("LoadPolicies: %v", err)
	}
	if lp.Policies[0].Policy.Revision != b.Policies.Policies[0].Policy.Revision || lp.SourceCommit != b.Policies.SourceCommit {
		t.Errorf("bundle 策略应与 LoadPolicies 一致: bundle=%s@%s policies=%s@%s",
			b.Policies.Policies[0].Policy.Revision, b.Policies.SourceCommit, lp.Policies[0].Policy.Revision, lp.SourceCommit)
	}
	ia, err := LoadIntent(rf.dir, res.Commit, "TASK-0042")
	if err != nil {
		t.Fatalf("LoadIntent: %v", err)
	}
	if !ia.Intent.Equal(b.Intent.Intent) || ia.SourceCommit != b.Intent.SourceCommit {
		t.Errorf("bundle intent 应与 LoadIntent 一致")
	}

	// 身份轴(任务 AC):Revisions 覆盖全部装载修订;同一提交重复装载得到
	// 同一 bundle 身份。
	for _, want := range []string{
		"intent:" + fa.Intent.Revision(),
		"task:" + fa.Task.Hash,
		"verification:" + fa.Verification.Hash(),
		"verification_revision:1",
		"policy:" + polV1.Revision,
	} {
		if !containsString(b.Revisions(), want) {
			t.Errorf("Revisions() 应含 %s, got %v", want, b.Revisions())
		}
	}
	if b.Identity() == "" {
		t.Error("bundle 身份不应为空")
	}
	again, err := LoadBundle(rf.dir, res.Commit, "TASK-0042")
	if err != nil {
		t.Fatalf("复装载: %v", err)
	}
	if again.Identity() != b.Identity() {
		t.Errorf("同一提交复装载身份应确定: %s vs %s", again.Identity(), b.Identity())
	}
}

// TestLoadBundlePinnedCommitNeverUpgrades:divergent commits 场景 —— 冻结之后
// 策略换版、仓库继续推进。bundle 钉在哪个提交就读哪个提交的树:pinned 提交里
// 缺的制品类型化拒收,绝不自动升级到 HEAD 补读(任务 Verification:divergent
// commits);bundle 身份同时覆盖来源提交与全部修订 —— 内容相同、来源提交不同
// 的两个 bundle 身份必不同。
func TestLoadBundlePinnedCommitNeverUpgrades(t *testing.T) {
	rf := initRepo(t)
	polV1 := mustParsePolicy(t, bundlePolicyYAML("冻结时策略"))
	commit1 := rf.commit(t, map[string]string{
		".tally/policies/db_gate.yaml": bundlePolicyYAML("冻结时策略"),
	}, "策略 v1")
	fa := freezeArtifacts(t, "TASK-0042")
	res, err := FreezeCommit(context.Background(), rf.dir, FreezeCommitRequest{
		ExpectedBase: commit1,
		Artifacts:    fa,
		Message:      "freeze: 冻结 TASK-0042",
		Signature:    &freezeSig,
	})
	if err != nil {
		t.Fatalf("FreezeCommit: %v", err)
	}
	commit2 := res.Commit
	commit3 := rf.commit(t, map[string]string{
		".tally/policies/db_gate.yaml": bundlePolicyYAML("冻结后换版策略"),
	}, "策略 v2")
	commit4 := rf.commit(t, map[string]string{"src/main.go": "package main\n"}, "无关推进")

	// 钉在冻结提交:三件套是冻结值,策略是冻结时的 v1 —— HEAD 上的 v2 不可见。
	b2, err := LoadBundle(rf.dir, commit2, "TASK-0042")
	if err != nil {
		t.Fatalf("LoadBundle(%s): %v", commit2, err)
	}
	if b2.SourceCommit != commit2 {
		t.Errorf("来源提交 = %s, want %s", b2.SourceCommit, commit2)
	}
	if b2.Policies.Policies[0].Policy.Revision != polV1.Revision {
		t.Errorf("冻结提交上的策略应是 v1, got %s", b2.Policies.Policies[0].Policy.Revision)
	}
	if !b2.Intent.Intent.Equal(fa.Intent) {
		t.Error("冻结提交上的 intent 应是冻结值, 不看后续提交")
	}

	// 钉在冻结之前:三件套尚不存在 → 类型化不存在,绝不自动升级到 HEAD。
	if _, err := LoadBundle(rf.dir, commit1, "TASK-0042"); !errors.Is(err, ErrIntentNotFound) {
		t.Errorf("冻结前提交应 ErrIntentNotFound(不自动升级), got %v", err)
	}

	// 钉在策略换版之后:三件套内容未变,策略是 v2;bundle 身份因修订变化而变。
	b3, err := LoadBundle(rf.dir, commit3, "TASK-0042")
	if err != nil {
		t.Fatalf("LoadBundle(%s): %v", commit3, err)
	}
	polV2 := mustParsePolicy(t, bundlePolicyYAML("冻结后换版策略"))
	if b3.Policies.Policies[0].Policy.Revision != polV2.Revision {
		t.Errorf("commit3 的策略应是 v2, got %s", b3.Policies.Policies[0].Policy.Revision)
	}
	if b3.Identity() == b2.Identity() {
		t.Error("策略换版后 bundle 身份必须变化")
	}

	// 内容与 commit3 完全相同、只差无关提交:修订全同,但来源提交不同,
	// 身份仍必变 —— bundle 身份包含来源提交本身(任务 AC)。
	b4, err := LoadBundle(rf.dir, commit4, "TASK-0042")
	if err != nil {
		t.Fatalf("LoadBundle(%s): %v", commit4, err)
	}
	if b4.SourceCommit != commit4 {
		t.Errorf("来源提交 = %s, want %s", b4.SourceCommit, commit4)
	}
	if len(b4.Revisions()) != len(b3.Revisions()) {
		t.Fatalf("无关提交不应改变修订集: %v vs %v", b4.Revisions(), b3.Revisions())
	}
	for i := range b4.Revisions() {
		if b4.Revisions()[i] != b3.Revisions()[i] {
			t.Errorf("无关提交不应改变修订集[%d]: %s vs %s", i, b4.Revisions()[i], b3.Revisions()[i])
		}
	}
	if b4.Identity() == b3.Identity() {
		t.Error("来源提交不同则 bundle 身份必须不同")
	}
}

// TestLoadBundleTypedRejections:各制品类的缺失/内容非法/身份漂移都类型化
// 拒收且互不误判;基础设施失败(解析不了的 revision)与非法请求身份按既有
// 分类穿透。装载顺序 intent → task → verification 固定。
func TestLoadBundleTypedRejections(t *testing.T) {
	t.Run("缺task", func(t *testing.T) {
		rf := initRepo(t)
		rf.commit(t, map[string]string{
			".tally/intents/TASK-0042.yaml": string(intentYAML(t, "TASK-0042", intentevidence.TaskTypeBugFix, "")),
		}, "只有 intent")
		_, err := LoadBundle(rf.dir, "HEAD", "TASK-0042")
		if !errors.Is(err, ErrTaskNotFound) {
			t.Errorf("缺 task 应 ErrTaskNotFound, got %v", err)
		}
		if errors.Is(err, ErrIntentNotFound) || errors.Is(err, ErrVerificationNotFound) {
			t.Errorf("不应误判为其他制品缺失, got %v", err)
		}
	})
	t.Run("缺verification", func(t *testing.T) {
		rf := initRepo(t)
		rf.commit(t, map[string]string{
			".tally/intents/TASK-0042.yaml": string(intentYAML(t, "TASK-0042", intentevidence.TaskTypeBugFix, "")),
			".tally/tasks/TASK-0042.yaml":   taskYAML("TASK-0042", "缺验证契约"),
		}, "intent + task")
		_, err := LoadBundle(rf.dir, "HEAD", "TASK-0042")
		if !errors.Is(err, ErrVerificationNotFound) {
			t.Errorf("缺 verification 应 ErrVerificationNotFound, got %v", err)
		}
	})
	t.Run("缺intent", func(t *testing.T) {
		rf := initRepo(t)
		rf.commit(t, map[string]string{
			".tally/tasks/TASK-0042.yaml":         taskYAML("TASK-0042", "缺意图证据"),
			".tally/verifications/TASK-0042.yaml": verificationYAML("TASK-0042", 1),
		}, "task + verification")
		_, err := LoadBundle(rf.dir, "HEAD", "TASK-0042")
		if !errors.Is(err, ErrIntentNotFound) {
			t.Errorf("缺 intent 应 ErrIntentNotFound, got %v", err)
		}
	})
	t.Run("task身份漂移", func(t *testing.T) {
		rf := initRepo(t)
		rf.commit(t, map[string]string{
			".tally/intents/TASK-0001.yaml": string(intentYAML(t, "TASK-0001", intentevidence.TaskTypeBugFix, "")),
			".tally/tasks/TASK-0001.yaml":   taskYAML("TASK-0002", "放错文件的合同"),
		}, "task 身份漂移")
		_, err := LoadBundle(rf.dir, "HEAD", "TASK-0001")
		if !errors.Is(err, ErrTaskIdentityMismatch) {
			t.Errorf("task 身份漂移应 ErrTaskIdentityMismatch, got %v", err)
		}
		if errors.Is(err, ErrTaskInvalid) || errors.Is(err, ErrTaskNotFound) {
			t.Errorf("身份漂移不应误判为内容非法/不存在, got %v", err)
		}
	})
	t.Run("verification身份漂移", func(t *testing.T) {
		rf := initRepo(t)
		rf.commit(t, map[string]string{
			".tally/intents/TASK-0001.yaml":       string(intentYAML(t, "TASK-0001", intentevidence.TaskTypeBugFix, "")),
			".tally/tasks/TASK-0001.yaml":         taskYAML("TASK-0001", "验证契约漂移"),
			".tally/verifications/TASK-0001.yaml": verificationYAML("TASK-0002", 1),
		}, "verification 身份漂移")
		_, err := LoadBundle(rf.dir, "HEAD", "TASK-0001")
		if !errors.Is(err, ErrVerificationIdentityMismatch) {
			t.Errorf("verification 身份漂移应 ErrVerificationIdentityMismatch, got %v", err)
		}
	})
	t.Run("task内容非法", func(t *testing.T) {
		rf := initRepo(t)
		rf.commit(t, map[string]string{
			".tally/intents/TASK-0002.yaml": string(intentYAML(t, "TASK-0002", intentevidence.TaskTypeBugFix, "")),
			".tally/tasks/TASK-0002.yaml":   brokenTaskYAML, // 缺 objective,Validate 必拒
		}, "坏 task")
		_, err := LoadBundle(rf.dir, "HEAD", "TASK-0002")
		if !errors.Is(err, ErrTaskInvalid) {
			t.Errorf("坏 task 应 ErrTaskInvalid, got %v", err)
		}
	})
	t.Run("verification内容非法", func(t *testing.T) {
		rf := initRepo(t)
		rf.commit(t, map[string]string{
			".tally/intents/TASK-0003.yaml":       string(intentYAML(t, "TASK-0003", intentevidence.TaskTypeBugFix, "")),
			".tally/tasks/TASK-0003.yaml":         taskYAML("TASK-0003", "坏验证契约"),
			".tally/verifications/TASK-0003.yaml": brokenVerificationYAML, // revision 0,Validate 必拒
		}, "坏 verification")
		_, err := LoadBundle(rf.dir, "HEAD", "TASK-0003")
		if !errors.Is(err, ErrVerificationInvalid) {
			t.Errorf("坏 verification 应 ErrVerificationInvalid, got %v", err)
		}
		if errors.Is(err, ErrVerificationNotFound) || errors.Is(err, ErrVerificationIdentityMismatch) {
			t.Errorf("内容失效不应误判为不存在/身份不一致, got %v", err)
		}
	})
	t.Run("基础设施失败不是制品拒收", func(t *testing.T) {
		rf := initRepo(t)
		rf.commit(t, map[string]string{
			".tally/intents/TASK-0042.yaml": string(intentYAML(t, "TASK-0042", intentevidence.TaskTypeBugFix, "")),
		}, "intent")
		_, err := LoadBundle(rf.dir, "definitely-not-a-revision", "TASK-0042")
		if err == nil {
			t.Fatal("解析不了的 revision 应报错")
		}
		for _, sentinel := range []error{ErrIntentNotFound, ErrTaskNotFound, ErrVerificationNotFound,
			ErrIntentInvalid, ErrTaskInvalid, ErrVerificationInvalid,
			ErrIntentIdentityMismatch, ErrTaskIdentityMismatch, ErrVerificationIdentityMismatch} {
			if errors.Is(err, sentinel) {
				t.Errorf("基础设施失败不应归类 %v, got %v", sentinel, err)
			}
		}
	})
	t.Run("非法请求身份穿透intentevidence", func(t *testing.T) {
		rf := initRepo(t)
		for _, bad := range []string{"", "TASK 0042"} {
			if _, err := LoadBundle(rf.dir, "HEAD", bad); !errors.Is(err, intentevidence.ErrInvalidEvidence) {
				t.Errorf("请求身份 %q 非法应穿透 intentevidence.ErrInvalidEvidence, got %v", bad, err)
			}
		}
	})
}

// TestLoadBundlePolicyRejectionsCarriedNotFatal:策略装载沿用 LoadPolicies 的
// 快照纪律 —— 无效策略逐文件拒收进 PolicySnapshot.Rejected,不中断 bundle
// 装载(是否阻断执行由消费方 P1-03 装配器按 fail-safe 裁决);一个策略都没有
// 的提交同样可装载(冻结三件套本身不含策略)。
func TestLoadBundlePolicyRejectionsCarriedNotFatal(t *testing.T) {
	rf := initRepo(t)
	base := rf.commit(t, map[string]string{
		".tally/policies/db_gate.yaml":     bundlePolicyYAML("有效策略"),
		".tally/policies/broken_gate.yaml": gateWithoutConditionYAML, // gate 无条件,必拒
	}, "一好一坏策略")
	fa := freezeArtifacts(t, "TASK-0042")
	res, err := FreezeCommit(context.Background(), rf.dir, FreezeCommitRequest{
		ExpectedBase: base,
		Artifacts:    fa,
		Message:      "freeze: 冻结 TASK-0042",
		Signature:    &freezeSig,
	})
	if err != nil {
		t.Fatalf("FreezeCommit: %v", err)
	}

	b, err := LoadBundle(rf.dir, res.Commit, "TASK-0042")
	if err != nil {
		t.Fatalf("LoadBundle: %v", err)
	}
	if n := len(b.Policies.Policies); n != 1 || b.Policies.Policies[0].Path != ".tally/policies/db_gate.yaml" {
		t.Errorf("有效策略应入列, got %+v", b.Policies.Policies)
	}
	if n := len(b.Policies.Rejected); n != 1 || b.Policies.Rejected[0].Kind != KindPolicy ||
		b.Policies.Rejected[0].Path != ".tally/policies/broken_gate.yaml" {
		t.Errorf("坏策略应逐文件拒收, got %+v", b.Policies.Rejected)
	}

	// 没有任何策略目录的仓库同样可装载(冻结三件套 + 零策略)。
	rf2 := initRepo(t)
	fa2 := freezeArtifacts(t, "TASK-0043")
	res2, err := FreezeCommit(context.Background(), rf2.dir, FreezeCommitRequest{
		Artifacts: fa2,
		Message:   "freeze: 冻结 TASK-0043(无策略)",
		Signature: &freezeSig,
	})
	if err != nil {
		t.Fatalf("FreezeCommit(无策略): %v", err)
	}
	b2, err := LoadBundle(rf2.dir, res2.Commit, "TASK-0043")
	if err != nil {
		t.Fatalf("LoadBundle(无策略): %v", err)
	}
	if n := len(b2.Policies.Policies); n != 0 {
		t.Errorf("无策略提交应装载零策略, got %d", n)
	}
	if b2.Identity() == "" {
		t.Error("零策略 bundle 仍应有身份")
	}
}
