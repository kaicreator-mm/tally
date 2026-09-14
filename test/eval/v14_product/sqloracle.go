package v14product

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Deterministic PostgreSQL-dialect static migration analyzer.
//
// This is the verification-side oracle for database_migration tasks. It
// derives the three POL-database_migration evidence keys (fresh_database,
// n_minus_1_upgrade, data_integrity) from the frozen SQL bytes alone —
// no database, no network. The live-DB evidence paths of PRD §24.2 are
// executed at trial time; this analyzer exists so the frozen expectations
// are provably coherent BEFORE any arm runs.
//
// Supported statement subset (everything else is ignored, deterministically):
//   CREATE TABLE t (col TYPE ..., ...)              — schema model
//   INSERT INTO t (cols) [OVERRIDING SYSTEM VALUE] VALUES (...),(...)  — seed
//   ALTER TABLE t ADD COLUMN c TYPE [NOT NULL] [DEFAULT x]
//   ALTER TABLE t ADD CONSTRAINT k UNIQUE (c) / FOREIGN KEY (c) REFERENCES r (rc) / CHECK (...)
//   ALTER TABLE t RENAME COLUMN a TO b
//   ALTER TABLE t DROP COLUMN c
//   ALTER TABLE t ALTER COLUMN c TYPE T
//   CREATE [UNIQUE] INDEX i ON t (c)
//   UPDATE t SET c = 'literal' | othercol [WHERE id = v | id IN (a, b) | (none)]
//   DELETE FROM t / TRUNCATE [TABLE] t / DROP TABLE t
//
// Detection classes (each tagged with the evidence key it fails):
//   not-null-no-default   → n_minus_1_upgrade (ADD COLUMN ... NOT NULL without DEFAULT on a seeded table)
//   unique-duplicate      → n_minus_1_upgrade (UNIQUE constraint over duplicated seeded values)
//   dangling-reference    → fresh_database AND n_minus_1_upgrade (statement still names a renamed-away column)
//   fk-orphan             → n_minus_1_upgrade (FOREIGN KEY over seeded rows referencing missing parents)
//   type-narrow-overflow  → n_minus_1_upgrade (BIGINT → INTEGER narrowing over out-of-range seeded values)
//   dropped-column-data   → data_integrity (DROP COLUMN discards uncopied seeded values)
//   destructive-statement → data_integrity (DELETE/TRUNCATE/DROP TABLE on seeded tables)

// Evidence keys (catalog/v14/policies/database_migration.yaml requires).
const (
	EvFresh   = "fresh_database"
	EvNMinus1 = "n_minus_1_upgrade"
	EvData    = "data_integrity"
)

type violation struct {
	key  string // evidence key this violation fails
	what string // "detector:table.column[=value]"
}

type tableDef struct {
	columns []string
	types   map[string]string
	rows    []map[string]string
}

type sqlModel struct {
	tables      map[string]*tableDef
	renamedAway map[string]map[string]bool // table -> old column names
	copySeen    map[string]map[string]bool // table -> source column already preserved
}

func newSQLModel() *sqlModel {
	return &sqlModel{
		tables:      map[string]*tableDef{},
		renamedAway: map[string]map[string]bool{},
		copySeen:    map[string]map[string]bool{},
	}
}

