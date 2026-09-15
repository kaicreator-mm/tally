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
				"Current development program: v1.4.1",
			},
			required: []string{
				"Software Engineering Harness for Coding Agents",
				"v1.5.0",
				"Engineering Control Plane",
				"NOT RELEASE QUALIFIED",
				"Hidden Validation",
			},
		},
		{
			file: "docs/STATUS.md",
			forbidden: []string{
				"visible implementation          NOT STARTED",
				"Current program:** v1.4.1",
				"Rule-first governance UI is not implemented",
				"DeepSeek Harness Runtime Adapter is not implemented",
			},
			required: []string{
				"v1.5.0",
				"Engineering Control Plane",
				"ENV-BLOCKED",
				"NOT RELEASE QUALIFIED",
			},
		},
		{
			file: "docs/architecture/CURRENT_CODEBASE.md",
			forbidden: []string{
				"Current version program:** v1.4.1",
			},
			required: []string{
				"v1.5.0",
				"Active TaskDAG",
				"BrownfieldHarness",
				"NOT RELEASE QUALIFIED",
			},
		},
		{
			file: "docs/implementation/CURRENT_IMPLEMENTATION_STATUS.md",
			forbidden: []string{
				"Current program:** v1.4.1",
			},
			required: []string{
				"v1.5.0",
				"Issue #9",
				"ENV-BLOCKED",
				"NOT RELEASE QUALIFIED",
			},
		},
		{
			file: "docs/tasks/v1.5.0/RELEASE_QUALIFICATION.md",
			required: []string{
				"NOT RELEASE QUALIFIED",
				"Full visible regression",
				"Hidden Validation",
			},
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
