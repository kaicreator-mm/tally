// Package holdout 定义 Holdout Evidence Store 抽象(P1-09,PRD §8.4 / INV-VER-05):
// held-out 证据的内容与可识别 verdict 存放在 Tally 控制的逻辑 Store 中,实现
// workspace 在机械上无法以任何项目文件系统路径寻址(PRD §8.4.1:普通 tracked
// 目录 + forbidden_paths 不构成隐藏机制,因此 Store 的 API 面拒绝一切路径语义)。
//
// 边界(P1):
//   - 本包只定义不透明引用(Ref)、记录元数据(Metadata)与按权威拆分的读取
//     能力接口(VerificationWriter / VerificationReader / Store)。具体存储
//     (受 Tally 控制的 filesystem root,位于受管项目 repo/worktree 之外,
//     决策锁 11)由后续任务在独立文件中实现本抽象;PRD 明确存储可替换。
//   - 读能力按权威分离(AC2):内容只能经 VerificationReader 读取,这是本包
//     唯一能从 Ref 解析出内容的类型。实现权威侧的代码在类型层面不存在任何
//     内容读取路径,按 PRD §8.4.3 连 store/ref 本身也不进入其 ContextPack;
//     控制面接线时只把窄能力交给对应权威,而不是交出整个 Store。
//   - 刻意不提供枚举/列表/恢复类 API(AC3):held-out 消费与恢复自动化按
//     决策锁 12 属 P2,本抽象的 API 面就是隔离 + 最终执行所需的最小闭包
//     (Put / Read / Stat),不做任何超出该闭包的产品承诺。
package holdout

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// 哨兵错误:调用方用 errors.Is 区分;包装文案只提供字段级上下文。
var (
	// ErrInvalidRef 表示 ref 不合法(空、超长、含路径分隔符/控制字符、
	// 前后空白等)。ref 不是路径,任何路径形态的输入都在此处被拒绝。
	ErrInvalidRef = errors.New("holdout: invalid ref")
	// ErrNotFound 表示 store 中不存在该 ref 对应的记录。
	ErrNotFound = errors.New("holdout: record not found")
	// ErrInvalidContent 表示 Put 收到空内容;held-out 证据记录不允许 0 字节。
	ErrInvalidContent = errors.New("holdout: empty content")
)

// MaxRefLen 是 ref 的防御性长度上限,防止实现把它拼进日志/错误信息时失控。
const MaxRefLen = 512

// Ref 是指向一条 held-out 证据记录的不透明引用(AC1)。
//
// 它刻意不是文件系统路径:ParseRef 拒绝路径分隔符与控制字符,因此一个合法
// Ref 无法表达「实现 workspace 内的某个位置」。Ref 与 VerificationContract
// 中 binding 的 evidence.ref 对应;只有持有 VerificationReader 的验证权威
// 代码才能把它解析回内容。
type Ref string

// String 返回 ref 原文。Ref 不携带任何可派生路径的语义,这里也没有任何
// 归一化:能通过 ParseRef 的串原样进出。
func (r Ref) String() string { return string(r) }

// ParseRef 校验并返回不透明 ref。合法 ref:非空、不超过 MaxRefLen、不含
// 路径分隔符(/ 或 \)、NUL 与其他控制字符、不是 "." 或 ".."、前后无空白。
func ParseRef(s string) (Ref, error) {
	switch {
	case s == "":
		return "", fmt.Errorf("%w: ref 不能为空", ErrInvalidRef)
	case len(s) > MaxRefLen:
		return "", fmt.Errorf("%w: ref 长度 %d 超过上限 %d", ErrInvalidRef, len(s), MaxRefLen)
	case s == "." || s == "..":
		return "", fmt.Errorf("%w: ref 不能是路径相对分量 %q", ErrInvalidRef, s)
	case s != strings.TrimSpace(s):
		return "", fmt.Errorf("%w: ref 有前后空白 %q", ErrInvalidRef, s)
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '/' || c == '\\' || c < 0x20 || c == 0x7f {
			return "", fmt.Errorf("%w: ref 含路径分隔符或控制字符(位置 %d)", ErrInvalidRef, i)
		}
	}
	return Ref(s), nil
}

// ContentDigest 返回内容摘要,格式与 VerificationContract 的 protection.
// baseline_hash 一致(sha256:<64 位十六进制>)。具体 Store 在 Put 时落成
// Metadata.Digest;最终执行侧用它对读回的内容做完整性校验。
func ContentDigest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Metadata 是一条 held-out 记录的元数据。它只含内容指纹与落成时间,
// 不含任何存放位置信息:实现侧即使看到 Metadata 也得不到文件系统地址。
type Metadata struct {
	Ref       Ref       // 不透明引用,与 Put 的返回值一致
	Size      int64     // 内容字节数
	Digest    string    // ContentDigest 格式的内容摘要
	CreatedAt time.Time // Store 落成记录的时间
}

// VerificationWriter 是验证权威侧的写入能力:在契约/基线 authoring 时把
// held-out 证据内容落入 Tally 控制的 Store,换取不透明 ref(evidence.ref)。
// 实现权威侧的接线永远拿不到这个能力。
type VerificationWriter interface {
	// Put 存储一份 held-out 证据内容,返回其不透明 ref。content 为空时返回
	// ErrInvalidContent。同一份内容是否产生同一 ref 由具体实现决定,调用方
	// 不得依赖该行为。
	Put(ctx context.Context, content []byte) (Ref, error)
}

// VerificationReader 是验证权威侧的读取能力:本包唯一能把 Ref 解析回内容的
// 类型,供 Tally 控制的最终验证执行(INV-VER-05 的 held-out 注入点)与
// exposure accounting 使用。它按能力值(capability)传递:实现权威侧组件
// 的类型签名里没有它,编译层面即无法触达内容。
type VerificationReader interface {
	// Read 返回 ref 对应的 held-out 内容。ref 不合法返回 ErrInvalidRef,
	// 记录不存在返回 ErrNotFound。实现不得在错误信息中回显存放位置。
	Read(ctx context.Context, ref Ref) ([]byte, error)
	// Stat 返回记录元数据,不读取内容;用于完整性核对与 exposure accounting。
	Stat(ctx context.Context, ref Ref) (Metadata, error)
}

// Store 是完整的验证权威 Store 面(Put/Read/Stat),由具体实现(决策锁 11:
// Tally 控制的 filesystem root,位于受管项目 repo/worktree 之外)提供。
// 控制面持有 Store,但按权威分发窄能力:验证/最终执行组件拿
// VerificationReader,authoring 组件拿 VerificationWriter;
// 实现权威侧组件两者都拿不到。
type Store interface {
	VerificationReader
	VerificationWriter
}
