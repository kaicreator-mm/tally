package evidence

import (
	"reflect"
	"testing"
)

// testAuthorship returns a fully provenanced authorship record with one
// field overridden. The base case is the strongest independence claim:
// pre_commit, pre-existing basis, no implementation-lineage contact.
func testAuthorship(override func(*EvidenceAuthorship)) EvidenceAuthorship {
	a := EvidenceAuthorship{
		Basis:         BasisPreExisting,
		AuthoringMode: AuthoringPreCommit,
		Authors: []Contributor{
			{Name: "dana", Role: "human_owner"},
		},
	}
	if override != nil {
		override(&a)
	}
	return a
}

// testLineage returns a provenanced implementation side with one contributor
// overridden.
func testLineage(override func(*ImplementationProvenance)) ImplementationProvenance {
	p := ImplementationProvenance{
		Contributors: []Contributor{
			{Name: "dana", Role: "human_owner"},
		},
	}
	if override != nil {
		override(&p)
	}
	return p
}

// —— AC1: pre-existing evidence can be independent when its authoring
// activity has no contact with the current implementation decision lineage ——

// Source fact: a solo developer authored a test historically and only later
// implements code. Identity equality with the implementation contributor
// alone must not destroy source independence.
func TestSoloDeveloperHistoricalTestStaysIndependent(t *testing.T) {
	dana := Contributor{Name: "dana", Role: "human_owner"}

	got := EffectiveSourceIndependence(
		testAuthorship(func(a *EvidenceAuthorship) {
			a.Basis = BasisPreExisting
			a.Authors = []Contributor{dana}
		}),
		testLineage(func(p *ImplementationProvenance) {
			p.Contributors = []Contributor{dana} // same person implements now
		}),
	)
	if !got {
		t.Fatal("historical pre-existing test of a solo developer must stay source-independent despite identity equality with the implementer")
	}
}

func TestEveryKnownNonImplementationBasisCanStayIndependent(t *testing.T) {
	for _, basis := range []SourceBasis{
		BasisVerificationAuthority,
		BasisPreExisting,
		BasisHuman,
		BasisExternalSystem,
	} {
		got := EffectiveSourceIndependence(
			testAuthorship(func(a *EvidenceAuthorship) { a.Basis = basis }),
			testLineage(nil),
		)
		if !got {
			t.Fatalf("basis %q without lineage contact must stay source-independent", basis)
		}
	}
}

// Rewriting the same test during the implementation run is lineage contact:
// the same solo developer loses independence through activity, not identity.
func TestChangingPreExistingTestInsideLineageLosesIndependence(t *testing.T) {
	dana := Contributor{Name: "dana", Role: "human_owner"}

	got := EffectiveSourceIndependence(
		testAuthorship(func(a *EvidenceAuthorship) {
			a.Authors = []Contributor{dana}
			a.ChangedInLineage = true // dana rewrote the test while implementing
		}),
		testLineage(func(p *ImplementationProvenance) {
			p.Contributors = []Contributor{dana}
		}),
	)
	if got {
		t.Fatal("evidence changed inside the current implementation decision lineage must be source-dependent, even when author and contributor are the same person")
	}
}

// —— AC2: implementation-authored current-run evidence is source-dependent ——

func TestImplementationAuthorityBasisIsDependentByDeclaration(t *testing.T) {
	got := EffectiveSourceIndependence(
		testAuthorship(func(a *EvidenceAuthorship) {
			a.Basis = BasisImplementationAuthority
		}),
		testLineage(nil),
	)
	if got {
		t.Fatal("declared implementation_authority basis must be source-dependent")
	}
}

func TestEvidenceAuthoredInsideLineageIsDependentWhoeverAuthoredIt(t *testing.T) {
	agent := Contributor{Name: "coding-agent-1", Role: "coding_agent"}
	colleague := Contributor{Name: "colleague", Role: "human_owner"}

	cases := []struct {
		name      string
		authors   []Contributor
		basis     SourceBasis
		mode      AuthoringMode
		postHocOK bool // candidate-blind attestation present
	}{
		{
			name:    "coding agent wrote the test during the run under a pre_existing claim",
			authors: []Contributor{agent},
			basis:   BasisPreExisting,
			mode:    AuthoringPreCommit,
		},
		{
			name:    "different person wrote the test while looking at the candidate",
			authors: []Contributor{colleague},
			basis:   BasisHuman,
			mode:    AuthoringPostHoc,
		},
		{
			name:    "implementation contributor authored under verification_authority claim",
			authors: []Contributor{{Name: "dana", Role: "coding_agent"}},
			basis:   BasisVerificationAuthority,
			mode:    AuthoringPreCommit,
		},
	}
	for _, tc := range cases {
		got := EffectiveSourceIndependence(
			testAuthorship(func(a *EvidenceAuthorship) {
				a.Basis = tc.basis
				a.AuthoringMode = tc.mode
				a.AuthoredInLineage = true
				a.Authors = tc.authors
			}),
			testLineage(nil),
		)
		if got {
			t.Fatalf("%s: must be source-dependent (authoring activity inside the implementation decision lineage, %q/%q)", tc.name, tc.basis, tc.mode)
		}
	}
}

