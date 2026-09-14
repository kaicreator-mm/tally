// verification.go 承载 v1.4.1 Confirm→Freeze 的**验证作者化与冻结准入**(P0-07):
// 把 owner 确认时声明的验收证据计划(planned evidence)作者化为冻结的
// VerificationContract,并守住准入边界 —— Confirm→Freeze 既不能发明、也不能
// 遗漏任何一条验收的证据绑定(PRD §3.2 的最后一跳:Structured Draft →
// Contract Validator → Frozen TaskContract / VerificationContract)。
//
// 准入规则(fail-safe,全部在任何冻结产物产生之前判定):
//
//  1. 发明防护:每条 evidence 声明必须命中确认草案里真实存在的验收条款
//     (AC-<数字>,taskcontract.ValidateStatic 已保证);指向不存在条款的声明
//     是被发明的绑定,类型化拒绝(ErrUnknownEvidenceTarget)。
//  2. 遗漏防护:每条验收条款必须至少有一条 planned evidence 声明;缺证据的
//     验收条款没有可执行的验证路径,缺失即类型化阻断(ErrMissingEvidence)——
//     这是确认路径应向 owner 追问的澄清/阻塞项,不是可默认跳过的空档。
//     verification 包的 freeze 规则(每 acceptance ≥1 binding)作为独立兜底
//     仍然生效。
//  3. held-out 语义更严:held_out 可见性**只来自 owner 的显式声明**
//     (EvidenceDeclaration.HeldOut),映射器绝不推断、绝不把任何生成内容
//     自动标注为 held-out(L3 pack「Don't」);且一旦声明 held-out,确认输入
//     必须同时声明实现侧 runtime 隔离能力(INV-VER-05:session_isolation +
//     held_out_unreachable + conformance_class),否则类型化阻断
//     (ErrHeldOutWithoutIsolation)。反向同理:visible 声明引用 holdout store
//     被既有校验拒绝(verification.Validate,§8.4.1)。
//  4. 权威委托:作者化产物必须通过 verification.Contract.Validate 与
//     freeze 专属规则(verification.FreezeContract,零值 FreezePolicy 即 P1
//     standard path);任何校验失败都不产生冻结产物(ErrInvalidVerification)。
//     本包不复制、不改写 verification 域的任何规则(DL-18:扩展既有域,
//     靠消费而非重复)。
//
// 作者化边界:
//
//   - authoring provenance 在 Confirm→Freeze 边界只有一种合法取值:
//     pre_commit(PRD §8.2.1)。post_hoc 是运行中治理升级路径(candidate-blind
//   - explicit review + NEW Run,P1 standard path 一律拒绝),确认时刻根本
//     不存在「事后补写」的作者行为 —— 声明形状里没有该字段,结构上不可表达。
//   - 验收条款的 independence_required 刻意保持未声明(保守默认 none):
//     确认输入没有独立性声明通道,准入不得发明治理属性(intake 同款边界);
//     verification 的 planned coverage 检查因此按 none 直通,显式等级的引入
//     需要先有确认侧输入通道,属后续任务。
//   - protection(protected evidence + baseline_hash)不在确认声明面上:它的
//     完整性锚点来自制品内容哈希,归制品落盘路径(P0-08)之后的语义;本包
//     既不发明 hash 也不接受无锚点的 protection 声明。
//   - 确定性:条款与声明都按内容排序后逐条映射(binding id 由条款 id + 序数
//     派生),输入切片顺序不影响冻结哈希;重复的等值声明被拒绝而不是去重
//     (「声明了什么」与「冻结了什么」不得分叉)。
//
// 纯函数边界与 Map 相同:没有时间、随机数、Git 写入或存储访问;Git 事务归
// P0-08。Map 只产出意图证据与任务合同,不是执行就绪的冻结;执行就绪的冻结
// 准入入口是本文件的 Freeze。
package intakefreeze

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"tally/internal/intentevidence"
	"tally/internal/taskcontract"
	"tally/internal/verification"
)

