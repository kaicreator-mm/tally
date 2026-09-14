package verification

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// fullContractYAML 与 PRD §8.11 的 VerificationContract 示例一一对应。
const fullContractYAML = `
task_id: TASK-023
revision: 6

acceptances:
  - id: AC-01
    statement: stale leases can be reclaimed
    independence_required: source
    evidence_bindings:
      - id: EV-AC01-01
        authoring_mode: pre_commit
        visibility_policy: visible
        independence_basis:
          source: verification_authority
          information: visible
        evidence:
          type: test
          path: internal/jobs/lease_owner_test.go
          selector: TestReclaimStaleLease
        protection:
          mode: immutable
          baseline_hash: sha256:a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2
        baseline:
          expectation:
            status: fail
            preferred_class: assertion
          observed:
            status: fail
            class: assertion
            fingerprint: sha256:0f8d6a9c1b2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f80
            commit: abc123
        qualification:
          baseline_sensitivity: strong
          change_overlap: required
          review_mode: compact

  - id: AC-02
    statement: active leases must never be reclaimed
    independence_required: information
    evidence_bindings:
      - id: EV-AC02-H1
        authoring_mode: pre_commit
        visibility_policy: held_out
        independence_basis:
          source: verification_authority
          information: held_out
        evidence:
          type: test
          store: holdout
          ref: jobs/lease_safety#TestNeverReclaimLiveLease
        protection:
          mode: immutable
          baseline_hash: sha256:b2c3d4e5f6b2c3d4e5f6b2c3d4e5f6b2c3d4e5f6b2c3d4e5f6b2c3d4e5f6b2c3
        execution_protocol:
          repetitions: 1
          aggregation: all_pass
        baseline:
          expectation:
            status: pass
          observed:
            status: pass
            commit: abc123
        qualification:
          baseline_sensitivity: none
          change_overlap: required
          review_mode: explicit
          rationale: Negative safety invariant; held out for information independence.

  - id: AC-03
    statement: retries must respect max_attempts
    independence_required: none
    evidence_bindings:
      - id: EV-AC03-01
        authoring_mode: pre_commit
        visibility_policy: visible
        evidence:
          type: validator
          validator: retry-budget-v1

implementation_runtime_requirements:
  session_isolation: true
  held_out_unreachable: true
  conformance_class: verification-isolation-v1

required_gates:
  build:
    required: true
    command:
      argv: [go, build, ./...]
      cwd: .
      timeout: 120s
  vet:
    required: false
`

// fullContractYAMLReordered 与 fullContractYAML 内容完全相同,仅 acceptances 与
// required_gates 的书写顺序不同 → 规范化哈希必须一致(AC4 稳定性)。
const fullContractYAMLReordered = `
revision: 6
task_id: TASK-023

implementation_runtime_requirements:
  held_out_unreachable: true
  session_isolation: true
  conformance_class: verification-isolation-v1

required_gates:
  vet:
    required: false
  build:
    required: true
    command:
      timeout: 120s
      cwd: .
      argv: [go, build, ./...]

acceptances:
  - id: AC-03
    statement: retries must respect max_attempts
    independence_required: none
    evidence_bindings:
      - id: EV-AC03-01
        authoring_mode: pre_commit
        visibility_policy: visible
        evidence:
          type: validator
          validator: retry-budget-v1
  - id: AC-02
    statement: active leases must never be reclaimed
    independence_required: information
    evidence_bindings:
      - id: EV-AC02-H1
        authoring_mode: pre_commit
        visibility_policy: held_out
        independence_basis:
          source: verification_authority
          information: held_out
        evidence:
          type: test
          store: holdout
          ref: jobs/lease_safety#TestNeverReclaimLiveLease
        protection:
          mode: immutable
          baseline_hash: sha256:b2c3d4e5f6b2c3d4e5f6b2c3d4e5f6b2c3d4e5f6b2c3d4e5f6b2c3d4e5f6b2c3
        execution_protocol:
          repetitions: 1
          aggregation: all_pass
        baseline:
          expectation:
            status: pass
          observed:
            status: pass
            commit: abc123
        qualification:
          baseline_sensitivity: none
          change_overlap: required
          review_mode: explicit
          rationale: Negative safety invariant; held out for information independence.
  - id: AC-01
    statement: stale leases can be reclaimed
    independence_required: source
    evidence_bindings:
      - id: EV-AC01-01
        authoring_mode: pre_commit
        visibility_policy: visible
        independence_basis:
          source: verification_authority
          information: visible
        evidence:
          type: test
          path: internal/jobs/lease_owner_test.go
          selector: TestReclaimStaleLease
        protection:
          mode: immutable
          baseline_hash: sha256:a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2
        baseline:
          expectation:
            status: fail
            preferred_class: assertion
          observed:
            status: fail
            class: assertion
            fingerprint: sha256:0f8d6a9c1b2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f80
            commit: abc123
        qualification:
          baseline_sensitivity: strong
          change_overlap: required
          review_mode: compact
`

