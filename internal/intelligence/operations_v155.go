package intelligence

// v1.5.5 adds consumer operations without changing the frozen Provider
// interface. Direct LLM and AI Runtime adapters therefore remain substitutable.
const (
	OperationReplanProposal         Operation = "replan_proposal"
	OperationEngineeringRuleProposal Operation = "engineering_rule_proposal"
)
