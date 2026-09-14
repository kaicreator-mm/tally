package assurance

// ExecutedFacts — 已执行验证事实的 typed 记录(观察事实,不是 Claim)。
// achieved 档位只能从这些事实推导,绝不从 Agent/runtime 声明推导
// (Agent done 是 Claim,不是 Delivery Fact)。
type ExecutedFacts struct {
	// SelfCheck — 仅实现侧自检记录(L0)。
	SelfCheck bool
	// Deterministic — visible+protected 确定性检查已执行(L1 特征事实)。
	Deterministic bool
	// Judge — 独立 LLM judge 已执行(实验记录;按 PRD §13.2,L2 = judge +
	// 确定性检查,单独的 judge 不构成 L2)。
	Judge bool
	// HeldOut — protected/held-out verifier 已执行(L3 特征事实)。
	HeldOut bool
	// HumanApproval — human authority 验证已记录(L4 特征事实)。
	HumanApproval bool
}

// AchievedBasis — 达成档位的依据(AC4 在达成侧的对应)。
type AchievedBasis struct {
	// Facts — 输入事实的原样留存。
	Facts ExecutedFacts
	// Reasons — 按判定优先级排列的理由;首个元素是决定档位的理由;
	// ReasonJudgeNotCounted 只作为附加理由出现。
	Reasons []Reason
	// NoVerificationFacts — true 表示没有任何验证事实:档位按单轴最低值
	// L0 记,但依据明确标注"无验证事实",不冒充 self-check。
	NoVerificationFacts bool
}

// Achieved 从已执行验证事实推导达成档位(与 required 严格区分,AC1)。
// 判定优先级:HumanApproval > HeldOut > (Judge && Deterministic) >
// Deterministic > SelfCheck > 无事实。
//
// judge 事实只有在 Deterministic 同时为真时才贡献 L2(PRD §13.2 的 L2
// 定义);只有 judge 时档位不变,并追加 ReasonJudgeNotCounted。achieved
// 侧允许出现 L2 —— 它是对已发生实验事实的诚实记录,与 required 侧
// "永不选择 L2"(Lock #21)是两回事。
func Achieved(f ExecutedFacts) (Level, AchievedBasis) {
	basis := AchievedBasis{Facts: f}
	var level Level

	switch {
	case f.HumanApproval:
		level = LevelL4
		basis.Reasons = append(basis.Reasons, ReasonFactHumanApproval)
	case f.HeldOut:
		level = LevelL3
		basis.Reasons = append(basis.Reasons, ReasonFactHeldOut)
	case f.Judge && f.Deterministic:
		level = LevelL2
		basis.Reasons = append(basis.Reasons, ReasonFactJudge)
	case f.Deterministic:
		level = LevelL1
		basis.Reasons = append(basis.Reasons, ReasonFactDeterministic)
	case f.SelfCheck:
		level = LevelL0
		basis.Reasons = append(basis.Reasons, ReasonFactSelfCheck)
	default:
		level = LevelL0
		basis.NoVerificationFacts = true
		basis.Reasons = append(basis.Reasons, ReasonFactNone)
	}

	if f.Judge && level < LevelL2 {
		basis.Reasons = append(basis.Reasons, ReasonJudgeNotCounted)
	}
	return level, basis
}
