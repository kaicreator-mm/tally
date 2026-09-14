package verification

import "fmt"

// FreezePolicy — freeze 时刻的策略开关。零值即 P1 standard path(最严格默认)。
type FreezePolicy struct {
	// AllowPostHoc 是否允许 authoring_mode: post_hoc 的 binding 通过 freeze。
	//
	// P1 standard path(零值)一律拒绝(PRD §8.2 第 2/3 条:post_hoc 仅允许
	// Light→Standard 的运行中治理升级,且必须 candidate-blind、explicit review、
	// NEW CodegenRun;该流程属 P2 post_hoc path)。strict 任务的 required
	// evidence 必须使用 pre_commit(§8.2 第 4 条),因此必须保持零值策略。
	AllowPostHoc bool
}

// ValidateFreeze 执行 freeze 前的全部确定性校验,但不产生冻结视图。校验分两层:
//
//  1. Contract.Validate —— binding 声明合法性(INV-VER-01:authoring provenance
//     必须声明;visibility/evidence 引用/INV-VER-03 protection 格式/execution
//     protocol)与 Acceptance 级 planned independence coverage 的全称检查
//     (PRD §8.5.1:设计阶段缺失 → FREEZE FAIL,无 Waiver。Waiver 只属于运行期
//     planned ✓ → effective ✗ 的 independence loss,§8.5.2,不在本包);
//  2. freeze 专属规则 —— 每个 acceptance 必须有功能验证计划(至少一条 planned
//     evidence binding)、post_hoc 默认拒绝(除非 FreezePolicy 显式允许)、
//     声明 protection 的 binding 必须在 freeze 时固定 baseline hash。
func ValidateFreeze(c *Contract, policy FreezePolicy) error {
	if c == nil {
		return fmt.Errorf("verification: contract 为 nil")
	}
	if err := c.Validate(); err != nil {
		return err
	}
	return validateFreezeRules(c, &policy)
}

// FreezeContract 是 P1 的完整 freeze 入口:ValidateFreeze 全部通过后,对内容做
// 深拷贝并计算规范化哈希,返回只读 Frozen(与 Contract.Freeze 对同一内容产生的
// 冻结结果完全一致,含相同哈希)。与类型层最小入口 Contract.Freeze 相比,它额外
// 执行 freeze 专属规则;任何校验失败都不产生 Frozen。
func FreezeContract(c *Contract, policy FreezePolicy) (*Frozen, error) {
	if err := ValidateFreeze(c, policy); err != nil {
		return nil, err
	}
	fc := c.clone()
	h, err := Hash(&fc)
	if err != nil {
		return nil, fmt.Errorf("verification: 计算契约哈希: %w", err)
	}
	return &Frozen{c: fc, hash: h}, nil
}

// validateFreezeRules — 只有在 freeze 时刻才判定的规则(契约本身的合法性归
// Contract.Validate)。按确定性顺序逐 acceptance、逐 binding 检查,返回第一条错误。
func validateFreezeRules(c *Contract, policy *FreezePolicy) error {
	for i := range c.Acceptances {
		acc := &c.Acceptances[i]

		// 功能验证计划:每个 acceptance 至少一条 planned evidence binding,
		// 否则该验收没有任何被验证的路径(PRD §8 Completeness)。
		if len(acc.EvidenceBindings) == 0 {
			return fmt.Errorf("verification: acceptance %s 没有任何 planned evidence binding(功能验证计划缺失)", acc.ID)
		}

		for j := range acc.EvidenceBindings {
			b := &acc.EvidenceBindings[j]

			// post_hoc:P1 standard path 默认拒绝;只有策略显式放行才继续。
			if b.AuthoringMode == AuthoringPostHoc && !policy.AllowPostHoc {
				return fmt.Errorf("verification: binding %s authoring_mode=post_hoc 被 P1 standard path 拒绝(§8.2:post_hoc 仅限运行中治理升级,且必须 candidate-blind + explicit review + NEW Run);如需启用请在 FreezePolicy.AllowPostHoc 显式声明", b.ID)
			}

			// protected hash:声明 protection 的 binding 必须在 freeze 时固定
			// baseline hash;没有 hash 的 protection 提供不了完整性锚点,其所授权的
			// 实现 Run 也就无从被约束(INV-VER-03 / §8.2 第 5 条:baseline hash
			// 冻结后不可被其所授权的实现 Run 修改)。
			if b.Protection != nil && b.Protection.BaselineHash == "" {
				return fmt.Errorf("verification: binding %s 声明了 protection 但缺少 baseline_hash(INV-VER-03:protected hash 必须在 freeze 时固定)", b.ID)
			}
		}
	}
	return nil
}