var (
	reCreateTable = regexp.MustCompile(`^create table ([a-z_][a-z0-9_]*) \((.+)\)$`)
	reInsert      = regexp.MustCompile(`^insert into ([a-z_][a-z0-9_]*) \(([^)]*)\) (?:overriding system value )?values (.+)$`)
	reAddColumn   = regexp.MustCompile(`^alter table ([a-z_][a-z0-9_]*) add column ([a-z_][a-z0-9_]*) (\S+)(?: (.*))?$`)
	reAddUnique   = regexp.MustCompile(`^alter table ([a-z_][a-z0-9_]*) add constraint [a-z_][a-z0-9_]* unique \(([a-z_][a-z0-9_]*)\)$`)
	reAddFK       = regexp.MustCompile(`^alter table ([a-z_][a-z0-9_]*) add constraint [a-z_][a-z0-9_]* foreign key \(([a-z_][a-z0-9_]*)\) references ([a-z_][a-z0-9_]*) \(([a-z_][a-z0-9_]*)\)$`)
	reAddCheck    = regexp.MustCompile(`^alter table ([a-z_][a-z0-9_]*) add constraint [a-z_][a-z0-9_]* check \(.*\)$`)
	reRenameCol   = regexp.MustCompile(`^alter table ([a-z_][a-z0-9_]*) rename column ([a-z_][a-z0-9_]*) to ([a-z_][a-z0-9_]*)$`)
	reDropColumn  = regexp.MustCompile(`^alter table ([a-z_][a-z0-9_]*) drop column ([a-z_][a-z0-9_]*)$`)
	reAlterType   = regexp.MustCompile(`^alter table ([a-z_][a-z0-9_]*) alter column ([a-z_][a-z0-9_]*) type ([a-z0-9]+)$`)
	reUniqueIndex = regexp.MustCompile(`^create unique index [a-z_][a-z0-9_]* on ([a-z_][a-z0-9_]*) \(([a-z_][a-z0-9_]*)\)$`)
	reIndex       = regexp.MustCompile(`^create index [a-z_][a-z0-9_]* on ([a-z_][a-z0-9_]*) \(([a-z_][a-z0-9_]*)\)$`)
	reUpdate      = regexp.MustCompile(`^update ([a-z_][a-z0-9_]*) set ([a-z_][a-z0-9_]*) = ('[^']*'|[a-z0-9_]+)(?: where (.*))?$`)
	reDelete      = regexp.MustCompile(`^delete from ([a-z_][a-z0-9_]*)`)
	reTruncate    = regexp.MustCompile(`^truncate (?:table )?([a-z_][a-z0-9_]*)$`)
	reDropTable   = regexp.MustCompile(`^drop table ([a-z_][a-z0-9_]*)$`)
)

// normalize strips comments, lowercases, and collapses whitespace.
func normalizeSQL(content string) []string {
	var lines []string
	for _, line := range strings.Split(content, "\n") {
		if i := strings.Index(line, "--"); i >= 0 {
			line = line[:i]
		}
		lines = append(lines, line)
	}
	joined := strings.Join(lines, " ")
	for strings.Contains(joined, "  ") {
		joined = strings.ReplaceAll(joined, "  ", " ")
	}
	joined = strings.ReplaceAll(joined, "\t", " ")
	var stmts []string
	for _, s := range strings.Split(joined, ";") {
		if s = strings.TrimSpace(s); s != "" {
			stmts = append(stmts, strings.ToLower(s))
		}
	}
	return stmts
}

func unquote(v string) string {
	if len(v) >= 2 && v[0] == '\'' && v[len(v)-1] == '\'' {
		return v[1 : len(v)-1]
	}
	return v
}

// splitTop splits s on sep at parenthesis depth 0 and outside single quotes.
func splitTop(s string, sep byte) []string {
	var out []string
	depth, inQ, start := 0, false, 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\'':
			inQ = !inQ
		case inQ:
		case c == '(':
			depth++
		case c == ')':
			depth--
		case c == sep && depth == 0:
			out = append(out, strings.TrimSpace(s[start:i]))
			start = i + 1
		}
	}
	out = append(out, strings.TrimSpace(s[start:]))
	return out
}