// Source fact: a different person looking at the candidate can still be
// source-dependent — post_hoc authoring without the candidate-blind
// attestation is dependent regardless of who authored it.
func TestPostHocWithoutCandidateBlindAttestationIsDependent(t *testing.T) {
	for _, authors := range [][]Contributor{
		{{Name: "colleague", Role: "human_owner"}},
		{{Name: "dana", Role: "human_owner"}},
		{{Name: "coding-agent-1", Role: "coding_agent"}},
	} {
		got := EffectiveSourceIndependence(
			testAuthorship(func(a *EvidenceAuthorship) {
				a.Basis = BasisHuman
				a.AuthoringMode = AuthoringPostHoc
				a.CandidateBlind = false
				a.Authors = authors
			}),
			testLineage(nil),
		)
		if got {
			t.Fatalf("post_hoc authoring without candidate-blind attestation must be source-dependent (authors %v)", authors)
		}
	}
}

// Candidate-blind post_hoc (the §8.2.1 governance-upgrade path) preserves
// independence, and lineage contact still overrides it.
func TestCandidateBlindPostHocPreservesIndependenceUnlessLineageContact(t *testing.T) {
	got := EffectiveSourceIndependence(
		testAuthorship(func(a *EvidenceAuthorship) {
			a.AuthoringMode = AuthoringPostHoc
			a.CandidateBlind = true
		}),
		testLineage(nil),
	)
	if !got {
		t.Fatal("candidate-blind post_hoc authoring must stay source-independent")
	}

	got = EffectiveSourceIndependence(
		testAuthorship(func(a *EvidenceAuthorship) {
			a.AuthoringMode = AuthoringPostHoc
			a.CandidateBlind = true
			a.ChangedInLineage = true
		}),
		testLineage(nil),
	)
	if got {
		t.Fatal("candidate-blind post_hoc evidence changed inside the lineage must be source-dependent")
	}
}

// —— AC3: human role/name alone is neither sufficient nor disqualifying ——

func TestRoleAndNameNeverFlipTheDerivation(t *testing.T) {
	identitySets := [][]Contributor{
		nil, // no recorded authors at all
		{{Name: "dana", Role: "human_owner"}},
		{{Name: "coding-agent-1", Role: "coding_agent"}},
		{{Name: "dana", Role: "coding_agent"}},    // same name, different role
		{{Name: "someone-else", Role: "plumber"}}, // different name, unknown role
	}
	contributorSets := [][]Contributor{
		{{Name: "dana", Role: "human_owner"}},
		{{Name: "coding-agent-1", Role: "coding_agent"}},
		{{Name: "dana", Role: "coding_agent"}, {Name: "someone-else", Role: "human_owner"}},
	}

	// Independent base case: no identity/role variation may disqualify it.
	for i, authors := range identitySets {
		for j, contributors := range contributorSets {
			got := EffectiveSourceIndependence(
				testAuthorship(func(a *EvidenceAuthorship) { a.Authors = authors }),
				testLineage(func(p *ImplementationProvenance) { p.Contributors = contributors }),
			)
			if !got {
				t.Fatalf("independent case must stay independent for authors[%d]/contributors[%d]: role or name acted as disqualifier", i, j)
			}
		}
	}

	// Dependent base case: no identity/role variation may rescue it — in
	// particular a human role or a name differing from every contributor.
	for i, authors := range identitySets {
		for j, contributors := range contributorSets {
			got := EffectiveSourceIndependence(
				testAuthorship(func(a *EvidenceAuthorship) {
					a.AuthoringMode = AuthoringPostHoc // candidate-exposed
					a.Authors = authors
				}),
				testLineage(func(p *ImplementationProvenance) { p.Contributors = contributors }),
			)
			if got {
				t.Fatalf("dependent case must stay dependent for authors[%d]/contributors[%d]: role or name acted as sufficiency", i, j)
			}
		}
	}
}

// —— AC4: derivation is deterministic from provenance inputs ——

