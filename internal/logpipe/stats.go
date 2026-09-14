package logpipe

import (
	"sort"

	"tally/internal/core"
)

// Stats 聚合一批失败事件的精确指纹,支撑碰撞率实验(cmd/tally scan)。
//
// 为什么只看精确指纹:知识库只认 fp_exact,碰撞率回答的是「有多少失败其实可以
// 归到同一条已验证修复」——这正是整个 logpipe 有没有用的那个指标。洗不干净,
// 碰撞率趋近 0,表现为「这些错误好像各不相同」。
//
// 纯累加器:除 Add 外无副作用,同样的 Add 序列必得同样结果。
type Stats struct {
	Total       int
	exact       map[core.Fingerprint]int
	byToolchain map[string]*toolchainStats
}

type toolchainStats struct {
	total int
	exact map[core.Fingerprint]int
}

// NewStats 返回空统计。
func NewStats() *Stats {
	return &Stats{
		exact:       map[core.Fingerprint]int{},
		byToolchain: map[string]*toolchainStats{},
	}
}

// Add 记一个事件:它属于哪个工具链、精确指纹是什么。
func (s *Stats) Add(toolchain string, exact core.Fingerprint) {
	s.Total++
	s.exact[exact]++
	ts, ok := s.byToolchain[toolchain]
	if !ok {
		ts = &toolchainStats{exact: map[core.Fingerprint]int{}}
		s.byToolchain[toolchain] = ts
	}
	ts.total++
	ts.exact[exact]++
}

// Distinct 返回去重后的精确指纹数。
func (s *Stats) Distinct() int { return len(s.exact) }

// CollisionRate 返回碰撞率 =(总数 - 去重数) / 总数:有多大比例的失败事件与先前
// 事件撞了同一个精确指纹。越高说明去重越有效。总数为 0 时返回 0。
func (s *Stats) CollisionRate() float64 {
	if s.Total == 0 {
		return 0
	}
	return float64(s.Total-s.Distinct()) / float64(s.Total)
}

// ToolchainStat 是单个工具链的统计快照。
type ToolchainStat struct {
	Name          string
	Total         int
	Distinct      int
	CollisionRate float64
}

// ByToolchain 返回各工具链的统计,按工具链名排序保证输出确定。
func (s *Stats) ByToolchain() []ToolchainStat {
	out := make([]ToolchainStat, 0, len(s.byToolchain))
	for name, ts := range s.byToolchain {
		distinct := len(ts.exact)
		rate := 0.0
		if ts.total > 0 {
			rate = float64(ts.total-distinct) / float64(ts.total)
		}
		out = append(out, ToolchainStat{
			Name:          name,
			Total:         ts.total,
			Distinct:      distinct,
			CollisionRate: rate,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
