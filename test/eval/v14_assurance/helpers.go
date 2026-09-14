package v14assurance

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// 语料库辅助函数:只服务 spike 语料的编解码,不属于生产路径。

func parse3(in string) (v, lo, hi int) {
	p := strings.Split(in, ",")
	return atoi(p[0]), atoi(p[1]), atoi(p[2])
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func itoa(n int) string { return strconv.Itoa(n) }

func fib(n int) int {
	a, b := 0, 1
	for i := 0; i < n; i++ {
		a, b = b, a+b
	}
	return a
}

func splitList(in string) []string {
	if in == "" {
		return nil
	}
	return strings.Split(in, ",")
}

func joinSortedUnique(xs []string) string {
	seen := map[string]bool{}
	var out []string
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

func joinAdjacentDedupe(xs []string) string {
	var out []string
	for i, x := range xs {
		if i > 0 && xs[i-1] == x {
			continue
		}
		out = append(out, x)
	}
	return strings.Join(out, ",")
}

func splitFirst(in string) string {
	if i := strings.Index(in, "="); i >= 0 {
		return in[:i]
	}
	return in
}

func splitAllEq(in string) []string { return strings.Split(in, "=") }

func splitKV(in string) (k, v string) {
	if i := strings.Index(in, "="); i >= 0 {
		return in[:i], in[i+1:]
	}
	return in, ""
}

func stripSpaces(s string) string {
	return strings.Join(strings.Fields(s), "")
}

func parseAvg(in string) (w int, ns []int) {
	i := strings.Index(in, ";")
	w = atoi(in[:i])
	for _, p := range strings.Split(in[i+1:], ",") {
		ns = append(ns, atoi(p))
	}
	return w, ns
}

func movingAvg(w int, ns []int) string {
	var out []string
	for i := 0; i+w <= len(ns); i++ {
		sum := 0
		for _, n := range ns[i : i+w] {
			sum += n
		}
		out = append(out, fmt.Sprintf("%.2f", float64(sum)/float64(w)))
	}
	if len(ns) > 0 && len(out) == 0 {
		sum := 0
		for _, n := range ns {
			sum += n
		}
		out = append(out, fmt.Sprintf("%.2f", float64(sum)/float64(len(ns))))
	}
	return strings.Join(out, ",")
}

func movingAvgInt(w int, ns []int) string {
	var out []string
	for i := 0; i+w <= len(ns); i++ {
		sum := 0
		for _, n := range ns[i : i+w] {
			sum += n
		}
		out = append(out, fmt.Sprintf("%.2f", float64(sum/w)))
	}
	return strings.Join(out, ",")
}

func slugNoCollapse(in string) string {
	s := strings.ToLower(in)
	var b strings.Builder
	for _, r := range s {
		if r == ' ' {
			b.WriteByte('-')
		} else {
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "-")
}

func slugNoLower(in string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.TrimSpace(in) {
		if r == ' ' {
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		} else {
			b.WriteRune(r)
			prevDash = false
		}
	}
	return b.String()
}
