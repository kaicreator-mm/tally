package logpipe

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"tally/internal/core"
)

// 正向(整套测试最重要的一组):同一错误的多份日志(不同时间/机器/路径前缀)
// 算出同一个精确指纹。它们不撞,知识库就永远查不到东西,而且不会报错。
func TestFingerprintExactCollides(t *testing.T) {
	runs := []string{
		"2026-07-21T10:33:02Z ERROR /home/alice/proj/src/a.ts:12:5 pid=48122\nTypeError: x is not a function",
		"2026-07-22T08:15:44Z ERROR /home/bob/proj/src/a.ts:12:5 pid=1099\nTypeError: x is not a function",
		"2026-07-23T23:01:09Z ERROR /tmp/build-xk29d/proj/src/a.ts:12:5 pid=7\nTypeError: x is not a function",
	}
	exact0, _ := Fingerprint(Normalize(runs[0]))
	for i, r := range runs[1:] {
		exact, _ := Fingerprint(Normalize(r))
		if exact != exact0 {
			t.Errorf("第 %d 份日志精确指纹不撞: %s vs %s", i+1, exact, exact0)
		}
	}
}

// 行号不同:精确指纹不撞(保留行号),模糊指纹撞(去行号)。
func TestFingerprintFuzzyIgnoresLineNumbers(t *testing.T) {
	a := Normalize("ERROR /home/alice/proj/src/a.ts:12:5\nTypeError: x is not a function")
	b := Normalize("ERROR /home/bob/proj/src/a.ts:88:3\nTypeError: x is not a function")
	exactA, fuzzyA := Fingerprint(a)
	exactB, fuzzyB := Fingerprint(b)
	if exactA == exactB {
		t.Error("行号不同,精确指纹应当不同")
	}
	if fuzzyA != fuzzyB {
		t.Errorf("行号不同,模糊指纹应当相同: %s vs %s", fuzzyA, fuzzyB)
	}
}

// 反向:不同根因的失败必须算出不同指纹。洗过头导致全撞成一个,同样是静默故障。
func TestFingerprintDifferentErrorsDiffer(t *testing.T) {
	pairs := [][2]string{
		{"TypeError: x is not a function", "ReferenceError: y is not defined"},
		{"exit status 137", "exit status 1"},
		{"npm ERR! code ERESOLVE", "npm ERR! code E404"},
	}
	for _, p := range pairs {
		e1, _ := Fingerprint(Normalize(p[0]))
		e2, _ := Fingerprint(Normalize(p[1]))
		if e1 == e2 {
			t.Errorf("不同失败撞成一个: %q 与 %q -> %s", p[0], p[1], e1)
		}
	}
}

// RuleVersion 必须参与指纹:同一规范化文本,换规则版本算出不同指纹。
// 没有这条,每优化一次规则,历史知识库就静默作废(新旧指纹混在一起)。
func TestFingerprintRuleVersionParticipates(t *testing.T) {
	norm := Normalize("TypeError: x is not a function")
	e1, f1 := fingerprintWith("1", norm)
	e2, f2 := fingerprintWith("2", norm)
	if e1 == e2 || f1 == f2 {
		t.Error("RuleVersion 未参与指纹:不同版本应算出不同指纹")
	}
	// 同版本稳定
	if a, b := fingerprintWith("1", norm); a != e1 || b != f1 {
		t.Error("同版本指纹不稳定")
	}
	// 导出的 Fingerprint 用的就是当前 RuleVersion
	ce, cf := fingerprintWith(RuleVersion, norm)
	if ge, gf := Fingerprint(norm); ge != ce || gf != cf {
		t.Error("Fingerprint 未使用当前 RuleVersion")
	}
}

// 整套测试最重要的一组(用真实样本):同一个原生模块 ABI 错误,在三个不同时间、
// 不同 pid、不同构建路径下记录的日志,经完整流水线后必须撞出**同一个** fp_exact,
// 且等于留存的黄金值 expected-fp.txt。它们不撞,知识库就永远查不到同一个修复,
// 而且这个故障是静默的 —— 只表现为「这些错误好像各不相同」。
func TestFingerprintNativeModuleABICollides(t *testing.T) {
	dir := filepath.Join("testdata", "npm", "native-module-abi")
	var fps []core.Fingerprint
	for i := 1; i <= 3; i++ {
		b, err := os.ReadFile(filepath.Join(dir, "run-"+strconv.Itoa(i)+".log"))
		if err != nil {
			t.Fatalf("读 run-%d.log: %v", i, err)
		}
		fps = append(fps, Process(string(b)).Exact)
	}
	for i := 1; i < len(fps); i++ {
		if fps[i] != fps[0] {
			t.Errorf("同一错误的第 %d 份日志精确指纹不撞: %s vs %s", i+1, fps[i], fps[0])
		}
	}

	wantB, err := os.ReadFile(filepath.Join(dir, "expected-fp.txt"))
	if err != nil {
		t.Fatalf("读 expected-fp.txt: %v", err)
	}
	want := core.Fingerprint(strings.TrimSpace(string(wantB)))
	if fps[0] != want {
		t.Errorf("精确指纹偏离黄金值: 得 %s, 期望 %s(规则变了应同步更新黄金值并说明)", fps[0], want)
	}
}
