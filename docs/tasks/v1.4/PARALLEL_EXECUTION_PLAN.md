# Tally v1.4 Parallel Execution Plan

## Scheduling rules

- Execute only tasks whose dependencies are DONE.
- Do not run two tasks concurrently when their Allowed paths overlap at a hotspot.
- Gate tasks require a stronger reviewer and cannot be self-approved by the implementation agent.
- `cmd/worker/main.go`, generated `internal/store/*.sql.go`, and root docs are serialized hotspots.
- L3 research is performed **after** this DAG exists, only for tasks marked `yes` (or `optional` when uncertainty remains).

## W00
Parallel candidates: `A0-01`, `A0-02`, `A0-03`, `A0-04`, `A0-05`

## W01
Parallel candidates: `A0-06`

## W02
Parallel candidates: `P0-01`, `P0-05`, `P0-07`, `P0-08`, `P0-09`, `P0-12`
Serialize hotspot owners: `P0-12`

## W03
Parallel candidates: `P0-02`, `P0-06`, `P0-10`
Serialize hotspot owners: `P0-06`

## W04
Parallel candidates: `P0-03`, `P0-11`
Serialize hotspot owners: `P0-11`

## W05
Parallel candidates: `P0-04`

## W06
Parallel candidates: `P0-13`

## W07
Parallel candidates: `P1-01`, `P1-02`, `P1-04`, `P1-06`

## W08
Parallel candidates: `P1-03`

## W09
Parallel candidates: `P1-05`

## W10
Parallel candidates: `P1-07`

## W11
Parallel candidates: `P1-08`, `P1-09`

## W12
Parallel candidates: `P1-10`

## W13
Parallel candidates: `P2-01`, `P2-02`, `P2-03`

## W14
Parallel candidates: `P2-04`, `P2-05`, `P2-07`

## W15
Parallel candidates: `P2-06`

## W16
Parallel candidates: `P2-08`

## W17
Parallel candidates: `P2-09`
Serialize hotspot owners: `P2-09`

## W18
Parallel candidates: `P2-10`

## W19
Parallel candidates: `P2-11`

## W20
Parallel candidates: `P3-01`, `P3-02`, `P3-05`, `P3-07`

## W21
Parallel candidates: `P3-03`, `P3-04`, `P3-06`

## W22
Parallel candidates: `P3-08`, `P3-09`

## W23
Parallel candidates: `P3-10`

## W24
Parallel candidates: `P4-01`

## W25
Parallel candidates: `P4-02`

## W26
Parallel candidates: `P4-03`

## W27
Parallel candidates: `P4-04`

## W28
Parallel candidates: `P4-05`
Serialize hotspot owners: `P4-05`
