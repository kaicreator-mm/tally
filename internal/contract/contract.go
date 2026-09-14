// Package contract 提供 v1.4 语义上的 Contract(PRD §7):以「钉住修订」的方式
// 聚合既有的 taskcontract.TaskContract 与冻结的 verification.Frozen,而不物理
// 合并或迁移两个既有合同格式(DECISION_LOCKS 5:pinned aggregate facade)。
//
// 不变量:
//
//   - snapshot 里的每个 pin 都从合同内容推导(Compile)或逐字段对照内容复核
//     (Open),绝不来自调用方的断言;task id 不一致或内容与 pin 不符直接拒绝;
//   - snapshot 不添加任何可覆盖源合同的语义字段:acceptance、scope、evidence、
//     assurance 一律只经只读访问器透出源合同本身;matched_policies / assurance
//     的绑定归 P1-07 compiler,不属于本聚合;
//   - 身份(Hash)是两个被钉修订哈希的确定性纯函数:同一内容对在任何进程、
//     任何时间重复编译得到同一哈希;
//   - snapshot 编译后只读:字段全部不导出,访问器返回深拷贝或只读冻结视图;
//     修订 = 新 snapshot(PRD §7.4:amendment/supersession 只增不改)。
package contract

import (
	"errors"
	"fmt"

	"tally/internal/taskcontract"
	"tally/internal/verification"
)

// 包内确定性错误的哨兵值:调用方用 errors.Is 判类,不匹配错误文本。
var (
	// ErrNilVerification:聚合时冻结验证契约为 nil。没有冻结验证权威的
	// 「Contract」不构成 v1.4 语义的 Contract(PRD §7.1)。
	ErrNilVerification = errors.New("contract: 冻结验证契约为 nil")
	// ErrInvalidTask:TaskContract 未通过结构校验(taskcontract.Validate)。
	ErrInvalidTask = errors.New("contract: 任务合同未通过结构校验")
	// ErrTaskMismatch:两个 pin / 合同声明的 task id 不一致。
	ErrTaskMismatch = errors.New("contract: task id 不一致")
	// ErrPinMismatch:pin 与被钉合同内容不符(内容漂移、revision 漂移或 pin 伪造)。
	ErrPinMismatch = errors.New("contract: pin 与被钉合同内容不符")
)

// TaskPin 钉住一个 TaskContract 修订。
type TaskPin struct {
	// TaskID 是被钉合同的任务 id(TaskContract.ID)。
	TaskID string
	// RevisionHash 是被钉 TaskContract 的内容修订哈希(sha256:<hex>),与
	// TaskContract.RevisionHash 对同一内容的计算结果一致。
	RevisionHash string
}

// VerificationPin 钉住一个冻结 VerificationContract 修订。
type VerificationPin struct {
	// TaskID 是被钉冻结契约的任务 id,必须与 TaskPin.TaskID 一致。
	TaskID string
	// Revision 是被钉冻结契约的 revision。
	Revision int
	// Hash 是冻结时计算的内容哈希(sha256:<hex>),冻结后不可变。
	Hash string
}

// Pins 是 snapshot 携带的一对修订 pin。持久化这两个值,之后可用 Open 对照
// 合同内容复核(内容漂移/tamper 检测)。
type Pins struct {
	Task         TaskPin
	Verification VerificationPin
}

// Snapshot 是 v1.4 语义的 Contract 聚合(ContractSnapshot):一个 TaskContract
// 修订加一个冻结 VerificationContract 修订,由内容推导的 pin 钉在一起。
//
// 字段全部不导出;构造入口只有 Compile 与 Open。
type Snapshot struct {
	pins   Pins
	task   taskcontract.TaskContract // 深拷贝,原样透出,不做任何字段覆盖
	frozen *verification.Frozen      // 已冻结只读视图,原样透出
	hash   string
}

// Compile 把一份通过结构校验的 TaskContract 与一份冻结 VerificationContract
// 聚合为一个 snapshot,pin 从合同内容推导(不信任调用方断言)。
//
// 拒绝:
//   - frozen 为 nil(ErrNilVerification);
//   - TaskContract 未通过 taskcontract.Validate(ErrInvalidTask);
//   - 两个合同声明的 task id 不同(ErrTaskMismatch)。
func Compile(tc taskcontract.TaskContract, frozen *verification.Frozen) (*Snapshot, error) {
	if frozen == nil {
		return nil, ErrNilVerification
	}
	if err := tc.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTask, err.Error())
	}
	pins := Pins{
		Task: TaskPin{
			TaskID:       tc.ID,
			RevisionHash: tc.RevisionHash(), // 由内容重算,不读可能过期的 tc.Hash 字段
		},
		Verification: VerificationPin{
			TaskID:   frozen.TaskID(),
			Revision: frozen.Revision(),
			Hash:     frozen.Hash(),
		},
	}
	if pins.Task.TaskID != pins.Verification.TaskID {
		return nil, fmt.Errorf("%w: TaskContract %s vs VerificationContract %s", ErrTaskMismatch, pins.Task.TaskID, pins.Verification.TaskID)
	}
	return &Snapshot{pins: pins, task: cloneTask(tc), frozen: frozen, hash: hashPins(pins)}, nil
}

