package assurance

import (
	"testing"
)

func TestLevelStringAndParseRoundTrip(t *testing.T) {
	for l := LevelL0; l <= LevelL4; l++ {
		got, err := ParseLevel(l.String())
		if err != nil {
			t.Fatalf("ParseLevel(%s) 意外错误: %v", l, err)
		}
		if got != l {
			t.Fatalf("round trip 失败: %s -> %d -> %s", l, l, got)
		}
		if !l.IsValid() {
			t.Fatalf("%s 应为合法档位", l)
		}
	}
}

func TestParseLevelInvalid(t *testing.T) {
	for _, s := range []string{"", "l1", "L5", "L-1", "one", "L2 "} {
		if l, err := ParseLevel(s); err == nil {
			t.Fatalf("ParseLevel(%q) 应失败,得到 %s", s, l)
		}
	}
}

func TestLevelIsValidRejectsOutOfRange(t *testing.T) {
	for _, l := range []Level{Level(-1), Level(5), Level(100)} {
		if l.IsValid() {
			t.Fatalf("%d 不应是合法档位", int(l))
		}
	}
}

func TestLevelOrdering(t *testing.T) {
	if !(LevelL0 < LevelL1 && LevelL1 < LevelL2 && LevelL2 < LevelL3 && LevelL3 < LevelL4) {
		t.Fatal("单轴档位序 L0<L1<L2<L3<L4 不成立")
	}
}

func TestSatisfiesSingleAxis(t *testing.T) {
	cases := []struct {
		achieved Level
		required Level
		want     bool
	}{
		{LevelL0, LevelL1, false},
		{LevelL1, LevelL1, true},
		{LevelL1, LevelL3, false},
		{LevelL3, LevelL3, true},
		{LevelL3, LevelL4, false},
		{LevelL4, LevelL4, true},
	}
	for _, c := range cases {
		if got := c.achieved.Satisfies(c.required); got != c.want {
			t.Fatalf("%s.Satisfies(%s) = %v, want %v", c.achieved, c.required, got, c.want)
		}
	}
}
