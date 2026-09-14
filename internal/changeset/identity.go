package changeset

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"tally/internal/core"
)

// patchIdentity 从改动文件推导 patch 身份:对规范编码取 sha256,带前缀便于排查。
//
// 规范编码:文件按路径排序(文件顺序不是语义),逐文件写入
// 长度前缀的路径、可执行标记、长度前缀的内容与 hunk 行坐标。
// 内容敏感:任何文件内容/路径/可执行性的变化都改变身份 ——
// 尤其可执行代码一变,候选身份必变(PRD §15.4:任何 executable-code mutation
// 都产生新 ChangeSet,需要新的适用验证证据)。这是机械推导,不经 LLM/Claim。
func patchIdentity(files []ChangedFile) string {
	sorted := append([]ChangedFile(nil), files...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })

	var b strings.Builder
	b.WriteString("changeset-patch-v1\n")
	for _, f := range sorted {
		fmt.Fprintf(&b, "F:%d:%s\n", len(f.Path), f.Path)
		fmt.Fprintf(&b, "E:%t\n", f.Executable)
		fmt.Fprintf(&b, "C:%d:%s\n", len(f.Content), f.Content)
		for _, h := range f.Hunks {
			fmt.Fprintf(&b, "H:%d,%d,%d,%d\n", h.OldStart, h.OldLines, h.NewStart, h.NewLines)
		}
		b.WriteString(".\n")
	}
	return "sha256:" + hex.EncodeToString(hashBytes(b.String()))
}

// deriveID 推导 ChangeSet 自身身份:project + base + candidate revision + patch 身份 + producer run。
// 全部输入不可变,故 ID 确定、可重算;同一 Run 的同一候选得到同一 ID,
// 不同 Run 的快照互不相同(producer run 参与推导,对应「一个 Run 0..1 个候选快照」)。
func deriveID(project core.ProjectID, base, candidate, patchID, run string) ChangeSetID {
	var b strings.Builder
	b.WriteString("changeset-v1\n")
	fmt.Fprintf(&b, "project:%s\n", project)
	fmt.Fprintf(&b, "base:%s\n", base)
	fmt.Fprintf(&b, "candidate:%s\n", candidate)
	fmt.Fprintf(&b, "patch:%s\n", patchID)
	fmt.Fprintf(&b, "run:%s\n", run)
	return ChangeSetID("sha256:" + hex.EncodeToString(hashBytes(b.String())))
}

func hashBytes(s string) []byte {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}
