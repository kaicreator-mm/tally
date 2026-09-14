# Intent

The legacy_token column on accounts is superseded by the sessions table and
must go away. Remove it from the existing revision chain without destroying
any account data on upgraded databases: values still present in
legacy_token must be preserved by the change itself.

A regression check must fail on the unfixed revision and pass after the
change; upgrade evidence must cover a fresh database, an N-1 upgrade with
seeded data, and data integrity of the seeded rows.
