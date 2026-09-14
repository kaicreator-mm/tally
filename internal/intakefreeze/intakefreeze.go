// Package intakefreeze 承载 v1.4.1 Confirm→Freeze 链路的**纯领域映射**(P0-06)
// 与**验证作者化/冻结准入**(P0-07):把 owner 确认后的结构化草案
// (intake.Draft,澄清答案已折叠)确定性地映射为冻结工程制品 —— FrozenIntent
// 意图证据(P0-01)与 TaskContract(既有权威),即 PRD §3.2 的「Structured
// Draft → Contract Validator → Frozen TaskContract」的映射跳;执行就绪的冻结
// 准入(验收证据计划 → 冻结 VerificationContract)见 verification.go 的 Freeze。
//
// 边界与不变量:
//
//   - 纯函数:同一份输入无论何时、何进程、以什么切片顺序给出,产出逐字段相等
//     的同一份结果(身份/修订只由内容决定:Freeze 的规范化 + RevisionHash 的
//     排序去重)。没有时间、随机数、Git 写入或存储访问 —— Git 事务归 P0-08,
//     存储归既有路径,本包不得预支(任务合同实现要求 1)。
//   - 确认即权威的边界在这里显式化:草案的候选风险信号只是 advisory
//     (intake.Draft 注释「只是候选,不作数」),进入冻结证据的只能是
//     ConfirmedDraft.RiskHints —— owner 确认保留的声明提示(DL-06)。映射器
//     绝不把 LLM 生成的候选自动提升为权威(L3 pack「Do」:生成的 UI/风险文本
//     不得未经确定性校验/冻结成为 authority)。
//   - 治理决策不由映射器发明:profile 是冻结路径的治理输入(intake.Propose
//     同款边界:独立性与保证等级是冻结/编译路径的决策),必须显式给出并过
//     taskcontract 词表校验;验收条款的独立性刻意映射为「未声明」(保守默认
//     none,taskcontract 同款:独立性要求只能被显式声明,不能被默认赋予)。
//   - fail-safe 优先于宽松归一(任务合同实现要求 3):任务类型不在 MVP 两个
//     值内、草案携带冻结制品无法承载的语义(non_scope / constraints 非空)、
//     映射产物未通过既有静态校验,一律类型化拒绝,绝不猜测、绝不静默丢弃
//     owner 确认过的内容(「声明了什么」与「生效了什么」不得分叉)。
//   - 依赖保持最小:internal/intake(确认输入的领域形状)、internal/intentevidence
//     (Freeze(Facts) 唯一冻结入口,P0-01 裁定 Draft→Facts 映射归本包)、
//     internal/taskcontract(既有合同权威,只调用其导出的构造/校验/哈希,
//     不改动它)。
package intakefreeze

import (
	"errors"
	"fmt"
	"strings"

	"tally/internal/intake"
	"tally/internal/intentevidence"
	"tally/internal/taskcontract"
)

// 类型化错误的哨兵值:调用方用 errors.Is/errors.As 判类,不匹配错误文本。
var (
	// ErrInvalidProfile:确认输入给出的治理等级不在 taskcontract 词表
	// (light/standard/strict)内。profile 是治理决策,必须显式给出,映射器
	// 不猜测、不设默认值。
	ErrInvalidProfile = errors.New("intakefreeze: profile 不在治理等级词表内(light/standard/strict)")
	// ErrUnsupportedTaskType:草案任务类型不在 MVP 两个闭环类型之内
	// (AC4 同款纪律:超出集合一律 fail-safe 拒绝,不猜测归类)。
	ErrUnsupportedTaskType = errors.New("intakefreeze: 草案任务类型不在 MVP 支持范围内(bug_fix/db_migration)")
	// ErrUnsupportedDraftSemantics:确认草案携带冻结工程制品无法承载的语义。
	// TaskContract schema(PRD §7.3)没有 non_scope / constraints 字段,
	// FrozenIntent schema 恰四字段(DL-06):这两类内容进不了任何冻结制品,
	// 静默丢弃会让 owner 确认的内容与可执行合同分叉 —— 必须先由 owner 把它
	// 解析进 objective/scope/acceptance(或在确认时删除),映射器不代劳。
	ErrUnsupportedDraftSemantics = errors.New("intakefreeze: 草案携带冻结工程制品无法承载的语义")
	// ErrInvalidContract:映射产物未通过 taskcontract 的结构/静态校验
	// (PRD §7.4 确定性子集)。底层 taskcontract 错误经 %w 保留在链上。
	ErrInvalidContract = errors.New("intakefreeze: 映射产物未通过任务合同校验")
)