func mustParse(t *testing.T, data string) *Contract {
	t.Helper()
	c, err := Parse([]byte(data))
	if err != nil {
		t.Fatalf("Parse: unexpected error: %v", err)
	}
	return c
}

// AC1:binding 的 provenance/source basis 与 visible/held_out 都有类型化表示且可解析。
func TestParseRepresentsProvenanceSourceBasisAndVisibility(t *testing.T) {
	c := mustParse(t, fullContractYAML)

	if c.TaskID != "TASK-023" || c.Revision != 6 {
		t.Fatalf("got task_id=%q revision=%d", c.TaskID, c.Revision)
	}
	if len(c.Acceptances) != 3 {
		t.Fatalf("acceptances = %d, want 3", len(c.Acceptances))
	}

	ac01 := c.Acceptances[0]
	if ac01.IndependenceRequired != IndependenceSource {
		t.Fatalf("AC-01 independence_required = %q, want source", ac01.IndependenceRequired)
	}
	b1 := ac01.EvidenceBindings[0]
	if b1.AuthoringMode != AuthoringPreCommit {
		t.Fatalf("EV-AC01-01 authoring_mode = %q, want pre_commit (INV-VER-01)", b1.AuthoringMode)
	}
	if b1.VisibilityPolicy != VisibilityVisible {
		t.Fatalf("EV-AC01-01 visibility_policy = %q, want visible", b1.VisibilityPolicy)
	}
	if b1.Basis == nil || b1.Basis.Source != SourceVerificationAuthority || b1.Basis.Information != InfoVisible {
		t.Fatalf("EV-AC01-01 independence_basis = %+v, want source=verification_authority information=visible", b1.Basis)
	}
	if b1.Evidence.Type != EvidenceTest || b1.Evidence.Path != "internal/jobs/lease_owner_test.go" || b1.Evidence.Selector != "TestReclaimStaleLease" {
		t.Fatalf("EV-AC01-01 evidence = %+v", b1.Evidence)
	}
	if b1.Protection == nil || b1.Protection.Mode != ProtectionImmutable {
		t.Fatalf("EV-AC01-01 protection = %+v, want immutable", b1.Protection)
	}

	// held_out binding:store/ref 引用,不含 repo path。
	ac02 := c.Acceptances[1]
	if ac02.IndependenceRequired != IndependenceInformation {
		t.Fatalf("AC-02 independence_required = %q, want information", ac02.IndependenceRequired)
	}
	h1 := ac02.EvidenceBindings[0]
	if h1.VisibilityPolicy != VisibilityHeldOut {
		t.Fatalf("EV-AC02-H1 visibility_policy = %q, want held_out", h1.VisibilityPolicy)
	}
	if h1.Basis == nil || h1.Basis.Information != InfoHeldOut {
		t.Fatalf("EV-AC02-H1 independence_basis = %+v, want information=held_out", h1.Basis)
	}
	if h1.Evidence.Store == "" || h1.Evidence.Ref == "" || h1.Evidence.Path != "" {
		t.Fatalf("EV-AC02-H1 evidence = %+v, want store/ref without path", h1.Evidence)
	}
	if c.Acceptances[2].EvidenceBindings[0].Evidence.Type != EvidenceValidator ||
		c.Acceptances[2].EvidenceBindings[0].Evidence.Validator != "retry-budget-v1" {
		t.Fatalf("EV-AC03-01 evidence = %+v", c.Acceptances[2].EvidenceBindings[0].Evidence)
	}
}

