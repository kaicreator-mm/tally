# Parallel Execution Plan

Goal: maximize low-cost-model parallelism without allowing concurrent writes to semantic hotspots.

Recommended executor policy:
- max 4–6 cheap workers per wave;
- max 1 strong worker on a hotspot at a time;
- gate tasks run after all wave dependencies are committed;
- each task starts from the exact previous gate HEAD;
- every worker reports changed paths and validation commands.

## Waves

- **W00**: A0-01, A0-02
- **W01**: A0-03, A0-04
- **W02**: A0-05
- **W03**: P0-01, P0-03
- **W04**: P0-02, P0-05
- **W05**: P0-04, P0-06, P0-07
- **W06**: P0-08
- **W07**: P0-09
- **W08**: P1-01
- **W09**: P1-02
- **W10**: P1-03
- **W11**: P1-04
- **W12**: P1-05, P1-06
- **W13**: P1-07
- **W14**: P1-08
- **W15**: P2-01
- **W16**: P2-02, P2-03
- **W17**: P2-04
- **W18**: P2-05
- **W19**: P3-01
- **W20**: P3-02, P3-03
- **W21**: P3-04, P3-05
- **W22**: P3-06
- **W23**: P3-07, P3-08
- **W24**: P3-09
- **W25**: P4-01, P4-03
- **W26**: P4-02
- **W27**: P4-04
- **W28**: P4-05, P4-06
- **W29**: P4-07
- **W30**: P5-01
- **W31**: P5-02
- **W32**: P5-03, P5-04
- **W33**: P5-05
- **W34**: P5-06
- **W35**: P6-01, P6-04, P6-06
- **W36**: P6-02, P6-05
- **W37**: P6-03
- **W38**: P6-07
- **W39**: P6-08

## Parallel lanes

### Authority lane
A0 → P0 → P1 is the critical path and has priority over UI/runtime expansion.

### UX lane
P2 begins only after P1 authority gate. P3 governance builds on P2 review projections.

### Runtime lane
P4-03 spike can start after A0; production DSH adapter waits for P4-03 GO and runtime factory.

### Git lane
P5 starts after P1 authority is stable. It does not depend on P3/P4.

### Validation/Outcome lane
P6-04 migration reproduction and P6-06 export can start early after A0; Outcome wiring waits for P1 acceptance semantics.

## Strong-model checkpoints
A0-02, A0-05, P0-05, P0-07, P0-08, P0-09, P1-02/03/04/06/08,
P2-05, P3-04/05/07/09, P4-01/03/04/05/07, P5-01/03/06,
P6-01/04/05/08.

Everything else should be executable by lower-cost models if task boundaries and L3 references are supplied.


## L3 model allocation

- LOW execution tasks: 40
- HIGH execution tasks: 17
- Prefer parallel LOW workers within path ownership limits.
- HIGH tasks should be serialized at their authority hotspot.
- LOW + HIGH-review tasks may be implemented in parallel but must not cross the next gate before strong semantic review.
