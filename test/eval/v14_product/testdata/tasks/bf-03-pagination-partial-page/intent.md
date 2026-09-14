# Intent

List pagination drops the last partial page: a collection of 10 items with
page size 3 reports 3 total pages, so the 4th page (holding the remaining
1 item) becomes unreachable, and every reachable page claims a full
page of items.

Fix the page accounting so that the final partial page exists and each page
reports the number of items it actually carries. Keep the existing public
function signature unchanged. A regression check must fail on the unfixed
code and pass after the fix.