// 冻结准入的类型化错误哨兵:调用方用 errors.Is 判类,不匹配错误文本。
var (
	// ErrMissingEvidence:至少一条验收条款没有任何 planned evidence 声明
	// (实现要求 2 的类型化澄清/阻塞:缺失证据阻止执行就绪的冻结)。错误
	// 信息按条款 id 升序列出全部缺失条款。
	ErrMissingEvidence = errors.New("intakefreeze: 验收条款缺少 planned evidence 声明(缺失证据阻止执行就绪冻结)")
	// ErrUnknownEvidenceTarget:evidence 声明指向确认草案里不存在的验收条款。
	// 孤儿声明是被发明的绑定 —— Confirm→Freeze 不能发明证据绑定(任务目标),
	// 一律 fail-safe 拒绝,绝不静默丢弃或改挂到别的条款。
	ErrUnknownEvidenceTarget = errors.New("intakefreeze: evidence 声明指向不存在的验收条款")
	// ErrHeldOutWithoutIsolation:声明了 held_out evidence 但确认输入没有同时
	// 声明实现侧 runtime 隔离能力(INV-VER-05)。held-out 的语义更严:没有
	// 隔离能力声明就没有资格携带 held-out 绑定进入冻结。
	ErrHeldOutWithoutIsolation = errors.New("intakefreeze: held_out evidence 需要同时声明实现侧 runtime 隔离能力(session_isolation + held_out_unreachable + conformance_class,INV-VER-05)")
	// ErrInvalidVerification:作者化产物未通过 verification 域的确定性校验
	// (未知证据类型、held-out 定位形状、重复声明、空验收集等)。底层
	// verification 错误经 %w 保留在链上。
	ErrInvalidVerification = errors.New("intakefreeze: 验证契约未通过冻结校验")
)

// EvidenceDeclaration 是 owner 在确认时为一条验收条款声明的 planned evidence:
// verification.Binding 的有界确认面子集。只有这里列出的字段能进入冻结 ——
// authoring provenance、可见性默认值与绑定 id 都由准入规则确定性地给出,
// 不在声明面上(声明面越窄,可被发明的东西越少)。
type EvidenceDeclaration struct {
	// AcceptanceID 是声明归属的验收条款 id(如 AC-01)。必须精确命中确认
	// 草案里的条款(首尾排版空白折叠后匹配);否则 ErrUnknownEvidenceTarget。
	AcceptanceID string
	// Kind 是证据类型,直接复用 verification 的封闭词表(test/validator/
	// build/schema_check/static_rule/human_verification/deployment_verification/
	// runtime_observation);词表之外的值由既有校验拒绝(ErrInvalidVerification)。
	Kind verification.EvidenceType
	// Path/Selector 是 visible 证据在仓库内的定位(repo 内测试路径 + 选择器)。
	// held-out 声明不得携带 Path(§8.4.1:held-out 内容不得以 repo 路径暴露,
	// 由既有校验强制)。
	Path     string
	Selector string
	// HeldOut 是 held-out 可见性的**唯一**来源:只有 owner 显式声明为 true
	// 的绑定才是 held-out;映射器绝不从任何其它信号推断该值(绝不把生成内容
	// 自动标注为 held-out)。
	HeldOut bool
	// Store/Ref 是 held-out 证据的 store 引用(HeldOut=true 时必填,由既有
	// 校验强制)。visible 声明引用 holdout store 会被既有校验拒绝。
	Store string
	Ref   string
}

// VerificationPlan 是确认时的验证作者化输入:逐条款的证据声明加上(仅在声明
// held-out 时必需的)实现侧 runtime 能力要求。
type VerificationPlan struct {
	// Evidence 是 planned evidence 声明列表;顺序不承载语义(准入按内容排序
	// 后映射)。等值重复声明被拒绝而不是去重。
	Evidence []EvidenceDeclaration
	// RuntimeRequirements 是实现侧 runtime 能力要求声明(INV-VER-05)。仅在
	// Evidence 含 held-out 声明时必需且必须完整;纯 visible 契约允许省略
	// (零值 = 未声明任何能力要求)。
	RuntimeRequirements *verification.RuntimeRequirements
}

