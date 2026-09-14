// Task fixture baseline (bf-06). Representative buggy source snapshot of the
// contact importer under repair. Input encoding for the frozen oracle:
// comma-separated e-mail addresses (spaces around entries allowed);
// output: unique addresses, lowercased, ascending byte order, comma-separated.
package contacts

// UniqueEmails lowercases and deduplicates the address list.
func UniqueEmails(in string) string {
	var seen []string // BUG: keeps original case, so case variants survive
	for _, part := range splitComma(in) {
		addr := trimSpace(part)
		if !containsExact(seen, addr) {
			seen = append(seen, addr)
		}
	}
	sortStrings(seen)
	return joinComma(seen)
}
