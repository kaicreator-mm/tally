package contract

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"tally/internal/taskcontract"
	"tally/internal/verification"
)

// taskYAML 一份能通过 taskcontract.Parse 的标准任务合同(TASK-001)。
const taskYAML = `
id: TASK-001
profile: standard
title: ContractSnapshot aggregate
objective: pin both source contract revisions into one aggregate
scope:
  allowed_paths:
    - internal/contract/**
  forbidden_paths:
    - internal/taskcontract/**
acceptance:
  - id: AC-01
    statement: snapshot pins the task revision
    independence_required: none
  - id: AC-02
    statement: snapshot pins the frozen verification revision
    independence_required: none
`

// taskYAMLRewritten 同一内容的另一种排版:条款换位、加引号、流式列表。
// taskcontract 的修订哈希与键序/条款顺序/排版无关,因此两个 YAML 必须钉住
// 同一修订、聚合出同一 snapshot 身份(P1-01 AC3)。
const taskYAMLRewritten = `
id: "TASK-001"
profile: "standard"
title: "ContractSnapshot aggregate"
objective: "pin both source contract revisions into one aggregate"
scope:
  allowed_paths: ["internal/contract/**"]
  forbidden_paths: ["internal/taskcontract/**"]
acceptance:
  - {id: AC-02, statement: "snapshot pins the frozen verification revision", independence_required: none}
  - {id: AC-01, statement: "snapshot pins the task revision", independence_required: none}
`

// taskYAMLOtherContent 同 id 不同实质内容(objective 不同):Open 时必须以
// pin 哈希不符被拒 —— id 相同救不了内容漂移。
const taskYAMLOtherContent = `
id: TASK-001
profile: standard
title: ContractSnapshot aggregate
objective: a different objective means a different revision
scope:
  allowed_paths:
    - internal/contract/**
acceptance:
  - id: AC-01
    statement: snapshot pins the task revision
    independence_required: none
  - id: AC-02
    statement: snapshot pins the frozen verification revision
    independence_required: none
`

// mustTask 解析任务合同 YAML,失败即测试失败。
func mustTask(t *testing.T, data string) taskcontract.TaskContract {
	t.Helper()
	c, err := taskcontract.Parse([]byte(data))
	if err != nil {
		t.Fatalf("taskcontract.Parse: %v", err)
	}
	return c
}

// mustFrozen 生成一份能通过 P1 standard path freeze 的最小冻结验证契约。
func mustFrozen(t *testing.T, taskID string, revision int) *verification.Frozen {
	t.Helper()
	c := &verification.Contract{
		TaskID:   taskID,
		Revision: revision,
		Acceptances: []verification.Acceptance{{
			ID:        "AC-01",
			Statement: "visible build gate passes",
			EvidenceBindings: []verification.Binding{{
				ID:               "B-1",
				AuthoringMode:    verification.AuthoringPreCommit,
				VisibilityPolicy: verification.VisibilityVisible,
				Evidence:         verification.EvidenceRef{Type: verification.EvidenceBuild, Path: "Makefile"},
			}},
		}},
	}
	frozen, err := verification.FreezeContract(c, verification.FreezePolicy{})
	if err != nil {
		t.Fatalf("FreezeContract(%s r%d): %v", taskID, revision, err)
	}
	return frozen
}

// AC1/AC2:Compile 从内容推导 pin,pin 与源合同一致;Task()/Verification()
// 原样透出源合同,snapshot 不添加任何覆盖源合同的语义字段。
func TestCompileDerivesPinsAndExposesSourceContracts(t *testing.T) {
	tc := mustTask(t, taskYAML)
	frozen := mustFrozen(t, "TASK-001", 3)

	s, err := Compile(tc, frozen)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	pins := s.Pins()
	if pins.Task.TaskID != "TASK-001" {
		t.Fatalf("task pin id = %q", pins.Task.TaskID)
	}
	if want := tc.RevisionHash(); pins.Task.RevisionHash != want {
		t.Fatalf("task pin hash = %q, want recomputed %q", pins.Task.RevisionHash, want)
	}
	if pins.Verification.TaskID != "TASK-001" || pins.Verification.Revision != 3 {
		t.Fatalf("verification pin id/revision = %s/%d", pins.Verification.TaskID, pins.Verification.Revision)
	}
	if pins.Verification.Hash != frozen.Hash() {
		t.Fatalf("verification pin hash = %q, want frozen hash %q", pins.Verification.Hash, frozen.Hash())
	}
	if s.TaskID() != "TASK-001" {
		t.Fatalf("TaskID() = %q", s.TaskID())
	}

	// 源合同原样透出:值相等,而非被 snapshot 改写过的投影。
	if got := s.Task(); !reflect.DeepEqual(got, tc) {
		t.Fatalf("Task() = %+v, want source contract %+v", got, tc)
	}
	if s.Verification() != frozen {
		t.Fatalf("Verification() 返回的不是被钉的冻结视图")
	}
	if s.Verification().Hash() != frozen.Hash() {
		t.Fatalf("Verification().Hash() = %q, want %q", s.Verification().Hash(), frozen.Hash())
	}
}

