package v14product

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Bug oracle registry (v14assurance corpus discipline).
//
// Each bug_fix task ships a deterministic baseline behavior (the defect) and
// a deterministic reference behavior (the fixed semantics). The frozen
// oracle cases on disk (oracle/oracle.json) are evaluated against BOTH:
// the baseline must fail exactly the frozen set of cases (reproducible
// failing baseline) and the reference must pass every case (stable oracle).
// The baseline snapshot under baseline/ documents the defect for the
// implementing side; these registry functions are the executable mirror used
// by the verification side only.

type bugTaskOracle struct {
	baseline  func(in string) string
	reference func(in string) string
}

var bugOracles = map[string]bugTaskOracle{
	"bf-01-token-bucket-offbyone": {
		baseline:  func(in string) string { return bf01Run(in, true) },
		reference: func(in string) string { return bf01Run(in, false) },
	},
	"bf-02-logout-session-invalidation": {
		baseline:  func(in string) string { return bf02Run(in, true) },
		reference: func(in string) string { return bf02Run(in, false) },
	},
	"bf-03-pagination-partial-page": {
		baseline:  func(in string) string { return bf03Run(in, true) },
		reference: func(in string) string { return bf03Run(in, false) },
	},
	"bf-04-date-validation-calendar": {
		baseline:  func(in string) string { return bf04Run(in, true) },
		reference: func(in string) string { return bf04Run(in, false) },
	},
	"bf-05-currency-rounding-halfup": {
		baseline:  func(in string) string { return bf05Run(in, true) },
		reference: func(in string) string { return bf05Run(in, false) },
	},
	"bf-06-email-dedupe-casefold": {
		baseline:  func(in string) string { return bf06Run(in, true) },
		reference: func(in string) string { return bf06Run(in, false) },
	},
}

// OracleForBugTask returns the registry entry for a task id.
func OracleForBugTask(taskID string) (bugTaskOracle, bool) {
	o, ok := bugOracles[taskID]
	return o, ok
}

// bf-01: in = "capacity,refilled,requestIndex" (1-based index).
// Defect: the boundary comparison admits one extra immediate request after
// the bucket is emptied (available >= 0 instead of > 0).
func bf01Run(in string, buggy bool) string {
	c, r, i := triple(in)
	limit := c + r
	if buggy {
		limit++
	}
	if i <= limit {
		return "allow"
	}
	return "reject"
}

// bf-02: ';'-separated ops create:<tok>|<user> / logout:<tok> / auth:<tok>.
// Defect: logout does not revoke the token.
func bf02Run(in string, buggy bool) string {
	valid := map[string]string{}
	var out []string
	for _, op := range strings.Split(in, ";") {
		switch {
		case strings.HasPrefix(op, "create:"):
			body := strings.TrimPrefix(op, "create:")
			k := strings.IndexByte(body, '|')
			if k < 0 {
				out = append(out, "reject")
				continue
			}
			valid[body[:k]] = body[k+1:]
			out = append(out, "created")
		case strings.HasPrefix(op, "logout:"):
			if !buggy {
				delete(valid, strings.TrimPrefix(op, "logout:"))
			}
			out = append(out, "logged-out")
		case strings.HasPrefix(op, "auth:"):
			if u, ok := valid[strings.TrimPrefix(op, "auth:")]; ok {
				out = append(out, "ok:"+u)
			} else {
				out = append(out, "reject")
			}
		default:
			out = append(out, "reject")
		}
	}
	return strings.Join(out, ";")
}

// bf-03: in = "total,pageSize,page" (1-based page).
// Defect: floor division drops the final partial page and every reachable
// page reports a full page of items.
func bf03Run(in string, buggy bool) string {
	total, pageSize, page := triple(in)
	totalPages := 0
	if total > 0 {
		if buggy {
			totalPages = total / pageSize
		} else {
			totalPages = (total + pageSize - 1) / pageSize
		}
	}
	if page < 1 || page > totalPages {
		return "out-of-range"
	}
	items := pageSize
	if !buggy {
		if rest := total - (page-1)*pageSize; rest < items {
			items = rest
		}
	}
	return fmt.Sprintf("items=%d,total_pages=%d", items, totalPages)
}

// bf-04: strict zero-padded "y-m-d" Gregorian validation.
// Defect: day range is checked as a flat 1..31, ignoring month lengths and
// leap years.
func bf04Run(in string, buggy bool) string {
	if len(in) != 10 || in[4] != '-' || in[7] != '-' ||
		!allDigits(in[:4]) || !allDigits(in[5:7]) || !allDigits(in[8:10]) {
		return "invalid:format"
	}
	y, _ := strconv.Atoi(in[:4])
	mo, _ := strconv.Atoi(in[5:7])
	d, _ := strconv.Atoi(in[8:10])
	if mo < 1 || mo > 12 {
		return "invalid:month"
	}
	if buggy {
		if d < 1 || d > 31 {
			return "invalid:day"
		}
		return "valid"
	}
	if d < 1 || d > monthLen(y, mo) {
		return "invalid:day"
	}
	return "valid"
}

func monthLen(y, mo int) int {
	switch mo {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	}
	if y%4 == 0 && (y%100 != 0 || y%400 == 0) {
		return 29
	}
	return 28
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// bf-05: signed micro-dollar integer -> dollars with exactly 2 decimals.
// Defect: truncation toward zero instead of rounding half away from zero.
func bf05Run(in string, buggy bool) string {
	n, err := strconv.ParseInt(strings.TrimSpace(in), 10, 64)
	if err != nil {
		return "invalid"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var cents int64
	if buggy {
		cents = n / 10000
	} else {
		cents = (n + 5000) / 10000
	}
	sign := ""
	if neg {
		sign = "-"
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

// bf-06: comma-separated addresses -> unique, lowercased, ascending order.
// Defect: comparison is case-sensitive, so case variants survive dedupe and
// the output is not normalized.
func bf06Run(in string, buggy bool) string {
	var xs []string
	for _, part := range strings.Split(in, ",") {
		a := strings.TrimSpace(part)
		if buggy {
			xs = append(xs, a)
		} else {
			xs = append(xs, strings.ToLower(a))
		}
	}
	sort.Strings(xs)
	var out []string
	for i, a := range xs {
		if i > 0 && xs[i-1] == a {
			continue
		}
		out = append(out, a)
	}
	return strings.Join(out, ",")
}

func triple(in string) (int, int, int) {
	p := strings.Split(strings.TrimSpace(in), ",")
	if len(p) != 3 {
		return 0, 0, 0
	}
	a, _ := strconv.Atoi(strings.TrimSpace(p[0]))
	b, _ := strconv.Atoi(strings.TrimSpace(p[1]))
	c, _ := strconv.Atoi(strings.TrimSpace(p[2]))
	return a, b, c
}
