package failure

// OutcomeKind 是失败的出口。修不了的要明确归类,不能一直重试,
// 不能含糊留着(PRD 第四部分规则 ④)。取值与 fingerprint_stat.status 枚举一致,
// 归类结果直接写库。
type OutcomeKind string

const (
	// OutcomeFixed:修复经四道闸验证,已入知识库。
	OutcomeFixed OutcomeKind = "fixed"
	// OutcomeFlaky:偶发,重现不稳定 —— 只识别并排除,不进知识库。
	OutcomeFlaky OutcomeKind = "flaky"
	// OutcomeUpstream:上游的问题 —— 记下,版本锁在当前能用的那个,写清为什么锁、什么时候解。
	OutcomeUpstream OutcomeKind = "upstream"
	// OutcomeInfra:环境/基础设施问题(磁盘满、网断)—— 运维事项,不进知识库。
	OutcomeInfra OutcomeKind = "infra"
	// OutcomeSpecDefect:代码没写错,是需求本身有问题 —— 退回 requirement 改需求。
	OutcomeSpecDefect OutcomeKind = "spec_defect"
	// 空串 "" 不是出口:本轮候选没过验证,等下一轮(受三轮预算约束)。
)

// terminalKinds:到了就是诊断终结,同指纹再命中直接返回该结论、不再调 LLM。
var terminalKinds = map[OutcomeKind]bool{
	OutcomeFixed:      true,
	OutcomeFlaky:      true,
	OutcomeUpstream:   true,
	OutcomeInfra:      true,
	OutcomeSpecDefect: true,
}

// IsTerminal 报告 status 是否终态出口。
func IsTerminal(status string) bool { return terminalKinds[OutcomeKind(status)] }

// Evidence 是归类的确定性依据:全部来自可执行/可测量的信号,
// 没有一项是 LLM 的结论(铁律 1:LLM 只能提案,判定永远在这里)。
type Evidence struct {
	ReproStable         bool // G1:稳定复现(连跑 5 次全红)
	FixVerified         bool // G4+G2+G3:修复验证通过
	SpecDefectConfirmed bool // spec_defect 提案过了确定性闸门(条款确实存在)
	CrossProject        bool // 同指纹命中多个项目 → 基础设施问题
	DepAligned          bool // 依赖变更与失败时间线对齐 → 上游问题
}

// Classify 从证据推出出口。顺序即权重:
// 验证过的修复压过一切;复现不稳则任何结论都立不住;
// 需求缺陷要往上退(回 requirement);跨项目是基础设施;依赖对齐是上游;
// 都不是 → 无出口(候选失败,预算内再试)。
func Classify(e Evidence) OutcomeKind {
	switch {
	case e.FixVerified:
		return OutcomeFixed
	case !e.ReproStable:
		return OutcomeFlaky
	case e.SpecDefectConfirmed:
		return OutcomeSpecDefect
	case e.CrossProject:
		return OutcomeInfra
	case e.DepAligned:
		return OutcomeUpstream
	default:
		return ""
	}
}
