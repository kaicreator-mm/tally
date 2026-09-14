// Task fixture baseline (bf-05). Representative buggy source snapshot of the
// billing formatter under repair. Input encoding for the frozen oracle:
// signed micro-dollar integer (1 dollar = 1000000 micros);
// output: dollars with exactly 2 decimals, e.g. "1.50", "-0.01".
package billing

// FormatDollars renders micros as a dollar string with 2 decimals.
func FormatDollars(micros string) string {
	n := atoi64(micros)
	neg := n < 0
	if neg {
		n = -n
	}
	cents := n / 10000 // BUG: truncates toward zero instead of rounding half away from zero
	return formatSigned(neg, cents)
}
