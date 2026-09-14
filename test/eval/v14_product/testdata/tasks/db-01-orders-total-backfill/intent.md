# Intent

We need to make orders.total_cents a required (NOT NULL) column. Apply the
change to the existing revision chain so that the upgrade works both on a
fresh database and on an upgraded database that already carries order rows.
Existing order rows must keep their identity and linkage columns intact.

A regression check must fail on the unfixed revision and pass after the
change; upgrade evidence must cover a fresh database, an N-1 upgrade with
seeded data, and data integrity of the seeded rows.