// FrozenArtifacts 是一次执行就绪冻结的完整产物:三份共享同一任务身份的冻结
// 制品内容(FrozenIntent 证据 + TaskContract + VerificationContract)。本包
// 只负责内容;把它们作为一个原子事务写进 Git 归 P0-08。
type FrozenArtifacts struct {
	// Intent 是 FrozenIntent 意图证据(经 intentevidence.Freeze 冻结,类型层不可变)。
	Intent intentevidence.FrozenIntent
	// Task 是映射出的 TaskContract(Hash 已按内容填充)。
	Task taskcontract.TaskContract
	// Verification 是冻结的 VerificationContract(经 verification.FreezeContract
	// 冻结,零值 FreezePolicy 即 P1 standard path;类型层不可变)。
	Verification verification.Frozen
}

// Freeze 是 Confirm→Freeze 的冻结准入入口:先走 P0-06 的纯映射(Map 的全部
// fail-safe 门原样生效),再把确认验收条款与证据计划作者化为验证契约,最后经
// verification.FreezeContract(零值策略,P1 standard path)冻结。任何准入
// 校验失败都返回类型化错误与零值,绝不产出部分冻结的中间态。
//
// 校验顺序固定:Map 门(profile → 任务类型 → 草案语义 → 意图冻结 → 合同校验)
// → 发明防护(孤儿声明)→ 等值重复声明 → 遗漏防护(缺证据条款,id 升序)→
// held-out 隔离门 → verification 域全量校验与冻结。
func Freeze(confirmed ConfirmedDraft, plan VerificationPlan) (FrozenArtifacts, error) {
	mapped, err := Map(confirmed)
	if err != nil {
		return FrozenArtifacts{}, err
	}
	contract, err := authorVerification(mapped.Task, plan)
	if err != nil {
		return FrozenArtifacts{}, err
	}
	frozen, err := verification.FreezeContract(contract, verification.FreezePolicy{})
	if err != nil {
		return FrozenArtifacts{}, fmt.Errorf("%w: %w", ErrInvalidVerification, err)
	}
	return FrozenArtifacts{Intent: mapped.Intent, Task: mapped.Task, Verification: *frozen}, nil
}

