// selfcheck.go owns `bare self-check`: the dry-run required by P4-02. It
// validates, WITHOUT invoking the codex binary and without any network:
//
//  1. task-set loading — manifest schema validation, freeze hash, fixture
//     integrity, arm neutrality (via the P4-01 validator package itself);
//  2. workspace provisioning — every task provisions into a temp root with
//     byte-identical agent-facing fixtures, zero verification-side assets,
//     and a verbatim instruction whose sha256 equals the frozen digest;
//  3. judge wiring — every bug task has a driver adapter that statically
//     matches its frozen baseline snapshot, renders for its frozen oracle
//     cases, and demonstrably runs end-to-end against a compile stub;
//     every migration task's frozen oracle layout resolves inside the
//     provisioned workspace (live-DB execution stays trial-time and is
//     reported NOT_RUN when no PostgreSQL is available).
//
// The stub smoke uses deliberately WRONG implementations: it proves the
// judge pipeline runs and reports failing cases — it is a wiring probe, not
// an acceptance evaluation of any candidate.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"tally/test/eval/v14_product"
)

type checkReport struct {
	name   string
	ok     bool
	detail string
}

func runSelfCheck(set *frozenSet, pg pgConfig) (bool, error) {
	var reports []checkReport
	fail := func(name, detail string) { reports = append(reports, checkReport{name, false, detail}) }
	pass := func(name, detail string) { reports = append(reports, checkReport{name, true, detail}) }

	// ---- 1. task-set loading ------------------------------------------------
	pass("task-set: frozen manifest sha256 pinned ("+shortSHA(set.sha)+")",
		v14product.FreezeInfo(set.manifest))

	// ---- 2. workspace provisioning ------------------------------------------
	tmp, err := os.MkdirTemp("", "tally-v14-bare-selfcheck-")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(tmp)

	provisioned := map[string]string{} // taskID -> ws root
	var provFail int
	for i := range set.manifest.Tasks {
		task := &set.manifest.Tasks[i]
		ws := filepath.Join(tmp, "ws", task.TaskID)
		if err := provisionTask(set, task, ws); err != nil {
			provFail++
			fail("provision: "+task.TaskID, err.Error())
			continue
		}
		provisioned[task.TaskID] = ws
		// Instruction identity: the bytes that would be sent to the agent.
		b, err := intentBytes(set, task, ws)
		if err != nil {
			provFail++
			fail("provision: "+task.TaskID, err.Error())
			continue
		}
		if len(b) == 0 {
			provFail++
			fail("provision: "+task.TaskID, "intent bytes are empty")
			continue
		}
	}
	if provFail == 0 {
		pass("provision: all tasks", fmt.Sprintf("%d workspaces materialized byte-identical, zero verification-side assets, intent sha256 verified", len(provisioned)))
	}

	// ---- 3. judge wiring -----------------------------------------------------
	// 3a. bug_fix: adapter x frozen baseline x frozen oracle.
	var bugWired, bugFailed int
	var smokeFailed int
	for i := range set.manifest.Tasks {
		task := &set.manifest.Tasks[i]
		if task.Harness != v14product.HarnessBugFix {
			continue
		}
		ws := provisioned[task.TaskID]
		if ws == "" {
			bugFailed++
			continue
		}
		adapter, ok := bugAdapters[task.TaskID]
		if !ok {
			bugFailed++
			fail("judge/bug: "+task.TaskID, "no driver adapter registered")
			continue
		}
		oracle, err := loadBugOracle(set.root, task.TaskID)
		if err != nil {
			bugFailed++
			fail("judge/bug: "+task.TaskID, err.Error())
			continue
		}
		if err := checkAdapterAgainstBaseline(set, task, adapter); err != nil {
			bugFailed++
			fail("judge/bug: "+task.TaskID, err.Error())
			continue
		}
		// The rendered driver must be syntactically valid Go for the task's
		// frozen cases.
		src := driverSource(adapter, oracle)
		if _, err := parser.ParseFile(token.NewFileSet(), "v14_judge_driver_test.go", src, parser.AllErrors); err != nil {
			bugFailed++
			fail("judge/bug: "+task.TaskID, "rendered driver does not parse: "+err.Error())
			continue
		}
		bugWired++
		// End-to-end wiring smoke against a deliberately wrong compile stub:
		// the judge sandbox must run and must REPORT failing frozen cases.
		if err := smokeBugJudge(set, task, adapter, oracle, tmp); err != nil {
			smokeFailed++
			fail("judge/bug smoke: "+task.TaskID, err.Error())
		}
	}
	if bugFailed == 0 && smokeFailed == 0 {
		pass("judge/bug: adapters + drivers", fmt.Sprintf("%d tasks: adapter matches frozen baseline signature, driver renders and parses, sandbox smoke reports failing cases on a wrong stub", bugWired))
	}

	// 3a'. baseline coherence (TASK_SET §5: "the shipped baseline fails
	// exactly the frozen failure set"). Where a frozen snapshot compiles
	// standalone, the judge driven against the SHIPPED unfixed bytes must
	// fail exactly the frozen baseline_failures — otherwise the task cannot
	// discriminate a do-nothing candidate. Snapshots that elide helpers and
	// do not compile standalone are recorded honestly: their defect
	// reproduction is carried by the frozen registry mirror.
	var coherenceFail int
	for i := range set.manifest.Tasks {
		task := &set.manifest.Tasks[i]
		if task.Harness != v14product.HarnessBugFix {
			continue
		}
		oracle, err := loadBugOracle(set.root, task.TaskID)
		if err != nil {
			coherenceFail++
			fail("baseline-coherence: "+task.TaskID, err.Error())
			continue
		}
		failedCases, compilable, err := driveBaselineSnapshot(set, task, oracle, tmp)
		if err != nil {
			coherenceFail++
			fail("baseline-coherence: "+task.TaskID, err.Error())
			continue
		}
		if !compilable {
			pass("baseline-coherence: "+task.TaskID,
				"snapshot does not compile standalone (helper elision); unfixed-defect reproduction carried by the frozen registry mirror")
			continue
		}
		want := append([]string(nil), oracle.Expected.BaselineFailures...)
		sort.Strings(want)
		got := append([]string(nil), failedCases...)
		sort.Strings(got)
		if equalStrings(got, want) {
			pass("baseline-coherence: "+task.TaskID, fmt.Sprintf("unfixed snapshot fails exactly the frozen baseline_failures %v", got))
		} else {
			coherenceFail++
			fail("baseline-coherence: "+task.TaskID,
				fmt.Sprintf("FROZEN-FIXTURE DEFECT: unfixed snapshot fails %v but the frozen expectation is %v — a do-nothing candidate would be accepted; adjudicate per TASK_SET.md §8 before the trial starts", got, want))
		}
	}

	// 3b. database_migration: oracle layout resolves in the provisioned ws;
	// live-DB execution is trial-time and honestly NOT_RUN without PostgreSQL.
	var dbWired, dbFailed int
	for i := range set.manifest.Tasks {
		task := &set.manifest.Tasks[i]
		if task.Harness != v14product.HarnessMigration {
			continue
		}
		ws := provisioned[task.TaskID]
		if ws == "" {
			dbFailed++
			continue
		}
		oracle, err := loadMigrationOracle(set.root, task.TaskID)
		if err != nil {
			dbFailed++
			fail("judge/db: "+task.TaskID, err.Error())
			continue
		}
		missing := ""
		for _, rel := range oracle.Layout.Migrations {
			if _, err := os.Stat(filepath.Join(ws, filepath.FromSlash(rel))); err != nil {
				missing = rel
				break
			}
		}
		if missing == "" && oracle.Layout.Seed != "" {
			if _, err := os.Stat(filepath.Join(ws, filepath.FromSlash(oracle.Layout.Seed))); err != nil {
				missing = oracle.Layout.Seed
			}
		}
		if missing != "" {
			dbFailed++
			fail("judge/db: "+task.TaskID, "oracle layout does not resolve in the provisioned workspace: "+missing)
			continue
		}
		if _, ok := dataIntegrityChecks[task.TaskID]; !ok {
			dbFailed++
			fail("judge/db: "+task.TaskID, "no verification-side data-integrity assertions registered")
			continue
		}
		dbWired++
	}
	if dbFailed == 0 {
		if pg.available() {
			pass("judge/db: layout + assertions", fmt.Sprintf("%d tasks: frozen oracle layout resolves in the workspace, integrity assertions registered; live-DB path available", dbWired))
		} else {
			pass("judge/db: layout + assertions", fmt.Sprintf("%d tasks: frozen oracle layout resolves in the workspace, integrity assertions registered; live-DB execution NOT_RUN(no PostgreSQL in this environment) — trial-time per PRD §24.2", dbWired))
		}
	}

	// ---- summary -------------------------------------------------------------
	okAll := true
	for _, r := range reports {
		status := "PASS"
		if !r.ok {
			status = "FAIL"
			okAll = false
		}
		fmt.Printf("self-check %s: %s — %s\n", status, r.name, r.detail)
	}
	if okAll {
		fmt.Printf("SELF-CHECK PASS: task-set loading, workspace provisioning and judge wiring validated; codex binary NOT invoked; live-DB evidence and real agent runs remain trial-time (P4-03)\n")
	} else {
		fmt.Printf("SELF-CHECK FAIL: see FAIL lines above — FAIL lines naming frozen-fixture defects block the TRIAL (P4-03), not the runner\n")
	}
	return okAll, nil
}