// Open 从持久化的 pins 与其声称钉住的合同内容重新物化 snapshot:逐项复核
// pin 与内容(task id、revision、两个内容哈希),任何一项不符即拒绝。用于
// 事后读取场景 —— 例如执行链路里拿着运行记录里的 pin 校验手头的合同内容
// 是否仍是冻结时的那一版(PRD §7.3:开始执行后不得漂移)。
//
// 拒绝:ErrNilVerification / ErrInvalidTask / ErrTaskMismatch / ErrPinMismatch。
func Open(pins Pins, tc taskcontract.TaskContract, frozen *verification.Frozen) (*Snapshot, error) {
	if frozen == nil {
		return nil, ErrNilVerification
	}
	if err := tc.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTask, err.Error())
	}
	if pins.Task.TaskID != tc.ID {
		return nil, fmt.Errorf("%w: task pin %s vs TaskContract %s", ErrTaskMismatch, pins.Task.TaskID, tc.ID)
	}
	if pins.Verification.TaskID != frozen.TaskID() {
		return nil, fmt.Errorf("%w: verification pin %s vs VerificationContract %s", ErrTaskMismatch, pins.Verification.TaskID, frozen.TaskID())
	}
	if tc.ID != frozen.TaskID() {
		return nil, fmt.Errorf("%w: TaskContract %s vs VerificationContract %s", ErrTaskMismatch, tc.ID, frozen.TaskID())
	}
	if actual := tc.RevisionHash(); actual != pins.Task.RevisionHash {
		return nil, fmt.Errorf("%w: task pin %s vs 实际内容 %s", ErrPinMismatch, pins.Task.RevisionHash, actual)
	}
	if frozen.Hash() != pins.Verification.Hash {
		return nil, fmt.Errorf("%w: verification pin %s vs 实际内容 %s", ErrPinMismatch, pins.Verification.Hash, frozen.Hash())
	}
	if frozen.Revision() != pins.Verification.Revision {
		return nil, fmt.Errorf("%w: verification pin revision %d vs 实际 %d", ErrPinMismatch, pins.Verification.Revision, frozen.Revision())
	}
	return &Snapshot{pins: pins, task: cloneTask(tc), frozen: frozen, hash: hashPins(pins)}, nil
}

// Pins 返回 snapshot 携带的一对修订 pin(纯值,无引用共享)。
func (s *Snapshot) Pins() Pins { return s.pins }

// Hash 返回 snapshot 身份哈希(sha256:<hex>):两个被钉修订哈希的确定性函数。
// 同一内容对重复编译结果相同;任一源合同换版都会改变它。
func (s *Snapshot) Hash() string { return s.hash }

// TaskID 返回聚合双方的共同 task id。
func (s *Snapshot) TaskID() string { return s.pins.Task.TaskID }

// Task 返回被钉 TaskContract 的深拷贝。返回值上的任何改动都不影响 snapshot
// (snapshot 不暴露可变引用,修订 = 新 snapshot)。
func (s *Snapshot) Task() taskcontract.TaskContract { return cloneTask(s.task) }

// Verification 返回被钉的冻结 VerificationContract 只读视图。*verification.Frozen
// 本身不可变(冻结时深拷贝、无修改路径),原样透出即安全。
func (s *Snapshot) Verification() *verification.Frozen { return s.frozen }

// cloneTask 深拷贝 TaskContract 的全部切片字段:Compile/Open 之后调用方改动
// 自己手里的合同(含切片底层数组)不得影响已钉住的 snapshot。AcceptanceClause
// 只含值字段,元素按值复制即足够。
func cloneTask(c taskcontract.TaskContract) taskcontract.TaskContract {
	cp := c
	cp.ProductRefs = cloneStrings(c.ProductRefs)
	cp.RequirementRefs = cloneStrings(c.RequirementRefs)
	cp.ArchitectureRefs = cloneStrings(c.ArchitectureRefs)
	cp.DependsOn = cloneStrings(c.DependsOn)
	cp.Scope.AllowedPaths = cloneStrings(c.Scope.AllowedPaths)
	cp.Scope.ForbiddenPaths = cloneStrings(c.Scope.ForbiddenPaths)
	cp.Acceptance = make([]taskcontract.AcceptanceClause, len(c.Acceptance))
	copy(cp.Acceptance, c.Acceptance)
	cp.DocumentationImpact = cloneStrings(c.DocumentationImpact)
	return cp
}

// cloneStrings 返回切片的深拷贝;nil 保持 nil(缺席与空在合同语义里可区分)。
func cloneStrings(vals []string) []string {
	if vals == nil {
		return nil
	}
	out := make([]string, len(vals))
	copy(out, vals)
	return out
}
