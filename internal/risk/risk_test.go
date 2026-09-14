package risk

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// ---- fixtures ---------------------------------------------------------------

var (
	sigAuth = Signal{
		ID:    SignalAuthSymbolChange,
		Basis: []string{"intent:diff-17"},
	}
	sigSchema = Signal{
		ID:    SignalSchemaMigration,
		Basis: []string{"contract:TASK-023", "intent:diff-17"},
	}
	sigProtected = Signal{
		ID:    SignalProtectedSurfaceChange,
		Basis: []string{"fact:e-42"},
	}
)

// ---- AC1: stable machine enums ----------------------------------------------

func TestKnownEnumValuesAreStable(t *testing.T) {
	want := []SignalID{"auth_symbol_change", "protected_surface_change", "schema_migration"}
	if got := KnownSignals(); !reflect.DeepEqual(got, want) {
		t.Fatalf("frozen v1.4 enum changed: got %v, want %v", got, want)
	}
	for _, id := range want {
		if !IsKnown(id) {
			t.Errorf("IsKnown(%s) = false, want true", id)
		}
	}
	for _, id := range []SignalID{"", "auth_symbol_changed", "Schema_Migration"} {
		if IsKnown(id) {
			t.Errorf("IsKnown(%q) = true, want false (exact machine match only)", id)
		}
	}
}

// ---- AC2: signals carry basis refs -------------------------------------------

func TestSignalRequiresBasisRefs(t *testing.T) {
	tests := []struct {
		name string
		sig  Signal
	}{
		{"nil basis", Signal{ID: SignalAuthSymbolChange}},
		{"empty basis", Signal{ID: SignalAuthSymbolChange, Basis: []string{}}},
		{"blank ref", Signal{ID: SignalAuthSymbolChange, Basis: []string{"intent:1", "   "}}},
	}
	for _, tc := range tests {
		if err := tc.sig.validate(); !errors.Is(err, ErrMalformedSignal) {
			t.Errorf("%s: validate() = %v, want ErrMalformedSignal", tc.name, err)
		}
		if _, err := Derive([]Signal{tc.sig}); !errors.Is(err, ErrMalformedSignal) {
			t.Errorf("%s: Derive() = %v, want ErrMalformedSignal", tc.name, err)
		}
	}
}

func TestDeriveMergesDuplicatesAndSortsBasis(t *testing.T) {
	res, err := Derive([]Signal{
		{ID: SignalSchemaMigration, Basis: []string{"intent:b", "contract:a"}},
		{ID: SignalSchemaMigration, Basis: []string{"intent:b", "fact:c"}},
	})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(res.Detected) != 1 {
		t.Fatalf("Detected = %d signals, want 1 merged", len(res.Detected))
	}
	want := []string{"contract:a", "fact:c", "intent:b"}
	if got := res.Detected[0].Basis; !reflect.DeepEqual(got, want) {
		t.Fatalf("merged basis = %v, want sorted dedup %v", got, want)
	}
}

// ---- AC4: unknown signals stay visible (INV-V14-09 fail-safe) ----------------

func TestDeriveKeepsUnknownSignalsVisible(t *testing.T) {
	unknown := Signal{ID: SignalID("payment_flow_change"), Basis: []string{"intent:diff-9"}}
	res, err := Derive([]Signal{sigAuth, unknown})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if !res.HasUnknown() {
		t.Fatal("HasUnknown() = false, want true")
	}
	if len(res.Unknown) != 1 || res.Unknown[0].ID != unknown.ID {
		t.Fatalf("Unknown = %v, want the input preserved verbatim", res.Unknown)
	}
	if got := res.Unknown[0].Basis; !reflect.DeepEqual(got, unknown.Basis) {
		t.Fatalf("unknown basis = %v, want %v", got, unknown.Basis)
	}
	// Unknown must remain part of the visible set, never dropped or mapped
	// onto a known ID.
	all := res.All()
	if len(all) != 2 {
		t.Fatalf("All() = %d signals, want 2 (detected + unknown)", len(all))
	}
	if !res.AboveMinimal() {
		t.Fatal("unknown-only presence must keep the set above the minimal-risk floor")
	}
}

// ---- AC3: no authoritative mutable score --------------------------------------

func TestResultCarriesNoScore(t *testing.T) {
	for _, spec := range []struct {
		name string
		typ  reflect.Type
	}{
		{"Signal", reflect.TypeOf(Signal{})},
		{"Result", reflect.TypeOf(Result{})},
	} {
		for i := 0; i < spec.typ.NumField(); i++ {
			name := spec.typ.Field(i).Name
			if strings.Contains(strings.ToLower(name), "score") {
				t.Errorf("%s.%s: score fields must not exist (PRD §22/§31: no authoritative risk score)", spec.name, name)
			}
		}
	}
}

