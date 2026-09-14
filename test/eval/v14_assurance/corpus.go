// Package v14assurance 是 A0-03 Assurance L1/L2/L3 Bake-off 的 spike harness
// (任务 docs/tasks/v1.4/A0/A0-03.md)。本包不是生产能力:
//   - 不改 internal/verification 或 internal/taskstate 的任何生产语义(AC4);
//   - 语料库在检视结果前冻结(AC1):FreezeHash 的常量由测试机械强制,
//     任何语料改动都会使 TestCorpusFrozen 失败;
//   - L2 的独立 LLM judge 经 Judge 接口注入;本环境无 LLM 凭据,实时
//     judge 一律记 NOT_RUN,绝不伪造判定。
//
// 方法论锚定 L3 Reference Pack(不复制外部代码):
//   - REF-SPECBENCH(Apache-2.0):visible/held-out 分离,测量两层捕获差;
//   - REF-EVILGENIE(MIT):judge 与 mutation 检测独立计分,split 确定性。
package v14assurance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// Case 是一条确定性检查用例。In 是按任务 Spec 编码的输入,Want 是期望输出。
// visible 用例对实现可见;held-out 用例只对最终验证可见。
type Case struct {
	Name string `json:"name"`
	In   string `json:"in"`
	Want string `json:"want"`
}

// Mutant 是一个注入单一缺陷的候选实现。DefectID 全局唯一,是跨层
// 「unique catch」比较的键。Impl 把 Case.In 映射到候选输出。
type Mutant struct {
	DefectID string                 `json:"defect_id"`
	Desc     string                 `json:"desc"`
	Impl     func(in string) string `json:"-"`
}

// Task 是语料库条目:一个验收语义 + 冻结的 visible/held-out 用例 +
// 注入缺陷集。
type Task struct {
	ID      string   `json:"id"`
	Spec    string   `json:"spec"`
	Visible []Case   `json:"visible"`
	HeldOut []Case   `json:"held_out"`
	Mutants []Mutant `json:"mutants"`
}

// corpus 是冻结语料库(2026-09-08 冻结)。每个任务至少含一个仅 visible
// 可捕的缺陷与一个仅 held-out 可捕的缺陷,使 L3\L1 差集可观测
// (SpecBench 的 visible/held-out gap 信号)。修改本切片即破坏冻结。
func corpus() []Task {
	return []Task{
		{
			ID:   "t1-clamp",
			Spec: "clamp: in=\"v,lo,hi\"(十进制整数,lo<=hi),out=把 v 夹到 [lo,hi] 后的十进制",
			Visible: []Case{
				{Name: "mid", In: "5,1,10", Want: "5"},
				{Name: "below", In: "0,1,10", Want: "1"},
			},
			HeldOut: []Case{
				{Name: "at-lo", In: "1,1,10", Want: "1"},
				{Name: "at-hi", In: "10,1,10", Want: "10"},
				{Name: "above", In: "11,1,10", Want: "10"},
			},
			Mutants: []Mutant{
				{DefectID: "t1-m1-hi-offbyone", Desc: "上界 off-by-one:v>=hi 时返回 hi-1", Impl: func(in string) string {
					v, lo, hi := parse3(in)
					if v < lo {
						return itoa(lo)
					}
					if v >= hi {
						return itoa(hi - 1)
					}
					return itoa(v)
				}},
				{DefectID: "t1-m2-no-lower", Desc: "忽略下界:只夹上界", Impl: func(in string) string {
					v, _, hi := parse3(in)
					if v > hi {
						return itoa(hi)
					}
					return itoa(v)
				}},
			},
		},
		{
			ID:   "t2-fib",
			Spec: "fib: in=n(0<=n<=20),out=fib(n),fib(0)=0,fib(1)=1",
			Visible: []Case{
				{Name: "five", In: "5", Want: "5"},
				{Name: "seven", In: "7", Want: "13"},
			},
			HeldOut: []Case{
				{Name: "zero", In: "0", Want: "0"},
				{Name: "one", In: "1", Want: "1"},
				{Name: "two", In: "2", Want: "1"},
			},
			Mutants: []Mutant{
				{DefectID: "t2-m1-fib0-eq-1", Desc: "fib(0)=1 基例错误", Impl: func(in string) string {
					n := atoi(in)
					if n == 0 {
						return "1"
					}
					return itoa(fib(n))
				}},
				{DefectID: "t2-m2-shifted", Desc: "整体位移:fib(n)=fib(n+1)", Impl: func(in string) string {
					return itoa(fib(atoi(in) + 1))
				}},
			},
		},
		{
			ID:   "t3-dedupe",
			Spec: "dedupe: in=逗号分隔列表,out=保序去重后的逗号分隔列表",
			Visible: []Case{
				{Name: "no-dupes", In: "a,b", Want: "a,b"},
			},
			HeldOut: []Case{
				{Name: "non-adjacent-repeat", In: "b,a,b", Want: "b,a"},
				{Name: "repeat-later", In: "a,b,a,b", Want: "a,b"},
				{Name: "all-same", In: "x,x,x", Want: "x"},
			},
			Mutants: []Mutant{
				{DefectID: "t3-m1-sorted", Desc: "去重但排序(破坏保序)", Impl: func(in string) string {
					return joinSortedUnique(splitList(in))
				}},
				{DefectID: "t3-m2-adjacent-only", Desc: "只去相邻重复", Impl: func(in string) string {
					return joinAdjacentDedupe(splitList(in))
				}},
			},
		},
		{
			ID:   "t4-kv",
			Spec: "kv: in=\"k=v\",out=\"k|v\";v 可含 '=' 与空值",
			Visible: []Case{
				{Name: "simple", In: "a=1", Want: "a|1"},
				{Name: "spaces-in-value", In: "k=v v", Want: "k|v v"},
			},
			HeldOut: []Case{
				{Name: "empty-value", In: "k=", Want: "k|"},
				{Name: "eq-in-value", In: "a=b=c", Want: "a|b=c"},
			},
			Mutants: []Mutant{
				{DefectID: "t4-m1-split-all", Desc: "按所有 '=' 切分(丢值尾段)", Impl: func(in string) string {
					k, v := splitFirst(in), ""
					parts := splitAllEq(in)
					if len(parts) > 1 {
						v = parts[1]
						_ = k
						return parts[0] + "|" + v
					}
					return in + "|"
				}},
				{DefectID: "t4-m2-strip-all-space", Desc: "剥掉值内全部空白", Impl: func(in string) string {
					k, v := splitKV(in)
					return k + "|" + stripSpaces(v)
				}},
			},
		},
		{
			ID:   "t5-avg",
			Spec: "avg: in=\"w;n1,n2,...\",out=窗口 w 的移动平均(逗号分隔,两位小数);w>len 时退化为整体平均",
			Visible: []Case{
				{Name: "exact", In: "2;1,3", Want: "2.00"},
				{Name: "slide", In: "2;1,2,4", Want: "1.50,3.00"},
			},
			HeldOut: []Case{
				{Name: "w-gt-len", In: "5;1,2,3", Want: "2.00"},
			},
			Mutants: []Mutant{
				{DefectID: "t5-m1-empty-on-overflow", Desc: "w>len 返回空而非整体平均", Impl: func(in string) string {
					w, ns := parseAvg(in)
					if w > len(ns) {
						return ""
					}
					return movingAvg(w, ns)
				}},
				{DefectID: "t5-m2-int-division", Desc: "整数除法截断", Impl: func(in string) string {
					w, ns := parseAvg(in)
					if w > len(ns) {
						w = len(ns)
					}
					return movingAvgInt(w, ns)
				}},
			},
		},
		{
			ID:   "t6-slug",
			Spec: "slug: in=短语,out=小写、空白转 '-'、连续 '-' 折叠、去首尾 '-'",
			Visible: []Case{
				{Name: "basic", In: "Hello World", Want: "hello-world"},
			},
			HeldOut: []Case{
				{Name: "multi-space", In: "a  b", Want: "a-b"},
				{Name: "edge-space", In: " a ", Want: "a"},
			},
			Mutants: []Mutant{
				{DefectID: "t6-m1-no-collapse", Desc: "不折叠连续 '-'", Impl: func(in string) string {
					return slugNoCollapse(in)
				}},
				{DefectID: "t6-m2-no-lower", Desc: "不小写", Impl: func(in string) string {
					return slugNoLower(in)
				}},
			},
		},
	}
}

