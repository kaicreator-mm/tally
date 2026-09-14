package arch

import (
	"os"
	"strings"
	"testing"
)

// INV-6:go.mod 的直接依赖必须在白名单内(docs/04-code-standards.md)。
// 加依赖 = 改白名单 = 出现在 diff 里被看见。
func TestDepsWhitelist(t *testing.T) {
	whitelist := map[string]bool{
		"github.com/jackc/pgx/v5":     true,
		"github.com/pressly/goose/v3": true,
		"github.com/go-chi/chi/v5":    true,
		"github.com/go-git/go-git/v5": true,
		"golang.org/x/crypto":         true,
		"golang.org/x/sync":           true,
		"gopkg.in/yaml.v3":            true, // 环境定义 YAML 解析
	}

	data, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("读 go.mod: %v", err)
	}

	inRequire := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "require (") {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}
		var entry string
		switch {
		case inRequire:
			entry = line
		case strings.HasPrefix(line, "require "):
			entry = strings.TrimPrefix(line, "require ")
		default:
			continue
		}
		if strings.Contains(entry, "// indirect") || entry == "" {
			continue
		}
		fields := strings.Fields(entry)
		if len(fields) == 0 {
			continue
		}
		if mod := fields[0]; !whitelist[mod] {
			t.Errorf("直接依赖 %s 不在白名单(docs/04-code-standards.md),INV-6", mod)
		}
	}
}
