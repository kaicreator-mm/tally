package harvest

import (
	"context"
	"fmt"
	"time"

	"tally/internal/airuntime"
)

// maxRounds 是单个服务的验证预算:提案 → 拉起验证不过 → 再提案,最多三轮。
// 三轮还不行就跳过该服务并记录原因,不无限重试(不闷声烧钱,同 failure 的预算思想)。
const maxRounds = 3

// Harvester 编排「取来源 → 通用推理提案 → 实际拉起验证 → 写 catalog」。
// 依赖全部注入:Executor(AI Runtime 推理缝,composition root 接传输)、
// Verifier(拉起+健康检查)、Catalog(写 catalog)、Sources(按来源名取原始材料)。
// 业务包互不导入,协作由上层编排。
type Harvester struct {
	Executor  airuntime.Executor
	Verifier  Verifier
	Catalog   Catalog
	Sources   map[string]Source
	Now       func() time.Time
	MaxRounds int // 0 → maxRounds(3)
}

// Catalog 是 harvest 对「写已验证定义」的期望(使用方定义)。真实实现是 GitCatalog;
// 单测注入 fake 以断言「只有验证通过的才写」。
type Catalog interface {
	Store(ctx context.Context, def Definition, proof Proof) error
}

// Outcome 是一个服务的抽取结果。
type Outcome struct {
	Service string
	Stored  bool   // 已验证并写入 catalog
	Skipped bool   // 跳过(未写入)
	Reason  string // 跳过原因(Stored 时为空)
	Proof   Proof  // Stored 时的验证证据
}

func (h *Harvester) rounds() int {
	if h.MaxRounds > 0 {
		return h.MaxRounds
	}
	return maxRounds
}

func (h *Harvester) now() func() time.Time {
	if h.Now != nil {
		return h.Now
	}
	return time.Now
}

// Run 逐个处理抽取清单,返回每个服务的结果。单个服务失败不影响其他(跳过并记录)。
// 清单必须是显式给定的有限列表——harvest 不提供「全量批处理」入口(见 source.go 与 PRD 规则 ①)。
func (h *Harvester) Run(ctx context.Context, services []Service) []Outcome {
	outcomes := make([]Outcome, 0, len(services))
	for _, svc := range services {
		outcomes = append(outcomes, h.runOne(ctx, svc))
	}
	return outcomes
}

// runOne 处理单个服务:取来源 → 至多 rounds 轮(提案 → 验证)→ 写 catalog 或跳过记因。
func (h *Harvester) runOne(ctx context.Context, svc Service) Outcome {
	src := h.Sources[svc.Source]
	if src == nil {
		return Outcome{Service: svc.Name, Skipped: true, Reason: "无来源适配: " + svc.Source}
	}
	mat, err := src.Fetch(ctx, svc)
	if err != nil {
		return Outcome{Service: svc.Name, Skipped: true, Reason: "取来源材料失败: " + err.Error()}
	}

	rounds := h.rounds()
	now := h.now()
	var lastErr error
	for attempt := 1; attempt <= rounds; attempt++ {
		prop, perr := Propose(ctx, h.Executor, mat, attempt, lastErr)
		if perr != nil {
			lastErr = perr // 提案失败(如缺健康检查):不进验证,进下一轮
			continue
		}
		def := Definition{
			Name:         svc.Name,
			Kind:         KindDockerCompose,
			ComposeFile:  composeFileName,
			HealthChecks: prop.HealthChecks,
			Wait:         prop.Wait,
			Compose:      prop.Compose,
			Source:       svc.Source + ":" + svc.Ref,
		}
		proof, verr := Verify(ctx, h.Verifier, def, now)
		if verr != nil {
			lastErr = verr // 拉起/健康检查不过:进下一轮让 LLM 改
			continue
		}
		if serr := h.Catalog.Store(ctx, def, proof); serr != nil {
			return Outcome{Service: svc.Name, Skipped: true, Reason: "写入 catalog 失败: " + serr.Error()}
		}
		return Outcome{Service: svc.Name, Stored: true, Proof: proof}
	}
	return Outcome{
		Service: svc.Name, Skipped: true,
		Reason: fmt.Sprintf("%d 轮未通过验证,跳过: %v", rounds, lastErr),
	}
}