func (m *sqlModel) apply(stmt string) []violation {
	var vs []violation
	if g := reCreateTable.FindStringSubmatch(stmt); g != nil {
		t := &tableDef{types: map[string]string{}}
		for _, col := range splitTop(g[2], ',') {
			f := strings.Fields(col)
			if len(f) < 2 {
				continue
			}
			switch f[0] {
			case "primary", "unique", "check", "foreign", "constraint":
				continue
			}
			t.columns = append(t.columns, f[0])
			t.types[f[0]] = f[1]
		}
		m.tables[g[1]] = t
		return nil
	}
	if g := reInsert.FindStringSubmatch(stmt); g != nil {
		t := m.tables[g[1]]
		if t == nil {
			return nil
		}
		var cols []string
		for _, c := range strings.Split(g[2], ",") {
			cols = append(cols, strings.TrimSpace(c))
		}
		for _, tuple := range splitTop(g[3], ',') {
			tuple = strings.TrimSpace(tuple)
			tuple = strings.TrimPrefix(tuple, "(")
			tuple = strings.TrimSuffix(tuple, ")")
			vals := splitTop(tuple, ',')
			if len(vals) != len(cols) {
				continue
			}
			row := map[string]string{}
			for i, c := range cols {
				row[c] = unquote(strings.TrimSpace(vals[i]))
			}
			t.rows = append(t.rows, row)
		}
		return nil
	}
	if g := reAddColumn.FindStringSubmatch(stmt); g != nil {
		t := m.tables[g[1]]
		if t == nil {
			return nil
		}
		rest := g[3] + " " + g[4]
		if strings.Contains(rest, "not null") && !strings.Contains(rest, "default") && len(t.rows) > 0 {
			vs = append(vs, violation{EvNMinus1, fmt.Sprintf("not-null-no-default:%s.%s", g[1], g[2])})
		}
		t.columns = append(t.columns, g[2])
		t.types[g[2]] = g[3]
		return vs
	}
	if g := reAddUnique.FindStringSubmatch(stmt); g != nil {
		t := m.tables[g[1]]
		if t == nil {
			return nil
		}
		seen := map[string]bool{}
		for _, row := range t.rows {
			v := row[g[2]]
			if v != "" && seen[v] {
				vs = append(vs, violation{EvNMinus1, fmt.Sprintf("unique-duplicate:%s.%s=%s", g[1], g[2], v)})
			}
			seen[v] = true
		}
		return vs
	}
	if g := reUniqueIndex.FindStringSubmatch(stmt); g != nil {
		t := m.tables[g[1]]
		if t == nil {
			return nil
		}
		seen := map[string]bool{}
		for _, row := range t.rows {
			v := row[g[2]]
			if v != "" && seen[v] {
				vs = append(vs, violation{EvNMinus1, fmt.Sprintf("unique-duplicate:%s.%s=%s", g[1], g[2], v)})
			}
			seen[v] = true
		}
		return vs
	}
	if g := reAddFK.FindStringSubmatch(stmt); g != nil {
		t, r := m.tables[g[1]], m.tables[g[3]]
		if t == nil || r == nil {
			return nil
		}
		parents := map[string]bool{}
		for _, row := range r.rows {
			parents[row[g[4]]] = true
		}
		for _, row := range t.rows {
			v := row[g[2]]
			if !parents[v] {
				vs = append(vs, violation{EvNMinus1, fmt.Sprintf("fk-orphan:%s.%s=%s", g[1], g[2], v)})
			}
		}
		return vs
	}
	if reAddCheck.MatchString(stmt) {
		return nil
	}
	if g := reRenameCol.FindStringSubmatch(stmt); g != nil {
		t := m.tables[g[1]]
		if t == nil {
			return nil
		}
		for i, c := range t.columns {
			if c == g[2] {
				t.columns[i] = g[3]
				t.types[g[3]] = t.types[g[2]]
				delete(t.types, g[2])
			}
		}
		if m.renamedAway[g[1]] == nil {
			m.renamedAway[g[1]] = map[string]bool{}
		}
		m.renamedAway[g[1]][g[2]] = true
		return nil
	}
	if g := reDropColumn.FindStringSubmatch(stmt); g != nil {
		t := m.tables[g[1]]
		if t == nil {
			return nil
		}
		hasData := false
		for _, row := range t.rows {
			if row[g[2]] != "" {
				hasData = true
				break
			}
		}
		if hasData && !m.copySeen[g[1]][g[2]] {
			vs = append(vs, violation{EvData, fmt.Sprintf("dropped-column-data:%s.%s", g[1], g[2])})
		}
		for i, c := range t.columns {
			if c == g[2] {
				t.columns = append(t.columns[:i], t.columns[i+1:]...)
				break
			}
		}
		delete(t.types, g[2])
		return vs
	}
	if g := reAlterType.FindStringSubmatch(stmt); g != nil {
		t := m.tables[g[1]]
		if t == nil {
			return nil
		}
		if strings.EqualFold(t.types[g[2]], "bigint") && (g[3] == "integer" || g[3] == "int") {
			for _, row := range t.rows {
				v := row[g[2]]
				if n, err := strconv.ParseInt(v, 10, 64); err == nil && (n > 2147483647 || n < -2147483648) {
					vs = append(vs, violation{EvNMinus1, fmt.Sprintf("type-narrow-overflow:%s.%s=%s", g[1], g[2], v)})
				}
			}
		}
		t.types[g[2]] = g[3]
		return vs
	}
	if g := reIndex.FindStringSubmatch(stmt); g != nil {
		vs = append(vs, m.dangling(g[1], g[2], stmt)...)
		return vs
	}
	if g := reUpdate.FindStringSubmatch(stmt); g != nil {
		t := m.tables[g[1]]
		if t == nil {
			return nil
		}
		raw := g[3]
		var val string
		isCopy := false
		switch {
		case raw[0] == '\'':
			val = unquote(raw)
		case raw[0] >= '0' && raw[0] <= '9':
			val = raw
		default: // column reference: copy semantics (expand before contract)
			val = raw
			isCopy = true
			if m.copySeen[g[1]] == nil {
				m.copySeen[g[1]] = map[string]bool{}
			}
			m.copySeen[g[1]][raw] = true
		}
		for _, row := range t.rows {
			if !updateMatches(row, g[4]) {
				continue
			}
			if isCopy {
				row[g[2]] = row[val]
			} else {
				row[g[2]] = val
			}
		}
		return nil
	}
	if g := reDelete.FindStringSubmatch(stmt); g != nil {
		if t := m.tables[g[1]]; t != nil && len(t.rows) > 0 {
			vs = append(vs, violation{EvData, fmt.Sprintf("destructive-statement:%s", g[1])})
		}
		return vs
	}
	if g := reTruncate.FindStringSubmatch(stmt); g != nil {
		if t := m.tables[g[1]]; t != nil && len(t.rows) > 0 {
			vs = append(vs, violation{EvData, fmt.Sprintf("destructive-statement:%s", g[1])})
		}
		return vs
	}
	if g := reDropTable.FindStringSubmatch(stmt); g != nil {
		if t := m.tables[g[1]]; t != nil && len(t.rows) > 0 {
			vs = append(vs, violation{EvData, fmt.Sprintf("dropped-table-data:%s", g[1])})
		}
		return vs
	}
	return nil
}

