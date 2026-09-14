package contextpack

import (
	"errors"
	"fmt"
	"sort"

	"tally/internal/verification"
)

// ErrNilContract 表示 Build/RedactVerification 收到 nil 的冻结 VerificationContract。
// 没有 frozen 契约就没有可投影的验证权威内容;保守失败,不构造半成品包。
var ErrNilContract = errors.New("contextpack: 冻结 VerificationContract 为 nil")

// RedactVerification 从冻结 VerificationContract 推导实现权威可见的
// redacted projection(PRD §8.4.3,INV-VER-05):
//
//   - 每条 acceptance 保留 Acceptance 语义(id、statement、independence_required,
//     Decision Lock 10:独立性属于 Acceptance Clause);
//   - visible binding 完整暴露 evidence 定位(path/selector/validator/store/ref,
//     AC2:可见证据按策略保持可用;held_out binding 不得引用 holdout store,
//     该约束由契约冻结校验保证);
//   - held_out binding 只暴露 evidence type —— binding id、store/ref、selector、
//     测试内容、baseline 细节、hidden failure 输出一律不出现在投影里。
//
// 投影按 acceptance id、binding id 排序,输出是契约内容的确定性函数。
// 返回的切片是新构造的值,与冻结契约无共享引用。
func RedactVerification(f *verification.Frozen) ([]AcceptanceProjection, error) {
	if f == nil {
		return nil, ErrNilContract
	}
	c := f.Contract() // 深拷贝,读侧与冻结对象脱钩

	out := make([]AcceptanceProjection, len(c.Acceptances))
	for i, acc := range c.Acceptances {
		proj := AcceptanceProjection{
			ID:                   acc.ID,
			Statement:            acc.Statement,
			IndependenceRequired: acc.IndependenceRequired,
			Evidence:             make([]EvidenceProjection, 0, len(acc.EvidenceBindings)),
		}
		bindings := make([]verification.Binding, len(acc.EvidenceBindings))
		copy(bindings, acc.EvidenceBindings)
		sort.Slice(bindings, func(a, b int) bool { return bindings[a].ID < bindings[b].ID })

		for _, b := range bindings {
			ep := EvidenceProjection{
				Type:       b.Evidence.Type,
				Visibility: b.VisibilityPolicy,
			}
			// §8.4.3 allowlist:held_out 只暴露 evidence type;定位字段仅
			// visible 才携带。契约冻结校验已保证 held_out 只经 store/ref 引用、
			// visible 不引用 holdout store,这里按可见性逐字段赋值。
			if b.VisibilityPolicy == verification.VisibilityVisible {
				ep.BindingID = b.ID
				ep.Path = b.Evidence.Path
				ep.Selector = b.Evidence.Selector
				ep.Validator = b.Evidence.Validator
				ep.Store = b.Evidence.Store
				ep.Ref = b.Evidence.Ref
			}
			proj.Evidence = append(proj.Evidence, ep)
		}
		out[i] = proj
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ID < out[b].ID })
	return out, nil
}

// verificationSelections 生成 VerificationContract 投影的 included/excluded
// 记录(§12.3)。held_out 条目的记录不携带 binding id/store/ref —— 记录面
// 本身也是实现权威可读的,Ref 字段只保留 acceptance id 作为节内锚点。
func verificationSelections(projections []AcceptanceProjection) (included, excluded []Selection) {
	for _, proj := range projections {
		for _, ev := range proj.Evidence {
			switch ev.Visibility {
			case verification.VisibilityVisible:
				included = append(included, Selection{
					Source: SectionVerification,
					Ref:    proj.ID + "/" + ev.BindingID,
					Reason: ReasonVisibleBinding,
				})
			case verification.VisibilityHeldOut:
				included = append(included, Selection{
					Source: SectionVerification,
					Ref:    proj.ID,
					Reason: ReasonRedactedProjection,
				})
			default:
				// 契约冻结校验已拒绝非法可见性,这里到不了;保守起见仍不放宽。
				excluded = append(excluded, Selection{
					Source: SectionVerification,
					Ref:    proj.ID,
					Reason: fmt.Sprintf("excluded: 未知可见性策略 %q", ev.Visibility),
				})
			}
		}
	}
	return included, excluded
}
