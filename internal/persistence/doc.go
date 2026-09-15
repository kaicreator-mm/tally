// Package persistence owns the database-independent persistence contracts for
// the v1.5 Engineering Control Plane. It may depend on Tally domain types but
// must not import concrete database drivers or database-specific lock/identity
// semantics.
package persistence
