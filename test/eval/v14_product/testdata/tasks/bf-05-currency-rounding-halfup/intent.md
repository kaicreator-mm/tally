# Intent

The billing formatter converts micro-dollar integers to dollar strings for
invoices, and it truncates instead of rounding: $0.005 renders as "0.00"
and $0.999999 renders as "0.99", so invoice totals silently lose up to a
cent per line.

Fix the formatter to round half away from zero at the cent (2 decimals).
Keep the existing public function signature unchanged. A regression check
must fail on the unfixed code and pass after the fix.
