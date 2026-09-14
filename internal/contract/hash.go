package contract

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Snapshot 身份哈希(P1-01 AC3:hash/identity 确定性):
// 由「哪两版内容被钉在一起」完全决定,与编译时间、进程、机器无关。
//
// TaskPin.RevisionHash 与 VerificationPin.Hash 各自已覆盖其源合同的全部实质
// 内容(前者含 task id/acceptance/scope 等,后者含 task_id/revision/acceptances
// /gates),因此 snapshot 规范文本只需按固定顺序排列这两个哈希 —— 内容对相同
// 则文本相同,任一源合同换版则文本必变。

// canonicalVersion 是 snapshot 规范文本的版本前缀。规范格式将来若变化,递增
// 它,避免不同代格式的哈希空间重叠(与 taskcontract/verification 的做法一致)。
const canonicalVersion = "contract/v1"

// canonical 返回 snapshot 的规范文本。
func (p Pins) canonical() string {
	return fmt.Sprintf("%s\ntask:%s\nverification:%s\n", canonicalVersion, p.Task.RevisionHash, p.Verification.Hash)
}

// hashPins 计算 snapshot 身份哈希,形如 "sha256:<hex>"。
func hashPins(p Pins) string {
	sum := sha256.Sum256([]byte(p.canonical()))
	return "sha256:" + hex.EncodeToString(sum[:])
}
