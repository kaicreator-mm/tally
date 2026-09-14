package logpipe

import (
	"math"
	"testing"

	"tally/internal/core"
)

func TestStatsCollisionRate(t *testing.T) {
	s := NewStats()
	// 4 个事件:A A A B → 总数 4,去重 2,碰撞率 (4-2)/4 = 0.5
	for _, fp := range []core.Fingerprint{"A", "A", "A", "B"} {
		s.Add("npm", fp)
	}
	if s.Total != 4 {
		t.Errorf("Total = %d, 想要 4", s.Total)
	}
	if s.Distinct() != 2 {
		t.Errorf("Distinct = %d, 想要 2", s.Distinct())
	}
	if got := s.CollisionRate(); math.Abs(got-0.5) > 1e-9 {
		t.Errorf("CollisionRate = %f, 想要 0.5", got)
	}
}

func TestStatsEmpty(t *testing.T) {
	s := NewStats()
	if s.Distinct() != 0 || s.CollisionRate() != 0 {
		t.Errorf("空统计应全为 0: Distinct=%d CollisionRate=%f", s.Distinct(), s.CollisionRate())
	}
}

// 全撞(去重有效)与全不撞(每个错误都不同)两个极端。
func TestStatsCollisionExtremes(t *testing.T) {
	all := NewStats()
	for i := 0; i < 5; i++ {
		all.Add("go", "SAME")
	}
	// 5 个全同:去重后 1 个,碰撞率 (5-1)/5 = 0.8(N 个全同时为 (N-1)/N,趋近 1)
	if math.Abs(all.CollisionRate()-0.8) > 1e-9 {
		t.Errorf("5 个全同碰撞率应为 0.8, 得 %f", all.CollisionRate())
	}

	none := NewStats()
	for _, fp := range []core.Fingerprint{"a", "b", "c"} {
		none.Add("go", fp)
	}
	if none.CollisionRate() != 0 {
		t.Errorf("全不撞时碰撞率应为 0, 得 %f", none.CollisionRate())
	}
}

// 各工具链分别统计,按名排序保证输出确定。
func TestStatsByToolchain(t *testing.T) {
	s := NewStats()
	s.Add("npm", "A")
	s.Add("npm", "A")
	s.Add("go", "B")
	s.Add("docker", "C")
	s.Add("docker", "D")

	got := s.ByToolchain()
	var names []string
	byName := map[string]ToolchainStat{}
	for _, ts := range got {
		names = append(names, ts.Name)
		byName[ts.Name] = ts
	}
	// 排序:docker < go < npm
	if len(names) != 3 || names[0] != "docker" || names[1] != "go" || names[2] != "npm" {
		t.Errorf("ByToolchain 顺序不确定: %v", names)
	}
	if npm := byName["npm"]; npm.Total != 2 || npm.Distinct != 1 || math.Abs(npm.CollisionRate-0.5) > 1e-9 {
		t.Errorf("npm 统计错: %+v", npm)
	}
	if docker := byName["docker"]; docker.Total != 2 || docker.Distinct != 2 || docker.CollisionRate != 0 {
		t.Errorf("docker 统计错: %+v", docker)
	}
}