// AC2:execution protocol 随 binding 一起冻结;冻结后作者侧任何改动都不影响冻结视图。
func TestExecutionProtocolFrozenWithBinding(t *testing.T) {
	c := mustParse(t, fullContractYAML)
	frozen, err := c.Freeze()
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}

	fc := frozen.Contract()
	got := fc.Acceptances[1].EvidenceBindings[0].ExecutionProtocol
	want := &ExecutionProtocol{Repetitions: 1, Aggregation: AggregationAllPass}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("frozen execution_protocol = %+v, want %+v", got, want)
	}

	// 冻结后改动作者侧原对象(包括指针指向的协议本身),冻结视图不变。
	c.Acceptances[1].EvidenceBindings[0].ExecutionProtocol.Repetitions = 7
	c.Acceptances[1].EvidenceBindings[0].ExecutionProtocol.Aggregation = "any_pass"
	if got := frozen.Contract().Acceptances[1].EvidenceBindings[0].ExecutionProtocol; !reflect.DeepEqual(got, want) {
		t.Fatalf("frozen execution_protocol mutated after freeze: %+v", got)
	}
}

// AC3:runtime requirements 只声明能力,不声明具体 runtime 身份。
func TestRuntimeRequirementsDeclareCapabilitiesOnly(t *testing.T) {
	c := mustParse(t, fullContractYAML)
	r := c.RuntimeRequirements
	if !r.SessionIsolation || !r.HeldOutUnreachable || r.ConformanceClass != "verification-isolation-v1" {
		t.Fatalf("runtime requirements = %+v", r)
	}

	// 结构层守护:字段集合必须恰好是能力声明;出现 runtime 身份字段(名称/版本/种类)即失败。
	typ := reflect.TypeOf(RuntimeRequirements{})
	allowed := map[string]bool{
		"SessionIsolation":   true,
		"HeldOutUnreachable": true,
		"ConformanceClass":   true,
	}
	if typ.NumField() != len(allowed) {
		t.Fatalf("RuntimeRequirements has %d fields, want exactly %v (capabilities only, no runtime identity)", typ.NumField(), allowed)
	}
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		if !allowed[name] {
			t.Fatalf("RuntimeRequirements field %q is not a capability declaration", name)
		}
	}
}

// AC4:revision/hash 稳定(与书写顺序无关、重复计算一致)且冻结后不可变。
func TestHashStableAcrossOrderAndRepeats(t *testing.T) {
	a := mustParse(t, fullContractYAML)
	b := mustParse(t, fullContractYAMLReordered)

	ha, err := Hash(a)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	hb, err := Hash(b)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if ha != hb {
		t.Fatalf("hash differs across equivalent contracts:\n%s\n%s", ha, hb)
	}
	if !strings.HasPrefix(ha, "sha256:") || len(ha) != len("sha256:")+64 {
		t.Fatalf("hash format = %q, want sha256:<64 hex>", ha)
	}

	fa1, err := a.Freeze()
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	fa2, err := a.Freeze()
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	fb, err := b.Freeze()
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	if fa1.Hash() != fa2.Hash() || fa1.Hash() != fb.Hash() {
		t.Fatalf("frozen hashes differ: %s %s %s", fa1.Hash(), fa2.Hash(), fb.Hash())
	}
	if fa1.Hash() != ha {
		t.Fatalf("Freeze hash %s != Hash %s", fa1.Hash(), ha)
	}
}

