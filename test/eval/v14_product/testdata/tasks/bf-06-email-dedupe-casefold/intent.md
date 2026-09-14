# Intent

The contact importer reports duplicate contacts because e-mail comparison
is case-sensitive: "A@x.com" and "a@X.com" are the same address but both
survive dedupe.

Fix dedupe so addresses compare case-insensitively; output the unique
addresses lowercased in ascending byte order, comma-separated. Keep the
existing public function signature unchanged. A regression check must fail
on the unfixed code and pass after the fix.
