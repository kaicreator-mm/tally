# Intent

Order rows must reference a real account. Add a foreign key from
orders.account_id to accounts.id to the existing revision chain so the
upgrade works both on a fresh database and on an upgraded database that
already carries orders imported with unknown account references, and so
that every seeded order row survives.

A regression check must fail on the unfixed revision and pass after the
change; upgrade evidence must cover a fresh database, an N-1 upgrade with
seeded data, and data integrity of the seeded rows.
