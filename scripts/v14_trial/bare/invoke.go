// invoke.go owns the bare-arm agent invocation: the codex exec process, its
// isolated per-attempt CODEX_HOME (Decision Lock 22 + ADR-0003 endpoint
// seeding), observed-identity capture, and the append-only attempt record.
//
// The argv shape MIRRORS the production adapter (runtimes/codex/process.go
// buildExecSpec, probe-recorded flags only — DECISION_LOCKS #13):
//
//	[bin] exec --json --skip-git-repo-check -C <ws> -s <sandbox> [-m <model>] -o <file> <instruction>
//
// It deliberately does NOT import runtimes/codex: the bare arm must not wire
// any Tally loop machinery, and parity is pinned at the parameter level and
// documented in docs/evals/v1.4/RUN_PROTOCOL.md. Unrecorded flag shapes and
// unrecorded JSONL event types are never interpreted (probe discipline).
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"tally/test/eval/v14_product"
)

const (
	codexHomeEnvKey = "CODEX_HOME"
	authFileName    = "auth.json"
	configTomlName  = "config.toml"

	// defaultSandbox mirrors the production adapter's conservative default:
	// writable disposable workspace, never danger-full-access.
	defaultSandbox = "workspace-write"
	defaultTimeout = 30 * time.Minute
)

var sandboxModes = map[string]bool{
	"read-only":          true,
	"workspace-write":    true,
	"danger-full-access": true,
}

// providerFlags is the ADR-0003 typed endpoint declaration (desired
// configuration only; structurally impossible to carry instructions or
// session state; no secret value is ever written by the seeding).
type providerFlags struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	BaseURL            string `json:"base_url"`
	WireAPI            string `json:"wire_api"`
	RequiresOpenAIAuth bool   `json:"requires_openai_auth"`
	EnvKey             string `json:"env_key"`
}

func (p *providerFlags) enabled() bool { return p != nil && strings.TrimSpace(p.ID) != "" }

func (p *providerFlags) validate() error {
	if !p.enabled() {
		return nil
	}
	switch {
	case strings.TrimSpace(p.Name) == "":
		return fmt.Errorf("--provider-name is required when --provider-id is set (ADR-0003)")
	case strings.TrimSpace(p.BaseURL) == "":
		return fmt.Errorf("--provider-base-url is required when --provider-id is set (ADR-0003)")
	case p.WireAPI != "responses" && p.WireAPI != "chat":
		return fmt.Errorf("--provider-wire-api must be responses|chat (ADR-0003), got %q", p.WireAPI)
	}
	return nil
}

// renderProviderConfigTOML renders EXACTLY the ADR-0003 §1 minimal config —
// same content the production adapter seeds, nothing else.
func renderProviderConfigTOML(p *providerFlags, model string) (string, error) {
	if err := p.validate(); err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("model_provider = " + tomlQuote(p.ID) + "\n")
	if strings.TrimSpace(model) != "" {
		b.WriteString("model = " + tomlQuote(model) + "\n")
	}
	b.WriteString("\n[model_providers." + p.ID + "]\n")
	b.WriteString("name = " + tomlQuote(p.Name) + "\n")
	b.WriteString("base_url = " + tomlQuote(p.BaseURL) + "\n")
	b.WriteString("wire_api = " + tomlQuote(p.WireAPI) + "\n")
	b.WriteString("requires_openai_auth = " + fmt.Sprintf("%v", p.RequiresOpenAIAuth) + "\n")
	if p.EnvKey != "" {
		b.WriteString("env_key = " + tomlQuote(p.EnvKey) + "\n")
	}
	return b.String(), nil
}

func tomlQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// invokeConfig is one bare attempt's requested configuration.
type invokeConfig struct {
	CodexBin string
	Model    string
	Sandbox  string
	Timeout  time.Duration
	Provider *providerFlags
	// OwnerCodexHome is the ONLY source of the auth.json handoff (Decision
	// Lock 22). Empty + no owner auth.json = no handoff, recorded honestly.
	OwnerCodexHome string
	KeepHome       bool
}