// ConfirmedDraft 是 owner 确认(含编辑与澄清折叠)之后的草案及其确认上下文:
// 进入冻结的唯一输入形状。字段是显式的,好让「什么经确认进入了冻结」在类型
// 上一目了然(DL-06:确认事实必须以不可变证据形式穿过 Freeze)。
type ConfirmedDraft struct {
	// TaskID 是目标任务身份(如 TASK-0042),必填。校验委托给下游权威:
	// intentevidence.Freeze 与 taskcontract.Validate 共同要求非空且不含空白。
	TaskID string
	// Profile 是冻结路径的治理等级决策(必填,light/standard/strict)。
	// intake 推理不产 profile,本包也不设默认值 —— 治理输入必须显式声明。
	Profile string
	// SourceRef 是可选的外部来源引用(与 intake.Intent.SourceRef 同语义),
	// 只进入 FrozenIntent 证据,不进入 TaskContract。
	SourceRef string
	// Draft 是确认后的结构化草案(澄清答案已按 P0-04 折叠进草案内容)。
	// 刻意不就地修改,也不保留其切片引用(全部深拷贝)。
	Draft intake.Draft
	// RiskHints 是 owner 确认保留的声明风险提示原文(可空:确认零条提示是
	// 合法证据)。Draft.RiskSignals 只是推理候选,绝不因确认而自动成为证据;
	// 哪些候选被保留,由确认路径显式写进本字段。
	RiskHints []string
}

// Frozen 是映射产物:一份冻结意图证据加一份任务合同修订。两者共享同一任务
// 身份;身份/修订全部由内容推导(FrozenIntent.Revision 与 TaskContract.Hash),
// 无任何声明出来的版本号。
type Frozen struct {
	// Intent 是 FrozenIntent 证据(经 intentevidence.Freeze 冻结,类型层不可变)。
	Intent intentevidence.FrozenIntent
	// Task 是映射出的 TaskContract,Hash 已按内容填充(与 taskcontract.Parse
	// 同一纪律:哈希来自内容重算,不来自声明)。
	Task taskcontract.TaskContract
}

