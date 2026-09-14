# v1.5.0 Model Assignment

| Work | Execution | Review | Rationale |
|---|---|---|---|
| Authority/L2/Decision Locks | HIGH | HIGH | cross-package architecture and scope authority |
| Typed leaf domain structs / serializers | LOW-capable after contracts freeze | HIGH sampling | bounded deterministic implementation |
| TaskDAG algorithms/state transitions | HIGH | HIGH | correctness-critical project state authority |
| Completion/Authority gates | HIGH | HIGH | release/transition correctness |
| Intelligence adapters | HIGH | HIGH | provider/state boundary |
| AgentExecutor/Environment/Workspace contracts | HIGH | HIGH | future v1.5 lane compatibility and runtime safety |
| Local compatibility implementation | HIGH | HIGH | process/cancellation/workspace correctness |
| Control-plane integration/composition | HIGH | HIGH independent review | brownfield hotspot |
| Visible tests/fixtures | LOW-capable for mechanical cases; HIGH for lifecycle | HIGH | held-out independence preserved |
| Hidden Validation / Closure / Release Qualification | HIGH | HIGH | integrated release judgment |

Low-cost work must use the frozen Task contract and L3 Reference. It may not redefine public/internal shared contracts or architecture boundaries.