// envIdentity is the host/requested/observed environment record attached to
// every attempt and every judge result (PRE_REGISTRATION §6, Decision Lock
// 15: actual != desired identity; absence is recorded, never fabricated).
type envIdentity struct {
	HostOS     string `json:"host_os"`
	HostArch   string `json:"host_arch"`
	GoVersion  string `json:"go_version"`
	TrialDate  string `json:"trial_date"`
	DBVersion  string `json:"db_version,omitempty"`
	DBRecorded bool   `json:"db_version_recorded"`
}

func captureHostIdentity() envIdentity {
	return envIdentity{
		HostOS:    runtime.GOOS,
		HostArch:  runtime.GOARCH,
		GoVersion: runtime.Version(),
		TrialDate: time.Now().UTC().Format("2006-01-02"),
	}
}

type requestedIdentity struct {
	CodexBin       string         `json:"codex_bin"`
	Model          string         `json:"model"`
	Sandbox        string         `json:"sandbox"`
	TimeoutSeconds int            `json:"timeout_seconds"`
	Provider       *providerFlags `json:"provider,omitempty"`
}

// observedIdentity records what THIS arm actually observed. Model is never
// backfilled from the request: the codex exec JSONL surface does not
// self-report the model (production identity.go makes the same honest
// omission). The bare arm additionally does NOT run Tally's adapter, so an
// adapter version is not an observable fact of this arm.
type observedIdentity struct {
	RuntimeName        string `json:"runtime_name"`
	RuntimeVersion     string `json:"runtime_version"`
	RuntimeVersionRaw  string `json:"runtime_version_raw,omitempty"`
	ThreadID           string `json:"thread_id,omitempty"`
	ModelObserved      bool   `json:"model_observed"`
	ModelNote          string `json:"model_note"`
	AdapterVersionNote string `json:"adapter_version_note"`
	AuthHandoff        string `json:"auth_handoff"`
}

type attemptRecord struct {
	TaskID             string            `json:"task_id"`
	Attempt            int               `json:"attempt"`
	StartedAt          string            `json:"started_at"`
	FinishedAt         string            `json:"finished_at"`
	WallSeconds        float64           `json:"wall_seconds"`
	InstructionSHA256  string            `json:"instruction_sha256"`
	Argv               []string          `json:"argv"`
	ExitStatus         string            `json:"exit_status"` // completed | failed | timeout
	Summary            string            `json:"summary"`
	ThreadID           string            `json:"thread_id,omitempty"`
	LastMessagePresent bool              `json:"last_message_present"`
	CodexHomeRemoved   bool              `json:"codex_home_removed"`
	Requested          requestedIdentity `json:"requested"`
	Observed           observedIdentity  `json:"observed"`
	Host               envIdentity       `json:"host"`
}

// attemptDir is the per-attempt record root: <runsRoot>/<task>/attempt-<N>.
func attemptDir(runsRoot, taskID string, attempt int) string {
	return filepath.Join(runsRoot, taskID, fmt.Sprintf("attempt-%d", attempt))
}

// workspaceDir is the disposable candidate workspace of one attempt.
// Attempt 1 provisions it fresh; later attempts REUSE it unchanged — that is
// the bare arm's nature: the human keeps working in the same workspace.
func workspaceDir(runsRoot, taskID string, attempt int) string {
	return filepath.Join(attemptDir(runsRoot, taskID, attempt), "ws")
}

