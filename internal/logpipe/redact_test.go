package logpipe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 正例:各类密钥必须被洗掉。用构造的假密钥当样本(docs/05-test-standards.md)。
func TestRedactCoarse(t *testing.T) {
	tests := []struct{ name, in, secret string }{
		{"GitHub PAT", "clone with ghp_aB3dE5gH7jK9mN1pQ3sT5vW7xY9z done", "ghp_aB3dE5gH7jK9mN1pQ3sT5vW7xY9z"},
		{"sk key", "using sk-aB3dE5gH7jK9mN1pQ3sT5vW7", "sk-aB3dE5gH7jK9mN1pQ3sT5vW7"},
		{"AWS key", "aws AKIAIOSFODNN7EXAMPLE ok", "AKIAIOSFODNN7EXAMPLE"},
		{"GitLab token", "glpat-abcdefghij0123456789xyz", "glpat-abcdefghij0123456789xyz"},
		{"Slack token", "xoxb-1234567890-abcdefghij", "xoxb-1234567890-abcdefghij"},
		{"JWT", "hdr eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abc123def456ghi789jkl", "eyJhbGciOiJIUzI1NiJ9"},
		{"Authorization", "Authorization: Bearer tokValue123", "tokValue123"},
		{"连接串口令", "postgres://admin:s3cr3tpw@db:5432/app", "s3cr3tpw"},
	}
	for _, tt := range tests {
		got := RedactCoarse(tt.in)
		if strings.Contains(got, tt.secret) {
			t.Errorf("%s: 密钥未被洗掉: %q -> %q", tt.name, tt.in, got)
		}
		if !strings.Contains(got, redacted) {
			t.Errorf("%s: 未出现占位符: %q -> %q", tt.name, tt.in, got)
		}
	}
}

// 细筛:key=value 口令与超长 base64。
func TestRedactFine(t *testing.T) {
	tests := []struct{ name, in, secret string }{
		{"password=", "db password=hunter2 connected", "hunter2"},
		{"token:", "config token: abc123xyz789", "abc123xyz789"},
		{"client_secret=", "client_secret=s3cr3tValueOK", "s3cr3tValueOK"},
		{"长 base64", "blob SGVsbG8gV29ybGQgdGhpcyBpcyBhIHZlcnkgbG9uZyBiYXNlNjQgc3RyaW5n end", "SGVsbG8gV29ybGQgdGhpcyBpcyBhIHZlcnkgbG9uZyBiYXNlNjQgc3RyaW5n"},
	}
	for _, tt := range tests {
		got := RedactFine(tt.in)
		if strings.Contains(got, tt.secret) {
			t.Errorf("%s: 密钥未被洗掉: %q -> %q", tt.name, tt.in, got)
		}
	}
}

// 反向:正常内容不能被误伤到改变含义(退出码、包名版本、错误类型必须保留)。
func TestRedactKeepsNormalContent(t *testing.T) {
	inputs := []string{
		"exit status 137",
		"Error: Cannot find module 'lodash' version 4.17.21",
		"TypeError: Cannot read properties of undefined (reading 'map')",
		"npm ERR! code ERESOLVE",
		"panic: runtime error: index out of range [5] with length 3",
	}
	for _, in := range inputs {
		if got := RedactFine(in); got != in {
			t.Errorf("正常内容被误伤: %q -> %q", in, got)
		}
	}
}

// MustBeRedacted:含密钥拒绝,干净放行,脱敏后应放行。
func TestMustBeRedacted(t *testing.T) {
	if err := MustBeRedacted("clone ghp_aB3dE5gH7jK9mN1pQ3sT5vW7xY9z"); err == nil {
		t.Error("含密钥的内容应被拒绝")
	}
	if err := MustBeRedacted("exit status 1"); err != nil {
		t.Errorf("干净内容应放行,却返回 %v", err)
	}
	raw := "Authorization: Bearer tok123 and password=hunter2"
	if err := MustBeRedacted(RedactFine(raw)); err != nil {
		t.Errorf("RedactFine 后仍被判定含密钥: %v", err)
	}
}

// 安全边界:提交进 testdata 的真实日志样本必须已脱敏。
// 遍历所有 .log,任何文件命中 MustBeRedacted 的密钥模式即失败 ——
// 防止以后有人把带密钥的日志直接丢进来(它会同时躺在 git、数据库和 LLM 那边)。
func TestTestdataIsRedacted(t *testing.T) {
	count := 0
	err := filepath.WalkDir("testdata", func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".log" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		count++
		if err := MustBeRedacted(string(b)); err != nil {
			t.Errorf("%s 疑似含密钥,提交前必须脱敏: %v", path, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("遍历 testdata: %v", err)
	}
	if count == 0 {
		t.Fatal("testdata 下没有任何 .log 样本,测试形同虚设")
	}
}
