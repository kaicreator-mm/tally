// env.go: environment identity capture (PRE_REGISTRATION §6 — recorded on
// every row; Decision Lock 15: requested configuration is never reported as an
// observed fact) and the trial's ADR-0003 provider surface (coordinator-provided
// environment: codex-cli 0.153.0 + deepseek endpoint + local credential file).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	providerID        = "custom"
	providerName      = "deepseek"
	providerBaseURL   = "https://api.deepseek.com"
	providerWireAPI   = "responses"
	defaultTrialModel = "deepseek-v4-flash"
	authFileEnv       = "TALLY_V14_AUTH_FILE"
	authFileDefault   = ".e2e/deepseek-auth.json"
)

func captureIdentity() identityRecord {
	return identityRecord{
		HostOS:        runtime.GOOS,
		HostArch:      runtime.GOARCH,
		GoVersion:     runtime.Version(),
		TrialDate:     time.Now().UTC().Format("2006-01-02"),
		ModelObserved: false,
	}
}

func trialProviderRecord() providerRecord {
	return providerRecord{
		ID:                 providerID,
		Name:               providerName,
		BaseURL:            providerBaseURL,
		WireAPI:            providerWireAPI,
		RequiresOpenAIAuth: true,
	}
}

// resolveAuthFile locates the credential file (repo-relative default, env
// override). It NEVER reads the contents into any record.
func resolveAuthFile(repoRoot string) string {
	if p := os.Getenv(authFileEnv); p != "" {
		return p
	}
	return filepath.Join(repoRoot, filepath.FromSlash(authFileDefault))
}

// prepareOwnerCodexHome builds the minimal owner CODEX_HOME: a directory that
// carries ONLY auth.json (Decision Lock 22 handoff source). Returns the home
// path and whether the credential was present (skip-not-fail when absent).
func prepareOwnerCodexHome(runsRoot, authFile string) (string, bool, error) {
	home := filepath.Join(runsRoot, "_owner-codex-home")
	if err := os.RemoveAll(home); err != nil {
		return "", false, err
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		return "", false, err
	}
	data, err := os.ReadFile(authFile)
	if err != nil {
		if os.IsNotExist(err) {
			return home, false, nil
		}
		return "", false, fmt.Errorf("read credential file: %w", err)
	}
	// Write mode 0600; contents are never logged, recorded or committed.
	if err := os.WriteFile(filepath.Join(home, "auth.json"), data, 0o600); err != nil {
		return "", false, err
	}
	return home, true, nil
}