// AC3:身份哈希确定性 —— 同内容重复编译同哈希;与排版/条款顺序无关;
// 任一源合同换版则哈希必变;格式为 sha256:<64 位十六进制>。
func TestCompileHashDeterministic(t *testing.T) {
	tc := mustTask(t, taskYAML)
	frozen := mustFrozen(t, "TASK-001", 3)

	s1, err := Compile(tc, frozen)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	s2, err := Compile(mustTask(t, taskYAML), frozen)
	if err != nil {
		t.Fatalf("Compile(again): %v", err)
	}
	if s1.Hash() != s2.Hash() {
		t.Fatalf("同内容两次编译哈希不同:%s != %s", s1.Hash(), s2.Hash())
	}

	// 同一内容、不同排版:同一 pin、同一 snapshot 身份。
	s3, err := Compile(mustTask(t, taskYAMLRewritten), frozen)
	if err != nil {
		t.Fatalf("Compile(rewritten): %v", err)
	}
	if s3.Hash() != s1.Hash() {
		t.Fatalf("同内容不同排版哈希不同:%s != %s", s3.Hash(), s1.Hash())
	}
	if s3.Pins() != s1.Pins() {
		t.Fatalf("同内容不同排版 pin 不同:%+v != %+v", s3.Pins(), s1.Pins())
	}

	// verification 换版 → 哈希必变。
	s4, err := Compile(mustTask(t, taskYAML), mustFrozen(t, "TASK-001", 4))
	if err != nil {
		t.Fatalf("Compile(revision 4): %v", err)
	}
	if s4.Hash() == s1.Hash() {
		t.Fatalf("verification 换版后 snapshot 哈希未变:%s", s1.Hash())
	}

	if !strings.HasPrefix(s1.Hash(), "sha256:") || len(s1.Hash()) != len("sha256:")+64 {
		t.Fatalf("hash 格式 = %q", s1.Hash())
	}
}

// AC1:Compile 拒绝 mismatch 与非法输入;错误可用 errors.Is 判类。
func TestCompileRejectsMismatchAndInvalid(t *testing.T) {
	tc := mustTask(t, taskYAML)

	// task id 不一致。
	if _, err := Compile(tc, mustFrozen(t, "TASK-002", 1)); !errors.Is(err, ErrTaskMismatch) {
		t.Fatalf("Compile(task TASK-001 vs verification TASK-002) = %v, want ErrTaskMismatch", err)
	}
	// 冻结验证契约缺失。
	if _, err := Compile(tc, nil); !errors.Is(err, ErrNilVerification) {
		t.Fatalf("Compile(nil frozen) = %v, want ErrNilVerification", err)
	}
	// 手工构造的非法任务合同(缺 objective/profile)必须被拒绝,不能被聚合。
	invalid := taskcontract.TaskContract{ID: "TASK-001", Scope: taskcontract.Scope{AllowedPaths: []string{"internal/**"}}}
	if _, err := Compile(invalid, mustFrozen(t, "TASK-001", 1)); !errors.Is(err, ErrInvalidTask) {
		t.Fatalf("Compile(invalid task) = %v, want ErrInvalidTask", err)
	}
}