// Map 把确认草案确定性地映射为冻结意图证据与任务合同。它只是完整冻结的
// 映射跳(不含验证权威面):执行就绪的冻结准入(验收证据计划 + 冻结
// VerificationContract)走 verification.go 的 Freeze,其复用本函数的全部门。
// 校验顺序固定,任何一步失败即返回类型化错误与零值,绝不产出「部分冻结」
// 的中间态:
//
//  1. profile 词表(显式治理输入);
//  2. 任务类型翻译(intake 词表 → 冻结词表,恰为显式相等映射);
//  3. 草案语义承载门(non_scope / constraints 必须已解析为空);
//  4. intentevidence.Freeze(Facts)(身份/提示的 fail-closed 校验与冻结);
//  5. taskcontract.Validate + ValidateStatic(既有权威校验,含 scope 矛盾);
//  6. 内容修订哈希(taskcontract.Parse 同款:Hash 永远来自重算)。
func Map(confirmed ConfirmedDraft) (Frozen, error) {
	// 1. profile:去首尾空白后必须精确命中词表(排版空白按 taskcontract.Parse
	// normalize 同一纪律折叠,内部空白/大小写变体不折叠,一律拒绝)。
	profile := strings.TrimSpace(confirmed.Profile)
	if !taskcontract.IsValidProfile(profile) {
		return Frozen{}, fmt.Errorf("%w: %q", ErrInvalidProfile, confirmed.Profile)
	}

	// 2. 任务类型:intake 词表 → 冻结词表。两侧取值故意逐字相等(P0-01 用
	// 测试钉住词表一致),映射仍是显式相等翻译而非字符串透传 —— intake 或
	// 冻结侧任何一侧扩词表,这里都是编译失败点,不会静默放行第三种类型。
	var taskType intentevidence.TaskType
	switch confirmed.Draft.TaskType {
	case intake.TaskTypeBugFix:
		taskType = intentevidence.TaskTypeBugFix
	case intake.TaskTypeMigration:
		taskType = intentevidence.TaskTypeMigration
	default:
		return Frozen{}, fmt.Errorf("%w: %q", ErrUnsupportedTaskType, confirmed.Draft.TaskType)
	}

	// 3. 语义承载门:这两类字段在冻结 schema 里没有归宿(见 ErrUnsupportedDraftSemantics)。
	// 有任何条目(含纯空白条目)都算「未解析完就拿来冻结」,fail-safe 拒绝,
	// 让确认路径显式解析,而不是映射器替 owner 决定扔掉什么。
	if n := len(confirmed.Draft.NonScope); n > 0 {
		return Frozen{}, fmt.Errorf("%w: non_scope 有 %d 条内容,TaskContract/FrozenIntent 无字段可承载;请先解析进 objective/scope/acceptance 或在确认时删除", ErrUnsupportedDraftSemantics, n)
	}
	if n := len(confirmed.Draft.Constraints); n > 0 {
		return Frozen{}, fmt.Errorf("%w: constraints 有 %d 条内容,TaskContract/FrozenIntent 无字段可承载;请先解析进 objective/scope/acceptance 或在确认时删除", ErrUnsupportedDraftSemantics, n)
	}

	// 4. 冻结意图证据:身份/来源/提示的校验与规范化全部委托 P0-01 的唯一
	// 冻结入口(intentevidence 类型化哨兵经包装原样可 errors.Is)。
	intent, err := intentevidence.Freeze(intentevidence.Facts{
		TaskID:    confirmed.TaskID,
		TaskType:  taskType,
		SourceRef: confirmed.SourceRef,
		RiskHints: confirmed.RiskHints,
	})
	if err != nil {
		return Frozen{}, fmt.Errorf("intakefreeze: 冻结意图证据: %w", err)
	}

	// 5. 映射任务合同:仅搬运双方 schema 都承载的字段,逐字段显式赋值;
	// intake 草案没有的 refs/depends_on/documentation_impact 一律留零值,
	// 不发明占位内容。切片深拷贝,输入在映射后改动不影响产物。
	task := taskcontract.TaskContract{
		ID:        strings.TrimSpace(confirmed.TaskID),
		Profile:   profile,
		Title:     confirmed.Draft.Summary,
		Objective: confirmed.Draft.Objective,
		Scope: taskcontract.Scope{
			AllowedPaths:   cloneStrings(confirmed.Draft.Scope.AllowedPaths),
			ForbiddenPaths: cloneStrings(confirmed.Draft.Scope.ForbiddenPaths),
		},
	}
	task.Acceptance = make([]taskcontract.AcceptanceClause, len(confirmed.Draft.Acceptance))
	for i, a := range confirmed.Draft.Acceptance {
		// IndependenceRequired 刻意留空:草案条款没有独立性字段(intake 刻意
		// 不让推理发明治理属性),映射器也不发明;未声明按 none 是 taskcontract
		// 的保守默认,显式等级归 P0-07 冻结准入/编译路径的治理决策。
		task.Acceptance[i] = taskcontract.AcceptanceClause{
			ID:        a.ID,
			Statement: a.Statement,
		}
	}
	if err := task.Validate(); err != nil {
		return Frozen{}, fmt.Errorf("%w: %w", ErrInvalidContract, err)
	}
	// ValidateStatic 是同一包内的纯确定性准入校验(ID/条款文法、scope 语法与
	// allowed×forbidden 矛盾):在制品提交前拒绝自相矛盾/形态非法的合同
	// (任务合同 Verification:「invalid drafts fail before artifact commit」),
	// 不访问文件系统/网络,不改变本包的纯函数边界。
	if err := task.ValidateStatic(); err != nil {
		return Frozen{}, fmt.Errorf("%w: %w", ErrInvalidContract, err)
	}

	// 6. 内容修订哈希:与 taskcontract.Parse 同一纪律,来自内容重算。
	task.Hash = task.RevisionHash()
	return Frozen{Intent: intent, Task: task}, nil
}

// cloneStrings 返回切片深拷贝;nil 保持 nil(缺席与空在合同语义里可区分,
// 与 internal/contract 同一约定)。
func cloneStrings(vals []string) []string {
	if vals == nil {
		return nil
	}
	out := make([]string, len(vals))
	copy(out, vals)
	return out
}