// frozenCorpusHash 是 2026-09-08 冻结时语料库规范 JSON 的 SHA-256。
// TestCorpusFrozen 机械强制语料不再变化(AC1:先冻结后看结果)。
// 冻结历史:
//
//	v1 = bf4cae9d1896763b007e1478f3b7ffc051506db7b2af75d124bf74ae2d7cab00(2026-09-08)
//	v2 = e83b8bd3c7c2c61954dc8898ef1c08cc71358e273bfa56a9a50a6534cdf5f51c
//
// v2 修正案:t3-dedupe 的 visible 用例 "b,a,b" 本身就能捕获全部设计缺陷,
// 使「仅 held-out 可捕」缺陷不存在(sanity 测试 TestHeldOutOnlyDefectsExist
// 在任何层间比较结论产生前发现)。修正 = 把非相邻重复用例移入 held-out,
// visible 改为无重复样本。属语料设计修正,非结果调优;v1 产物已删除,
// 修正案完整记录于 docs/tasks/v1.4/research/A0-03-findings.md。
const frozenCorpusHash = "e83b8bd3c7c2c61954dc8898ef1c08cc71358e273bfa56a9a50a6534cdf5f51c"

// FreezeHash 计算语料库数据(不含函数体)的规范 JSON 摘要。
func FreezeHash() string {
	type caseDTO struct {
		Name, In, Want string
	}
	type mutantDTO struct {
		DefectID, Desc string
	}
	type taskDTO struct {
		ID, Spec         string
		Visible, HeldOut []caseDTO
		Mutants          []mutantDTO
	}
	var dto []taskDTO
	for _, t := range corpus() {
		d := taskDTO{ID: t.ID, Spec: t.Spec}
		for _, c := range t.Visible {
			d.Visible = append(d.Visible, caseDTO{c.Name, c.In, c.Want})
		}
		for _, c := range t.HeldOut {
			d.HeldOut = append(d.HeldOut, caseDTO{c.Name, c.In, c.Want})
		}
		for _, m := range t.Mutants {
			d.Mutants = append(d.Mutants, mutantDTO{m.DefectID, m.Desc})
		}
		dto = append(dto, d)
	}
	b, _ := json.Marshal(dto)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
