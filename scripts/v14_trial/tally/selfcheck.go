// selfcheck.go implements `tally self-check`: the P4-03 dry validation. It
// invokes NO codex binary and NO network; it validates, for every frozen task:
//
//  1. frozen-set loading — the same pinned hash, schema, fixture-integrity and
//     arm-neutrality checks the bare arm runs (shared discipline);
//  2. Contract composition — the composed task contract parses and validates
//     under the production taskcontract parser; the composed verification
//     contract parses and freezes under the production verification parser
//     (verification.Parse → Freeze discipline); the acceptance gate keys map
//     one-to-one onto composed visible bindings;
//  3. fixture repository composition — the two-commit repo builds, the
//     production artifacts.Load reads task + verification + policy bundle from
//     HEAD, contract.Compile + contract.Bind succeed and the compiled requires
//     / gate views resolve; the verification authority exists ONLY in HEAD
//     (never in the agent base tree);
//  4. verification probe wiring — every bug task's driver renders as valid Go
//     against its frozen oracle; the repro driver runs against the FROZEN
//     baseline in a sandbox and fails exactly the frozen baseline_failures;
//     every migration task's frozen oracle layout resolves inside the
//     provisioned tree and has registered data-integrity assertions;
//     live-DB execution stays trial-time (NOT_RUN without PostgreSQL).
//
// FAIL lines name composition defects and block the trial run.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"tally/internal/artifacts"
	"tally/internal/contract"
	"tally/internal/engineeringpolicy"
	"tally/internal/risk"
	"tally/internal/taskcontract"
	"tally/internal/verification"
	"tally/test/eval/v14_product"
)

const placeholderProbe = "<probe>"

type scReport struct {
	name   string
	ok     bool
	detail string
}

func cmdSelfCheck(args []string) error {
	fs := flag.NewFlagSet("self-check", flag.ExitOnError)
	repo, _ := commonFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	set, _, err := loadSet(*repo)
	if err != nil {
		return err
	}
	ok, err := runSelfCheck(set)
	if err != nil {
		return err
	}
	if !ok {
		os.Exit(1)
	}
	return nil
}

func runSelfCheck(set *frozenSet) (bool, error) {
	var reports []scReport
	fail := func(name, detail string) { reports = append(reports, scReport{name, false, detail}) }
	pass := func(name, detail string) { reports = append(reports, scReport{name, true, detail}) }

	pass("task-set: frozen manifest sha256 pinned ("+shortSHA(set.sha)+")",
		v14product.FreezeInfo(set.manifest))

	tmp, err := os.MkdirTemp("", "tally-v14-tally-selfcheck-")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(tmp)

	// The probe is the trial's own verification tooling; building it here
	// exercises its compile wiring without invoking any codex binary.
	probePath, err := ensureProbe(filepath.Join(tmp, "tools"))
	if err != nil {
		return false, fmt.Errorf("probe build failed: %w", err)
	}

	var composeFail int
	for i := range set.manifest.Tasks {
		task := &set.manifest.Tasks[i]
		name := "compose: " + task.TaskID
		if err := selfCheckTask(set, task, tmp, probePath, fail, pass); err != nil {
			composeFail++
			if !reported(reports, name) {
				fail(name, err.Error())
			}
		}
	}
	if composeFail == 0 {
		pass("compose: all tasks", fmt.Sprintf("%d tasks: task contract + verification contract parse under the production parsers; repos compose; contract.Compile/Bind succeed; verification authority confined to HEAD", len(set.manifest.Tasks)))
	}

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
		fmt.Printf("SELF-CHECK PASS: task-set loading, contract composition, fixture-repo composition and probe wiring validated; codex binary NOT invoked; real loops, live-DB evidence and judging remain trial-time (P4-03)\n")
	} else {
		fmt.Printf("SELF-CHECK FAIL: composition defects above block the trial run\n")
	}
	return okAll, nil
}

func reported(reports []scReport, name string) bool {
	for _, r := range reports {
		if r.name == name {
			return true
		}
	}
	return false
}