// runBare executes one bare codex attempt for a task and appends its record.
func runBare(set *frozenSet, task *v14product.ManifestTask, runsRoot string, attempt int, cfg invokeConfig) error {
	if attempt < 1 {
		return fmt.Errorf("--attempt must be >= 1")
	}
	if !sandboxModes[cfg.Sandbox] {
		return fmt.Errorf("illegal sandbox %q (legal: read-only | workspace-write | danger-full-access)", cfg.Sandbox)
	}
	if err := cfg.Provider.validate(); err != nil {
		return err
	}
	aDir := attemptDir(runsRoot, task.TaskID, attempt)
	ws := workspaceDir(runsRoot, task.TaskID, attempt)
	if attempt == 1 {
		if err := provisionTask(set, task, ws); err != nil {
			return err
		}
	} else {
		if _, err := os.Stat(ws); err != nil {
			return fmt.Errorf("attempt %d reuses the attempt-1 workspace but %s does not exist (run attempt 1 first)", attempt, ws)
		}
	}
	instruction, err := intentBytes(set, task, ws)
	if err != nil {
		return err
	}

	rec := attemptRecord{
		TaskID:            task.TaskID,
		Attempt:           attempt,
		StartedAt:         time.Now().UTC().Format(time.RFC3339Nano),
		InstructionSHA256: sha256Hex(instruction),
		ExitStatus:        "failed",
		Requested: requestedIdentity{
			CodexBin:       cfg.CodexBin,
			Model:          cfg.Model,
			Sandbox:        cfg.Sandbox,
			TimeoutSeconds: int(cfg.Timeout.Seconds()),
		},
		Host: captureHostIdentity(),
		Observed: observedIdentity{
			ModelObserved: false,
			ModelNote: "the codex exec JSONL surface does not self-report the model; the requested " +
				"-m value is recorded separately and never backfilled (Decision Lock 15, TASK_SET.md §9)",
			AdapterVersionNote: "the bare arm does not run Tally's codex adapter; an adapter version is not an observable fact of this arm",
		},
	}
	if cfg.Provider.enabled() {
		rec.Requested.Provider = cfg.Provider
	}
	if cfg.Model == "" {
		fmt.Printf("WARN: no --model given; parity requires the SAME explicit model id in both arms (recorded as requested model <empty>)\n")
	}

	// Isolated per-attempt CODEX_HOME (Decision Lock 22): fresh directory,
	// auth.json handoff from the owner home only, never the whole home.
	home := filepath.Join(aDir, "codex-home")
	if err := os.RemoveAll(home); err != nil {
		return err
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		return err
	}
	handoff, err := handOffFileAuth(home, cfg.OwnerCodexHome)
	if err != nil {
		return err
	}
	rec.Observed.AuthHandoff = handoff
	if cfg.Provider.enabled() {
		content, err := renderProviderConfigTOML(cfg.Provider, cfg.Model)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(home, configTomlName), []byte(content), 0o644); err != nil {
			return fmt.Errorf("seed per-run config.toml: %w", err)
		}
	}

	// Observed CLI identity: one real `--version` probe, parsed with the
	// production discipline (self-reported "codex-cli <version>" form only).
	name, version, raw := probeCLIIdentity(context.Background(), cfg.CodexBin)
	rec.Observed.RuntimeName = name
	rec.Observed.RuntimeVersion = version
	rec.Observed.RuntimeVersionRaw = raw

	argv := buildExecArgv(cfg, ws, aDir, instruction)
	rec.Argv = argv

	stdoutPath := filepath.Join(aDir, "codex-stdout.jsonl")
	lastMsgPath := filepath.Join(aDir, "last-message.txt")
	_ = os.Remove(lastMsgPath)

	started := time.Now()
	exitStatus, summary, threadID := spawnCodex(cfg, argv, ws, home, stdoutPath, lastMsgPath, cfg.Timeout)
	rec.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	rec.WallSeconds = time.Since(started).Seconds()
	rec.ExitStatus = exitStatus
	rec.Summary = summary
	rec.ThreadID = threadID
	if _, err := os.Stat(lastMsgPath); err == nil {
		rec.LastMessagePresent = true
	}
	removed := true
	if cfg.KeepHome {
		// A retained home is recorded, never hidden.
		removed = false
	} else if err := os.RemoveAll(home); err != nil {
		// Best-effort deletion mirrors the production Close discipline.
		removed = false
	}
	rec.CodexHomeRemoved = removed

	if err := appendJSONLLines(aDir, "attempt.json", rec); err != nil {
		return err
	}
	if _, err := appendEvent(runsRoot, task.TaskID, attempt, evAgentAttempt, exitStatus+": "+summary); err != nil {
		return err
	}
	fmt.Printf("task %s attempt %d: %s (%.1fs) — records in %s\n", task.TaskID, attempt, exitStatus, rec.WallSeconds, aDir)
	return nil
}

