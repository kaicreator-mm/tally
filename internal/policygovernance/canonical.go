// canonical.go 拥有候选内容的正典化与验证语义(P3-03):提案携带的策略内容
// 必须经既有 EngineeringPolicy 解析器正典解析(要求 1:不建第二 DSL),且其
// 内容修订必须等于正典内容的真实修订语义(AC)。
//
// 修订语义(engineeringpolicy.hash.go):RevisionHash 输入是解析+normalize
// 之后的规范字段(列表去重排序、空白折叠),因此对同一份策略内容,排版差异
// (键序、缩进、行尾空白)不改变修订 —— 修订是「内容」的标识,不是「字节」
// 的标识。本文件把该语义固化为提案域的写入边界校验:
//   - VerifyCandidate:重放正典解析,声称修订必须与解析修订一致 —— 提交流
//     里携带的 candidate_rev 永远可由 candidate_yaml 重导出(DL-09 的治理侧
//     对应面:内容可独立验证,不信任调用方自报);
//   - 不新增任何策略字段/语法:全部校验复用 engineeringpolicy.Parse
//     (KnownFields 严格模式 + 执行策略字段拒绝)。
package policygovernance

import "fmt"

// VerifyCandidate 验证候选内容:经既有解析器正典解析(不可解析 = 拒绝),
// 并要求声称修订与解析修订一致(AC:候选修订等于实际正典策略内容修订语义)。
// 返回解析出的候选(修订已由 Parse 填充)。
func VerifyCandidate(raw []byte, claimedRevision string) (Candidate, error) {
	cand, err := NewCandidate(raw)
	if err != nil {
		return Candidate{}, err
	}
	if claimedRevision == "" {
		return cand, nil
	}
	if claimedRevision != cand.Revision() {
		return Candidate{}, fmt.Errorf("%w: 声称修订 %s 与正典解析修订 %s 不一致(内容修订是解析语义,不是调用方自报)", ErrInvalidProposal, claimedRevision, cand.Revision())
	}
	return cand, nil
}
