# Intent

The ingest date validator accepts impossible calendar dates such as
2026-02-30 and 2026-04-31: it only checks that the day is within 1..31 and
ignores month lengths and leap years.

Fix the validator so Gregorian calendar rules are enforced exactly: correct
per-month day ranges, leap years divisible by 4 except centuries not
divisible by 400, zero-padded y-m-d formatting only. Keep the existing
public function signature unchanged. A regression check must fail on the
unfixed code and pass after the fix.
