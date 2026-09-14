# Current Tally Code Evidence

These excerpts come from the user-provided `tally.zip` baseline and are local authority.

## internal/policy/execution.go:61-73

```text
   61: // 编码 agent」(PRD §32.1)由类型形状保证 —— 没有可置位的生产作用域。
   62: type CredentialPolicy struct {
   63: 	ScopedSecrets []string // capability-scoped secret 引用名;空 = 不授予任何凭据
   64: }
   65: 
   66: // RetryPolicy 约束同一 CodegenRun 内的重试次数。必须有限(>=1)。
   67: type RetryPolicy struct {
   68: 	MaxIterations int
   69: }
   70: 
   71: // Budgets 约束一次执行的资源上限。两项都必须有限且为正。
   72: type Budgets struct {
   73: 	MaxTokens   int64
```

## internal/policy/execution.go:76-86

```text
   76: 
   77: // EscalationPolicy 描述失败迭代达到阈值后的 runtime/model/behavior 升级。
   78: // 升级动作的生命周期语义(PRD §11.2.1:终止当前 Run、新 Run 记录 escalated_from)
   79: // 归 CodegenRun 域;本包只拥有阈值与目标 profile。
   80: type EscalationPolicy struct {
   81: 	AfterFailedIterations int
   82: 	TargetProfile         string
   83: }
   84: 
   85: // StopCondition 是触发立即停止当前执行的条件名。三条硬停止条件(PRD §11.2 示例)
   86: // 属于硬下限:任何策略都必须包含;自定义条件只允许追加,移除在 Proposal 类型上
```

## internal/policy/execution.go:104-114

```text
  104: // 构造后必须通过 Validate 才能视为有效。
  105: type ExecutionPolicy struct {
  106: 	Network        NetworkPolicy
  107: 	Shell          ShellPolicy
  108: 	Credentials    CredentialPolicy
  109: 	Retry          RetryPolicy
  110: 	Budgets        Budgets
  111: 	Escalation     EscalationPolicy
  112: 	StopConditions []StopCondition
  113: }
  114: 
```

## internal/policy/execution.go:119-131

```text
  119: func DefaultExecutionPolicy() ExecutionPolicy {
  120: 	return ExecutionPolicy{
  121: 		Network:        NetworkPolicy{Mode: NetworkDeny},
  122: 		Shell:          ShellPolicy{AllowShell: false},
  123: 		Credentials:    CredentialPolicy{},
  124: 		Retry:          RetryPolicy{MaxIterations: 4},
  125: 		Budgets:        Budgets{MaxTokens: 180000, MaxWallTime: 30 * time.Minute},
  126: 		Escalation:     EscalationPolicy{AfterFailedIterations: 2, TargetProfile: "strong-coding"},
  127: 		StopConditions: slices.Clone(hardStopConditions),
  128: 	}
  129: }
  130: 
  131: // Validate 校验策略自身:预算有限且为正、升级阈值不超过重试预算、网络模式自洽、
```

## internal/policy/execution.go:136-146

```text
  136: 		return err
  137: 	}
  138: 	if err := validateCredentials(p.Credentials); err != nil {
  139: 		return err
  140: 	}
  141: 	if p.Retry.MaxIterations < 1 {
  142: 		return fmt.Errorf("%w: retry.max_iterations=%d, 必须 >=1(预算必须有限)", ErrInvalidPolicy, p.Retry.MaxIterations)
  143: 	}
  144: 	if p.Budgets.MaxTokens < 1 {
  145: 		return fmt.Errorf("%w: budgets.max_tokens=%d, 必须 >=1", ErrInvalidPolicy, p.Budgets.MaxTokens)
  146: 	}
```