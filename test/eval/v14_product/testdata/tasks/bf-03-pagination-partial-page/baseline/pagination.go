// Task fixture baseline (bf-03). Representative buggy source snapshot of the
// paginator under repair. Input encoding for the frozen oracle:
// "total,pageSize,page" (page is 1-based);
// output: "items=<n>,total_pages=<p>" or "out-of-range".
package web

// Page computes the item count and total page count for a list page.
func Page(total, pageSize, page int) string {
	totalPages := total / pageSize // BUG: floor division drops the partial page
	if page < 1 || page > totalPages {
		return "out-of-range"
	}
	items := pageSize // BUG: assumes every page is full
	return "items=" + itoa(items) + ",total_pages=" + itoa(totalPages)
}