// buildExecArgv mirrors runtimes/codex buildExecSpec exactly (probe-recorded
// flags only; flags before any subcommand token; instruction last because it
// never begins with "-" — it begins with the frozen intent header).
func buildExecArgv(cfg invokeConfig, ws, aDir string, instruction []byte) []string {
	argv := make([]string, 0, 12)
	argv = append(argv, cfg.CodexBin, "exec",
		"--json",
		"--skip-git-repo-check",
		"-C", ws,
		"-s", cfg.Sandbox,
	)
	if cfg.Model != "" {
		argv = append(argv, "-m", cfg.Model)
	}
	argv = append(argv, "-o", filepath.Join(aDir, "last-message.txt"))
	argv = append(argv, string(instruction))
	return argv
}

// spawnCodex runs the codex exec process: streams stdout JSONL to the attempt
// record, extracts the observed thread id (thread.started — a probe-recorded
// shape; all other shapes are ignored, not interpreted), enforces the
// wall-clock ceiling, and classifies the outcome with the production priority:
// timeout > exit code + last-message artifact > in-stream error text.
func spawnCodex(cfg invokeConfig, argv []string, wsDir, home, stdoutPath, lastMsgPath string, timeout time.Duration) (status, summary, threadID string) {
	cmd := exec.Command(argv[0], argv[1:]...) //nolint:gosec // argv is assembled structurally above, never a shell string
	cmd.Dir = wsDir
	cmd.Env = append(os.Environ(), codexHomeEnvKey+"="+home)

	outF, err := os.Create(stdoutPath)
	if err != nil {
		return "failed", fmt.Sprintf("open stdout record: %v", err), ""
	}
	defer outF.Close()
	cmd.Stdout = outF
	var stderr strings.Builder
	cmd.Stderr = &limitedWriter{w: &stderr, limit: 64 * 1024}

	if err := cmd.Start(); err != nil {
		return "failed", fmt.Sprintf("spawn codex: %v", err), ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Poll loop: enforce the wall-clock ceiling and harvest the observed
	// thread id while the process runs; classify after Wait returns.
	thread := ""
	streamErr := ""
	timedOut := false
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
waitLoop:
	for {
		select {
		case <-ctx.Done():
			timedOut = true
			_ = cmd.Process.Kill()
			<-done
			break waitLoop
		case waitErr := <-done:
			_ = waitErr
			break waitLoop
		case <-ticker.C:
			if t, _ := scanThreadID(stdoutPath, &streamErr); t != "" {
				thread = t
			}
		}
	}

	// Post-run full scan for thread id / in-stream error text.
	if t, _ := scanThreadID(stdoutPath, &streamErr); t != "" {
		thread = t
	}
	lastOK := fileExists(lastMsgPath)
	exitCode := -1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	switch {
	case timedOut:
		return "timeout", fmt.Sprintf("wall-clock ceiling %.0fs reached (equal per-task budget, RUN_PROTOCOL §budgets)", timeout.Seconds()), thread
	case exitCode == 0 && lastOK:
		return "completed", "codex exec exited 0 and produced the last-message artifact", thread
	case exitCode == 0:
		return "failed", "codex exec exited 0 but produced no last-message artifact" + streamSuffix(streamErr, stderr.String()), thread
	default:
		return "failed", fmt.Sprintf("codex exec exited %d%s", exitCode, streamSuffix(streamErr, stderr.String())), thread
	}
}

func streamSuffix(streamErr, stderrTxt string) string {
	if streamErr != "" {
		return ": " + streamErr
	}
	if s := strings.TrimSpace(stderrTxt); s != "" {
		return ": " + truncate(s, 500)
	}
	return ""
}

// rawExecEvent mirrors only the probe-recorded JSONL shapes the production
// adapter interprets (thread.started / error / item.completed); everything
// else is ignored, never given invented semantics.
type rawExecEvent struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id"`
	Message  string `json:"message"`
	Item     *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"item"`
}

// scanThreadID reads the recorded stdout JSONL and returns the first observed
// thread id plus the last in-stream error message, if any.
func scanThreadID(path string, lastErr *string) (string, string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()
	thread := ""
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var ev rawExecEvent
		if json.Unmarshal([]byte(line), &ev) != nil {
			continue
		}
		switch ev.Type {
		case "thread.started":
			if thread == "" {
				thread = ev.ThreadID
			}
		case "error":
			if ev.Message != "" {
				*lastErr = ev.Message
			}
		case "item.completed":
			if ev.Item != nil && ev.Item.Type == "error" && ev.Item.Message != "" {
				*lastErr = ev.Item.Message
			}
		}
	}
	return thread, *lastErr
}

var cliVersionRe = regexp.MustCompile(`^(codex-cli)\s+(\S+)`)

// probeCLIIdentity runs one real `codex --version` and parses the self-report
// (production identity.go discipline: name only from the "codex-cli" form;
// unparseable output is recorded verbatim, never invented into fields).
func probeCLIIdentity(ctx context.Context, bin string) (name, version, raw string) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "--version").Output() //nolint:gosec // bin is an explicit flag value
	if err != nil {
		return "", "", fmt.Sprintf("probe failed: %v", err)
	}
	line := strings.TrimSpace(string(out))
	if m := cliVersionRe.FindStringSubmatch(line); m != nil {
		return m[1], m[2], line
	}
	return "", "", line
}

// handOffFileAuth copies the owner home's auth.json into the isolated home
// (Decision Lock 22: the only cross-home transfer; copy, never link). A
// missing owner auth.json is recorded as "none" — keyring deployments are
// unsupported by this handoff and must not be fabricated.
func handOffFileAuth(home, ownerHome string) (string, error) {
	if strings.TrimSpace(ownerHome) == "" {
		return "none (no owner codex home resolved)", nil
	}
	src := filepath.Join(ownerHome, authFileName)
	data, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return "none (owner auth.json absent — keyring-style deployments are not handed off, DECISION_LOCKS #22)", nil
		}
		return "", fmt.Errorf("read owner %s: %w", src, err)
	}
	if err := os.WriteFile(filepath.Join(home, authFileName), data, 0o600); err != nil {
		return "", fmt.Errorf("hand off %s: %w", authFileName, err)
	}
	return "auth.json copied from " + src + " (contents never recorded)", nil
}

// resolveOwnerCodexHome mirrors production ownerCodexHome: explicit flag >
// CODEX_HOME env > ~/.codex.
func resolveOwnerCodexHome(explicit string) string {
	if strings.TrimSpace(explicit) != "" {
		return explicit
	}
	if p := strings.TrimSpace(os.Getenv(codexHomeEnvKey)); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".codex")
}

// appendJSONLLines writes one JSON document (indented) to dir/name.
func appendJSONLLines(dir, name string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), append(b, '\n'), 0o644)
}

type limitedWriter struct {
	w     io.Writer
	limit int
	n     int
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if l.n >= l.limit {
		return len(p), nil
	}
	room := l.limit - l.n
	if len(p) > room {
		l.n += room
		_, err := l.w.Write(p[:room])
		return len(p), err
	}
	l.n += len(p)
	_, err := l.w.Write(p)
	return len(p), err
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
