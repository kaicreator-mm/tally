// Command docscheck guards CURRENT documentation against implementation-state drift.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type rule struct {
	file      string
	forbidden []string
	required  []string
}

func main() {
	rules := []rule{
		{
			file: "README.md",
			forbidden: []string{
				"v1.4.1 is authorized but not implemented",
			},
			required: []string{
				"Software Engineering Harness for Coding Agents",
				"v1.4.1",
				"Hidden Validation",
			},
		},
		{
			file: "docs/STATUS.md",
			forbidden: []string{
				"visible implementation          NOT STARTED",
				"Rule-first governance UI is not implemented",
				"DeepSeek Harness Runtime Adapter is not implemented",
			},
			required: []string{
				"v1.4.1",
				"P6-08",
				"PENDING_EXTERNAL",
			},
		},
		{
			file:     "docs/architecture/CURRENT_CODEBASE.md",
			required: []string{"v1.4.1"},
		},
		{
			file:     "docs/implementation/CURRENT_IMPLEMENTATION_STATUS.md",
			required: []string{"v1.4.1"},
		},
	}

	failed := false
	for _, r := range rules {
		b, err := os.ReadFile(filepath.Clean(r.file))
		if err != nil {
			fmt.Fprintf(os.Stderr, "docscheck: read %s: %v\n", r.file, err)
			failed = true
			continue
		}
		s := string(b)
		for _, bad := range r.forbidden {
			if strings.Contains(s, bad) {
				fmt.Fprintf(os.Stderr, "docscheck: %s contains stale claim %q\n", r.file, bad)
				failed = true
			}
		}
		for _, want := range r.required {
			if !strings.Contains(s, want) {
				fmt.Fprintf(os.Stderr, "docscheck: %s missing required current-state marker %q\n", r.file, want)
				failed = true
			}
		}
	}
	if failed {
		os.Exit(1)
	}
	fmt.Println("docscheck: PASS")
}
