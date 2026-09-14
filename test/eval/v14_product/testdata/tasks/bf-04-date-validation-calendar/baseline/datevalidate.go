// Task fixture baseline (bf-04). Representative buggy source snapshot of the
// ingest date validator under repair. Input encoding for the frozen oracle:
// strict zero-padded "y-m-d";
// output: "valid", "invalid:format", "invalid:month" or "invalid:day".
package ingest

// ValidateDate reports whether s is a real Gregorian calendar date.
func ValidateDate(s string) string {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return "invalid:format"
	}
	y, mo, d := atoi(s[:4]), atoi(s[5:7]), atoi(s[8:10])
	if pad(s[:4], 4) || pad(s[5:7], 2) || pad(s[8:10], 2) {
		return "invalid:format"
	}
	if mo < 1 || mo > 12 {
		return "invalid:month"
	}
	// BUG: ignores month lengths and leap years.
	if d < 1 || d > 31 {
		return "invalid:day"
	}
	return "valid"
}