// AC4:内容任何变化(含 revision)都改变哈希;冻结视图对作者侧后续改动免疫。
func TestFrozenImmutableAfterFreeze(t *testing.T) {
	c := mustParse(t, fullContractYAML)
	frozen, err := c.Freeze()
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	before := frozen.Hash()
	snapshot := frozen.Contract()

	// 冻结视图必须与解析出的原契约内容完全一致(深拷贝不得丢内容)。
	if got := frozen.Contract(); !reflect.DeepEqual(got, *c) {
		t.Fatalf("frozen contract differs from parsed original")
	}

	// 改动作者侧:字段值、嵌套指针、gate map。
	c.Revision = 7
	c.Acceptances[0].Statement = "mutated"
	c.Acceptances[0].EvidenceBindings[0].Protection.BaselineHash = "sha256:" + strings.Repeat("0", 64)
	c.RequiredGates["extra"] = Gate{Required: false}

	if frozen.Hash() != before {
		t.Fatalf("frozen hash changed after mutating original: %s -> %s", before, frozen.Hash())
	}
	if got := frozen.Contract(); !reflect.DeepEqual(got, snapshot) {
		t.Fatalf("frozen contract changed after mutating original")
	}
	if frozen.Revision() != 6 || frozen.TaskID() != "TASK-023" {
		t.Fatalf("frozen revision/task_id = %d/%q, want 6/TASK-023", frozen.Revision(), frozen.TaskID())
	}

	// 改动冻结视图的拷贝也不影响 Frozen 本身。
	cp := frozen.Contract()
	cp.Revision = 99
	if frozen.Revision() != 6 {
		t.Fatalf("frozen revision changed via Contract() copy: %d", frozen.Revision())
	}

	// 内容(revision)变化 → 新冻结哈希不同。
	changed := mustParse(t, fullContractYAML)
	changed.Revision = 7
	fc, err := changed.Freeze()
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	if fc.Hash() == before {
		t.Fatalf("hash unchanged after revision change")
	}
}

// minimalValid — 只含一条 none-independence validator binding 的最小合法契约。
func minimalValid() *Contract {
	return &Contract{
		TaskID:   "TASK-001",
		Revision: 1,
		Acceptances: []Acceptance{{
			ID:        "AC-01",
			Statement: "does the thing",
			EvidenceBindings: []Binding{{
				ID:               "EV-01",
				AuthoringMode:    AuthoringPreCommit,
				VisibilityPolicy: VisibilityVisible,
				Evidence:         EvidenceRef{Type: EvidenceValidator, Validator: "v1"},
			}},
		}},
	}
}