// authorVerification 把映射后的任务合同条款与 owner 证据计划作者化为一份
// 未冻结的 verification.Contract。条款集与任务合同 1:1(不发明、不遗漏),
// 声明按内容排序后逐条映射为 binding(绑定 id 由条款 id + 内容序数派生)。
func authorVerification(task taskcontract.TaskContract, plan VerificationPlan) (*verification.Contract, error) {
	// 条款索引:映射后的条款 id 已被 taskcontract 校验为 AC-<数字> 且唯一。
	clauses := make(map[string]string, len(task.Acceptance))
	for _, a := range task.Acceptance {
		clauses[a.ID] = a.Statement
	}

	// 1. 路由声明并守护发明边界:每条声明必须命中真实条款;同条款内等值
	// 重复声明是写错了的输入,拒绝而不是去重(声明的每一份内容都必须在
	// 冻结里可追溯,静默去重会让「声明了什么」与「冻结了什么」分叉)。
	byClause := make(map[string][]EvidenceDeclaration)
	seen := make(map[string]bool)
	for _, d := range plan.Evidence {
		id := strings.TrimSpace(d.AcceptanceID)
		if _, ok := clauses[id]; !ok {
			return nil, fmt.Errorf("%w: %q(每条声明必须命中确认草案里的 AC-<数字> 条款)", ErrUnknownEvidenceTarget, d.AcceptanceID)
		}
		key := id + "\x00" + declarationKey(d)
		if seen[key] {
			return nil, fmt.Errorf("%w: 条款 %s 存在等值重复的 evidence 声明(%s)", ErrInvalidVerification, id, declarationKey(d))
		}
		seen[key] = true
		byClause[id] = append(byClause[id], d)
	}

	// 2. 遗漏防护:每条验收条款至少一条 planned evidence 声明;缺失即类型化
	// 阻断,按条款 id 升序一次列全(确认路径据此向 owner 追问)。
	var missing []string
	for _, a := range task.Acceptance {
		if len(byClause[a.ID]) == 0 {
			missing = append(missing, a.ID)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("%w: %s(每条验收至少一条 planned evidence;缺失证据阻止执行就绪冻结)", ErrMissingEvidence, strings.Join(missing, ", "))
	}

	// 3. held-out 隔离门:任何 held-out 声明都必须伴随完整的实现侧 runtime
	// 隔离能力要求(INV-VER-05);既有校验是兜底,这里的预检给出独立类型化
	// 阻断,让「held-out 语义更严」在准入层显式可见。
	requirements := verification.RuntimeRequirements{}
	if plan.RuntimeRequirements != nil {
		requirements = *plan.RuntimeRequirements
	}
	for _, d := range plan.Evidence {
		if !d.HeldOut {
			continue
		}
		if plan.RuntimeRequirements == nil ||
			!requirements.SessionIsolation || !requirements.HeldOutUnreachable ||
			strings.TrimSpace(requirements.ConformanceClass) == "" {
			return nil, ErrHeldOutWithoutIsolation
		}
		break
	}

	// 4. 逐条款作者化:条款按 id 升序、条款内声明按内容键升序(输入顺序是
	// 排版,不是内容),binding id 由条款 id + 内容序数派生 —— 与条款的对应
	// 关系是机械的,不依赖调用方拼写,且跨条款唯一(条款 id 唯一)。
	ids := make([]string, 0, len(task.Acceptance))
	for _, a := range task.Acceptance {
		ids = append(ids, a.ID)
	}
	sort.Strings(ids)
	acceptances := make([]verification.Acceptance, 0, len(ids))
	for _, id := range ids {
		decls := byClause[id]
		sort.Slice(decls, func(i, j int) bool {
			return declarationKey(decls[i]) < declarationKey(decls[j])
		})
		acc := verification.Acceptance{
			ID:        id,
			Statement: clauses[id],
			// IndependenceRequired 刻意未声明(保守 none,verification 同款):
			// 确认输入没有独立性声明通道,准入不得发明治理属性。
		}
		acc.EvidenceBindings = make([]verification.Binding, 0, len(decls))
		for i, d := range decls {
			acc.EvidenceBindings = append(acc.EvidenceBindings, evidenceBinding(id, i, d))
		}
		acceptances = append(acceptances, acc)
	}

	return &verification.Contract{
		TaskID:              task.ID,
		Revision:            1, // Confirm→Freeze 的首次作者化;修订只能来自新的确认/再冻结
		Acceptances:         acceptances,
		RuntimeRequirements: requirements,
	}, nil
}

// evidenceBinding 把一条声明确定性地映射为 verification.Binding:
// authoring provenance 固定为 pre_commit(Confirm→Freeze 边界唯一可作者的
// 模式;post_hoc 是运行中治理升级,在声明形状上不可表达),可见性默认
// visible、仅由显式 HeldOut 声明翻转为 held_out,其余字段逐字搬运。
func evidenceBinding(clauseID string, ordinal int, d EvidenceDeclaration) verification.Binding {
	b := verification.Binding{
		ID:            fmt.Sprintf("%s-EV%02d", clauseID, ordinal+1),
		AuthoringMode: verification.AuthoringPreCommit,
		// 默认 visible 是安全方向:可见性只会因显式声明而收紧,绝不因缺席而放开。
		VisibilityPolicy: verification.VisibilityVisible,
		Evidence: verification.EvidenceRef{
			Type:     d.Kind,
			Path:     d.Path,
			Selector: d.Selector,
			Store:    d.Store,
			Ref:      d.Ref,
		},
	}
	if d.HeldOut {
		b.VisibilityPolicy = verification.VisibilityHeldOut
	}
	return b
}

// declarationKey 返回一条声明的内容键(排序与等值判定的规范形):除路由字段
// AcceptanceID 外的全部声明字段。首尾空白按排版折叠(与 profile/条款 id 同一
// 纪律);其余内容逐字参与,不做大小写归一。
func declarationKey(d EvidenceDeclaration) string {
	return strings.Join([]string{
		fmt.Sprintf("%t", d.HeldOut),
		string(d.Kind),
		strings.TrimSpace(d.Path),
		strings.TrimSpace(d.Selector),
		strings.TrimSpace(d.Store),
		strings.TrimSpace(d.Ref),
	}, "\x00")
}
