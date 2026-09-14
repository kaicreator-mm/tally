package failure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"tally/internal/core"
)

// depAlignWindow 是「失败与依赖升级对齐」的窗口:失败发生前这段时间内的
// 依赖变更视为可疑的上游信号。宽了会误判、窄了会漏,7 天是自用规模下的折中。
const depAlignWindow = 7 * 24 * time.Hour

// Handle 跑一轮诊断:查库 → 预算 → 复现 → 提案 → 四道闸 → 归类 → 记账。
//
// 契约:事件已被 RecordEvent 记录(索引行存在);本函数不碰事实面,只读改索引面。
//
// 铁律 1 的落点:一轮里 LLM 至多被调一次且只提候选;所有「算不算数」——
// 是否命中、是否复现、修复过没过关、归哪个出口、预算到没到 —— 全是确定性代码的回答。
func Handle(ctx context.Context, d Deps, ev Event) (Conclusion, error) {
	var leads []core.Lead

	// 查库:精确指纹命中 → 直接返回修复,一次 LLM 都不调(PRD 第四部分流程)。
	hit, ok, err := d.KB.LookupExact(ctx, ev.Exact, ev.Combo)
	if err != nil {
		return Conclusion{}, fmt.Errorf("查知识库: %w", err)
	}
	if ok {
		if !hit.Stale {
			fix := hit.Fix
			return Conclusion{Kind: OutcomeFixed, Fix: &fix}, nil
		}
		// 命中但过期:降级成线索,继续走诊断流程。
		leads = append(leads, core.Lead{
			ExactFP:       ev.Exact,
			Status:        string(OutcomeFixed),
			VerifiedCombo: hit.VerifiedCombo,
			Stale:         true,
		})
	}

	// 模糊指纹 → 只是线索。
	fuzzyLeads, err := d.KB.LookupFuzzy(ctx, ev.Fuzzy, ev.Combo)
	if err != nil {
		return Conclusion{}, fmt.Errorf("召回模糊指纹: %w", err)
	}
	leads = append(leads, fuzzyLeads...)

	// 预算闸门:终态出口或三轮用尽 → 直接返回结论,零 LLM 调用。
	stat, err := d.Stats.GetStat(ctx, ev.Exact)
	if err != nil && !errors.Is(err, ErrNoStat) {
		return Conclusion{}, err
	}
	found := err == nil
	dec := ConsultBudget(found, stat)
	if dec.Concluded != nil {
		return Conclusion{Kind: *dec.Concluded, FixRef: stat.FixRef}, nil
	}
	if dec.Archived {
		return Conclusion{Archived: true}, nil
	}
	if err := d.Stats.BumpAttempts(ctx, ev.Exact); err != nil {
		return Conclusion{}, err
	}

	// 缺复现上下文(Repro 无 Argv):G1/G2 无从跑,本轮不提候选也不验证,跳过(D3)。
	// 次数照计:预算闸门仍然生效,三轮后归档;知识库精确命中在本函数入口已短路,不受影响。
	if len(ev.Repro.Argv) == 0 {
		return Conclusion{SkippedRepro: true}, nil
	}

	// G1:复现不稳 → flaky,不调 LLM(不给偶发故障花一分钱)。
	stable, err := ConfirmRepro(ctx, d.Runner, RunSpec{Command: ev.Repro})
	if err != nil {
		return Conclusion{}, err
	}
	if !stable {
		kind := Classify(Evidence{ReproStable: false})
		if err := d.Stats.SetClassification(ctx, ev.Exact, kind); err != nil {
			return Conclusion{}, err
		}
		return Conclusion{Kind: kind}, nil
	}

	// 全流程唯一的推理调用:提候选(送出前已脱敏)。
	cand, err := Diagnose(ctx, d.Executor, ev, leads)
	if err != nil {
		return Conclusion{}, err
	}

	// spec_defect 提议 → 确定性闸门:条款确实存在才算数,否则当没提过,继续修。
	if cand.SpecDefect != nil && cand.SpecDefect.ClauseID != "" && d.SpecSink != nil {
		if err := d.SpecSink.ReportSpecDefect(ctx, ev.ProjectID, cand.SpecDefect.ClauseID, cand.SpecDefect.Ambiguity); err == nil {
			kind := Classify(Evidence{ReproStable: true, SpecDefectConfirmed: true})
			if err := d.Stats.SetClassification(ctx, ev.Exact, kind); err != nil {
				return Conclusion{}, err
			}
			return Conclusion{Kind: kind}, nil
		}
		// 条款不存在:提议不成立,落到修复验证。
	}

	// G4(范围)→ G2(修复后连绿)→ G3(其余目标)。
	proof, verifyErr := VerifyCandidate(ctx, d.Runner, VerifyInput{
		ReproCommand: ev.Repro,
		Candidate:    cand,
		OtherTargets: ev.OtherTargets,
		Combo:        ev.Combo,
		Reds:         reproRuns,
		Now:          d.Now,
	})
	switch {
	case errors.Is(verifyErr, ErrFixTouchesTests):
		// 改到验证用文件:直接拒绝,原样返回给调用方(不是暂停)。
		return Conclusion{}, verifyErr
	case errors.Is(verifyErr, ErrFixTouchesSource):
		// 改到应用源码:暂停等人确认,不归类、不入库。
		return Conclusion{Paused: true}, nil
	case verifyErr != nil && !errors.Is(verifyErr, ErrFixNotGreen) && !errors.Is(verifyErr, ErrOthersAffected):
		return Conclusion{}, verifyErr
	}

	// 归类:全部依据确定性信号。
	depAligned, err := d.Stats.HasRecentDepChange(ctx, ev.ProjectID, ev.OccurredAt.Add(-depAlignWindow))
	if err != nil {
		return Conclusion{}, err
	}
	kind := Classify(Evidence{
		ReproStable:  true,
		FixVerified:  verifyErr == nil,
		CrossProject: found && len(stat.AffectedProjects) > 1,
		DepAligned:   depAligned,
	})

	switch kind {
	case OutcomeFixed:
		// core.Fix.Repro 是 string(core 不属本任务范围):落结构化命令的规范 JSON,
		// 不退化成 shell 字符串(PRD §21.6),可原样 Unmarshal 回 CommandSpec。
		repro, err := json.Marshal(ev.Repro)
		if err != nil {
			return Conclusion{}, fmt.Errorf("序列化复现命令: %w", err)
		}
		fix := core.Fix{Repro: string(repro), Summary: cand.Summary, Changes: cand.Changes}
		ref, err := d.KB.Store(ctx, ev.Exact, fix, proof)
		if err != nil {
			return Conclusion{}, fmt.Errorf("已验证修复入库: %w", err)
		}
		if err := d.Stats.SetFix(ctx, ev.Exact, ref, proof.VerifiedAt(), proof.VerifiedCombo()); err != nil {
			return Conclusion{}, err
		}
		return Conclusion{Kind: OutcomeFixed, FixRef: ref, Fix: &fix}, nil
	case OutcomeInfra, OutcomeUpstream:
		if err := d.Stats.SetClassification(ctx, ev.Exact, kind); err != nil {
			return Conclusion{}, err
		}
		return Conclusion{Kind: kind}, nil
	default:
		// 候选没过验证,本轮无出口;要不要再来一轮由预算管(下一轮入口会查次数)。
		return Conclusion{}, nil
	}
}
