// Task fixture baseline (bf-01). Representative buggy source snapshot of the
// limiter under repair. Input encoding for the frozen oracle:
// "capacity,refilled,requestIndex" (requestIndex is 1-based);
// output: "allow" or "reject".
package limiter

// Allow reports whether the next immediate request is admitted.
// capacity: burst size; refilled: tokens regained before this request;
// requestIndex: 1-based position in the immediate request sequence.
func Allow(capacity, refilled, requestIndex int) bool {
	// BUG: the +1 admits one extra request after the bucket empties
	// (fails oracle cases fourth-immediate, fifth-after-refill,
	// capacity-two-third; fixed by removing the +1).
	available := capacity + refilled - requestIndex + 1
	if available >= 0 {
		return true
	}
	return false
}