func TestValidationErrors(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(c *Contract)
		wantSub string
	}{
		{
			name:    "missing task_id",
			mutate:  func(c *Contract) { c.TaskID = "" },
			wantSub: "task_id",
		},
		{
			name:    "revision zero",
			mutate:  func(c *Contract) { c.Revision = 0 },
			wantSub: "revision",
		},
		{
			name:    "no acceptances",
			mutate:  func(c *Contract) { c.Acceptances = nil },
			wantSub: "acceptances",
		},
		{
			name: "binding without authoring_mode (INV-VER-01)",
			mutate: func(c *Contract) {
				c.Acceptances[0].EvidenceBindings[0].AuthoringMode = ""
			},
			wantSub: "INV-VER-01",
		},
		{
			name: "bad visibility_policy",
			mutate: func(c *Contract) {
				c.Acceptances[0].EvidenceBindings[0].VisibilityPolicy = "hidden"
			},
			wantSub: "visibility_policy",
		},
		{
			name: "protection mode not immutable (INV-VER-03)",
			mutate: func(c *Contract) {
				b := &c.Acceptances[0].EvidenceBindings[0]
				b.Protection = &Protection{Mode: "readonly"}
			},
			wantSub: "INV-VER-03",
		},
		{
			name: "protection baseline_hash malformed",
			mutate: func(c *Contract) {
				c.Acceptances[0].EvidenceBindings[0].Protection = &Protection{Mode: ProtectionImmutable, BaselineHash: "deadbeef"}
			},
			wantSub: "baseline_hash",
		},
		{
			name: "held_out binding with repo path (8.4.1)",
			mutate: func(c *Contract) {
				c.Acceptances[0].EvidenceBindings[0].VisibilityPolicy = VisibilityHeldOut
				c.Acceptances[0].EvidenceBindings[0].Evidence.Path = "internal/x_test.go"
			},
			wantSub: "held_out",
		},
		{
			name: "held_out binding without runtime capability (INV-VER-05)",
			mutate: func(c *Contract) {
				b := &c.Acceptances[0].EvidenceBindings[0]
				b.VisibilityPolicy = VisibilityHeldOut
				b.Evidence = EvidenceRef{Type: EvidenceTest, Store: "holdout", Ref: "x#T"}
				b.Basis = &IndependenceBasis{Source: SourceVerificationAuthority, Information: InfoHeldOut}
				c.Acceptances[0].IndependenceRequired = IndependenceInformation
			},
			wantSub: "INV-VER-05",
		},
		{
			name: "visible binding referencing holdout store",
			mutate: func(c *Contract) {
				c.Acceptances[0].EvidenceBindings[0].Evidence.Store = "holdout"
			},
			wantSub: "holdout",
		},
		{
			name: "basis information contradicts visibility",
			mutate: func(c *Contract) {
				c.Acceptances[0].EvidenceBindings[0].Basis = &IndependenceBasis{Source: SourceHuman, Information: InfoHeldOut}
			},
			wantSub: "矛盾",
		},
		{
			name: "duplicate binding ids",
			mutate: func(c *Contract) {
				c.Acceptances = append(c.Acceptances, Acceptance{
					ID: "AC-02", Statement: "another",
					EvidenceBindings: []Binding{{ID: "EV-01", AuthoringMode: AuthoringPreCommit, VisibilityPolicy: VisibilityVisible, Evidence: EvidenceRef{Type: EvidenceBuild}}},
				})
			},
			wantSub: "重复",
		},
		{
			name: "source independence without basis (8.5.1 freeze fail)",
			mutate: func(c *Contract) {
				c.Acceptances[0].IndependenceRequired = IndependenceSource
			},
			wantSub: "FREEZE FAIL",
		},
		{
			name: "information independence without held_out (8.5.1 freeze fail)",
			mutate: func(c *Contract) {
				c.Acceptances[0].IndependenceRequired = IndependenceInformation
				c.Acceptances[0].EvidenceBindings[0].Basis = &IndependenceBasis{Source: SourcePreExisting}
			},
			wantSub: "FREEZE FAIL",
		},
		{
			name: "required gate without command",
			mutate: func(c *Contract) {
				c.RequiredGates = map[string]Gate{"build": {Required: true}}
			},
			wantSub: "command",
		},
		{
			name: "empty argv",
			mutate: func(c *Contract) {
				c.RequiredGates = map[string]Gate{"build": {Required: true, Command: &CommandSpec{}}}
			},
			wantSub: "argv",
		},
		{
			name: "negative timeout",
			mutate: func(c *Contract) {
				c.RequiredGates = map[string]Gate{"build": {Required: true, Command: &CommandSpec{Argv: []string{"go"}, Timeout: Duration(-time.Second)}}}
			},
			wantSub: "timeout",
		},
		{
			name: "execution protocol without repetitions",
			mutate: func(c *Contract) {
				c.Acceptances[0].EvidenceBindings[0].ExecutionProtocol = &ExecutionProtocol{Aggregation: AggregationAllPass}
			},
			wantSub: "repetitions",
		},
		{
			name: "execution protocol with unsupported aggregation",
			mutate: func(c *Contract) {
				c.Acceptances[0].EvidenceBindings[0].ExecutionProtocol = &ExecutionProtocol{Repetitions: 2, Aggregation: "any_pass"}
			},
			wantSub: "aggregation",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := minimalValid()
			tc.mutate(c)
			if _, err := c.Freeze(); err == nil {
				t.Fatalf("Freeze: expected error containing %q, got nil", tc.wantSub)
			} else if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("Freeze: error %q does not contain %q", err, tc.wantSub)
			}
		})
	}
}

func TestParseStrictRejectsUnknownFields(t *testing.T) {
	data := fullContractYAML + "\ntypo_field: oops\n"
	if _, err := Parse([]byte(data)); err == nil {
		t.Fatalf("Parse: expected unknown-field error, got nil")
	}
}

func TestParseDuration(t *testing.T) {
	c := mustParse(t, fullContractYAML)
	got := c.RequiredGates["build"].Command.Timeout
	if want := Duration(120 * time.Second); got != want {
		t.Fatalf("gate timeout = %v, want %v", time.Duration(got), time.Duration(want))
	}

	_, err := Parse([]byte("task_id: T\nrevision: 1\nacceptances:\n  - id: A\n    statement: s\n    evidence_bindings:\n      - id: E\n        authoring_mode: pre_commit\n        visibility_policy: visible\n        evidence: {type: build}\nimplementation_runtime_requirements: {}\nrequired_gates:\n  g:\n    required: false\n"))
	if err != nil {
		t.Fatalf("Parse minimal yaml: %v", err)
	}
}