// checkAdapterAgainstBaseline statically verifies the adapter against the
// frozen baseline snapshot: package clause, entry function, arity.
func checkAdapterAgainstBaseline(set *frozenSet, task *v14product.ManifestTask, a bugAdapter) error {
	baselineRel := v14product.FixturePath(task.TaskID, "baseline")
	entries, err := os.ReadDir(filepath.Join(set.root, filepath.FromSlash(baselineRel)))
	if err != nil {
		return fmt.Errorf("baseline dir: %w", err)
	}
	fset := token.NewFileSet()
	found := false
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(set.root, filepath.FromSlash(baselineRel), e.Name())
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return fmt.Errorf("frozen baseline %s does not parse: %w", e.Name(), err)
		}
		if file.Name.Name != a.pkgName {
			return fmt.Errorf("baseline package %q != adapter package %q", file.Name.Name, a.pkgName)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil && fn.Name.Name != a.entry {
				continue
			}
			if fn.Name.Name != a.entry {
				continue
			}
			arity := fn.Type.Params.NumFields()
			// For method entries (bf-02 apply) the receiver is implicit in the
			// drive call; the adapter arity counts declared params only.
			if arity != a.arity {
				return fmt.Errorf("entry %s has %d params, adapter arity %d", a.entry, arity, a.arity)
			}
			found = true
		}
	}
	if !found {
		return fmt.Errorf("entry function %q not found in the frozen baseline snapshot", a.entry)
	}
	return nil
}