func TestDerivationIsDeterministicAndPure(t *testing.T) {
	bases := []SourceBasis{
		BasisVerificationAuthority, BasisPreExisting, BasisHuman,
		BasisExternalSystem, BasisImplementationAuthority, "", SourceBasis("alien"),
	}
	modes := []AuthoringMode{AuthoringPreCommit, AuthoringPostHoc, "", AuthoringMode("posthoc")}
	booleans := []bool{false, true}
	identitySets := [][]Contributor{nil, {{Name: "dana", Role: "human_owner"}}}

	type determinismKey struct {
		basis            SourceBasis
		mode             AuthoringMode
		blind            bool
		authored         bool
		changed          bool
		withContributors bool
	}
	first := map[determinismKey]bool{}
	for _, basis := range bases {
		for _, mode := range modes {
			for _, blind := range booleans {
				for _, authored := range booleans {
					for _, changed := range booleans {
						for _, withContributors := range booleans {
							for _, identities := range identitySets {
								a := testAuthorship(func(x *EvidenceAuthorship) {
									x.Basis = basis
									x.AuthoringMode = mode
									x.CandidateBlind = blind
									x.AuthoredInLineage = authored
									x.ChangedInLineage = changed
									x.Authors = identities
								})
								p := testLineage(func(x *ImplementationProvenance) {
									if !withContributors {
										x.Contributors = nil
									}
								})
								aBefore := testAuthorship(func(x *EvidenceAuthorship) {
									x.Basis = basis
									x.AuthoringMode = mode
									x.CandidateBlind = blind
									x.AuthoredInLineage = authored
									x.ChangedInLineage = changed
									x.Authors = identities
								})
								pBefore := testLineage(func(x *ImplementationProvenance) {
									if !withContributors {
										x.Contributors = nil
									}
								})

								got := EffectiveSourceIndependence(a, p)
								if again := EffectiveSourceIndependence(a, p); got != again {
									t.Fatalf("non-deterministic result for basis=%q mode=%q blind=%v authored=%v changed=%v contributors=%v: %v then %v",
										basis, mode, blind, authored, changed, withContributors, got, again)
								}
								key := determinismKey{basis, mode, blind, authored, changed, withContributors}
								if prev, ok := first[key]; ok && prev != got {
									t.Fatalf("result for key %v flipped between calls: %v then %v", key, prev, got)
								} else if !ok {
									first[key] = got
								}
								if !reflect.DeepEqual(a, aBefore) {
									t.Fatalf("authorship input mutated: basis=%q mode=%q", basis, mode)
								}
								if !reflect.DeepEqual(p, pBefore) {
									t.Fatalf("implementation input mutated: contributors=%v", withContributors)
								}
							}
						}
					}
				}
			}
		}
	}
}

// Conservative defaults: unknown or incomplete provenance derives to
// source-dependent, never to independence.
func TestIncompleteProvenanceDerivesDependent(t *testing.T) {
	cases := []struct {
		name           string
		authorship     EvidenceAuthorship
		implementation ImplementationProvenance
	}{
		{
			name:           "no recorded implementation contributors",
			authorship:     testAuthorship(nil),
			implementation: ImplementationProvenance{},
		},
		{
			name:           "undeclared basis",
			authorship:     testAuthorship(func(a *EvidenceAuthorship) { a.Basis = "" }),
			implementation: testLineage(nil),
		},
		{
			name:           "unknown basis value",
			authorship:     testAuthorship(func(a *EvidenceAuthorship) { a.Basis = SourceBasis("alien_basis") }),
			implementation: testLineage(nil),
		},
		{
			name:           "undeclared authoring mode",
			authorship:     testAuthorship(func(a *EvidenceAuthorship) { a.AuthoringMode = "" }),
			implementation: testLineage(nil),
		},
		{
			name:           "unknown authoring mode value",
			authorship:     testAuthorship(func(a *EvidenceAuthorship) { a.AuthoringMode = AuthoringMode("after_the_fact") }),
			implementation: testLineage(nil),
		},
	}
	for _, tc := range cases {
		if got := EffectiveSourceIndependence(tc.authorship, tc.implementation); got {
			t.Fatalf("%s: must derive source-dependent (conservative default)", tc.name)
		}
	}
}

// Declared basis strings must stay compatible with the frozen contract
// vocabulary (PRD §8.2.2), so callers can convert with plain string
// conversions without a package dependency.
func TestBasisVocabularyMatchesContractVocabulary(t *testing.T) {
	want := map[SourceBasis]string{
		BasisVerificationAuthority:   "verification_authority",
		BasisPreExisting:             "pre_existing",
		BasisHuman:                   "human",
		BasisExternalSystem:          "external_system",
		BasisImplementationAuthority: "implementation_authority",
	}
	for basis, literal := range want {
		if string(basis) != literal {
			t.Fatalf("basis %q drifted from contract vocabulary %q", basis, literal)
		}
	}
	if AuthoringPreCommit != "pre_commit" || AuthoringPostHoc != "post_hoc" {
		t.Fatalf("authoring mode vocabulary drifted: %q/%q", AuthoringPreCommit, AuthoringPostHoc)
	}
}