// ---- determinism and purity (PRD §22: stably recomputable) --------------------

func TestDeriveIsOrderIndependent(t *testing.T) {
	input := []Signal{
		sigSchema,
		{ID: SignalID("payment_flow_change"), Basis: []string{"intent:diff-9"}},
		sigAuth,
		{ID: SignalSchemaMigration, Basis: []string{"intent:b"}},
		sigProtected,
	}
	// Every permutation of the same append-input set must derive the same
	// Result (Signal values carry slice fields, so compare field-wise).
	perms := [][]int{
		{0, 1, 2, 3, 4},
		{4, 3, 2, 1, 0},
		{2, 0, 3, 1, 4},
		{3, 1, 4, 0, 2},
	}
	var want Result
	var err error
	if want, err = Derive(pick(input, perms[0])); err != nil {
		t.Fatalf("Derive: %v", err)
	}
	for _, p := range perms[1:] {
		got, err := Derive(pick(input, p))
		if err != nil {
			t.Fatalf("Derive: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("permutation %v derived %+v, want %+v", p, got, want)
		}
	}
}

func TestDeriveDoesNotMutateInputs(t *testing.T) {
	input := []Signal{
		{ID: SignalSchemaMigration, Basis: []string{"intent:b", "contract:a"}},
		sigAuth,
	}
	frozen := copySignals(input)
	if _, err := Derive(input); err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if !reflect.DeepEqual(input, frozen) {
		t.Fatalf("Derive mutated its input: got %+v, want %+v", input, frozen)
	}
}

func TestDeriveIsPure(t *testing.T) {
	input := []Signal{sigSchema, sigAuth, {ID: SignalSchemaMigration, Basis: []string{"fact:c"}}}
	first, err := Derive(input)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	// A caller mutating the first result must not affect re-derivation.
	for i := range first.Detected {
		first.Detected[i].ID = "mutated"
		if len(first.Detected[i].Basis) > 0 {
			first.Detected[i].Basis[0] = "mutated"
		}
	}
	for i := range first.Unknown {
		first.Unknown[i].ID = "mutated"
	}
	second, err := Derive(input)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	want, err := Derive(copySignals(input))
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if !reflect.DeepEqual(second, want) {
		t.Fatalf("re-derivation observed caller mutation of a prior result: got %+v, want %+v", second, want)
	}
}

// ---- empty input and AboveMinimal ----------------------------------------------

func TestDeriveEmptyInput(t *testing.T) {
	for _, input := range [][]Signal{nil, {}} {
		res, err := Derive(input)
		if err != nil {
			t.Fatalf("Derive(%v): %v", input, err)
		}
		if len(res.Detected) != 0 || len(res.Unknown) != 0 || len(res.All()) != 0 {
			t.Fatalf("Derive(%v) = %+v, want empty result", input, res)
		}
		if res.HasUnknown() {
			t.Error("HasUnknown() = true for empty input")
		}
		if res.AboveMinimal() {
			t.Error("AboveMinimal() = true for empty input; only a non-empty set rules out the minimal floor")
		}
	}
}

func TestAboveMinimalForDetectedSignals(t *testing.T) {
	res, err := Derive([]Signal{sigProtected})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if res.HasUnknown() {
		t.Error("HasUnknown() = true for a known signal input")
	}
	if !res.AboveMinimal() {
		t.Error("AboveMinimal() = false for a detected signal; any signal rules out the minimal floor")
	}
}

func TestAllIsCanonicallyOrdered(t *testing.T) {
	res, err := Derive([]Signal{sigSchema, {ID: SignalID("aaa_unknown"), Basis: []string{"intent:x"}}, sigAuth})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	var got []string
	for _, s := range res.All() {
		got = append(got, string(s.ID))
	}
	want := []string{"aaa_unknown", "auth_symbol_change", "schema_migration"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("All() order = %v, want %v", got, want)
	}
}

// ---- helpers --------------------------------------------------------------------

func pick(in []Signal, idx []int) []Signal {
	out := make([]Signal, len(idx))
	for i, k := range idx {
		out[i] = in[k]
	}
	return out
}

func copySignals(in []Signal) []Signal {
	out := make([]Signal, len(in))
	for i, s := range in {
		out[i] = Signal{ID: s.ID, Basis: append([]string(nil), s.Basis...)}
	}
	return out
}