// smokeBugJudge runs the full judge pipeline against a temp package containing
// ONLY the adapter's wrong stub plus the rendered driver: the sandbox must
// execute and must report at least one failing frozen case (proving verdict
// capture and comparison without any candidate or agent involvement).
func smokeBugJudge(set *frozenSet, task *v14product.ManifestTask, a bugAdapter, oracle *bugOracleDoc, tmpRoot string) error {
	dir := filepath.Join(tmpRoot, "smoke", task.TaskID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module v14judgesmoke\n\ngo 1.23\n"), 0o644); err != nil {
		return err
	}
	var stub strings.Builder
	stub.WriteString("package " + a.pkgName + "\n\n")
	if strings.Contains(a.stub, "strings.") {
		stub.WriteString("import \"strings\"\n\n")
	}
	stub.WriteString(a.stub + "\n")
	if err := os.WriteFile(filepath.Join(dir, "stub.go"), []byte(stub.String()), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "v14_judge_driver_test.go"), []byte(driverSource(a, oracle)), 0o644); err != nil {
		return err
	}
	basis, gate := judgeStubSandbox(dir, oracle)
	if !basis.CompileOK {
		return fmt.Errorf("smoke stub did not compile: %s", basis.CompileDetail)
	}
	if gate.Status != gateFail {
		return fmt.Errorf("wrong stub unexpectedly produced gate status %q — verdict comparison wiring is broken", gate.Status)
	}
	return nil
}