// AC1:Open 逐项复核 pin 与内容;pin 正确则物化出同一身份,任一项漂移即拒绝。
func TestOpenVerifiesPins(t *testing.T) {
	tc := mustTask(t, taskYAML)
	frozen := mustFrozen(t, "TASK-001", 3)
	compiled, err := Compile(tc, frozen)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	pins := compiled.Pins()

	// 正确的 pin → 同一 snapshot 身份。
	opened, err := Open(pins, mustTask(t, taskYAMLRewritten), frozen)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if opened.Hash() != compiled.Hash() {
		t.Fatalf("Open hash = %s, want %s", opened.Hash(), compiled.Hash())
	}

	// task pin 哈希不符(内容漂移或 pin 伪造)。
	badPins := pins
	badPins.Task.RevisionHash = "sha256:" + strings.Repeat("0", 64)
	if _, err := Open(badPins, mustTask(t, taskYAML), frozen); !errors.Is(err, ErrPinMismatch) {
		t.Fatalf("Open(stale task pin) = %v, want ErrPinMismatch", err)
	}

	// 同 id 不同内容:pin 哈希不符,id 相同救不了。
	if _, err := Open(pins, mustTask(t, taskYAMLOtherContent), frozen); !errors.Is(err, ErrPinMismatch) {
		t.Fatalf("Open(drifted task content) = %v, want ErrPinMismatch", err)
	}

	// verification pin 哈希不符。
	badPins = pins
	badPins.Verification.Hash = "sha256:" + strings.Repeat("f", 64)
	if _, err := Open(badPins, mustTask(t, taskYAML), frozen); !errors.Is(err, ErrPinMismatch) {
		t.Fatalf("Open(stale verification pin hash) = %v, want ErrPinMismatch", err)
	}

	// verification pin revision 漂移。
	badPins = pins
	badPins.Verification.Revision = 99
	if _, err := Open(badPins, mustTask(t, taskYAML), frozen); !errors.Is(err, ErrPinMismatch) {
		t.Fatalf("Open(drifted verification pin revision) = %v, want ErrPinMismatch", err)
	}

	// verification pin task id 与冻结契约不符。
	badPins = pins
	badPins.Verification.TaskID = "TASK-002"
	if _, err := Open(badPins, mustTask(t, taskYAML), frozen); !errors.Is(err, ErrTaskMismatch) {
		t.Fatalf("Open(wrong verification pin id) = %v, want ErrTaskMismatch", err)
	}

	// 两个 pin 的 task id 互相不一致(task pin 指向的合同 id 也随之不符)。
	if _, err := Open(Pins{Task: TaskPin{TaskID: "TASK-002", RevisionHash: pins.Task.RevisionHash}, Verification: pins.Verification}, mustTask(t, taskYAML), frozen); !errors.Is(err, ErrTaskMismatch) {
		t.Fatalf("Open(mismatched pin ids) = %v, want ErrTaskMismatch", err)
	}

	// Open 的 nil/非法输入拒绝。
	if _, err := Open(pins, mustTask(t, taskYAML), nil); !errors.Is(err, ErrNilVerification) {
		t.Fatalf("Open(nil frozen) = %v, want ErrNilVerification", err)
	}
}

// AC2/不变量:snapshot 只读 —— 编译后调用方改动自己手里的合同(含切片底层
// 数组)或改动 Task() 返回的拷贝,都不影响已钉住的 snapshot 与身份哈希。
func TestSnapshotReadOnlyAgainstCallerMutation(t *testing.T) {
	tc := mustTask(t, taskYAML)
	frozen := mustFrozen(t, "TASK-001", 3)
	s, err := Compile(tc, frozen)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	wantHash := s.Hash()

	// 改调用方手里的合同(值字段 + 切片元素)。
	tc.Objective = "mutated after compile"
	tc.Acceptance[0].Statement = "mutated acceptance"
	tc.Scope.AllowedPaths[0] = "everything/**"

	if got := s.Task(); got.Objective != "pin both source contract revisions into one aggregate" {
		t.Fatalf("snapshot 内的 objective 被调用方改写:%q", got.Objective)
	}
	if got := s.Task(); got.Acceptance[0].Statement != "snapshot pins the task revision" {
		t.Fatalf("snapshot 内的 acceptance 被调用方改写:%q", got.Acceptance[0].Statement)
	}
	if got := s.Task(); len(got.Scope.AllowedPaths) != 1 || got.Scope.AllowedPaths[0] != "internal/contract/**" {
		t.Fatalf("snapshot 内的 scope 被调用方改写:%v", got.Scope.AllowedPaths)
	}
	if s.Hash() != wantHash {
		t.Fatalf("调用方改动后 snapshot 哈希变化:%s != %s", s.Hash(), wantHash)
	}

	// 改 Task() 返回的拷贝,同样不影响 snapshot。
	tampered := s.Task()
	tampered.Objective = "tampered via accessor"
	tampered.Acceptance[0].ID = "AC-99"
	if s.Task().Objective != "pin both source contract revisions into one aggregate" || s.Task().Acceptance[0].ID != "AC-01" {
		t.Fatalf("通过访问器的改动泄漏进 snapshot")
	}
	if s.Hash() != wantHash {
		t.Fatalf("访问器改动后 snapshot 哈希变化:%s != %s", s.Hash(), wantHash)
	}
}
