package v14assurance

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"time"
)

// Layer 标识 bake-off 比较的保证层。
const (
	LayerL1 = "L1" // 确定性 visible 检查
	LayerL2 = "L2" // L1 + 独立 LLM judge
	LayerL3 = "L3" // L2 + protected/held-out 验证
)

// Judge 是独立 LLM judge 的注入点(REF-EVILGENIE:judge 与 mutation 检测
// 独立计分)。judge 只看 Spec 与候选在 visible 用例上的行为样本,绝不接触
// held-out 用例/结果(包 §N:judge 不得见 held-out 名字或失败)。
type Judge func(ctx context.Context, spec string, visibleSamples []Case) (suspect bool, reason string, err error)

// LayerResult 是一次 (candidate, layer) 评估的结构化记录(RQ-05:身份完整)。
type LayerResult struct {
	TaskID        string `json:"task_id"`
	Candidate     string `json:"candidate"` // mutant defect id
	DefectID      string `json:"defect_id"`
	Layer         string `json:"layer"`
	Caught        bool   `json:"caught"`
	JudgeRan      bool   `json:"judge_ran"`    // L2 的 judge 是否真实执行
	JudgeStatus   string `json:"judge_status"` // ran | not_run(reason) | n/a
	LatencyMicros int64  `json:"latency_micros"`
	CostMicros    int64  `json:"cost_micros"` // 确定性检查恒 0
	CostKnown     bool   `json:"cost_known"`
	Runtime       string `json:"runtime"`     // 执行 runtime 身份
	Model         string `json:"model"`       // judge 模型;无 judge 时 "none"
	Environment   string `json:"environment"` // goos/goarch/go 版本
}

// environment 记录当前环境身份,随每条结果输出。
func environment() string {
	return fmt.Sprintf("%s/%s %s", runtime.GOOS, runtime.GOARCH, runtime.Version())
}

// evalCases 对候选运行一组确定性检查;caught = 任一用例失败。
func evalCases(m Mutant, cases []Case) (caught bool, micros int64) {
	start := time.Now()
	for _, c := range cases {
		if m.Impl(c.In) != c.Want {
			caught = true
		}
	}
	return caught, time.Since(start).Microseconds()
}

// visibleSamples 返回 judge 可见的行为样本(只有 visible 用例)。
func visibleSamples(t Task, m Mutant) []Case {
	var out []Case
	for _, c := range t.Visible {
		out = append(out, Case{Name: c.Name, In: c.In, Want: m.Impl(c.In)})
	}
	return out
}

// Evaluate 评估一个候选在一个保证层的结果。judge 仅在 L2/L3 且非 nil 时
// 执行;judge 为 nil 时 L2/L3 的 judge 贡献记 not_run,Caught 只反映
// 确定性部分(报告中据此标注 L2 捕获为下界/NOT_RUN,不冒充完整 L2)。
func Evaluate(ctx context.Context, t Task, m Mutant, layer string, judge Judge) LayerResult {
	res := LayerResult{
		TaskID: t.ID, Candidate: m.DefectID, DefectID: m.DefectID, Layer: layer,
		Runtime: "harness-local", Model: "none", Environment: environment(),
		JudgeStatus: "n/a", CostKnown: true,
	}
	visCaught, visMicros := evalCases(m, t.Visible)
	res.LatencyMicros += visMicros
	switch layer {
	case LayerL1:
		res.Caught = visCaught
	case LayerL2, LayerL3:
		caught := visCaught
		if judge == nil {
			res.JudgeRan = false
			res.JudgeStatus = "not_run(no LLM judge credentials in this environment)"
		} else {
			suspect, _, err := judge(ctx, t.Spec, visibleSamples(t, m))
			if err != nil {
				res.JudgeRan = false
				res.JudgeStatus = "not_run(judge error: " + err.Error() + ")"
			} else {
				res.JudgeRan = true
				res.JudgeStatus = "ran"
				res.Model = "injected"
				if suspect {
					caught = true
				}
			}
		}
		if layer == LayerL3 {
			hoCaught, hoMicros := evalCases(m, t.HeldOut)
			res.LatencyMicros += hoMicros
			caught = caught || hoCaught
		}
		res.Caught = caught
	}
	return res
}

// Summary 是一次 bake-off 的层间比较结果。
type Summary struct {
	ByLayer    map[string][]string `json:"by_layer"`    // layer → caught defect ids(排序)
	L2MinusL1  []string            `json:"l2_minus_l1"` // L2\L1 unique catches
	L3MinusL2  []string            `json:"l3_minus_l2"` // L3\L2 unique catches
	L3MinusL1  []string            `json:"l3_minus_l1"`
	L2Complete bool                `json:"l2_complete"` // judge 全部真实执行
	JudgeNote  string              `json:"judge_note"`
}

// Summarize 计算各层捕获集合与 unique-catch 差集(包 §N:不只看平均通过率)。
func Summarize(results []LayerResult) Summary {
	byLayer := map[string]map[string]bool{}
	judgeMissing := false
	for _, r := range results {
		if byLayer[r.Layer] == nil {
			byLayer[r.Layer] = map[string]bool{}
		}
		if r.Caught {
			byLayer[r.Layer][r.DefectID] = true
		}
		if (r.Layer == LayerL2 || r.Layer == LayerL3) && !r.JudgeRan {
			judgeMissing = true
		}
	}
	s := Summary{ByLayer: map[string][]string{}, L2Complete: !judgeMissing}
	for l, set := range byLayer {
		for d := range set {
			s.ByLayer[l] = append(s.ByLayer[l], d)
		}
		sort.Strings(s.ByLayer[l])
	}
	diff := func(a, b map[string]bool) []string {
		var out []string
		for d := range a {
			if !b[d] {
				out = append(out, d)
			}
		}
		sort.Strings(out)
		return out
	}
	s.L2MinusL1 = diff(byLayer[LayerL2], byLayer[LayerL1])
	s.L3MinusL2 = diff(byLayer[LayerL3], byLayer[LayerL2])
	s.L3MinusL1 = diff(byLayer[LayerL3], byLayer[LayerL1])
	if judgeMissing {
		s.JudgeNote = "L2 judge 未真实执行:L2 捕获只含确定性下界,L2\\L1 与 L3\\L2 差集记 NOT_RUN 语义,不作默认映射证据"
	}
	return s
}

// RunBakeoff 对冻结语料库执行全量 (task × mutant × layer) 评估。
func RunBakeoff(ctx context.Context, judge Judge) []LayerResult {
	var out []LayerResult
	for _, t := range corpus() {
		for _, m := range t.Mutants {
			for _, layer := range []string{LayerL1, LayerL2, LayerL3} {
				out = append(out, Evaluate(ctx, t, m, layer, judge))
			}
		}
	}
	return out
}
