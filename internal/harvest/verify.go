package harvest

import (
	"context"
	"fmt"
	"time"

	"tally/internal/core"
)

// Verifier 是 harvest 对「实际拉起 + 健康检查」的期望(使用方定义,业务包互不导入):
// 真实实现把 internal/environment 的拉起能力包进来,由上层(cmd/web)注入——
// harvest 本身不 import environment。单测注入确定性 fake,离线可跑。
type Verifier interface {
	// Verify 按定义实际拉起并跑健康检查。全部通过才返回 nil error + 证据素材;
	// 拉起失败或健康检查不过 → 返回 error。
	Verify(ctx context.Context, def Definition) (VerifyResult, error)
}

// VerifyResult 是一次成功拉起留下的证据素材(由注入的 Verifier 产出)。
type VerifyResult struct {
	VerifiedAt time.Time         // 通过的时刻;零值时 harvest 用注入时钟补
	Combo      core.VersionCombo // 验证当时的完整版本组合(镜像 tag + 运行时版本)
	Record     string            // 机器记录(如 "health=3/3ok"),不是 LLM 的分析
}

// Proof 是验证通过后产出的证据(实现 core.VerificationProof)。
//
// 字段全部不导出、没有导出的构造函数:harvest 之外构造不出带内容的 Proof——
// 这是铁律 2(未验证不入库)的编译期基础。唯一产生路径是 Verify(经注入的 Verifier 真拉起)。
// 照 internal/failure 的 Proof 模式:那边有 test/arch 扫字段不导出,harvest 用自己的单测反射断言。
type Proof struct {
	at     time.Time
	combo  core.VersionCombo
	record string
}

// 静态断言:Proof 实现 core.VerificationProof。
var _ core.VerificationProof = Proof{}

// VerifiedAt 返回验证通过的时刻。
func (p Proof) VerifiedAt() time.Time { return p.at }

// VerifiedCombo 返回验证当时的完整版本组合(定义的有效性边界)。
func (p Proof) VerifiedCombo() core.VersionCombo { return p.combo }

// GateRecord 返回健康检查的机器记录。
func (p Proof) GateRecord() string { return p.record }

// Valid 报告证据是否由真实验证产出:零值(未经验证)无效。
func (p Proof) Valid() bool { return !p.at.IsZero() }

// newProof 从验证素材构造证据。未导出:包外造不出带内容的证据。
// now 在 VerifyResult.VerifiedAt 为零时兜底(验证时刻总得有)。
func newProof(res VerifyResult, now time.Time) Proof {
	at := res.VerifiedAt
	if at.IsZero() {
		at = now
	}
	return Proof{at: at.UTC(), combo: res.Combo, record: res.Record}
}

// Verify 经注入的 Verifier 实际拉起并跑健康检查,通过返回证据。
// Verifier 返回 error(拉起失败/健康检查不过)→ 证据零值、原样上抛。
func Verify(ctx context.Context, v Verifier, def Definition, now func() time.Time) (Proof, error) {
	res, err := v.Verify(ctx, def)
	if err != nil {
		return Proof{}, fmt.Errorf("验证 %s: %w", def.Name, err)
	}
	if now == nil {
		now = time.Now
	}
	return newProof(res, now()), nil
}
