package deps

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Change 是一条依赖变更:From 空 = 新增,To 空 = 移除,都非空 = 版本变化。
type Change struct {
	Package string
	From    string
	To      string
}

// ParseGoMod 把 go.mod 解析成「模块 → 版本」。覆盖块状 require(...)与
// 单行 require;indirect 依赖一并记(D-a5:升级信号不分直接间接)。
// replace 行不是依赖声明,跳过。
func ParseGoMod(b []byte) (map[string]string, error) {
	out := map[string]string{}
	inBlock := false
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "require (":
			inBlock = true
			continue
		case inBlock && line == ")":
			inBlock = false
			continue
		case inBlock:
			addRequire(out, line)
		case strings.HasPrefix(line, "require "):
			addRequire(out, strings.TrimPrefix(line, "require "))
		}
	}
	return out, nil
}

// addRequire 记一条「模块 版本」;版本缺失(如本地 replace 目标)的行跳过。
func addRequire(out map[string]string, line string) {
	if line == "" || strings.HasPrefix(line, "//") {
		return
	}
	fields := strings.Fields(line) // 尾部 // indirect 注释自然被 Fields 切开
	if len(fields) < 2 {
		return
	}
	out[fields[0]] = fields[1]
}

// ParsePackageLock 把 npm 锁文件(lockfileVersion 2/3)解析成「包名 → 版本」。
// 包名取 node_modules/ 分隔后的最后一段;嵌套同名依赖以深处为准(D-a6)。
// 根条目(空键)不是依赖。v1 锁文件没有 packages 段,明确不支持。
func ParsePackageLock(b []byte) (map[string]string, error) {
	var doc struct {
		Packages map[string]struct {
			Version string `json:"version"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("解析 package-lock.json: %w", err)
	}
	if doc.Packages == nil {
		return nil, fmt.Errorf("解析 package-lock.json:缺少 packages 段(仅支持 lockfileVersion 2/3)")
	}
	// 迭代序不定:嵌套同名依赖(浅处 node_modules/x 与深处 a/node_modules/x)
	// 按路径深度确定性取舍 —— 深处为准(D-a6),与遍历顺序无关。
	type entry struct {
		version string
		depth   int
	}
	best := map[string]entry{}
	for key, pkg := range doc.Packages {
		name, ok := lockPackageName(key)
		if !ok {
			continue
		}
		depth := strings.Count(key, lockSeg)
		if cur, seen := best[name]; !seen || depth >= cur.depth {
			best[name] = entry{version: pkg.Version, depth: depth}
		}
	}
	out := make(map[string]string, len(best))
	for name, e := range best {
		out[name] = e.version
	}
	return out, nil
}

// lockSeg 是 npm 锁文件键里的依赖目录段;包名取它最后一次出现之后的部分。
const lockSeg = "node_modules/"

// lockPackageName 从锁文件键提取包名;非依赖键(根条目)返回 ok=false。
func lockPackageName(key string) (string, bool) {
	i := strings.LastIndex(key, lockSeg)
	if i < 0 {
		return "", false
	}
	name := key[i+len(lockSeg):]
	return name, name != ""
}

// Diff 比较前后两份「包 → 版本」,产出按包名排序的变更列表(确定序,重跑可 diff)。
func Diff(before, after map[string]string) []Change {
	var out []Change
	for pkg, to := range after {
		from, existed := before[pkg]
		if !existed {
			out = append(out, Change{Package: pkg, To: to})
		} else if from != to {
			out = append(out, Change{Package: pkg, From: from, To: to})
		}
	}
	for pkg, from := range before {
		if _, still := after[pkg]; !still {
			out = append(out, Change{Package: pkg, From: from})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Package < out[j].Package })
	return out
}
