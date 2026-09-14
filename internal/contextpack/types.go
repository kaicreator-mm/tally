// Package contextpack 实现 PRD §12 Context Engine 的 P1 最小 TaskContextPack:
// 组装实现权威可见的上下文(§12.2 节选)、按 binding provenance 过滤历史失败与
// 知识输入(§12.3:任何上下文来源默认受 INV-VER-05 约束,不得成为 held-out
// 泄漏侧信道)、并携带 §8.4.3 的 VerificationContract redacted projection。
//
// 边界(P1,task P1-21):
//   - context_hash 在 redaction/filter 全部完成之后、对「实现权威实际可见的
//     最终内容」计算(§12.3 / §8.4.3);redacted 掉的 held-out 内容(store/ref、
//     selector、测试内容、baseline 细节)既不出现在包内容里,也不影响哈希。
//   - 本包是确定性纯函数:不读文件、不碰 Git、不执行命令;相关性选择引擎、
//     token 计量、Repo Map/Symbol 检索等属于后续任务,这里只保留 §12.3 要求的
//     记录面(included/excluded、selection reason、budget、hash)。
package contextpack

import (
	"tally/internal/taskcontract"
	"tally/internal/verification"
)

// Section 名,用于 Selection.Source,便于按来源审计 included/excluded(§12.3)。
const (
	SectionProductDoc        = "product_doc"
	SectionRequirementDoc    = "requirement_doc"
	SectionArchitectureDoc   = "architecture_doc"
	SectionDependencyTask    = "dependency_task"
	SectionVisibleTest       = "visible_test"
	SectionHistoricalFailure = "historical_failure"
	SectionKnowledge         = "knowledge"
	SectionVerification      = "verification_binding"
	// SectionPolicyRequires — 编译合同(contract.Bind,P1-07)产出的执行前
	// policy requires 验收告知上下文(PRD §8.3 Prevention,决策锁 18:注入
	// 冻结 Policy.requires,无独立 Guidance authoring plane)。
	SectionPolicyRequires = "policy_requires"
)

// Budget 记录组装时应用的规模预算(§12.3 要求记录 token/size budget;P1 最小
// 只支持每节条目数上限,token 计量归后续任务)。零值 = 不限制。
type Budget struct {
	// MaxItemsPerSection 是每个上下文小节的条目数上限;超限条目按确定性顺序
	// 排除并记录 Excluded。0 表示不限制。
	MaxItemsPerSection int
}

// Selection 是一条 include/exclude 决定记录(§12.3:必须记录 included refs、
// excluded refs 与 selection reason)。
//
// 防泄漏约束:凡是因 held-out provenance 被排除的条目,Ref 一律为空 —— 记录面
// 本身也是实现权威可读的,不得借 Excluded 记录回泄 held-out 内容的存在性线索。
type Selection struct {
	Source string // Section 常量之一
	Ref    string // 条目引用;held-out 派生条目恒为 ""
	Reason string // 选择/排除原因(确定性文本)
}

// 排除原因(P1 固定文案,保证同一输入产生同一记录与同一哈希)。
const (
	// ReasonHeldOutProvenance — 条目的 binding provenance 指向 held_out binding。
	ReasonHeldOutProvenance = "excluded: provenance 绑定 held_out binding,内容不得进入实现侧通道(INV-VER-05)"
	// ReasonUnresolvedProvenance — 条目声称的 binding 在冻结契约中不存在。
	// 保守默认:无法机械证明其可见,就不进入实现侧包。
	ReasonUnresolvedProvenance = "excluded: provenance 引用的 binding 不在冻结契约中,可见性无法证实"
	// ReasonBudget — 超出每节条目数预算。
	ReasonBudget = "excluded: 超过每节条目数预算"
	// ReasonProvided — 调用方选定的一般上下文条目(P1 无相关性引擎,直通)。
	ReasonProvided = "included: 调用方选定(P1 无相关性选择引擎)"
	// ReasonVisibleBinding — 可见 binding 按其可见性策略完整暴露。
	ReasonVisibleBinding = "included: visible binding 按可见性策略暴露"
	// ReasonRedactedProjection — held_out binding 仅以 redacted projection
	// (acceptance 语义 + evidence type)进入实现侧包(§8.4.3)。
	ReasonRedactedProjection = "included: redacted projection(仅 acceptance 语义与 evidence type,§8.4.3)"
	// ReasonPolicyRequires — 冻结已编译合同的 policy requires 验收告知条目。
	// 来源是 contract.Bind 的 RequiresContext(策略公开内容),held-out 验证
	// 内容按编译器保证绝不进入该小节(INV-V14-07)。
	ReasonPolicyRequires = "included: 冻结合同的 policy requires 验收告知(§8.3 Prevention,编译自 contract.Bind)"
)

// EvidenceProjection 是单条 evidence binding 在实现侧包中的投影。
//
// PRD §8.4.3 的 allowlist:held_out binding 只暴露 evidence type(不暴露
// binding id、store/ref、selector、测试内容、baseline 细节);visible binding
// 按 AC2 保持其标识与定位信息(binding id、path/selector/validator/store/ref)
// 可用。
type EvidenceProjection struct {
	Type       verification.EvidenceType
	Visibility verification.VisibilityPolicy

	// 以下字段仅 Visibility == visible 时非空;held_out 时恒为零值。
	BindingID string
	Path      string
	Selector  string
	Validator string
	Store     string
	Ref       string
}

// AcceptanceProjection 是一条 acceptance 的 redacted projection(§8.4.3):
// Acceptance 语义(statement + independence requirement)+ 每条 binding 的
// evidence 投影。evidence 条目按 binding id 排序。
type AcceptanceProjection struct {
	ID                   string
	Statement            string
	IndependenceRequired verification.IndependenceRequired
	Evidence             []EvidenceProjection
}

// Note 是一条通过 visibility filter 后进入包的历史失败/知识条目。
// 只有未被过滤的条目才携带 Ref/Summary,因此这里不会泄漏 held-out 内容。
type Note struct {
	Ref     string
	Summary string
}

// Pack 是实现权威可见的 TaskContextPack(P1 最小节选,PRD §12.2):
// TaskContract、VerificationContract redacted projection、文档/依赖任务/可见
// 测试引用、经 provenance 过滤的历史失败与知识,以及 §12.3 的记录面。
//
// 所有小节由 Build 排序规范化;Hash 是对规范化后最终内容(不含 hash 本身)的
// sha256,即「Agent 实际看到了什么」的指纹。
type Pack struct {
	TaskID string
	Task   taskcontract.TaskContract

	Verification []AcceptanceProjection

	ProductDocs      []string
	RequirementDocs  []string
	ArchitectureDocs []string
	DependencyTasks  []string
	VisibleTests     []string

	HistoricalFailures []Note
	Knowledge          []Note

	// PolicyRequires 是执行前 policy requires 验收告知小节(PRD §8.3
	// Prevention):contract.RequiresContext.Lines() 的渲染行,按策略升序。
	// 属冻结合同派生的必现内容,与合同 refs 同一纪律 —— 不参与每节预算,
	// 不得被裁剪(实现权威必须在执行前知道验收条件)。
	PolicyRequires []string

	Budget Budget

	Included []Selection
	Excluded []Selection

	hash string
}

// Hash 返回上下文哈希("sha256:<hex>"):在 redaction/filter 之后对实现权威
// 实际可见的最终内容计算(§12.3)。哈希为空表示包未经 Build 组装。
func (p *Pack) Hash() string { return p.hash }
