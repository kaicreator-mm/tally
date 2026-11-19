/** Agent outputs are untrusted observations. Only the existing Host qualifies Task outcomes. */
export const AGENT_PROVIDER_CONTRACT_VERSION: 1;
export const DEFAULT_AGENT_PROVIDER: 'codex';
export type AgentProviderId = string;
export type ProviderOperation = 'probe' | 'execute' | 'cancel' | 'reconcile';
export type ProviderState =
  | 'AVAILABLE' | 'UNAVAILABLE' | 'UNKNOWN' | 'COMPLETED' | 'FAILED'
  | 'CANCELLED' | 'CANCEL_REQUESTED' | 'NOT_RUNNING' | 'RUNNING';
export interface ProviderObservation {
  state: ProviderState;
  reason?: string | null;
  /** Opaque, untrusted, adapter-redacted evidence; never authoritative. */
  evidence?: Record<string, unknown> | null;
}
export interface ProviderAttemptPin {
  readonly taskId: string;
  readonly attemptId: string;
  readonly sourceSha: string;
  readonly providerId: AgentProviderId;
}
export interface ProviderExecutionRequest {
  readonly taskId: string;
  readonly attemptId: string;
  readonly sourceSha: string;
  readonly prompt: string;
  readonly worktreePath: string;
  readonly timeoutMs?: number;
}
/** Supplied ONLY by trusted Host. No task permission is conferred by this type. */
export interface ProviderExecutionContext {
  readonly env: Readonly<Record<string, string>>;
  readonly signal?: AbortSignal;
}
export interface AgentProvider {
  readonly id: AgentProviderId;
  readonly contractVersion: 1;
  readonly capabilities: readonly string[];
  /** All four operations share the Host-supplied execution context shape. */
  probe(context?: ProviderExecutionContext): Promise<ProviderObservation>;
  execute(request: ProviderExecutionRequest, context: ProviderExecutionContext): Promise<ProviderObservation>;
  cancel(attempt: ProviderAttemptPin, context?: ProviderExecutionContext): Promise<ProviderObservation>;
  reconcile(attempt: ProviderAttemptPin, context?: ProviderExecutionContext): Promise<ProviderObservation>;
}
export interface ProviderRegistry {
  list(): readonly Pick<AgentProvider, 'id' | 'contractVersion' | 'capabilities'>[];
  resolve(selection?: {
    providerId?: string;
    enabledProviderIds?: readonly string[];
    pin?: ProviderAttemptPin;
  }): AgentProvider;
}
export function normalizeProviderObservation(operation: ProviderOperation, observation: unknown): Readonly<Required<ProviderObservation>>;
/** Validates binding fields (review P1-1) and returns a frozen shallow copy. */
export function validateProviderExecutionRequest(request: unknown): Readonly<ProviderExecutionRequest>;
export function defineAgentProvider(provider: AgentProvider): AgentProvider;
export function createProviderAttemptPin(pin: ProviderAttemptPin): ProviderAttemptPin;
export function createAgentProviderRegistry(providers: readonly AgentProvider[], options?: {defaultProviderId?: string}): ProviderRegistry;