func shortSHA(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

// selfCheckTask runs the full dry composition for one task. fail/pass record
// task-level lines for the finer-grained stages; a returned error marks the
// task as failed overall.
func selfCheckTask(set *frozenSet, task *v14product.ManifestTask, tmpRoot, probePath string,
	fail func(name, detail string), pass func(name, detail string)) error {

	name := "compose: " + task.TaskID
	acc, err := loadAcceptanceDoc(set.root, task.TaskID)
	if err != nil {
		fail(name, "frozen acceptance unreadable: "+err.Error())
		return err
	}
	intentRel := "test/eval/v14_product/testdata/tasks/" + task.TaskID + "/intent.md"
	intent, err := os.ReadFile(filepath.Join(set.root, filepath.FromSlash(intentRel)))
	if err != nil {
		fail(name, "frozen intent unreadable: "+err.Error())
		return err
	}
	taskYAML, verifYAML, reqs, rec, err := composeInputs(set, task, acc, intent, placeholderProbe, set.root, "postgres://probe-placeholder:5433")
	if err != nil {
		fail(name, "composition failed: "+err.Error())
		return err
	}

	// 2a. composed task contract must parse + validate statically.
	tc, err := taskcontract.Parse(taskYAML)
	if err != nil {
		fail(name, "composed task contract rejected by the production parser: "+err.Error())
		return err
	}
	if err := tc.ValidateStatic(); err != nil {
		fail(name, "composed task contract fails static validation: "+err.Error())
		return err
	}
	if tc.ID != rec.ContractTaskID {
		fail(name, fmt.Sprintf("composed contract id %q != %q", tc.ID, rec.ContractTaskID))
		return fmt.Errorf("id mismatch")
	}
	// 2b. composed verification contract must parse and freeze.
	vc, err := verification.Parse(verifYAML)
	if err != nil {
		fail(name, "composed verification contract rejected by the production parser: "+err.Error())
		return err
	}
	frozen, err := vc.Freeze()
	if err != nil {
		fail(name, "composed verification contract cannot freeze: "+err.Error())
		return err
	}
	// 2c. gate keys ↔ bindings one-to-one, same order (deterministic).
	var bindIDs []string
	c := frozen.Contract()
	for _, a := range c.Acceptances {
		for _, b := range a.EvidenceBindings {
			bindIDs = append(bindIDs, b.ID)
		}
	}
	if strings.Join(bindIDs, "|") != strings.Join(acc.GateCond.AllOf, "|") {
		fail(name, fmt.Sprintf("bindings %v != frozen gate keys %v", bindIDs, acc.GateCond.AllOf))
		return fmt.Errorf("gate key mismatch")
	}
	// 2d. held-out discipline: this set is visible-only (L1 floor).
	if candidateverifyHasHeldOut(frozen) {
		fail(name, "composed contract unexpectedly contains held_out bindings")
		return fmt.Errorf("held-out in composed contract")
	}

	// 3. fixture repository composition + production loads + compiler seams.
	repoDir := filepath.Join(tmpRoot, "repo", task.TaskID)
	baseRev, _, err := buildTrialRepo(set, task, acc, rec, taskYAML, verifYAML, reqs, repoDir)
	if err != nil {
		fail(name, "fixture repo build failed: "+err.Error())
		return err
	}
	asnap, err := artifacts.Load(repoDir, "HEAD")
	if err != nil {
		fail(name, "artifacts.Load(HEAD): "+err.Error())
		return err
	}
	var tcLoaded taskcontract.TaskContract
	found := false
	for _, a := range asnap.Tasks {
		if a.Contract.ID == rec.ContractTaskID {
			tcLoaded, found = a.Contract, true
		}
	}
	if !found {
		fail(name, "HEAD snapshot lacks the composed task contract; rejections: "+rejectionsSummary(asnap.Rejected))
		return fmt.Errorf("contract not loaded")
	}
	var frozenLoaded *verification.Frozen
	for _, v := range asnap.Verifications {
		if v.Frozen.TaskID() == rec.ContractTaskID {
			frozenLoaded = v.Frozen
		}
	}
	if frozenLoaded == nil {
		fail(name, "HEAD snapshot lacks the composed verification contract; rejections: "+rejectionsSummary(asnap.Rejected))
		return fmt.Errorf("verification not loaded")
	}
	psnap, err := artifacts.LoadPolicies(repoDir, "HEAD")
	if err != nil || len(psnap.Rejected) != 0 || len(psnap.Policies) != 1 {
		detail := fmt.Sprintf("%v", err)
		if psnap != nil {
			detail = fmt.Sprintf("policies=%d rejected=%d %v", len(psnap.Policies), len(psnap.Rejected), psnap.Rejected)
		}
		fail(name, "policy bundle load: "+detail)
		return fmt.Errorf("policy load")
	}
	if psnap.Policies[0].Policy.ID != rec.BundleID {
		fail(name, fmt.Sprintf("bundle id %q != %q", psnap.Policies[0].Policy.ID, rec.BundleID))
		return fmt.Errorf("bundle mismatch")
	}
	var signals []risk.Signal
	if len(rec.RiskSignals) > 0 {
		basis := rec.RiskBasis
		if len(basis) == 0 {
			basis = []string{"self-check:" + task.TaskID}
		}
		signals = append(signals, risk.Signal{ID: risk.SignalID(rec.RiskSignals[0]), Basis: basis})
	}
	riskRes, err := risk.Derive(signals)
	if err != nil {
		fail(name, "risk derivation: "+err.Error())
		return err
	}
	snapAgg, err := contract.Compile(tcLoaded, frozenLoaded)
	if err != nil {
		fail(name, "contract.Compile: "+err.Error())
		return err
	}
	compiled, err := contract.Bind(snapAgg, []engineeringpolicy.Policy{psnap.Policies[0].Policy}, riskRes)
	if err != nil {
		fail(name, "contract.Bind: "+err.Error())
		return err
	}
	// The compiled requires/gate views must resolve (the loop's Prevention
	// context coordinate).
	_ = compiled.Requires()
	_ = compiled.GateKeys()
	// The verification authority must exist ONLY in HEAD, never in the agent
	// base tree.
	inBase, err := treeHasFile(repoDir, baseRev, ".tally/verifications/"+rec.ContractTaskID+".yaml")
	if err != nil {
		fail(name, "base tree inspection: "+err.Error())
		return err
	}
	if inBase {
		fail(name, "verification authority leaked into the agent base commit")
		return fmt.Errorf("verification authority in base tree")
	}

	// 4. probe wiring (the probe binary is trial tooling — no codex, no
	// network): bug tasks run the repro subcommand against the frozen
	// baseline; migration tasks run the wiring subcommand against the
	// composed repo tree.
	if err := runProbeWiring(probePath, set, task, repoDir); err != nil {
		fail("probe: "+task.TaskID, err.Error())
		return err
	}

	pass(name, fmt.Sprintf("contract %s: bundle=%s risk=%v bindings=%v; repo composed; verification authority confined to HEAD",
		rec.ContractTaskID, rec.BundleID, rec.RiskSignals, bindIDs))
	return nil
}

// runProbeWiring executes the probe's no-DB wiring surface for one task:
//   - bug_fix `repro`: the rendered driver compiles and the FROZEN baseline
//     fails exactly the frozen baseline_failures (sandbox, no candidate);
//   - database_migration `wiring`: the frozen oracle layout resolves in the
//     given tree and the integrity assertions are registered (live-DB
//     execution stays trial-time).
func runProbeWiring(probePath string, set *frozenSet, task *v14product.ManifestTask, tree string) error {
	sub := "wiring"
	if task.Harness == v14product.HarnessBugFix {
		sub = "repro"
	}
	out, err := runProbeCommand(sub, probePath, task.TaskID, set.root, tree, "")
	if err != nil {
		return fmt.Errorf("probe %s %s: %v: %s", sub, task.TaskID, err, truncateUTF8(out, 500))
	}
	return nil
}

// treeHasFile reports whether the given revision's tree contains rel.
func treeHasFile(repoDir, rev, rel string) (bool, error) {
	out, err := git(repoDir, "ls-tree", "-r", "--name-only", rev)
	if err != nil {
		return false, err
	}
	for _, p := range strings.Split(out, "\n") {
		if strings.TrimSpace(p) == rel {
			return true, nil
		}
	}
	return false, nil
}

func candidateverifyHasHeldOut(f *verification.Frozen) bool {
	c := f.Contract()
	for _, a := range c.Acceptances {
		for _, b := range a.EvidenceBindings {
			if b.VisibilityPolicy == verification.VisibilityHeldOut {
				return true
			}
		}
	}
	return false
}

// runProbeCommand executes one no-DB probe subcommand with structured argv in
// the given working directory.
func runProbeCommand(sub, probePath, taskID, repoRoot, cwd, dsn string) (string, error) {
	argv := []string{probePath, sub, "-task", taskID, "-repo", repoRoot}
	if dsn != "" {
		argv = append(argv, "-dsn", dsn)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...) //nolint:gosec // structured argv, explicit flags
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	return string(out), err
}