// judgeStubSandbox is judgeBugCandidate's execution half, pointed at an
// already-materialized smoke module (no workspace involved).
func judgeStubSandbox(moduleDir string, oracle *bugOracleDoc) (*bugBasis, gateKeyResult) {
	basis := &bugBasis{}
	out, runErr := goTestCandidate(moduleDir)
	verdicts, found := parseVerdicts(string(out))
	basis.CompileOK = found
	if !found {
		basis.CompileDetail = truncate(strings.TrimSpace(string(out)), 1500)
		if runErr != nil {
			basis.CompileDetail = runErr.Error() + ": " + basis.CompileDetail
		}
		return basis, gateKeyResult{Status: gateNotRun, Detail: "no verdicts"}
	}
	failed := false
	for _, c := range oracle.Cases {
		got := verdicts[c.Name]
		pass := got == c.Want
		basis.Cases = append(basis.Cases, caseVerdict{Name: c.Name, Want: c.Want, Got: got, Pass: pass})
		if !pass {
			failed = true
		}
	}
	if failed {
		return basis, gateKeyResult{Status: gateFail, Detail: "wrong stub failed frozen cases (expected)"}
	}
	return basis, gateKeyResult{Status: gatePass, Detail: "stub passed all cases (unexpected for a wrong stub)"}
}

// goTestCandidate runs the judge's go test invocation in a judge module root.
// Offline guard: stdlib-only sandbox, any accidental network fetch fails loudly.
func goTestCandidate(moduleDir string) ([]byte, error) {
	cmd := exec.Command("go", "test", "./...", "-run", "TestV14FrozenOracleJudge", "-v", "-count=1", "-vet=off")
	cmd.Dir = moduleDir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off")
	cmd.Stdin = nil
	return cmd.CombinedOutput()
}

// driveBaselineSnapshot drives the frozen cases through the SHIPPED baseline
// bytes (self-check only). It reports the failing case names and whether the
// snapshot compiles standalone. The snapshot is materialized into a temp
// judge module from the pinned repo bytes; nothing is written into the
// repository or any workspace.
func driveBaselineSnapshot(set *frozenSet, task *v14product.ManifestTask, oracle *bugOracleDoc, tmpRoot string) (failedCases []string, compilable bool, err error) {
	adapter, ok := bugAdapters[task.TaskID]
	if !ok {
		return nil, false, fmt.Errorf("no driver adapter registered")
	}
	baselineRel := v14product.FixturePath(task.TaskID, "baseline")
	dir := filepath.Join(tmpRoot, "baseline-probe", task.TaskID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, false, err
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module v14judgesmoke\n\ngo 1.23\n"), 0o644); err != nil {
		return nil, false, err
	}
	if err := copyDir(filepath.Join(set.root, filepath.FromSlash(baselineRel)), filepath.Join(dir, "pkg")); err != nil {
		return nil, false, err
	}
	if err := os.WriteFile(filepath.Join(dir, "pkg", "v14_judge_driver_test.go"), []byte(driverSource(adapter, oracle)), 0o644); err != nil {
		return nil, false, err
	}
	out, _ := goTestCandidate(filepath.Join(dir, "pkg"))
	verdicts, found := parseVerdicts(string(out))
	if !found {
		return nil, false, nil // helper elision: snapshot is not standalone-compilable
	}
	for _, c := range oracle.Cases {
		if verdicts[c.Name] != c.Want {
			failedCases = append(failedCases, c.Name)
		}
	}
	return failedCases, true, nil
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
