# Intent

Order totals must never go negative, and someone proposed enforcing this by
narrowing orders.total_micros from BIGINT to INTEGER in the next revision.
Narrowing like that breaks upgraded databases that already carry
out-of-range values.

Change the chain so the invariant "total_micros >= 0" is enforced on both a
fresh database and an upgraded database with seeded rows, without
destroying or rejecting existing wide values.

A regression check must fail on the unfixed revision and pass after the
change; upgrade evidence must cover a fresh database, an N-1 upgrade with
seeded data, and data integrity of the seeded rows.
