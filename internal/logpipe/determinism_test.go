package logpipe

import "testing"

// INV-8:每个阶段必须是纯函数 —— 相同输入必然相同输出(byte 级)。
// 指纹不可复现 = 知识库永远查不到东西,且这个故障是静默的。
// 每加一个阶段,在这里补一条。
func TestDeterminism(t *testing.T) {
	sample := "Authorization: Bearer tok123\n" +
		"npm ERR! code E401\n" +
		"password=hunter2\n" +
		"clone ghp_aB3dE5gH7jK9mN1pQ3sT5vW7xY9z\n" +
		"exit status 1\n"

	if a, b := RedactCoarse(sample), RedactCoarse(sample); a != b {
		t.Error("RedactCoarse 不确定:两次输出不同")
	}
	if a, b := RedactFine(sample), RedactFine(sample); a != b {
		t.Error("RedactFine 不确定:两次输出不同")
	}
	if a, b := StripANSI(sample), StripANSI(sample); a != b {
		t.Error("StripANSI 不确定:两次输出不同")
	}
	if a, b := CollectString(sample), CollectString(sample); a != b {
		t.Error("CollectString 不确定:两次输出不同")
	}
	if a, b := Extract(sample), Extract(sample); a != b {
		t.Error("Extract 不确定:两次输出不同")
	}
	norm := Normalize(sample)
	if Normalize(sample) != norm {
		t.Error("Normalize 不确定:两次输出不同")
	}
	e1, f1 := Fingerprint(norm)
	e2, f2 := Fingerprint(norm)
	if e1 != e2 || f1 != f2 {
		t.Error("Fingerprint 不确定:两次输出不同")
	}
	if a, b := Process(sample), Process(sample); a != b {
		t.Error("Process 不确定:两次输出不同")
	}
}
