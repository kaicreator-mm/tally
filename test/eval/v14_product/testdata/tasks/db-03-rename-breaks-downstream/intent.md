# Intent

We renamed accounts.email to accounts.email_address to make the column name
self-describing. A later revision in the chain still creates an index on
the old column name, which breaks the upgrade.

Fix the chain so the downstream index follows the renamed column and the
whole chain applies on both a fresh database and an upgraded database with
seeded rows.

A regression check must fail on the unfixed revision and pass after the
change; upgrade evidence must cover a fresh database, an N-1 upgrade with
seeded data, and data integrity of the seeded rows.