// dangling flags statements that still name a column renamed away on the
// given table. Static: fires on both evidence paths (any upgrade path).
func (m *sqlModel) dangling(table, col, stmt string) []violation {
	if m.renamedAway[table][col] {
		t := m.tables[table]
		if t != nil && !containsCol(t.columns, col) {
			return []violation{
				{EvFresh, fmt.Sprintf("dangling-reference:%s.%s", table, col)},
				{EvNMinus1, fmt.Sprintf("dangling-reference:%s.%s", table, col)},
			}
		}
	}
	return nil
}

func containsCol(cols []string, c string) bool {
	for _, x := range cols {
		if x == c {
			return true
		}
	}
	return false
}

// updateMatches supports the frozen subset: no WHERE (all rows), id = v,
// id in (a, b, ...).
func updateMatches(row map[string]string, where string) bool {
	where = strings.TrimSpace(where)
	if where == "" {
		return true
	}
	if g := strings.SplitN(where, "=", 2); len(g) == 2 && strings.TrimSpace(g[0]) == "id" {
		return row["id"] == unquote(strings.TrimSpace(g[1]))
	}
	if strings.HasPrefix(where, "id in (") && strings.HasSuffix(where, ")") {
		list := where[len("id in (") : len(where)-1]
		for _, v := range strings.Split(list, ",") {
			if row["id"] == unquote(strings.TrimSpace(v)) {
				return true
			}
		}
	}
	return false
}

// pathEvidence is the derived per-path oracle result.
type pathEvidence struct {
	keys  map[string]string   // evidence key -> pass | fail
	viols map[string][]string // evidence key -> deterministic violation list
}

func newPathEvidence() *pathEvidence {
	return &pathEvidence{
		keys:  map[string]string{EvFresh: "pass", EvNMinus1: "pass", EvData: "pass"},
		viols: map[string][]string{EvFresh: {}, EvNMinus1: {}, EvData: {}},
	}
}

func (e *pathEvidence) add(v violation) {
	e.viols[v.key] = append(e.viols[v.key], v.what)
	e.keys[v.key] = "fail"
}

