# Intent

Account e-mail addresses must be unique. Add a uniqueness constraint on
accounts.email to the existing revision chain so the upgrade works both on a
fresh database and on an upgraded database that already carries duplicate
addresses, without deleting any account row.

A regression check must fail on the unfixed revision and pass after the
change; upgrade evidence must cover a fresh database, an N-1 upgrade with
seeded data, and data integrity of the seeded rows.