func (e *pathEvidence) normalized() *pathEvidence {
	for k := range e.viols {
		sort.Strings(e.viols[k])
	}
	return e
}

// derivePath walks the revision chain in order. seedContent (if non-empty)
// is applied once the revision whose file base starts with seedAfter has
// been processed.
func derivePath(revFiles []struct{ base, content string }, seedContent, seedAfter string) *pathEvidence {
	m := newSQLModel()
	e := newPathEvidence()
	applySeed := func() {
		for _, stmt := range normalizeSQL(seedContent) {
			_ = m.apply(stmt)
		}
	}
	for _, rev := range revFiles {
		for _, stmt := range normalizeSQL(rev.content) {
			for _, v := range m.apply(stmt) {
				e.add(v)
			}
		}
		if seedAfter != "" && strings.HasPrefix(rev.base, seedAfter) {
			applySeed()
			seedContent = ""
		}
	}
	return e.normalized()
}

// MigrationOracle is the frozen oracle/oracle.json document for a
// database_migration task.
type MigrationOracle struct {
	Layout struct {
		Migrations        []string `json:"migrations"`
		Seed              string   `json:"seed"`
		SeedAfterRevision string   `json:"seed_after_revision"`
		ReferenceRepair   string   `json:"reference_repair"`
		ReferenceReplaces string   `json:"reference_replaces"`
	} `json:"layout"`
	Expected struct {
		Baseline  evidenceExpectation `json:"baseline"`
		Reference evidenceExpectation `json:"reference"`
	} `json:"expected"`
}

type evidenceExpectation struct {
	FreshDatabase  string              `json:"fresh_database"`
	NMinus1Upgrade string              `json:"n_minus_1_upgrade"`
	DataIntegrity  string              `json:"data_integrity"`
	Violations     map[string][]string `json:"violations"`
}

// baselineRevisions returns the ordered (base, content) pair list for the
// baseline chain, reading from repoRoot.
func (o *MigrationOracle) baselineRevisions(repoRoot string) ([]struct{ base, content string }, error) {
	var revs []struct{ base, content string }
	for _, p := range o.Layout.Migrations {
		b, err := readRepoFile(repoRoot, p)
		if err != nil {
			return nil, err
		}
		revs = append(revs, struct{ base, content string }{base: fileBase(p), content: string(b)})
	}
	return revs, nil
}

// derive evaluates one candidate chain. If reference is true, the file named
// by reference_replaces is swapped for reference_repair (new ChangeSet
// content; append-only supersession — the defect file is never mutated).
func (o *MigrationOracle) derive(repoRoot string, reference bool) (*pathEvidence, error) {
	revs, err := o.baselineRevisions(repoRoot)
	if err != nil {
		return nil, err
	}
	if reference {
		swapped := false
		for i := range revs {
			if o.Layout.Migrations[i] == o.Layout.ReferenceReplaces {
				b, err := readRepoFile(repoRoot, o.Layout.ReferenceRepair)
				if err != nil {
					return nil, err
				}
				revs[i].content = string(b)
				swapped = true
			}
		}
		if !swapped {
			return nil, fmt.Errorf("reference_replaces %q not found among migrations", o.Layout.ReferenceReplaces)
		}
	}
	seed := ""
	if o.Layout.Seed != "" {
		b, err := readRepoFile(repoRoot, o.Layout.Seed)
		if err != nil {
			return nil, err
		}
		seed = string(b)
	}
	return derivePath(revs, seed, o.Layout.SeedAfterRevision), nil
}

func readRepoFile(repoRoot, rel string) ([]byte, error) {
	return os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rel)))
}

func fileBase(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[i+1:]
		}
	}
	return p
}

// Expectation converts the frozen expectation into comparable form.
func (x evidenceExpectation) normalized() *pathEvidence {
	e := newPathEvidence()
	e.keys[EvFresh] = orPass(x.FreshDatabase)
	e.keys[EvNMinus1] = orPass(x.NMinus1Upgrade)
	e.keys[EvData] = orPass(x.DataIntegrity)
	for k, list := range x.Violations {
		cp := append([]string(nil), list...)
		sort.Strings(cp)
		e.viols[k] = cp
	}
	return e.normalized()
}

func orPass(s string) string {
	if s == "" {
		return "pass"
	}
	return s
}
